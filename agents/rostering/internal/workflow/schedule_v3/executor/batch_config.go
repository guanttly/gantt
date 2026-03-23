package executor

import (
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 分批处理配置和数据结构
// 用于控制排班和校验时的人员批次大小，避免LLM处理能力超限
// ============================================================

const (
	// DefaultBatchSize 默认每批处理人数
	DefaultBatchSize = 5

	// DefaultMaxRetryPerBatch 单批最大重试次数
	DefaultMaxRetryPerBatch = 2

	// DefaultMaxFixRounds 最大修复轮次
	DefaultMaxFixRounds = 3
)

// BatchConfig 分批处理配置
type BatchConfig struct {
	// BatchSize 每批处理的人数上限
	BatchSize int `json:"batchSize" yaml:"batchSize"`

	// MaxRetryPerBatch 单批最大重试次数
	MaxRetryPerBatch int `json:"maxRetryPerBatch" yaml:"maxRetryPerBatch"`

	// ContinueOnError 单批失败是否继续处理后续批次
	ContinueOnError bool `json:"continueOnError" yaml:"continueOnError"`

	// MaxFixRounds 全部校验完成后LLM4修复的最大轮次
	MaxFixRounds int `json:"maxFixRounds" yaml:"maxFixRounds"`

	// EnableProgressNotify 是否启用批次进度通知
	EnableProgressNotify bool `json:"enableProgressNotify" yaml:"enableProgressNotify"`
}

// DefaultBatchConfig 返回默认分批配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		BatchSize:            DefaultBatchSize,
		MaxRetryPerBatch:     DefaultMaxRetryPerBatch,
		ContinueOnError:      true,
		MaxFixRounds:         DefaultMaxFixRounds,
		EnableProgressNotify: true,
	}
}

// Validate 验证配置有效性
func (c *BatchConfig) Validate() error {
	if c.BatchSize < 1 {
		return fmt.Errorf("batchSize must be at least 1, got %d", c.BatchSize)
	}
	if c.BatchSize > 20 {
		return fmt.Errorf("batchSize should not exceed 20 to ensure LLM quality, got %d", c.BatchSize)
	}
	if c.MaxRetryPerBatch < 0 {
		c.MaxRetryPerBatch = 0
	}
	if c.MaxFixRounds < 1 {
		c.MaxFixRounds = 1
	}
	return nil
}

// ============================================================
// 分批校验相关结构
// ============================================================

// BatchValidationContext 批次校验上下文
type BatchValidationContext struct {
	// 基本信息
	ShiftID   string `json:"shiftId"`
	ShiftName string `json:"shiftName"`

	// 人员统计
	TotalStaff     int      `json:"totalStaff"`     // 总人数
	ProcessedStaff []string `json:"processedStaff"` // 已处理人员ID列表

	// 批次信息
	CurrentBatch int `json:"currentBatch"` // 当前批次号（从1开始）
	TotalBatches int `json:"totalBatches"` // 总批次数

	// 结果收集
	AllViolations []*d_model.ValidationIssue `json:"allViolations"` // 累积的所有违规问题
	BatchResults  []*BatchValidationResult   `json:"batchResults"`  // 各批次结果

	// 状态
	AllPassed bool    `json:"allPassed"` // 所有批次是否都通过
	TotalTime float64 `json:"totalTime"` // 总执行时间（秒）
}

// BatchValidationResult 单批次校验结果
type BatchValidationResult struct {
	BatchIndex    int                        `json:"batchIndex"`    // 批次序号（从1开始）
	StaffIDs      []string                   `json:"staffIds"`      // 本批人员ID列表
	StaffNames    []string                   `json:"staffNames"`    // 本批人员姓名列表
	Passed        bool                       `json:"passed"`        // 是否通过
	Violations    []*d_model.ValidationIssue `json:"violations"`    // 违规问题
	ExecutionTime float64                    `json:"executionTime"` // 执行耗时（秒）
	RetryCount    int                        `json:"retryCount"`    // 重试次数
	ErrorMessage  string                     `json:"errorMessage"`  // 错误信息（如有）
}

// NewBatchValidationContext 创建批次校验上下文
func NewBatchValidationContext(shiftID, shiftName string, totalStaff, totalBatches int) *BatchValidationContext {
	return &BatchValidationContext{
		ShiftID:        shiftID,
		ShiftName:      shiftName,
		TotalStaff:     totalStaff,
		TotalBatches:   totalBatches,
		CurrentBatch:   0,
		ProcessedStaff: make([]string, 0),
		AllViolations:  make([]*d_model.ValidationIssue, 0),
		BatchResults:   make([]*BatchValidationResult, 0),
		AllPassed:      true,
	}
}

