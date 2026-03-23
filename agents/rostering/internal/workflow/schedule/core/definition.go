// Package core 提供排班核心子工作流定义
// 这是一个可被 Create 和 Adjust 工作流调用的子工作流
package core

// import (
// 	"jusha/mcp/pkg/workflow/engine"

// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // ============================================================
// // 排班核心子工作流定义
// // ============================================================

// // init 函数在包被导入时自动注册子工作流
// func init() {
// 	engine.Register(GetSchedulingCoreWorkflowDefinition())
// }

// // GetSchedulingCoreWorkflowDefinition 获取排班核心子工作流定义
// func GetSchedulingCoreWorkflowDefinition() *engine.WorkflowDefinition {
// 	return &engine.WorkflowDefinition{
// 		Name:         WorkflowSchedulingCore,
// 		InitialState: CoreStateGeneratingTodo,
// 		Transitions:  buildCoreTransitions(),

// 		// 子工作流标记
// 		IsSubWorkflow: true,

// 		// 子工作流生命周期钩子
// 		OnSubWorkflowEnter: onCoreEnter,
// 		OnSubWorkflowExit:  onCoreExit,
// 	}
// }

// // buildCoreTransitions 构建核心子工作流的状态转换
// func buildCoreTransitions() []engine.Transition {
// 	return []engine.Transition{
// 		// ========== 阶段1: 生成 Todo 计划 ==========
// 		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
// 		{
// 			From:     CoreStateGeneratingTodo,
// 			Event:    engine.EventSubWorkflowStart,
// 			To:       CoreStateGeneratingTodo,
// 			Act:      actCoreGenerateTodoPlan,
// 			AfterAct: actCoreAfterGenerateTodoPlan, // 生成成功后直接触发执行
// 		},
// 		{
// 			From:       CoreStateGeneratingTodo,
// 			Event:      CoreEventTodoGenerated,
// 			To:         CoreStateExecutingTodo,
// 			StateLabel: "正在执行排班计划",
// 			AfterAct:   actCoreTriggerFirstTodo, // 触发第一个 Todo 执行
// 		},
// 		{
// 			From:  CoreStateGeneratingTodo,
// 			Event: CoreEventError,
// 			To:    CoreStateFailed,
// 			Act:   actCoreHandleError,
// 		},

// 		// ========== 阶段2: 执行 Todo 计划 ==========
// 		{
// 			From:       CoreStateExecutingTodo,
// 			Event:      CoreEventExecuteTodo,
// 			To:         CoreStateExecutingTodo,
// 			StateLabel: "正在执行排班任务",
// 			Act:        actCoreExecuteTodoItem,
// 		},
// 		{
// 			From:       CoreStateExecutingTodo,
// 			Event:      CoreEventTodoComplete,
// 			To:         CoreStateCompleted,
// 			StateLabel: "排班核心流程完成",
// 			Act:        actCoreComplete, // Todo 完成后直接进入 Completed
// 		},
// 		{
// 			From:  CoreStateExecutingTodo,
// 			Event: CoreEventError,
// 			To:    CoreStateFailed,
// 			Act:   actCoreHandleError,
// 		},

// 		// ========== 错误恢复 ==========
// 		// 重试：从头开始重新执行核心流程
// 		{
// 			From:       CoreStateFailed,
// 			Event:      CoreEventRetry,
// 			To:         CoreStateGeneratingTodo,
// 			StateLabel: "正在重新生成排班计划",
// 			Act:        actCoreRetry,
// 		},
// 		// 跳过：跳过当前班次，返回空结果给父工作流
// 		{
// 			From:       CoreStateFailed,
// 			Event:      CoreEventSkip,
// 			To:         CoreStateCompleted,
// 			StateLabel: "已跳过当前班次",
// 			Act:        actCoreSkip,
// 		},

// 		// ========== 全局错误处理 ==========
// 		// 捕获所有未处理的错误
// 		{
// 			From:  engine.State("*"),
// 			Event: CoreEventError,
// 			To:    CoreStateFailed,
// 			Act:   actCoreHandleError,
// 		},

// 		// ========== 终态 ==========
// 		// 注意：CoreStateCompleted 和 CoreStateFailed 是终态
// 		// 返回父工作流的逻辑直接在 actCoreComplete/actCoreSkip 中调用，不需要额外的转换
// 	}
// }

// // ============================================================
// // 生命周期钩子
// // ============================================================

// // onCoreEnter 子工作流进入时的钩子
// func onCoreEnter(ctx engine.Context, parentWorkflow string) error {
// 	logger := ctx.Logger()
// 	logger.Info("Entering scheduling core sub-workflow",
// 		"sessionID", ctx.ID(),
// 		"parentWorkflow", parentWorkflow,
// 	)

// 	// 可以在这里做一些初始化工作
// 	// 例如：验证输入数据是否完整

// 	return nil
// }

// // onCoreExit 子工作流退出时的钩子
// func onCoreExit(ctx engine.Context, success bool) error {
// 	logger := ctx.Logger()
// 	logger.Info("Exiting scheduling core sub-workflow",
// 		"sessionID", ctx.ID(),
// 		"success", success,
// 	)

// 	// 可以在这里做一些清理工作
// 	// 例如：清理临时数据

// 	return nil
// }
