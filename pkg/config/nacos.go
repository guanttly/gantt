package config

import (
	"fmt"
	"jusha/mcp/pkg/logging"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client" // 显式导入 naming_client
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func newNacosConfigClient(cfg *NacosConfig, logger logging.ILogger) (config_client.IConfigClient, error) {
	logger = logger.With("component", "NacosClient") // 添加组件标签

	// 1. 解析服务器地址
	serverConfigs := []constant.ServerConfig{}
	addrs := strings.Split(cfg.Addresses, ",")
	if len(addrs) == 0 || addrs[0] == "" {
		return nil, fmt.Errorf("Nacos 配置错误：缺少服务器地址 (addresses)")
	}
	for _, addr := range addrs {
		parts := strings.Split(strings.TrimSpace(addr), ":")
		if len(parts) != 2 {
			logger.Warn("无效的 Nacos 服务器地址格式 (应为 host:port)", "address", addr)
			continue
		}
		host := parts[0]
		port, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			logger.Warn("无效的 Nacos 服务器端口", "address", addr, "error", err)
			continue
		}
		serverConfig := *constant.NewServerConfig(host, port)
		serverConfig.GrpcPort = port + 1000 // Nacos 默认 gRPC 端口为 HTTP 端口 + 1000
		serverConfigs = append(serverConfigs, serverConfig)
	}
	if len(serverConfigs) == 0 {
		return nil, fmt.Errorf("nacos 配置错误：没有有效的服务器地址")
	}

	// 2. 创建客户端配置
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(cfg.NamespaceID),
		constant.WithTimeoutMs(cfg.TimeoutMs),
		constant.WithNotLoadCacheAtStart(!cfg.NotLoadCache), // 注意 Nacos SDK 的 NotLoadCacheAtStart 逻辑相反
		constant.WithLogDir(cfg.LogDir),
		constant.WithCacheDir(cfg.CacheDir),
		constant.WithLogLevel(cfg.LogLevel),
		constant.WithUpdateThreadNum(cfg.UpdateThreadNum),
		constant.WithUsername(cfg.Username),
		constant.WithPassword(cfg.Password),
	)

	// 3. 创建 ConfigClient
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		logger.Error("创建 Nacos ConfigClient 失败", "error", err)
		return nil, fmt.Errorf("无法创建 Nacos 配置客户端: %w", err)
	}

	return configClient, nil
}

// nacosConfigManager 实现了 IConfigManager 接口，用于 Nacos 配置中心
type nacosConfigManager struct {
	client config_client.IConfigClient
	logger logging.ILogger
}

// NewNacosConfigManager 创建一个新的 Nacos IConfigManager 实例
func newNacosConfigManager(client config_client.IConfigClient, logger logging.ILogger) IConfigManager {
	return &nacosConfigManager{
		client: client,
		logger: logger.With("component", "NacosConfigManager"),
	}
}

// GetConfig 从 Nacos 获取配置
func (n *nacosConfigManager) GetConfig(dataId, group string) (string, error) {
	n.logger.Debug("从 Nacos 获取配置...", "dataId", dataId, "group", group)
	content, err := n.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		n.logger.Error("从 Nacos 获取配置失败", "error", err, "dataId", dataId, "group", group)
		return "", fmt.Errorf("获取 Nacos 配置失败 (dataId: %s, group: %s): %w", dataId, group, err)
	}
	n.logger.Info("从 Nacos 获取配置成功", "dataId", dataId, "group", group)
	return content, nil
}

// PublishConfig 发布配置到 Nacos
func (n *nacosConfigManager) PublishConfig(dataId, group, content string) (bool, error) {
	n.logger.Debug("向 Nacos 发布配置...", "dataId", dataId, "group", group)
	success, err := n.client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
	})
	if err != nil {
		n.logger.Error("向 Nacos 发布配置失败", "error", err, "dataId", dataId, "group", group)
		return false, fmt.Errorf("发布 Nacos 配置失败 (dataId: %s, group: %s): %w", dataId, group, err)
	}
	if success {
		n.logger.Info("向 Nacos 发布配置成功", "dataId", dataId, "group", group)
	} else {
		n.logger.Warn("向 Nacos 发布配置调用成功，但返回值为 false", "dataId", dataId, "group", group)
	}
	return success, nil
}

// DeleteConfig 从 Nacos 删除配置
func (n *nacosConfigManager) DeleteConfig(dataId, group string) (bool, error) {
	n.logger.Debug("从 Nacos 删除配置...", "dataId", dataId, "group", group)
	success, err := n.client.DeleteConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		n.logger.Error("从 Nacos 删除配置失败", "error", err, "dataId", dataId, "group", group)
		return false, fmt.Errorf("删除 Nacos 配置失败 (dataId: %s, group: %s): %w", dataId, group, err)
	}
	if success {
		n.logger.Info("从 Nacos 删除配置成功", "dataId", dataId, "group", group)
	} else {
		n.logger.Warn("从 Nacos 删除配置调用成功，但返回值为 false", "dataId", dataId, "group", group)
	}
	return success, nil
}

// ListenConfig 监听 Nacos 配置变更
func (n *nacosConfigManager) ListenConfig(dataId, group string, onChange func(namespace, group, dataId, data string)) error {
	n.logger.Debug("开始监听 Nacos 配置变更...", "dataId", dataId, "group", group)
	err := n.client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			n.logger.Info("Nacos 配置发生变更", "namespace", namespace, "group", group, "dataId", dataId)
			onChange(namespace, group, dataId, data) // 调用外部传入的回调
		},
	})
	if err != nil {
		n.logger.Error("监听 Nacos 配置变更失败", "error", err, "dataId", dataId, "group", group)
		return fmt.Errorf("监听 Nacos 配置失败 (dataId: %s, group: %s): %w", dataId, group, err)
	}
	return nil
}

// CancelListenConfig 取消监听 Nacos 配置变更
func (n *nacosConfigManager) CancelListenConfig(dataId, group string) error {
	n.logger.Debug("取消监听 Nacos 配置变更...", "dataId", dataId, "group", group)
	err := n.client.CancelListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		n.logger.Error("取消监听 Nacos 配置变更失败", "error", err, "dataId", dataId, "group", group)
		return fmt.Errorf("取消监听 Nacos 配置失败 (dataId: %s, group: %s): %w", dataId, group, err)
	}
	n.logger.Info("成功取消 Nacos 配置监听", "dataId", dataId, "group", group)
	return nil
}
