package adjust

import (
	"context"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
)

// ============================================================
// 调整子工作流 Actions
// ============================================================

/********************** 1.初始化 ****************************/
func actAdjustV2Init(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("AdjustV2: Initializing adjustment context", "sessionID", sess.ID)

	// 从 session 加载调整上下文
	adjustCtx, err := LoadAdjustV2Context(sess)
	if err != nil {
		logger.Error("AdjustV2: Failed to load adjustment context", "error", err)
		return fmt.Errorf("failed to load adjustment context: %w", err)
	}

	// 验证必要数据
	if adjustCtx.ShiftID == "" {
		return fmt.Errorf("shift ID is required")
	}
	if adjustCtx.OriginalDraft == nil {
		return fmt.Errorf("original draft is required")
	}

	logger.Info("AdjustV2: Context loaded",
		"shiftID", adjustCtx.ShiftID,
		"shiftName", adjustCtx.ShiftName,
		"hasOriginalDraft", adjustCtx.OriginalDraft != nil,
	)

	// 保存上下文
	if err := SaveAdjustV2Context(ctx, wctx, adjustCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

func actAfterInit(ctx context.Context, wctx engine.Context, payload any) error {
	return nil
}

/********************** 2.直接应用调整（跳过意图分析） ****************************/
// actApplyAdjustmentDirect 直接应用调整，不再进行意图分析
func actApplyAdjustmentDirect(ctx context.Context, wctx engine.Context, payload any) error {
	return actApplyAdjustment(ctx, wctx, payload)
}

/********************** 3.修改模式 ****************************/
func actApplyAdjustment(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("AdjustV2: Applying adjustment with AI (modify mode)", "sessionID", sess.ID)

	// 加载调整上下文
	adjustCtx, err := LoadAdjustV2Context(sess)
	if err != nil {
		return fmt.Errorf("failed to load adjustment context: %w", err)
	}

	// 获取 AI 服务
	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		return fmt.Errorf("AI service not available")
	}

	// 临时需求已在父工作流的 actOnUserAdjustmentMessage 中提取
	// 这里直接使用 adjustCtx.TemporaryNeeds（如果为空，说明父工作流提取失败或没有临时需求）
	logger.Info("AdjustV2: Using temporary needs from parent workflow",
		"count", len(adjustCtx.TemporaryNeeds))

	// 准备班次信息
	shiftInfo := &d_model.ShiftInfo{
		ShiftID:   adjustCtx.ShiftID,
		ShiftName: adjustCtx.ShiftName,
		StartDate: adjustCtx.StartDate,
		EndDate:   adjustCtx.EndDate,
	}

	// 转换员工列表
	staffList := d_model.NewStaffInfoListFromEmployees(adjustCtx.StaffList)

	// 转换规则列表
	rules := d_model.NewRuleInfoListFromRules(adjustCtx.Rules)

	// 转换 ExistingScheduleMarks 为 bool 类型（AI 服务需要）
	// ExistingScheduleMarks 的结构是 map[staffID]map[date][]*ShiftMark
	// 需要转换为 map[date]map[staffID]bool
	existingMarks := make(map[string]map[string]bool)
	for staffID, staffDates := range adjustCtx.ExistingScheduleMarks {
		for date := range staffDates {
			if existingMarks[date] == nil {
				existingMarks[date] = make(map[string]bool)
			}
			existingMarks[date][staffID] = true
		}
	}

	// 直接调用 AI 服务调整排班
	adjustResult, err := aiService.AdjustShiftSchedule(
		ctx,
		adjustCtx.UserMessage,
		adjustCtx.OriginalDraft,
		shiftInfo,
		staffList,
		adjustCtx.AllStaffList,
		rules,
		adjustCtx.StaffRequirements,
		existingMarks,
		adjustCtx.FixedShiftAssignments, // 传递固定排班信息
	)
	if err != nil {
		logger.Error("AdjustV2: AI adjustment failed", "error", err)
		return fmt.Errorf("AI adjustment failed: %w", err)
	}

	// 保存结果（AI返回的排班已经包含了所有日期，因为我们在AI服务中已经合并了）
	adjustCtx.ResultDraft = adjustResult.Draft
	adjustCtx.AdjustSummary = adjustResult.Summary
	adjustCtx.AdjustChanges = adjustResult.Changes

	// 保存上下文
	if err := SaveAdjustV2Context(ctx, wctx, adjustCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	logger.Info("AdjustV2: Adjustment completed",
		"shiftID", adjustCtx.ShiftID,
		"shiftName", adjustCtx.ShiftName,
		"scheduleCount", len(adjustResult.Draft.Schedule),
		"summaryLength", len(adjustResult.Summary),
		"changesCount", len(adjustResult.Changes))

	// 直接返回父工作流（因为工作流已经进入 Completed 状态，AfterAct 不会再被调用）
	return actAdjustV2AfterComplete(ctx, wctx, nil)
}

/********************** 3.完成 ****************************/
func actAdjustV2AfterComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("AdjustV2: Adjustment sub-workflow completed", "sessionID", sess.ID)

	// 加载调整上下文
	adjustCtx, err := LoadAdjustV2Context(sess)
	if err != nil {
		logger.Error("AdjustV2: Failed to load adjustment context for return", "error", err)
		return returnToParentWithResult(ctx, wctx, nil, err)
	}

	// 检查 ResultDraft 是否存在
	if adjustCtx.ResultDraft == nil {
		logger.Error("AdjustV2: ResultDraft is nil, cannot return to parent",
			"shiftID", adjustCtx.ShiftID,
			"shiftName", adjustCtx.ShiftName)
		return returnToParentWithResult(ctx, wctx, nil, fmt.Errorf("result draft is nil, adjustment may have failed"))
	}

	// 记录结果信息
	logger.Info("AdjustV2: Preparing to return result to parent",
		"shiftID", adjustCtx.ShiftID,
		"shiftName", adjustCtx.ShiftName,
		"resultDraftScheduleCount", len(adjustCtx.ResultDraft.Schedule))

	// 构建返回结果
	output := make(map[string]any)
	// 将结果保存到 session，父工作流会读取
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "shift_scheduling_context", map[string]any{
		"result_draft": adjustCtx.ResultDraft,
	}); err != nil {
		logger.Warn("AdjustV2: Failed to save result to session", "error", err)
	}
	output["result_draft"] = adjustCtx.ResultDraft
	output["shift_id"] = adjustCtx.ShiftID
	output["shift_name"] = adjustCtx.ShiftName
	output["adjust_summary"] = adjustCtx.AdjustSummary
	output["adjust_changes"] = adjustCtx.AdjustChanges
	output["temporary_needs"] = adjustCtx.TemporaryNeeds

	// 返回父工作流
	logger.Info("AdjustV2: Returning to parent workflow with result",
		"shiftID", adjustCtx.ShiftID,
		"outputKeys", len(output))
	return returnToParentWithResult(ctx, wctx, output, nil)
}

// returnToParentWithResult 辅助函数：返回父工作流
func returnToParentWithResult(ctx context.Context, wctx engine.Context, output map[string]any, err error) error {
	logger := wctx.Logger()

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		if err != nil {
			return err
		}
		return nil
	}

	var result *engine.SubWorkflowResult
	if err != nil {
		result = engine.NewSubWorkflowError(err)
	} else {
		result = engine.NewSubWorkflowResult(output)
	}

	return actor.ReturnToParent(ctx, result)
}

/********************** 6.错误处理 ****************************/
func actAdjustV2HandleError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("AdjustV2: Handling error in adjustment sub-workflow", "sessionID", sess.ID)

	var errMsg string
	if err, ok := payload.(error); ok {
		errMsg = err.Error()
	} else if errStr, ok := payload.(string); ok {
		errMsg = errStr
	} else {
		errMsg = "unknown error"
	}

	logger.Error("AdjustV2: Error occurred", "error", errMsg)

	// 返回错误给父工作流
	return returnToParentWithResult(ctx, wctx, nil, fmt.Errorf(errMsg))
}
