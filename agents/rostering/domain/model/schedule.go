package model

import (
	scheduling_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的排班类型
type ScheduleAssignment = scheduling_model.ScheduleAssignment
type ScheduleEntry = scheduling_model.ScheduleEntry
type ScheduleUpsertRequest = scheduling_model.ScheduleUpsertRequest
type ScheduleQueryRequest = scheduling_model.ScheduleQueryRequest
type ScheduleQueryResponse = scheduling_model.ScheduleQueryResponse
type BatchAssignRequest = scheduling_model.BatchAssignRequest
type BatchUpsertResult = scheduling_model.BatchUpsertResult
type BatchItemResult = scheduling_model.BatchItemResult
type GetScheduleByDateRangeRequest = scheduling_model.GetScheduleByDateRangeRequest
type GetScheduleSummaryRequest = scheduling_model.GetScheduleSummaryRequest

// ScheduleBatch 批量排班操作（保持向后兼容）
type ScheduleBatch struct {
	Items          []ScheduleUpsertRequest `json:"items"`
	IdempotencyKey string                  `json:"idempotencyKey,omitempty"`
	OnConflict     string                  `json:"onConflict,omitempty"` // upsert/skip/replace
}

// ScheduleQueryFilter 排班查询过滤器（向后兼容）
type ScheduleQueryFilter struct {
	EmployeeID string
	StartDate  string
	EndDate    string
	OrgID      string
	Status     string
	Page       int
	PageSize   int
}

// ScheduleQueryResult 排班查询结果（向后兼容）
type ScheduleQueryResult struct {
	Schedules  []*ScheduleAssignment `json:"schedules"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	HasMore    bool                  `json:"hasMore"`
	Department string                `json:"department,omitempty"`
	Modality   string                `json:"modality,omitempty"`
}

// ============================================================
// 排班创建工作流数据模型
// ============================================================

// Session.Data 中的数据键常量
const (
	DataKeyScheduleCreateContext = "schedule_create_context"
)

// ScheduleCreateContext 排班创建工作流上下文
type ScheduleCreateContext struct {
	// 排班周期
	StartDate string `json:"startDate"` // YYYY-MM-DD (默认下周一)
	EndDate   string `json:"endDate"`   // YYYY-MM-DD (默认下周日)

	// 班次配置
	AvailableShifts        []*Shift                  `json:"availableShifts"`        // SDK 查询的所有班次
	SelectedShifts         []*Shift                  `json:"selectedShifts"`         // 用户选择的班次列表 (按优先级排序)
	ShiftStaffRequirements map[string]map[string]int `json:"shiftStaffRequirements"` // shiftId -> { date: count }

	// 数据收集
	StaffList     []*Employee               `json:"staffList"`     // 可用人员列表
	ShiftStaffIDs map[string][]string       `json:"shiftStaffIds"` // shiftId -> []staffId (复用人员检索结果)
	StaffLeaves   map[string][]*LeaveRecord `json:"staffLeaves"`   // staffId -> leaves
	GlobalRules   []*Rule                   `json:"globalRules"`   // 全局规则
	ShiftRules    map[string][]*Rule        `json:"shiftRules"`    // shiftId -> rules
	GroupRules    map[string][]*Rule        `json:"groupRules"`    // groupId -> rules
	EmployeeRules map[string][]*Rule        `json:"employeeRules"` // employeeId -> rules

	// 生成阶段（新增三阶段逻辑）
	CurrentShiftIndex  int                                `json:"currentShiftIndex"`  // 当前处理的班次索引
	ScheduledStaffSet  map[string]bool                    `json:"scheduledStaffSet"`  // [已废弃] 保留兼容
	StaffScheduleMarks map[string]map[string][]*ShiftMark `json:"staffScheduleMarks"` // 人员已排班标记 (staffID -> date -> []ShiftMark)
	DraftSchedule      *ScheduleDraft                     `json:"draftSchedule"`      // 排班草案
	AISummaries        []string                           `json:"aiSummaries"`        // 各轮AI总结 (供下一轮参考)

	// 新增：Todo任务分解和执行
	ShiftTodoPlans     map[string]*ShiftTodoPlan `json:"shiftTodoPlans"`     // shiftId -> Todo计划
	CurrentTodoIndex   int                       `json:"currentTodoIndex"`   // 当前执行的Todo索引
	TodoExecutionLogs  []string                  `json:"todoExecutionLogs"`  // Todo执行日志
	ValidationAttempts map[string]int            `json:"validationAttempts"` // 校验尝试次数 (shiftId -> attempts)

	// 完成阶段
	FinalSchedule *ScheduleDraft `json:"finalSchedule"` // 最终排班
}

// ScheduleDraft 排班草案
type ScheduleDraft struct {
	StartDate  string                 `json:"startDate"`  // YYYY-MM-DD
	EndDate    string                 `json:"endDate"`    // YYYY-MM-DD
	Shifts     map[string]*ShiftDraft `json:"shifts"`     // 班次ID -> 班次排班
	Summary    string                 `json:"summary"`    // AI生成的整体排班说明
	StaffStats map[string]*StaffStats `json:"staffStats"` // 人员统计 (staffID -> 统计信息)
	Conflicts  []*ScheduleConflict    `json:"conflicts"`  // 冲突列表
}

// ShiftDraft 单个班次的排班草案
type ShiftDraft struct {
	ShiftID  string               `json:"shiftId"`  // 班次ID
	Priority int                  `json:"priority"` // 优先级
	Days     map[string]*DayShift `json:"days"`     // 日期 -> 当天排班
}

// ============================================================
// StaffAssignment 人员分配记录（per-staff 粒度的排班元数据）
// ============================================================

// AssignmentSource 分配来源
type AssignmentSource string

const (
	AssignmentSourceFixed      AssignmentSource = "fixed"      // 固定排班（阶段零）
	AssignmentSourceRule       AssignmentSource = "rule"       // 规则占位（阶段一：PersonnelRule/Exclusive 等）
	AssignmentSourceDependency AssignmentSource = "dependency" // 依赖补排（DependencyResolver 为满足 A→B 依赖而补入 B）
	AssignmentSourceDefault    AssignmentSource = "default"    // 兜底填充（阶段二：DefaultScheduler）
)

// AssignmentPriority 分配优先级
type AssignmentPriority string

const (
	AssignmentPriorityHigh AssignmentPriority = "high" // 高优先级：fixed/rule/dependency，不可被依赖补排置换
	AssignmentPriorityLow  AssignmentPriority = "low"  // 低优先级：default 兜底填充，可被依赖补排置换
)

// StaffAssignment 单个人员的分配记录
type StaffAssignment struct {
	StaffID   string             `json:"staffId"`   // 人员ID
	StaffName string             `json:"staffName"` // 人员姓名
	Source    AssignmentSource   `json:"source"`    // 分配来源
	Priority  AssignmentPriority `json:"priority"`  // 优先级（high=不可被置换, low=可被依赖补排置换）
}

// SourceToPriority 根据分配来源推导优先级
func SourceToPriority(source AssignmentSource) AssignmentPriority {
	switch source {
	case AssignmentSourceFixed, AssignmentSourceRule, AssignmentSourceDependency:
		return AssignmentPriorityHigh
	default:
		return AssignmentPriorityLow
	}
}

// DayShift 某一天某个班次的排班
type DayShift struct {
	Staff         []string           `json:"staff"`                 // 人员姓名列表（兼容字段，由 Assignments 计算得出）
	StaffIDs      []string           `json:"staffIds"`              // 人员ID列表（兼容字段，由 Assignments 计算得出）
	Assignments   []*StaffAssignment `json:"assignments,omitempty"` // 人员分配详情（per-staff 粒度元数据）
	RequiredCount int                `json:"requiredCount"`         // 需要人数
	ActualCount   int                `json:"actualCount"`           // 实际人数
	IsFixed       bool               `json:"isFixed"`               // 是否为固定排班（用于标识预先填充的固定班次）
}

// GetReplaceableStaffIDs 获取可被置换的低优先级人员ID列表
func (d *DayShift) GetReplaceableStaffIDs() []string {
	if len(d.Assignments) == 0 {
		return nil
	}
	var ids []string
	for _, a := range d.Assignments {
		if a.Priority == AssignmentPriorityLow {
			ids = append(ids, a.StaffID)
		}
	}
	return ids
}

// RemoveStaffByID 从分配列表中移除指定人员，同步更新 Staff/StaffIDs/ActualCount
func (d *DayShift) RemoveStaffByID(staffID string) bool {
	for i, a := range d.Assignments {
		if a.StaffID == staffID {
			d.Assignments = append(d.Assignments[:i], d.Assignments[i+1:]...)
			d.rebuildCompatFields()
			return true
		}
	}
	return false
}

// AddAssignment 添加一个人员分配，同步更新 Staff/StaffIDs/ActualCount
func (d *DayShift) AddAssignment(a *StaffAssignment) {
	// 去重
	for _, existing := range d.Assignments {
		if existing.StaffID == a.StaffID {
			return
		}
	}
	d.Assignments = append(d.Assignments, a)
	d.rebuildCompatFields()
}

// rebuildCompatFields 从 Assignments 重建兼容字段 Staff/StaffIDs/ActualCount
func (d *DayShift) rebuildCompatFields() {
	d.StaffIDs = make([]string, 0, len(d.Assignments))
	d.Staff = make([]string, 0, len(d.Assignments))
	for _, a := range d.Assignments {
		d.StaffIDs = append(d.StaffIDs, a.StaffID)
		d.Staff = append(d.Staff, a.StaffName)
	}
	d.ActualCount = len(d.Assignments)
}

// StaffStats 人员统计信息
type StaffStats struct {
	StaffID    string   `json:"staffId"`    // 人员ID
	StaffName  string   `json:"staffName"`  // 人员姓名
	WorkDays   int      `json:"workDays"`   // 工作天数
	Shifts     []string `json:"shifts"`     // 各天的班次列表
	TotalHours float64  `json:"totalHours"` // 总工作小时数
}

// ShiftMark 班次标记（记录人员已排班信息）
type ShiftMark struct {
	ShiftID   string `json:"shiftId"`   // 班次ID
	ShiftName string `json:"shiftName"` // 班次名称
	StartTime string `json:"startTime"` // 开始时间 HH:MM
	EndTime   string `json:"endTime"`   // 结束时间 HH:MM
}

// 冲突类型常量
const (
	ConflictTypeUnderstaffed         = "understaffed"          // 人数不足
	ConflictTypeDependencyUnresolved = "dependency_unresolved" // 依赖关系无法满足
	ConflictTypeRuleConflict         = "rule_conflict"         // 规则冲突
	ConflictTypeCyclicDependency     = "cyclic_dependency"     // 循环依赖
)

// ScheduleConflict 排班冲突
type ScheduleConflict struct {
	Date            string   `json:"date"`                      // 日期 YYYY-MM-DD
	Shift           string   `json:"shift"`                     // 班次名称
	Issue           string   `json:"issue"`                     // 问题描述
	Severity        string   `json:"severity"`                  // 严重程度: warning/error
	ConflictType    string   `json:"conflictType,omitempty"`    // 冲突类型
	RelatedShiftIDs []string `json:"relatedShiftIds,omitempty"` // 关联班次ID
	RelatedRuleIDs  []string `json:"relatedRuleIds,omitempty"`  // 关联规则ID
	Suggestion      string   `json:"suggestion,omitempty"`      // 建议解决方案
	Detail          string   `json:"detail,omitempty"`          // 诊断详情（主体/客体班次人员快照）
}

// ============================================================
// 新增：Todo任务分解相关数据结构
// ============================================================

// ShiftTodoPlan 单个班次的Todo计划
type ShiftTodoPlan struct {
	ShiftID     string            `json:"shiftId"`     // 班次ID
	ShiftName   string            `json:"shiftName"`   // 班次名称
	TodoList    []*SchedulingTodo `json:"todoList"`    // AI生成的有序Todo列表
	PlanSummary string            `json:"planSummary"` // AI对整体计划的说明
	CreatedAt   string            `json:"createdAt"`   // 创建时间
}

// SchedulingTodo 排班Todo任务
type SchedulingTodo struct {
	ID          string   `json:"id"`          // Todo唯一标识
	Order       int      `json:"order"`       // 执行顺序（1开始）
	Title       string   `json:"title"`       // 任务标题（简短描述）
	Description string   `json:"description"` // 任务详细说明
	RuleIDs     []string `json:"ruleIds"`     // 相关规则ID列表
	TargetDates []string `json:"targetDates"` // 目标日期列表（可选，如果任务针对特定日期）
	TargetCount int      `json:"targetCount"` // 目标人数（可选）
	Priority    string   `json:"priority"`    // 优先级: high/medium/low
	Status      string   `json:"status"`      // 状态: pending/executing/completed/failed
	Result      string   `json:"result"`      // 执行结果说明
	ExecutedAt  string   `json:"executedAt"`  // 执行时间
}

// ============================================================
// ShiftScheduleDraft - 班次排班草案（用于Todo执行过程）
// ============================================================

// ShiftScheduleDraft 班次排班草案（强类型，用于Todo执行过程中的临时存储）
type ShiftScheduleDraft struct {
	Schedule        map[string][]string `json:"schedule"`        // 日期 -> 员工ID列表
	UpdatedStaff    map[string]bool     `json:"updatedStaff"`    // 已安排的员工ID集合（用于快速查找）
	RemainingIssues []string            `json:"remainingIssues"` // 遗留问题列表
}

// NewShiftScheduleDraft 创建空的班次排班草案
func NewShiftScheduleDraft() *ShiftScheduleDraft {
	return &ShiftScheduleDraft{
		Schedule:        make(map[string][]string),
		UpdatedStaff:    make(map[string]bool),
		RemainingIssues: make([]string, 0),
	}
}

// ============================================================
// AdjustScheduleResult - 调整排班结果
// ============================================================

// AdjustScheduleResult 调整排班结果
type AdjustScheduleResult struct {
	Draft   *ShiftScheduleDraft    // 完整的调整后排班（包含所有日期）
	Summary string                 // AI的总结说明
	Changes []AdjustScheduleChange // 变化列表（可选，用于对比显示）
}

// AdjustScheduleChange 调整排班变化（用于显示对比）
type AdjustScheduleChange struct {
	Date    string   // 日期 YYYY-MM-DD
	Added   []string // 新增的人员ID
	Removed []string // 移除的人员ID
}

// MergeTodoResult 合并 Todo 执行结果（强类型版本）
// 根据 scheduleActions 决定是追加还是替换：
//   - append: 追加到已有排班（用于新增排班）
//   - replace: 替换已有排班（用于修正错误）
func (d *ShiftScheduleDraft) MergeTodoResult(result *TodoExecutionResult) {
	if result == nil {
		return
	}

	// 合并 schedule - 根据 action 类型处理
	for date, staffIDs := range result.Schedule {
		// 获取该日期的操作类型，默认为 append（向后兼容）
		action := ScheduleActionAppend
		if result.ScheduleActions != nil {
			if act, exists := result.ScheduleActions[date]; exists {
				action = act
			}
		}

		switch action {
		case ScheduleActionReplace:
			// 替换模式：先清除旧的人员标记，再设置新的排班
			if oldStaffIDs, exists := d.Schedule[date]; exists {
				for _, oldID := range oldStaffIDs {
					// 检查该人员是否在其他日期还有排班
					stillScheduled := false
					for otherDate, otherStaffIDs := range d.Schedule {
						if otherDate == date {
							continue
						}
						for _, oid := range otherStaffIDs {
							if oid == oldID {
								stillScheduled = true
								break
							}
						}
						if stillScheduled {
							break
						}
					}
					// 如果该人员在其他日期没有排班，则移除标记
					if !stillScheduled {
						delete(d.UpdatedStaff, oldID)
					}
				}
			}
			// 替换该日期的排班
			d.Schedule[date] = staffIDs

		case ScheduleActionAppend:
			// 追加模式：追加到已有排班
			d.Schedule[date] = append(d.Schedule[date], staffIDs...)
		}

		// 标记新安排的员工
		for _, staffID := range staffIDs {
			d.UpdatedStaff[staffID] = true
		}
	}

	// 合并 issues
	d.RemainingIssues = append(d.RemainingIssues, result.Issues...)
}

// ApplyAdjustment 应用校验调整（替换指定日期的排班）
func (d *ShiftScheduleDraft) ApplyAdjustment(adjusted map[string]any) {
	for date, staffsAny := range adjusted {
		var staffIDs []string
		switch staffs := staffsAny.(type) {
		case []any:
			staffIDs = make([]string, 0, len(staffs))
			for _, s := range staffs {
				if sid, ok := s.(string); ok {
					staffIDs = append(staffIDs, sid)
				}
			}
		case []string:
			staffIDs = staffs
		}
		d.Schedule[date] = staffIDs // 替换
	}
	// 重建 UpdatedStaff
	d.rebuildUpdatedStaff()
}

// rebuildUpdatedStaff 重建已安排员工集合
func (d *ShiftScheduleDraft) rebuildUpdatedStaff() {
	d.UpdatedStaff = make(map[string]bool)
	for _, staffIDs := range d.Schedule {
		for _, sid := range staffIDs {
			d.UpdatedStaff[sid] = true
		}
	}
}

// GetAllScheduledStaffIDs 获取所有已安排的员工ID列表
func (d *ShiftScheduleDraft) GetAllScheduledStaffIDs() []string {
	result := make([]string, 0, len(d.UpdatedStaff))
	for sid := range d.UpdatedStaff {
		result = append(result, sid)
	}
	return result
}

// IsStaffScheduled 检查员工是否已被安排
func (d *ShiftScheduleDraft) IsStaffScheduled(staffID string) bool {
	return d.UpdatedStaff[staffID]
}

// ToMap 转换为 map[string]any（用于 session 存储和向后兼容）
func (d *ShiftScheduleDraft) ToMap() map[string]any {
	// 将 Schedule 转换为 map[string]any 以便类型断言兼容
	scheduleAny := make(map[string]any)
	for date, staffIDs := range d.Schedule {
		scheduleAny[date] = staffIDs
	}

	return map[string]any{
		"schedule":        scheduleAny,
		"updatedStaff":    d.UpdatedStaff,
		"remainingIssues": d.RemainingIssues,
	}
}

// ShiftScheduleDraftFromMap 从 map 恢复 ShiftScheduleDraft（用于从 session 读取）
func ShiftScheduleDraftFromMap(m map[string]any) *ShiftScheduleDraft {
	draft := NewShiftScheduleDraft()

	// 恢复 schedule
	if schedule, ok := m["schedule"].(map[string]any); ok {
		for date, staffsAny := range schedule {
			switch staffs := staffsAny.(type) {
			case []any:
				staffIDs := make([]string, 0, len(staffs))
				for _, s := range staffs {
					if sid, ok := s.(string); ok {
						staffIDs = append(staffIDs, sid)
					}
				}
				draft.Schedule[date] = staffIDs
			case []string:
				draft.Schedule[date] = staffs
			}
		}
	} else if schedule, ok := m["schedule"].(map[string][]string); ok {
		draft.Schedule = schedule
	}

	// 恢复 updatedStaff
	if updatedStaff, ok := m["updatedStaff"].(map[string]any); ok {
		for sid, v := range updatedStaff {
			if b, ok := v.(bool); ok && b {
				draft.UpdatedStaff[sid] = true
			}
		}
	} else if updatedStaff, ok := m["updatedStaff"].(map[string]bool); ok {
		draft.UpdatedStaff = updatedStaff
	}

	// 恢复 remainingIssues
	if issues, ok := m["remainingIssues"].([]any); ok {
		for _, issue := range issues {
			if issueStr, ok := issue.(string); ok {
				draft.RemainingIssues = append(draft.RemainingIssues, issueStr)
			}
		}
	} else if issues, ok := m["remainingIssues"].([]string); ok {
		draft.RemainingIssues = issues
	}

	return draft
}

// ShiftValidationResult 班次排班校验结果
type ShiftValidationResult struct {
	ShiftID       string             `json:"shiftId"`       // 班次ID
	IsValid       bool               `json:"isValid"`       // 是否通过校验
	Summary       string             `json:"summary"`       // 校验总结
	Issues        []*ValidationIssue `json:"issues"`        // 发现的问题列表
	Suggestions   []string           `json:"suggestions"`   // 改进建议
	AdjustedDraft *ShiftDraft        `json:"adjustedDraft"` // 调整后的草案（如有）
	ValidatedAt   string             `json:"validatedAt"`   // 校验时间
}

// ValidationIssue 校验发现的问题
type ValidationIssue struct {
	Type           string   `json:"type"`                     // 问题类型: rule_violation/staff_shortage/conflict/staff_count/shift_rule/rule_compliance等
	Severity       string   `json:"severity"`                 // 严重程度: critical/warning/info 或 high/medium/low
	Description    string   `json:"description"`              // 问题描述
	AffectedDates  []string `json:"affectedDates"`            // 受影响的日期
	AffectedStaff  []string `json:"affectedStaff"`            // 受影响的人员
	AffectedShifts []string `json:"affectedShifts,omitempty"` // 受影响的班次ID列表（可选）
	RuleID         string   `json:"ruleId,omitempty"`         // 相关规则ID（如有）
}

// ============================================================
// 共享排班上下文（用于 Create 和 Adjust 工作流复用）
// ============================================================

// Session.Data 中的数据键常量
const (
	DataKeyShiftSchedulingContext = "shift_scheduling_context"
)

// ShiftSchedulingContext 单个班次排班的共享上下文
// 用于 Create 工作流的三阶段排班和 Adjust 工作流的重排班次功能
type ShiftSchedulingContext struct {
	// 班次信息
	Shift     *Shift `json:"shift"`     // 当前排班的班次
	StartDate string `json:"startDate"` // 排班开始日期 YYYY-MM-DD
	EndDate   string `json:"endDate"`   // 排班结束日期 YYYY-MM-DD

	// 人员数据（仅该班次可用的人员）
	StaffList         []*Employee               `json:"staffList"`              // 可用人员列表
	AllStaffList      []*Employee               `json:"allStaffList,omitempty"` // 所有员工列表（用于姓名映射，确保显示正确的姓名而不是UUID）
	StaffRequirements map[string]int            `json:"staffRequirements"`      // 日期 -> 人数需求
	StaffLeaves       map[string][]*LeaveRecord `json:"staffLeaves"`            // staffId -> 请假记录

	// 规则数据
	GlobalRules []*Rule `json:"globalRules"` // 全局规则
	ShiftRules  []*Rule `json:"shiftRules"`  // 班次规则

	// 已有排班标记（其他班次的排班，避免时段冲突）
	ExistingScheduleMarks map[string]map[string][]*ShiftMark `json:"existingScheduleMarks"` // staffID -> date -> []ShiftMark

	// 固定排班人员（按日期组织，date -> staffIds）
	// 这些人员已经在固定班次中安排，绝对不能从当前班次中调整
	FixedShiftAssignments map[string][]string `json:"fixedShiftAssignments,omitempty"` // date -> staffIds

	// 临时需求列表（从用户调整需求中提取的临时需求）
	// 这些需求应该被考虑，确保不会安排已出差、请假或有事的员工
	TemporaryNeeds []*PersonalNeed `json:"temporaryNeeds,omitempty"` // 临时需求列表

	// 三阶段排班数据
	TodoPlan          *ShiftTodoPlan      `json:"todoPlan"`          // AI生成的Todo计划
	CurrentTodoIndex  int                 `json:"currentTodoIndex"`  // 当前执行的Todo索引
	ResultDraft       *ShiftScheduleDraft `json:"resultDraft"`       // 排班结果草案
	TodoExecutionLogs []string            `json:"todoExecutionLogs"` // 执行日志

	// 校验相关
	ValidationAttempts    int `json:"validationAttempts"`    // 校验尝试次数
	MaxValidationAttempts int `json:"maxValidationAttempts"` // 最大校验次数（默认3）

	// 来源标识
	SourceWorkflow string `json:"sourceWorkflow"` // 来源工作流: "create" / "adjust"

	// 状态标记
	Skipped bool `json:"skipped"` // 是否因错误被用户跳过

	// 进度回调（可选，用于 Create workflow 的实时进度更新）
	// 如果设置，则排班过程中的消息通过此回调发送，否则直接发送到 session
	ProgressCallback func(msg string) `json:"-"` // 不序列化
}

// NewShiftSchedulingContext 创建新的共享排班上下文
func NewShiftSchedulingContext(shift *Shift, startDate, endDate string, sourceWorkflow string) *ShiftSchedulingContext {
	return &ShiftSchedulingContext{
		Shift:                 shift,
		StartDate:             startDate,
		EndDate:               endDate,
		StaffList:             make([]*Employee, 0),
		AllStaffList:          make([]*Employee, 0),
		StaffRequirements:     make(map[string]int),
		StaffLeaves:           make(map[string][]*LeaveRecord),
		GlobalRules:           make([]*Rule, 0),
		ShiftRules:            make([]*Rule, 0),
		ExistingScheduleMarks: make(map[string]map[string][]*ShiftMark),
		FixedShiftAssignments: make(map[string][]string),
		TemporaryNeeds:        make([]*PersonalNeed, 0),
		TodoExecutionLogs:     make([]string, 0),
		MaxValidationAttempts: 3,
		SourceWorkflow:        sourceWorkflow,
	}
}

// ============================================================
// 排班调整工作流数据模型
// ============================================================

// Session.Data 中的数据键常量
const (
	DataKeyScheduleAdjustContext = "schedule_adjust_context"
)

// AdjustSourceType 调整来源类型
type AdjustSourceType string

const (
	AdjustSourceSessionDraft AdjustSourceType = "session_draft" // 当前会话草案
	AdjustSourceDateRange    AdjustSourceType = "date_range"    // 日期范围查询
	AdjustSourceUnknown      AdjustSourceType = "unknown"       // 未知，需要用户选择
)

// AdjustIntentType 调整意图类型
type AdjustIntentType string

const (
	AdjustIntentSwap       AdjustIntentType = "swap"       // 调班（两人互换）
	AdjustIntentReplace    AdjustIntentType = "replace"    // 替换（A替换B）
	AdjustIntentAdd        AdjustIntentType = "add"        // 添加人员
	AdjustIntentRemove     AdjustIntentType = "remove"     // 移除人员
	AdjustIntentModify     AdjustIntentType = "modify"     // 修改（通用）
	AdjustIntentBatch      AdjustIntentType = "batch"      // 批量调整
	AdjustIntentRegenerate AdjustIntentType = "regenerate" // 重新生成
	AdjustIntentCustom     AdjustIntentType = "custom"     // 自定义（AI理解）
	AdjustIntentOther      AdjustIntentType = "other"      // 其他
)

// ScheduleAdjustContext 排班调整工作流上下文
type ScheduleAdjustContext struct {
	// 来源信息
	SourceType    AdjustSourceType `json:"sourceType"`    // 来源类型
	StartDate     string           `json:"startDate"`     // 日期范围开始 YYYY-MM-DD
	EndDate       string           `json:"endDate"`       // 日期范围结束 YYYY-MM-DD
	SourceDraftID string           `json:"sourceDraftId"` // 来源草案ID（如果有）

	// 草案数据
	OriginalDraft   *ScheduleDraft `json:"originalDraft"`   // 原始草案（不可变，用于对比和回退）
	CurrentDraft    *ScheduleDraft `json:"currentDraft"`    // 当前草案（可修改）
	SelectedShiftID string         `json:"selectedShiftId"` // 当前调整的班次ID

	// 数据缓存
	AvailableShifts        []*Shift                  `json:"availableShifts"`        // 可用班次列表
	StaffList              []*Employee               `json:"staffList"`              // 可用人员列表
	ShiftStaffIDs          map[string][]string       `json:"shiftStaffIds"`          // shiftId -> []staffId (班次可用人员)
	ExistingSchedules      []*ScheduleAssignment     `json:"existingSchedules"`      // 日期范围内已有的排班
	ShiftScheduleCounts    map[string]int            `json:"shiftScheduleCounts"`    // 各班次在日期范围内的排班数量
	ShiftStaffRequirements map[string]map[string]int `json:"shiftStaffRequirements"` // 班次人数需求 map[shiftID]map[date]count

	// 规则缓存
	GlobalRules []*Rule            `json:"globalRules"` // 全局规则
	ShiftRules  map[string][]*Rule `json:"shiftRules"`  // shiftId -> rules

	// 修改意图
	UserIntent   string        `json:"userIntent"`   // 用户原始输入
	ParsedIntent *AdjustIntent `json:"parsedIntent"` // 解析后的意图

	// 调整方案
	AdjustmentPlan *AdjustmentPlan `json:"adjustmentPlan"` // AI生成的方案

	// 重排班次相关
	RegenerateOriginalShift *ShiftDraft `json:"regenerateOriginalShift"` // 重排前的原班次快照（用于差异对比）

	// 历史记录（支持撤销/重做）
	History      []*AdjustmentRecord `json:"history"`      // 修改历史栈
	HistoryIndex int                 `json:"historyIndex"` // 当前历史位置（-1表示无历史）

	// 日志
	AdjustmentLogs []string `json:"adjustmentLogs"` // 调整日志
}

// NewScheduleAdjustContext 创建新的调整上下文
func NewScheduleAdjustContext() *ScheduleAdjustContext {
	return &ScheduleAdjustContext{
		SourceType:     AdjustSourceUnknown,
		ShiftStaffIDs:  make(map[string][]string),
		GlobalRules:    make([]*Rule, 0),
		ShiftRules:     make(map[string][]*Rule),
		History:        make([]*AdjustmentRecord, 0),
		HistoryIndex:   -1,
		AdjustmentLogs: make([]string, 0),
	}
}

// AdjustIntent 解析后的调整意图
type AdjustIntent struct {
	Type             AdjustIntentType `json:"type"`             // 意图类型
	TargetDates      []string         `json:"targetDates"`      // 目标日期
	TargetStaff      []string         `json:"targetStaff"`      // 目标人员（被调整的）
	ReplacementStaff []string         `json:"replacementStaff"` // 替换人员（新安排的）
	TargetCount      int              `json:"targetCount"`      // 目标人数（批量调整时）
	Reason           string           `json:"reason"`           // 调整原因
	Confidence       float64          `json:"confidence"`       // AI理解的置信度 (0-1)
	RawDescription   string           `json:"rawDescription"`   // AI对意图的描述

	// 快捷操作使用的简化字段（与 TargetStaff/ReplacementStaff 二选一使用）
	Date    string `json:"date,omitempty"`    // 单个目标日期
	StaffA  string `json:"staffA,omitempty"`  // 员工A（源员工）
	StaffB  string `json:"staffB,omitempty"`  // 员工B（目标员工/替换员工）
	RawText string `json:"rawText,omitempty"` // 用户原始输入
}

// AdjustmentPlan 调整方案
type AdjustmentPlan struct {
	Summary   string            `json:"summary"`   // 方案摘要
	Changes   []*ScheduleChange `json:"changes"`   // 变更列表
	Impact    *AdjustmentImpact `json:"impact"`    // 影响评估
	Warnings  []string          `json:"warnings"`  // 警告信息
	CreatedAt string            `json:"createdAt"` // 生成时间
}

// ScheduleChange 单个排班变更
type ScheduleChange struct {
	Date          string   `json:"date"`          // 日期 YYYY-MM-DD
	ShiftID       string   `json:"shiftId"`       // 班次ID
	ShiftName     string   `json:"shiftName"`     // 班次名称
	ChangeType    string   `json:"changeType"`    // 变更类型: add/remove/replace/swap/modify
	OldStaff      []string `json:"oldStaff"`      // 原人员ID列表
	OldStaffNames []string `json:"oldStaffNames"` // 原人员姓名列表
	NewStaff      []string `json:"newStaff"`      // 新人员ID列表
	NewStaffNames []string `json:"newStaffNames"` // 新人员姓名列表
	Reason        string   `json:"reason"`        // 变更原因

	// 【V3新增】兼容字段（用于前端展示）
	BeforeIDs   []string `json:"beforeIds,omitempty"`   // 变更前人员ID列表（兼容）
	AfterIDs    []string `json:"afterIds,omitempty"`    // 变更后人员ID列表（兼容）
	BeforeNames []string `json:"beforeNames,omitempty"` // 变更前姓名列表（兼容）
	AfterNames  []string `json:"afterNames,omitempty"`  // 变更后姓名列表（兼容）

	// 【V3增强】工时变化信息（需前端渲染支持）
	WorkloadChanges map[string]*WorkloadChange `json:"workloadChanges,omitempty"` // 人员ID -> 工时变化
}

// WorkloadChange 人员工时变化
type WorkloadChange struct {
	StaffID        string  `json:"staffId"`        // 人员ID
	StaffName      string  `json:"staffName"`      // 人员姓名
	WorkloadBefore float64 `json:"workloadBefore"` // 变更前工时（小时）
	WorkloadAfter  float64 `json:"workloadAfter"`  // 变更后工时（小时）
	WorkloadDelta  float64 `json:"workloadDelta"`  // 工时变化（正数=增加，负数=减少）
}

// ScheduleChangeBatch 变更批次（V3新增：一个任务的所有变更）
type ScheduleChangeBatch struct {
	// TaskID 任务ID
	TaskID string `json:"taskId"`

	// TaskTitle 任务标题
	TaskTitle string `json:"taskTitle"`

	// TaskIndex 任务序号（从1开始）
	TaskIndex int `json:"taskIndex"`

	// Changes 变更列表
	Changes []*ScheduleChange `json:"changes"`

	// Timestamp 时间戳
	Timestamp string `json:"timestamp"`

	// 【V3增强】总工时变化（需前端渲染支持）
	TotalWorkloadDelta float64 `json:"totalWorkloadDelta,omitempty"` // 本批次导致的总工时变化（小时）

	// stats 延迟计算的统计信息（不序列化）
	stats *ChangeBatchStats `json:"-"`
}

// ChangeBatchStats 变更批次统计信息（V3新增）
type ChangeBatchStats struct {
	// AddCount 新增数量
	AddCount int

	// ModifyCount 修改数量
	ModifyCount int

	// RemoveCount 删除数量
	RemoveCount int

	// AffectedShifts 涉及的班次ID集合
	AffectedShifts map[string]bool

	// AffectedDates 涉及的日期集合
	AffectedDates map[string]bool

	// TotalStaffSlots 总人次（所有 AfterIDs 的数量之和）
	TotalStaffSlots int
}

// GetStats 获取统计信息（延迟计算）
func (batch *ScheduleChangeBatch) GetStats() *ChangeBatchStats {
	if batch.stats != nil {
		return batch.stats
	}

	stats := &ChangeBatchStats{
		AffectedShifts: make(map[string]bool),
		AffectedDates:  make(map[string]bool),
	}

	for _, change := range batch.Changes {
		// 统计变更类型
		switch change.ChangeType {
		case "add":
			stats.AddCount++
		case "modify":
			stats.ModifyCount++
		case "remove":
			stats.RemoveCount++
		}

		// 收集涉及的班次和日期
		stats.AffectedShifts[change.ShiftID] = true
		stats.AffectedDates[change.Date] = true

		// 累计总人次（使用变更后的人员数）
		afterIDs := change.AfterIDs
		if len(afterIDs) == 0 {
			afterIDs = change.NewStaff
		}
		stats.TotalStaffSlots += len(afterIDs)
	}

	batch.stats = stats
	return stats
}

// AdjustmentImpact 调整影响评估
type AdjustmentImpact struct {
	AffectedDates      []string       `json:"affectedDates"`      // 受影响的日期
	AffectedStaff      []string       `json:"affectedStaff"`      // 受影响的人员（姓名，用于显示）
	AffectedStaffIDs   []string       `json:"affectedStaffIds"`   // 受影响的人员ID
	AffectedStaffNames []string       `json:"affectedStaffNames"` // 受影响的人员姓名
	Conflicts          []string       `json:"conflicts"`          // 冲突描述列表
	Warnings           []string       `json:"warnings"`           // 警告信息
	RuleViolations     []string       `json:"ruleViolations"`     // 可能违反的规则
	WorkloadChanges    map[string]int `json:"workloadChanges"`    // 工作量变化 (staffID -> delta)
}

// AdjustmentRecord 调整历史记录（用于撤销/重做）
type AdjustmentRecord struct {
	ID            string            `json:"id"`            // 记录ID
	Timestamp     string            `json:"timestamp"`     // 时间戳
	Description   string            `json:"description"`   // 操作描述
	Changes       []*ScheduleChange `json:"changes"`       // 变更内容
	DraftSnapshot *ScheduleDraft    `json:"draftSnapshot"` // 操作后的草案快照
}

// CanUndo 检查是否可以撤销
func (ctx *ScheduleAdjustContext) CanUndo() bool {
	return ctx.HistoryIndex >= 0
}

// CanRedo 检查是否可以重做
func (ctx *ScheduleAdjustContext) CanRedo() bool {
	return ctx.HistoryIndex < len(ctx.History)-1
}

// PushHistory 添加历史记录
func (ctx *ScheduleAdjustContext) PushHistory(record *AdjustmentRecord) {
	// 如果当前不在历史末尾，截断后续历史
	if ctx.HistoryIndex < len(ctx.History)-1 {
		ctx.History = ctx.History[:ctx.HistoryIndex+1]
	}
	ctx.History = append(ctx.History, record)
	ctx.HistoryIndex = len(ctx.History) - 1
}

// Undo 撤销操作，返回上一个草案状态
func (ctx *ScheduleAdjustContext) Undo() *ScheduleDraft {
	if !ctx.CanUndo() {
		return nil
	}
	ctx.HistoryIndex--
	if ctx.HistoryIndex < 0 {
		// 回到原始状态
		return ctx.OriginalDraft
	}
	return ctx.History[ctx.HistoryIndex].DraftSnapshot
}

// Redo 重做操作，返回下一个草案状态
func (ctx *ScheduleAdjustContext) Redo() *ScheduleDraft {
	if !ctx.CanRedo() {
		return nil
	}
	ctx.HistoryIndex++
	return ctx.History[ctx.HistoryIndex].DraftSnapshot
}

// AddLog 添加日志
func (ctx *ScheduleAdjustContext) AddLog(log string) {
	ctx.AdjustmentLogs = append(ctx.AdjustmentLogs, log)
}

// ============================================================
// PersonalNeed - 个人需求定义
// ============================================================

// PersonalNeed 个人需求（常态化或临时）
type PersonalNeed struct {
	// StaffID 人员ID
	StaffID string `json:"staffId"`

	// StaffName 人员姓名
	StaffName string `json:"staffName"`

	// NeedType 需求类型: "permanent" (常态化) | "temporary" (临时)
	NeedType string `json:"needType"`

	// RequestType 请求类型: "prefer" (偏好) | "avoid" (回避) | "must" (必须)
	RequestType string `json:"requestType"`

	// TargetShiftID 目标班次ID（如果指定）
	TargetShiftID string `json:"targetShiftId,omitempty"`

	// TargetShiftName 目标班次名称
	TargetShiftName string `json:"targetShiftName,omitempty"`

	// TargetDates 目标日期列表 (YYYY-MM-DD)
	// 为空表示整个周期都生效
	TargetDates []string `json:"targetDates,omitempty"`

	// Description 需求描述
	Description string `json:"description"`

	// Priority 优先级 (1-10, 数字越小优先级越高)
	Priority int `json:"priority"`

	// RuleID 关联的规则ID（如果来自规则系统）
	RuleID string `json:"ruleId,omitempty"`

	// Source 来源: "rule" (从规则提取) | "user" (用户补充)
	Source string `json:"source"`

	// Confirmed 是否已确认
	Confirmed bool `json:"confirmed"`
}

// ============================================================
// UnavailableStaffMap - 不可用人员清单
// ============================================================

// UnavailableStaffMap 不可用人员清单
// 表示哪些人员在哪些日期不可用（因请假、回避等原因）
type UnavailableStaffMap struct {
	// StaffDates 人员不可用日期映射 (staffID -> 日期集合)
	StaffDates map[string]map[string]bool `json:"staffDates"`
}

// NewUnavailableStaffMap 创建新的不可用人员清单
func NewUnavailableStaffMap() *UnavailableStaffMap {
	return &UnavailableStaffMap{
		StaffDates: make(map[string]map[string]bool),
	}
}

// IsUnavailable 检查指定人员在指定日期是否不可用
func (m *UnavailableStaffMap) IsUnavailable(staffID, date string) bool {
	if dates, ok := m.StaffDates[staffID]; ok {
		return dates[date]
	}
	return false
}

// AddUnavailable 添加不可用记录
func (m *UnavailableStaffMap) AddUnavailable(staffID, date string) {
	if m.StaffDates[staffID] == nil {
		m.StaffDates[staffID] = make(map[string]bool)
	}
	m.StaffDates[staffID][date] = true
}

// AddUnavailableDates 批量添加不可用日期
func (m *UnavailableStaffMap) AddUnavailableDates(staffID string, dates []string) {
	if m.StaffDates[staffID] == nil {
		m.StaffDates[staffID] = make(map[string]bool)
	}
	for _, date := range dates {
		m.StaffDates[staffID][date] = true
	}
}

// IsUnavailableInAnyDate 检查指定人员是否在目标日期列表中的任何日期不可用
func (m *UnavailableStaffMap) IsUnavailableInAnyDate(staffID string, targetDates []string) bool {
	if len(targetDates) == 0 {
		// 如果没有目标日期，检查是否整个周期都不可用（通过检查是否有任何日期记录）
		_, hasAnyDate := m.StaffDates[staffID]
		return hasAnyDate
	}

	dates, ok := m.StaffDates[staffID]
	if !ok {
		return false
	}

	// 检查是否在任何一个目标日期不可用
	for _, date := range targetDates {
		if dates[date] {
			return true
		}
	}
	return false
}

// ============================================================
// 变更预览数据结构（V3改进：使用强类型替代嵌套map）
// ============================================================

// ChangeDetailPreview 变更详情预览（前端展示用的强类型结构）
type ChangeDetailPreview struct {
	// TaskID 任务ID
	TaskID string `json:"taskId"`

	// TaskTitle 任务标题
	TaskTitle string `json:"taskTitle"`

	// TaskIndex 任务序号（从1开始）
	TaskIndex int `json:"taskIndex"`

	// Timestamp 时间戳
	Timestamp string `json:"timestamp"`

	// Shifts 班次变更列表（使用数组而非嵌套map）
	Shifts []*ShiftChangePreview `json:"shifts"`
}

// ShiftChangePreview 班次变更预览
type ShiftChangePreview struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`

	// Changes 该班次下的所有日期变更（使用数组而非map）
	Changes []*DateChangePreview `json:"changes"`
}

// DateChangePreview 日期变更预览
type DateChangePreview struct {
	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// ChangeType 变更类型
	ChangeType string `json:"changeType"`

	// BeforeIDs 变更前人员ID列表
	BeforeIDs []string `json:"beforeIds"`

	// AfterIDs 变更后人员ID列表
	AfterIDs []string `json:"afterIds"`

	// BeforeNames 变更前人员姓名（用于展示）
	BeforeNames []string `json:"before"`

	// AfterNames 变更后人员姓名（用于展示）
	AfterNames []string `json:"after"`
}

// ============================================================
// AI输入数据结构（V3改进：使用强类型替代嵌套map）
// ============================================================

// DailyRequirement 每日人员需求
type DailyRequirement struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称（便于AI理解）
	ShiftName string `json:"shiftName"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// RequiredCount 需要的人数
	RequiredCount int `json:"requiredCount"`

	// CurrentCount 当前已安排人数（如有）
	CurrentCount int `json:"currentCount,omitempty"`
}

// FixedAssignmentForAI 固定排班分配（用于AI输入）
type FixedAssignmentForAI struct {
	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`

	// StaffIDs 已固定安排的人员ID列表
	StaffIDs []string `json:"staffIds"`

	// StaffNames 已固定安排的人员姓名（用于展示）
	StaffNames []string `json:"staffNames"`

	// IsFixedSchedule 标识这是固定排班（与AI生成的排班区分）
	IsFixedSchedule bool `json:"isFixedSchedule"`
}

