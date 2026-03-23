package rostering

import (
	"context"
	"fmt"
	"jusha/mcp/pkg/logging"
	"net/http"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/mcp"
	"jusha/mcp/pkg/serviceinit"

	"jusha/gantt/mcp/rostering/tool"

	rostering_config "jusha/gantt/mcp/rostering/config"
	pkg_config "jusha/mcp/pkg/config"
)

// SetupDependenciesAndRun 初始化依赖并启动 HTTP/MCP 服务
func SetupDependenciesAndRun(
	ctx context.Context,
	configurator rostering_config.IRosteringConfigurator,
	initialLogger logging.ILogger,
	configManager pkg_config.IConfigurationManager[rostering_config.IRosteringConfigurator],
	registrar discovery.IRegistrar,
	namingClient naming_client.INamingClient,
) (cleanupFunc func(), errChan chan error, err error) {
	logger := initialLogger
	if configManager != nil {
		logger = configManager.GetCurrentLogger()
	}
	cfg := configurator.GetConfig()

	logger.Info("Data server configuration loaded",
		"server_port", cfg.Config.Ports.HTTPPort,
		"enabled_tool", cfg.Tools.EnabledTools)

	errChan = make(chan error, 1)

	// 1. 初始化工具管理器（使用 management-service 客户端）
	tm := tool.NewToolManager(logger, configurator, namingClient)
	if err := tm.Init(ctx); err != nil {
		logger.Error("ToolManager init failed", "error", err)
	}

	// 2. 创建 MCP 服务器并注册工具（基于 ToolManager）
	mcpServer := mcp.NewMCPServer("data-server", "1.0.0", logger)
	for _, t := range tm.GetTools() {
		rError := mcpServer.RegisterTool(t)
		if rError != nil {
			logger.Warn("tool register failed", "tool", t.Name(), "error", rError)
		}
	}

	// 3. HTTP 服务器与路由
	baseCfg := cfg.Config
	timeout := 5 * time.Minute
	if baseCfg != nil && baseCfg.MCP != nil && baseCfg.MCP.ClientTimeout > 0 {
		timeout = time.Duration(baseCfg.MCP.ClientTimeout) * time.Second
	}

	mcpTransport := mcp.NewHTTPTransport(mcpServer, timeout, logger)
	mux := http.NewServeMux()
	var healthCheckHandler http.HandlerFunc
	if appInstance, ok := ctx.Value("appInstance").(*serviceinit.App[rostering_config.IRosteringConfigurator]); ok && appInstance.HealthCheckFunc != nil {
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

	go func() {
		l := configManager.GetCurrentLogger()
		l.Info("Starting Data Server MCP server", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("HTTP server failed", "error", err)
			errChan <- err
		}
	}()

	cleanupFunc = func() {
		cl := configManager.GetCurrentLogger()
		cl.Info("Shutting down data-server...")

		// 停止 HTTP 服务器
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			cl.Error("Error stopping HTTP server", "error", err)
		} else {
			cl.Info("HTTP server shut down successfully.")
		}

		// 停止 MCP
		if err := mcpServer.Stop(ctx); err != nil {
			cl.Error("Error stopping MCP server", "error", err)
		}

		cl.Info("data-server shutdown complete")
	}

	return cleanupFunc, errChan, nil
}
