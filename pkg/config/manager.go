package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"jusha/mcp/pkg/logging"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
)

// IConfigManager 定义了配置管理的操作接口
type IConfigManager interface {
	// GetConfig 获取指定 dataId 和 group 的配置内容
	GetConfig(dataId, group string) (string, error)
	// PublishConfig 发布（或更新）指定 dataId 和 group 的配置内容
	PublishConfig(dataId, group, content string) (bool, error)
	// DeleteConfig 删除指定 dataId 和 group 的配置
	DeleteConfig(dataId, group string) (bool, error)
	// ListenConfig 监听指定 dataId 和 group 的配置变更
	// onChange 回调函数会在配置发生变化时被调用
	ListenConfig(dataId, group string, onChange func(namespace, group, dataId, data string)) error
	// CancelListenConfig 取消监听指定 dataId 和 group 的配置变更
	CancelListenConfig(dataId, group string) error
}

// IServiceConfigurator is an interface that all service-specific configurations should implement.
type IServiceConfigurator interface {
	GetLogConfig() logging.Config
	GetOldBaseConfig() (Config, error)
	GetBaseConfig() Config
	GetHost() string
	GetHTTPPort() int
	GetGRPCPort() int
	LoadConfig(path string, serviceName string) error
	Raw() string
}

type IConfigurationManager[C IServiceConfigurator] interface {
	GetConfigurator() C
	GetCurrentLogger() logging.ILogger
	StartListener()
	CloseNacosListenerClient()
}

// HandleServiceConfigChangeFunc 是服务具体定义的配置变更处理函数
// 它在 Nacos 配置变更后被调用，用于比较新旧配置，决定是否需要重启，
// 并可以选择性地返回一个新的日志记录器实例
type HandleServiceConfigChangeFunc[C IServiceConfigurator] func(
	appCtx context.Context, // 应用主上下文
	configurator C, // 服务特定的配置实例
	currentLogger logging.ILogger, // 配置变更发生时的当前日志记录器
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger)

// NotifyAppFunc 是 configurationManager 用来通知 App 主逻辑的函数
// newEffLogger 是经过处理后最终生效的日志记录器
// restartSignaledBySvc 指示服务是否通过 HandleServiceConfigChange 发出了重启信号
type NotifyAppFunc[C IServiceConfigurator] func(
	configurator C,
	newEffLogger logging.ILogger,
	restartSignaledBySvc bool,
	reasonsForRestart []string,
)

// configurationManager 负责管理服务的配置加载和热更新
type configurationManager[C IServiceConfigurator] struct {
	appCtx                 context.Context
	serviceName            string
	configDir              string
	nacosDataID            string
	nacosCommonDataID      string
	initialBootstrapLogger logging.ILogger // 用于加载初始配置之前的日志

	handleServiceConfigChangeFunc HandleServiceConfigChangeFunc[C]
	notifyAppFunc                 NotifyAppFunc[C]

	mu            sync.RWMutex
	configurator  C
	currentLogger logging.ILogger
	nacosClient   config_client.IConfigClient // Nacos 配置客户端，用于监听
}

// NewConfigurationManager 创建一个新的 configurationManager 实例
// 它会加载初始配置并设置初始日志记录器
func NewConfigurationManager[C IServiceConfigurator](
	appCtx context.Context,
	serviceName string,
	configDir string,
	nacosDataID string,
	nacosCommonDataID string,
	bootstrapLogger logging.ILogger, // 初始的、不基于配置的日志记录器
	loadServiceSpecificConfigFunc func(configDir string) (C, error),
	handleServiceConfigChangeFunc HandleServiceConfigChangeFunc[C],
	notifyAppFunc NotifyAppFunc[C],
) (*configurationManager[C], error) {
	manager := &configurationManager[C]{
		appCtx:                        appCtx,
		serviceName:                   serviceName,
		configDir:                     configDir,
		nacosDataID:                   nacosDataID,
		nacosCommonDataID:             nacosCommonDataID,
		initialBootstrapLogger:        bootstrapLogger,
		handleServiceConfigChangeFunc: handleServiceConfigChangeFunc,
		notifyAppFunc:                 notifyAppFunc,
	}

	// 加载初始配置
	configurator, configSource, err := LoadInitialConfig(
		manager.initialBootstrapLogger, // 使用 bootstrap logger 加载初始配置
		manager.configDir,
		manager.serviceName,
		manager.nacosDataID,
		manager.nacosCommonDataID,
		loadServiceSpecificConfigFunc,
	)
	if err != nil {
		return nil, fmt.Errorf("配置管理器初始化失败：无法加载初始配置: %w", err)
	}
	manager.configurator = configurator

	logConfig := configurator.GetLogConfig()

	// 基于初始配置创建第一个 "真实" 的日志记录器
	// logging.NewLogger 现在需要 context
	logger := logging.NewLogger(appCtx, logConfig).With("service", serviceName)
	manager.currentLogger = logger

	manager.initialBootstrapLogger.Info(fmt.Sprintf("%s: configurationManager 初始化完成，配置已从 '%s' 加载", serviceName, configSource))
	manager.currentLogger.Debug(fmt.Sprintf("%s: configurationManager 生效配置", serviceName), "config", fmt.Sprintf("%+v", manager.configurator.Raw()))

	return manager, nil
}

