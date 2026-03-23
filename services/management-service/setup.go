package management

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/gantt/service/management/config"
	"jusha/gantt/service/management/internal/wiring"
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"

	http_port "jusha/gantt/service/management/internal/port/http"
	pkg_config "jusha/mcp/pkg/config"
)

// SetupDependenciesAndRun 组装依赖并启动 HTTP 服务
func SetupDependenciesAndRun(
	ctx context.Context,
	configurator config.IManagementServiceConfigurator,
	initialLogger logging.ILogger,
	configManager pkg_config.IConfigurationManager[config.IManagementServiceConfigurator],
	registrar discovery.IRegistrar,
	namingClient naming_client.INamingClient,
) (cleanupFunc func(), errChan chan error, err error) {
	logger := initialLogger
	if configManager != nil {
		logger = configManager.GetCurrentLogger()
	}

	cfg := configurator.GetBaseConfig()

	logger.Info("Management service configuration loaded",
		"server_port", cfg.Ports.HTTPPort,
		"discovery_enabled", cfg.Discovery != nil && cfg.Discovery.Enabled,
	)

	errChan = make(chan error, 1)

	// 1) 构建服务 Provider（内部会完成依赖注入容器初始化、仓储/服务等装配）
	container, err := wiring.NewContainer(ctx, logger, configurator)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container: %w", err)
	}

	// 2) 构建 HTTP 路由
	muxHandler := http_port.NewHTTPHandler(container, logger)

	// 3) 创建 HTTP 服务器
	readTimeout := time.Duration(cfg.Timeout.ReadTimeout) * time.Second
	writeTimeout := time.Duration(cfg.Timeout.WriteTimeout) * time.Second
	serverAddr := fmt.Sprintf(":%d", cfg.Ports.HTTPPort)

	httpServer := &http.Server{
		Addr:         serverAddr,
		Handler:      muxHandler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// 4) 启动服务器
	go func() {
		logger.Info("Starting HTTP server", "addr", serverAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// 5) 注册到服务发现
	if cfg.Discovery != nil && cfg.Discovery.Enabled && registrar != nil {
		// 服务实例信息
		serviceName := "management-service"
		host := cfg.Host
		if host == "" {
			host = "127.0.0.1" // 默认本地地址
		}
		port := cfg.Ports.HTTPPort
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

	// 6) 清理函数
	cleanupFunc = func() {
		logger.Info("Shutting down management service...")

		// 停止 HTTP 服务器
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown HTTP server", "error", err)
		}

		// 注销服务
		if cfg.Discovery != nil && cfg.Discovery.Enabled && registrar != nil {
			serviceName := "management-service"
			host := cfg.Host
			if host == "" {
				host = "127.0.0.1"
			}
			port := cfg.Ports.HTTPPort

			if err := registrar.Deregister(serviceName, host, port); err != nil {
				logger.Warn("Failed to deregister service", "error", err)
			}
		}

		// 关闭容器资源
		if err := container.Close(); err != nil {
			logger.Error("Failed to close container", "error", err)
		}

		logger.Info("Management service shutdown complete")
	}

	return cleanupFunc, errChan, nil
}
