// Package infocollect 提供信息收集子工作流的 Action 实现
package infocollect

import (
	"context"
	"fmt"
	"time"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 初始化和启动
// ============================================================

// actInfoCollectStart 启动信息收集子工作流
// 职责：从 Input 或 Intent 中提取参数，使用智能默认值，构建时间确认消息
func actInfoCollectStart(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	if sess == nil {
		logger.Error("session is nil in actInfoCollectStart")
		return fmt.Errorf("session not found")
	}

	logger.Info("Starting info-collect sub-workflow", "sessionID", sess.ID)

	// 1. 获取子工作流输入
	var input *InfoCollectInput
	if inputRaw, ok := sess.Data["info_collect_input"]; ok {
		input, _ = inputRaw.(*InfoCollectInput)
	}

	// 2. 从 Session.Data 中获取 Intent 信息（由基础设施层设置）
	var startDate, endDate, orgID string
	var hasExplicitDates bool

	// 尝试从输入参数获取预设日期
	if input != nil && input.PresetStartDate != "" && input.PresetEndDate != "" {
		startDate = input.PresetStartDate
		endDate = input.PresetEndDate
		hasExplicitDates = true
	}

	// 尝试从 payload 获取
	if !hasExplicitDates && payload != nil {
		var req struct {
			StartDate string `json:"startDate,omitempty"`
			EndDate   string `json:"endDate,omitempty"`
			OrgID     string `json:"orgId,omitempty"`
		}
		if err := ParsePayload(payload, &req); err == nil {
			if req.StartDate != "" && req.EndDate != "" {
				startDate = req.StartDate
				endDate = req.EndDate
				hasExplicitDates = true
			}
			if req.OrgID != "" {
				orgID = req.OrgID
			}
		}
	}

	// 尝试从 Intent.Entities 提取
	if !hasExplicitDates {
		if intentRaw, ok := sess.Data["intent"]; ok {
			if intent, ok := intentRaw.(*session.Intent); ok {
				if dateRange, ok := intent.Entities["dateRange"].(string); ok && dateRange != "" {
					if s, e, err := ParseDateRange(dateRange); err == nil {
						startDate = s
						endDate = e
						hasExplicitDates = true
					}
				}
				if startDate == "" {
					if sd, ok := intent.Entities["startDate"].(string); ok && sd != "" {
						startDate = sd
					}
				}
				if endDate == "" {
					if ed, ok := intent.Entities["endDate"].(string); ok && ed != "" {
						endDate = ed
					}
				}
				if orgID == "" {
					if oid, ok := intent.Entities["orgId"].(string); ok && oid != "" {
						orgID = oid
					}
				}
			}
		}
	}

	// 3. 使用默认值：下周一到周日
	if startDate == "" || endDate == "" {
		defaultStart, defaultEnd, err := GetDefaultNextWeekRange()
		if err != nil {
			return fmt.Errorf("failed to get default week range: %w", err)
		}
		startDate = defaultStart
		endDate = defaultEnd
		logger.Info("Using default next week range (Mon-Sun)", "start", startDate, "end", endDate)
	}

	// 4. 验证日期
	if err := ValidateDateRange(startDate, endDate); err != nil {
		return fmt.Errorf("invalid date range: %w", err)
	}

	// 5. 从 Session 获取 OrgID
	if orgID == "" {
		orgID = sess.OrgID
	}
	if orgID == "" {
		return fmt.Errorf("orgId is required but not provided")
	}

	// 6. 创建排班上下文
	scheduleCtx := GetOrCreateScheduleContext(sess)
	scheduleCtx.StartDate = startDate
	scheduleCtx.EndDate = endDate

	// 7. 保存到 Session.Data
	if s, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	} else {
		sess = s
	}

	// 8. 检查是否需要跳过周期确认
	if input != nil && input.ShouldSkipPhase("period") {
		// 跳过周期确认，直接进入下一阶段
		logger.Info("Info-collect started - skipping period confirmation",
			"startDate", startDate,
			"endDate", endDate)
		return nil
	}

	// 9. 构建时间确认消息
	dateDisplay := FormatDateRangeForDisplay(startDate, endDate)
	confirmMessage := fmt.Sprintf("好的，我将为您安排 %s 的排班。\n\n请确认排班周期是否正确？", dateDisplay)

	// 10. 设置工作流元数据
	err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, confirmMessage, []session.WorkflowAction{
		{
			ID:    "confirm_period",
			Type:  session.ActionTypeWorkflow,
			Label: "确认周期",
			Event: InfoCollectEventPeriodConfirmed,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"startDate": startDate,
				"endDate":   endDate,
			},
		},
		{
			ID:    "modify_period",
			Type:  session.ActionTypeWorkflow,
			Label: "修改周期",
			Event: InfoCollectEventPeriodModified,
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:         "startDate",
					Label:        "开始日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					Placeholder:  "请选择开始日期",
					DefaultValue: startDate,
				},
				{
					Name:         "endDate",
					Label:        "结束日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					Placeholder:  "请选择结束日期",
					DefaultValue: endDate,
				},
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: InfoCollectEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	logger.Info("Info-collect started - period confirmation",
		"startDate", startDate,
		"endDate", endDate)

	return nil
}

