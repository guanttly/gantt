// Package regenerate 提供重排班次子工作流
// 这是一个可被 Adjust 工作流调用的独立子工作流
// 实现三阶段排班：Todo计划 -> Todo执行 -> 校验
package regenerate

// import (
// 	"context"

// 	"jusha/mcp/pkg/workflow/engine"

// 	d_model "jusha/agent/rostering/domain/model"

// 	"jusha/agent/rostering/internal/workflow/schedule/core"
// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // ============================================================
// // 重排子工作流定义
// // ============================================================

// // init 函数在包被导入时自动注册子工作流
// func init() {
// 	engine.Register(GetRegenerateWorkflowDefinition())
// }

// // GetRegenerateWorkflowDefinition 获取重排子工作流定义
// func GetRegenerateWorkflowDefinition() *engine.WorkflowDefinition {
// 	return &engine.WorkflowDefinition{
// 		Name:         WorkflowRegenerate,
// 		InitialState: RegenerateStateInit,
// 		Transitions:  buildRegenerateTransitions(),

// 		// 子工作流标记
// 		IsSubWorkflow: true,

// 		// 子工作流生命周期钩子
// 		OnSubWorkflowEnter: onRegenerateEnter,
// 		OnSubWorkflowExit:  onRegenerateExit,
// 	}
// }

// // buildRegenerateTransitions 构建重排子工作流的状态转换
// func buildRegenerateTransitions() []engine.Transition {
// 	return []engine.Transition{
// 		// ========== 初始化 ==========
// 		// 子工作流标准启动事件（由引擎 SpawnSubWorkflow 发送）
// 		{
// 			From:       RegenerateStateInit,
// 			Event:      engine.EventSubWorkflowStart,
// 			To:         RegenerateStatePreparing,
// 			StateLabel: "正在准备重排数据",
// 			Act:        actRegeneratePrepare,
// 			AfterAct:   actAfterRegeneratePrepare,
// 		},

// 		// ========== 准备阶段 ==========
// 		// 准备完成，有人数配置 -> 直接生成Todo
// 		{
// 			From:     RegenerateStatePreparing,
// 			Event:    RegenerateEventPrepared,
// 			To:       RegenerateStateTodo,
// 			Act:      actRegenerateGenerateTodo,
// 			AfterAct: actAfterRegenerateGenerateTodo,
// 		},

// 		// 需要用户确认人数
// 		{
// 			From:       RegenerateStatePreparing,
// 			Event:      RegenerateEventNeedStaffCount,
// 			To:         RegenerateStateConfirmingCount,
// 			StateLabel: "请确认每日排班人数",
// 			Act:        actRegeneratePromptStaffCount,
// 		},

// 		// 准备失败
// 		{
// 			From:  RegenerateStatePreparing,
// 			Event: RegenerateEventPrepareFailed,
// 			To:    RegenerateStateFailed,
// 			Act:   actRegenerateOnFailed,
// 		},

// 		// ========== 确认人数阶段 ==========
// 		{
// 			From:     RegenerateStateConfirmingCount,
// 			Event:    RegenerateEventStaffCountDone,
// 			To:       RegenerateStateTodo,
// 			Act:      actRegenerateGenerateTodo,
// 			AfterAct: actAfterRegenerateGenerateTodo,
// 		},

// 		// 用户放弃
// 		{
// 			From:  RegenerateStateConfirmingCount,
// 			Event: RegenerateEventAborted,
// 			To:    RegenerateStateCancelled,
// 			Act:   actRegenerateOnCancelled,
// 		},

// 		// ========== Todo生成阶段 ==========
// 		{
// 			From:       RegenerateStateTodo,
// 			Event:      RegenerateEventTodoGenerated,
// 			To:         RegenerateStateExec,
// 			StateLabel: "正在执行排班任务",
// 			Act:        actRegenerateExecuteTodos,
// 			AfterAct:   actAfterRegenerateExecuteTodos,
// 		},

// 		// Todo生成失败
// 		{
// 			From:  RegenerateStateTodo,
// 			Event: RegenerateEventFailed,
// 			To:    RegenerateStateFailed,
// 			Act:   actRegenerateOnFailed,
// 		},

