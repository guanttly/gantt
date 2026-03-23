package config

import (
	"fmt"
	"os"
	"path/filepath"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"
)

// LoadInitialConfig loads the initial configuration for a service.
// It prioritizes Nacos, falls back to local, and can push local to Nacos.
// loadServiceSpecificConfig is a function like `config.Load(dir, serviceName)` from your specific service.
func LoadInitialConfig[C IServiceConfigurator](
	logger logging.ILogger,
	configDir string,
	serviceName string,
	nacosDataID string,
	nacosCommonDataID string,
	loadServiceSpecificConfig func(configDir string) (C, error),
) (cfg C, configSource string, err error) {
	configSource = "local"

	// 1. Load local config first to get Nacos connection details if needed.
	localCfg, loadErr := loadServiceSpecificConfig(configDir)
	if loadErr != nil {
		logger.Error("Failed to load initial local config, exiting", "error", loadErr, "path", configDir)
		return localCfg, configSource, fmt.Errorf("failed to load initial local config from %s: %w", configDir, loadErr)
	}
	cfg = localCfg // Default to local config

	discoveryCfg := cfg.GetBaseConfig().Discovery
	if !discoveryCfg.Enabled || discoveryCfg.Type != "nacos" {
		logger.Info("Nacos discovery not enabled or type is not 'nacos'. Using local configuration.", "source", configSource)
		return cfg, configSource, nil
	}

	logger.Info("Local base config loaded. Attempting to fetch latest config from Nacos...")
	nacosCfg := discoveryCfg.Nacos
	nacosGroup := nacosCfg.GroupName

	configClient, err := newNacosConfigClient(nacosCfg, logger)
	if err != nil {
		logger.Error("Failed to create Nacos Config Client. Using local configuration.", "error", err)
		return cfg, configSource, nil // Fallback to local
	}
	defer configClient.CloseClient()
	configManager := newNacosConfigManager(configClient, logger)

	logger.Info("Attempting to load service config from Nacos...", "dataId", nacosDataID, "group", nacosGroup)
	nacosServiceConfigContent, err := configManager.GetConfig(nacosDataID, nacosGroup)
	if err != nil {
		logger.Warn("Failed to get service config from Nacos. Will use local and attempt to push.", "error", err, "dataId", nacosDataID)
		PushLocalConfigToNacos(configManager, configDir, nacosDataID, nacosGroup, logger)
		// Continue with local config if common config also fails or is not present
	}

	nacosCommonConfigContent, err := configManager.GetConfig(nacosCommonDataID, nacosGroup)
	if err != nil {
		logger.Warn("Failed to get common config from Nacos. Will use local and attempt to push.", "error", err, "dataId", nacosCommonDataID)
		PushLocalConfigToNacos(configManager, configDir, nacosCommonDataID, nacosGroup, logger)
		// If service config also failed, we are already using localCfg.
		// If service config succeeded but common failed, we might have partial Nacos config.
		// For simplicity, if any Nacos fetch fails, we might prefer full local or full Nacos.
		// Here, if service config was fetched, but common failed, we might still try to use the fetched service config.
	}

	if nacosServiceConfigContent == "" {
		logger.Warn("Service config from Nacos is empty. Using local configuration.", "dataId", nacosDataID)
		PushLocalConfigToNacos(configManager, configDir, nacosDataID, nacosGroup, logger) // Push local if Nacos is empty
		PushLocalConfigToNacos(configManager, configDir, nacosCommonDataID, nacosGroup, logger)
		return cfg, configSource, nil // Use initially loaded local config
	}

	// Create a temporary directory to load Nacos configs
	tmpDir, err := os.MkdirTemp("", serviceName+"-nacos-init-")
	if err != nil {
		logger.Error("Failed to create temp dir for Nacos config. Using local configuration.", "error", err)
		return cfg, configSource, nil
	}
	defer os.RemoveAll(tmpDir)

	tmpServiceCfgPath := filepath.Join(tmpDir, nacosDataID) // Use Nacos Data ID as filename
	if err := os.WriteFile(tmpServiceCfgPath, []byte(nacosServiceConfigContent), 0644); err != nil {
		logger.Error("Failed to write Nacos service config to temp file. Using local configuration.", "error", err, "path", tmpServiceCfgPath)
		return cfg, configSource, nil
	}

	tmpCommonCfgPath := filepath.Join(tmpDir, nacosCommonDataID) // Use Nacos Common Data ID
	if nacosCommonConfigContent != "" {
		if err := os.WriteFile(tmpCommonCfgPath, []byte(nacosCommonConfigContent), 0644); err != nil {
			logger.Error("Failed to write Nacos common config to temp file. Using local configuration.", "error", err, "path", tmpCommonCfgPath)
			return cfg, configSource, nil
		}
	} else {
		// If common config is empty on Nacos, try to copy local common config to temp dir
		localCommonPath := filepath.Join(configDir, nacosCommonDataID)
		if _, statErr := os.Stat(localCommonPath); statErr == nil {
			if copyErr := utils.CopyFile(localCommonPath, tmpCommonCfgPath); copyErr != nil {
				logger.Error("Failed to copy local common config to temp dir. Using local configuration.", "error", copyErr)
				return cfg, configSource, nil
			}
		} else {
			logger.Warn("Nacos common config is empty, and local common config does not exist.", "path", localCommonPath)
			// It's possible a service doesn't rely on common.yml, or common.yml is truly empty.
			// Create an empty common file so viper doesn't complain if it expects it.
			if err := os.WriteFile(tmpCommonCfgPath, []byte(""), 0644); err != nil {
				logger.Error("Failed to write empty common config to temp file.", "error", err, "path", tmpCommonCfgPath)
				return cfg, configSource, nil
			}
		}
	}

	logger.Info("Attempting to load config from Nacos temp directory", "path", tmpDir)
	nacosLoadedCfg, err := loadServiceSpecificConfig(tmpDir) // serviceName is used by Load to find files like serviceName.yml
	if err != nil {
		logger.Error("Failed to parse Nacos config from temp dir. Using local configuration.", "error", err)
		// Optionally push local config to Nacos if Nacos version was bad
		PushLocalConfigToNacos(configManager, configDir, nacosDataID, nacosGroup, logger)
		PushLocalConfigToNacos(configManager, configDir, nacosCommonDataID, nacosGroup, logger)
		return cfg, configSource, nil
	}

	logger.Info("Successfully loaded configuration from Nacos.")
	cfg = nacosLoadedCfg
	configSource = "nacos"

	// Update local files with Nacos content as a backup
	UpdateLocalFileWithNacosContent(filepath.Join(configDir, nacosDataID), []byte(nacosServiceConfigContent), logger)
	if nacosCommonConfigContent != "" {
		UpdateLocalFileWithNacosContent(filepath.Join(configDir, nacosCommonDataID), []byte(nacosCommonConfigContent), logger)
	}

	return cfg, configSource, nil
}

