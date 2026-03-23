package adjust

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"jusha/mcp/pkg/workflow/engine"
// 	"jusha/mcp/pkg/workflow/session"

// 	d_model "jusha/agent/rostering/domain/model"
// 	d_service "jusha/agent/rostering/domain/service"

// 	. "jusha/agent/rostering/internal/workflow/common"
// 	"jusha/agent/rostering/internal/workflow/schedule/core"
// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // ============================================================
// // 子工作流编排 Actions
// // ============================================================

// // actScheduleAdjustSpawnCollectStaffCount 启动 CollectStaffCount 子工作流
// // 职责：收集调整后的班次人数需求
// func actScheduleAdjustSpawnCollectStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	// 获取调整上下文
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return fmt.Errorf("failed to get adjust context: %w", err)
// 	}

// 	// 构建 CollectStaffCount 子工作流的强类型输入
// 	input := &CollectStaffCountInput{
// 		StartDate: adjustCtx.StartDate,
// 		EndDate:   adjustCtx.EndDate,
// 		ShiftIDs:  []string{adjustCtx.SelectedShiftID},
// 		OrgID:     sess.OrgID,
// 	}

// 	// 获取 Actor 并启动子工作流
// 	actor, ok := wctx.(*engine.Actor)
// 	if !ok {
// 		return fmt.Errorf("context is not an Actor")
// 	}

// 	// 配置子工作流
// 	config := engine.SubWorkflowConfig{
// 		WorkflowName: WorkflowCollectStaffCount,
// 		Input:        input,
// 		OnComplete:   EventAdjustStaffCountCollected,
// 		OnError:      EventAdjustSubFailed,
// 		Timeout:      0, // 无超时，等待用户输入
// 	}

// 	logger.Info("Adjust: Spawning CollectStaffCount sub-workflow",
// 		"shiftID", adjustCtx.SelectedShiftID,
// 		"dateRange", fmt.Sprintf("%s to %s", adjustCtx.StartDate, adjustCtx.EndDate),
// 	)

// 	return actor.SpawnSubWorkflow(ctx, config)
// }

// // actScheduleAdjustOnStaffCountCollected 处理 CollectStaffCount 子工作流完成
// // 职责：接收人数需求，准备启动 Core 子工作流
// func actScheduleAdjustOnStaffCountCollected(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Adjust: CollectStaffCount sub-workflow completed", "sessionID", sess.ID)

// 	// 从 payload 解析子工作流结果
// 	result, ok := payload.(*engine.SubWorkflowResult)
// 	if !ok {
// 		return fmt.Errorf("invalid payload type for staff count collected event")
// 	}

// 	if !result.Success {
// 		return fmt.Errorf("collect staff count sub-workflow failed: %v", result.Error)
// 	}

// 	// 从结果中获取强类型人数需求
// 	var requirements []ShiftDailyRequirement

// 	if outputRaw, ok := result.Output["output"]; ok {
// 		if output, ok := outputRaw.(*CollectStaffCountOutput); ok {
// 			requirements = output.ShiftStaffRequirements
// 		}
// 	}

// 	if requirements == nil {
// 		logger.Error("Failed to extract staff requirements from sub-workflow result",
// 			"outputKeys", getOutputKeys(result.Output))
// 		return fmt.Errorf("staff requirements not found in sub-workflow result")
// 	}

// 	logger.Info("Extracted staff requirements from CollectStaffCount",
// 		"shiftsCount", len(requirements))

// 	// 保存到调整上下文
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	// 转换为内部map格式（确保向后兼容）
// 	if adjustCtx.ShiftStaffRequirements == nil {
// 		adjustCtx.ShiftStaffRequirements = make(map[string]map[string]int)
// 	}

// 	for _, shiftReq := range requirements {
// 		dateMap := make(map[string]int)
// 		for _, dailyReq := range shiftReq.DailyRequirements {
// 			// 仅添加有效日期（staffCount > 0已在子工作流中过滤）
// 			dateMap[dailyReq.Date] = dailyReq.StaffCount
// 		}
// 		if len(dateMap) > 0 {
// 			adjustCtx.ShiftStaffRequirements[shiftReq.ShiftID] = dateMap
// 		}
// 	}

