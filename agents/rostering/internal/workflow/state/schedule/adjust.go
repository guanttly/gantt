package schedule

import "jusha/mcp/pkg/workflow/engine"

const (
	WorkflowScheduleAdjust engine.Workflow = "schedule.adjust"
)

// ============================================================
// schedule.adjust 工作流 - 状态定义
// 命名规范: _[workflow]_[sub]_[state]_
// ============================================================

const (
	// 阶段1: 初始化/解析来源
	StateAdjustInit engine.State = "_schedule_adjust_init_"

	// 阶段2: 加载排班数据
	StateAdjustLoadingData engine.State = "_schedule_adjust_loading_data_"

	// 阶段3: 选择日期范围（当无法从语义提取时）
	StateAdjustSelectingDateRange engine.State = "_schedule_adjust_selecting_date_range_"

	// 阶段4: 选择班次
	StateAdjustSelectingShift engine.State = "_schedule_adjust_selecting_shift_"

	// 阶段5: 收集修改意图
	StateAdjustCollectingIntent engine.State = "_schedule_adjust_collecting_intent_"

	// 阶段5.5: 分析意图
	StateAdjustAnalyzingIntent engine.State = "_schedule_adjust_analyzing_intent_"

	// 阶段6: AI分析意图并生成方案
	StateAdjustGeneratingPlan engine.State = "_schedule_adjust_generating_plan_"

	// 阶段7: 确认调整方案
	StateAdjustConfirmingPlan engine.State = "_schedule_adjust_confirming_plan_"

	// 阶段8: 执行调整
	StateAdjustExecuting engine.State = "_schedule_adjust_executing_"

	// 阶段9: 保存调整
	StateAdjustSaving engine.State = "_schedule_adjust_saving_"

	// ========== 重排班次相关状态 ==========

	// 收集班次人数阶段（CollectStaffCount 子工作流）
	StateAdjustCollectingStaffCount engine.State = "_schedule_adjust_collecting_staff_count_"

	// 核心排班阶段（Core 子工作流）
	StateAdjustCoreScheduling engine.State = "_schedule_adjust_core_scheduling_"

	// 确认人数（用于快速操作场景）
	StateAdjustConfirmingStaffCount engine.State = "_schedule_adjust_confirming_staff_count_"

	// 终态 - 复用通用常量
	StateAdjustCompleted = engine.StateCompleted // "_completed_"
	StateAdjustFailed    = engine.StateFailed    // "_failed_"
	StateAdjustCancelled = engine.StateCancelled // "_cancelled_"
)

// ============================================================
// schedule.adjust 工作流 - 事件定义
// ============================================================