// actInfoCollectAfterStart 启动后根据 skip_phases 决定下一步
func actInfoCollectAfterStart(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取输入参数
	var input *InfoCollectInput
	if inputRaw, ok := sess.Data["info_collect_input"]; ok {
		input, _ = inputRaw.(*InfoCollectInput)
	}

	// 如果没有输入或不需要跳过，走正常流程
	if input == nil || len(input.SkipPhases) == 0 {
		return nil
	}

	// 根据 skip_phases 决定跳过哪些阶段
	if input.ShouldSkipPhase("period") && input.ShouldSkipPhase("shifts") {
		// 跳过周期和班次，直接到人数确认
		logger.Info("Skipping period and shifts, going to staff count confirmation")
		return wctx.Send(ctx, InfoCollectEventSkipShifts, nil)
	} else if input.ShouldSkipPhase("period") {
		// 只跳过周期，进入班次选择
		logger.Info("Skipping period, going to shift selection")
		return wctx.Send(ctx, InfoCollectEventSkipPeriod, nil)
	}

	// 不跳过，走正常流程
	return nil
}

// ============================================================
// 阶段1: 确认排班周期
// ============================================================

// actInfoCollectConfirmPeriod 确认排班周期
func actInfoCollectConfirmPeriod(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Confirming schedule period", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 从 payload 中获取日期
	var req struct {
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	}
	if err := ParsePayload(payload, &req); err == nil {
		if req.StartDate != "" && req.EndDate != "" {
			if err := ValidateDateRange(req.StartDate, req.EndDate); err != nil {
				return fmt.Errorf("date validation failed: %w", err)
			}
			scheduleCtx.StartDate = req.StartDate
			scheduleCtx.EndDate = req.EndDate
			if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
				return fmt.Errorf("failed to save context: %w", err)
			}
		}
	}

	if scheduleCtx.StartDate == "" || scheduleCtx.EndDate == "" {
		return fmt.Errorf("schedule period not initialized")
	}

	dateDisplay := FormatDateRangeForDisplay(scheduleCtx.StartDate, scheduleCtx.EndDate)
	logger.Info("Period confirmed", "period", dateDisplay)

	return nil
}

