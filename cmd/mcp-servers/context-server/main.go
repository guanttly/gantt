// cmd/mcp-servers/context-server/main.go
package main

import (
	"context"
	"log/slog"
	"strings"

	"jusha/agent/server/context/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/serviceinit"

	mcp_context "jusha/agent/server/context"
)

const (
	serviceName       = "context-server"
	configDir         = "./config/"
	nacosDataId       = "context-server.yml"
	nacosCommonDataId = "common.yml"
)

func main() {
	app := serviceinit.App[config.IContextConfigurator]{
		ServiceName:       serviceName,
		ConfigDir:         configDir,
		NacosDataID:       nacosDataId,
		NacosCommonDataID: nacosCommonDataId,
		LoadServiceConfig: func(dir string) (config.IContextConfigurator, error) {
			configurator := config.NewContextConConfigurator(slog.Default())
			if err := configurator.LoadConfig(dir, serviceName); err != nil {
				return nil, err
			}
			return configurator, nil
		},
		SetupDependenciesAndRun:   mcp_context.SetupDependenciesAndRun,
		HandleServiceConfigChange: HandleServiceConfigChange,
		HealthCheckFunc:           nil,
	}
	app.Run()
}

// HandleServiceConfigChange Context Server 的配置变更处理
func HandleServiceConfigChange(
	ctx context.Context,
	configurator config.IContextConfigurator,
	currentLogger logging.ILogger,
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger) {
	logger := currentLogger
	logger.Info("Handling Context Server configuration change...")

	oldConfig, err := configurator.GetOldConfig()
	if err != nil {
		logger.Info("No old config to compare, considering as first-time config load")
		return true, []string{"first-time config load"}, logger
	}

	newConfig := configurator.GetConfig()
	restartNeeded = false
	reasons = []string{}

	// 比较端口配置
	if oldConfig.Config != nil && newConfig.Config != nil {
		if oldConfig.Config.Ports != nil && newConfig.Config.Ports != nil {
			if oldConfig.Config.Ports.HTTPPort != newConfig.Config.Ports.HTTPPort {
				restartNeeded = true
				reasons = append(reasons, "HTTP port changed")
			}
		}
	}

	// 比较工具配置
	if oldConfig.Config != nil && newConfig.Config != nil {
		// 注意：这里需要根据实际的配置结构来比较
		// 如果配置中有 Tools 字段，需要比较
	}

	if restartNeeded {
		logger.Info("Configuration changes require service restart",
			"reasons", strings.Join(reasons, ", "))
	} else {
		logger.Info("Configuration changes do not require restart")
	}

	return restartNeeded, reasons, logger
}
