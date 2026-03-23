package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule.collect-staff-count 子工作流 - 收集班次人数需求
// ============================================================

const (
	// WorkflowCollectStaffCount 收集班次人数需求子工作流名称
	WorkflowCollectStaffCount = "schedule.collect-staff-count"
)

// ============================================================
// 状态定义
// ============================================================

const (
	// CollectStaffCountStateInit 初始状态
	CollectStaffCountStateInit engine.State = "init"

	// CollectStaffCountStateConfirming 确认人数
	CollectStaffCountStateConfirming engine.State = "confirming"

	// CollectStaffCountStateCompleted 完成
	CollectStaffCountStateCompleted engine.State = "completed"

	// CollectStaffCountStateCancelled 取消
	CollectStaffCountStateCancelled engine.State = "cancelled"
)

// ============================================================
// 事件定义
// ============================================================

const (
	// CollectStaffCountEventConfirmed 人数已确认
	CollectStaffCountEventConfirmed engine.Event = "staff_count_confirmed"

	// CollectStaffCountEventModified 人数已修改
	CollectStaffCountEventModified engine.Event = "staff_count_modified"

	// CollectStaffCountEventReturn 返回父工作流
	CollectStaffCountEventReturn engine.Event = "return"

	// CollectStaffCountEventCancel 取消
	CollectStaffCountEventCancel engine.Event = "cancel"
)

// ============================================================
// 输入输出定义
// ============================================================

// CollectStaffCountInput 收集班次人数输入
type CollectStaffCountInput struct {
	// StartDate 排班开始日期
	StartDate string `json:"start_date"`

	// EndDate 排班结束日期
	EndDate string `json:"end_date"`

	// ShiftIDs 班次ID列表
	ShiftIDs []string `json:"shift_ids"`

	// OrgID 组织ID（可选，从 session 获取）
	OrgID string `json:"org_id,omitempty"`
}

// CollectStaffCountOutput 收集班次人数输出
type CollectStaffCountOutput struct {
	// ShiftStaffRequirements 班次人数需求列表
	ShiftStaffRequirements []ShiftDailyRequirement `json:"shift_staff_requirements"`
}

// ShiftDailyRequirement 班次每日人数需求
type ShiftDailyRequirement struct {
	// ShiftID 班次ID
	ShiftID string `json:"shift_id"`

	// DailyRequirements 每日人数需求列表（仅包含需要排班的日期，staffCount > 0）
	DailyRequirements []DailyStaffRequirement `json:"daily_requirements"`
}

// DailyStaffRequirement 每日人数需求
type DailyStaffRequirement struct {
	// Date 日期 YYYY-MM-DD
	Date string `json:"date"`

	// StaffCount 所需人数（必须 > 0）
	StaffCount int `json:"staff_count"`
}

// ============================================================
// 状态描述
// ============================================================

var CollectStaffCountStateDescriptions = map[engine.State]string{
	CollectStaffCountStateInit:       "初始化人数收集",
	CollectStaffCountStateConfirming: "确认班次人数",
	CollectStaffCountStateCompleted:  "人数收集完成",
	CollectStaffCountStateCancelled:  "人数收集已取消",
}

// GetCollectStaffCountStateDescription 获取状态描述
func GetCollectStaffCountStateDescription(state engine.State) string {
	if desc, ok := CollectStaffCountStateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}
