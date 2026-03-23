// Package confirmsave 提供确认保存子工作流定义
// 这是一个可被 Create 和 Adjust 工作流调用的子工作流
package confirmsave

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 确认保存子工作流定义
// ============================================================

// init 函数在包被导入时自动注册子工作流
func init() {
	engine.Register(GetConfirmSaveWorkflowDefinition())
}

// GetConfirmSaveWorkflowDefinition 获取确认保存子工作流定义
func GetConfirmSaveWorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowConfirmSave,
		InitialState: ConfirmSaveStatePreview,
		Transitions:  buildConfirmSaveTransitions(),

		// 子工作流标记
		IsSubWorkflow: true,

		// 子工作流生命周期钩子
		OnSubWorkflowEnter: onConfirmSaveEnter,
		OnSubWorkflowExit:  onConfirmSaveExit,
	}
}

// buildConfirmSaveTransitions 构建确认保存子工作流的状态转换
func buildConfirmSaveTransitions() []engine.Transition {
	return []engine.Transition{
		// ========== 阶段1: 预览草案 ==========
		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
		{
			From:       ConfirmSaveStatePreview,
			Event:      engine.EventSubWorkflowStart,
			To:         ConfirmSaveStatePreview,
			StateLabel: "正在生成排班预览",
			Act:        actConfirmSaveGeneratePreview,
		},
		{
			From:       ConfirmSaveStatePreview,
			Event:      ConfirmSaveEventPreviewReady,
			To:         ConfirmSaveStateConfirming,
			StateLabel: "请确认排班方案",
			Act:        nil,                       // 状态转换，无动作
			AfterAct:   actConfirmSaveShowButtons, // 状态转换后显示按钮
		},
		{
			From:  ConfirmSaveStatePreview,
			Event: ConfirmSaveEventCancel,
			To:    ConfirmSaveStateCancelled,
			Act:   actConfirmSaveCancel,
		},

		// ========== 阶段2: 确认草案 ==========
		{
			From:       ConfirmSaveStateConfirming,
			Event:      ConfirmSaveEventConfirm,
			To:         ConfirmSaveStateSaving,
			StateLabel: "正在保存排班",
			Act:        actConfirmSaveConfirm,
		},
		{
			From:       ConfirmSaveStateConfirming,
			Event:      ConfirmSaveEventReject,
			To:         ConfirmSaveStateCancelled,
			StateLabel: "草案已拒绝",
			Act:        actConfirmSaveReject,
		},
		{
			From:  ConfirmSaveStateConfirming,
			Event: ConfirmSaveEventCancel,
			To:    ConfirmSaveStateCancelled,
			Act:   actConfirmSaveCancel,
		},

		// ========== 阶段3: 保存 ==========
		{
			From:       ConfirmSaveStateSaving,
			Event:      ConfirmSaveEventSaveSuccess,
			To:         ConfirmSaveStateCompleted,
			StateLabel: "保存成功",
			Act:        actConfirmSaveSaveSuccess,
		},
		{
			From:       ConfirmSaveStateSaving,
			Event:      ConfirmSaveEventSaveFailed,
			To:         ConfirmSaveStateFailed,
			StateLabel: "保存失败",
			Act:        actConfirmSaveSaveFailed,
		},

		// ========== 失败后重试 ==========
		{
			From:       ConfirmSaveStateFailed,
			Event:      ConfirmSaveEventRetry,
			To:         ConfirmSaveStateSaving,
			StateLabel: "正在重试保存",
			Act:        actConfirmSaveRetry,
		},
		{
			From:  ConfirmSaveStateFailed,
			Event: ConfirmSaveEventCancel,
			To:    ConfirmSaveStateCancelled,
			Act:   actConfirmSaveCancel,
		},

		// ========== 终态 ==========
		{
			From:  ConfirmSaveStateCompleted,
			Event: ConfirmSaveEventReturn,
			To:    ConfirmSaveStateCompleted,
			Act:   actConfirmSaveReturnToParent,
		},
		{
			From:  ConfirmSaveStateCancelled,
			Event: ConfirmSaveEventReturn,
			To:    ConfirmSaveStateCancelled,
			Act:   actConfirmSaveReturnToParentWithCancel,
		},
		{
			From:  ConfirmSaveStateFailed,
			Event: ConfirmSaveEventReturn,
			To:    ConfirmSaveStateFailed,
			Act:   actConfirmSaveReturnToParentWithError,
		},
	}
}

// ============================================================
// 生命周期钩子
// ============================================================

// onConfirmSaveEnter 子工作流进入时的钩子
func onConfirmSaveEnter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	logger.Info("Entering confirm-save sub-workflow",
		"sessionID", ctx.ID(),
		"parentWorkflow", parentWorkflow,
	)

	return nil
}

// onConfirmSaveExit 子工作流退出时的钩子
func onConfirmSaveExit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	logger.Info("Exiting confirm-save sub-workflow",
		"sessionID", ctx.ID(),
		"success", success,
	)

	return nil
}