// 	// 保存回 session
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 		return fmt.Errorf("failed to save adjust context: %w", err)
// 	}

// 	logger.Info("Staff count collected", "shiftID", adjustCtx.SelectedShiftID)

// 	// 发送进度消息
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
// 		"📊 人数配置完成！开始生成排班..."); err != nil {
// 		logger.Warn("Failed to send progress message", "error", err)
// 	}

// 	return nil
// }

// // actScheduleAdjustSpawnCoreWorkflow 启动 Core 子工作流
// // 职责：构建 ShiftSchedulingContext 并启动 Core 子工作流
// func actScheduleAdjustSpawnCoreWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	// 获取调整上下文
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	// 获取目标班次信息
// 	var targetShift *d_model.Shift
// 	for _, shift := range adjustCtx.AvailableShifts {
// 		if shift.ID == adjustCtx.SelectedShiftID {
// 			targetShift = shift
// 			break
// 		}
// 	}
// 	if targetShift == nil {
// 		return fmt.Errorf("shift not found: %s", adjustCtx.SelectedShiftID)
// 	}

// 	// 如果人员列表为空，需要加载
// 	if len(adjustCtx.StaffList) == 0 {
// 		logger.Info("StaffList is empty, loading from service")
// 		service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
// 		if !ok {
// 			return fmt.Errorf("rosteringService not found")
// 		}

// 		// 获取所有已选班次的人员（使用和InfoCollect相同的逻辑）
// 		allStaffMap := make(map[string]*d_model.Employee)
// 		shiftStaffMap := make(map[string][]*d_model.Employee)

// 		shiftsToLoad := adjustCtx.AvailableShifts
// 		if len(shiftsToLoad) == 0 {
// 			// 如果没有AvailableShifts，至少加载当前选中的班次
// 			if targetShift != nil {
// 				shiftsToLoad = []*d_model.Shift{targetShift}
// 			}
// 		}

// 		for _, shift := range shiftsToLoad {
// 			members, err := service.GetShiftGroupMembers(ctx, shift.ID)
// 			if err != nil {
// 				logger.Warn("Failed to get members for shift", "shiftID", shift.ID, "error", err)
// 				continue
// 			}
// 			shiftStaffMap[shift.ID] = members
// 			for _, m := range members {
// 				if _, exists := allStaffMap[m.ID]; !exists {
// 					allStaffMap[m.ID] = m
// 				}
// 			}
// 		}

// 		allStaff := make([]*d_model.Employee, 0, len(allStaffMap))
// 		for _, staff := range allStaffMap {
// 			allStaff = append(allStaff, staff)
// 		}

// 		logger.Info("Retrieved unique staff members", "totalCount", len(allStaff))

// 		adjustCtx.StaffList = allStaff
// 		if adjustCtx.ShiftStaffIDs == nil {
// 			adjustCtx.ShiftStaffIDs = make(map[string][]string)
// 		}
// 		// 保存班次-人员ID映射
// 		for shiftID, members := range shiftStaffMap {
// 			ids := make([]string, len(members))
// 			for i, m := range members {
// 				ids[i] = m.ID
// 			}
// 			adjustCtx.ShiftStaffIDs[shiftID] = ids
// 		}

// 		// 保存回session
// 		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 			return fmt.Errorf("failed to save adjust context with staff: %w", err)
// 		}

// 		logger.Info("Loaded staff list for adjust workflow", "staffCount", len(allStaff), "shiftsCount", len(shiftsToLoad))
// 	}

// 	// 构建 ShiftSchedulingContext
// 	shiftCtx := d_model.NewShiftSchedulingContext(
// 		targetShift,
// 		adjustCtx.StartDate,
// 		adjustCtx.EndDate,
// 		"adjust",
// 	)

// 	// 设置人员列表
// 	shiftCtx.StaffList = adjustCtx.StaffList