// GetConfigurator 返回当前加载的配置
func (cm *configurationManager[C]) GetConfigurator() C {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.configurator
}

// GetCurrentLogger 返回当前配置的日志记录器
func (cm *configurationManager[C]) GetCurrentLogger() logging.ILogger {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentLogger
}

// StartListener 启动 Nacos 配置监听器
func (cm *configurationManager[C]) StartListener() {
	cm.mu.RLock()
	configurator := cm.configurator
	logger := cm.currentLogger
	cm.mu.RUnlock()

	baseConfig := configurator.GetBaseConfig()

	discoveryCfg := baseConfig.Discovery
	if !discoveryCfg.Enabled || discoveryCfg.Type != "nacos" {
		logger.Info("Nacos 服务发现未启用或类型不是 'nacos'，不启动配置监听器")
		return
	}

	nacosDiscoveryCfg := discoveryCfg.Nacos
	configClient, err := newNacosConfigClient(nacosDiscoveryCfg, logger)
	if err != nil {
		logger.Error("Nacos 监听器：创建 Nacos Config Client 失败", "error", err)
		return
	}
	cm.nacosClient = configClient // 保存客户端以便后续可能关闭

	configManagerNacos := newNacosConfigManager(configClient, logger)
	nacosGroup := nacosDiscoveryCfg.GroupName

	onChange := func(namespace, group, dataId, data string) {
		cm.handleNacosChange(namespace, group, dataId, data, configManagerNacos)
	}

	err = configManagerNacos.ListenConfig(cm.nacosDataID, nacosGroup, onChange)
	if err != nil {
		logger.Error("Nacos 监听器：监听服务配置失败", "error", err, "dataId", cm.nacosDataID)
	} else {
		logger.Info("Nacos 监听器：服务配置监听已启动", "dataId", cm.nacosDataID, "group", nacosGroup)
	}

	err = configManagerNacos.ListenConfig(cm.nacosCommonDataID, nacosGroup, onChange)
	if err != nil {
		logger.Error("Nacos 监听器：监听通用配置失败", "error", err, "dataId", cm.nacosCommonDataID)
	} else {
		logger.Info("Nacos 监听器：通用配置监听已启动", "dataId", cm.nacosCommonDataID, "group", nacosGroup)
	}
	// 监听器会阻塞，或者 Nacos SDK 会在后台处理确保 Nacos Client 在服务关闭时被关闭
}

// CloseNacosListenerClient 关闭用于监听的 Nacos 客户端（如果已创建）
func (cm *configurationManager[C]) CloseNacosListenerClient() {
	if cm.nacosClient != nil {
		cm.nacosClient.CloseClient()
		cm.currentLogger.Info("Nacos 配置监听客户端已关闭")
	}
}

