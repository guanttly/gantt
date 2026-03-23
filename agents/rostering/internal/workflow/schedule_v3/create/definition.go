// Package create 排班创建工作流 V3
//
// 渐进式排班架构：LLM评估所有需求，生成渐进式任务计划，分阶段执行
// 流程：信息收集 -> 个人需求 -> 需求评估 -> 渐进式任务执行 -> 确认保存
package create

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班创建工作流 V3 定义（父工作流）
// ============================================================

// init 函数在包被导入时自动注册工作流
func init() {
	engine.Register(GetScheduleCreateV3WorkflowDefinition())
}

// GetScheduleCreateV3WorkflowDefinition 获取排班创建工作流 V3 定义
func GetScheduleCreateV3WorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowScheduleCreateV3,
		InitialState: CreateV3StateInit,
		Transitions:  buildCreateV3Transitions(),
	}
}

// buildCreateV3Transitions 构建所有状态转换
func buildCreateV3Transitions() []engine.Transition {
	return []engine.Transition{
		// ============================================================
		// 阶段 0: 初始化
		// ============================================================
		{
			From:       CreateV3StateInit,
			Event:      CreateV3EventStart,
			To:         CreateV3StateInfoCollecting,
			StateLabel: "收集信息",
			Act:        actStartInfoCollect,
		},
		{
			From:  CreateV3StateInit,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1: 信息收集
		// ============================================================
		{
			From:  CreateV3StateInfoCollecting,
			Event: CreateV3EventInfoCollected,
			To:    CreateV3StateConfirmPeriod,
			Act:   actOnInfoCollected,
		},
		{
			From:  CreateV3StateInfoCollecting,
			Event: CreateV3EventSubCancelled,
			To:    CreateV3StateCancelled,
			Act:   actOnSubCancelled,
		},
		{
			From:  CreateV3StateInfoCollecting,
			Event: CreateV3EventSubFailed,
			To:    CreateV3StateFailed,
			Act:   actOnSubFailed,
		},

		// ============================================================
		// 阶段 1.5: 确认排班时间
		// ============================================================
		{
			From:  CreateV3StateConfirmPeriod,
			Event: CreateV3EventPeriodConfirmed,
			To:    CreateV3StateConfirmShifts,
			Act:   actOnPeriodConfirmed,
		},
		{
			From:       CreateV3StateConfirmPeriod,
			Event:      CreateV3EventUserModify,
			To:         CreateV3StateConfirmPeriod,
			Act:        actModifyPeriod,
			StateLabel: "正在收集班次人数要求",
		},
		{
			From:  CreateV3StateConfirmPeriod,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1.6: 确认班次选择
		// ============================================================
		{
			From:  CreateV3StateConfirmShifts,
			Event: CreateV3EventShiftsConfirmed,
			To:    CreateV3StateConfirmStaffCount,
			Act:   actOnShiftsConfirmed,
		},
		{
			From:  CreateV3StateConfirmShifts,
			Event: CreateV3EventUserModify,
			To:    CreateV3StateConfirmShifts,
			Act:   actModifyShifts,
		},
		{
			From:  CreateV3StateConfirmShifts,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1.7: 确认人数配置
		// ============================================================
		{
			From:  CreateV3StateConfirmStaffCount,
			Event: CreateV3EventStaffCountConfirmed,
			To:    CreateV3StatePersonalNeeds,
			Act:   actOnStaffCountConfirmed,
		},
		{
			From:  CreateV3StateConfirmStaffCount,
			Event: CreateV3EventModifyStaffCount,
			To:    CreateV3StateConfirmStaffCount,
			Act:   actModifyStaffCount,
		},
		{
			From:  CreateV3StateConfirmStaffCount,
			Event: CreateV3EventUserModify,
			To:    CreateV3StateConfirmShifts,
			Act:   actModifyShifts,
		},
		{
			From:  CreateV3StateConfirmStaffCount,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 2: 个人需求收集与确认
		// ============================================================
		{
			From:  CreateV3StatePersonalNeeds,
			Event: CreateV3EventPersonalNeedsConfirmed,
			To:    CreateV3StateRequirementAssessment,
			Act:   actOnPersonalNeedsConfirmed,
		},
		{
			From:  CreateV3StatePersonalNeeds,
			Event: CreateV3EventTemporaryNeedsTextSubmitted,
			To:    CreateV3StatePersonalNeeds, // 保持在个人需求状态，等待用户确认需求预览
			Act:   actOnTemporaryNeedsTextSubmitted,
		},
		{
			From:  CreateV3StatePersonalNeeds,
			Event: CreateV3EventUserModify,
			To:    CreateV3StatePersonalNeeds,
			Act:   actModifyPersonalNeeds,
		},
		{
			From:  CreateV3StatePersonalNeeds,
			Event: CreateV3EventSkipPhase,
			To:    CreateV3StatePersonalNeeds,
			Act:   actReturnToPersonalNeedsConfirm,
		},
		{
			From:  CreateV3StatePersonalNeeds,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 3: 需求评估（V3核心）
		// ============================================================
		{
			From:  CreateV3StateRequirementAssessment,
			Event: CreateV3EventRequirementAssessed,
			To:    CreateV3StatePlanReview,
			Act:   actOnRequirementAssessed,
		},
		{
			From:  CreateV3StateRequirementAssessment,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 3.5: 任务计划预览与确认
		// ============================================================
		{
			From:       CreateV3StatePlanReview,
			Event:      CreateV3EventPlanConfirmed,
			To:         CreateV3StateProgressiveTask,
			StateLabel: "正在执行渐进式任务...",
			Act:        actOnPlanConfirmed,
			AfterAct:   actAfterPlanConfirmed,
		},
		{
			From:  CreateV3StatePlanReview,
			Event: CreateV3EventPlanAdjust,
			To:    CreateV3StateWaitingAdjustment,
			Act:   actOnPlanAdjust,
		},
		{
			From:  CreateV3StatePlanReview,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 4: 渐进式任务执行循环
		// ============================================================
		{
			From:     CreateV3StateProgressiveTask,
			Event:    CreateV3EventTaskCompleted,
			To:       CreateV3StateProgressiveTask, // 循环在同一状态（连续排班）或进入审核状态（非连续排班）
			Act:      actOnTaskCompleted,
			AfterAct: actAfterTaskCompleted,
		},
		// 非连续排班模式：任务完成后进入审核状态
		{
			From:  CreateV3StateProgressiveTask,
			Event: CreateV3EventEnterTaskReview,
			To:    CreateV3StateTaskReview,
			Act:   actEnterTaskReviewState,
		},
		{
			From:  CreateV3StateProgressiveTask,
			Event: CreateV3EventAllTasksComplete,
			To:    CreateV3StateGlobalReview,
			Act:   actOnAllTasksComplete,
		},
		{
			From:     CreateV3StateProgressiveTask,
			Event:    CreateV3EventSubFailed,
			To:       CreateV3StateProgressiveTask,
			Act:      actOnTaskFailed,
			AfterAct: actSpawnNextTaskOrComplete,
		},
		{
			From:       CreateV3StateProgressiveTask,
			Event:      CreateV3EventTaskFailed,
			To:         CreateV3StateTaskFailed,
			StateLabel: "任务执行失败，等待用户决定...",
			Act:        actOnEnterTaskFailedState,
		},
		{
			From:  CreateV3StateProgressiveTask,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 任务审核状态（非连续排班模式）
		// ============================================================
		{
			From:     CreateV3StateTaskReview,
			Event:    CreateV3EventTaskReviewConfirmed,
			To:       CreateV3StateProgressiveTask, // 临时状态，AfterAct 会继续下一个任务
			Act:      actOnTaskReviewConfirmed,
			AfterAct: actSpawnNextTaskOrComplete,
		},
		{
			From:  CreateV3StateTaskReview,
			Event: CreateV3EventTaskReviewAdjust,
			To:    CreateV3StateWaitingAdjustment,
			Act:   actOnTaskReviewAdjust,
		},
		{
			From:  CreateV3StateTaskReview,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 等待用户输入调整需求状态
		// ============================================================
		{
			From:  CreateV3StateWaitingAdjustment,
			Event: CreateV3EventUserAdjustmentMessage,
			To:    CreateV3StateWaitingAdjustment, // 保持在等待状态，子工作流完成后会触发 TaskAdjusted 事件
			Act:   actOnUserAdjustmentMessage,
		},
		{
			From:  CreateV3StateWaitingAdjustment,
			Event: CreateV3EventPlanAdjusted,
			To:    CreateV3StatePlanReview,
			Act:   actOnPlanAdjusted,
		},
		{
			From:  CreateV3StateWaitingAdjustment,
			Event: CreateV3EventTaskAdjusted,
			To:    CreateV3StateTaskReview, // 调整完成后进入审核状态，让用户确认结果
			Act:   actOnTaskAdjusted,
		},
		{
			From:  CreateV3StateWaitingAdjustment,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 任务执行失败处理
		// ============================================================
		{
			From:     CreateV3StateTaskFailed,
			Event:    CreateV3EventTaskFailedContinue,
			To:       CreateV3StateProgressiveTask, // 临时状态，AfterAct 会继续下一个任务
			Act:      actOnTaskFailedContinue,
			AfterAct: actSpawnNextTaskOrComplete,
		},
		{
			From:  CreateV3StateTaskFailed,
			Event: CreateV3EventTaskFailedCancel,
			To:    CreateV3StateCancelled,
			Act:   actOnTaskFailedCancel,
		},
		{
			From:  CreateV3StateTaskFailed,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 4.5: 全局评审（规则和个人需求逐条评审、对论迭代）
		// ============================================================
		{
			From:       CreateV3StateGlobalReview,
			Event:      CreateV3EventStartGlobalReview,
			To:         CreateV3StateGlobalReview,
			StateLabel: "正在执行全局规则评审...",
			Act:        actStartGlobalReview,
			AfterAct:   actAfterGlobalReview,
		},
		{
			From:  CreateV3StateGlobalReview,
			Event: CreateV3EventGlobalReviewCompleted,
			To:    CreateV3StateConfirmSaving,
			Act:   actOnGlobalReviewCompleted,
		},
		{
			From:       CreateV3StateGlobalReview,
			Event:      CreateV3EventGlobalReviewNeedsManual,
			To:         CreateV3StateGlobalReviewManual,
			StateLabel: "全局评审需人工处理",
			Act:        actOnGlobalReviewNeedsManual,
		},
		{
			From:  CreateV3StateGlobalReview,
			Event: CreateV3EventGlobalReviewSkip,
			To:    CreateV3StateConfirmSaving,
			Act:   actOnGlobalReviewSkip,
		},
		{
			From:  CreateV3StateGlobalReview,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 全局评审人工处理状态
		// ============================================================
		{
			From:  CreateV3StateGlobalReviewManual,
			Event: CreateV3EventGlobalReviewManualConfirmed,
			To:    CreateV3StateConfirmSaving,
			Act:   actOnGlobalReviewManualConfirmed,
		},
		{
			From:  CreateV3StateGlobalReviewManual,
			Event: CreateV3EventUserModify,
			To:    CreateV3StateGlobalReviewManual,
			Act:   actModifyGlobalReviewManual,
		},
		{
			From:  CreateV3StateGlobalReviewManual,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 5: 确认保存
		// ============================================================
		{
			From:       CreateV3StateConfirmSaving,
			Event:      CreateV3EventSaveCompleted,
			To:         CreateV3StateCompleted,
			StateLabel: "排班已保存 ✅",
			Act:        actOnSaveCompleted,
		},
		{
			From:  CreateV3StateConfirmSaving,
			Event: CreateV3EventUserModify,
			To:    CreateV3StateConfirmSaving,
			Act:   actModifyBeforeSave,
		},
		{
			From:  CreateV3StateConfirmSaving,
			Event: CreateV3EventUserCancel,
			To:    CreateV3StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 全局错误处理
		// ============================================================
		{
			From:  engine.State("*"),
			Event: CreateV3EventError,
			To:    CreateV3StateFailed,
			Act:   actHandleError,
		},
	}
}
