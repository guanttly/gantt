package adjust

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班调整子工作流 V2 定义
// ============================================================

// init 函数在包被导入时自动注册子工作流
func init() {
	engine.Register(GetScheduleAdjustV2WorkflowDefinition())
}

// GetScheduleAdjustV2WorkflowDefinition 获取排班调整子工作流定义
func GetScheduleAdjustV2WorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowScheduleAdjustV2,
		InitialState: AdjustV2StateInit,
		Transitions:  buildAdjustV2Transitions(),

		// 子工作流标记
		IsSubWorkflow: true,

		// 子工作流生命周期钩子
		OnSubWorkflowEnter: onAdjustV2Enter,
		OnSubWorkflowExit:  onAdjustV2Exit,
	}
}

// buildAdjustV2Transitions 构建调整子工作流的状态转换
func buildAdjustV2Transitions() []engine.Transition {
	return []engine.Transition{
		// ========== 阶段1: 初始化并直接应用调整 ==========
		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
		// 直接进入修改模式，不再进行意图分析
		{
			From:     AdjustV2StateInit,
			Event:    engine.EventSubWorkflowStart,
			To:       AdjustV2StateCompleted,
			Act:      actAdjustV2Init,
			AfterAct: actApplyAdjustmentDirect, // 直接应用调整，跳过意图分析
		},

		// ========== 全局错误处理 ==========
		{
			From:  engine.State("*"),
			Event: AdjustV2EventError,
			To:    AdjustV2StateFailed,
			Act:   actAdjustV2HandleError,
		},

		// ========== 终态 ==========
		// AdjustV2StateCompleted 和 AdjustV2StateFailed 是终态
		// 返回父工作流的逻辑直接在 actAdjustV2AfterComplete 中调用
	}
}

// ============================================================
// 生命周期钩子
// ============================================================

// onAdjustV2Enter 子工作流进入时的钩子
func onAdjustV2Enter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	logger.Info("Entering schedule adjust V2 sub-workflow",
		"sessionID", ctx.ID(),
		"parentWorkflow", parentWorkflow,
	)

	return nil
}

// onAdjustV2Exit 子工作流退出时的钩子
func onAdjustV2Exit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	logger.Info("Exiting schedule adjust V2 sub-workflow",
		"sessionID", ctx.ID(),
		"success", success,
	)

	return nil
}
