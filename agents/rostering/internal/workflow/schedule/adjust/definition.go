package adjust

// ============================================================
// 排班调整工作流 - Actions
// 工作流程：启动 → 解析来源 → 加载数据 → 选择班次 → 收集意图 →
//          生成方案 → 确认方案 → 执行调整 → 保存/继续
//
// 支持功能：
// - 调整当前会话的排班草案
// - 调整指定日期范围的历史排班
// - 快速调班、替换、添加、移除
// - 撤销/重做操作
// ============================================================

// import (
// 	"jusha/mcp/pkg/workflow/engine"

// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // init 函数在包被导入时自动注册工作流
// func init() {
// 	// 注册排班调整工作流
// 	engine.Register(GetScheduleAdjustWorkflowDefinition())
// }

// // GetScheduleAdjustWorkflowDefinition 获取排班调整工作流定义
// func GetScheduleAdjustWorkflowDefinition() *engine.WorkflowDefinition {
// 	return &engine.WorkflowDefinition{
// 		Name:         WorkflowScheduleAdjust,
// 		InitialState: StateAdjustInit,
// 		Transitions:  buildAdjustTransitions(),
// 	}
// }

// // buildAdjustTransitions 构建所有状态转换
// func buildAdjustTransitions() []engine.Transition {
// 	return []engine.Transition{
// 		// ========== 阶段1: 工作流启动与初始化 ==========
// 		// 使用通用 _start_ 事件（与意图映射系统统一）
// 		{
// 			From:       StateAdjustInit,
// 			Event:      engine.EventStart,
// 			To:         StateAdjustInit,
// 			StateLabel: "正在分析调整请求",
// 			Act:        actScheduleAdjustStart,
// 			AfterAct:   actScheduleAdjustAfterStart,
// 		},

// 		// 来源已解析 -> 加载数据
// 		{
// 			From:       StateAdjustInit,
// 			Event:      EventAdjustSourceResolved,
// 			To:         StateAdjustLoadingData,
// 			StateLabel: "正在加载排班数据",
// 			Act:        actScheduleAdjustLoadData,
// 			AfterAct:   actScheduleAdjustAfterLoadData,
// 		},

// 		// 需要用户选择日期范围
// 		{
// 			From:  StateAdjustInit,
// 			Event: EventAdjustNeedDateRange,
// 			To:    StateAdjustSelectingDateRange,
// 			Act:   actScheduleAdjustPromptDateRange,
// 		},

// 		// ========== 阶段2: 选择日期范围 ==========
// 		{
// 			From:       StateAdjustSelectingDateRange,
// 			Event:      EventAdjustDateRangeSelected,
// 			To:         StateAdjustLoadingData,
// 			StateLabel: "正在加载排班数据",
// 			Act:        actScheduleAdjustOnDateRangeSelected,
// 			AfterAct:   actScheduleAdjustAfterDateRangeSelected,
// 		},
// 		{
// 			From:  StateAdjustSelectingDateRange,
// 			Event: EventAdjustUserCancelled,
// 			To:    StateAdjustCancelled,
// 			Act:   actScheduleAdjustHandleCancel,
// 		},

// 		// ========== 阶段3: 加载数据完成 -> 选择班次 ==========
// 		{
// 			From:     StateAdjustLoadingData,
// 			Event:    EventAdjustDataLoaded,
// 			To:       StateAdjustSelectingShift,
// 			Act:      actScheduleAdjustSelectShift,
// 			AfterAct: actScheduleAdjustAfterSelectShift,
// 		},

// 		// ========== 阶段4: 选择班次 ==========
// 		// 注意：不设置 StateLabel，由 Act 中的消息带按钮展示
// 		{
// 			From:  StateAdjustSelectingShift,
// 			Event: EventAdjustNeedShiftSelection,
// 			To:    StateAdjustSelectingShift,
// 			Act:   actScheduleAdjustPromptShiftSelection,
// 		},
// 		{
// 			From:       StateAdjustSelectingShift,
// 			Event:      EventAdjustShiftSelected,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "请描述您的调整需求",
// 			Act:        actScheduleAdjustOnShiftSelected,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},
// 		{
// 			From:  StateAdjustSelectingShift,
// 			Event: EventAdjustUserCancelled,
// 			To:    StateAdjustCancelled,
// 			Act:   actScheduleAdjustHandleCancel,
// 		},

