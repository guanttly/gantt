package core

import (
	"context"
	"fmt"
	"strings"

	"jusha/agent/rostering/config"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/workflow/wsbridge"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	i_service "jusha/agent/rostering/internal/service"

	"jusha/agent/rostering/internal/workflow/schedule_v3/executor"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 部分成功处理 Actions
// ============================================================

// actOnPartialSuccess 处理部分成功场景
func actOnPartialSuccess(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Handling partial success", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	if taskCtx.TaskResult == nil || !taskCtx.TaskResult.PartiallySucceeded {
		return fmt.Errorf("task result is not partially succeeded")
	}

	// 提取失败班次列表
	failedShiftsList := make([]*d_model.ShiftFailureInfo, 0, len(taskCtx.TaskResult.FailedShifts))
	for _, failInfo := range taskCtx.TaskResult.FailedShifts {
		failedShiftsList = append(failedShiftsList, failInfo)
	}

	// 构建用户可读的文本消息
	var msgBuilder strings.Builder
	msgBuilder.WriteString("⚠️ **部分班次执行失败**\n\n")
	msgBuilder.WriteString(fmt.Sprintf("✅ 成功完成：%d 个班次\n", len(taskCtx.TaskResult.SuccessfulShifts)))
	msgBuilder.WriteString(fmt.Sprintf("❌ 失败班次：%d 个\n\n", len(taskCtx.TaskResult.FailedShifts)))

	if len(failedShiftsList) > 0 {
		msgBuilder.WriteString("**失败详情：**\n")
		for _, failInfo := range failedShiftsList {
			msgBuilder.WriteString(fmt.Sprintf("- **%s**：%s（已自动重试%d次）\n",
				failInfo.ShiftName,
				failInfo.FailureSummary,
				failInfo.AutoRetryCount))
		}
	}

	msgBuilder.WriteString("\n请选择操作：")

	// 发送消息（带结构化元数据）
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: msgBuilder.String(),
		Metadata: map[string]any{
			"type":         "partial_success",
			"successCount": len(taskCtx.TaskResult.SuccessfulShifts),
			"failedCount":  len(taskCtx.TaskResult.FailedShifts),
			"failedShifts": failedShiftsList,
		},
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		logger.Warn("Failed to send partial success message", "error", err)
	}

	// 构建标准的WorkflowAction按钮
	workflowActions := []session.WorkflowAction{
		{
			ID:    "retry_failed",
			Type:  session.ActionTypeWorkflow,
			Label: "重试失败班次",
			Event: session.WorkflowEvent(CoreV3EventRetryFailed),
			Style: session.ActionStylePrimary,
		},
		{
			ID:    "skip_failed",
			Type:  session.ActionTypeWorkflow,
			Label: "跳过失败班次，保存成功部分",
			Event: session.WorkflowEvent(CoreV3EventSkipFailed),
			Style: session.ActionStyleSuccess,
		},
		{
			ID:    "cancel_task",
			Type:  session.ActionTypeWorkflow,
			Label: "取消整个任务",
			Event: session.WorkflowEvent(CoreV3EventCancelTask),
			Style: session.ActionStyleDanger,
		},
	}

	// 设置工作流按钮
	if err := session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, workflowActions); err != nil {
		logger.Warn("Failed to set workflow actions", "error", err)
	}

	// 保存上下文
	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 转换到部分成功状态（需要在状态机中添加这个转换）
	// 注意：这里不能直接Send，需要由状态机定义的转换来处理
	// 暂时返回nil，等待用户输入触发相应事件
	logger.Info("CoreV3: Waiting for user action on partial success")

	return nil
}

