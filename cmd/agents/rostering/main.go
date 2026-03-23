// cmd/mcp-servers/data-server/main.go
package main

import (
	"context"
	"log/slog"
	"strings"

	"jusha/agent/rostering"
	"jusha/agent/rostering/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/serviceinit"
)

const (
	serviceName       = "rostering-agent"
	configDir         = "./config/"
	nacosDataId       = "rostering-agent.yml"
	nacosCommonDataId = "common.yml"
)

func main() {
	app := serviceinit.App[config.IRosteringConfigurator]{
		ServiceName:       serviceName,
		ConfigDir:         configDir,
		NacosDataID:       nacosDataId,
		NacosCommonDataID: nacosCommonDataId,
		LoadServiceConfig: func(dir string) (config.IRosteringConfigurator, error) {
			configurator := config.NewRosteringConfigurator(slog.Default())
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

// HandleServiceConfigChange 数据服务的配置变更处理
func HandleServiceConfigChange(
	ctx context.Context,
	configurator config.IRosteringConfigurator,
	currentLogger logging.ILogger,
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger) {
	logger := currentLogger
	logger.Info("Handling Scheduling Service configuration change...")

	if !restartNeeded {
		logger.Info("No significant configuration changes detected for Scheduling Service.")
	} else {
		logger.Warn("Configuration changes require service restart for Scheduling Service.", "reasons", strings.Join(reasons, ", "))
	}
	return restartNeeded, reasons, nil
}
