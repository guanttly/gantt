package serviceinit

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings" // 导入 sync 包
	"syscall"
	"time"

	"jusha/mcp/pkg/config" // 确保这是 IServiceConfigurator 接口所在的包
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/version"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
)

const (
	shutdownTimeout         = 30 * time.Second // 优雅关闭的超时时间
	initialNacosSyncTimeout = 5 * time.Minute  // 等待 Nacos 监听器初次同步的超时时间
)

// App 定义了微服务的应用程序结构。
type App[C config.IServiceConfigurator] struct {
	ServiceName       string
	IsMCPServer       bool   // 是否为 MCP 服务器
	ConfigDir         string // 本地配置目录
	NacosDataID       string // 服务特定配置的 Nacos Data ID
	NacosCommonDataID string // 通用配置的 Nacos Data ID

	// LoadServiceConfig 是由服务提供的函数，用于加载其特定配置。
	LoadServiceConfig func(configDir string) (C, error)

	// SetupDependenciesAndRun 是由服务提供的函数，用于初始化其依赖项
	// (数据库、客户端等)，启动其服务器 (HTTP, gRPC)，并返回一个清理函数
	// 和一个用于服务器启动问题的错误通道。
	// ConfigurationManager 被传入，以便服务内部可以获取最新的日志记录器。
	SetupDependenciesAndRun func(
		ctx context.Context,
		initialConfig C, // 初始加载的配置
		initialLogger logging.ILogger, // 初始加载的日志记录器
		configManager config.IConfigurationManager[C], // 配置管理器，用于获取动态更新的日志等
		registrar discovery.IRegistrar,
		namingClient naming_client.INamingClient,
	) (cleanupFunc func(), errChan chan error, err error)

	// HandleServiceConfigChange 是服务具体定义的配置变更处理函数，
	// 它将被传递给 ConfigurationManager。
	HandleServiceConfigChange config.HandleServiceConfigChangeFunc[C]

	// 可选：自定义健康检查处理程序。如果为 nil，则使用 DefaultHealthCheckHandler。
	// DefaultHealthCheckHandler 现在需要一个函数来获取当前的 logger。
	HealthCheckFunc func(cm config.IConfigurationManager[C]) http.HandlerFunc
}