// actRetryFailedShifts 重试失败的班次
func actRetryFailedShifts(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Retrying failed shifts", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	if taskCtx.TaskResult == nil || len(taskCtx.TaskResult.FailedShifts) == 0 {
		logger.Warn("No failed shifts to retry")
		return wctx.Send(ctx, CoreV3EventCompleted, nil)
	}

	// 提取失败的班次ID列表
	failedShiftIDs := make([]string, 0, len(taskCtx.TaskResult.FailedShifts))
	for shiftID := range taskCtx.TaskResult.FailedShifts {
		failedShiftIDs = append(failedShiftIDs, shiftID)
	}

	// 创建新的任务，只包含失败的班次
	retryTask := &d_model.ProgressiveTask{
		ID:           taskCtx.Task.ID + "_retry",
		Title:        "重试失败班次：" + taskCtx.Task.Title,
		Description:  taskCtx.Task.Description,
		TargetShifts: failedShiftIDs,
		TargetDates:  taskCtx.Task.TargetDates,
		TargetStaff:  taskCtx.Task.TargetStaff,
		RuleIDs:      taskCtx.Task.RuleIDs,
		Priority:     taskCtx.Task.Priority,
		Status:       "executing",
	}

	// 更新任务上下文
	taskCtx.Task = retryTask

	// 获取服务
	rosteringService := engine.MustGetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	ruleValidator := i_service.NewRuleLevelValidator(logger)

	var configurator config.IRosteringConfigurator
	if cfg, ok := engine.GetService[config.IRosteringConfigurator](wctx, "configurator"); ok {
		configurator = cfg
	}

	var aiFactory *ai.AIProviderFactory
	if factory, ok := engine.GetService[*ai.AIProviderFactory](wctx, engine.ServiceKeyAIFactory); ok {
		aiFactory = factory
	}

	// 创建任务执行器
	taskExecutorImpl := executor.NewProgressiveTaskExecutor(
		logger,
		schedulingAIService,
		ruleValidator,
		rosteringService,
		aiFactory,
		configurator,
	).(*executor.ProgressiveTaskExecutor)

	// 获取 Bridge 用于实时进度广播
	var bridge wsbridge.IBridge
	if b, ok := engine.GetService[wsbridge.IBridge](wctx, engine.ServiceKeyBridge); ok {
		bridge = b
	}

	// 设置进度回调，将进度信息发送给前端
	taskExecutorImpl.SetProgressCallback(func(info *executor.ShiftProgressInfo) {
		// 1. 广播结构化的 shift_progress 消息（用于进度条组件）
		if bridge != nil {
			progressData := map[string]any{
				"shiftId":        info.ShiftID,
				"shiftName":      info.ShiftName,
				"current":        info.Current,
				"total":          info.Total,
				"status":         info.Status,
				"message":        info.Message,
				"reasoning":      info.Reasoning,
				"previewData":    info.PreviewData,
				"currentDay":     info.CurrentDay,
				"totalDays":      info.TotalDays,
				"currentDate":    info.CurrentDate,
				"completedDates": info.CompletedDates,
				"draftPreview":   info.DraftPreview,
			}
			if err := bridge.BroadcastToSession(sess.ID, "shift_progress", progressData); err != nil {
				logger.Warn("Failed to broadcast shift progress", "error", err, "shiftName", info.ShiftName)
			}
		}

		// 2. 同时发送消息（用于聊天区域显示）
		// 验证失败消息使用助手样式，其他使用系统消息样式
		if info.Status == "day_validated" && strings.Contains(info.Message, "发现错误") {
			// 验证失败消息使用助手样式
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, info.Message); err != nil {
				logger.Warn("Failed to send validation error message", "error", err, "shiftName", info.ShiftName)
			}
		} else {
			if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, info.Message); err != nil {
				logger.Warn("Failed to send progress message", "error", err, "shiftName", info.ShiftName)
			}
		}

		// AI解释使用助手消息样式
		if info.Status == "success" && info.Reasoning != "" {
			reasoningMsg := fmt.Sprintf("💡 [%s] AI解释: %s", info.ShiftName, info.Reasoning)
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, reasoningMsg); err != nil {
				logger.Warn("Failed to send reasoning message", "error", err)
			}
		}
	})

	var taskExecutor executor.IProgressiveTaskExecutor = taskExecutorImpl

	// 更新任务上下文中的任务为重试任务
	taskCtx.Task = retryTask

	// 重新执行失败的班次（使用新接口）
	taskResult, err := taskExecutor.ExecuteTask(ctx, taskCtx)
	if err != nil {
		logger.Error("CoreV3: Retry failed shifts execution failed", "error", err)
		return wctx.Send(ctx, CoreV3EventFailed, nil)
	}

	// 合并结果
	// 将重试成功的班次从 FailedShifts 移除
	for shiftID := range taskResult.ShiftSchedules {
		delete(taskCtx.TaskResult.FailedShifts, shiftID)
		// 添加到成功列表（如果还不在）
		found := false
		for _, successID := range taskCtx.TaskResult.SuccessfulShifts {
			if successID == shiftID {
				found = true
				break
			}
		}
		if !found {
			taskCtx.TaskResult.SuccessfulShifts = append(taskCtx.TaskResult.SuccessfulShifts, shiftID)
		}
		// 更新 ShiftSchedules
		if taskCtx.TaskResult.ShiftSchedules == nil {
			taskCtx.TaskResult.ShiftSchedules = make(map[string]*d_model.ShiftScheduleDraft)
		}
		taskCtx.TaskResult.ShiftSchedules[shiftID] = taskResult.ShiftSchedules[shiftID]
	}

	// 添加新的失败班次
	if taskResult.FailedShifts != nil {
		for shiftID, failInfo := range taskResult.FailedShifts {
			taskCtx.TaskResult.FailedShifts[shiftID] = failInfo
		}
	}

	// 更新部分成功标志
	taskCtx.TaskResult.PartiallySucceeded = len(taskCtx.TaskResult.FailedShifts) > 0 && len(taskCtx.TaskResult.SuccessfulShifts) > 0

	// 保存上下文
	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 检查是否还有失败班次
	if len(taskCtx.TaskResult.FailedShifts) > 0 {
		// 仍有失败班次，再次进入部分成功处理
		logger.Info("CoreV3: Still have failed shifts after retry",
			"failedCount", len(taskCtx.TaskResult.FailedShifts))
		return actOnPartialSuccess(ctx, wctx, payload)
	}

	// 全部成功
	logger.Info("CoreV3: All failed shifts retried successfully")
	taskCtx.TaskResult.Success = true
	taskCtx.TaskResult.PartiallySucceeded = false

	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	return wctx.Send(ctx, CoreV3EventCompleted, nil)
}