// 		// ========== Todo执行阶段 ==========
// 		{
// 			From:       RegenerateStateExec,
// 			Event:      RegenerateEventTodosExecuted,
// 			To:         RegenerateStateValidate,
// 			StateLabel: "正在校验排班结果",
// 			Act:        actRegenerateValidate,
// 			AfterAct:   actAfterRegenerateValidate,
// 		},

// 		// 执行失败
// 		{
// 			From:  RegenerateStateExec,
// 			Event: RegenerateEventFailed,
// 			To:    RegenerateStateFailed,
// 			Act:   actRegenerateOnFailed,
// 		},

// 		// ========== 校验阶段 ==========
// 		{
// 			From:       RegenerateStateValidate,
// 			Event:      RegenerateEventValidated,
// 			To:         RegenerateStateCompleted,
// 			StateLabel: "重排完成",
// 			Act:        actRegenerateComplete,
// 		},

// 		// 校验失败
// 		{
// 			From:  RegenerateStateValidate,
// 			Event: RegenerateEventFailed,
// 			To:    RegenerateStateFailed,
// 			Act:   actRegenerateOnFailed,
// 		},
// 	}
// }

// // ============================================================
// // 子工作流生命周期钩子
// // ============================================================

// // onRegenerateEnter 子工作流进入时的钩子
// func onRegenerateEnter(ctx engine.Context, parentWorkflow string) error {
// 	logger := ctx.Logger()
// 	logger.Info("Entering regenerate sub-workflow",
// 		"sessionID", ctx.ID(),
// 		"parentWorkflow", parentWorkflow,
// 	)

// 	// 解析输入参数
// 	sess := ctx.Session()
// 	if sess != nil {
// 		if input, ok := sess.Data["_subworkflow_input"].(*RegenerateInput); ok {
// 			logger.Info("Regenerate input received",
// 				"shiftId", input.ShiftID,
// 				"startDate", input.StartDate,
// 				"endDate", input.EndDate,
// 			)
// 			// 保存解析后的输入供后续使用
// 			sess.Data["regenerate_input"] = input
// 		}
// 	}

// 	return nil
// }

// // onRegenerateExit 子工作流退出时的钩子
// func onRegenerateExit(ctx engine.Context, success bool) error {
// 	logger := ctx.Logger()
// 	logger.Info("Exiting regenerate sub-workflow",
// 		"sessionID", ctx.ID(),
// 		"success", success,
// 	)

// 	// 构建输出结果
// 	sess := ctx.Session()
// 	if sess != nil {
// 		shiftCtx, err := core.GetShiftSchedulingContext(sess)
// 		if err == nil && shiftCtx != nil {
// 			output := &RegenerateOutput{
// 				Success:    success,
// 				ShiftDraft: shiftCtx.ResultDraft,
// 			}

// 			if shiftCtx.TodoPlan != nil {
// 				output.TodoCount = len(shiftCtx.TodoPlan.TodoList)
// 				for _, todo := range shiftCtx.TodoPlan.TodoList {
// 					switch todo.Status {
// 					case "completed":
// 						output.CompletedCount++
// 					case "failed":
// 						output.FailedCount++
// 					}
// 				}
// 			}

// 			sess.Data["_subworkflow_output"] = output
// 		}
// 	}

// 	return nil
// }

// // ============================================================
// // Action 实现
// // ============================================================

// // actRegeneratePrepare 准备重排数据
// func actRegeneratePrepare(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Preparing regenerate data")

// 	// 获取输入参数
// 	input, ok := sess.Data["regenerate_input"].(*RegenerateInput)
// 	if !ok {
// 		return wctx.Send(ctx, RegenerateEventPrepareFailed, "missing regenerate input")
// 	}

// 	// 创建或获取 ShiftSchedulingContext
// 	shiftCtx := &d_model.ShiftSchedulingContext{
// 		SourceWorkflow:    WorkflowRegenerate,
// 		StartDate:         input.StartDate,
// 		EndDate:           input.EndDate,
// 		StaffRequirements: input.StaffRequirements,
// 	}

// 	// 设置班次信息（需要从父工作流获取完整的 Shift 对象）
// 	// 这里暂时创建一个简化的 Shift
// 	shiftCtx.Shift = &d_model.Shift{
// 		ID:   input.ShiftID,
// 		Name: input.ShiftName,
// 	}

// 	// 保存上下文
// 	if err := core.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return err
// 	}

// 	return nil
// }