// AddBatchResult 添加批次结果
func (c *BatchValidationContext) AddBatchResult(result *BatchValidationResult) {
	c.BatchResults = append(c.BatchResults, result)
	c.ProcessedStaff = append(c.ProcessedStaff, result.StaffIDs...)
	c.AllViolations = append(c.AllViolations, result.Violations...)
	if !result.Passed {
		c.AllPassed = false
	}
}

// GetProgress 获取进度百分比
func (c *BatchValidationContext) GetProgress() float64 {
	if c.TotalBatches == 0 {
		return 100.0
	}
	return float64(c.CurrentBatch) / float64(c.TotalBatches) * 100
}

// GetSummary 获取校验摘要
func (c *BatchValidationContext) GetSummary() string {
	if c.AllPassed {
		return fmt.Sprintf("校验完成：%d人共%d批次，全部通过", c.TotalStaff, c.TotalBatches)
	}
	return fmt.Sprintf("校验完成：%d人共%d批次，发现%d个违规问题",
		c.TotalStaff, c.TotalBatches, len(c.AllViolations))
}

// ============================================================
// 分批排班相关结构
// ============================================================

// BatchSchedulingContext 批次排班上下文
type BatchSchedulingContext struct {
	// 基本信息
	ShiftID    string `json:"shiftId"`
	ShiftName  string `json:"shiftName"`
	TargetDate string `json:"targetDate"`

	// 需求与进度
	TotalRequired  int      `json:"totalRequired"`  // 总需求人数
	ScheduledStaff []string `json:"scheduledStaff"` // 已安排人员ID列表

	// 批次信息
	CurrentBatch int `json:"currentBatch"` // 当前批次号（从1开始）
	TotalBatches int `json:"totalBatches"` // 预计总批次数

	// 候选人池
	RemainingCandidates []*d_model.Employee `json:"-"`                // 剩余候选人（不序列化）
	ExcludedStaffIDs    []string            `json:"excludedStaffIds"` // 被排除的人员ID（因规则冲突等）

	// 结果收集
	BatchResults []*BatchSchedulingResult `json:"batchResults"` // 各批次结果

	// 状态
	Completed    bool    `json:"completed"`    // 是否完成
	TotalTime    float64 `json:"totalTime"`    // 总执行时间（秒）
	ErrorMessage string  `json:"errorMessage"` // 错误信息（如有）
}

// BatchSchedulingResult 单批次排班结果
type BatchSchedulingResult struct {
	BatchIndex     int      `json:"batchIndex"`     // 批次序号（从1开始）
	RequestedCount int      `json:"requestedCount"` // 请求安排人数
	ScheduledIDs   []string `json:"scheduledIds"`   // 实际安排的人员ID
	ScheduledNames []string `json:"scheduledNames"` // 实际安排的人员姓名
	Reasoning      string   `json:"reasoning"`      // 决策理由
	ExecutionTime  float64  `json:"executionTime"`  // 执行耗时（秒）
	RetryCount     int      `json:"retryCount"`     // 重试次数
	ErrorMessage   string   `json:"errorMessage"`   // 错误信息（如有）
}

// NewBatchSchedulingContext 创建批次排班上下文
func NewBatchSchedulingContext(shiftID, shiftName, targetDate string, totalRequired int, candidates []*d_model.Employee) *BatchSchedulingContext {
	totalBatches := (totalRequired + DefaultBatchSize - 1) / DefaultBatchSize
	return &BatchSchedulingContext{
		ShiftID:             shiftID,
		ShiftName:           shiftName,
		TargetDate:          targetDate,
		TotalRequired:       totalRequired,
		TotalBatches:        totalBatches,
		CurrentBatch:        0,
		ScheduledStaff:      make([]string, 0),
		RemainingCandidates: candidates,
		ExcludedStaffIDs:    make([]string, 0),
		BatchResults:        make([]*BatchSchedulingResult, 0),
		Completed:           false,
	}
}

// AddBatchResult 添加批次结果
func (c *BatchSchedulingContext) AddBatchResult(result *BatchSchedulingResult) {
	c.BatchResults = append(c.BatchResults, result)
	c.ScheduledStaff = append(c.ScheduledStaff, result.ScheduledIDs...)

	// 检查是否已完成
	if len(c.ScheduledStaff) >= c.TotalRequired {
		c.Completed = true
	}
}

