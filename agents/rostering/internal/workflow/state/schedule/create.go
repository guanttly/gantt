package schedule

import "jusha/mcp/pkg/workflow/engine"

const (
	// WorkflowScheduleCreate 排班创建工作流
	WorkflowScheduleCreate engine.Workflow = "schedule.create"
)

// ============================================================
// schedule.create 工作流（重构版）- 状态定义
// 新架构：Create 作为父工作流，编排三个子工作流
// 1. InfoCollect 子工作流 - 信息收集
// 2. Core 子工作流 - 核心排班（循环执行每个班次）
// 3. ConfirmSave 子工作流 - 确认保存
// ============================================================

const (
	// ========== 父工作流主状态 ==========

	// 初始状态
	CreateStateInit engine.State = "_schedule_create_init_"

	// 信息收集阶段（调用 InfoCollect 子工作流）
	CreateStateInfoCollecting engine.State = "_schedule_create_info_collecting_"

	// 核心排班阶段（循环调用 Core 子工作流）
	CreateStateCoreScheduling engine.State = "_schedule_create_core_scheduling_"

	// 确认保存阶段（调用 ConfirmSave 子工作流）
	CreateStateConfirmSaving engine.State = "_schedule_create_confirm_saving_"

	// ========== 旧状态（保留用于兼容，逐步迁移后删除） ==========

	// 阶段1: 确认排班周期
	StateConfirmingPeriod engine.State = "_schedule_create_confirming_period_"

	// 阶段2: 查询可用班次
	StateQueryingShifts engine.State = "_schedule_create_querying_shifts_"

	// 阶段3: 确认排班班次
	StateConfirmingShifts engine.State = "_schedule_create_confirming_shifts_"

	// 阶段4: 确认班次人数
	StateConfirmingStaffCount engine.State = "_schedule_create_confirming_staff_count_"

	// 阶段5: 检索班次相关人员
	StateRetrievingStaff engine.State = "_schedule_create_retrieving_staff_"

	// 阶段5: 检索班次相关规则
	StateRetrievingRules engine.State = "_schedule_create_retrieving_rules_"

	// 阶段6: 生成排班
	StateGeneratingSchedule engine.State = "_schedule_create_generating_schedule_"

	// 阶段6子状态（三阶段优化）
	StateGeneratingTodoPlan engine.State = "_schedule_create_generating_todo_plan_" // 生成Todo计划
	StateExecutingTodoTasks engine.State = "_schedule_create_executing_todo_tasks_" // 执行Todo任务
	StateValidatingSchedule engine.State = "_schedule_create_validating_schedule_"  // 校验排班结果

	// 旧的子状态（保留向后兼容）
	StateQueryingShiftGroupRules engine.State = "_schedule_create_querying_shift_group_rules_"
	StateQueryingShiftStaffRules engine.State = "_schedule_create_querying_shift_staff_rules_"
	StateGeneratingShiftSchedule engine.State = "_schedule_create_generating_shift_schedule_"
	StateMergingDraft            engine.State = "_schedule_create_merging_draft_"

	// 阶段7: 预览并确认排班
	StateConfirmingDraft engine.State = "_schedule_create_confirming_draft_"

	// 阶段8: 存储排班
	StateSavingSchedule engine.State = "_schedule_create_saving_schedule_"

	// 终态 - 复用通用常量
	StateCompleted = engine.StateCompleted // "_completed_"
	StateFailed    = engine.StateFailed    // "_failed_"
	StateCancelled = engine.StateCancelled // "_cancelled_"
)

// ============================================================
// schedule.create 工作流（重构版）- 事件定义
// ============================================================

