// Package confirmsave 提供确认保存子工作流的 Action 实现
package confirmsave

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 阶段1: 预览草案
// ============================================================

// actConfirmSaveGeneratePreview 生成排班预览
// 只负责生成预览消息，按钮设置移到 AfterAct 执行以避免被引擎清除
func actConfirmSaveGeneratePreview(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Generating schedule preview", "sessionID", sess.ID)

	// 获取排班上下文
	scheduleCtx := GetOrCreateScheduleContext(sess)

	if scheduleCtx.DraftSchedule == nil {
		return fmt.Errorf("no draft schedule found")
	}

	// 生成预览统计信息
	shiftCount := len(scheduleCtx.DraftSchedule.Shifts)
	staffCount := len(scheduleCtx.DraftSchedule.StaffStats)

	// 计算总排班记录数
	totalRecords := 0
	for _, shiftDraft := range scheduleCtx.DraftSchedule.Shifts {
		for _, day := range shiftDraft.Days {
			totalRecords += len(day.StaffIDs)
		}
	}

	// 构建预览总结消息
	previewSummary := fmt.Sprintf("### 📋 排班预览总结\n\n"+
		"**排班周期**：%s 至 %s\n"+
		"**班次数量**：%d 个\n"+
		"**涉及人员**：%d 人\n"+
		"**排班记录**：%d 条\n",
		scheduleCtx.StartDate, scheduleCtx.EndDate,
		shiftCount, staffCount, totalRecords)

	// 构建预览数据（用于前端甘特图显示）
	// 将 DraftSchedule 序列化为 JSON，以便前端解析
	draftJSON, err := json.Marshal(scheduleCtx.DraftSchedule)
	if err != nil {
		logger.Warn("Failed to marshal draft schedule", "error", err)
		draftJSON = []byte("{}")
	}

	previewData := map[string]interface{}{
		"startDate":     scheduleCtx.StartDate,
		"endDate":       scheduleCtx.EndDate,
		"shiftCount":    shiftCount,
		"staffCount":    staffCount,
		"totalRecords":  totalRecords,
		"draftSchedule": string(draftJSON), // 将 DraftSchedule 作为 JSON 字符串传递
	}

	// 发送预览总结消息，附带预览按钮
	actions := []session.WorkflowAction{
		{
			ID:      "preview_full_schedule",
			Type:    session.ActionTypeQuery,
			Label:   "📊 预览完整排班",
			Event:   session.WorkflowEvent(""), // Query 类型不需要事件
			Style:   session.ActionStyleSuccess,
			Payload: previewData,
		},
	}

	if _, err := wctx.SessionService().AddAssistantMessageWithActions(ctx, sess.ID, previewSummary, actions); err != nil {
		logger.Warn("Failed to send preview message", "error", err)
	}

	// 触发预览就绪事件（确认按钮设置在 AfterAct 中执行）
	return wctx.Send(ctx, ConfirmSaveEventPreviewReady, nil)
}