// 	// 设置规则
// 	shiftCtx.GlobalRules = adjustCtx.GlobalRules
// 	if rules, ok := adjustCtx.ShiftRules[adjustCtx.SelectedShiftID]; ok {
// 		shiftCtx.ShiftRules = rules
// 	}

// 	// 清空当前班次的排班数据，让Core子工作流在干净环境下重新排班
// 	if adjustCtx.CurrentDraft != nil && adjustCtx.CurrentDraft.Shifts != nil {
// 		if _, exists := adjustCtx.CurrentDraft.Shifts[adjustCtx.SelectedShiftID]; exists {
// 			// 保存原班次快照（用于差异对比）
// 			if adjustCtx.RegenerateOriginalShift == nil {
// 				adjustCtx.RegenerateOriginalShift = cloneShiftDraft(adjustCtx.CurrentDraft.Shifts[adjustCtx.SelectedShiftID])
// 			}
// 			// 清空当前班次数据
// 			delete(adjustCtx.CurrentDraft.Shifts, adjustCtx.SelectedShiftID)
// 			logger.Info("Cleared existing shift data for regeneration", "shiftID", adjustCtx.SelectedShiftID)
// 		}
// 	}

// 	// 设置人数需求（从 CollectStaffCount 获取）
// 	if dateReqs, ok := adjustCtx.ShiftStaffRequirements[adjustCtx.SelectedShiftID]; ok {
// 		shiftCtx.StaffRequirements = dateReqs
// 	} else {
// 		// 如果没有配置，使用默认值（每天1人）
// 		shiftCtx.StaffRequirements = buildDefaultStaffRequirements(
// 			adjustCtx.StartDate,
// 			adjustCtx.EndDate,
// 			1,
// 		)
// 	}

// 	// 构建其他班次的已排班标记（避免时段冲突）
// 	// 注意：当前班次的数据已被清空，只会包含其他班次的排班标记
// 	if adjustCtx.CurrentDraft != nil && adjustCtx.CurrentDraft.Shifts != nil {
// 		shiftCtx.ExistingScheduleMarks = core.BuildExistingScheduleMarks(
// 			adjustCtx.CurrentDraft,
// 			adjustCtx.SelectedShiftID,
// 			adjustCtx.AvailableShifts,
// 		)
// 	}

// 	// 保存 ShiftSchedulingContext 到 session（Core 子工作流会读取）
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
// 		return fmt.Errorf("failed to save shift context: %w", err)
// 	}

// 	// 保存 AdjustContext（因为已清空了当前班次数据并保存了原始快照）
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 		return fmt.Errorf("failed to save adjust context: %w", err)
// 	}

// 	// 获取 Actor 并启动子工作流
// 	actor, ok := wctx.(*engine.Actor)
// 	if !ok {
// 		return fmt.Errorf("context is not an Actor")
// 	}

// 	config := engine.SubWorkflowConfig{
// 		WorkflowName: WorkflowSchedulingCore,
// 		Input:        nil, // Core 从 session 读取 ShiftSchedulingContext
// 		OnComplete:   EventAdjustCoreCompleted,
// 		OnError:      EventAdjustSubFailed,
// 		Timeout:      10 * 60 * 1e9, // 10 分钟超时（纳秒）
// 	}

// 	logger.Info("Adjust: Spawning Core sub-workflow",
// 		"shiftName", targetShift.Name,
// 		"staffCount", len(shiftCtx.StaffList),
// 		"existingMarksCount", len(shiftCtx.ExistingScheduleMarks),
// 	)

// 	return actor.SpawnSubWorkflow(ctx, config)
// }

// // actScheduleAdjustOnCoreCompleted 处理 Core 子工作流完成
// // 职责：将 Core 结果合并到 CurrentDraft，准备展示给用户确认
// func actScheduleAdjustOnCoreCompleted(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Adjust: Core sub-workflow completed", "sessionID", sess.ID)