// actInfoCollectModifyPeriod 修改排班周期
func actInfoCollectModifyPeriod(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	scheduleCtx := GetOrCreateScheduleContext(sess)

	var req struct {
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	}
	if err := ParsePayload(payload, &req); err == nil {
		if req.StartDate != "" && req.EndDate != "" {
			if err := ValidateDateRange(req.StartDate, req.EndDate); err != nil {
				return fmt.Errorf("date validation failed: %w", err)
			}
			scheduleCtx.StartDate = req.StartDate
			scheduleCtx.EndDate = req.EndDate
			if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
				return fmt.Errorf("failed to save context: %w", err)
			}
			logger.Info("Modified schedule period", "startDate", req.StartDate, "endDate", req.EndDate)
		}
	}

	// 重新显示确认界面
	dateDisplay := FormatDateRangeForDisplay(scheduleCtx.StartDate, scheduleCtx.EndDate)
	confirmMessage := fmt.Sprintf("好的，我将为您安排 %s 的排班。\n\n请确认排班周期是否正确？", dateDisplay)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, confirmMessage, []session.WorkflowAction{
		{
			ID:    "confirm_period",
			Type:  session.ActionTypeWorkflow,
			Label: "确认周期",
			Event: InfoCollectEventPeriodConfirmed,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"startDate": scheduleCtx.StartDate,
				"endDate":   scheduleCtx.EndDate,
			},
		},
		{
			ID:    "modify_period",
			Type:  session.ActionTypeWorkflow,
			Label: "修改周期",
			Event: InfoCollectEventPeriodModified,
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:         "startDate",
					Label:        "开始日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					DefaultValue: scheduleCtx.StartDate,
				},
				{
					Name:         "endDate",
					Label:        "结束日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					DefaultValue: scheduleCtx.EndDate,
				},
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: InfoCollectEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// actInfoCollectTriggerQueryShifts 触发查询班次
func actInfoCollectTriggerQueryShifts(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, InfoCollectEventShiftsQueried, nil)
}

// ============================================================
// 阶段2: 查询可用班次
// ============================================================