// // actAfterRegeneratePrepare 准备完成后的流转
// func actAfterRegeneratePrepare(ctx context.Context, wctx engine.Context, payload any) error {
// 	sess := wctx.Session()

// 	// 获取输入参数
// 	input, ok := sess.Data["regenerate_input"].(*RegenerateInput)
// 	if !ok {
// 		return wctx.Send(ctx, RegenerateEventPrepareFailed, "missing regenerate input")
// 	}

// 	// 检查是否有人数配置
// 	if len(input.StaffRequirements) > 0 || input.SkipStaffCountConfirm {
// 		return wctx.Send(ctx, RegenerateEventPrepared, nil)
// 	}

// 	// 需要用户确认人数
// 	return wctx.Send(ctx, RegenerateEventNeedStaffCount, nil)
// }

// // actRegeneratePromptStaffCount 提示用户确认人数
// func actRegeneratePromptStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	logger.Info("Prompting for staff count confirmation")

// 	// TODO: 发送人数确认表单消息
// 	// 这里需要构建表单让用户输入每日人数

// 	return nil
// }

// // actRegenerateGenerateTodo 生成Todo计划
// func actRegenerateGenerateTodo(ctx context.Context, wctx engine.Context, payload any) error {
// 	// 调用核心的 GenerateShiftTodoPlan
// 	return core.GenerateShiftTodoPlan(ctx, wctx)
// }

// // actAfterRegenerateGenerateTodo Todo生成后的流转
// func actAfterRegenerateGenerateTodo(ctx context.Context, wctx engine.Context, payload any) error {
// 	sess := wctx.Session()

// 	shiftCtx, err := core.GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return wctx.Send(ctx, RegenerateEventFailed, err.Error())
// 	}

// 	if shiftCtx.TodoPlan == nil || len(shiftCtx.TodoPlan.TodoList) == 0 {
// 		return wctx.Send(ctx, RegenerateEventFailed, "no todo plan generated")
// 	}

// 	return wctx.Send(ctx, RegenerateEventTodoGenerated, nil)
// }

// // actRegenerateExecuteTodos 执行Todo任务
// func actRegenerateExecuteTodos(ctx context.Context, wctx engine.Context, payload any) error {
// 	// 调用核心的 ExecuteShiftTodos
// 	return core.ExecuteShiftTodos(ctx, wctx)
// }

// // actAfterRegenerateExecuteTodos Todo执行后的流转
// func actAfterRegenerateExecuteTodos(ctx context.Context, wctx engine.Context, payload any) error {
// 	return wctx.Send(ctx, RegenerateEventTodosExecuted, nil)
// }

// // actRegenerateValidate 校验排班结果
// func actRegenerateValidate(ctx context.Context, wctx engine.Context, payload any) error {
// 	// 调用核心的 ValidateShiftSchedule
// 	return core.ValidateShiftSchedule(ctx, wctx)
// }

// // actAfterRegenerateValidate 校验后的流转
// func actAfterRegenerateValidate(ctx context.Context, wctx engine.Context, payload any) error {
// 	return wctx.Send(ctx, RegenerateEventValidated, nil)
// }

// // actRegenerateComplete 重排完成
// func actRegenerateComplete(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	shiftCtx, _ := core.GetShiftSchedulingContext(sess)
// 	if shiftCtx != nil {
// 		logger.Info("Regenerate completed",
// 			"shiftId", shiftCtx.Shift.ID,
// 			"shiftName", shiftCtx.Shift.Name,
// 		)
// 	}

// 	// 清理共享上下文
// 	_ = core.ClearShiftSchedulingContext(ctx, wctx)

// 	return nil
// }

// // actRegenerateOnFailed 处理失败
// func actRegenerateOnFailed(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()

// 	errMsg := "unknown error"
// 	if msg, ok := payload.(string); ok {
// 		errMsg = msg
// 	}

// 	logger.Error("Regenerate failed", "error", errMsg)

// 	// 清理共享上下文
// 	_ = core.ClearShiftSchedulingContext(ctx, wctx)

// 	return nil
// }

// // actRegenerateOnCancelled 处理取消
// func actRegenerateOnCancelled(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	logger.Info("Regenerate cancelled by user")

// 	// 清理共享上下文
// 	_ = core.ClearShiftSchedulingContext(ctx, wctx)

// 	return nil
// }