func (cm *configurationManager[C]) handleNacosChange(namespace, group, dataId, data string, nacosCfgManager IConfigManager) {
	cm.mu.Lock()
	oldCfg := cm.configurator
	// 使用当前的 logger 实例进行此轮操作的日志记录，后续可能会更新它
	processingLogger := cm.currentLogger
	cm.mu.Unlock()

	processingLogger.Info("Nacos 配置变更通知", "dataId", dataId, "group", group, "namespace", namespace)

	tmpDir, err := os.MkdirTemp("", cm.serviceName+"-nacos-hotload-")
	if err != nil {
		processingLogger.Error("Nacos 热加载：创建临时目录失败", "error", err)
		return
	}
	defer os.RemoveAll(tmpDir)
	processingLogger.Debug("Nacos 热加载：临时目录已创建", "path", tmpDir)

	var serviceConfigContent, commonConfigContent string
	var localServiceConfigPath = filepath.Join(cm.configDir, cm.nacosDataID)
	var localCommonConfigPath = filepath.Join(cm.configDir, cm.nacosCommonDataID)

	switch dataId {
	case cm.nacosDataID:
		serviceConfigContent = data
		commonConfigContent, err = nacosCfgManager.GetConfig(cm.nacosCommonDataID, group)
		if err != nil {
			processingLogger.Warn("Nacos 热加载：获取通用配置失败，尝试读取本地", "error", err, "dataId", cm.nacosCommonDataID)
			commonBytes, readErr := os.ReadFile(localCommonConfigPath)
			if readErr != nil {
				processingLogger.Error("Nacos 热加载：读取本地通用配置也失败，取消更新", "error", readErr)
				return
			}
			commonConfigContent = string(commonBytes)
		}
	case cm.nacosCommonDataID:
		commonConfigContent = data
		serviceConfigContent, err = nacosCfgManager.GetConfig(cm.nacosDataID, group)
		if err != nil {
			processingLogger.Warn("Nacos 热加载：获取服务配置失败，尝试读取本地", "error", err, "dataId", cm.nacosDataID)
			serviceBytes, readErr := os.ReadFile(localServiceConfigPath)
			if readErr != nil {
				processingLogger.Error("Nacos 热加载：读取本地服务配置也失败，取消更新", "error", readErr)
				return
			}
			serviceConfigContent = string(serviceBytes)
		}
	default:
		processingLogger.Warn("Nacos 热加载：收到未知 DataID 的配置变更通知", "dataId", dataId)
		return
	}

	if serviceConfigContent == "" { // 通用配置可以为空
		processingLogger.Error("Nacos 热加载：服务配置内容为空，取消更新")
		return
	}

	tmpServiceCfgPath := filepath.Join(tmpDir, cm.nacosDataID)
	if err := os.WriteFile(tmpServiceCfgPath, []byte(serviceConfigContent), 0644); err != nil {
		processingLogger.Error("Nacos 热加载：写入临时服务配置失败", "error", err)
		return
	}
	tmpCommonCfgPath := filepath.Join(tmpDir, cm.nacosCommonDataID)
	if err := os.WriteFile(tmpCommonCfgPath, []byte(commonConfigContent), 0644); err != nil {
		processingLogger.Error("Nacos 热加载：写入临时通用配置失败", "error", err)
		return
	}

	cm.mu.Lock()
	cm.configurator.LoadConfig(tmpDir, cm.serviceName)
	cm.mu.Unlock()

	// 调用服务特定的处理逻辑
	restartNeeded, reasons, serviceSpecificLogger := cm.handleServiceConfigChangeFunc(
		cm.appCtx,        // 传递应用主上下文
		cm.configurator,  // 服务特定的配置实例
		processingLogger, // 传递变更发生时的 logger
	)

	// 更新 ConfigurationManager持有的当前配置和日志记录器
	var finalNewLogger logging.ILogger
	if serviceSpecificLogger != nil {
		finalNewLogger = serviceSpecificLogger
		processingLogger.Info("服务提供了特定的新日志记录器实例")
	} else if !logging.ConfigsEqual(oldCfg.GetLogConfig(), cm.configurator.GetLogConfig()) {
		processingLogger.Info("日志配置已更改，将重新初始化日志记录器")
		// logging.NewLogger 现在需要 context
		finalNewLogger = logging.NewLogger(cm.appCtx, cm.configurator.GetLogConfig()).With("service", cm.serviceName)
	} else {
		finalNewLogger = processingLogger // 日志配置未变，或服务未提供新logger，则沿用旧logger
	}

	cm.mu.Lock()
	cm.currentLogger = finalNewLogger
	cm.mu.Unlock()

	processingLogger.Info("configurationManager 内部配置和日志记录器已更新")

	// 通知 App 主逻辑
	if cm.notifyAppFunc != nil {
		cm.notifyAppFunc(cm.configurator, finalNewLogger, restartNeeded, reasons)
	}

	// 更新本地备份文件
	UpdateLocalFileWithNacosContent(localServiceConfigPath, []byte(serviceConfigContent), finalNewLogger)
	UpdateLocalFileWithNacosContent(localCommonConfigPath, []byte(commonConfigContent), finalNewLogger)
}

// DefaultHealthCheckHandler 创建一个默认的健康检查处理器
// 它现在接收一个函数，该函数可以返回当前的 slog.Logger 实例
func DefaultHealthCheckHandler[C IServiceConfigurator](cm IConfigurationManager[C]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"status": "ok"}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			cm.GetCurrentLogger().Error("写入健康检查响应失败", "error", err)
			http.Error(w, `{"status":"error","message":"failed to write json response"}`, http.StatusInternalServerError)
		}
	}
}