// 	// 从 payload 解析子工作流结果
// 	result, ok := payload.(*engine.SubWorkflowResult)
// 	if !ok {
// 		return fmt.Errorf("invalid payload type for core completed event")
// 	}

// 	if !result.Success {
// 		return fmt.Errorf("core sub-workflow failed: %v", result.Error)
// 	}

// 	// 获取调整上下文
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	// 从结果中提取 ShiftScheduleDraft
// 	var shiftDraft *d_model.ShiftScheduleDraft
// 	if draftRaw, ok := result.Output["result_draft"]; ok {
// 		shiftDraft, _ = draftRaw.(*d_model.ShiftScheduleDraft)
// 	}

// 	if shiftDraft == nil {
// 		return fmt.Errorf("no result draft from core sub-workflow")
// 	}

// 	// 将结果合并到 CurrentDraft
// 	if adjustCtx.CurrentDraft == nil {
// 		adjustCtx.CurrentDraft = &d_model.ScheduleDraft{
// 			StartDate:  adjustCtx.StartDate,
// 			EndDate:    adjustCtx.EndDate,
// 			Shifts:     make(map[string]*d_model.ShiftDraft),
// 			StaffStats: make(map[string]*d_model.StaffStats),
// 			Conflicts:  make([]*d_model.ScheduleConflict, 0),
// 		}
// 	}

// 	// 转换 ShiftScheduleDraft 为 ShiftDraft
// 	updatedShiftDraft := convertShiftScheduleDraftToShiftDraft(
// 		shiftDraft,
// 		adjustCtx.SelectedShiftID,
// 		adjustCtx.StaffList,
// 		adjustCtx.AvailableShifts,
// 	)

// 	// 更新班次草稿
// 	adjustCtx.CurrentDraft.Shifts[adjustCtx.SelectedShiftID] = updatedShiftDraft

// 	// 重新计算人员统计
// 	recalculateStaffStats(adjustCtx.CurrentDraft, adjustCtx.StaffList, adjustCtx.AvailableShifts)

// 	// 保存更新后的调整上下文
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 		return fmt.Errorf("failed to save adjust context: %w", err)
// 	}

// 	// 同时更新 ScheduleCreateContext.DraftSchedule（用于 ConfirmSave 子工作流）
// 	scheduleCtx := GetOrCreateScheduleContext(sess)
// 	scheduleCtx.DraftSchedule = adjustCtx.CurrentDraft
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
// 		return fmt.Errorf("failed to save schedule context with draft: %w", err)
// 	}

// 	logger.Info("Adjust: Core result merged to CurrentDraft",
// 		"shiftID", adjustCtx.SelectedShiftID,
// 		"scheduledDays", len(shiftDraft.Schedule),
// 	)

// 	// 不在这里发送消息，由 AfterAct (actScheduleAdjustShowRegenerateConfirm) 发送带按钮的确认界面
// 	return nil
// }

// // actScheduleAdjustShowRegenerateConfirm 显示重排确认界面
// // 职责：发送带有确认/重新排班/放弃按钮的确认消息
// func actScheduleAdjustShowRegenerateConfirm(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	// 获取调整上下文
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	// 获取班次名称
// 	shiftName := ""
// 	for _, shift := range adjustCtx.AvailableShifts {
// 		if shift.ID == adjustCtx.SelectedShiftID {
// 			shiftName = shift.Name
// 			break
// 		}
// 	}
// 	if shiftName == "" {
// 		shiftName = adjustCtx.SelectedShiftID
// 	}

// 	// 构建差异对比消息
// 	var diffMsg string
// 	if adjustCtx.RegenerateOriginalShift != nil && adjustCtx.CurrentDraft != nil {
// 		if currentShift, ok := adjustCtx.CurrentDraft.Shifts[adjustCtx.SelectedShiftID]; ok {
// 			diff := core.FormatScheduleDiff(adjustCtx.RegenerateOriginalShift, currentShift, adjustCtx.StaffList)
// 			if diff != nil && (diff.AddedCount > 0 || diff.RemovedCount > 0) {
// 				diffMsg = core.FormatScheduleDiffMessage(diff, shiftName)
// 			}
// 		}
// 	}

