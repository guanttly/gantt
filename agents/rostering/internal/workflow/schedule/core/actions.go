// Package core 提供排班核心逻辑的 Action 实现
package core

// import (
// 	"context"
// 	"fmt"

// 	"jusha/mcp/pkg/workflow/engine"
// 	"jusha/mcp/pkg/workflow/session"

// 	d_model "jusha/agent/rostering/domain/model"

// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // ============================================================
// // 核心子工作流 Actions
// // ============================================================

// // actCoreGenerateTodoPlan 生成 Todo 计划
// func actCoreGenerateTodoPlan(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()
// 	sessionID := sess.ID

// 	logger.Info("Core: Generating Todo plan", "sessionID", sessionID)

// 	// 调用核心函数生成 Todo 计划
// 	if err := GenerateShiftTodoPlan(ctx, wctx); err != nil {
// 		logger.Error("Core: Failed to generate Todo plan", "error", err)
// 		// 返回错误，由 AfterAct 处理
// 		return err
// 	}

// 	logger.Info("Core: Todo plan generated successfully")
// 	return nil
// }

// // actCoreAfterGenerateTodoPlan 处理 Todo 计划生成后的流转
// func actCoreAfterGenerateTodoPlan(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()

// 	// 如果 Act 返回了错误，这里不会被调用
// 	// 错误情况会由工作流引擎自动触发 CoreEventError

// 	logger.Info("Core: Todo plan generated, transitioning to execution state")
// 	// 触发 TodoGenerated 事件以转换到 ExecutingTodo 状态
// 	return wctx.Send(ctx, CoreEventTodoGenerated, nil)
// }

// // actCoreTriggerFirstTodo 触发执行第一个 Todo（AfterAct）
// func actCoreTriggerFirstTodo(ctx context.Context, wctx engine.Context, payload any) error {
// 	return wctx.Send(ctx, CoreEventExecuteTodo, nil)
// }

// // actCoreExecuteTodoItem 执行单个 Todo 项
// func actCoreExecuteTodoItem(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	// 获取排班上下文
// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	todoCount := 0
// 	if shiftCtx.TodoPlan != nil {
// 		todoCount = len(shiftCtx.TodoPlan.TodoList)
// 	}

// 	logger.Info("Core: Executing Todo item",
// 		"sessionID", sess.ID,
// 		"currentIndex", shiftCtx.CurrentTodoIndex,
// 		"totalTodos", todoCount,
// 	)

// 	// 执行 Todo 任务
// 	if err := ExecuteShiftTodos(ctx, wctx); err != nil {
// 		return err
// 	}

// 	// 重新获取更新后的上下文检查是否所有 Todo 完成
// 	shiftCtx, _ = GetShiftSchedulingContext(sess)
// 	todoCount = 0
// 	if shiftCtx.TodoPlan != nil {
// 		todoCount = len(shiftCtx.TodoPlan.TodoList)
// 	}

// 	if shiftCtx.CurrentTodoIndex >= todoCount {
// 		logger.Info("Core: All Todo items completed")
// 		return wctx.Send(ctx, CoreEventTodoComplete, nil)
// 	}

// 	// 继续执行下一个 Todo
// 	return wctx.Send(ctx, CoreEventExecuteTodo, nil)
// }

// // actCoreComplete 核心流程完成
// func actCoreComplete(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Core: Schedule core completed", "sessionID", sess.ID)

// 	// 直接调用返回逻辑（避免事件处理时状态不匹配）
// 	return actCoreReturnToParent(ctx, wctx, payload)
// }

// // actCoreHandleError 处理错误，显示重试/跳过选项
// func actCoreHandleError(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	var errMsg string
// 	if err, ok := payload.(error); ok {
// 		errMsg = err.Error()
// 	} else if s, ok := payload.(string); ok {
// 		errMsg = s
// 	} else {
// 		errMsg = "unknown error"
// 	}

// 	logger.Error("Core: Schedule core failed",
// 		"sessionID", sess.ID,
// 		"error", errMsg,
// 	)

