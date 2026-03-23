package config

import (
	"encoding/json"
	"fmt"

	common_config "jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	"github.com/spf13/viper"
)

// 排班约束默认值常量
const (
	// DefaultMaxDailyHours 默认每日最大工时
	DefaultMaxDailyHours = 12.0
	// DefaultMinRestHours 默认最小休息时间
	DefaultMinRestHours = 12.0
	// DefaultAllowOncallOverlap 默认允许待命班叠加
	DefaultAllowOncallOverlap = true
	// DefaultStrictTimeCheck 默认启用严格时间检查
	DefaultStrictTimeCheck = true
	// DefaultCheckConsecutiveShifts 默认检查连续班次
	DefaultCheckConsecutiveShifts = true
	// DefaultMaxConsecutiveDays 默认最大连续工作天数
	DefaultMaxConsecutiveDays = 6
)

type IRosteringConfigurator interface {
	common_config.IServiceConfigurator

	GetConfig() RosteringConfig
	GetOldConfig() (RosteringConfig, error)
	GetScheduleV3Config() ScheduleV3Config
}

// RosteringConfig 继承通用配置，并添加数据服务特有配置
type RosteringConfig struct {
	*common_config.Config `mapstructure:",squash"`

	// 服务特有配置
	Intent                IntentConfig                `json:"intent" yaml:"intent" mapstructure:"intent"`
	SchedulingAI          SchedulingAIConfig          `json:"schedulingAI" yaml:"schedulingAI" mapstructure:"schedulingAI"`
	SchedulingConstraints SchedulingConstraintsConfig `json:"scheduling_constraints" yaml:"scheduling_constraints" mapstructure:"scheduling_constraints"`
	ScheduleV3            ScheduleV3Config            `json:"schedule_v3" yaml:"schedule_v3" mapstructure:"schedule_v3"`
}

// IntentConfig 排班意图识别相关配置
type IntentConfig struct {
	// 通用配置
	MaxHistory  int    `json:"maxHistory" yaml:"maxHistory" mapstructure:"maxHistory"`
	FailureHint string `json:"failureHint" yaml:"failureHint" mapstructure:"failureHint"`

	// 意图识别策略配置
	// 策略类型: initial(初始对话), inWorkflow(工作流内), confirmation(确认阶段)
	Strategies map[string]IntentStrategyConfig `json:"strategies" yaml:"strategies" mapstructure:"strategies"`
}

// IntentStrategyConfig 意图识别策略配置
type IntentStrategyConfig struct {
	// 系统提示词
	SystemPrompt string `json:"systemPrompt" yaml:"systemPrompt" mapstructure:"systemPrompt"`
	// 用户提示词模板（可选）
	UserPromptTemplate string `json:"userPromptTemplate,omitempty" yaml:"userPromptTemplate,omitempty" mapstructure:"userPromptTemplate"`
	// 专用AI模型配置（可选，不配置则使用默认模型）
	Model *common_config.AIModelProvider `json:"model,omitempty" yaml:"model,omitempty" mapstructure:"model"`
	// 策略描述
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`
}

// SchedulingAIConfig 排班AI决策相关配置
type SchedulingAIConfig struct {
	SystemPrompt        string `json:"systemPrompt" yaml:"systemPrompt" mapstructure:"systemPrompt"`
	TimeoutSeconds      int    `json:"timeoutSeconds" yaml:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	EnableDraftForStep5 bool   `json:"enableDraftForStep5" yaml:"enableDraftForStep5" mapstructure:"enableDraftForStep5"`

	// TaskModels 为不同AI排班任务配置专用模型（可选）
	// 键为任务名称：staffPreSelection, scheduleDraft, ruleSorting
	TaskModels map[string]common_config.AIModelProvider `json:"taskModels,omitempty" yaml:"taskModels,omitempty" mapstructure:"taskModels"`
}

// SchedulingConstraintsConfig 排班约束配置
type SchedulingConstraintsConfig struct {
	MaxDailyHours          float64 `json:"max_daily_hours" yaml:"max_daily_hours" mapstructure:"max_daily_hours"`
	MinRestHours           float64 `json:"min_rest_hours" yaml:"min_rest_hours" mapstructure:"min_rest_hours"`
	AllowOncallOverlap     bool    `json:"allow_oncall_overlap" yaml:"allow_oncall_overlap" mapstructure:"allow_oncall_overlap"`
	StrictTimeCheck        bool    `json:"strict_time_check" yaml:"strict_time_check" mapstructure:"strict_time_check"`
	CheckConsecutiveShifts bool    `json:"check_consecutive_shifts" yaml:"check_consecutive_shifts" mapstructure:"check_consecutive_shifts"`
}