// 	// 构建确认消息
// 	var message string
// 	if diffMsg != "" {
// 		message = diffMsg + "\n\n请确认是否使用此排班结果？"
// 	} else {
// 		// 没有差异对比时，显示简单消息
// 		scheduledDays := 0
// 		if adjustCtx.CurrentDraft != nil {
// 			if shiftDraft, ok := adjustCtx.CurrentDraft.Shifts[adjustCtx.SelectedShiftID]; ok {
// 				scheduledDays = len(shiftDraft.Days)
// 			}
// 		}
// 		message = fmt.Sprintf("### ✅ 班次【%s】重排完成\n\n共排班 %d 天\n\n请确认是否使用此排班结果？", shiftName, scheduledDays)
// 	}

// 	// 构建操作按钮
// 	actions := []session.WorkflowAction{
// 		{
// 			ID:    "confirm_regenerate",
// 			Event: session.WorkflowEvent(EventAdjustPlanConfirmed),
// 			Label: "✅ 确认使用此排班",
// 			Type:  session.ActionTypeWorkflow,
// 			Style: session.ActionStylePrimary,
// 		},
// 		{
// 			ID:    "regenerate_again",
// 			Event: session.WorkflowEvent(EventAdjustRegenerateStart),
// 			Label: "🔄 重新排班",
// 			Type:  session.ActionTypeWorkflow,
// 			Style: session.ActionStyleSecondary,
// 		},
// 		{
// 			ID:    "abort_regenerate",
// 			Event: session.WorkflowEvent(EventAdjustRegenerateAborted),
// 			Label: "❌ 放弃重排",
// 			Type:  session.ActionTypeWorkflow,
// 			Style: session.ActionStyleSecondary,
// 		},
// 	}

// 	// 发送带按钮的消息
// 	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, actions); err != nil {
// 		return fmt.Errorf("failed to set workflow meta with actions: %w", err)
// 	}

// 	logger.Info("Adjust: Regenerate confirm UI sent",
// 		"shiftID", adjustCtx.SelectedShiftID,
// 		"shiftName", shiftName,
// 	)

// 	return nil
// }

// // actScheduleAdjustOnSubFailed 处理子工作流失败
// func actScheduleAdjustOnSubFailed(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Error("Adjust: Sub-workflow failed", "sessionID", sess.ID)

// 	var errMsg string
// 	if result, ok := payload.(*engine.SubWorkflowResult); ok && result.Error != nil {
// 		errMsg = result.Error.Error()
// 	} else {
// 		errMsg = "未知错误"
// 	}

// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
// 		fmt.Sprintf("❌ 重排失败：%s\n\n请重新选择操作。", errMsg)); err != nil {
// 		logger.Warn("Failed to send error message", "error", err)
// 	}

// 	return nil
// }

// // actScheduleAdjustOnRegenerateAborted 用户取消重排
// func actScheduleAdjustOnRegenerateAborted(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	logger.Info("Adjust: User cancelled regeneration", "sessionID", sess.ID)

// 	// 清除原班次快照
// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err == nil && adjustCtx.RegenerateOriginalShift != nil {
// 		adjustCtx.RegenerateOriginalShift = nil
// 		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 			logger.Warn("Failed to save adjust context", "error", err)
// 		}
// 	}

