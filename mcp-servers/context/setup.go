package context

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/agent/server/context/config"
	"jusha/agent/server/context/internal/wiring"
	"jusha/agent/server/context/tool"
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
	"jusha/mcp/pkg/serviceinit"

	pkg_config "jusha/mcp/pkg/config"
)

// SetupDependenciesAndRun 组装依赖并启动 HTTP 服务
func SetupDependenciesAndRun(
	ctx context.Context,
	configurator config.IContextConfigurator,
	initialLogger logging.ILogger,
	configManager pkg_config.IConfigurationManager[config.IContextConfigurator],
	registrar discovery.IRegistrar,
	namingClient naming_client.INamingClient,
) (cleanupFunc func(), errChan chan error, err error) {
	logger := initialLogger
	if configManager != nil {
		logger = configManager.GetCurrentLogger()
	}

	cfg := configurator.GetConfig()

	logger.Info("Context server configuration loaded",
		"server_port", cfg.Config.Ports.HTTPPort,
		"discovery_enabled", cfg.Config.Discovery != nil && cfg.Config.Discovery.Enabled,
		"enabled_tools", cfg.Tools.EnabledTools,
	)

	errChan = make(chan error, 1)

	// 1) 构建服务 Provider（内部会完成依赖注入容器初始化、仓储/服务等装配）
	container, err := wiring.NewContainer(ctx, logger, configurator)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container: %w", err)
	}

	// 2) 初始化工具管理器
	tm := tool.NewToolManager(logger, configurator, container.GetServiceProvider())
	if err := tm.Init(ctx); err != nil {
		logger.Error("ToolManager init failed", "error", err)
		return nil, nil, fmt.Errorf("tool manager init failed: %w", err)
	}

	// 3) 创建 MCP 服务器并注册工具
	mcpServer := mcp.NewMCPServer("context-server", "1.0.0", logger)
	for _, t := range tm.GetTools() {
		if err := mcpServer.RegisterTool(t); err != nil {
			logger.Warn("tool register failed", "tool", t.Name(), "error", err)
		}
	}

	// 4) HTTP 服务器与路由
	baseCfg := cfg.Config
	timeout := 5 * time.Minute
	if baseCfg != nil && baseCfg.MCP != nil && baseCfg.MCP.ClientTimeout > 0 {
		timeout = time.Duration(baseCfg.MCP.ClientTimeout) * time.Second
	}

	mcpTransport := mcp.NewHTTPTransport(mcpServer, timeout, logger)
	mux := http.NewServeMux()
	var healthCheckHandler http.HandlerFunc
	if appInstance, ok := ctx.Value("appInstance").(*serviceinit.App[config.IContextConfigurator]); ok && appInstance.HealthCheckFunc != nil {
		healthCheckHandler = appInstance.HealthCheckFunc(configManager)
	} else {
		healthCheckHandler = pkg_config.DefaultHealthCheckHandler(configManager)
	}
	mux.HandleFunc("/health", healthCheckHandler)
	mux.Handle("/mcp", mcpTransport)
	mux.Handle("/", mcpTransport)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", baseCfg.Ports.HTTPPort),
		Handler:           mux,
		ReadTimeout:       time.Duration(baseCfg.Timeout.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(baseCfg.Timeout.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(baseCfg.Timeout.IdleTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(baseCfg.Timeout.ReadHeaderTimeout) * time.Second,
	}

	// 5) 启动 HTTP 服务器
	go func() {
		l := logger
		if configManager != nil {
			l = configManager.GetCurrentLogger()
		}
		l.Info("Starting Context Server MCP server", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("HTTP server failed", "error", err)
			errChan <- err
		}
	}()

	// 6) 注册到服务发现
	if baseCfg.Discovery != nil && baseCfg.Discovery.Enabled && registrar != nil {
		serviceName := "context-server"
		host := baseCfg.Host
		if host == "" {
			host = "127.0.0.1" // 默认本地地址
		}
		port := baseCfg.Ports.HTTPPort
		instanceID := fmt.Sprintf("%s:%d", host, port)
		metadata := map[string]string{
			"version": "1.0.0",
			"region":  "default",
		}

		if err := registrar.Register(serviceName, instanceID, host, port, metadata); err != nil {
			logger.Warn("Failed to register service", "error", err)
		} else {
			logger.Info("Service registered successfully", "service", serviceName, "instance", instanceID)
		}
	}

	// 7) 清理函数
	cleanupFunc = func() {
		cl := logger
		if configManager != nil {
			cl = configManager.GetCurrentLogger()
		}
		cl.Info("Shutting down context-server...")

		// 停止 HTTP 服务器
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			cl.Error("Error stopping HTTP server", "error", err)
		} else {
			cl.Info("HTTP server shut down successfully.")
		}

		// 停止 MCP 服务器
		if err := mcpServer.Stop(ctx); err != nil {
			cl.Error("Error stopping MCP server", "error", err)
		}

		// 注销服务
		if baseCfg.Discovery != nil && baseCfg.Discovery.Enabled && registrar != nil {
			serviceName := "context-server"
			host := baseCfg.Host
			if host == "" {
				host = "127.0.0.1"
			}
			port := baseCfg.Ports.HTTPPort

			if err := registrar.Deregister(serviceName, host, port); err != nil {
				cl.Warn("Failed to deregister service", "error", err)
			}
		}

		// 关闭容器资源
		if err := container.Close(); err != nil {
			cl.Error("Failed to close container", "error", err)
		}

		cl.Info("context-server shutdown complete")
	}

	return cleanupFunc, errChan, nil
}
