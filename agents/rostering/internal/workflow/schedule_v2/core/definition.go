// 这是一个单班次创建排班的核心工作流
package core

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班核心子工作流定义
// ============================================================

/*
常规工作流：进入班次->检验必要数据->成功->生成 Todo 计划->执行 Todo 计划->询问用户是否需要调整->不需要调整->完成班次
															|                                                        |->用户调整需求->执行用户需求->询问用户是否需要调整
												      |->失败->用户交互->用户填充数据->检验必要数据
																						 |->用户取消->结束

异常工作流：任何步骤出错->进入失败状态->用户选择重试或跳过->完成班次
*/

// init 函数在包被导入时自动注册子工作流
func init() {
	engine.Register(GetSchedulingCoreWorkflowDefinition())
}

// GetSchedulingCoreWorkflowDefinition 获取排班核心子工作流定义
func GetSchedulingCoreWorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowSchedulingCore,
		// 子工作流从预验证阶段开始，然后再进入生成 Todo 等后续阶段
		// 注意：这里必须与 buildCoreTransitions 中的 EventSubWorkflowStart 的 From 状态保持一致
		InitialState: CoreStatePreValidate,
		Transitions:  buildCoreTransitions(),

		// 子工作流标记
		IsSubWorkflow: true,

		// 子工作流生命周期钩子
		OnSubWorkflowEnter: onCoreEnter,
		OnSubWorkflowExit:  onCoreExit,
	}
}

// buildCoreTransitions 构建核心子工作流的状态转换
func buildCoreTransitions() []engine.Transition {
	return []engine.Transition{
		// ========== 阶段1: 验证数据 ==========
		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
		{
			From:     CoreStatePreValidate,
			Event:    engine.EventSubWorkflowStart,
			To:       CoreStateValidating,
			Act:      actValidate,
			AfterAct: actAfterValidate,
		},

		// ========== 阶段1.1: 验证不通过 ==========
		{
			From:     CoreStateValidating,
			Event:    CoreEventValidate,
			To:       CoreStateValidating,
			Act:      actValidate,
			AfterAct: actAfterValidate,
		},

		// ========== 阶段2: 生成计划 ==========
		{
			From:     CoreStateValidating,
			Event:    CoreEventTodoGenerated,
			To:       CoreStateGeneratingTodo,
			Act:      actTodoGenerated,
			AfterAct: actAfterTodoGenerated,
		},

		// ========== 阶段2.1: 调整计划 ==========
		{
			From:     CoreStateGeneratingTodo,
			Event:    CoreEventAdjustPlan,
			To:       CoreStateGeneratingTodo,
			Act:      actTodoAdjust,
			AfterAct: actAfterTodoAdjust,
		},

		// ========== 阶段3: 执行计划 ==========
		{
			From:     CoreStateGeneratingTodo,
			Event:    CoreEventExecuteTodo,
			To:       CoreStateExecutingTodo,
			Act:      actTodoExecute,
			AfterAct: actAfterTodoExecute,
		},

		{
			From:     CoreStateExecutingTodo,
			Event:    CoreEventExecuteTodo,
			To:       CoreStateExecutingTodo,
			Act:      actTodoExecute,
			AfterAct: actAfterTodoExecute,
		},

		// ========== 阶段3.1: 调整排班 ==========
		{
			From:     CoreStateExecutingTodo,
			Event:    CoreEventUserRequest,
			To:       CoreStateExecutingTodo,
			Act:      actUserRequestExecute,
			AfterAct: actAfterTodoExecute,
		},

		// ========== 阶段4: 确认排班 ==========
		{
			From:     CoreStateExecutingTodo,
			Event:    CoreEventTodoComplete,
			To:       CoreStateCompleted,
			Act:      actCoreComplete,
			AfterAct: actCoreAfterComplete,
		},

		// ========== 全局错误处理 ==========
		// 捕获所有未处理的错误
		{
			From:  engine.State("*"),
			Event: CoreEventError,
			To:    CoreStateFailed,
			Act:   actCoreHandleError,
		},

		// ========== 终态 ==========
		// 注意：CoreStateCompleted 和 CoreStateFailed 是终态
		// 返回父工作流的逻辑直接在 actCoreComplete/actCoreSkip 中调用，不需要额外的转换
	}
}

// ============================================================
// 生命周期钩子
// ============================================================

// onCoreEnter 子工作流进入时的钩子
func onCoreEnter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	logger.Info("Entering scheduling core sub-workflow",
		"sessionID", ctx.ID(),
		"parentWorkflow", parentWorkflow,
	)

	// 可以在这里做一些初始化工作
	// 例如：验证输入数据是否完整

	return nil
}

// onCoreExit 子工作流退出时的钩子
func onCoreExit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	logger.Info("Exiting scheduling core sub-workflow",
		"sessionID", ctx.ID(),
		"success", success,
	)

	// 可以在这里做一些清理工作
	// 例如：清理临时数据

	return nil
}