// 	// 发送消息
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "❌ 已取消重排班次"); err != nil {
// 		logger.Warn("Failed to send message", "error", err)
// 	}

// 	return nil
// }

// // ============================================================
// // 辅助函数
// // ============================================================

// // getOutputKeys 获取output map的所有key（用于调试）
// func getOutputKeys(output map[string]any) []string {
// 	keys := make([]string, 0, len(output))
// 	for k := range output {
// 		keys = append(keys, k)
// 	}
// 	return keys
// }

// // buildDefaultStaffRequirements 构建默认的人员需求配置
// func buildDefaultStaffRequirements(startDate, endDate string, dailyCount int) map[string]int {
// 	result := make(map[string]int)

// 	if dailyCount <= 0 {
// 		dailyCount = 1
// 	}

// 	start, err := time.Parse("2006-01-02", startDate)
// 	if err != nil {
// 		return result
// 	}
// 	end, err := time.Parse("2006-01-02", endDate)
// 	if err != nil {
// 		return result
// 	}

// 	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
// 		result[d.Format("2006-01-02")] = dailyCount
// 	}

// 	return result
// }

// // convertShiftScheduleDraftToShiftDraft 将 ShiftScheduleDraft 转换为 ShiftDraft
// func convertShiftScheduleDraftToShiftDraft(
// 	shiftScheduleDraft *d_model.ShiftScheduleDraft,
// 	shiftID string,
// 	staffList []*d_model.Employee,
// 	shifts []*d_model.Shift,
// ) *d_model.ShiftDraft {
// 	// 构建人员名称映射
// 	staffNameMap := make(map[string]string)
// 	for _, staff := range staffList {
// 		staffNameMap[staff.ID] = staff.Name
// 	}

// 	// 获取班次优先级
// 	var priority int
// 	for _, shift := range shifts {
// 		if shift.ID == shiftID {
// 			priority = shift.Priority
// 			break
// 		}
// 	}

// 	shiftDraft := &d_model.ShiftDraft{
// 		ShiftID:  shiftID,
// 		Priority: priority,
// 		Days:     make(map[string]*d_model.DayShift),
// 	}

// 	// 转换每天的排班
// 	for date, staffIDs := range shiftScheduleDraft.Schedule {
// 		// 获取人员姓名
// 		staffNames := make([]string, 0, len(staffIDs))
// 		for _, staffID := range staffIDs {
// 			if name, ok := staffNameMap[staffID]; ok {
// 				staffNames = append(staffNames, name)
// 			} else {
// 				staffNames = append(staffNames, staffID)
// 			}
// 		}

// 		shiftDraft.Days[date] = &d_model.DayShift{
// 			Staff:         staffNames,
// 			StaffIDs:      staffIDs,
// 			RequiredCount: len(staffIDs), // 使用实际人数作为需求
// 			ActualCount:   len(staffIDs),
// 		}
// 	}

// 	return shiftDraft
// }

// // recalculateStaffStats 重新计算人员统计信息
// func recalculateStaffStats(
// 	draft *d_model.ScheduleDraft,
// 	staffList []*d_model.Employee,
// 	shifts []*d_model.Shift,
// ) {
// 	// 构建人员和班次名称映射
// 	staffNameMap := make(map[string]string)
// 	for _, staff := range staffList {
// 		staffNameMap[staff.ID] = staff.Name
// 	}

// 	shiftNameMap := make(map[string]string)
// 	for _, shift := range shifts {
// 		shiftNameMap[shift.ID] = shift.Name
// 	}

// 	// 重置统计
// 	draft.StaffStats = make(map[string]*d_model.StaffStats)

// 	// 遍历所有班次和日期
// 	for shiftID, shiftDraft := range draft.Shifts {
// 		for _, dayShift := range shiftDraft.Days {
// 			for _, staffID := range dayShift.StaffIDs {
// 				if _, exists := draft.StaffStats[staffID]; !exists {
// 					draft.StaffStats[staffID] = &d_model.StaffStats{
// 						StaffID:   staffID,
// 						StaffName: staffNameMap[staffID],
// 						WorkDays:  0,
// 						Shifts:    make([]string, 0),
// 					}
// 				}
// 				stats := draft.StaffStats[staffID]
// 				stats.WorkDays++
// 				// 避免重复添加班次名称
// 				shiftName := shiftNameMap[shiftID]
// 				found := false
// 				for _, s := range stats.Shifts {
// 					if s == shiftName {
// 						found = true
// 						break
// 					}
// 				}
// 				if !found {
// 					stats.Shifts = append(stats.Shifts, shiftName)
// 				}
// 			}
// 		}
// 	}
// }
