package collectstaffcount

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

// actStart 启动人数收集
func actStart(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 直接获取强类型输入参数
	input, ok := sess.Data[engine.DataKeySubWorkflowInput].(*CollectStaffCountInput)
	if !ok || input == nil {
		return fmt.Errorf("sub-workflow input (*CollectStaffCountInput) not found")
	}

	if input.StartDate == "" || input.EndDate == "" {
		return fmt.Errorf("start_date and end_date are required")
	}

	if len(input.ShiftIDs) == 0 {
		return fmt.Errorf("shift_ids is required")
	}

	// 从 session 获取 OrgID
	orgID := input.OrgID
	if orgID == "" {
		orgID = sess.OrgID
	}

	// 获取 rosteringService
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 查询班次详情
	shifts, err := service.ListShifts(ctx, orgID, "")
	if err != nil {
		return fmt.Errorf("failed to list shifts: %w", err)
	}

	// 过滤出需要的班次
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		shiftMap[shift.ID] = shift
	}

	selectedShifts := make([]*d_model.Shift, 0, len(input.ShiftIDs))
	for _, id := range input.ShiftIDs {
		if shift, ok := shiftMap[id]; ok {
			selectedShifts = append(selectedShifts, shift)
		}
	}

	if len(selectedShifts) == 0 {
		return fmt.Errorf("no valid shifts found")
	}

	SortShiftsByPriority(selectedShifts)

	// 获取周人数配置
	weeklyConfigs := make(map[string]map[int]int)
	for _, shift := range selectedShifts {
		weeklyConfig, err := service.GetWeeklyStaffConfig(ctx, orgID, shift.ID)
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

	// 创建临时上下文用于初始化人数需求
	tempCtx := &d_model.ScheduleCreateContext{
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		SelectedShifts: selectedShifts,
	}

	// 初始化人数需求
	if err := InitializeStaffRequirementsWithWeeklyConfig(tempCtx, weeklyConfigs); err != nil {
		return fmt.Errorf("failed to initialize staff requirements: %w", err)
	}

	// 保存到 session（显式调用SetData确保持久化）
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "staff_count_context", tempCtx); err != nil {
		return fmt.Errorf("failed to save staff_count_context: %w", err)
	}

	// 构建人数配置界面
	dateDisplay := FormatDateRangeForDisplay(input.StartDate, input.EndDate)
	var shiftNames []string
	for _, shift := range selectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	infoMessage := fmt.Sprintf("已选择 %d 个班次：%s\n\n排班周期：%s\n\n请配置每个班次每天所需的人数。如果使用默认值，可以直接点击「确认人数」。",
		len(selectedShifts), FormatList(shiftNames), dateDisplay)

	staffCountFields := buildStaffCountFields(tempCtx)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, infoMessage, []session.WorkflowAction{
		{
			ID:     "confirm_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "确认人数",
			Event:  CollectStaffCountEventConfirmed,
			Style:  session.ActionStylePrimary,
			Fields: staffCountFields,
		},
		{
			ID:     "modify_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "调整人数",
			Event:  CollectStaffCountEventModified,
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: CollectStaffCountEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// actConfirm 确认人数
func actConfirm(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 从数据库读取上下文（确保获取最新持久化的数据）
	tempCtxRaw, ok, err := wctx.SessionService().GetData(ctx, sess.ID, "staff_count_context")
	if err != nil {
		return fmt.Errorf("failed to get staff_count_context: %w", err)
	}
	if !ok {
		return fmt.Errorf("staff_count_context not found in session")
	}

	tempCtx, ok := tempCtxRaw.(*d_model.ScheduleCreateContext)
	if !ok || tempCtx == nil {
		return fmt.Errorf("staff_count_context has invalid type")
	}

	// 解析人数配置
	if payload != nil {
		var payloadMap map[string]any
		if err := ParsePayload(payload, &payloadMap); err == nil {
			if err := parseStaffCountPayload(payloadMap, tempCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			}
		}
	}

	logger.Info("Staff count confirmed", "shiftsCount", len(tempCtx.SelectedShifts))
	return nil
}

// actModify 修改人数
func actModify(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 从数据库读取上下文（确保获取最新持久化的数据）
	tempCtxRaw, ok, err := wctx.SessionService().GetData(ctx, sess.ID, "staff_count_context")
	if err != nil {
		return fmt.Errorf("failed to get staff_count_context: %w", err)
	}
	if !ok {
		return fmt.Errorf("staff_count_context not found in session")
	}

	tempCtx, ok := tempCtxRaw.(*d_model.ScheduleCreateContext)
	if !ok || tempCtx == nil {
		return fmt.Errorf("staff_count_context has invalid type")
	}

	// 解析并更新人数配置
	if payload != nil {
		var payloadMap map[string]any
		if err := ParsePayload(payload, &payloadMap); err == nil {
			if err := parseStaffCountPayload(payloadMap, tempCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			}
		}
	}

	// 重新显示配置界面
	dateDisplay := FormatDateRangeForDisplay(tempCtx.StartDate, tempCtx.EndDate)
	var shiftNames []string
	for _, shift := range tempCtx.SelectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	infoMessage := fmt.Sprintf("已更新人数配置。\n\n班次：%s\n排班周期：%s\n\n可以继续调整或点击「确认人数」进入下一步。",
		FormatList(shiftNames), dateDisplay)

	staffCountFields := buildStaffCountFields(tempCtx)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, infoMessage, []session.WorkflowAction{
		{
			ID:     "confirm_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "确认人数",
			Event:  CollectStaffCountEventConfirmed,
			Style:  session.ActionStylePrimary,
			Fields: staffCountFields,
		},
		{
			ID:     "modify_staff_count",
			Type:   session.ActionTypeWorkflow,
			Label:  "调整人数",
			Event:  CollectStaffCountEventModified,
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: CollectStaffCountEventCancel,
			Style: session.ActionStyleSecondary,
		},
	})
}

// actTriggerReturn 触发返回
func actTriggerReturn(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, CollectStaffCountEventReturn, nil)
}

