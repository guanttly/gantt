package config

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"

	"jusha/mcp/pkg/logging"

	common_config "jusha/mcp/pkg/config"
)

type IManagementServiceConfigurator interface {
	common_config.IServiceConfigurator

	GetConfig() ManagementServiceConfig
	GetOldConfig() (ManagementServiceConfig, error)
	GetStaffingConfig() StaffingConfig
}

// StaffingConfig 排班人数计算配置
type StaffingConfig struct {
	DefaultAvgReportLimit int `mapstructure:"default_avg_report_limit" yaml:"default_avg_report_limit"` // 默认人均报告处理上限
}

// ManagementServiceConfig 继承通用配置，并添加管理服务特有配置
type ManagementServiceConfig struct {
	*common_config.Config `mapstructure:",squash"`

	// 服务特有配置
	Staffing *StaffingConfig `mapstructure:"staffing" yaml:"staffing"` // 排班人数计算配置
}

// managementServiceConfigurator 配置器
type managementServiceConfigurator struct {
	logger    logging.ILogger
	oldConfig *ManagementServiceConfig
	config    *ManagementServiceConfig
}

func NewManagementServiceConfigurator(logger logging.ILogger) IManagementServiceConfigurator {
	return &managementServiceConfigurator{
		logger: logger,
	}
}

// Load 加载 management-service 配置
func Load(path, serviceName string) (*ManagementServiceConfig, error) {
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

	cfg := ManagementServiceConfig{
		Config: commonConfig,
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
func (c *managementServiceConfigurator) GetConfig() ManagementServiceConfig {
	if c.config == nil {
		return ManagementServiceConfig{}
	}
	return *c.config
}

func (c *managementServiceConfigurator) GetOldConfig() (ManagementServiceConfig, error) {
	if c.oldConfig == nil {
		return ManagementServiceConfig{}, fmt.Errorf("old config is nil")
	}
	return *c.oldConfig, nil
}

// GetStaffingConfig 获取排班人数计算配置
func (c *managementServiceConfigurator) GetStaffingConfig() StaffingConfig {
	if c.config == nil || c.config.Staffing == nil {
		return StaffingConfig{
			DefaultAvgReportLimit: 50, // 默认值
		}
	}
	return *c.config.Staffing
}