// actConfirmSaveShowButtons 显示确认按钮（AfterAct）
// 在状态转换完成后设置按钮，避免被引擎在 Act 前清除
func actConfirmSaveShowButtons(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Showing confirm buttons", "sessionID", sess.ID)

	// 设置工作流元数据和按钮
	err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "请确认排班方案是否正确？", []session.WorkflowAction{
		{
			ID:    "confirm",
			Type:  session.ActionTypeWorkflow,
			Label: "确认保存",
			Event: ConfirmSaveEventConfirm,
			Style: session.ActionStylePrimary,
		},
		{
			ID:    "reject",
			Type:  session.ActionTypeWorkflow,
			Label: "重新生成",
			Event: ConfirmSaveEventReject,
			Style: session.ActionStyleSecondary,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: ConfirmSaveEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	return nil
}

// ============================================================
// 阶段2: 确认草案
// ============================================================

// actConfirmSaveConfirm 确认排班草案
func actConfirmSaveConfirm(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Confirming draft schedule", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)

	if scheduleCtx.DraftSchedule == nil {
		return fmt.Errorf("no draft schedule found")
	}

	// 标记草案为最终版本
	scheduleCtx.FinalSchedule = scheduleCtx.DraftSchedule
	logger.Info("Draft marked as final",
		"shifts", len(scheduleCtx.FinalSchedule.Shifts),
		"staff", len(scheduleCtx.FinalSchedule.StaffStats))

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 执行保存操作
	return saveScheduleToDatabase(ctx, wctx)
}

// actConfirmSaveReject 拒绝草案
func actConfirmSaveReject(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Rejecting draft, will notify parent", "sessionID", sess.ID)

	// 解析用户反馈
	var req struct {
		Feedback    string         `json:"feedback"`
		Adjustments map[string]any `json:"adjustments"`
	}
	if err := ParsePayload(payload, &req); err != nil {
		logger.Warn("Failed to parse feedback", "error", err)
	}

	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 添加用户反馈到AI总结
	if req.Feedback != "" {
		feedbackSummary := fmt.Sprintf("用户反馈：%s", req.Feedback)
		scheduleCtx.AISummaries = append(scheduleCtx.AISummaries, feedbackSummary)
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
			logger.Warn("Failed to save feedback", "error", err)
		}
	}

	// 发送消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "已收到您的反馈，将重新生成排班方案。"); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return nil
}

// ============================================================
// 阶段3: 保存
// ============================================================

// saveScheduleToDatabase 保存排班到数据库
func saveScheduleToDatabase(ctx context.Context, wctx engine.Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Saving schedule to database", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)
	if scheduleCtx.FinalSchedule == nil {
		return fmt.Errorf("no final schedule found")
	}

	// 获取 rosteringService
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 转换格式
	scheduleItems, err := ConvertDraftToScheduleRequests(scheduleCtx, sess.OrgID, logger)
	if err != nil {
		wctx.Send(ctx, ConfirmSaveEventSaveFailed, err.Error())
		return fmt.Errorf("failed to convert draft: %w", err)
	}

	logger.Info("Converted draft to schedule requests", "count", len(scheduleItems))

	// 批量保存
	batch := d_model.ScheduleBatch{
		Items:      scheduleItems,
		OnConflict: "upsert",
	}

	result, err := service.BatchUpsertSchedules(ctx, batch)
	if err != nil {
		logger.Error("Failed to save schedules", "error", err)
		// 保存错误信息
		sess.Data["save_error"] = err.Error()
		wctx.SessionService().SetData(ctx, sess.ID, "save_error", err.Error())
		return wctx.Send(ctx, ConfirmSaveEventSaveFailed, err.Error())
	}

	// 保存结果
	sess.Data["save_result"] = result
	wctx.SessionService().SetData(ctx, sess.ID, "save_result", map[string]any{
		"total":    result.Total,
		"upserted": result.Upserted,
		"failed":   result.Failed,
	})

	logger.Info("Schedule saved successfully",
		"total", result.Total,
		"upserted", result.Upserted,
		"failed", result.Failed)

	// 更新 Conversation.Meta（保存排班ID和状态）
	conversationSvc, ok := engine.GetService[d_service.IConversationService](wctx, "conversation")
	if ok && conversationSvc != nil {
		// 通过 SaveConversation 触发更新（它会自动调用 updateConversationMetaFromSession）
		// 同时更新 WorkflowContext 和 Meta
		if err := conversationSvc.SaveConversation(ctx, sess.ID, sess.Messages); err != nil {
			logger.Warn("Failed to update conversation meta after save", "error", err)
		} else {
			logger.Debug("Conversation meta updated after schedule save", "sessionID", sess.ID)
		}
	}

	return wctx.Send(ctx, ConfirmSaveEventSaveSuccess, nil)
}

// actConfirmSaveSaveSuccess 保存成功
func actConfirmSaveSaveSuccess(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取保存结果
	var upserted int
	if resultRaw, ok := sess.Data["save_result"]; ok {
		if result, ok := resultRaw.(map[string]any); ok {
			if u, ok := result["upserted"].(int); ok {
				upserted = u
			}
		}
	}

	// 更新 WorkflowMeta
	if _, err := wctx.SessionService().UpdateWorkflowMeta(ctx, sess.ID, func(meta *session.WorkflowMeta) error {
		meta.Description = fmt.Sprintf("✅ 排班保存成功：%d 条记录", upserted)
		meta.Actions = nil
		return nil
	}); err != nil {
		logger.Warn("Failed to update workflow meta", "error", err)
	}

	// 发送完成消息
	completeMsg := fmt.Sprintf("### ✅ 排班已完成\n\n成功保存 **%d** 条排班记录。\n\n您可以在排班日历中查看详情。", upserted)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, completeMsg); err != nil {
		logger.Warn("Failed to send completion message", "error", err)
	}

	// 更新 Conversation.Meta（标记为已完成状态）
	conversationSvc, ok := engine.GetService[d_service.IConversationService](wctx, "conversation")
	if ok && conversationSvc != nil {
		// 通过 SaveConversation 触发更新（它会自动调用 updateConversationMetaFromSession）
		// 同时更新 WorkflowContext 和 Meta（包括 workflowPhase）
		if err := conversationSvc.SaveConversation(ctx, sess.ID, sess.Messages); err != nil {
			logger.Warn("Failed to update conversation meta after completion", "error", err)
		} else {
			logger.Debug("Conversation meta updated after schedule completion", "sessionID", sess.ID)
		}
	}

	// 自动返回父工作流
	return wctx.Send(ctx, ConfirmSaveEventReturn, nil)
}

