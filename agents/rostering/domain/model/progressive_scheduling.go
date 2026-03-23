package model

// ============================================================
// 渐进式排班数据模型
// ============================================================

// ProgressiveTaskPlan 渐进式任务计划
type ProgressiveTaskPlan struct {
	// Tasks 有序的任务列表
	Tasks []*ProgressiveTask `json:"tasks"`

	// Summary 整体规划说明（简要阐述排班策略）
	Summary string `json:"summary"`

	// Reasoning AI的思考过程（为什么这样拆分任务）
	Reasoning string `json:"reasoning"`
}

// ProgressiveTask 渐进式任务
type ProgressiveTask struct {
	// ID 任务ID（从1开始递增）
	ID string `json:"id"`

	// Order 执行顺序（1开始）
	Order int `json:"order"`

	// Title 任务标题（简明扼要，如"安排固定班人员"）
	Title string `json:"title"`

	// Description 任务详细说明（任务概述 + 完整规则原文）
	Description string `json:"description"`

	// Type 任务类型: "ai"(AI执行), "fill"(填充逻辑), "validation"(规则校验)
	Type string `json:"type"`

	// TargetShifts 涉及的班次ID列表
	TargetShifts []string `json:"targetShifts"`

	// TargetDates 涉及的日期列表 (YYYY-MM-DD)
	TargetDates []string `json:"targetDates"`

	// TargetStaff 涉及的人员ID列表（可选）
	TargetStaff []string `json:"targetStaff,omitempty"`

	// RuleIDs 相关规则ID列表
	RuleIDs []string `json:"ruleIds"`

	// Priority 优先级 (1-高，2-中，3-低)
	Priority int `json:"priority"`

	// Status 状态: "pending", "executing", "completed", "failed"
	Status string `json:"status"`

	// Result 执行结果说明
	Result string `json:"result,omitempty"`

	// ExecutedAt 执行时间
	ExecutedAt string `json:"executedAt,omitempty"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	// TaskID 任务ID
	TaskID string `json:"taskId"`

	// Success 是否成功
	Success bool `json:"success"`

	// ScheduleDraft 排班草案（如果有）
	// 【已废弃】保留用于向后兼容，新代码应使用 ShiftSchedules
	ScheduleDraft *ShiftScheduleDraft `json:"scheduleDraft,omitempty"`

	// ShiftSchedules 按班次组织的排班草案（shiftID -> ShiftScheduleDraft）
	// 【新增】支持班次维度，避免不同班次的数据混淆
	ShiftSchedules map[string]*ShiftScheduleDraft `json:"shiftSchedules,omitempty"`

	// RuleValidationResult 规则级校验结果
	RuleValidationResult *RuleValidationResult `json:"ruleValidationResult,omitempty"`

	// LLMQCResult LLMQC校验结果
	LLMQCResult *ValidationResult `json:"llmqcResult,omitempty"`

	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`

	// ExecutionTime 执行时间（秒）
	ExecutionTime float64 `json:"executionTime"`

	// Metadata 元数据（用于存储额外信息，如通知状态）
	Metadata map[string]any `json:"metadata,omitempty"`

	// PartiallySucceeded 是否部分成功（有班次失败但有班次成功）
	PartiallySucceeded bool `json:"partiallySucceeded,omitempty"`

	// SuccessfulShifts 成功执行的班次ID列表
	SuccessfulShifts []string `json:"successfulShifts,omitempty"`

	// FailedShifts 失败的班次详细信息（shiftID -> ShiftFailureInfo）
	FailedShifts map[string]*ShiftFailureInfo `json:"failedShifts,omitempty"`
}