// GetRemainingCount 获取剩余需要安排的人数
func (c *BatchSchedulingContext) GetRemainingCount() int {
	remaining := c.TotalRequired - len(c.ScheduledStaff)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetProgress 获取进度百分比
func (c *BatchSchedulingContext) GetProgress() float64 {
	if c.TotalRequired == 0 {
		return 100.0
	}
	return float64(len(c.ScheduledStaff)) / float64(c.TotalRequired) * 100
}

// GetSummary 获取排班摘要
func (c *BatchSchedulingContext) GetSummary() string {
	if c.Completed {
		return fmt.Sprintf("%s排班完成：需要%d人，已安排%d人，共%d批次",
			c.TargetDate, c.TotalRequired, len(c.ScheduledStaff), len(c.BatchResults))
	}
	return fmt.Sprintf("%s排班进行中：需要%d人，已安排%d人，进度%.0f%%",
		c.TargetDate, c.TotalRequired, len(c.ScheduledStaff), c.GetProgress())
}

// ============================================================
// 分批修复相关结构
// ============================================================

// BatchFixContext LLM4分批修复上下文
type BatchFixContext struct {
	// 基本信息
	CurrentRound int `json:"currentRound"` // 当前修复轮次
	MaxRounds    int `json:"maxRounds"`    // 最大修复轮次

	// 问题统计
	InitialViolations int                        `json:"initialViolations"` // 初始违规数量
	CurrentViolations []*d_model.ValidationIssue `json:"currentViolations"` // 当前剩余违规

	// 修复记录
	FixResults []*BatchFixResult `json:"fixResults"` // 各轮修复结果

	// 状态
	AllFixed     bool    `json:"allFixed"`     // 是否全部修复
	TotalTime    float64 `json:"totalTime"`    // 总执行时间（秒）
	ErrorMessage string  `json:"errorMessage"` // 错误信息（如有）
}

// BatchFixResult 单轮修复结果
type BatchFixResult struct {
	Round            int               `json:"round"`            // 轮次
	ViolationsBefore int               `json:"violationsBefore"` // 修复前违规数
	ViolationsAfter  int               `json:"violationsAfter"`  // 修复后违规数
	FixedCount       int               `json:"fixedCount"`       // 修复的违规数
	ModifiedStaff    []string          `json:"modifiedStaff"`    // 被修改的人员ID
	Changes          []*ScheduleChange `json:"changes"`          // 具体变更
	ExecutionTime    float64           `json:"executionTime"`    // 执行耗时（秒）
}

// ScheduleChange 排班变更记录
type ScheduleChange struct {
	StaffID    string `json:"staffId"`
	StaffName  string `json:"staffName"`
	Date       string `json:"date"`
	ChangeType string `json:"changeType"` // "add", "remove", "replace"
	OldShiftID string `json:"oldShiftId"` // 原班次（replace时）
	NewShiftID string `json:"newShiftId"` // 新班次（replace时）
	Reason     string `json:"reason"`     // 变更原因
}

// NewBatchFixContext 创建修复上下文
func NewBatchFixContext(maxRounds int, violations []*d_model.ValidationIssue) *BatchFixContext {
	return &BatchFixContext{
		CurrentRound:      0,
		MaxRounds:         maxRounds,
		InitialViolations: len(violations),
		CurrentViolations: violations,
		FixResults:        make([]*BatchFixResult, 0),
		AllFixed:          len(violations) == 0,
	}
}

// ShouldContinue 是否应继续修复
func (c *BatchFixContext) ShouldContinue() bool {
	if c.AllFixed {
		return false
	}
	if c.CurrentRound >= c.MaxRounds {
		return false
	}
	if len(c.CurrentViolations) == 0 {
		c.AllFixed = true
		return false
	}
	return true
}

// GetSummary 获取修复摘要
func (c *BatchFixContext) GetSummary() string {
	if c.AllFixed {
		return fmt.Sprintf("LLM4修复完成：初始%d个违规，经%d轮修复全部解决",
			c.InitialViolations, c.CurrentRound)
	}
	return fmt.Sprintf("LLM4修复结束：初始%d个违规，经%d轮修复后剩余%d个",
		c.InitialViolations, c.CurrentRound, len(c.CurrentViolations))
}

// ============================================================
// 进度通知结构
// ============================================================

// BatchProgressInfo 批次进度信息（用于通知前端）
type BatchProgressInfo struct {
	// 类型标识
	Type string `json:"type"` // "validation", "scheduling", "fix"

	// 批次信息
	CurrentBatch int `json:"currentBatch"`
	TotalBatches int `json:"totalBatches"`

	// 人员信息
	CurrentStaffNames []string `json:"currentStaffNames"` // 当前批次处理的人员
	TotalStaff        int      `json:"totalStaff"`
	ProcessedStaff    int      `json:"processedStaff"`

	// 进度
	Progress float64 `json:"progress"` // 0-100

	// 状态
	Status  string `json:"status"`  // "processing", "completed", "failed"
	Message string `json:"message"` // 人类可读的消息
}