// actConfirmSaveSaveFailed 保存失败
func actConfirmSaveSaveFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	var errMsg string
	if s, ok := payload.(string); ok {
		errMsg = s
	} else if err, ok := payload.(error); ok {
		errMsg = err.Error()
	} else {
		errMsg = "未知错误"
	}

	logger.Error("Schedule save failed", "error", errMsg)

	// 设置重试选项
	err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
		fmt.Sprintf("❌ 保存失败：%s\n\n请选择操作：", errMsg),
		[]session.WorkflowAction{
			{
				ID:    "retry",
				Type:  session.ActionTypeWorkflow,
				Label: "重试",
				Event: ConfirmSaveEventRetry,
				Style: session.ActionStylePrimary,
			},
			{
				ID:    "cancel",
				Type:  session.ActionTypeWorkflow,
				Label: "取消",
				Event: ConfirmSaveEventCancel,
				Style: session.ActionStyleSecondary,
			},
		})
	if err != nil {
		logger.Warn("Failed to set workflow meta", "error", err)
	}

	return nil
}

// actConfirmSaveRetry 重试保存
func actConfirmSaveRetry(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	logger.Info("Retrying schedule save")
	return saveScheduleToDatabase(ctx, wctx)
}

// ============================================================
// 终态处理
// ============================================================

// actConfirmSaveReturnToParent 成功返回父工作流
func actConfirmSaveReturnToParent(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Returning to parent workflow with success", "sessionID", sess.ID)

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		return nil
	}

	// 获取保存结果
	var savedCount, failedCount int
	if resultRaw, ok := sess.Data["save_result"]; ok {
		if result, ok := resultRaw.(map[string]any); ok {
			if u, ok := result["upserted"].(int); ok {
				savedCount = u
			}
			if f, ok := result["failed"].(int); ok {
				failedCount = f
			}
		}
	}

	output := &ConfirmSaveOutput{
		Success:     true,
		SavedCount:  savedCount,
		FailedCount: failedCount,
	}

	result := engine.NewSubWorkflowResult(map[string]any{
		"output": output,
	})
	return actor.ReturnToParent(ctx, result)
}

// actConfirmSaveReturnToParentWithCancel 取消返回父工作流
func actConfirmSaveReturnToParentWithCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Returning to parent workflow with cancel", "sessionID", sess.ID)

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		return fmt.Errorf("user cancelled")
	}

	result := engine.NewSubWorkflowError(fmt.Errorf("user cancelled or rejected"))
	return actor.ReturnToParent(ctx, result)
}

// actConfirmSaveReturnToParentWithError 错误返回父工作流
func actConfirmSaveReturnToParentWithError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Returning to parent workflow with error", "sessionID", sess.ID)

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		return fmt.Errorf("save failed")
	}

	var errMsg string
	if s, ok := sess.Data["save_error"].(string); ok {
		errMsg = s
	} else {
		errMsg = "save failed"
	}

	result := engine.NewSubWorkflowError(fmt.Errorf(errMsg))
	return actor.ReturnToParent(ctx, result)
}

// actConfirmSaveCancel 处理取消
func actConfirmSaveCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Confirm-save cancelled", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "已取消保存操作。"); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	return nil
}