// actCancel 处理取消
func actCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Staff count collection cancelled", "sessionID", sess.ID)

	// 发送取消消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "已取消人数配置。"); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	// 触发返回
	return wctx.Send(ctx, CollectStaffCountEventReturn, nil)
}

// actReturnToParent 返回父工作流
func actReturnToParent(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 从数据库读取上下文（确保获取最新持久化的数据）
	tempCtxRaw, ok, err := wctx.SessionService().GetData(ctx, sess.ID, "staff_count_context")
	if err != nil {
		return fmt.Errorf("failed to get staff_count_context: %w", err)
	}
	if !ok {
		return fmt.Errorf("staff_count_context not found in session")
	}

	tempCtx, ok := tempCtxRaw.(*d_model.ScheduleCreateContext)
	if !ok || tempCtx == nil {
		return fmt.Errorf("staff_count_context has invalid type")
	}

	// 构建强类型输出（仅包含staffCount > 0的日期）
	requirements := make([]ShiftDailyRequirement, 0, len(tempCtx.ShiftStaffRequirements))
	for shiftID, dateMap := range tempCtx.ShiftStaffRequirements {
		dailyReqs := make([]DailyStaffRequirement, 0, len(dateMap))
		for date, count := range dateMap {
			if count > 0 { // 仅包含需要排班的日期
				dailyReqs = append(dailyReqs, DailyStaffRequirement{
					Date:       date,
					StaffCount: count,
				})
			}
		}

		if len(dailyReqs) > 0 {
			requirements = append(requirements, ShiftDailyRequirement{
				ShiftID:           shiftID,
				DailyRequirements: dailyReqs,
			})
		}
	}

	// 构建输出
	output := &CollectStaffCountOutput{
		ShiftStaffRequirements: requirements,
	}

	logger.Info("Returning to parent workflow with success",
		"sessionID", sess.ID,
		"shiftCount", len(tempCtx.SelectedShifts))

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

// actReturnToParentWithCancel 取消返回父工作流
func actReturnToParentWithCancel(ctx context.Context, wctx engine.Context, payload any) error {
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

// ============================================================
// 辅助函数
// ============================================================

// buildStaffCountFields 构建人员数量配置表单字段
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

		// 使用 daily-grid 类型
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

	// 解析日期范围
	startDate, startErr := time.Parse("2006-01-02", scheduleCtx.StartDate)
	endDate, endErr := time.Parse("2006-01-02", scheduleCtx.EndDate)

	for _, shift := range scheduleCtx.SelectedShifts {
		fieldName := fmt.Sprintf("shift_%s_count", shift.ID)

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
						shiftReqs[dateStr] = 1
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
				continue
			}

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
