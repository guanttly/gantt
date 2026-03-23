package adjust

import (
	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// AdjustV2Context - 排班调整子工作流 V2 上下文
// ============================================================

// AdjustV2Context 调整子工作流的核心上下文
type AdjustV2Context struct {
	// ========== 用户输入 ==========

	// UserMessage 用户调整需求消息
	UserMessage string `json:"userMessage"`

	// ========== 班次信息 ==========

	// ShiftID 当前班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 当前班次名称
	ShiftName string `json:"shiftName"`

	// ========== 排班周期 ==========

	// StartDate 排班周期开始日期 (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// EndDate 排班周期结束日期 (YYYY-MM-DD)
	EndDate string `json:"endDate"`

	// ========== 排班数据 ==========

	// OriginalDraft 上次排班结果（ShiftScheduleDraft）
	OriginalDraft *d_model.ShiftScheduleDraft `json:"originalDraft"`

	// ResultDraft 调整后的排班结果
	ResultDraft *d_model.ShiftScheduleDraft `json:"resultDraft"`

	// AdjustSummary AI的调整总结说明
	AdjustSummary string `json:"adjustSummary"`

	// AdjustChanges 排班变化列表（用于对比显示）
	AdjustChanges []d_model.AdjustScheduleChange `json:"adjustChanges"`

	// ========== 基础数据 ==========

	// StaffList 可用人员列表
	StaffList []*d_model.Employee `json:"staffList"`

	// AllStaffList 所有员工列表
	AllStaffList []*d_model.Employee `json:"allStaffList"`

	// StaffRequirements 人数需求 (date -> count)
	StaffRequirements map[string]int `json:"staffRequirements"`

	// StaffLeaves 人员请假信息 (staff_id -> 请假记录列表)
	StaffLeaves map[string][]*d_model.LeaveRecord `json:"staffLeaves"`

	// Rules 所有排班规则
	Rules []*d_model.Rule `json:"rules"`

	// ExistingScheduleMarks 已有排班标记（用于时段冲突检查）
	ExistingScheduleMarks map[string]map[string][]*d_model.ShiftMark `json:"existingScheduleMarks"`

	// FixedShiftAssignments 固定排班人员（按日期组织，date -> staffIds）
	// 这些人员已经在固定班次中安排，绝对不能从当前班次中调整
	FixedShiftAssignments map[string][]string `json:"fixedShiftAssignments"`

	// TemporaryNeeds 从用户消息中提取的临时需求列表
	// 这些需求应该被添加到临时需求列表中，以便后续排班时能够响应
	TemporaryNeeds []*d_model.PersonalNeed `json:"temporaryNeeds"`
}

// NewAdjustV2Context 创建新的调整上下文
func NewAdjustV2Context() *AdjustV2Context {
	return &AdjustV2Context{
		StaffRequirements:     make(map[string]int),
		StaffLeaves:           make(map[string][]*d_model.LeaveRecord),
		Rules:                 make([]*d_model.Rule, 0),
		ExistingScheduleMarks: make(map[string]map[string][]*d_model.ShiftMark),
		FixedShiftAssignments: make(map[string][]string),
	}
}