const (
	// ========== 父工作流事件（新增） ==========

	// 启动事件
	CreateEventStart engine.Event = "_schedule_create_start_"

	// 子工作流完成事件
	CreateEventInfoCollected  engine.Event = "_schedule_create_info_collected_"  // 信息收集完成
	CreateEventShiftCompleted engine.Event = "_schedule_create_shift_completed_" // 单个班次排班完成
	CreateEventAllShiftsDone  engine.Event = "_schedule_create_all_shifts_done_" // 所有班次排班完成
	CreateEventSaveCompleted  engine.Event = "_schedule_create_save_completed_"  // 保存完成
	CreateEventSubCancelled   engine.Event = "_schedule_create_sub_cancelled_"   // 子工作流取消
	CreateEventSubFailed      engine.Event = "_schedule_create_sub_failed_"      // 子工作流失败

	// 用户操作事件
	CreateEventUserCancel engine.Event = "_schedule_create_user_cancel_" // 用户取消

	// ========== 旧事件（保留用于兼容） ==========

	// 通用生命周期事件 - 直接复用
	EventStart  = engine.EventStart  // "_start_"
	EventCancel = engine.EventCancel // "_cancel_"

	// 通用确认事件 - 直接复用（业务特定的确认使用业务前缀）
	EventConfirm = engine.EventConfirm // "_confirm_" - 用于通用确认操作
	EventModify  = engine.EventModify  // "_modify_"  - 用于通用修改操作
	EventReject  = engine.EventReject  // "_reject_"  - 用于拒绝操作

	// 周期确认事件(业务特定)
	EventPeriodConfirmed engine.Event = "_schedule_create_period_confirmed_"
	EventPeriodModified  engine.Event = "_schedule_create_period_modified_"

	// 班次查询事件(业务特定)
	EventShiftsQueried engine.Event = "_schedule_create_shifts_queried_"

	// 班次确认事件(业务特定)
	EventShiftsConfirmed engine.Event = "_schedule_create_shifts_confirmed_"
	EventShiftsModified  engine.Event = "_schedule_create_shifts_modified_"

	// 人数确认事件（业务特定）
	EventStaffCountConfirmed engine.Event = "_schedule_create_staff_count_confirmed_"
	EventStaffCountModified  engine.Event = "_schedule_create_staff_count_modified_"

	// 数据检索完成事件（业务特定）
	EventStaffRetrieved engine.Event = "_schedule_create_staff_retrieved_"
	EventRulesRetrieved engine.Event = "_schedule_create_rules_retrieved_"

	// 生成进度事件（业务特定）
	EventShiftProcessed    engine.Event = "_schedule_create_shift_processed_"
	EventAllShiftsComplete engine.Event = "_schedule_create_all_shifts_complete_"

	// 三阶段排班事件（新增）
	EventTodosPlanGenerated     engine.Event = "_schedule_create_todos_plan_generated_"     // Todo计划生成完成
	EventTodosExecutionComplete engine.Event = "_schedule_create_todos_execution_complete_" // 所有Todo任务执行完成
	EventValidationComplete     engine.Event = "_schedule_create_validation_complete_"      // 校验完成（无论是否通过）
	EventValidationFailed       engine.Event = "_schedule_create_validation_failed_"        // 校验失败（严重问题）
	EventValidationRetry        engine.Event = "_schedule_create_validation_retry_"         // 重试校验

	// 草案确认事件（业务特定）
	EventDraftConfirmed engine.Event = "_schedule_create_draft_confirmed_"
	EventDraftRejected  engine.Event = "_schedule_create_draft_rejected_"

	// 保存完成事件（业务特定）
	EventSaveSuccess engine.Event = "_schedule_create_save_success_"
	EventSaveFailed  engine.Event = "_schedule_create_save_failed_"

	// 错误事件（业务特定）
	EventUserCancelled engine.Event = "_schedule_create_user_cancelled_"
	EventAIFailed      engine.Event = "_schedule_create_ai_failed_"
	EventSystemError   engine.Event = "_schedule_create_system_error_"

	// 重试事件（业务特定）
	EventRetryGeneration engine.Event = "_schedule_create_retry_generation_"
	EventRetrySave       engine.Event = "_schedule_create_retry_save_"
)

// ============================================================
// schedule.create 工作流 - 状态描述（用于前端显示）
// ============================================================

var StateDescriptions = map[engine.State]string{
	// 新架构状态
	CreateStateInit:           "排班初始化",
	CreateStateInfoCollecting: "收集排班信息",
	CreateStateCoreScheduling: "生成排班方案",
	CreateStateConfirmSaving:  "确认并保存",

	// 旧状态（保留兼容）
	StateConfirmingPeriod:        "确认排班周期",
	StateQueryingShifts:          "查询可用班次",
	StateConfirmingShifts:        "确认排班班次",
	StateConfirmingStaffCount:    "确认班次人数",
	StateRetrievingStaff:         "检索可用人员",
	StateRetrievingRules:         "检索排班规则",
	StateGeneratingSchedule:      "生成排班中",
	StateGeneratingTodoPlan:      "生成排班计划",
	StateExecutingTodoTasks:      "执行排班任务",
	StateValidatingSchedule:      "校验排班结果",
	StateQueryingShiftGroupRules: "查询班次分组规则",
	StateQueryingShiftStaffRules: "查询人员规则",
	StateGeneratingShiftSchedule: "AI生成班次排班",
	StateMergingDraft:            "合并排班草案",
	StateConfirmingDraft:         "预览排班草案",
	StateSavingSchedule:          "保存排班",
	StateCompleted:               "排班完成",
	StateFailed:                  "排班失败",
	StateCancelled:               "已取消",
}

// GetStateDescription 获取状态的中文描述
func GetStateDescription(state engine.State) string {
	if desc, ok := StateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}
