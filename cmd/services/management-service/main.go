// cmd/services/management-service/main.go
package main

import (
	"context"
	"jusha/gantt/service/management"
	"jusha/gantt/service/management/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/serviceinit"
	"log/slog"
	"reflect"
	"strings"
)

const (
	serviceName       = "management-service"
	configDir         = "./config/"
	nacosDataId       = "management-service.yml"
	nacosCommonDataId = "common.yml"
)

func main() {
	app := serviceinit.App[config.IManagementServiceConfigurator]{
		ServiceName:       serviceName,
		ConfigDir:         configDir,
		NacosDataID:       nacosDataId,
		NacosCommonDataID: nacosCommonDataId,
		LoadServiceConfig: func(dir string) (config.IManagementServiceConfigurator, error) {
			configurator := config.NewManagementServiceConfigurator(slog.Default())
			if err := configurator.LoadConfig(dir, serviceName); err != nil {
				return nil, err
			}
			return configurator, nil
		},
		SetupDependenciesAndRun:   management.SetupDependenciesAndRun,
		HandleServiceConfigChange: HandleServiceConfigChange,
		HealthCheckFunc:           nil,
	}
	app.Run()
}

// HandleServiceConfigChange 管理服务的配置变更处理
func HandleServiceConfigChange(
	ctx context.Context,
	configurator config.IManagementServiceConfigurator,
	currentLogger logging.ILogger,
) (restartNeeded bool, reasons []string, serviceSpecificLogger logging.ILogger) {
	logger := currentLogger
	logger.Info("Handling Management Service configuration change...")

	oldConfig, err := configurator.GetOldConfig()
	if err != nil {
		logger.Error("Failed to get old config during hot-reload", "error", err)
		return false, nil, nil
	}
	newConfig := configurator.GetConfig()

	// 若数据库连接信息变化，需重启
	if oldConfig.Config != nil && oldConfig.Config.Database != nil &&
		newConfig.Config != nil && newConfig.Config.Database != nil {
		if !reflect.DeepEqual(oldConfig.Config.Database.MySQL, newConfig.Config.Database.MySQL) {
			restartNeeded = true
			reasons = append(reasons, "MySQL config changed")
		}
	}

	if !restartNeeded {
		logger.Info("No significant configuration changes detected for Management Service.")
	} else {
		logger.Warn("Configuration changes require service restart for Management Service.", "reasons", strings.Join(reasons, ", "))
	}
	return restartNeeded, reasons, nil
}
