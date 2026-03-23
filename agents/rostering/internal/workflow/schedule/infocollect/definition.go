// Package infocollect 提供信息收集子工作流定义
// 这是一个可被 Create 和 Adjust 工作流调用的子工作流
// 支持多分支收集不同信息
package infocollect

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 信息收集子工作流定义
// ============================================================

// init 函数在包被导入时自动注册子工作流
func init() {
	engine.Register(GetInfoCollectWorkflowDefinition())
}

// GetInfoCollectWorkflowDefinition 获取信息收集子工作流定义
func GetInfoCollectWorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowInfoCollect,
		InitialState: InfoCollectStateInit,
		Transitions:  buildInfoCollectTransitions(),

		// 子工作流标记
		IsSubWorkflow: true,

		// 子工作流生命周期钩子
		OnSubWorkflowEnter: onInfoCollectEnter,
		OnSubWorkflowExit:  onInfoCollectExit,
	}
}

// buildInfoCollectTransitions 构建信息收集子工作流的状态转换
func buildInfoCollectTransitions() []engine.Transition {
	return []engine.Transition{
		// ========== 初始状态 - 多分支起点 ==========
		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
		{
			From:       InfoCollectStateInit,
			Event:      engine.EventSubWorkflowStart,
			To:         InfoCollectStateConfirmingPeriod,
			StateLabel: "正在确认排班周期",
			Act:        actInfoCollectStart,
			AfterAct:   actInfoCollectAfterStart,
		},
		// 分支2: 跳过周期（直接查询班次）
		{
			From:       InfoCollectStateInit,
			Event:      InfoCollectEventSkipPeriod,
			To:         InfoCollectStateQueryingShifts,
			StateLabel: "正在查询可用班次",
			Act:        actInfoCollectQueryShifts,
		},
		// 分支3: 跳过班次选择（直接确认人数）
		{
			From:       InfoCollectStateInit,
			Event:      InfoCollectEventSkipShifts,
			To:         InfoCollectStateConfirmingStaffCount,
			StateLabel: "正在确认班次人数",
			Act:        actInfoCollectConfirmStaffCount,
		},
		// 分支4: 直接数据检索
		{
			From:       InfoCollectStateInit,
			Event:      InfoCollectEventSkipToData,
			To:         InfoCollectStateRetrievingStaff,
			StateLabel: "正在检索可用人员",
			Act:        actInfoCollectRetrieveStaff,
		},

		// ========== 阶段1: 确认排班周期 ==========
		{
			From:       InfoCollectStateConfirmingPeriod,
			Event:      InfoCollectEventPeriodConfirmed,
			To:         InfoCollectStateQueryingShifts,
			StateLabel: "正在查询可用班次",
			Act:        actInfoCollectConfirmPeriod,
			AfterAct:   actInfoCollectTriggerQueryShifts,
		},
		{
			From:  InfoCollectStateConfirmingPeriod,
			Event: InfoCollectEventPeriodModified,
			To:    InfoCollectStateConfirmingPeriod,
			Act:   actInfoCollectModifyPeriod,
		},
		{
			From:  InfoCollectStateConfirmingPeriod,
			Event: InfoCollectEventCancel,
			To:    InfoCollectStateCancelled,
			Act:   actInfoCollectCancel,
		},

		// ========== 阶段2: 查询可用班次 ==========
		{
			From:       InfoCollectStateQueryingShifts,
			Event:      InfoCollectEventShiftsQueried,
			To:         InfoCollectStateConfirmingShifts,
			StateLabel: "正在确认排班班次",
			Act:        actInfoCollectQueryShifts,
		},
		{
			From:  InfoCollectStateQueryingShifts,
			Event: InfoCollectEventCancel,
			To:    InfoCollectStateCancelled,
			Act:   actInfoCollectCancel,
		},

		// ========== 阶段3: 确认排班班次 ==========
		{
			From:       InfoCollectStateConfirmingShifts,
			Event:      InfoCollectEventShiftsConfirmed,
			To:         InfoCollectStateConfirmingStaffCount,
			StateLabel: "正在确认班次人数",
			Act:        actInfoCollectConfirmShifts,
		},
		{
			From:  InfoCollectStateConfirmingShifts,
			Event: InfoCollectEventShiftsModified,
			To:    InfoCollectStateConfirmingShifts,
			Act:   actInfoCollectModifyShifts,
		},
		{
			From:  InfoCollectStateConfirmingShifts,
			Event: InfoCollectEventCancel,
			To:    InfoCollectStateCancelled,
			Act:   actInfoCollectCancel,
		},

		// ========== 阶段4: 确认班次人数 ==========
		{
			From:       InfoCollectStateConfirmingStaffCount,
			Event:      InfoCollectEventStaffCountConfirmed,
			To:         InfoCollectStateRetrievingStaff,
			StateLabel: "正在检索可用人员",
			Act:        actInfoCollectConfirmStaffCount,
			AfterAct:   actInfoCollectTriggerRetrieveStaff,
		},
		{
			From:  InfoCollectStateConfirmingStaffCount,
			Event: InfoCollectEventStaffCountModified,
			To:    InfoCollectStateConfirmingStaffCount,
			Act:   actInfoCollectModifyStaffCount,
		},
		{
			From:  InfoCollectStateConfirmingStaffCount,
			Event: InfoCollectEventCancel,
			To:    InfoCollectStateCancelled,
			Act:   actInfoCollectCancel,
		},

		// ========== 阶段5: 检索可用人员 ==========
		{
			From:       InfoCollectStateRetrievingStaff,
			Event:      InfoCollectEventStaffRetrieved,
			To:         InfoCollectStateRetrievingRules,
			StateLabel: "正在检索排班规则",
			Act:        actInfoCollectRetrieveStaff,
			AfterAct:   actInfoCollectTriggerRetrieveRules,
		},

		// ========== 阶段6: 检索排班规则 ==========
		{
			From:       InfoCollectStateRetrievingRules,
			Event:      InfoCollectEventRulesRetrieved,
			To:         InfoCollectStateCompleted,
			StateLabel: "信息收集完成",
			Act:        actInfoCollectRetrieveRules,
			AfterAct:   actInfoCollectTriggerComplete,
		},

		// ========== 终态 ==========
		{
			From:  InfoCollectStateCompleted,
			Event: InfoCollectEventReturn,
			To:    InfoCollectStateCompleted,
			Act:   actInfoCollectReturnToParent,
		},
		{
			From:  InfoCollectStateCancelled,
			Event: InfoCollectEventReturn,
			To:    InfoCollectStateCancelled,
			Act:   actInfoCollectReturnToParentWithCancel,
		},
	}
}

// ============================================================
// 生命周期钩子
// ============================================================

// onInfoCollectEnter 子工作流进入时的钩子
func onInfoCollectEnter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	logger.Info("Entering info-collect sub-workflow",
		"sessionID", ctx.ID(),
		"parentWorkflow", parentWorkflow,
	)

	// 解析输入参数，设置跳过标记
	sess := ctx.Session()
	if sess != nil {
		if input, ok := sess.Data["_subworkflow_input"].(*InfoCollectInput); ok {
			logger.Info("InfoCollect input received",
				"sourceType", input.SourceType,
				"skipPhases", input.SkipPhases,
			)
			// 保存解析后的输入供后续使用
			sess.Data["info_collect_input"] = input
		}
	}

	return nil
}

// onInfoCollectExit 子工作流退出时的钩子
func onInfoCollectExit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	logger.Info("Exiting info-collect sub-workflow",
		"sessionID", ctx.ID(),
		"success", success,
	)

	return nil
}