// actInfoCollectQueryShifts 查询可用班次
func actInfoCollectQueryShifts(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Querying available shifts", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)
	if scheduleCtx.StartDate == "" || scheduleCtx.EndDate == "" {
		return fmt.Errorf("schedule period not initialized")
	}

	// 获取 rosteringService
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 查询可用班次列表
	shifts, err := service.ListShifts(ctx, sess.OrgID, "")
	if err != nil {
		return fmt.Errorf("failed to list shifts: %w", err)
	}

	scheduleCtx.AvailableShifts = shifts
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	dateDisplay := FormatDateRangeForDisplay(scheduleCtx.StartDate, scheduleCtx.EndDate)

	if len(shifts) == 0 {
		return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
			"❌ 当前没有可用的班次，无法进行排班。\n\n请先在系统中创建班次后再试。",
			[]session.WorkflowAction{
				{
					ID:    "close",
					Type:  session.ActionTypeWorkflow,
					Label: "关闭",
					Event: InfoCollectEventCancel,
					Style: session.ActionStyleSecondary,
				},
			})
	}

	// 构建班次选项列表
	shiftOptions := make([]session.FieldOption, 0, len(shifts))
	for _, shift := range shifts {
		shiftType := shift.Type
		if shiftType == "" {
			shiftType = "常规班次"
		}
		description := fmt.Sprintf("%s-%s", shift.StartTime, shift.EndTime)
		if shift.IsOvernight {
			description += " (跨夜)"
		}
		shiftOptions = append(shiftOptions, session.FieldOption{
			Label:       shift.Name,
			Value:       shift.ID,
			Description: description,
			Extra: map[string]any{
				"type":      shiftType,
				"startTime": shift.StartTime,
				"endTime":   shift.EndTime,
				"color":     shift.Color,
				"duration":  shift.Duration,
			},
		})
	}

	shiftsInfo := fmt.Sprintf("系统中共有 %d 个班次可供选择，排班周期：%s\n\n点击「选择班次」按钮可以查看并选择需要使用的班次。", len(shifts), dateDisplay)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, shiftsInfo, []session.WorkflowAction{
		{
			ID:    "select_shifts",
			Type:  session.ActionTypeWorkflow,
			Label: "选择班次",
			Event: InfoCollectEventShiftsConfirmed,
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:     "shiftIds",
					Label:    "选择需要排班的班次",
					Type:     session.FieldTypeMultiSelect,
					Required: true,
					Options:  shiftOptions,
					DefaultValue: func() []string {
						ids := make([]string, 0, len(shifts))
						for _, s := range shifts {
							if s.IsActive {
								ids = append(ids, s.ID)
							}
						}
						return ids
					}(),
				},
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: InfoCollectEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// ============================================================
// 阶段3: 确认排班班次
// ============================================================

// actInfoCollectConfirmShifts 确认班次列表
func actInfoCollectConfirmShifts(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Confirming shifts", "sessionID", sess.ID)

	var req struct {
		ShiftIDs []string `json:"shiftIds"`
	}
	if err := ParsePayload(payload, &req); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	scheduleCtx := GetOrCreateScheduleContext(sess)
	if len(scheduleCtx.AvailableShifts) == 0 {
		return fmt.Errorf("no available shifts found")
	}

	// 过滤出用户选择的班次
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range scheduleCtx.AvailableShifts {
		shiftMap[shift.ID] = shift
	}

	selectedShifts := make([]*d_model.Shift, 0, len(req.ShiftIDs))
	for _, id := range req.ShiftIDs {
		if shift, ok := shiftMap[id]; ok {
			selectedShifts = append(selectedShifts, shift)
		}
	}

	SortShiftsByPriority(selectedShifts)
	scheduleCtx.SelectedShifts = selectedShifts

	// 获取周人数配置
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	weeklyConfigs := make(map[string]map[int]int)
	for _, shift := range selectedShifts {
		weeklyConfig, err := service.GetWeeklyStaffConfig(ctx, sess.OrgID, shift.ID)
		if err != nil {
			logger.Warn("Failed to get weekly staff config", "shiftID", shift.ID, "error", err)
			continue
		}
		if weeklyConfig != nil {
			dayConfig := make(map[int]int)
			for _, dc := range weeklyConfig.WeeklyConfig {
				dayConfig[dc.Weekday] = dc.StaffCount
			}
			weeklyConfigs[shift.ID] = dayConfig
		}
	}

	// 初始化人数需求
	if err := InitializeStaffRequirementsWithWeeklyConfig(scheduleCtx, weeklyConfigs); err != nil {
		return fmt.Errorf("failed to initialize staff requirements: %w", err)
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 构建人数配置界面
	dateDisplay := FormatDateRangeForDisplay(scheduleCtx.StartDate, scheduleCtx.EndDate)
	var shiftNames []string
	for _, shift := range selectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	infoMessage := fmt.Sprintf("已选择 %d 个班次：%s\n\n排班周期：%s\n\n请配置每个班次每天所需的人数。如果使用默认值，可以直接点击「确认人数」。",
		len(selectedShifts), FormatList(shiftNames), dateDisplay)

	staffCountFields := buildStaffCountFields(scheduleCtx)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, infoMessage, []session.WorkflowAction{
		{
			ID:     "confirm_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "确认人数",
			Event:  InfoCollectEventStaffCountConfirmed,
			Style:  session.ActionStylePrimary,
			Fields: staffCountFields,
		},
		{
			ID:     "modify_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "调整人数",
			Event:  InfoCollectEventStaffCountModified,
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: InfoCollectEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// actInfoCollectModifyShifts 修改班次列表
func actInfoCollectModifyShifts(ctx context.Context, wctx engine.Context, payload any) error {
	// 与 ConfirmShifts 逻辑相同，只是停留在同一状态
	return actInfoCollectConfirmShifts(ctx, wctx, payload)
}

// ============================================================
// 阶段4: 确认班次人数
// ============================================================

// actInfoCollectConfirmStaffCount 确认班次人数
func actInfoCollectConfirmStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Confirming staff count", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 解析人数配置
	if payload != nil {
		var payloadMap map[string]any
		if err := ParsePayload(payload, &payloadMap); err == nil {
			if err := parseStaffCountPayload(payloadMap, scheduleCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			}
		}
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	logger.Info("Staff count confirmed", "shiftsCount", len(scheduleCtx.SelectedShifts))
	return nil
}

// actInfoCollectModifyStaffCount 修改班次人数
func actInfoCollectModifyStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 解析并更新人数配置
	if payload != nil {
		var payloadMap map[string]any
		if err := ParsePayload(payload, &payloadMap); err == nil {
			if err := parseStaffCountPayload(payloadMap, scheduleCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			}
		}
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 重新显示配置界面
	dateDisplay := FormatDateRangeForDisplay(scheduleCtx.StartDate, scheduleCtx.EndDate)
	var shiftNames []string
	for _, shift := range scheduleCtx.SelectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	infoMessage := fmt.Sprintf("已更新人数配置。\n\n班次：%s\n排班周期：%s\n\n可以继续调整或点击「确认人数」进入下一步。",
		FormatList(shiftNames), dateDisplay)

	staffCountFields := buildStaffCountFields(scheduleCtx)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, infoMessage, []session.WorkflowAction{
		{
			ID:     "confirm_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "确认人数",
			Event:  InfoCollectEventStaffCountConfirmed,
			Style:  session.ActionStylePrimary,
			Fields: staffCountFields,
		},
		{
			ID:     "modify_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "调整人数",
			Event:  InfoCollectEventStaffCountModified,
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: InfoCollectEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// actInfoCollectTriggerRetrieveStaff 触发检索人员
func actInfoCollectTriggerRetrieveStaff(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, InfoCollectEventStaffRetrieved, nil)
}

// ============================================================
// 阶段5: 检索可用人员
// ============================================================

// actInfoCollectRetrieveStaff 检索可用人员
func actInfoCollectRetrieveStaff(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Retrieving available staff", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)
	if len(scheduleCtx.SelectedShifts) == 0 {
		return fmt.Errorf("no shifts selected")
	}

	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 获取每个班次的人员
	allStaffMap := make(map[string]*d_model.Employee)
	shiftStaffMap := make(map[string][]*d_model.Employee)

	for _, shift := range scheduleCtx.SelectedShifts {
		members, err := service.GetShiftGroupMembers(ctx, shift.ID)
		if err != nil {
			logger.Warn("Failed to get members for shift", "shiftID", shift.ID, "error", err)
			continue
		}
		shiftStaffMap[shift.ID] = members
		for _, m := range members {
			if _, exists := allStaffMap[m.ID]; !exists {
				allStaffMap[m.ID] = m
			}
		}
	}

	allStaff := make([]*d_model.Employee, 0, len(allStaffMap))
	for _, staff := range allStaffMap {
		allStaff = append(allStaff, staff)
	}

	logger.Info("Retrieved unique staff members", "totalCount", len(allStaff))

	// 批量查询人员请假记录（优化：一次查询获取所有员工的请假记录）
	staffLeaveMap, err := service.BatchGetLeaveRecords(ctx, sess.OrgID, scheduleCtx.StartDate, scheduleCtx.EndDate)
	if err != nil {
		logger.Warn("Failed to batch get leave records", "error", err)
		// 如果批量查询失败，使用空 map，不影响后续流程
		staffLeaveMap = make(map[string][]*d_model.LeaveRecord)
	} else {
		// 只保留有请假记录的员工（过滤空记录）
		for staffID, leaves := range staffLeaveMap {
			if len(leaves) == 0 {
				delete(staffLeaveMap, staffID)
			}
		}
		logger.Info("Batch retrieved leave records", "staffWithLeave", len(staffLeaveMap))
	}

	scheduleCtx.StaffList = allStaff
	scheduleCtx.StaffLeaves = staffLeaveMap

	// 保存班次-人员ID映射
	scheduleCtx.ShiftStaffIDs = make(map[string][]string)
	for shiftID, members := range shiftStaffMap {
		ids := make([]string, len(members))
		for i, m := range members {
			ids[i] = m.ID
		}
		scheduleCtx.ShiftStaffIDs[shiftID] = ids
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	logger.Info("Staff retrieval completed",
		"totalStaff", len(allStaff),
		"staffWithLeave", len(staffLeaveMap))

	return nil
}

// actInfoCollectTriggerRetrieveRules 触发检索规则
func actInfoCollectTriggerRetrieveRules(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, InfoCollectEventRulesRetrieved, nil)
}

// ============================================================
// 阶段6: 检索排班规则
// ============================================================

// actInfoCollectRetrieveRules 检索排班规则
func actInfoCollectRetrieveRules(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Retrieving scheduling rules", "sessionID", sess.ID)

	scheduleCtx := GetOrCreateScheduleContext(sess)

	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 查询全局规则
	globalRules, err := service.ListRules(ctx, d_model.ListRulesRequest{
		OrgID:      sess.OrgID,
		ApplyScope: "global",
		IsActive:   BoolPtr(true),
		Page:       1,
		PageSize:   100,
	})
	if err != nil {
		logger.Warn("Failed to query global rules", "error", err)
		globalRules = []*d_model.Rule{}
	}
	scheduleCtx.GlobalRules = globalRules

	// 批量查询班次规则
	if scheduleCtx.ShiftRules == nil {
		scheduleCtx.ShiftRules = make(map[string][]*d_model.Rule)
	}
	if len(scheduleCtx.SelectedShifts) > 0 {
		shiftIDs := make([]string, 0, len(scheduleCtx.SelectedShifts))
		for _, shift := range scheduleCtx.SelectedShifts {
			shiftIDs = append(shiftIDs, shift.ID)
		}
		shiftRulesMap, err := service.GetRulesForShifts(ctx, sess.OrgID, shiftIDs)
		if err != nil {
			logger.Warn("Failed to batch query shift rules", "error", err)
			// 初始化空规则映射
			for _, shiftID := range shiftIDs {
				scheduleCtx.ShiftRules[shiftID] = []*d_model.Rule{}
			}
		} else {
			scheduleCtx.ShiftRules = shiftRulesMap
		}
	}

	// 批量查询分组规则
	if scheduleCtx.GroupRules == nil {
		scheduleCtx.GroupRules = make(map[string][]*d_model.Rule)
	}
	groupIDsMap := make(map[string]bool)
	for _, staff := range scheduleCtx.StaffList {
		if staff.Groups != nil {
			for _, group := range staff.Groups {
				if group.ID != "" {
					groupIDsMap[group.ID] = true
				}
			}
		}
	}
	if len(groupIDsMap) > 0 {
		groupIDs := make([]string, 0, len(groupIDsMap))
		for groupID := range groupIDsMap {
			groupIDs = append(groupIDs, groupID)
		}
		groupRulesMap, err := service.GetRulesForGroups(ctx, sess.OrgID, groupIDs)
		if err != nil {
			logger.Warn("Failed to batch query group rules", "error", err)
			// 初始化空规则映射
			for _, groupID := range groupIDs {
				scheduleCtx.GroupRules[groupID] = []*d_model.Rule{}
			}
		} else {
			scheduleCtx.GroupRules = groupRulesMap
		}
	}

	// 批量查询人员规则
	if scheduleCtx.EmployeeRules == nil {
		scheduleCtx.EmployeeRules = make(map[string][]*d_model.Rule)
	}
	if len(scheduleCtx.StaffList) > 0 {
		employeeIDs := make([]string, 0, len(scheduleCtx.StaffList))
		for _, staff := range scheduleCtx.StaffList {
			employeeIDs = append(employeeIDs, staff.ID)
		}
		employeeRulesMap, err := service.GetRulesForEmployees(ctx, sess.OrgID, employeeIDs)
		if err != nil {
			logger.Warn("Failed to batch query employee rules", "error", err)
			// 初始化空规则映射
			for _, employeeID := range employeeIDs {
				scheduleCtx.EmployeeRules[employeeID] = []*d_model.Rule{}
			}
		} else {
			scheduleCtx.EmployeeRules = employeeRulesMap
		}
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 统计规则数量
	shiftRuleCount := 0
	for _, rules := range scheduleCtx.ShiftRules {
		shiftRuleCount += len(rules)
	}
	groupRuleCount := 0
	for _, rules := range scheduleCtx.GroupRules {
		groupRuleCount += len(rules)
	}
	employeeRuleCount := 0
	for _, rules := range scheduleCtx.EmployeeRules {
		employeeRuleCount += len(rules)
	}

	logger.Info("Rules retrieval completed",
		"globalRules", len(globalRules),
		"shiftRules", shiftRuleCount,
		"groupRules", groupRuleCount,
		"employeeRules", employeeRuleCount)

	return nil
}

// actInfoCollectTriggerComplete 触发完成
func actInfoCollectTriggerComplete(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, InfoCollectEventReturn, nil)
}

// ============================================================
// 终态处理
// ============================================================

// actInfoCollectReturnToParent 成功返回父工作流
func actInfoCollectReturnToParent(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Returning to parent workflow with success", "sessionID", sess.ID)

	// 获取收集的信息
	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 构建输出
	output := &InfoCollectOutput{
		StartDate:              scheduleCtx.StartDate,
		EndDate:                scheduleCtx.EndDate,
		SelectedShiftIDs:       make([]string, len(scheduleCtx.SelectedShifts)),
		ShiftStaffRequirements: scheduleCtx.ShiftStaffRequirements,
		StaffIDs:               make([]string, len(scheduleCtx.StaffList)),
		StaffLeaveDates:        make(map[string][]string),
		GlobalRuleCount:        len(scheduleCtx.GlobalRules),
		ShiftRuleCount:         make(map[string]int),
		GroupRuleCount:         make(map[string]int),
		EmployeeRuleCount:      make(map[string]int),
	}

	for i, shift := range scheduleCtx.SelectedShifts {
		output.SelectedShiftIDs[i] = shift.ID
	}
	for i, staff := range scheduleCtx.StaffList {
		output.StaffIDs[i] = staff.ID
	}
	for staffID, leaves := range scheduleCtx.StaffLeaves {
		dates := make([]string, len(leaves))
		for i, leave := range leaves {
			dates[i] = leave.StartDate // 使用 StartDate
		}
		output.StaffLeaveDates[staffID] = dates
	}
	for shiftID, rules := range scheduleCtx.ShiftRules {
		output.ShiftRuleCount[shiftID] = len(rules)
	}
	for groupID, rules := range scheduleCtx.GroupRules {
		output.GroupRuleCount[groupID] = len(rules)
	}
	for employeeID, rules := range scheduleCtx.EmployeeRules {
		output.EmployeeRuleCount[employeeID] = len(rules)
	}

	// 更新 Conversation.Meta（信息收集完成后，存储排班信息）
	// 通过 SaveConversation 触发更新（它会自动调用 updateConversationMetaFromSession）
	// 这样可以确保 Meta 和 WorkflowContext 都得到更新
	conversationSvc, ok := engine.GetService[d_service.IConversationService](wctx, "conversation")
	if ok && conversationSvc != nil {
		// 触发保存，这会自动更新 Meta
		if err := conversationSvc.SaveConversation(ctx, sess.ID, sess.Messages); err != nil {
			logger.Warn("Failed to update conversation meta after info collection", "error", err)
		} else {
			logger.Debug("Conversation meta updated after info collection", "sessionID", sess.ID)
		}
	}

	// 使用 Actor 的 ReturnToParent 方法返回
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		return nil
	}

	result := engine.NewSubWorkflowResult(map[string]any{
		"output": output,
	})
	return actor.ReturnToParent(ctx, result)
}

// actInfoCollectReturnToParentWithCancel 取消返回父工作流
func actInfoCollectReturnToParentWithCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Returning to parent workflow with cancel", "sessionID", sess.ID)

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		return fmt.Errorf("user cancelled")
	}

	result := engine.NewSubWorkflowError(fmt.Errorf("user cancelled"))
	return actor.ReturnToParent(ctx, result)
}

// actInfoCollectCancel 处理取消
func actInfoCollectCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Info-collect cancelled", "sessionID", sess.ID)

	// 发送取消消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "已取消信息收集。"); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// buildStaffCountFields 构建人员数量配置表单字段
// 使用 daily-grid 类型，支持按天配置每个班次的人数需求
func buildStaffCountFields(scheduleCtx *d_model.ScheduleCreateContext) []session.WorkflowActionField {
	fields := make([]session.WorkflowActionField, 0)

	for _, shift := range scheduleCtx.SelectedShifts {
		// 构建默认值（map[date]count 格式）
		defaultValue := make(map[string]int)
		if reqs, ok := scheduleCtx.ShiftStaffRequirements[shift.ID]; ok {
			for date, count := range reqs {
				defaultValue[date] = count
			}
		}

		// 使用 daily-grid 类型，通过 Options[0].Extra 传递额外信息
		fields = append(fields, session.WorkflowActionField{
			Name:         fmt.Sprintf("shift_%s_count", shift.ID),
			Label:        shift.Name,
			Type:         session.FieldTypeDailyGrid,
			Required:     false,
			DefaultValue: defaultValue,
			Options: []session.FieldOption{
				{
					Label: shift.Name,
					Value: shift.ID,
					Extra: map[string]any{
						"shiftId":    shift.ID,
						"shiftName":  shift.Name,
						"startTime":  shift.StartTime,
						"endTime":    shift.EndTime,
						"shiftColor": shift.Color,
						"startDate":  scheduleCtx.StartDate,
						"endDate":    scheduleCtx.EndDate,
					},
				},
			},
		})
	}

	return fields
}

// parseStaffCountPayload 解析人数配置 payload
// 支持两种格式：
// 1. 单个数字（float64/int）：应用到排班周期内的所有日期
// 2. JSON 对象 map[date]count：每天设置不同人数
func parseStaffCountPayload(payloadMap map[string]any, scheduleCtx *d_model.ScheduleCreateContext) error {
	// 检查是否有修改
	hasModification := false
	for k := range payloadMap {
		if len(k) > 6 && k[:6] == "shift_" {
			hasModification = true
			break
		}
	}

	if !hasModification {
		return nil
	}

	// 确保 ShiftStaffRequirements 已初始化
	if scheduleCtx.ShiftStaffRequirements == nil {
		scheduleCtx.ShiftStaffRequirements = make(map[string]map[string]int)
	}

	// 解析日期范围（用于单个数字时展开到所有日期）
	startDate, startErr := time.Parse("2006-01-02", scheduleCtx.StartDate)
	endDate, endErr := time.Parse("2006-01-02", scheduleCtx.EndDate)

	for _, shift := range scheduleCtx.SelectedShifts {
		// 字段名与 buildStaffCountFields 保持一致
		fieldName := fmt.Sprintf("shift_%s_count", shift.ID)

		// 如果用户输入了值
		if countVal, exists := payloadMap[fieldName]; exists {
			// 格式1：JSON 对象 {"2024-01-01": 2, "2024-01-02": 3, ...}
			if dailyMap, ok := countVal.(map[string]any); ok {
				shiftReqs := make(map[string]int)
				for dateStr, count := range dailyMap {
					switch v := count.(type) {
					case float64:
						shiftReqs[dateStr] = int(v)
					case int:
						shiftReqs[dateStr] = v
					case int64:
						shiftReqs[dateStr] = int(v)
					default:
						shiftReqs[dateStr] = 1 // 默认值
					}
				}
				scheduleCtx.ShiftStaffRequirements[shift.ID] = shiftReqs
				continue
			}

			// 格式2：单个数字，应用到所有日期
			var staffCount int
			switch v := countVal.(type) {
			case float64:
				staffCount = int(v)
			case int:
				staffCount = v
			case int64:
				staffCount = int(v)
			default:
				continue // 无法解析，跳过
			}

			// 将单个数字展开到排班周期内的所有日期
			if startErr == nil && endErr == nil && staffCount > 0 {
				shiftReqs := make(map[string]int)
				for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
					shiftReqs[d.Format("2006-01-02")] = staffCount
				}
				scheduleCtx.ShiftStaffRequirements[shift.ID] = shiftReqs
			}
		}
	}

	return nil
}
