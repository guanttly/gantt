package executor

import (
	"context"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"

	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// ============================================================
// 任务解析相关数据结构
// ============================================================

// ShiftTaskSpec 班次任务规格
// 用于存储解析出的班次信息和该班次的任务说明
type ShiftTaskSpec struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`
	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`
	// Description 该班次的任务说明
	Description string `json:"description"`
}

// ============================================================
// 关联班次分组相关数据结构
// ============================================================

// ShiftGroup 班次分组
// 表示一组需要一起排班的关联班次（共享规则和人员信息）
type ShiftGroup struct {
	// GroupID 分组ID（自动生成）
	GroupID string `json:"groupId"`
	// Shifts 分组内的班次列表
	Shifts []ShiftTaskSpec `json:"shifts"`
	// RelatedReason 关联原因（为什么这些班次需要一起排班）
	RelatedReason string `json:"relatedReason"`
	// SharedRuleIDs 组内共享的规则ID列表（需要在所有班次的提示词中体现）
	SharedRuleIDs []string `json:"sharedRuleIds"`
}

// ShiftGroupingResult 班次分组识别结果
// LLM识别出哪些班次需要一起排班，哪些可以独立处理
type ShiftGroupingResult struct {
	// Groups 分组列表（每组内的班次需要一起排班）
	Groups []ShiftGroup `json:"groups"`
	// Reasoning 分组决策说明
	Reasoning string `json:"reasoning"`
}

// ShiftGroupRelation 班次关联关系（用于LLM返回）
type ShiftGroupRelation struct {
	// ShiftID1 班次1的ID
	ShiftID1 string `json:"shiftId1"`
	// ShiftID2 班次2的ID
	ShiftID2 string `json:"shiftId2"`
	// RelationType 关联类型
	RelationType string `json:"relationType"`
	// Reason 关联原因
	Reason string `json:"reason"`
	// SharedRuleIDs 共享的规则ID
	SharedRuleIDs []string `json:"sharedRuleIds"`
}

// ============================================================
// 渐进式任务执行器
// ============================================================