// 	// 保存错误信息到 session
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "core_error", errMsg); err != nil {
// 		logger.Warn("Failed to save error to session", "error", err)
// 	}

// 	// 获取当前班次信息用于提示
// 	shiftCtx, _ := GetShiftSchedulingContext(sess)
// 	shiftName := "当前班次"
// 	if shiftCtx != nil && shiftCtx.Shift != nil {
// 		shiftName = shiftCtx.Shift.Name
// 	}

// 	// 发送错误消息，包含重试/跳过按钮
// 	message := fmt.Sprintf("❌ 排班过程中遇到错误：\n\n%s\n\n请选择操作：", errMsg)
// 	actions := []session.WorkflowAction{
// 		{
// 			ID:    "retry",
// 			Type:  session.ActionTypeWorkflow,
// 			Label: "🔄 重试",
// 			Event: session.WorkflowEvent(CoreEventRetry),
// 			Style: session.ActionStylePrimary,
// 		},
// 		{
// 			ID:    "skip",
// 			Type:  session.ActionTypeWorkflow,
// 			Label: fmt.Sprintf("⏭️ 跳过【%s】", shiftName),
// 			Event: session.WorkflowEvent(CoreEventSkip),
// 			Style: session.ActionStyleSecondary,
// 		},
// 	}
// 	if _, err := wctx.SessionService().AddAssistantMessageWithActions(ctx, sess.ID, message, actions); err != nil {
// 		logger.Warn("Failed to send error options message", "error", err)
// 	}

// 	return nil // 等待用户选择
// }

// // actCoreReturnToParent 成功返回父工作流
// func actCoreReturnToParent(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Core: Returning to parent workflow with success", "sessionID", sess.ID)

// 	// 获取排班结果
// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		logger.Warn("Failed to get shift scheduling context", "error", err)
// 	}

// 	// 构建返回结果（包含排班结果草稿）
// 	output := make(map[string]any)
// 	if shiftCtx != nil {
// 		// 班次信息
// 		if shiftCtx.Shift != nil {
// 			output["shift_id"] = shiftCtx.Shift.ID
// 			output["shift_name"] = shiftCtx.Shift.Name
// 		}
// 		// Todo 执行统计
// 		if shiftCtx.TodoPlan != nil {
// 			output["todo_count"] = len(shiftCtx.TodoPlan.TodoList)
// 		}
// 		// 排班结果草稿（核心输出）
// 		if shiftCtx.ResultDraft != nil {
// 			output["result_draft"] = shiftCtx.ResultDraft
// 		}
// 		// 跳过标记
// 		output["skipped"] = shiftCtx.Skipped
// 	}

// 	// 使用 Actor 的 ReturnToParent 方法返回
// 	actor, ok := wctx.(*engine.Actor)
// 	if !ok {
// 		logger.Warn("Context is not an Actor, cannot return to parent workflow")
// 		return nil // 非子工作流模式，静默返回
// 	}

// 	result := engine.NewSubWorkflowResult(output)
// 	return actor.ReturnToParent(ctx, result)
// }

// // actCoreReturnToParentWithError 错误返回父工作流
// func actCoreReturnToParentWithError(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Core: Returning to parent workflow with error", "sessionID", sess.ID)

// 	// 获取错误信息
// 	var coreErr error
// 	if errData, ok := sess.Data["core_error"]; ok {
// 		if errStr, ok := errData.(string); ok {
// 			coreErr = fmt.Errorf(errStr)
// 		}
// 	}
// 	if coreErr == nil {
// 		coreErr = fmt.Errorf("scheduling core failed")
// 	}

// 	// 使用 Actor 的 ReturnToParent 方法返回
// 	actor, ok := wctx.(*engine.Actor)
// 	if !ok {
// 		logger.Warn("Context is not an Actor, cannot return to parent workflow with error")
// 		return coreErr // 非子工作流模式，返回原始错误
// 	}

// 	result := engine.NewSubWorkflowError(coreErr)
// 	return actor.ReturnToParent(ctx, result)
// }

// // ============================================================
// // 辅助类型（用于 payload）
// // ============================================================

// // CoreStartPayload 核心子工作流启动时的 payload
// type CoreStartPayload struct {
// 	ShiftContext *d_model.ShiftSchedulingContext `json:"shift_context"`
// }

// // ============================================================
// // 错误恢复动作
// // ============================================================

// // actCoreRetry 重试当前班次排班
// func actCoreRetry(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Core: Retrying scheduling", "sessionID", sess.ID)

// 	// 获取排班上下文
// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return fmt.Errorf("failed to get shift context for retry: %w", err)
// 	}

// 	shiftName := "当前班次"
// 	if shiftCtx != nil && shiftCtx.Shift != nil {
// 		shiftName = shiftCtx.Shift.Name
// 	}

// 	// 清除之前的错误状态（设置为 nil）
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "core_error", nil); err != nil {
// 		logger.Warn("Failed to clear error from session", "error", err)
// 	}

// 	// 重置 Todo 执行进度
// 	if shiftCtx != nil {
// 		shiftCtx.CurrentTodoIndex = 0
// 		shiftCtx.TodoPlan = nil // 清空 Todo 计划，重新生成
// 		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
// 			logger.Warn("Failed to reset shift context", "error", err)
// 		}
// 	}

// 	// 发送重试提示消息
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
// 		fmt.Sprintf("🔄 正在重新为【%s】生成排班计划...", shiftName)); err != nil {
// 		logger.Warn("Failed to send retry message", "error", err)
// 	}

// 	// 触发重新开始
// 	return wctx.Send(ctx, CoreEventStart, nil)
// }

// // actCoreSkip 跳过当前班次
// func actCoreSkip(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Core: Skipping current shift", "sessionID", sess.ID)

// 	// 获取排班上下文
// 	shiftCtx, _ := GetShiftSchedulingContext(sess)
// 	shiftName := "当前班次"
// 	if shiftCtx != nil && shiftCtx.Shift != nil {
// 		shiftName = shiftCtx.Shift.Name
// 	}

// 	// 清除错误状态（设置为 nil）
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "core_error", nil); err != nil {
// 		logger.Warn("Failed to clear error from session", "error", err)
// 	}

// 	// 标记为已跳过
// 	if shiftCtx != nil {
// 		shiftCtx.Skipped = true
// 		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
// 			logger.Warn("Failed to mark shift as skipped", "error", err)
// 		}
// 	}

// 	// 发送跳过提示消息
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
// 		fmt.Sprintf("⏭️ 已跳过【%s】的排班，将继续处理下一个班次。", shiftName)); err != nil {
// 		logger.Warn("Failed to send skip message", "error", err)
// 	}

// 	// 直接调用返回逻辑（避免事件处理时状态不匹配）
// 	return actCoreReturnToParent(ctx, wctx, payload)
// }