// ScheduleV3RetryConfig V3排班单班次重试配置
type ScheduleV3RetryConfig struct {
	// MaxShiftRetries 单班次最大自动重试次数（默认3次）
	MaxShiftRetries int `json:"max_shift_retries" yaml:"max_shift_retries" mapstructure:"max_shift_retries"`
	// EnableAIAnalysis 是否启用AI失败分析（默认true）
	EnableAIAnalysis bool `json:"enable_ai_analysis" yaml:"enable_ai_analysis" mapstructure:"enable_ai_analysis"`
	// StopOnAllFailed 所有班次都失败时是否立即停止任务（默认false）
	StopOnAllFailed bool `json:"stop_on_all_failed" yaml:"stop_on_all_failed" mapstructure:"stop_on_all_failed"`
	// SemanticHistoryFormat 历史记录格式（brief/detailed，默认brief）
	SemanticHistoryFormat string `json:"semantic_history_format" yaml:"semantic_history_format" mapstructure:"semantic_history_format"`
}

// ScheduleV3Config V3排班相关配置
type ScheduleV3Config struct {
	// Retry 重试相关配置
	Retry ScheduleV3RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`

	// GlobalReview 全局评审配置
	GlobalReview GlobalReviewConfig `json:"globalReview" yaml:"globalReview" mapstructure:"globalReview"`

	// TaskModels 任务模型配置 (任务类型 -> 模型名称)
	TaskModels map[string]string `json:"taskModels" yaml:"taskModels" mapstructure:"taskModels"`
}

// GlobalReviewConfig 全局评审配置
type GlobalReviewConfig struct {
	// Enabled 是否启用全局评审
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// MaxDebateRounds 最大对论轮次（默认3）
	MaxDebateRounds int `json:"maxDebateRounds" yaml:"maxDebateRounds" mapstructure:"maxDebateRounds"`

	// SkipOnError 评审出错时是否跳过（默认true）
	SkipOnError bool `json:"skipOnError" yaml:"skipOnError" mapstructure:"skipOnError"`
}

type rosteringConfigurator struct {
	logger    logging.ILogger
	oldConfig *RosteringConfig
	config    *RosteringConfig
}

func NewRosteringConfigurator(logger logging.ILogger) IRosteringConfigurator {
	return &rosteringConfigurator{
		logger: logger,
	}
}

// Load 加载 data-server 配置
func Load(path, serviceName string) (*RosteringConfig, error) {
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

	// 4. 初始化 RelationalGraphConfig 并赋予通用配置的值
	cfg := RosteringConfig{
		Config: commonConfig,
	}

	// 5. 解析服务专用配置
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// 实现 IServiceConfigurator 接口
func (c *rosteringConfigurator) GetLogConfig() logging.Config {
	if c.config == nil || c.config.Config == nil || c.config.Config.Log == nil {
		return logging.Config{}
	}
	return *c.config.Config.Log
}

func (c *rosteringConfigurator) GetOldBaseConfig() (common_config.Config, error) {
	if c.oldConfig == nil || c.oldConfig.Config == nil {
		return common_config.Config{}, fmt.Errorf("old config or its base part is nil")
	}
	return *c.oldConfig.Config, nil
}

func (c *rosteringConfigurator) GetBaseConfig() common_config.Config {
	if c.config == nil || c.config.Config == nil {
		return common_config.Config{}
	}
	return *c.config.Config
}

func (c *rosteringConfigurator) GetHTTPPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.HTTPPort
}

func (c *rosteringConfigurator) GetGRPCPort() int {
	if c.config == nil || c.config.Config == nil || c.config.Config.Ports == nil {
		return 0
	}
	return c.config.Config.Ports.GRPCPort
}

func (c *rosteringConfigurator) GetHost() string {
	if c.config == nil || c.config.Config == nil {
		return "127.0.0.1"
	}
	return c.config.Config.Host
}

func (c *rosteringConfigurator) LoadConfig(path string, serviceName string) error {
	c.oldConfig = c.config
	cfg, err := Load(path, serviceName)
	if err != nil {
		return err
	}
	c.config = cfg
	return nil
}

func (c *rosteringConfigurator) Raw() string {
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
func (c *rosteringConfigurator) GetConfig() RosteringConfig {
	if c.config == nil {
		return RosteringConfig{}
	}
	return *c.config
}

func (c *rosteringConfigurator) GetOldConfig() (RosteringConfig, error) {
	if c.oldConfig == nil {
		return RosteringConfig{}, fmt.Errorf("old config is nil")
	}
	return *c.oldConfig, nil
}

func (c *rosteringConfigurator) GetScheduleV3Config() ScheduleV3Config {
	if c.config == nil {
		return ScheduleV3Config{
			GlobalReview: GlobalReviewConfig{
				Enabled:         true,
				MaxDebateRounds: 3,
				SkipOnError:     true,
			},
		}
	}
	// 设置默认值
	cfg := c.config.ScheduleV3
	if cfg.GlobalReview.MaxDebateRounds <= 0 {
		cfg.GlobalReview.MaxDebateRounds = 3
	}
	return cfg
}