// 		// ========== 阶段5: 收集修改意图 ==========
// 		// 进入收集意图阶段，展示操作选项
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustIntentSubmitted,
// 			To:         StateAdjustAnalyzingIntent,
// 			StateLabel: "正在分析您的需求",
// 			Act:        actScheduleAdjustAnalyzeIntent,
// 			AfterAct:   actScheduleAdjustAfterAnalyzeIntent,
// 		},

// 		// 用户输入自由文本意图
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustIntentReceived,
// 			To:         StateAdjustAnalyzingIntent,
// 			StateLabel: "正在分析您的需求",
// 			Act:        actScheduleAdjustAnalyzeIntent,
// 			AfterAct:   actScheduleAdjustAfterAnalyzeIntent,
// 		},

// 		// 意图分析完成 -> 生成方案
// 		{
// 			From:       StateAdjustAnalyzingIntent,
// 			Event:      EventAdjustIntentAnalyzed,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成调整方案",
// 			Act:        actScheduleAdjustGeneratePlan,
// 			AfterAct:   actScheduleAdjustAfterGeneratePlan,
// 		},

// 		// 快速操作 - 直接进入生成方案
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustQuickSwap,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成调班方案",
// 			Act:        actScheduleAdjustQuickSwap,
// 			AfterAct:   actScheduleAdjustAfterQuickSwap,
// 		},
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustQuickReplace,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成替换方案",
// 			Act:        actScheduleAdjustQuickReplace,
// 			AfterAct:   actScheduleAdjustAfterQuickReplace,
// 		},
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustQuickAdd,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成添加方案",
// 			Act:        actScheduleAdjustQuickAdd,
// 			AfterAct:   actScheduleAdjustAfterQuickAdd,
// 		},
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustQuickRemove,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成移除方案",
// 			Act:        actScheduleAdjustQuickRemove,
// 			AfterAct:   actScheduleAdjustAfterQuickRemove,
// 		},

// 		// ========== 重排班次流程（使用 CollectStaffCount 和 Core 子工作流）==========
// 		// 从意图收集启动重排 -> CollectStaffCount 子工作流
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustRegenerateStart,
// 			To:         StateAdjustCollectingStaffCount,
// 			StateLabel: "正在收集班次人数",
// 			Act:        actScheduleAdjustPrepareRegenerate,
// 			AfterAct:   actScheduleAdjustSpawnCollectStaffCount,
// 		},

// 		// CollectStaffCount 完成 -> Core 子工作流
// 		{
// 			From:       StateAdjustCollectingStaffCount,
// 			Event:      EventAdjustStaffCountCollected,
// 			To:         StateAdjustCoreScheduling,
// 			StateLabel: "正在生成排班",
// 			Act:        actScheduleAdjustOnStaffCountCollected,
// 			AfterAct:   actScheduleAdjustSpawnCoreWorkflow,
// 		},

// 		// Core 完成 -> 确认方案
// 		{
// 			From:       StateAdjustCoreScheduling,
// 			Event:      EventAdjustCoreCompleted,
// 			To:         StateAdjustConfirmingPlan,
// 			StateLabel: "请确认重排结果",
// 			Act:        actScheduleAdjustOnCoreCompleted,
// 			AfterAct:   actScheduleAdjustShowRegenerateConfirm,
// 		},

// 		// 子工作流失败 -> 返回意图收集
// 		{
// 			From:       StateAdjustCollectingStaffCount,
// 			Event:      EventAdjustSubFailed,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "人数收集失败",
// 			Act:        actScheduleAdjustOnSubFailed,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},
// 		{
// 			From:       StateAdjustCoreScheduling,
// 			Event:      EventAdjustSubFailed,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "重排失败",
// 			Act:        actScheduleAdjustOnSubFailed,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},

// 		// 取消重排 -> 返回意图收集
// 		{
// 			From:       StateAdjustCollectingStaffCount,
// 			Event:      EventAdjustRegenerateAborted,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "已取消重排",
// 			Act:        actScheduleAdjustOnRegenerateAborted,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},
// 		{
// 			From:       StateAdjustCoreScheduling,
// 			Event:      EventAdjustRegenerateAborted,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "已取消重排",
// 			Act:        actScheduleAdjustOnRegenerateAborted,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},

// 		// 切换班次
// 		{
// 			From:     StateAdjustCollectingIntent,
// 			Event:    EventAdjustShiftChanged,
// 			To:       StateAdjustSelectingShift,
// 			Act:      actScheduleAdjustSelectShift,
// 			AfterAct: actScheduleAdjustAfterSelectShift,
// 		},

// 		// 撤销操作
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustUndo,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "已撤销上一步操作",
// 			Act:        actScheduleAdjustUndo,
// 		},