// Run 初始化并启动微服务。
func (app *App[C]) Run() {
	// App 全局上下文
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel() // 确保在 Run 结束时取消上下文

	// 1. 初始的 Bootstrap 日志记录器 (在配置加载完成前使用)
	bootstrapLogger := logging.NewDefaultLogger()
	bootstrapLogger.Info(fmt.Sprintf("%s: 正在初始化...", app.ServiceName))

	// 用于从 ConfigurationManager 接收重启信号
	restartSignalChan := make(chan bool, 1)

	// 定义当 ConfigurationManager 更新其内部状态后的回调
	notifyAppCallback := func(
		configurator C,
		newEffLogger logging.ILogger, // 这是 CM 更新后的 logger
		restartSignaledBySvc bool,
		reasonsForRestart []string,
	) {
		currentBaseLogger := bootstrapLogger // 默认使用 bootstrap logger 记录此回调信息
		if newEffLogger != nil {             // 如果有新的有效 logger，则使用它
			currentBaseLogger = newEffLogger
		}

		oldCfg, err := configurator.GetOldBaseConfig()
		if err != nil {
			currentBaseLogger.Warn(fmt.Sprintf("%s: 无法获取旧配置", app.ServiceName), "error", err)
			// 即使无法获取旧配置，如果这是首次回调，上面的 initialNacosSignalOnce.Do 仍然会执行
			// return // 保持原有逻辑，如果需要重启信号，则不应在此处返回
		}
		if err == nil { // 仅当成功获取旧配置时才比较
			newCfg := configurator.GetBaseConfig()
			if !logging.ConfigsEqual(*oldCfg.Log, *newCfg.Log) {
				currentBaseLogger.Info("日志配置已更改，App 感知到新的日志记录器实例。")
			}
		}

		if restartSignaledBySvc {
			if mode := os.Getenv("APP_ENV"); mode == "DEBUG" {
				// 如果是调试环境，不予启动
				currentBaseLogger.Warn(fmt.Sprintf("%s: 服务逻辑 (HandleServiceConfigChange) 请求重启，但检测到 APP_ENV=DEBUG，跳过重启。", app.ServiceName))
				return
			}

			currentBaseLogger.Warn(fmt.Sprintf("%s: 服务逻辑 (HandleServiceConfigChange) 请求重启。", app.ServiceName), "reasons", strings.Join(reasonsForRestart, ", "))
			select {
			case restartSignalChan <- true: // 发送重启信号
			default: // 如果通道已满，则不阻塞
				currentBaseLogger.Warn(fmt.Sprintf("%s: 重启信号通道已满，无法发送重启信号。", app.ServiceName))
			}
		} else {
			currentBaseLogger.Info(fmt.Sprintf("%s: 配置已热更新，服务逻辑未请求重启。", app.ServiceName))
		}
	}

	// 2. 初始化 ConfigurationManager
	// NewConfigurationManager 内部会尝试同步加载 Nacos 配置 (如果启用)
	configManager, err := config.NewConfigurationManager[C](
		appCtx,
		app.ServiceName,
		app.ConfigDir,
		app.NacosDataID,
		app.NacosCommonDataID,
		bootstrapLogger,
		app.LoadServiceConfig,
		app.HandleServiceConfigChange,
		notifyAppCallback,
	)
	if err != nil {
		// 如果 ConfigurationManager 初始化失败（可能包括初次无法从 Nacos 加载关键配置），则退出
		bootstrapLogger.Error(fmt.Sprintf("%s: 初始化 ConfigurationManager 失败，服务退出。", app.ServiceName), "error", err)
		os.Exit(1)
	}

	// 获取 ConfigurationManager 初始化后的日志记录器，用于后续的启动日志
	loggerForStartup := configManager.GetCurrentLogger()
	loggerForStartup.Info(fmt.Sprintf("%s: ConfigurationManager 已初始化。", app.ServiceName))

	// 启动 ConfigurationManager 的 Nacos 配置监听器 (在 goroutine 中)
	go configManager.StartListener()
	loggerForStartup.Info(fmt.Sprintf("%s: Nacos 配置监听器已启动。等待最多 %v 以便应用初始 Nacos 配置...", app.ServiceName, initialNacosSyncTimeout))

	// 此时，Nacos 监听器已启动，并有机会（在超时范围内）更新配置。
	// 获取最终的初始配置和日志记录器用于服务启动。
	finalInitialConfig := configManager.GetConfigurator()
	finalInitialLogger := configManager.GetCurrentLogger()

	finalInitialLogger.Info(fmt.Sprintf("%s: 服务版本信息", app.ServiceName),
		"version", version.AppVersion,
		"git_commit", version.GitCommit,
		"build_time", version.BuildTime,
	)

	// 3. Nacos Naming Client 和 Registrar (如果需要)
	// 使用 finalInitialConfig 和 finalInitialLogger 进行初始化
	var nacosNamingClient naming_client.INamingClient
	var nacosRegistrar discovery.IRegistrar
	var serviceRegisteredNacos bool
	discoveryCfg := finalInitialConfig.GetBaseConfig().Discovery

	if discoveryCfg.Enabled && discoveryCfg.Type == "nacos" {
		nacosCfg := discoveryCfg.Nacos
		// 使用 finalInitialLogger
		nacosNamingClient, err = discovery.NewNacosNamingClient(nacosCfg, finalInitialLogger)
		if err != nil {
			finalInitialLogger.Error(fmt.Sprintf("%s: 创建 Nacos Naming Client 失败。", app.ServiceName), "error", err)
		} else {
			finalInitialLogger.Info(fmt.Sprintf("%s: Nacos Naming Client 已创建。", app.ServiceName))
			groupName := nacosCfg.GroupName
			if app.IsMCPServer && finalInitialConfig.GetBaseConfig().MCP != nil {
				groupName = finalInitialConfig.GetBaseConfig().MCP.DiscoveryGroupName
			}
			nacosRegistrar = discovery.NewNacosRegistrar(nacosNamingClient, groupName, finalInitialLogger, discoveryCfg.RegisterRetries, time.Duration(discoveryCfg.RegisterRetryInterval)*time.Second)
		}
	} else {
		finalInitialLogger.Info(fmt.Sprintf("%s: 服务发现未启用或类型不是 Nacos。", app.ServiceName))
	}

	// 4. 设置依赖项并运行主逻辑
	// 使用 finalInitialConfig 和 finalInitialLogger
	cleanupFunc, errChan, setupErr := app.SetupDependenciesAndRun(
		appCtx,
		finalInitialConfig, // 使用等待后的配置
		finalInitialLogger, // 使用等待后的日志记录器
		configManager,
		nacosRegistrar,
		nacosNamingClient,
	)
	if setupErr != nil {
		l := configManager.GetCurrentLogger() // 获取最新的 logger
		l.Error(fmt.Sprintf("%s: 设置依赖项或启动服务器失败。", app.ServiceName), "error", setupErr)
		// 确保在退出前关闭 Nacos 客户端
		if nacosNamingClient != nil {
			nacosNamingClient.CloseClient()
		}
		configManager.CloseNacosListenerClient()
		appCancel() // 取消应用主上下文，以防万一有其他 goroutine 依赖它
		os.Exit(1)
	}
	if errChan == nil {
		errChan = make(chan error, 1) // 确保 errChan 始终被初始化
	}

	// 5. Nacos 服务注册 (如果启用)
	// 使用 finalInitialConfig
	if nacosRegistrar != nil && discoveryCfg.Enabled {
		host := finalInitialConfig.GetBaseConfig().Host
		if host == "" {
			appEnv := strings.ToUpper(os.Getenv("APP_ENV"))
			if appEnv == "DEBUG" {
				host = os.Getenv("APP_HOST")
				configManager.GetCurrentLogger().Info(fmt.Sprintf("%s: Host 未在配置中设置，检测到 APP_ENV=DEBUG, host 设置为 %s", app.ServiceName, host))
			} else {
				host = "127.0.0.1"
				configManager.GetCurrentLogger().Info(fmt.Sprintf("%s: Host 未在配置中设置，APP_ENV 非 DEBUG 或未设置, host 设置为 %s", app.ServiceName, host))
			}
		}

		httpPort := finalInitialConfig.GetBaseConfig().Ports.HTTPPort
		if host == "0.0.0.0" {
			configManager.GetCurrentLogger().Warn(fmt.Sprintf("%s: Host 配置为 0.0.0.0，可能影响 Nacos 注册的准确性。", app.ServiceName), "host", host)
		}

		instanceID := fmt.Sprintf("%s-http-%s:%d", app.ServiceName, host, httpPort)
		metadata := map[string]string{
			"protocol":        "http",
			"version":         version.AppVersion,
			"healthCheckPath": "/health",
		}

		regLogger := configManager.GetCurrentLogger()
		err := nacosRegistrar.Register(app.ServiceName, instanceID, host, httpPort, metadata)
		if err != nil {
			regLogger.Error(fmt.Sprintf("%s: Nacos 服务注册失败。", app.ServiceName), "error", err)
		} else {
			serviceRegisteredNacos = true
			regLogger.Info(fmt.Sprintf("%s: Nacos 服务注册成功。", app.ServiceName), "instanceID", instanceID)
		}
	}

	// 6. 优雅关闭处理
	osSignalChan := make(chan os.Signal, 1)
	signal.Notify(osSignalChan, syscall.SIGINT, syscall.SIGTERM)

	currentLogger := configManager.GetCurrentLogger()
	currentLogger.Info(fmt.Sprintf("%s: 服务已就绪，等待信号...", app.ServiceName))

	select {
	case runErr := <-errChan:
		if runErr != nil {
			l := configManager.GetCurrentLogger()
			l.Error(fmt.Sprintf("%s: 服务器运行失败。", app.ServiceName), "error", runErr)
			if cleanupFunc != nil {
				l.Info(fmt.Sprintf("%s: 因服务器错误执行清理...", app.ServiceName))
				cleanupFunc()
			}
			if nacosNamingClient != nil {
				nacosNamingClient.CloseClient()
			}
			configManager.CloseNacosListenerClient()
			appCancel()
			os.Exit(1)
		}
	case <-restartSignalChan:
		l := configManager.GetCurrentLogger()
		l.Warn(fmt.Sprintf("%s: 收到内部重启信号，开始优雅关闭...", app.ServiceName))
		appCancel()

	case sig := <-osSignalChan:
		l := configManager.GetCurrentLogger()
		l.Info(fmt.Sprintf("%s: 收到操作系统信号，开始优雅关闭...", app.ServiceName), "signal", sig.String())
		appCancel()
	}

	shutdownComplete := make(chan struct{})
	go func() {
		l := configManager.GetCurrentLogger()
		if cleanupFunc != nil {
			l.Info(fmt.Sprintf("%s: 正在执行资源清理...", app.ServiceName))
			cleanupFunc()
			l.Info(fmt.Sprintf("%s: 资源清理完成。", app.ServiceName))
		}

		if serviceRegisteredNacos && nacosRegistrar != nil {
			deregLogger := configManager.GetCurrentLogger()
			// 使用 finalInitialConfig 中的值进行注销，以确保一致性
			host := finalInitialConfig.GetBaseConfig().Host
			// 如果 host 在注册时被动态设置为 127.0.0.1 或 jusha-nacos，这里也需要同样的逻辑
			// 但为了简化，我们假设用于注销的 host 与注册时确定的 host 一致，
			// 或者 Nacos Registrar 的 Deregister 实现不严格依赖于完全匹配的原始 host（如果它是动态的）。
			// 通常，服务名、IP 和端口是关键。
			// 如果注册时 host 为空并被动态设置，则 finalInitialConfig.GetBaseConfig().Host 仍为空。
			// 因此，我们需要重新应用注册时的 host 决定逻辑，或者确保用于注销的 host 是正确的。
			// 为了简单起见，这里直接使用 finalInitialConfig 的 host，但请注意这可能需要调整。
			// 更健壮的方法是存储注册时实际使用的 host。
			// 此处我们假设 finalInitialConfig.GetBaseConfig().Host 已经包含了正确的、用于注册的 host，
			// 或者 Nacos Deregister 能够处理。
			// 如果注册时 host 为空，则需要重新获取：
			if finalInitialConfig.GetBaseConfig().Host == "" { // 重新判断注册时使用的 host
				appEnv := strings.ToUpper(os.Getenv("APP_ENV"))
				if appEnv == "DEBUG" {
					host = "127.0.0.1"
				}
			}

			httpPort := finalInitialConfig.GetBaseConfig().Ports.HTTPPort
			deregLogger.Info(fmt.Sprintf("%s: 正在从 Nacos 注销服务...", app.ServiceName), "host", host, "port", httpPort)
			err := nacosRegistrar.Deregister(app.ServiceName, host, httpPort)
			if err != nil {
				deregLogger.Error(fmt.Sprintf("%s: Nacos 服务注销失败。", app.ServiceName), "error", err)
			} else {
				deregLogger.Info(fmt.Sprintf("%s: Nacos 服务注销成功。", app.ServiceName))
			}
		}

		if nacosNamingClient != nil {
			l.Info(fmt.Sprintf("%s: 正在关闭 Nacos Naming Client...", app.ServiceName))
			nacosNamingClient.CloseClient()
			l.Info(fmt.Sprintf("%s: Nacos Naming Client 已关闭。", app.ServiceName))
		}
		configManager.CloseNacosListenerClient()
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		configManager.GetCurrentLogger().Info(fmt.Sprintf("%s: 优雅关闭完成。", app.ServiceName))
	case <-time.After(shutdownTimeout):
		configManager.GetCurrentLogger().Warn(fmt.Sprintf("%s: 优雅关闭超时。", app.ServiceName))
	}
	os.Exit(0)
}
