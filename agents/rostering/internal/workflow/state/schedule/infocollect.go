// Package schedule 定义信息收集子工作流的状态和事件
package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// 信息收集子工作流定义
// 可被 Create 和 Adjust 工作流调用，支持多分支收集不同信息
// ============================================================

const (
	// WorkflowInfoCollect 信息收集子工作流
	WorkflowInfoCollect engine.Workflow = "schedule.info-collect"
)

// ============================================================
// schedule.info-collect 子工作流 - 状态定义
// 支持多分支：完整收集、跳过周期、跳过班次、仅数据检索
// ============================================================

const (
	// 初始状态（分支起点）
	InfoCollectStateInit engine.State = "_info_collect_init_"

	// 阶段1: 确认排班周期
	InfoCollectStateConfirmingPeriod engine.State = "_info_collect_confirming_period_"

	// 阶段2: 查询可用班次
	InfoCollectStateQueryingShifts engine.State = "_info_collect_querying_shifts_"

	// 阶段3: 确认排班班次
	InfoCollectStateConfirmingShifts engine.State = "_info_collect_confirming_shifts_"

	// 阶段4: 确认班次人数
	InfoCollectStateConfirmingStaffCount engine.State = "_info_collect_confirming_staff_count_"

	// 阶段5: 检索可用人员
	InfoCollectStateRetrievingStaff engine.State = "_info_collect_retrieving_staff_"

	// 阶段6: 检索排班规则
	InfoCollectStateRetrievingRules engine.State = "_info_collect_retrieving_rules_"

	// 终态
	InfoCollectStateCompleted engine.State = "_info_collect_completed_"
	InfoCollectStateCancelled engine.State = "_info_collect_cancelled_"
)

// ============================================================
// schedule.info-collect 子工作流 - 事件定义
// ============================================================

const (
	// 启动事件（分支选择）
	InfoCollectEventStart      engine.Event = "_info_collect_start_"        // 完整流程
	InfoCollectEventSkipPeriod engine.Event = "_info_collect_skip_period_"  // 跳过周期确认
	InfoCollectEventSkipShifts engine.Event = "_info_collect_skip_shifts_"  // 跳过班次选择
	InfoCollectEventSkipToData engine.Event = "_info_collect_skip_to_data_" // 直接数据检索

	// 周期确认事件
	InfoCollectEventPeriodConfirmed engine.Event = "_info_collect_period_confirmed_"
	InfoCollectEventPeriodModified  engine.Event = "_info_collect_period_modified_"

	// 班次查询事件
	InfoCollectEventShiftsQueried engine.Event = "_info_collect_shifts_queried_"

	// 班次确认事件
	InfoCollectEventShiftsConfirmed engine.Event = "_info_collect_shifts_confirmed_"
	InfoCollectEventShiftsModified  engine.Event = "_info_collect_shifts_modified_"

	// 人数确认事件
	InfoCollectEventStaffCountConfirmed engine.Event = "_info_collect_staff_count_confirmed_"
	InfoCollectEventStaffCountModified  engine.Event = "_info_collect_staff_count_modified_"

	// 数据检索事件
	InfoCollectEventStaffRetrieved engine.Event = "_info_collect_staff_retrieved_"
	InfoCollectEventRulesRetrieved engine.Event = "_info_collect_rules_retrieved_"

	// 完成和取消事件
	InfoCollectEventComplete engine.Event = "_info_collect_complete_"
	InfoCollectEventCancel   engine.Event = "_info_collect_cancel_"
	InfoCollectEventError    engine.Event = "_info_collect_error_"
	InfoCollectEventReturn   engine.Event = "_info_collect_return_"
)

// ============================================================
// schedule.info-collect 子工作流 - 输入输出结构
// ============================================================

// InfoCollectInput 信息收集子工作流输入
type InfoCollectInput struct {
	// SourceType 来源类型: "create" | "adjust" | "regenerate"
	SourceType string `json:"source_type"`

	// 预设值（可选，用于跳过某些阶段）
	PresetStartDate string   `json:"preset_start_date,omitempty"`
	PresetEndDate   string   `json:"preset_end_date,omitempty"`
	PresetShiftIDs  []string `json:"preset_shift_ids,omitempty"`

	// SkipPhases 跳过的阶段列表
	// 可选值: "period", "shifts", "staff_count"
	SkipPhases []string `json:"skip_phases,omitempty"`
}

// InfoCollectOutput 信息收集子工作流输出
type InfoCollectOutput struct {
	// 周期信息
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`

	// 班次信息（序列化后的 JSON，避免循环引用）
	SelectedShiftIDs []string `json:"selected_shift_ids"`

	// 人数需求 map[shiftID]map[date]count
	ShiftStaffRequirements map[string]map[string]int `json:"shift_staff_requirements"`

	// 人员信息
	StaffIDs []string `json:"staff_ids"`

	// 请假信息 map[staffID][]leaveDate
	StaffLeaveDates map[string][]string `json:"staff_leave_dates"`

	// 规则数量统计（详细规则存在 session.Data 中）
	GlobalRuleCount   int            `json:"global_rule_count"`
	ShiftRuleCount    map[string]int `json:"shift_rule_count"`
	GroupRuleCount    map[string]int `json:"group_rule_count"`
	EmployeeRuleCount map[string]int `json:"employee_rule_count"`
}

// ============================================================
// schedule.info-collect 子工作流 - 状态描述
// ============================================================

var InfoCollectStateDescriptions = map[engine.State]string{
	InfoCollectStateInit:                 "初始化信息收集",
	InfoCollectStateConfirmingPeriod:     "确认排班周期",
	InfoCollectStateQueryingShifts:       "查询可用班次",
	InfoCollectStateConfirmingShifts:     "确认排班班次",
	InfoCollectStateConfirmingStaffCount: "确认班次人数",
	InfoCollectStateRetrievingStaff:      "检索可用人员",
	InfoCollectStateRetrievingRules:      "检索排班规则",
	InfoCollectStateCompleted:            "信息收集完成",
	InfoCollectStateCancelled:            "信息收集已取消",
}

// GetInfoCollectStateDescription 获取信息收集子工作流状态描述
func GetInfoCollectStateDescription(state engine.State) string {
	if desc, ok := InfoCollectStateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}

// ShouldSkipPhase 检查是否应该跳过某个阶段
func (input *InfoCollectInput) ShouldSkipPhase(phase string) bool {
	for _, p := range input.SkipPhases {
		if p == phase {
			return true
		}
	}
	return false
}
