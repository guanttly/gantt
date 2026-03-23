package collectstaffcount

import (
	"fmt"

	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// init 函数在包被导入时自动注册工作流
func init() {
	engine.Register(&collectStaffCountDefinition)
}

// collectStaffCountDefinition 收集班次人数工作流定义
var collectStaffCountDefinition = engine.WorkflowDefinition{
	Name:               WorkflowCollectStaffCount,
	InitialState:       CollectStaffCountStateInit,
	Transitions:        buildTransitions(),
	IsSubWorkflow:      true,
	OnSubWorkflowEnter: onEnter,
	OnSubWorkflowExit:  onExit,
}

// GetDefinition 获取收集班次人数子工作流定义（保持向后兼容）
func GetDefinition() engine.WorkflowDefinition {
	return collectStaffCountDefinition
}

// buildTransitions 构建状态转换
func buildTransitions() []engine.Transition {
	return []engine.Transition{
		// 子工作流启动
		{
			From:       CollectStaffCountStateInit,
			Event:      engine.EventSubWorkflowStart,
			To:         CollectStaffCountStateConfirming,
			StateLabel: "正在确认班次人数",
			Act:        actStart,
		},

		// 确认人数
		{
			From:       CollectStaffCountStateConfirming,
			Event:      CollectStaffCountEventConfirmed,
			To:         CollectStaffCountStateCompleted,
			StateLabel: "人数收集完成",
			Act:        actConfirm,
			AfterAct:   actTriggerReturn,
		},

		// 修改人数（停留在当前状态）
		{
			From:  CollectStaffCountStateConfirming,
			Event: CollectStaffCountEventModified,
			To:    CollectStaffCountStateConfirming,
			Act:   actModify,
		},

		// 取消
		{
			From:  CollectStaffCountStateConfirming,
			Event: CollectStaffCountEventCancel,
			To:    CollectStaffCountStateCancelled,
			Act:   actCancel,
		},

		// 返回父工作流（成功）
		{
			From:  CollectStaffCountStateCompleted,
			Event: CollectStaffCountEventReturn,
			To:    CollectStaffCountStateCompleted,
			Act:   actReturnToParent,
		},

		// 返回父工作流（取消）
		{
			From:  CollectStaffCountStateCancelled,
			Event: CollectStaffCountEventReturn,
			To:    CollectStaffCountStateCancelled,
			Act:   actReturnToParentWithCancel,
		},
	}
}

// onEnter 子工作流进入时的钩子
func onEnter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	sess := ctx.Session()

	logger.Info("Entering collect-staff-count sub-workflow",
		"sessionID", ctx.ID(),
		"parentWorkflow", parentWorkflow,
	)

	if sess == nil {
		return fmt.Errorf("session not found")
	}

	// 直接从session获取强类型输入
	input, ok := sess.Data[engine.DataKeySubWorkflowInput].(*CollectStaffCountInput)
	if !ok {
		logger.Error("SubWorkflow input type mismatch",
			"expectedType", "*CollectStaffCountInput",
			"actualType", fmt.Sprintf("%T", sess.Data[engine.DataKeySubWorkflowInput]),
		)
		return fmt.Errorf("sub-workflow input must be *CollectStaffCountInput, got %T", sess.Data[engine.DataKeySubWorkflowInput])
	}

	if input == nil {
		return fmt.Errorf("sub-workflow input is nil")
	}

	logger.Info("CollectStaffCount input validated",
		"startDate", input.StartDate,
		"endDate", input.EndDate,
		"shiftCount", len(input.ShiftIDs),
	)

	return nil
}

// onExit 子工作流退出时的钩子
func onExit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	logger.Info("Exiting collect-staff-count sub-workflow",
		"sessionID", ctx.ID(),
		"success", success,
	)

	return nil
}