// actSkipFailedShifts 跳过失败班次，保存成功部分
func actSkipFailedShifts(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Skipping failed shifts", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	// 记录失败班次信息到日志
	for shiftID, failInfo := range taskCtx.TaskResult.FailedShifts {
		logger.Warn("Skipping failed shift",
			"shiftID", shiftID,
			"shiftName", failInfo.ShiftName,
			"failureSummary", failInfo.FailureSummary)
	}

	// 清理失败班次信息（只保留成功的）
	taskCtx.TaskResult.FailedShifts = make(map[string]*d_model.ShiftFailureInfo)
	taskCtx.TaskResult.PartiallySucceeded = false
	taskCtx.TaskResult.Success = true // 标记为成功（部分成功也算成功）

	// 发送消息
	msg := fmt.Sprintf("✅ 已保存 %d 个成功班次的排班数据，跳过失败班次。",
		len(taskCtx.TaskResult.SuccessfulShifts))
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, msg); err != nil {
		logger.Warn("Failed to send skip message", "error", err)
	}

	// 保存上下文
	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 转到完成状态
	return wctx.Send(ctx, CoreV3EventCompleted, nil)
}

// actCancelPartialTask 取消部分成功任务
func actCancelPartialTask(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Cancelling partial task", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	// 清理所有结果数据
	taskCtx.TaskResult.ShiftSchedules = make(map[string]*d_model.ShiftScheduleDraft)
	taskCtx.TaskResult.SuccessfulShifts = []string{}
	taskCtx.TaskResult.FailedShifts = make(map[string]*d_model.ShiftFailureInfo)
	taskCtx.TaskResult.Success = false
	taskCtx.TaskResult.PartiallySucceeded = false
	taskCtx.TaskResult.Error = "用户取消了部分成功的任务"

	// 发送消息
	msg := "❌ 已取消任务，未保存任何排班数据。"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, msg); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	// 保存上下文
	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 转到失败状态
	return wctx.Send(ctx, CoreV3EventFailed, nil)
}