// ShiftProgressInfo 班次进度信息
type ShiftProgressInfo struct {
	ShiftID     string `json:"shiftId"`
	ShiftName   string `json:"shiftName"`
	Current     int    `json:"current"`     // 当前进度（第几个班次）
	Total       int    `json:"total"`       // 总班次数
	Status      string `json:"status"`      // 状态：executing, success, failed, validating, retrying, day_generating, day_completed, shift_validating, shift_retrying
	Message     string `json:"message"`     // 进度消息
	Reasoning   string `json:"reasoning"`   // AI解释（可选）
	PreviewData string `json:"previewData"` // 预览数据（可选，JSON格式的排班结果）

	// 天级别进度字段（渐进式排班新增）
	CurrentDay     int      `json:"currentDay,omitempty"`     // 当前第几天（从1开始）
	TotalDays      int      `json:"totalDays,omitempty"`      // 总天数
	CurrentDate    string   `json:"currentDate,omitempty"`    // 当前处理的日期（YYYY-MM-DD）
	CompletedDates []string `json:"completedDates,omitempty"` // 已完成的日期列表
	DraftPreview   string   `json:"draftPreview,omitempty"`   // 当前完整草案预览（JSON格式）
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(info *ShiftProgressInfo)

// IProgressiveTaskExecutor 渐进式任务执行器接口
type IProgressiveTaskExecutor interface {
	// ExecuteProgressiveTask 执行渐进式任务（旧接口，保持兼容）
	// Deprecated: 请使用 ExecuteTask，直接传递 CoreV3TaskContext
	// currentDraft 只包含动态排班数据，不包含固定排班
	ExecuteProgressiveTask(
		ctx context.Context,
		task *d_model.ProgressiveTask,
		shifts []*d_model.Shift,
		rules []*d_model.Rule,
		staffList []*d_model.Employee,
		staffRequirements map[string]map[string]int,
		occupiedSlots map[string]map[string]string,
		currentDraft *d_model.ShiftScheduleDraft,
	) (*d_model.TaskResult, error)

	// ExecuteTask 执行渐进式任务（新接口，推荐使用）
	// 使用 CoreV3TaskContext 对象，简化参数传递
	ExecuteTask(
		ctx context.Context,
		taskCtx *utils.CoreV3TaskContext,
	) (*d_model.TaskResult, error)

	// ExecuteShiftTask 执行单个班次任务（V3改进，使用ShiftTaskContext）
	// 在单班次级别动态计算候选人员，根据当前占位状态实时过滤
	ExecuteShiftTask(
		ctx context.Context,
		shiftCtx *utils.ShiftTaskContext,
		orgID string,
		workingDraft *d_model.ScheduleDraft,
	) (*d_model.ShiftScheduleDraft, error)

	// SetProgressCallback 设置进度回调函数
	SetProgressCallback(callback ProgressCallback)

	// ValidateFinalSchedule 执行最终严格人数校验
	// 在所有任务完成后调用，检查所有班次的人数是否严格等于需求
	// 如果存在缺员，自动生成 LLM 补充任务
	ValidateFinalSchedule(
		ctx context.Context,
		workingDraft *d_model.ScheduleDraft,
		staffRequirements map[string]map[string]int,
		shifts []*d_model.Shift,
		rules []*d_model.Rule,
		staffList []*d_model.Employee,
		maxFillRounds int,
	) *FinalValidationResult

	// ValidateTaskStaffCount 校验任务相关班次的人数需求（任务级校验）
	// 在任务完成后立即调用，检查该任务相关班次的人数是否满足需求
	// 如果发现缺员，立即生成补充任务
	ValidateTaskStaffCount(
		ctx context.Context,
		task *d_model.ProgressiveTask,
		workingDraft *d_model.ScheduleDraft,
		staffRequirements []d_model.ShiftDateRequirement,
		selectedShifts []*d_model.Shift,
	) *TaskValidationResult
}

// ISchedulingAIService 排班AI服务接口（用于任务执行）
type ISchedulingAIService interface {
	ExecuteTodoTask(
		ctx context.Context,
		todoTask *d_model.SchedulingTodo,
		shiftInfo *d_model.ShiftInfo,
		availableStaff []*d_model.StaffInfoForAI,
		currentDraft *d_model.ShiftScheduleDraft,
		staffRequirements map[string]int,
		fixedShiftAssignments map[string][]string,
		temporaryNeeds []*d_model.PersonalNeed,
		allStaffList []*d_model.Employee,
		allShifts []*d_model.Shift,
		workingDraft *d_model.ScheduleDraft,
	) (*d_model.TodoExecutionResult, error)
}

// ProgressiveTaskExecutor 渐进式任务执行器实现
type ProgressiveTaskExecutor struct {
	logger              logging.ILogger
	schedulingAIService ISchedulingAIService
	ruleValidator       d_service.IRuleLevelValidator
	rosteringService    d_service.IRosteringService   // 排班服务（用于获取班次分组成员）
	taskContext         *utils.CoreV3TaskContext      // 任务上下文（用于获取固定排班等信息）
	aiFactory           *ai.AIProviderFactory         // AI工厂
	configurator        config.IRosteringConfigurator // 配置器
	progressCallback    ProgressCallback              // 进度回调函数

	// 用于存储最后一次执行的失败信息（临时存储，供外层获取）
	lastFailedShifts     map[string]*d_model.ShiftFailureInfo
	lastSuccessfulShifts []string
}

// ============================================================
// 规则预分析相关数据结构（策划阶段）
// ============================================================

// DayRuleAnalysis 单日规则分析结果
// 用于在执行排班前，让LLM先分析当日需要关注的规则和约束
type DayRuleAnalysis struct {
	// RelevantRules 当日相关的规则（已过滤）
	RelevantRules []RelevantRule `json:"relevantRules"`
	// RelevantPersonalNeeds 当日相关的个人需求
	RelevantPersonalNeeds []RelevantPersonalNeed `json:"relevantPersonalNeeds"`
	// Summary 当日排班注意事项总结
	Summary string `json:"summary"`
}

// RelevantRule 相关规则
type RelevantRule struct {
	// RuleName 规则名称
	RuleName string `json:"ruleName"`
	// Reason 为什么此规则与当日排班相关
	Reason string `json:"reason"`
	// Constraint 具体约束描述
	Constraint string `json:"constraint"`
}

// RelevantPersonalNeed 相关个人需求
type RelevantPersonalNeed struct {
	// StaffName 员工姓名
	StaffName string `json:"staffName"`
	// StaffID 员工ID
	StaffID string `json:"staffId"`
	// NeedType 需求类型（如：请假、不可用等）
	NeedType string `json:"needType"`
	// Description 需求描述
	Description string `json:"description"`
}

// ============================================================
// 拆分后的分析结构体（LLM1: 人员过滤, LLM2: 规则过滤）
// ============================================================

// PersonalNeedsAnalysis 人员可用性分析结果（LLM1输出）
type PersonalNeedsAnalysis struct {
	// UnavailableStaff 当日不可用的员工列表
	UnavailableStaff []UnavailableStaffInfo `json:"unavailableStaff"`
}

// UnavailableStaffInfo 不可用员工信息
type UnavailableStaffInfo struct {
	// StaffName 员工姓名
	StaffName string `json:"staffName"`
	// StaffID 员工ID
	StaffID string `json:"staffId"`
	// Reason 不可用原因
	Reason string `json:"reason"`
}

// RulesAnalysis 规则过滤分析结果（LLM2输出）
type RulesAnalysis struct {
	// RelevantRules 与当前班次相关的规则
	RelevantRules []FilteredRule `json:"relevantRules"`
	// ExcludedRules 被排除的规则名（用于验证）
	ExcludedRules []string `json:"excludedRules"`
}

// FilteredRule 过滤后的规则信息
type FilteredRule struct {
	// RuleName 规则名称
	RuleName string `json:"ruleName"`
	// Constraint 具体约束描述（改写后的约束）
	Constraint string `json:"constraint"`
}

// RuleConflictAnalysis 规则冲突人员分析结果（LLM3输出）
type RuleConflictAnalysis struct {
	// ConflictStaff 因规则冲突而不可用的员工列表
	ConflictStaff []RuleConflictStaffInfo `json:"conflictStaff"`
}

// RuleConflictStaffInfo 规则冲突员工信息
type RuleConflictStaffInfo struct {
	// StaffName 员工姓名
	StaffName string `json:"staffName"`
	// StaffID 员工ID
	StaffID string `json:"staffId"`
	// ConflictRule 冲突的规则名
	ConflictRule string `json:"conflictRule"`
	// Reason 冲突原因（如：本周2/11已有固定排班，违反每周最多1次规则）
	Reason string `json:"reason"`
}

// ============================================================
// 任务间规则匹配度校验（LLM）
// ============================================================

// RuleMatchingResult 规则匹配度校验结果
type RuleMatchingResult struct {
	Passed      bool                 `json:"passed"`      // 是否通过
	MatchScore  float64              `json:"matchScore"`  // 匹配度分数 0-1
	Issues      []*RuleMatchingIssue `json:"issues"`      // 匹配问题列表
	LLMAnalysis string               `json:"llmAnalysis"` // LLM分析说明
	NeedsRetry  bool                 `json:"needsRetry"`  // 是否需要重试（高优先级规则失败）
}

// RuleMatchingIssue 规则匹配问题
type RuleMatchingIssue struct {
	RuleID        string   `json:"ruleId"`        // 规则ID
	RuleName      string   `json:"ruleName"`      // 规则名称
	RuleType      string   `json:"ruleType"`      // 规则类型
	Priority      int      `json:"priority"`      // 规则优先级
	Severity      string   `json:"severity"`      // 严重程度：critical/warning
	Description   string   `json:"description"`   // 问题描述
	AffectedDates []string `json:"affectedDates"` // 受影响日期
	Suggestion    string   `json:"suggestion"`    // 改进建议
}

// ============================================================
// 最终校验与 LLM 补充任务生成
// ============================================================

// FinalValidationResult 最终校验结果
type FinalValidationResult struct {
	Passed                  bool                       `json:"passed"`                  // 是否通过
	ShortageDetails         []*ShortageDetail          `json:"shortageDetails"`         // 缺员详情列表
	SupplementTasks         []*d_model.ProgressiveTask `json:"supplementTasks"`         // 生成的补充任务
	Summary                 string                     `json:"summary"`                 // 校验总结
	NeedsManualIntervention bool                       `json:"needsManualIntervention"` // 是否需要人工介入
}

// ShortageDetail 缺员详情
type ShortageDetail struct {
	ShiftID       string `json:"shiftId"`       // 班次ID
	ShiftName     string `json:"shiftName"`     // 班次名称
	Date          string `json:"date"`          // 日期
	RequiredCount int    `json:"requiredCount"` // 需求人数
	ActualCount   int    `json:"actualCount"`   // 实际人数
	ShortageCount int    `json:"shortageCount"` // 缺员数量
}

// TaskValidationResult 任务级人数校验结果
type TaskValidationResult struct {
	Passed          bool             `json:"passed"`          // 是否通过
	ShortageDetails []*ShortageDetail `json:"shortageDetails"` // 缺员详情列表
	Summary         string           `json:"summary"`        // 校验摘要
}

// ============================================================
// LLM4: 跨日期班次调整校验
// ============================================================

// ScheduleAdjustmentResult LLM4班次调整结果
type ScheduleAdjustmentResult struct {
	AdjustedSchedule map[string][]string `json:"adjustedSchedule"` // 调整后的排班结果：date -> staffIDs
	Reasoning        string              `json:"reasoning"`        // 调整说明
	ReplacedStaff    map[string][]string `json:"replacedStaff"`    // 被替换的人员：date -> 被替换的staffIDs
	NewStaff         map[string][]string `json:"newStaff"`         // 新增的人员：date -> 新增的staffIDs
}
