// cmd/mcp-servers/rostering-server/main.go
package main

import (
	"context"
	"jusha/mcp/pkg/logging"
	"log/slog"
	"reflect"
	"strings"

	"jusha/gantt/mcp/rostering"
	"jusha/gantt/mcp/rostering/config"
	"jusha/mcp/pkg/serviceinit"
)

const (
	serviceName       = "rostering-server"
	configDir         = "./config/"
	nacosDataId       = "rostering-server.yml"
	nacosCommonDataId = "common.yml"
)

func main() {
	app := serviceinit.App[config.IRosteringConfigurator]{
		ServiceName:       serviceName,
		ConfigDir:         configDir,
		NacosDataID:       nacosDataId,
		NacosCommonDataID: nacosCommonDataId,
		LoadServiceConfig: func(dir string) (config.IRosteringConfigurator, error) {
			configurator := config.NewDataRosteringConfigurator(slog.Default())
			if err := configurator.LoadConfig(dir, serviceName); err != nil {
				return nil, err
			}
			return configurator, nil
		},
		SetupDependenciesAndRun:   rostering.SetupDependenciesAndRun,
		HandleServiceConfigChange: HandleServiceConfigChange,
		HealthCheckFunc:           nil,
	}
	app.Run()
}

// HandleServiceConfigChange 排班服务的配置变更处理
func HandleServiceConfigChange(
	ctx context.Context,
	configurator config.IRosteringConfigurator,
	currentLogger logging.ILogger,
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger) {
	logger := currentLogger
	logger.Info("Handling Rostering Server configuration change...")

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

	// 比较 Management Service 配置
	if !reflect.DeepEqual(oldConfig.ManagementService, newConfig.ManagementService) {
		restartNeeded = true
		reasons = append(reasons, "management service config changed")
	}

	// 比较工具配置
	if !reflect.DeepEqual(oldConfig.Tools, newConfig.Tools) {
		restartNeeded = true
		reasons = append(reasons, "tools config changed")
	}

	if restartNeeded {
		logger.Info("Configuration changes require service restart",
			"reasons", strings.Join(reasons, ", "))
	} else {
		logger.Info("Configuration changes do not require restart")
	}

	return restartNeeded, reasons, logger
}