const (
	// 初始化事件
	EventAdjustStart engine.Event = "_schedule_adjust_start_"

	// 来源解析事件
	EventAdjustSourceResolved    engine.Event = "_schedule_adjust_source_resolved_"     // 来源已解析（会话草案或历史排班）
	EventAdjustNeedDateRange     engine.Event = "_schedule_adjust_need_date_range_"     // 需要用户选择日期范围
	EventAdjustDateRangeSelected engine.Event = "_schedule_adjust_date_range_selected_" // 日期范围已选择

	// 数据加载事件
	EventAdjustDataLoaded engine.Event = "_schedule_adjust_data_loaded_" // 排班数据加载完成

	// 班次选择事件
	EventAdjustNeedShiftSelection engine.Event = "_schedule_adjust_need_shift_selection_" // 需要用户选择班次
	EventAdjustShiftSelected      engine.Event = "_schedule_adjust_shift_selected_"       // 班次已选择
	EventAdjustShiftChanged       engine.Event = "_schedule_adjust_shift_changed_"        // 切换班次

	// 意图收集事件
	EventAdjustIntentReceived  engine.Event = "_schedule_adjust_intent_received_"  // 收到用户修改意图
	EventAdjustIntentSubmitted engine.Event = "_schedule_adjust_intent_submitted_" // 意图表单提交
	EventAdjustIntentAnalyzed  engine.Event = "_schedule_adjust_intent_analyzed_"  // 意图分析完成
	EventAdjustQuickSwap       engine.Event = "_schedule_adjust_quick_swap_"       // 快速调班
	EventAdjustQuickReplace    engine.Event = "_schedule_adjust_quick_replace_"    // 快速替换
	EventAdjustQuickAdd        engine.Event = "_schedule_adjust_quick_add_"        // 快速添加人员
	EventAdjustQuickRemove     engine.Event = "_schedule_adjust_quick_remove_"     // 快速移除人员
	EventAdjustBackToIntent    engine.Event = "_schedule_adjust_back_to_intent_"   // 返回意图收集
	EventAdjustFinish          engine.Event = "_schedule_adjust_finish_"           // 完成调整

	// 方案生成事件
	EventAdjustPlanGenerated engine.Event = "_schedule_adjust_plan_generated_" // 方案生成完成
	EventAdjustPlanFailed    engine.Event = "_schedule_adjust_plan_failed_"    // 方案生成失败

	// 方案确认事件
	EventAdjustPlanConfirmed engine.Event = "_schedule_adjust_plan_confirmed_" // 确认方案
	EventAdjustPlanRejected  engine.Event = "_schedule_adjust_plan_rejected_"  // 拒绝方案
	EventAdjustPlanModified  engine.Event = "_schedule_adjust_plan_modified_"  // 修改方案要求

	// 执行事件
	EventAdjustExecuted      engine.Event = "_schedule_adjust_executed_"       // 执行完成
	EventAdjustExecuteFailed engine.Event = "_schedule_adjust_execute_failed_" // 执行失败

	// 历史操作事件
	EventAdjustUndo engine.Event = "_schedule_adjust_undo_" // 撤销
	EventAdjustRedo engine.Event = "_schedule_adjust_redo_" // 重做

	// 保存事件
	EventAdjustSaved          engine.Event = "_schedule_adjust_saved_"           // 保存完成
	EventAdjustSaveSuccess    engine.Event = "_schedule_adjust_save_success_"    // 保存成功
	EventAdjustSaveFailed     engine.Event = "_schedule_adjust_save_failed_"     // 保存失败
	EventAdjustContinueAdjust engine.Event = "_schedule_adjust_continue_adjust_" // 继续调整

	// 通用事件
	EventAdjustUserCancelled engine.Event = "_schedule_adjust_user_cancelled_" // 用户取消
	EventAdjustSystemError   engine.Event = "_schedule_adjust_system_error_"   // 系统错误

	// ========== 重排班次相关事件 ==========
	EventAdjustRegenerateStart     engine.Event = "_schedule_adjust_regenerate_start_"      // 开始重排
	EventAdjustStaffCountCollected engine.Event = "_schedule_adjust_staff_count_collected_" // CollectStaffCount子工作流完成
	EventAdjustCoreCompleted       engine.Event = "_schedule_adjust_core_completed_"        // Core子工作流完成
	EventAdjustSubFailed           engine.Event = "_schedule_adjust_sub_failed_"            // 子工作流失败
	EventAdjustRegenerateFailed    engine.Event = "_schedule_adjust_regenerate_failed_"     // 重排失败
	EventAdjustRegenerateAborted   engine.Event = "_schedule_adjust_regenerate_aborted_"    // 放弃重排

	// 快速操作相关（保留用于非重排场景）
	EventAdjustNeedStaffCount      engine.Event = "_schedule_adjust_need_staff_count_"      // 需要用户确认人数
	EventAdjustStaffCountConfirmed engine.Event = "_schedule_adjust_staff_count_confirmed_" // 人数已确认
)

// ============================================================
// schedule.adjust 工作流 - 状态描述（用于前端显示）
// ============================================================

var AdjustStateDescriptions = map[engine.State]string{
	StateAdjustInit:                 "初始化调整",
	StateAdjustLoadingData:          "加载排班数据",
	StateAdjustSelectingDateRange:   "选择日期范围",
	StateAdjustSelectingShift:       "选择班次",
	StateAdjustCollectingIntent:     "收集修改意图",
	StateAdjustAnalyzingIntent:      "分析调整意图",
	StateAdjustGeneratingPlan:       "生成调整方案",
	StateAdjustConfirmingPlan:       "确认调整方案",
	StateAdjustExecuting:            "执行调整",
	StateAdjustSaving:               "保存调整",
	StateAdjustCollectingStaffCount: "正在收集班次人数",
	StateAdjustCoreScheduling:       "正在生成排班",
	StateAdjustConfirmingStaffCount: "确认排班人数",
	StateAdjustCompleted:            "调整完成",
	StateAdjustFailed:               "调整失败",
	StateAdjustCancelled:            "已取消",
}

// GetAdjustStateDescription 获取调整工作流状态的中文描述
func GetAdjustStateDescription(state engine.State) string {
	if desc, ok := AdjustStateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}
