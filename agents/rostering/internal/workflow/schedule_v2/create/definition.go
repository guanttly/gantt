// Package create 排班创建工作流 V2
//
// 新架构：按优先级顺序处理不同类型的班次
// 流程：信息收集 -> 个人需求 -> 固定班次 -> 特殊班次 -> 普通班次 -> 科研班次 -> 填充班次 -> 确认保存
package create

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班创建工作流 V2 定义（父工作流）
// ============================================================

// init 函数在包被导入时自动注册工作流
func init() {
	engine.Register(GetScheduleCreateV2WorkflowDefinition())
}

// GetScheduleCreateV2WorkflowDefinition 获取排班创建工作流 V2 定义
func GetScheduleCreateV2WorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowScheduleCreateV2,
		InitialState: CreateV2StateInit,
		Transitions:  buildCreateV2Transitions(),
	}
}

// buildCreateV2Transitions 构建所有状态转换
func buildCreateV2Transitions() []engine.Transition {
	return []engine.Transition{
		// ============================================================
		// 阶段 0: 初始化
		// ============================================================
		{
			From:       CreateV2StateInit,
			Event:      CreateV2EventStart,
			To:         CreateV2StateInfoCollecting,
			StateLabel: "收集信息",
			Act:        actStartInfoCollect,
		},
		{
			From:  CreateV2StateInit,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1: 信息收集
		// ============================================================
		{
			From:  CreateV2StateInfoCollecting,
			Event: CreateV2EventInfoCollected,
			To:    CreateV2StateConfirmPeriod,
			Act:   actOnInfoCollected,
		},
		{
			From:  CreateV2StateInfoCollecting,
			Event: CreateV2EventSubCancelled,
			To:    CreateV2StateCancelled,
			Act:   actOnSubCancelled,
		},
		{
			From:  CreateV2StateInfoCollecting,
			Event: CreateV2EventSubFailed,
			To:    CreateV2StateFailed,
			Act:   actOnSubFailed,
		},

		// ============================================================
		// 阶段 1.5: 确认排班时间
		// ============================================================
		{
			From:  CreateV2StateConfirmPeriod,
			Event: CreateV2EventPeriodConfirmed,
			To:    CreateV2StateConfirmShifts,
			Act:   actOnPeriodConfirmed,
		},
		{
			From:       CreateV2StateConfirmPeriod,
			Event:      CreateV2EventUserModify,
			To:         CreateV2StateConfirmPeriod,
			Act:        actModifyPeriod,
			StateLabel: "正在收集班次人数要求",
		},
		{
			From:  CreateV2StateConfirmPeriod,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1.6: 确认班次选择
		// ============================================================
		{
			From:  CreateV2StateConfirmShifts,
			Event: CreateV2EventShiftsConfirmed,
			To:    CreateV2StateConfirmStaffCount,
			Act:   actOnShiftsConfirmed,
		},
		{
			From:  CreateV2StateConfirmShifts,
			Event: CreateV2EventUserModify,
			To:    CreateV2StateConfirmShifts,
			Act:   actModifyShifts,
		},
		{
			From:  CreateV2StateConfirmShifts,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1.7: 确认人数配置
		// ============================================================
		{
			From:  CreateV2StateConfirmStaffCount,
			Event: CreateV2EventStaffCountConfirmed,
			To:    CreateV2StatePersonalNeeds,
			Act:   actOnStaffCountConfirmed,
		},
		{
			From:  CreateV2StateConfirmStaffCount,
			Event: CreateV2EventModifyStaffCount,
			To:    CreateV2StateConfirmStaffCount,
			Act:   actModifyStaffCount,
		},
		{
			From:  CreateV2StateConfirmStaffCount,
			Event: CreateV2EventUserModify,
			To:    CreateV2StateConfirmShifts,
			Act:   actModifyShifts,
		},
		{
			From:  CreateV2StateConfirmStaffCount,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 2: 个人需求收集与确认
		// ============================================================
		{
			From:  CreateV2StatePersonalNeeds,
			Event: CreateV2EventPersonalNeedsConfirmed,
			To:    CreateV2StateFixedShift,
			Act:   actOnPersonalNeedsConfirmed,
		},
		{
			From:  CreateV2StatePersonalNeeds,
			Event: CreateV2EventUserModify,
			To:    CreateV2StatePersonalNeeds,
			Act:   actModifyPersonalNeeds,
		},
		{
			From:  CreateV2StatePersonalNeeds,
			Event: CreateV2EventSkipPhase, // 从修改需求界面返回到需求确认界面
			To:    CreateV2StatePersonalNeeds,
			Act:   actReturnToPersonalNeedsConfirm,
		},
		{
			From:  CreateV2StatePersonalNeeds,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 3: 固定班次处理
		// ============================================================
		{
			From:       CreateV2StateFixedShift,
			Event:      CreateV2EventFixedShiftConfirmed,
			To:         CreateV2StateSpecialShift,
			StateLabel: "正在排特殊班次...",
			Act:        actOnFixedShiftConfirmed,
		},
		{
			From:  CreateV2StateFixedShift,
			Event: CreateV2EventUserModify,
			To:    CreateV2StatePersonalNeeds, // 返回到个人需求阶段，允许用户修改需求
			Act:   actModifyPersonalNeeds,
		},
		{
			From:  CreateV2StateFixedShift,
			Event: CreateV2EventPersonalNeedsConfirmed, // 支持从固定班次阶段返回修改需求后再确认
			To:    CreateV2StateFixedShift,             // 保持在固定班次阶段
			Act:   actOnPersonalNeedsConfirmedFromFixedShift,
		},
		{
			From:  CreateV2StateFixedShift,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},
		{
			From:  CreateV2StateFixedShift,
			Event: CreateV2EventSkipPhase,
			To:    CreateV2StateSpecialShift,
			Act:   actSkipFixedShift,
		},

		// ============================================================
		// 阶段 4: 特殊班次排班（循环调用 Core 子工作流）
		// ============================================================
		{
			From:     CreateV2StateSpecialShift,
			Event:    CreateV2EventShiftCompleted,
			To:       CreateV2StateSpecialShift, // 循环在同一状态（连续排班）或进入审核状态（非连续排班）
			Act:      actOnShiftCompleted,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		// 非连续排班模式：班次完成后进入审核状态
		{
			From:       CreateV2StateSpecialShift,
			Event:      CreateV2EventEnterShiftReview,
			To:         CreateV2StateShiftReview,
			StateLabel: "班次审核中...",
			Act:        actEnterShiftReviewState,
		},
		{
			From:  CreateV2StateSpecialShift,
			Event: CreateV2EventShiftPhaseComplete,
			To:    CreateV2StateNormalShift,
			Act:   actOnPhaseComplete,
		},
		{
			From:     CreateV2StateSpecialShift,
			Event:    CreateV2EventSubFailed,
			To:       CreateV2StateSpecialShift,
			Act:      actOnShiftFailed,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		{
			From:       CreateV2StateSpecialShift,
			Event:      CreateV2EventShiftFailed,
			To:         CreateV2StateShiftFailed,
			StateLabel: "班次排班失败，等待用户决定...",
			Act:        actOnEnterShiftFailedState, // 确保按钮正确显示
		},
		{
			From:       CreateV2StateSpecialShift,
			Event:      CreateV2EventSkipPhase,
			To:         CreateV2StateNormalShift,
			StateLabel: "跳过特殊班次，正在排普通班次...",
			Act:        actSkipPhase,
		},
		{
			From:  CreateV2StateSpecialShift,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 5: 普通班次排班（循环调用 Core 子工作流）
		// ============================================================
		{
			From:     CreateV2StateNormalShift,
			Event:    CreateV2EventShiftCompleted,
			To:       CreateV2StateNormalShift, // 循环在同一状态（连续排班）或进入审核状态（非连续排班）
			Act:      actOnShiftCompleted,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		// 非连续排班模式：班次完成后进入审核状态
		{
			From:  CreateV2StateNormalShift,
			Event: CreateV2EventEnterShiftReview,
			To:    CreateV2StateShiftReview,
			Act:   actEnterShiftReviewState,
		},
		{
			From:  CreateV2StateNormalShift,
			Event: CreateV2EventShiftPhaseComplete,
			To:    CreateV2StateResearchShift,
			Act:   actOnPhaseComplete,
		},
		{
			From:     CreateV2StateNormalShift,
			Event:    CreateV2EventSubFailed,
			To:       CreateV2StateNormalShift,
			Act:      actOnShiftFailed,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		{
			From:  CreateV2StateNormalShift,
			Event: CreateV2EventShiftFailed,
			To:    CreateV2StateShiftFailed,
			Act:   actOnEnterShiftFailedState, // 确保按钮正确显示
		},
		{
			From:  CreateV2StateNormalShift,
			Event: CreateV2EventSkipPhase,
			To:    CreateV2StateResearchShift,
			Act:   actSkipPhase,
		},
		{
			From:  CreateV2StateNormalShift,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 6: 科研班次排班（循环调用 Core 子工作流）
		// ============================================================
		{
			From:     CreateV2StateResearchShift,
			Event:    CreateV2EventShiftCompleted,
			To:       CreateV2StateResearchShift, // 循环在同一状态（连续排班）或进入审核状态（非连续排班）
			Act:      actOnShiftCompleted,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		// 非连续排班模式：班次完成后进入审核状态
		{
			From:  CreateV2StateResearchShift,
			Event: CreateV2EventEnterShiftReview,
			To:    CreateV2StateShiftReview,
			Act:   actEnterShiftReviewState,
		},
		{
			From:  CreateV2StateResearchShift,
			Event: CreateV2EventShiftPhaseComplete,
			To:    CreateV2StateFillShift,
			Act:   actOnPhaseComplete,
		},
		{
			From:     CreateV2StateResearchShift,
			Event:    CreateV2EventSubFailed,
			To:       CreateV2StateResearchShift,
			Act:      actOnShiftFailed,
			AfterAct: actSpawnNextShiftOrComplete,
		},
		{
			From:  CreateV2StateResearchShift,
			Event: CreateV2EventShiftFailed,
			To:    CreateV2StateShiftFailed,
			Act:   actOnEnterShiftFailedState, // 确保按钮正确显示
		},
		{
			From:  CreateV2StateResearchShift,
			Event: CreateV2EventSkipPhase,
			To:    CreateV2StateFillShift,
			Act:   actSkipPhase,
		},
		{
			From:  CreateV2StateResearchShift,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 班次审核状态（非连续排班模式）
		// ============================================================
		{
			From:     CreateV2StateShiftReview,
			Event:    CreateV2EventShiftReviewConfirmed,
			To:       CreateV2StateSpecialShift, // 临时状态，AfterAct 会根据当前阶段动态转换到正确状态
			Act:      actOnShiftReviewConfirmed,
			AfterAct: actOnShiftReviewConfirmedAfterAct, // 在 AfterAct 中根据当前阶段动态转换状态
		},
		{
			From:  CreateV2StateShiftReview,
			Event: CreateV2EventShiftReviewAdjust,
			To:    CreateV2StateWaitingAdjustment,
			Act:   actOnShiftReviewAdjust,
		},
		{
			From:  CreateV2StateShiftReview,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 等待用户输入调整需求状态
		// ============================================================
		{
			From:  CreateV2StateWaitingAdjustment,
			Event: CreateV2EventUserAdjustmentMessage,
			To:    CreateV2StateWaitingAdjustment, // 保持在等待状态，子工作流完成后会触发 ShiftAdjusted 事件
			Act:   actOnUserAdjustmentMessage,
		},
		{
			From:  CreateV2StateWaitingAdjustment,
			Event: CreateV2EventShiftAdjusted,
			To:    CreateV2StateShiftReview, // 调整完成后进入审核状态，让用户确认结果
			Act:   actOnShiftAdjusted,
		},
		{
			From:  CreateV2StateWaitingAdjustment,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 班次排班失败处理
		// ============================================================
		{
			From:     CreateV2StateShiftFailed,
			Event:    CreateV2EventShiftFailedContinue,
			To:       CreateV2StateSpecialShift, // 临时状态，AfterAct 会根据当前阶段动态转换
			Act:      actOnShiftFailedContinue,
			AfterAct: actOnShiftReviewConfirmedAfterAct, // 复用相同的AfterAct来动态转换状态
		},
		{
			From:  CreateV2StateShiftFailed,
			Event: CreateV2EventShiftFailedCancel,
			To:    CreateV2StateCancelled,
			Act:   actOnShiftFailedCancel,
		},
		{
			From:  CreateV2StateShiftFailed,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 7: 填充班次处理
		// ============================================================
		{
			From:       CreateV2StateFillShift,
			Event:      CreateV2EventShiftPhaseComplete,
			To:         CreateV2StateConfirmSaving,
			StateLabel: "正在准备保存",
			Act:        actOnFillShiftComplete,
			AfterAct:   actSpawnConfirmSaveWorkflow, // 启动 ConfirmSave 子工作流
		},
		{
			From:  CreateV2StateFillShift,
			Event: CreateV2EventUserModify,
			To:    CreateV2StateFillShift,
			Act:   actModifyFillShifts,
		},
		{
			From:       CreateV2StateFillShift,
			Event:      CreateV2EventSkipPhase,
			To:         CreateV2StateConfirmSaving,
			StateLabel: "跳过填充班次，正在准备保存...",
			Act:        actSkipFillShift,
		},
		{
			From:  CreateV2StateFillShift,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 8: 确认保存
		// ============================================================
		{
			From:       CreateV2StateConfirmSaving,
			Event:      CreateV2EventSaveCompleted,
			To:         CreateV2StateCompleted,
			StateLabel: "排班已保存 ✅",
			Act:        actOnSaveCompleted,
		},
		{
			From:  CreateV2StateConfirmSaving,
			Event: CreateV2EventUserModify,
			To:    CreateV2StateConfirmSaving,
			Act:   actModifyBeforeSave,
		},
		{
			From:  CreateV2StateConfirmSaving,
			Event: CreateV2EventUserCancel,
			To:    CreateV2StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 全局错误处理
		// ============================================================
		{
			From:  engine.State("*"),
			Event: CreateV2EventError,
			To:    CreateV2StateFailed,
			Act:   actHandleError,
		},
	}
}