// 		// 重做操作
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustRedo,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "已重做操作",
// 			Act:        actScheduleAdjustRedo,
// 		},

// 		// 返回意图收集
// 		{
// 			From:       StateAdjustAnalyzingIntent,
// 			Event:      EventAdjustBackToIntent,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "请重新描述您的调整需求",
// 			Act:        actScheduleAdjustCollectIntent,
// 		},

// 		// 完成调整
// 		{
// 			From:       StateAdjustCollectingIntent,
// 			Event:      EventAdjustFinish,
// 			To:         StateAdjustSaving,
// 			StateLabel: "正在保存调整结果",
// 			Act:        actScheduleAdjustSave,
// 			AfterAct:   actScheduleAdjustAfterSave,
// 		},

// 		{
// 			From:  StateAdjustCollectingIntent,
// 			Event: EventAdjustUserCancelled,
// 			To:    StateAdjustCancelled,
// 			Act:   actScheduleAdjustHandleCancel,
// 		},

// 		// ========== 阶段6: 生成调整方案 ==========
// 		{
// 			From:       StateAdjustGeneratingPlan,
// 			Event:      EventAdjustIntentAnalyzed,
// 			To:         StateAdjustGeneratingPlan,
// 			StateLabel: "正在生成调整方案",
// 			Act:        actScheduleAdjustGeneratePlan,
// 			AfterAct:   actScheduleAdjustAfterGeneratePlan,
// 		},
// 		{
// 			From:       StateAdjustGeneratingPlan,
// 			Event:      EventAdjustPlanGenerated,
// 			To:         StateAdjustConfirmingPlan,
// 			StateLabel: "请确认调整方案",
// 			Act:        actScheduleAdjustPreviewPlan,
// 		},
// 		{
// 			From:       StateAdjustGeneratingPlan,
// 			Event:      EventAdjustBackToIntent,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "请重新描述您的调整需求",
// 			Act:        actScheduleAdjustCollectIntent,
// 		},

// 		// ========== 阶段7: 确认调整方案 ==========
// 		{
// 			From:       StateAdjustConfirmingPlan,
// 			Event:      EventAdjustPlanConfirmed,
// 			To:         StateAdjustExecuting,
// 			StateLabel: "正在执行调整",
// 			Act:        actScheduleAdjustExecute,
// 			AfterAct:   actScheduleAdjustAfterExecute,
// 		},
// 		{
// 			From:       StateAdjustConfirmingPlan,
// 			Event:      EventAdjustPlanRejected,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "请重新描述您的调整需求",
// 			Act:        actScheduleAdjustCollectIntent,
// 		},
// 		{
// 			From:  StateAdjustConfirmingPlan,
// 			Event: EventAdjustUserCancelled,
// 			To:    StateAdjustCancelled,
// 			Act:   actScheduleAdjustHandleCancel,
// 		},

// 		// 在确认重排结果时放弃
// 		{
// 			From:       StateAdjustConfirmingPlan,
// 			Event:      EventAdjustRegenerateAborted,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "已放弃重排",
// 			Act:        actScheduleAdjustOnRegenerateAborted,
// 			AfterAct:   actScheduleAdjustAfterShiftSelected,
// 		},

// 		// 在确认重排结果时重新排班
// 		{
// 			From:       StateAdjustConfirmingPlan,
// 			Event:      EventAdjustRegenerateStart,
// 			To:         StateAdjustCollectingStaffCount,
// 			StateLabel: "正在收集班次人数",
// 			Act:        actScheduleAdjustPrepareRegenerate,
// 			AfterAct:   actScheduleAdjustSpawnCollectStaffCount,
// 		},

// 		// ========== 阶段8: 执行调整 ==========
// 		{
// 			From:       StateAdjustExecuting,
// 			Event:      EventAdjustExecuted,
// 			To:         StateAdjustCollectingIntent,
// 			StateLabel: "调整完成，您可以继续调整或完成",
// 			Act:        actScheduleAdjustCollectIntent,
// 		},

// 		// ========== 阶段9: 保存 ==========
// 		{
// 			From:       StateAdjustSaving,
// 			Event:      EventAdjustSaved,
// 			To:         StateAdjustCompleted,
// 			StateLabel: "调整已保存 ✅",
// 			Act:        actScheduleAdjustComplete,
// 		},

// 		// ========== 全局错误处理 ==========
// 		{
// 			From:  engine.State("*"),
// 			Event: EventAdjustSystemError,
// 			To:    StateAdjustFailed,
// 			Act:   actScheduleAdjustHandleError,
// 		},
// 	}
// }