// PushLocalConfigToNacos pushes a local configuration file to Nacos.
func PushLocalConfigToNacos(cm IConfigManager, localConfigDir, dataID, group string, logger logging.ILogger) {
	localPath := filepath.Join(localConfigDir, dataID)
	if _, statErr := os.Stat(localPath); statErr != nil {
		logger.Info("Local config file does not exist, skipping push to Nacos.", "path", localPath)
		return
	}

	localBytes, readErr := os.ReadFile(localPath)
	if readErr != nil {
		logger.Error("Failed to read local config file for Nacos push.", "error", readErr, "path", localPath)
		return
	}
	if len(localBytes) == 0 {
		logger.Warn("Local config file is empty, not pushing to Nacos.", "path", localPath)
		return
	}

	success, pushErr := cm.PublishConfig(dataID, group, string(localBytes))
	if pushErr != nil {
		logger.Error("Failed to push local config to Nacos.", "error", pushErr, "dataId", dataID, "group", group)
	} else if success {
		logger.Info("Successfully pushed local config to Nacos.", "dataId", dataID, "group", group)
	} else {
		logger.Warn("Push local config to Nacos call succeeded but Nacos returned false.", "dataId", dataID, "group", group)
	}
}

// UpdateLocalFileWithNacosContent updates a local file with content from Nacos.
func UpdateLocalFileWithNacosContent(filePath string, content []byte, logger logging.ILogger) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create directory for local config backup.", "error", err, "dir", dir)
		return
	}
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		logger.Error("Failed to update local config file from Nacos content.", "error", err, "path", filePath)
	} else {
		logger.Info("Successfully updated local config file with Nacos content.", "path", filePath)
	}
}
