package config

import (
	"encoding/json"
	"fmt"

	common_config "jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	"github.com/spf13/viper"
)

type IRosteringConfigurator interface {
	common_config.IServiceConfigurator

	GetConfig() DataRosteringConfig
	GetOldConfig() (DataRosteringConfig, error)

	GetToolConfig() ToolsConfig
}

// DataRosteringConfig 继承通用配置，并添加排班管理特有配置
type DataRosteringConfig struct {
	*common_config.Config `mapstructure:",squash"`

	// 服务特有配置
	Tools             ToolsConfig              `mapstructure:"tools" yaml:"tools"`
	ManagementService *ManagementServiceConfig `mapstructure:"management_service" yaml:"management_service"`
}

// ManagementServiceConfig management-service 配置
type ManagementServiceConfig struct {
	BaseURL string `mapstructure:"base_url" yaml:"base_url"` // 静态配置的 URL，如果为空则使用服务发现
}

type ToolsConfig struct {
	EnabledTools []string         `mapstructure:"enabled_tools" yaml:"enabled_tools"`
	Dangerous    DangerousOptions `mapstructure:"dangerous" yaml:"dangerous"`
}

// DangerousOptions 用于控制危险操作（如删表、清空表）
type DangerousOptions struct {
	Enabled         bool     `mapstructure:"enabled" yaml:"enabled"`                   // 是否允许任何危险操作（默认 false）
	RequirePasscode bool     `mapstructure:"require_passcode" yaml:"require_passcode"` // 是否需要口令
	Passcode        string   `mapstructure:"passcode" yaml:"passcode"`                 // 口令（可通过环境变量或配置中心注入）
	AllowedActions  []string `mapstructure:"allowed_actions" yaml:"allowed_actions"`   // 允许的操作集合：drop_table, truncate_table, delete_all
	AllowedTables   []string `mapstructure:"allowed_tables" yaml:"allowed_tables"`     // 允许操作的表名白名单
}

// dataRosteringConfigurator 配置器
type dataRosteringConfigurator struct {
	logger    logging.ILogger
	oldConfig *DataRosteringConfig
	config    *DataRosteringConfig
}

func NewDataRosteringConfigurator(logger logging.ILogger) IRosteringConfigurator {
	return &dataRosteringConfigurator{
		logger: logger,
	}
}

// Load 加载 data-rostering 配置
func Load(path, serviceName string) (*DataRosteringConfig, error) {
	// 加载通用配置
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

	cfg := DataRosteringConfig{
		Config: commonConfig,
		Tools: ToolsConfig{
			EnabledTools: []string{},
			Dangerous: DangerousOptions{
				Enabled:         false,
				RequirePasscode: true,
				Passcode:        "", // 默认空，需外部配置
				AllowedActions:  []string{"truncate_table"},
				AllowedTables:   []string{"schedules"},
			},
		},
	}

	// 5. 解析服务专用配置
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 此处可按需补充对 serviceName.yaml 的解析；为减少外部依赖，暂不强制读取
	return &cfg, nil
}

// 实现 IServiceConfigurator 接口
func (c *dataRosteringConfigurator) GetLogConfig() logging.Config {
	if c.config == nil || c.config.Config == nil || c.config.Config.Log == nil {
		return logging.Config{}
	}
	return *c.config.Config.Log
}

func (c *dataRosteringConfigurator) GetOldBaseConfig() (common_config.Config, error) {
	if c.oldConfig == nil || c.oldConfig.Config == nil {
		return common_config.Config{}, fmt.Errorf("old config or its base part is nil")
	}
	return *c.oldConfig.Config, nil
}

func (c *dataRosteringConfigurator) GetBaseConfig() common_config.Config {
	if c.config == nil || c.config.Config == nil {
		return common_config.Config{}
	}
	return *c.config.Config
}

func (c *dataRosteringConfigurator) GetHTTPPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.HTTPPort
}

func (c *dataRosteringConfigurator) GetGRPCPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.GRPCPort
}

func (c *dataRosteringConfigurator) GetHost() string {
	if c.config == nil || c.config.Config == nil {
		return "127.0.0.1"
	}
	return c.config.Config.Host
}

func (c *dataRosteringConfigurator) LoadConfig(path string, serviceName string) error {
	c.oldConfig = c.config
	cfg, err := Load(path, serviceName)
	if err != nil {
		return err
	}
	c.config = cfg
	return nil
}

func (c *dataRosteringConfigurator) Raw() string {
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
func (c *dataRosteringConfigurator) GetConfig() DataRosteringConfig {
	if c.config == nil {
		return DataRosteringConfig{}
	}
	return *c.config
}

func (c *dataRosteringConfigurator) GetOldConfig() (DataRosteringConfig, error) {
	if c.oldConfig == nil {
		return DataRosteringConfig{}, fmt.Errorf("old config is nil")
	}
	return *c.oldConfig, nil
}

func (c *dataRosteringConfigurator) GetToolConfig() ToolsConfig {
	if c.config == nil {
		return ToolsConfig{}
	}
	return c.config.Tools
}