// ShiftFailureInfo 班次失败信息
type ShiftFailureInfo struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`

	// FailureSummary 失败摘要（简要描述）
	FailureSummary string `json:"failureSummary"`

	// AutoRetryCount 自动重试次数
	AutoRetryCount int `json:"autoRetryCount"`

	// ManualRetryCount 手动重试次数
	ManualRetryCount int `json:"manualRetryCount"`

	// FailureHistory 历史失败记录（语义化描述列表）
	FailureHistory []string `json:"failureHistory,omitempty"`

	// LastError 最后一次错误信息
	LastError string `json:"lastError,omitempty"`

	// ValidationIssues 校验问题列表
	ValidationIssues []*ValidationIssue `json:"validationIssues,omitempty"`
}

// ShiftRetryContext 班次重试上下文
type ShiftRetryContext struct {
	// RetryCount 当前重试次数（从0开始）
	RetryCount int

	// IsManualRetry 是否为手动重试
	IsManualRetry bool

	// FailureHistory 历史失败的语义化描述列表
	// 格式示例："尝试1：安排了3人但需要5人，违反人数规则"
	FailureHistory []string

	// AIRecommendations AI分析的改进建议
	AIRecommendations string

	// LastValidationResult 上次校验结果
	LastValidationResult *RuleValidationResult

	// TargetRetryDates 需要重排的目标日期列表
	// 仅当此列表非空时，重试只针对这些日期进行
	TargetRetryDates []string

	// ConflictingShiftIDs 冲突的班次ID列表（用于互斥规则违反时）
	// 如果校验发现当前班次与其他班次冲突，记录这些班次ID
	ConflictingShiftIDs []string

	// RetryOnlyTargetDates 是否只重排目标日期
	// 如果为true，其他日期保留原有排班结果
	RetryOnlyTargetDates bool

	// LLM4AdjustmentCount LLM4调整次数（用于防止循环调整）
	// 每个班次最多调用1次LLM4，避免无限循环
	LLM4AdjustmentCount int
}

// RuleValidationResult 规则级校验结果
type RuleValidationResult struct {
	// Passed 是否通过校验
	Passed bool `json:"passed"`

	// StaffCountIssues 人数校验问题列表
	StaffCountIssues []*ValidationIssue `json:"staffCountIssues,omitempty"`

	// ShiftRuleIssues 班次规则校验问题列表
	ShiftRuleIssues []*ValidationIssue `json:"shiftRuleIssues,omitempty"`

	// RuleComplianceIssues 规则合规性校验问题列表
	RuleComplianceIssues []*ValidationIssue `json:"ruleComplianceIssues,omitempty"`

	// Summary 校验总结
	Summary string `json:"summary"`
}

// ValidationIssue 校验问题
// 注意：ValidationIssue 已在 schedule.go 中定义，这里使用该定义
// 如果需要 AffectedShifts 字段，应该添加到 schedule.go 中的定义

// ============================================================
// 统一排班输出协议
// ============================================================

// ScheduleOutputMode 排班输出模式
type ScheduleOutputMode string

const (
	// ScheduleOutputModeAdd 追加模式：将输出人员追加到已有排班
	ScheduleOutputModeAdd ScheduleOutputMode = "add"
	// ScheduleOutputModeReplace 替换模式：用输出替换动态排班（保留固定排班）
	ScheduleOutputModeReplace ScheduleOutputMode = "replace"
)

// ScheduleOutput 统一排班输出结构
// 所有LLM排班输出（生成、调整、重试）统一使用此结构
type ScheduleOutput struct {
	// Mode 输出模式：add（追加）或 replace（替换动态排班）
	Mode ScheduleOutputMode `json:"mode"`
	// Schedule 排班结果：日期 -> 员工ID列表
	Schedule map[string][]string `json:"schedule"`
	// Reasoning AI推理说明
	Reasoning string `json:"reasoning,omitempty"`
}

// ============================================================
// 渐进式执行检查点（用于中断恢复）
// ============================================================

// ShiftExecutionCheckpoint 班次执行检查点
// 每天完成后保存，用于中断恢复
type ShiftExecutionCheckpoint struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`
	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`
	// LastCompletedDate 最后完成的日期（YYYY-MM-DD）
	LastCompletedDate string `json:"lastCompletedDate"`
	// CompletedDates 已完成的日期列表
	CompletedDates []string `json:"completedDates"`
	// AllDates 所有需要处理的日期列表
	AllDates []string `json:"allDates"`
	// DraftSnapshot 草案快照（当前排班状态）
	DraftSnapshot *ShiftScheduleDraft `json:"draftSnapshot"`
	// OccupiedSlotsSnapshot 占位信息快照
	OccupiedSlotsSnapshot map[string]map[string]string `json:"occupiedSlotsSnapshot"`
	// UpdatedAt 更新时间
	UpdatedAt string `json:"updatedAt"`
}

// ============================================================
// 天级别进度信息（扩展 ShiftProgressInfo）
// ============================================================

// DayProgressInfo 天级别进度信息
type DayProgressInfo struct {
	// CurrentDay 当前第几天（从1开始）
	CurrentDay int `json:"currentDay"`
	// TotalDays 总天数
	TotalDays int `json:"totalDays"`
	// CurrentDate 当前处理的日期（YYYY-MM-DD）
	CurrentDate string `json:"currentDate"`
	// CompletedDates 已完成的日期列表
	CompletedDates []string `json:"completedDates"`
	// DraftPreview 当前草案预览（完整JSON）
	DraftPreview *ShiftScheduleDraft `json:"draftPreview,omitempty"`
}

// DayProgressStatus 天级别进度状态
type DayProgressStatus string

const (
	// DayStatusGenerating 正在生成当天排班
	DayStatusGenerating DayProgressStatus = "day_generating"
	// DayStatusCompleted 当天排班完成
	DayStatusCompleted DayProgressStatus = "day_completed"
	// ShiftStatusValidating 班次校验中
	ShiftStatusValidating DayProgressStatus = "shift_validating"
	// ShiftStatusRetrying 班次重试中
	ShiftStatusRetrying DayProgressStatus = "shift_retrying"
	// ShiftStatusSuccess 班次成功
	ShiftStatusSuccess DayProgressStatus = "shift_success"
	// ShiftStatusFailed 班次失败
	ShiftStatusFailed DayProgressStatus = "shift_failed"
)
