package rostering

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	svc_config "jusha/agent/rostering/config"
	http_port "jusha/agent/rostering/internal/port/http"
	pkg_config "jusha/mcp/pkg/config"

	"jusha/agent/rostering/internal/wiring"
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/serviceinit"
)

// SetupDependenciesAndRun 组装依赖并启动 HTTP 服务（与其它项目保持一致的 setup 入口）
func SetupDependenciesAndRun(
	ctx context.Context,
	configurator svc_config.IRosteringConfigurator,
	initialLogger logging.ILogger,
	configManager pkg_config.IConfigurationManager[svc_config.IRosteringConfigurator],
	registrar discovery.IRegistrar,
	namingClient naming_client.INamingClient,
) (cleanupFunc func(), errChan chan error, err error) {
	logger := initialLogger
	if configManager != nil {
		logger = configManager.GetCurrentLogger()
	}
	cfg := configurator.GetBaseConfig()

	logger.Info("Scheduling service configuration loaded",
		"server_port", cfg.Ports.HTTPPort,
		"discovery_enabled", cfg.Discovery.Enabled,
	)

	errChan = make(chan error, 1)

	// 1) 构建服务 Provider（内部会完成依赖注入容器初始化、仓储/服务/工作流等装配）
	provider, err := wiring.NewServiceProvider(ctx, logger, configurator)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create service provider: %w", err)
	}

	// 2) 构建 HTTP 路由
	muxHandler := http_port.NewHTTPHandler(provider, logger)

	// 3) 创建 HTTP 服务器（端口与超时从通用配置读取）
	readTimeout := time.Duration(cfg.Timeout.ReadTimeout) * time.Second
	writeTimeout := time.Duration(cfg.Timeout.WriteTimeout) * time.Second
	idleTimeout := time.Duration(cfg.Timeout.IdleTimeout) * time.Second
	readHeaderTimeout := time.Duration(cfg.Timeout.ReadHeaderTimeout) * time.Second

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Ports.HTTPPort),
		Handler:           muxHandler,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// 4) 健康检查路由（与 serviceinit 约定一致）
	// 由于 muxHandler 已经是我们的主 Handler，这里通过额外的 ServeMux 包裹以追加 /health
	root := http.NewServeMux()
	var healthCheckHandler http.HandlerFunc
	if appInstance, ok := ctx.Value("appInstance").(*serviceinit.App[svc_config.IRosteringConfigurator]); ok && appInstance.HealthCheckFunc != nil {
		healthCheckHandler = appInstance.HealthCheckFunc(configManager)
	} else {
		healthCheckHandler = pkg_config.DefaultHealthCheckHandler(configManager)
	}
	root.HandleFunc("/health", healthCheckHandler)
	root.Handle("/", muxHandler)
	httpServer.Handler = root

	// 5) 启动 HTTP 服务器（服务注册由 serviceinit.Run 统一处理，此处不重复注册）
	go func() {
		l := configManager.GetCurrentLogger()
		l.Info("Starting Scheduling Service HTTP server", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("HTTP server failed", "error", err)
			errChan <- err
		}
	}()

	// 6) 关闭函数：优雅关闭 HTTP 与容器中的资源
	cleanupFunc = func() {
		cl := configManager.GetCurrentLogger()
		cl.Info("Shutting down scheduling-service...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			cl.Error("Error stopping HTTP server", "error", err)
		} else {
			cl.Info("HTTP server shut down successfully.")
		}

		// 关闭容器其他资源
		if c, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
			if err := c.Shutdown(context.Background()); err != nil {
				cl.Error("Container shutdown error", "error", err)
			}
		}

		cl.Info("scheduling-service shutdown complete")
	}

	return cleanupFunc, errChan, nil
}

// HandleServiceConfigChange 处理配置热更，判断是否需要重启
func HandleServiceConfigChange(
	ctx context.Context,
	configurator svc_config.IRosteringConfigurator,
	currentLogger logging.ILogger,
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger) {
	logger := currentLogger
	logger.Info("Handling Scheduling Service configuration change...")

	oldCfg, err := configurator.GetOldBaseConfig()
	if err != nil {
		logger.Error("Failed to get old base config during hot-reload", "error", err)
		return false, nil, nil
	}
	newCfg := configurator.GetBaseConfig()

	// 影响监听端口/网络栈的变更需重启
	if oldCfg.Ports.HTTPPort != newCfg.Ports.HTTPPort || oldCfg.Host != newCfg.Host {
		restartNeeded = true
		reasons = append(reasons, "HTTP port or host changed")
	}
	// 超时等 Server 配置变更
	if oldCfg.Timeout != newCfg.Timeout {
		restartNeeded = true
		reasons = append(reasons, "HTTP timeout settings changed")
	}
	// 服务发现配置影响客户端/网关，可要求重启以重新建链
	if oldCfg.Discovery != newCfg.Discovery {
		restartNeeded = true
		reasons = append(reasons, "Service discovery config changed")
	}

	if restartNeeded {
		logger.Warn("Configuration changes require service restart for Scheduling Service.", "reasons", reasons)
	} else {
		logger.Info("No significant configuration changes detected for Scheduling Service.")
	}

	return restartNeeded, reasons, nil
}