// ============================================================
// 排班上下文数据结构（V3 LLM增强：提供完整排班上下文）
// ============================================================

// AssignedShiftInfo 已分配的班次信息（包含完整时间信息）
type AssignedShiftInfo struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称
	ShiftName string `json:"shiftName"`

	// StartTime 开始时间 (HH:MM)
	StartTime string `json:"startTime"`

	// EndTime 结束时间 (HH:MM)
	EndTime string `json:"endTime"`

	// Duration 时长（小时）
	Duration float64 `json:"duration"`

	// IsOvernight 是否跨夜
	IsOvernight bool `json:"isOvernight"`

	// IsFixed 是否固定排班（不可调整）
	IsFixed bool `json:"isFixed"`

	// TaskID 来源任务ID（用于追溯是哪个任务创建的排班，固定排班为空）
	TaskID string `json:"taskId,omitempty"`
}

// StaffCurrentSchedule 人员当前排班状态（某一天）
type StaffCurrentSchedule struct {
	// StaffID 人员ID
	StaffID string `json:"staffId"`

	// StaffName 人员姓名
	StaffName string `json:"staffName"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// Shifts 该天已安排的班次列表
	Shifts []*AssignedShiftInfo `json:"shifts"`

	// TotalHours 该天总工作时长（小时）
	TotalHours float64 `json:"totalHours"`

	// Errors 阻断性错误（如时间冲突、违反硬约束），必须解决
	Errors []string `json:"errors,omitempty"`

	// Warnings 提示性警告（如接近超时、建议优化），可以容忍
	Warnings []string `json:"warnings,omitempty"`
}

// V3SchedulingContext 排班上下文快照（V3增强：传递给LLM的完整信息）
type V3SchedulingContext struct {
	// ========== 当前任务信息 ==========

	// TargetDate 当前处理的日期
	TargetDate string `json:"targetDate"`

	// TargetShiftID 当前处理的班次ID
	TargetShiftID string `json:"targetShiftId"`

	// TargetShiftName 当前处理的班次名称
	TargetShiftName string `json:"targetShiftName"`

	// TargetShiftTime 当前班次的时间段（便于展示）
	TargetShiftTime string `json:"targetShiftTime"` // 如 "08:00-16:00"

	// RequiredCount 需要的人数
	RequiredCount int `json:"requiredCount"`

	// ========== 全局信息 ==========

	// AllShifts 所有班次信息（包含完整时间）
	AllShifts []*Shift `json:"allShifts"`

	// StaffSchedules 人员当前排班状态（该日期）
	StaffSchedules []*StaffCurrentSchedule `json:"staffSchedules"`

	// ========== 约束配置 ==========

	// MaxDailyHours 单日最大工作时长（小时）
	MaxDailyHours float64 `json:"maxDailyHours"`

	// MinRestHours 班次间最小休息时长（小时）
	MinRestHours float64 `json:"minRestHours"`
}
