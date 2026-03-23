package create

import (
	"context"
	"fmt"

	d_service "jusha/agent/rostering/domain/service"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 阶段 9: 排班审核（预览、确认保存、取消、重新排班）
// ============================================================

// startReviewPhase 开始审核阶段
func startReviewPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting review phase", "sessionID", sess.ID)

	// ============================================================
	// 1. 构建带有预览按钮的消息（前端通过消息级 Action 渲染预览按钮）
	// ============================================================

	reviewContent := "📊 **排班结果预览（V4 确定性引擎）**\n\n"
	reviewContent += fmt.Sprintf("- 排班时间：%s ~ %s\n", createCtx.StartDate, createCtx.EndDate)
	reviewContent += fmt.Sprintf("- 班次数量：%d\n", len(createCtx.SelectedShifts))
	reviewContent += fmt.Sprintf("- 参与人员：%d 人\n", len(createCtx.AllStaff))
	reviewContent += fmt.Sprintf("- LLM 调用次数：%d\n", createCtx.LLMCallCount)
	reviewContent += fmt.Sprintf("- 排班耗时：%d ms\n", createCtx.SchedulingDuration)

	if createCtx.ValidationResult != nil {
		reviewContent += fmt.Sprintf("\n**校验结果**：%s\n", createCtx.ValidationResult.Summary)
		if !createCtx.ValidationResult.IsValid {
			reviewContent += "⚠️ 校验发现一些问题，请点击「查看检查结果」按钮查看详情。\n"
		}
	}

	reviewContent += "\n请点击下方按钮查看排班详情，确认无误后点击「确认保存」。"

	// 构建消息级 Action 按钮（预览按钮，ActionTypeQuery 不触发状态转换）
	msgActions := make([]session.WorkflowAction, 0)

	// 构建完整排班预览数据（与 V3 相同的格式，前端通过 action.id 匹配渲染）
	fullSchedulePreview := createCtx.BuildFullSchedulePreview()

	// 「预览完整排班」按钮 — 前端通过 preview_full_schedule ID 渲染 MultiShiftScheduleDialog
	msgActions = append(msgActions, session.WorkflowAction{
		ID:      "preview_full_schedule",
		Type:    session.ActionTypeQuery,
		Label:   "📅 预览完整排班",
		Payload: fullSchedulePreview,
		Style:   session.ActionStylePrimary,
	})

	// 「查看排班详情」按钮 — 前端通过 view_task_schedule_detail ID 渲染排班详情
	msgActions = append(msgActions, session.WorkflowAction{
		ID:      "view_task_schedule_detail",
		Type:    session.ActionTypeQuery,
		Label:   "📊 查看排班详情",
		Payload: fullSchedulePreview,
		Style:   session.ActionStyleSuccess,
	})

	// 「查看检查结果」按钮 — 前端通过 view_validation_result ID 渲染 ValidationResultDialog
	if createCtx.ValidationResult != nil {
		validationPayload := buildValidationResultPayload(createCtx.ValidationResult)
		validationStyle := session.ActionStyleInfo
		if !createCtx.ValidationResult.IsValid {
			validationStyle = session.ActionStyleWarning
		}
		msgActions = append(msgActions, session.WorkflowAction{
			ID:      "view_validation_result",
			Type:    session.ActionTypeQuery,
			Label:   "🔍 查看检查结果",
			Payload: validationPayload,
			Style:   validationStyle,
		})
	}

	// 同时构建 draftSchedule 格式的预览数据（兼容前端旧格式处理逻辑）
	schedulePreviewData := map[string]any{
		"draftSchedule": createCtx.WorkingDraft,
		"startDate":     createCtx.StartDate,
		"endDate":       createCtx.EndDate,
	}
	_ = schedulePreviewData // 备用，fullSchedulePreview 已包含所有字段

	// 通过 AddMessage 发送带 Action 按钮的消息
	reviewMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: reviewContent,
		Actions: msgActions,
		Metadata: map[string]any{
			"scheduleDetail": fullSchedulePreview,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, reviewMsg); err != nil {
		logger.Warn("Failed to send review message with preview actions", "error", err)
	}

	// ============================================================
	// 2. 设置工作流级操作按钮（确认保存 / 重新排班 / 取消）
	// ============================================================

	statusMessage := ""
	if createCtx.ValidationResult != nil && !createCtx.ValidationResult.IsValid {
		statusMessage = "⚠️ 校验发现一些问题，请查看排班预览后再确认。"
	}

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认保存",
			Event: session.WorkflowEvent(CreateV4EventReviewConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "重新排班",
			Event: session.WorkflowEvent(CreateV4EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, statusMessage, workflowActions)
}

// actOnReviewConfirmed 审核确认
func actOnReviewConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Review confirmed, saving schedule", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取排班服务
	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rostering service not found")
	}

	// 保存排班
	scheduleID, err := saveSchedule(ctx, rosteringService, createCtx)
	if err != nil {
		logger.Error("Failed to save schedule", "error", err)
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
			fmt.Sprintf("❌ 保存失败：%v", err)); err != nil {
			logger.Warn("Failed to send error message", "error", err)
		}
		return fmt.Errorf("failed to save schedule: %w", err)
	}

	createCtx.SavedScheduleID = scheduleID

	// 发送成功消息
	successMessage := fmt.Sprintf("✅ **排班保存成功！**\n\n"+
		"- 排班ID：%s\n"+
		"- 时间范围：%s ~ %s\n"+
		"- 班次数量：%d\n"+
		"- LLM调用次数：%d\n"+
		"- 总耗时：%d ms\n",
		scheduleID,
		createCtx.StartDate,
		createCtx.EndDate,
		len(createCtx.SelectedShifts),
		createCtx.LLMCallCount,
		createCtx.SchedulingDuration)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, successMessage); err != nil {
		logger.Warn("Failed to send success message", "error", err)
	}

	// 清除工作流操作按钮
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// actUserCancel 用户取消
func actUserCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: User cancelled", "sessionID", sess.ID)

	// 发送取消消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"❌ 排班已取消。如需重新开始，请说「开始排班」。"); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	// 清除工作流操作按钮
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	return nil
}

// actOnUserModify 用户修改
func actOnUserModify(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: User requested modification", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送修改消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"🔄 正在重新生成排班..."); err != nil {
		logger.Warn("Failed to send modify message", "error", err)
	}

	// 清空当前排班结果
	createCtx.WorkingDraft = nil
	createCtx.ValidationResult = nil
	createCtx.LLMCallCount = 0

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 重新执行排班
	return wctx.Send(ctx, CreateV4EventSchedulingComplete, nil)
}
