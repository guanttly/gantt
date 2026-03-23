// Package schedule 定义排班相关工作流的状态和事件常量
package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule.regenerate 子工作流 - 状态定义
// 用于重新生成班次排班的独立子工作流
// ============================================================

const (
	// 子工作流名称
	WorkflowRegenerate = "schedule.regenerate"

	// ========== 状态定义 ==========
	// 初始状态
	RegenerateStateInit engine.State = "_regenerate_init_"

	// 准备阶段
	RegenerateStatePreparing engine.State = "_regenerate_preparing_" // 准备重排数据

	// 确认人数（当需要用户确认时）
	RegenerateStateConfirmingCount engine.State = "_regenerate_confirming_count_"

	// 三阶段排班
	RegenerateStateTodo     engine.State = "_regenerate_todo_"     // 生成Todo计划
	RegenerateStateExec     engine.State = "_regenerate_exec_"     // 执行Todo任务
	RegenerateStateValidate engine.State = "_regenerate_validate_" // 校验排班结果

	// 终态
	RegenerateStateCompleted engine.State = engine.StateCompleted // 成功完成
	RegenerateStateFailed    engine.State = engine.StateFailed    // 失败
	RegenerateStateCancelled engine.State = engine.StateCancelled // 取消
)

// ============================================================
// schedule.regenerate 子工作流 - 事件定义
// ============================================================

const (
	// 启动事件
	RegenerateEventStart engine.Event = "_regenerate_start_"

	// 准备阶段事件
	RegenerateEventPrepared       engine.Event = "_regenerate_prepared_"         // 准备完成（有草案数据）
	RegenerateEventNeedStaffCount engine.Event = "_regenerate_need_staff_count_" // 需要确认人数
	RegenerateEventStaffCountDone engine.Event = "_regenerate_staff_count_done_" // 人数确认完成
	RegenerateEventPrepareFailed  engine.Event = "_regenerate_prepare_failed_"   // 准备失败

	// 三阶段事件
	RegenerateEventTodoGenerated engine.Event = "_regenerate_todo_generated_" // Todo计划生成完成
	RegenerateEventTodosExecuted engine.Event = "_regenerate_todos_executed_" // Todos执行完成
	RegenerateEventValidated     engine.Event = "_regenerate_validated_"      // 校验完成

	// 失败/取消事件
	RegenerateEventFailed  engine.Event = "_regenerate_failed_"  // 执行失败
	RegenerateEventAborted engine.Event = "_regenerate_aborted_" // 用户放弃
)

// ============================================================
// 状态描述（用于前端显示）
// ============================================================

var RegenerateStateDescriptions = map[engine.State]string{
	RegenerateStateInit:            "初始化重排",
	RegenerateStatePreparing:       "准备重排数据",
	RegenerateStateConfirmingCount: "确认排班人数",
	RegenerateStateTodo:            "生成排班计划",
	RegenerateStateExec:            "执行排班任务",
	RegenerateStateValidate:        "校验排班结果",
	RegenerateStateCompleted:       "重排完成",
	RegenerateStateFailed:          "重排失败",
	RegenerateStateCancelled:       "已取消重排",
}

// ============================================================
// 子工作流输入/输出定义
// ============================================================

// RegenerateInput 重排子工作流输入参数
type RegenerateInput struct {
	// 班次信息
	ShiftID   string `json:"shiftId"`
	ShiftName string `json:"shiftName"`

	// 排班周期
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`

	// 人数配置（可选，如果为空则需要用户确认）
	StaffRequirements map[string]int `json:"staffRequirements,omitempty"`

	// 是否跳过人数确认
	SkipStaffCountConfirm bool `json:"skipStaffCountConfirm,omitempty"`
}

// RegenerateOutput 重排子工作流输出结果
type RegenerateOutput struct {
	// 是否成功
	Success bool `json:"success"`

	// 生成的排班草案
	ShiftDraft interface{} `json:"shiftDraft,omitempty"` // *ShiftScheduleDraft

	// 错误信息（如果失败）
	Error string `json:"error,omitempty"`

	// 执行统计
	TodoCount      int `json:"todoCount"`
	CompletedCount int `json:"completedCount"`
	FailedCount    int `json:"failedCount"`
}
