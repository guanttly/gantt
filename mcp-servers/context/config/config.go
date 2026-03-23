package config

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"

	"jusha/mcp/pkg/logging"

	common_config "jusha/mcp/pkg/config"
)

type IContextConfigurator interface {
	common_config.IServiceConfigurator

	GetConfig() ContextConConfig
	GetOldConfig() (ContextConConfig, error)
}

// ContextConConfig 继承通用配置，并添加上下文服务特有配置
type ContextConConfig struct {
	*common_config.Config `mapstructure:",squash"`

	// 服务特有配置
	Tools ToolsConfig `mapstructure:"tools" yaml:"tools"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	EnabledTools []string         `mapstructure:"enabled_tools" yaml:"enabled_tools"`
	Dangerous    DangerousOptions `mapstructure:"dangerous" yaml:"dangerous"`
}

// DangerousOptions 用于控制危险操作
type DangerousOptions struct {
	Enabled         bool     `mapstructure:"enabled" yaml:"enabled"`
	RequirePasscode bool     `mapstructure:"require_passcode" yaml:"require_passcode"`
	Passcode        string   `mapstructure:"passcode" yaml:"passcode"`
	AllowedActions  []string `mapstructure:"allowed_actions" yaml:"allowed_actions"`
	AllowedTables   []string `mapstructure:"allowed_tables" yaml:"allowed_tables"`
}

// managementServiceConfigurator 配置器
type managementServiceConfigurator struct {
	logger    logging.ILogger
	oldConfig *ContextConConfig
	config    *ContextConConfig
}

func NewContextConConfigurator(logger logging.ILogger) IContextConfigurator {
	return &managementServiceConfigurator{
		logger: logger,
	}
}

// Load 加载 management-service 配置
func Load(path, serviceName string) (*ContextConConfig, error) {
	// 1. 加载通用配置
	commonConfig, err := common_config.Load(path)
	if err != nil {
		fmt.Printf("Failed to load common config: %v\n", err)
		// 创建默认的通用配置，确保所有重要字段都有默认值
		commonConfig = common_config.CreateEmptyConfig()
	}

	// 2. 创建服务专用的 Viper 实例
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName(serviceName)
	v.SetConfigType("yaml")

	// 3. 读取特定于服务的配置文件（可选）
	_ = v.ReadInConfig()

	cfg := ContextConConfig{
		Config: commonConfig,
		Tools: ToolsConfig{
			EnabledTools: []string{},
			Dangerous: DangerousOptions{
				Enabled:         false,
				RequirePasscode: true,
				Passcode:        "",
				AllowedActions:  []string{},
				AllowedTables:   []string{},
			},
		},
	}

	// 5. 解析服务专用配置
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// 实现 IServiceConfigurator 接口
func (c *managementServiceConfigurator) GetLogConfig() logging.Config {
	if c.config == nil || c.config.Config == nil || c.config.Config.Log == nil {
		return logging.Config{}
	}
	return *c.config.Config.Log
}

func (c *managementServiceConfigurator) GetOldBaseConfig() (common_config.Config, error) {
	if c.oldConfig == nil || c.oldConfig.Config == nil {
		return common_config.Config{}, fmt.Errorf("old config or its base part is nil")
	}
	return *c.oldConfig.Config, nil
}

func (c *managementServiceConfigurator) GetBaseConfig() common_config.Config {
	if c.config == nil || c.config.Config == nil {
		return common_config.Config{}
	}
	return *c.config.Config
}

func (c *managementServiceConfigurator) GetHTTPPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.HTTPPort
}

func (c *managementServiceConfigurator) GetGRPCPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.GRPCPort
}

func (c *managementServiceConfigurator) GetHost() string {
	if c.config == nil || c.config.Config == nil {
		return "127.0.0.1"
	}
	return c.config.Config.Host
}

func (c *managementServiceConfigurator) LoadConfig(path string, serviceName string) error {
	c.oldConfig = c.config
	cfg, err := Load(path, serviceName)
	if err != nil {
		return err
	}
	c.config = cfg
	return nil
}

func (c *managementServiceConfigurator) Raw() string {
	if c.config == nil {
		return ""
	}
	b, err := json.Marshal(c.config)
	if err != nil {
		return ""
	}
	return string(b)
}

// Service-specific accessors
func (c *managementServiceConfigurator) GetConfig() ContextConConfig {
	if c.config == nil {
		return ContextConConfig{}
	}
	return *c.config
}

func (c *managementServiceConfigurator) GetOldConfig() (ContextConConfig, error) {
	if c.oldConfig == nil {
		return ContextConConfig{}, fmt.Errorf("old config is nil")
	}
	return *c.oldConfig, nil
}
