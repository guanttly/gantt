// Package create 排班创建工作流 V2 - Actions 实现
//
// 实现工作流各阶段的 Action 函数
package create

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	"jusha/agent/rostering/internal/workflow/common"
	"jusha/agent/rostering/internal/workflow/schedule_v2/adjust"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 会话数据 Keys
// ============================================================

const (
	// KeyCreateV2Context 创建工作流 V2 上下文
	KeyCreateV2Context = "create_v2_context"
)

// ============================================================
// 工作流元数据辅助函数
// ============================================================

// setAllowUserInput 设置是否允许用户输入
func setAllowUserInput(ctx context.Context, wctx engine.Context, sessionID string, allow bool) error {
	_, err := wctx.SessionService().UpdateWorkflowMeta(ctx, sessionID, func(meta *session.WorkflowMeta) error {
		if meta.Extra == nil {
			meta.Extra = make(map[string]any)
		}
		meta.Extra["allowUserInput"] = allow
		return nil
	})
	return err
}

// ============================================================
// Payload 序列化辅助函数
// ============================================================

// serializePayload 将结构体序列化为 map[string]any（用于 WorkflowAction.Payload）
// 由于 WorkflowAction.Payload 的类型是 map[string]any，需要通过 JSON 序列化/反序列化来转换
func serializePayload(payload any) map[string]any {
	if payload == nil {
		return nil
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}
	return result
}

// parsePayloadToMap 将 payload 解析为 map[string]any
func parsePayloadToMap(payload any, result *map[string]any) error {
	if payload == nil {
		*result = nil
		return nil
	}
	// 如果已经是 map，直接使用
	if m, ok := payload.(map[string]any); ok {
		*result = m
		return nil
	}
	// 否则通过 JSON 序列化/反序列化转换
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, result)
}

// ============================================================
// 阶段 0: 初始化
// ============================================================

// actStartInfoCollect 启动信息收集子工作流
func actStartInfoCollect(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting workflow", "sessionID", sess.ID)

	// 初始化工作流上下文
	createCtx := NewCreateV2Context()
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 禁用用户输入（排班流程中默认禁用输入框）
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 发送流程转换消息
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
		"🚀 开始创建排班方案"); err != nil {
		logger.Warn("Failed to send welcome message", "error", err)
	}

	// 直接获取数据（替代 InfoCollect 子工作流）
	// 数据获取将在 actOnInfoCollected 中完成
	// 触发信息收集完成事件
	return wctx.Send(ctx, CreateV2EventInfoCollected, nil)
}

// ============================================================
// 阶段 1: 信息收集完成处理
// ============================================================

// actOnInfoCollected 处理信息收集完成事件
func actOnInfoCollected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Info collection completed", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// TODO: 从 InfoCollect 子工作流的输出中提取数据
	// 当前使用模拟数据
	if err := populateInfoFromSubWorkflow(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to populate info: %w", err)
	}

	// 按类型分类班次
	createCtx.ClassifiedShifts = ClassifyShiftsByType(createCtx.SelectedShifts)

	// 从规则中提取个人需求
	personalNeeds := ExtractPersonalNeeds(createCtx.Rules, createCtx.StaffList)
	createCtx.PersonalNeeds = personalNeeds

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 开始时间确认阶段
	return startPeriodConfirmPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 1.5: 确认排班时间
// ============================================================

// startPeriodConfirmPhase 开始时间确认阶段
func startPeriodConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting period confirmation phase", "sessionID", sess.ID)

	// 格式化日期范围用于显示
	dateRangeDisplay := common.FormatDateRangeForDisplay(createCtx.StartDate, createCtx.EndDate)

	message := fmt.Sprintf("📅 **请确认排班时间范围**\n\n当前时间范围：**%s**\n\n", dateRangeDisplay)
	message += "如需修改，请点击「修改时间」按钮。"

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认时间",
			Event: session.WorkflowEvent(CreateV2EventPeriodConfirmed),
			Style: session.ActionStylePrimary,
			Payload: serializePayload(&PeriodConfirmPayload{
				StartDate: createCtx.StartDate,
				EndDate:   createCtx.EndDate,
			}),
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改时间",
			Event: session.WorkflowEvent(CreateV2EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 发送消息（不包含工作流操作按钮，只包含元数据）
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":      "periodConfirm",
			"startDate": createCtx.StartDate,
			"endDate":   createCtx.EndDate,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send period confirm message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnPeriodConfirmed 处理时间确认完成
func actOnPeriodConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Period confirmed", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从 payload 中获取确认的时间（如果用户修改了）
	// 前端表单对话框直接返回 formData 对象，包含 startDate 和 endDate
	if periodPayload, ok := payload.(*PeriodConfirmPayload); ok && periodPayload != nil {
		if periodPayload.StartDate != "" {
			createCtx.StartDate = periodPayload.StartDate
		}
		if periodPayload.EndDate != "" {
			createCtx.EndDate = periodPayload.EndDate
		}
	} else if payloadMap, ok := payload.(map[string]any); ok {
		// 兼容旧格式（临时）
		if startDate, ok := payloadMap["startDate"].(string); ok && startDate != "" {
			createCtx.StartDate = startDate
		}
		if endDate, ok := payloadMap["endDate"].(string); ok && endDate != "" {
			createCtx.EndDate = endDate
		}
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送确认消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		fmt.Sprintf("✅ 时间范围已确认：%s\n\n", common.FormatDateRangeForDisplay(createCtx.StartDate, createCtx.EndDate))); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 清空工作流 action（离开可操作节点，进入下一个可操作节点）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 开始班次和人数确认阶段
	return startShiftsConfirmPhase(ctx, wctx, createCtx)
}

// actModifyPeriod 修改时间
func actModifyPeriod(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Modifying period", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	// 前端会检测到 fields 并弹出表单对话框
	workflowActions := []session.WorkflowAction{
		{
			ID:    "modify_period",
			Type:  session.ActionTypeWorkflow,
			Label: "修改时间范围",
			Event: session.WorkflowEvent(CreateV2EventPeriodConfirmed),
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:         "startDate",
					Label:        "开始日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					Placeholder:  "选择开始日期",
					DefaultValue: createCtx.StartDate,
				},
				{
					Name:         "endDate",
					Label:        "结束日期",
					Type:         session.FieldTypeDate,
					Required:     true,
					Placeholder:  "选择结束日期",
					DefaultValue: createCtx.EndDate,
				},
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	message := "📅 请选择新的排班时间范围："

	// 发送消息（不包含工作流操作按钮）
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":      "modifyPeriod",
			"startDate": createCtx.StartDate,
			"endDate":   createCtx.EndDate,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send modify period message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// ============================================================
// 阶段 1.6: 确认班次选择
// ============================================================

// startShiftsConfirmPhase 开始班次选择确认阶段
func startShiftsConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting shifts confirmation phase", "sessionID", sess.ID)

	// 构建班次确认消息（只显示班次选择，不显示人数配置）
	message := "🏢 **请确认排班班次**\n\n"
	message += fmt.Sprintf("共 **%d** 个班次需要排班：\n\n", len(createCtx.SelectedShifts))

	// 显示每个班次的基本信息
	for i, shift := range createCtx.SelectedShifts {
		if i >= 10 { // 最多显示10个
			message += fmt.Sprintf("\n... 还有 %d 个班次\n", len(createCtx.SelectedShifts)-10)
			break
		}

		message += fmt.Sprintf("  • **%s**", shift.Name)
		description := fmt.Sprintf("%s-%s", shift.StartTime, shift.EndTime)
		if shift.IsOvernight {
			description += " (跨夜)"
		}
		message += fmt.Sprintf("：%s\n", description)
	}

	message += "\n确认班次后，将进入人数配置阶段。如需修改班次选择，请点击「修改班次」按钮。"

	// 获取服务以加载班次分组人员数据
	service, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !serviceOk {
		return fmt.Errorf("rosteringService not found")
	}

	// 加载每个班次的分组人员数据（用于预览）
	shiftGroupsData := make([]map[string]any, 0, len(createCtx.SelectedShifts))
	totalStaff := 0
	allStaffMap := make(map[string]bool) // 用于统计总人数（去重）

	for _, shift := range createCtx.SelectedShifts {
		members, err := service.GetShiftGroupMembers(ctx, shift.ID)
		if err != nil {
			logger.Warn("Failed to get members for shift", "shiftID", shift.ID, "error", err)
			continue
		}

		// 统计去重后的总人数
		for _, m := range members {
			if !allStaffMap[m.ID] {
				allStaffMap[m.ID] = true
				totalStaff++
			}
		}

		// 构建班次人员数据（匹配前端 StaffDetailsDialog 的接口）
		staffList := make([]map[string]any, 0, len(members))
		for _, m := range members {
			staffItem := map[string]any{
				"id":   m.ID,
				"name": m.Name,
			}
			if m.Position != "" {
				staffItem["position"] = m.Position
			}
			if m.DepartmentID != "" {
				staffItem["departmentId"] = m.DepartmentID
			}
			staffList = append(staffList, staffItem)
		}

		shiftGroupsData = append(shiftGroupsData, map[string]any{
			"shiftId":    shift.ID,
			"shiftName":  shift.Name,
			"startTime":  shift.StartTime,
			"endTime":    shift.EndTime,
			"staffCount": len(staffList),
			"staffList":  staffList,
		})
	}

	// 构建查询按钮（保留在消息上）
	queryActions := []session.WorkflowAction{
		{
			ID:    "preview_shift_groups",
			Type:  session.ActionTypeQuery,
			Label: "预览班次分组人员",
			Style: session.ActionStyleSuccess,
			Payload: serializePayload(map[string]any{
				"totalStaff": totalStaff,
				"totalTeams": len(shiftGroupsData), // 使用 totalTeams 以匹配前端接口
				"shifts":     shiftGroupsData,
			}),
		},
	}

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	// 注意：这里只确认班次选择，不包含人数配置
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认班次",
			Event: session.WorkflowEvent(CreateV2EventShiftsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改班次",
			Event: session.WorkflowEvent(CreateV2EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 发送消息（只包含查询按钮）
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: queryActions, // 只包含查询类型的按钮
		Metadata: map[string]any{
			"type":         "shiftsConfirm",
			"shiftsCount":  len(createCtx.SelectedShifts),
			"shifts":       createCtx.SelectedShifts,
			"requirements": createCtx.StaffRequirements,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send shifts confirm message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnShiftsConfirmed 处理班次选择确认完成
func actOnShiftsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Shifts confirmed", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从 payload 中获取确认的班次ID列表（如果用户修改了）
	// 由于 payload 可能通过 JSON 序列化/反序列化，需要支持从 map 转换
	var shiftsPayload *ShiftsConfirmPayload
	if payload != nil {
		// 尝试直接类型断言
		if p, ok := payload.(*ShiftsConfirmPayload); ok {
			shiftsPayload = p
		} else if p, ok := payload.(ShiftsConfirmPayload); ok {
			shiftsPayload = &p
		} else if payloadMap, ok := payload.(map[string]any); ok && len(payloadMap) > 0 {
			// 从 map 转换为结构体（JSON 反序列化后的情况）
			jsonBytes, err := json.Marshal(payloadMap)
			if err == nil {
				var p ShiftsConfirmPayload
				if err := json.Unmarshal(jsonBytes, &p); err == nil {
					shiftsPayload = &p
				}
			}
		}
	}

	if shiftsPayload != nil && len(shiftsPayload.ShiftIDs) > 0 {
		shiftIDs := shiftsPayload.ShiftIDs
		// 获取所有班次
		service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
		if !ok {
			return fmt.Errorf("rosteringService not found")
		}

		allShifts, err := service.ListShifts(ctx, sess.OrgID, "")
		if err != nil {
			return fmt.Errorf("failed to list shifts: %w", err)
		}

		// 构建班次ID到班次的映射（只包含已启用的班次）
		shiftMap := make(map[string]*d_model.Shift)
		for _, shift := range allShifts {
			// 只允许选择已启用的班次
			if shift.IsActive {
				shiftMap[shift.ID] = shift
			}
		}

		// 根据用户选择的ID更新 SelectedShifts
		selectedShifts := make([]*d_model.Shift, 0, len(shiftIDs))
		for _, id := range shiftIDs {
			if shift, exists := shiftMap[id]; exists {
				selectedShifts = append(selectedShifts, shift)
			}
		}

		if len(selectedShifts) > 0 {
			createCtx.SelectedShifts = selectedShifts
			logger.Info("Updated selected shifts", "count", len(selectedShifts))

			// 重新加载人员配置需求（基于新的班次列表）
			// 使用公共方法重新加载基础信息
			basicCtx, err := common.LoadScheduleBasicContext(
				ctx,
				wctx,
				sess.OrgID,
				createCtx.StartDate,
				createCtx.EndDate,
				func() []string {
					ids := make([]string, 0, len(selectedShifts))
					for _, s := range selectedShifts {
						ids = append(ids, s.ID)
					}
					return ids
				}(),
			)
			if err != nil {
				logger.Warn("Failed to reload basic context", "error", err)
			} else {
				// 更新人员配置需求
				createCtx.StaffRequirements = basicCtx.StaffRequirements
				// 更新人员列表（可能因为班次变化而变化）
				createCtx.StaffList = basicCtx.StaffList       // 班次关联的员工（用于AI排班）
				createCtx.AllStaffList = basicCtx.AllStaffList // 所有员工（用于信息检索）
				createCtx.StaffLeaves = basicCtx.StaffLeaves
				// 更新规则（可能因为班次变化而变化）
				createCtx.Rules = basicCtx.Rules

				// 重新分类班次（基于更新后的 SelectedShifts）
				createCtx.ClassifiedShifts = ClassifyShiftsByType(createCtx.SelectedShifts)
			}
		}
	}

	// 确保 ClassifiedShifts 是基于 SelectedShifts 的（即使没有修改班次）
	// 防止之前使用了错误的班次列表
	if createCtx.ClassifiedShifts == nil || len(createCtx.ClassifiedShifts) == 0 {
		createCtx.ClassifiedShifts = ClassifyShiftsByType(createCtx.SelectedShifts)
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送确认消息
	message := "✅ 班次选择已确认：\n\n"
	message += fmt.Sprintf("  • 共 **%d** 个班次\n\n", len(createCtx.SelectedShifts))
	message += "接下来请配置每个班次每天所需的人数..."

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 清空工作流 action（离开可操作节点，进入下一个可操作节点）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 开始人数配置确认阶段
	return startStaffCountConfirmPhase(ctx, wctx, createCtx)
}

// actModifyShifts 修改班次和人数
func actModifyShifts(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Modifying shifts", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取所有班次
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	allShifts, err := service.ListShifts(ctx, sess.OrgID, "")
	if err != nil {
		return fmt.Errorf("failed to list shifts: %w", err)
	}

	// 过滤出已启用的班次（不包含已禁用的班次）
	activeShifts := make([]*d_model.Shift, 0, len(allShifts))
	for _, shift := range allShifts {
		if shift.IsActive {
			activeShifts = append(activeShifts, shift)
		}
	}

	// 构建班次选项（用于 multi-select，只包含已启用的班次）
	shiftOptions := make([]session.FieldOption, 0, len(activeShifts))
	for _, shift := range activeShifts {
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

	// 构建默认选中的班次ID列表
	defaultShiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		defaultShiftIDs = append(defaultShiftIDs, shift.ID)
	}

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	workflowActions := []session.WorkflowAction{
		{
			ID:    "modify_shifts",
			Type:  session.ActionTypeWorkflow,
			Label: "修改班次选择",
			Event: session.WorkflowEvent(CreateV2EventShiftsConfirmed),
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:         "shiftIds",
					Label:        "选择需要排班的班次",
					Type:         session.FieldTypeMultiSelect,
					Required:     true,
					Options:      shiftOptions,
					DefaultValue: defaultShiftIDs,
					Placeholder:  "搜索班次...",
				},
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	message := "🏢 请选择需要排班的班次："

	// 发送消息（不包含工作流操作按钮）
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":          "modifyShifts",
			"shiftsCount":   len(activeShifts),
			"selectedCount": len(defaultShiftIDs),
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send modify shifts message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// ============================================================
// 阶段 1.7: 确认人数配置
// ============================================================

// startStaffCountConfirmPhase 开始人数配置确认阶段
func startStaffCountConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting staff count confirmation phase", "sessionID", sess.ID)

	// 构建人数配置确认消息
	dateDisplay := fmt.Sprintf("%s 至 %s", createCtx.StartDate, createCtx.EndDate)
	var shiftNames []string
	for _, shift := range createCtx.SelectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	message := "📊 **请配置每个班次每天所需的人数**\n\n"
	message += fmt.Sprintf("已选择 %d 个班次：%s\n\n", len(createCtx.SelectedShifts), formatList(shiftNames))
	message += fmt.Sprintf("排班周期：%s\n\n", dateDisplay)
	message += "请配置每个班次每天所需的人数。如果使用默认值，可以直接点击「确认人数」。"

	// 构建人数配置字段（使用 daily-grid 类型）
	staffCountFields := buildStaffCountFields(createCtx)

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	workflowActions := []session.WorkflowAction{
		{
			Type:   session.ActionTypeWorkflow,
			Label:  "确认人数",
			Event:  session.WorkflowEvent(CreateV2EventStaffCountConfirmed),
			Style:  session.ActionStylePrimary,
			Fields: staffCountFields,
		},
		{
			Type:   session.ActionTypeWorkflow,
			Label:  "修改人数",
			Event:  session.WorkflowEvent(CreateV2EventModifyStaffCount),
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "返回修改班次",
			Event: session.WorkflowEvent(CreateV2EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 发送消息
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":         "staffCountConfirm",
			"shiftsCount":  len(createCtx.SelectedShifts),
			"requirements": createCtx.StaffRequirements,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send staff count confirm message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnStaffCountConfirmed 处理人数配置确认完成
func actOnStaffCountConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Staff count confirmed", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 解析人数配置 payload
	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			// 解析人数配置
			if err := parseStaffCountPayload(payloadMap, createCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			} else {
				logger.Info("Parsed staff count payload", "shiftsCount", len(createCtx.SelectedShifts))
			}
		}
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送确认消息
	message := "✅ 人数配置已确认：\n\n"
	message += fmt.Sprintf("  • 共 **%d** 个班次\n", len(createCtx.SelectedShifts))

	totalRequirements := 0
	for _, reqs := range createCtx.StaffRequirements {
		totalRequirements += len(reqs)
	}
	message += fmt.Sprintf("  • 共 **%d** 个日期需要排班\n\n", totalRequirements)
	message += "开始确认个人需求..."

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 清空工作流 action（离开可操作节点，进入下一个可操作节点）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 开始个人需求阶段
	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// actModifyStaffCount 修改人数配置
func actModifyStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Modifying staff count", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 解析人数配置 payload（如果存在）
	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			// 解析人数配置
			if err := parseStaffCountPayload(payloadMap, createCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			} else {
				logger.Info("Parsed staff count payload", "shiftsCount", len(createCtx.SelectedShifts))
			}
		}
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 重新显示人数配置界面（返回到人数确认阶段）
	return startStaffCountConfirmPhase(ctx, wctx, createCtx)
}

// formatList 格式化列表为字符串（用于显示）
func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) <= 3 {
		result := ""
		for i, item := range items {
			if i > 0 {
				result += "、"
			}
			result += item
		}
		return result
	}
	result := ""
	for i := 0; i < 3; i++ {
		if i > 0 {
			result += "、"
		}
		result += items[i]
	}
	return fmt.Sprintf("%s 等 %d 个", result, len(items))
}

// ============================================================
// 阶段 2: 个人需求收集与确认
// ============================================================

// startPersonalNeedsPhase 开始个人需求阶段
func startPersonalNeedsPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting personal needs phase", "sessionID", sess.ID)

	// 构建个人需求确认界面
	needsCount := 0
	temporaryCount := 0
	permanentCount := 0
	for _, needs := range createCtx.PersonalNeeds {
		needsCount += len(needs)
		for _, need := range needs {
			if need.NeedType == "temporary" {
				temporaryCount++
			} else {
				permanentCount++
			}
		}
	}

	// 根据需求来源构建不同的消息
	var message string
	if needsCount == 0 {
		message = "📋 当前 **0** 条个人需求。\n\n"
		message += "未发现个人需求规则，将直接开始排班。\n\n是否需要添加临时需求？"
	} else {
		message = fmt.Sprintf("📋 当前 **%d** 条个人需求", needsCount)
		if permanentCount > 0 && temporaryCount > 0 {
			message += fmt.Sprintf("（其中 %d 条常态化需求，%d 条临时需求）", permanentCount, temporaryCount)
		} else if temporaryCount > 0 {
			message += fmt.Sprintf("（%d 条临时需求）", temporaryCount)
		} else {
			message += fmt.Sprintf("（%d 条常态化需求）", permanentCount)
		}
		message += "。\n\n"
		message += "这些需求将作为排班约束条件，请确认或补充："
	}

	if needsCount > 0 {

		// 构建预览数据
		previewData := buildPersonalNeedsPreviewData(createCtx.PersonalNeeds)

		// 构建查询按钮（保留在消息上）
		queryActions := []session.WorkflowAction{
			{
				ID:      "view_personal_needs",
				Type:    session.ActionTypeQuery,
				Label:   "📋 预览需求",
				Payload: previewData,
				Style:   session.ActionStyleSuccess, // 绿色背景
			},
		}

		// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
		workflowActions := buildPersonalNeedsActions(createCtx.PersonalNeeds, previewData)

		// 发送主消息（只包含查询按钮）
		mainMsg := session.Message{
			Role:    session.RoleAssistant,
			Content: message,
			Actions: queryActions, // 只包含查询类型的按钮
			Metadata: map[string]any{
				"type":        "personalNeeds",
				"previewData": previewData,
				"needsCount":  needsCount,
			},
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
			logger.Warn("Failed to send personal needs message", "error", err)
		}

		// 设置工作流 meta（包含工作流操作按钮）
		return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
	} else {
		// 没有需求时，提供添加需求的选项
		workflowActions := []session.WorkflowAction{
			{
				Type:  session.ActionTypeWorkflow,
				Label: "不需要，继续",
				Event: session.WorkflowEvent(CreateV2EventPersonalNeedsConfirmed),
				Style: session.ActionStylePrimary,
			},
			{
				Type:  session.ActionTypeWorkflow,
				Label: "添加需求",
				Event: session.WorkflowEvent(CreateV2EventUserModify),
				Style: session.ActionStyleSecondary,
			},
			{
				Type:  session.ActionTypeWorkflow,
				Label: "取消排班",
				Event: session.WorkflowEvent(CreateV2EventUserCancel),
				Style: session.ActionStyleDanger,
			},
		}

		// 发送消息（不包含工作流操作按钮）
		mainMsg := session.Message{
			Role:    session.RoleAssistant,
			Content: message,
			Actions: nil, // 工作流操作按钮不放在消息上
			Metadata: map[string]any{
				"type":       "personalNeeds",
				"needsCount": 0,
			},
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
			logger.Warn("Failed to send personal needs message", "error", err)
		}

		// 设置工作流 meta（包含工作流操作按钮）
		return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
	}
}

// buildPersonalNeedsActions 构建个人需求确认的 Actions（仅工作流操作按钮）
func buildPersonalNeedsActions(_ map[string][]*PersonalNeed, _ map[string]interface{}) []session.WorkflowAction {
	// 预览按钮已作为独立消息发送，这里只返回工作流操作按钮
	return []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认并继续",
			Event: session.WorkflowEvent(CreateV2EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改需求",
			Event: session.WorkflowEvent(CreateV2EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}
}

// buildPersonalNeedsPreviewData 构建个人需求预览数据
func buildPersonalNeedsPreviewData(personalNeeds map[string][]*PersonalNeed) map[string]interface{} {
	totalNeeds := 0
	permanentCount := 0
	temporaryCount := 0
	allNeeds := make([]*PersonalNeed, 0)
	needsByStaff := make(map[string][]*PersonalNeed)

	for staffID, needs := range personalNeeds {
		totalNeeds += len(needs)
		needsByStaff[staffID] = needs
		allNeeds = append(allNeeds, needs...)

		for _, need := range needs {
			if need.NeedType == "permanent" {
				permanentCount++
			} else {
				temporaryCount++
			}
		}
	}

	return map[string]interface{}{
		"totalNeeds":          totalNeeds,
		"permanentNeedsCount": permanentCount,
		"temporaryNeedsCount": temporaryCount,
		"needsByStaff":        needsByStaff,
		"allNeeds":            allNeeds,
	}
}

// actOnPersonalNeedsConfirmed 处理个人需求确认
func actOnPersonalNeedsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Personal needs confirmed", "sessionID", sess.ID, "payload", payload)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 检查 payload 是否包含临时需求数据（从修改需求表单提交）
	// 添加临时需求时，payload 应该是 AddTemporaryNeedPayload 结构体
	// 直接确认时，payload 可能是 nil
	// 由于 payload 可能通过 JSON 序列化/反序列化，需要支持从 map 转换
	var addNeedPayload *AddTemporaryNeedPayload
	if payload != nil {
		// 尝试直接类型断言
		if p, ok := payload.(*AddTemporaryNeedPayload); ok {
			addNeedPayload = p
		} else if p, ok := payload.(AddTemporaryNeedPayload); ok {
			addNeedPayload = &p
		} else if payloadMap, ok := payload.(map[string]any); ok && len(payloadMap) > 0 {
			// 从 map 转换为结构体（JSON 反序列化后的情况）
			jsonBytes, err := json.Marshal(payloadMap)
			if err == nil {
				var p AddTemporaryNeedPayload
				if err := json.Unmarshal(jsonBytes, &p); err == nil {
					addNeedPayload = &p
				} else {
					logger.Warn("Failed to unmarshal payload map to AddTemporaryNeedPayload", "error", err)
				}
			}
		}
	}

	// 如果 payload 为空或没有临时需求数据，说明是直接确认（用户点击"确认并继续"）
	// 这种情况下，应该直接进入固定班次阶段，而不是要求添加临时需求
	if addNeedPayload == nil || len(addNeedPayload.Needs) == 0 {
		logger.Info("No temporary needs in payload, treating as direct confirmation")
		// 直接进入确认流程（不要求必须添加临时需求）
		createCtx.PersonalNeedsConfirmed = true
		if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
			"✅ 个人需求已确认，将在排班时作为约束条件。\n\n开始处理固定班次..."); err != nil {
			logger.Warn("Failed to send message", "error", err)
		}

		// 清空工作流 action（进入自动处理阶段）
		if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
			logger.Warn("Failed to clear workflow actions", "error", err)
		}

		return startFixedShiftPhase(ctx, wctx, createCtx)
	}

	// 验证：检查是否有未完成的填写（必填字段为空）
	invalidItems := make([]int, 0)
	for i, needItem := range addNeedPayload.Needs {
		if needItem.StaffID == "" || needItem.Description == "" || needItem.RequestType == "" {
			invalidItems = append(invalidItems, i+1)
		}
	}
	if len(invalidItems) > 0 {
		// 发送错误提示消息
		errorMsg := "❌ **验证失败**\n\n以下需求项未完整填写（请填写所有必填字段）：\n"
		for _, idx := range invalidItems {
			errorMsg += fmt.Sprintf("- 临时需求 %d\n", idx)
		}
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); err != nil {
			logger.Warn("Failed to send validation error message", "error", err)
		}
		// 重新显示添加临时需求界面
		return actModifyPersonalNeeds(ctx, wctx, nil)
	}

	if len(addNeedPayload.Needs) > 0 {
		// 批量添加/更新临时需求（先清除所有临时需求，再添加新的）
		logger.Info("Updating temporary personal needs", "count", len(addNeedPayload.Needs))

		// 清除所有已有的临时需求
		for staffID, needs := range createCtx.PersonalNeeds {
			filteredNeeds := make([]*PersonalNeed, 0)
			for _, need := range needs {
				if need.NeedType != "temporary" {
					filteredNeeds = append(filteredNeeds, need)
				}
			}
			if len(filteredNeeds) > 0 {
				createCtx.PersonalNeeds[staffID] = filteredNeeds
			} else {
				delete(createCtx.PersonalNeeds, staffID)
			}
		}
		// 获取服务
		service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
		if !ok {
			return fmt.Errorf("rosteringService not found")
		}

		// 构建人员ID到姓名的映射（避免重复查询）
		staffNameMap := make(map[string]string)
		if len(createCtx.StaffList) > 0 {
			for _, staff := range createCtx.StaffList {
				staffNameMap[staff.ID] = staff.Name
			}
		}
		// 如果上下文中找不到，尝试从服务获取（作为回退）
		if len(staffNameMap) == 0 {
			staffResult, err := service.ListStaff(ctx, d_model.StaffListFilter{
				OrgID: sess.OrgID,
			})
			if err == nil && staffResult != nil && len(staffResult.Items) > 0 {
				for _, staff := range staffResult.Items {
					staffNameMap[staff.UserID] = staff.Name
				}
			}
		}

		// 构建班次ID到名称的映射（避免重复查询）
		shiftNameMap := make(map[string]string)
		if createCtx.SelectedShifts != nil {
			for _, shift := range createCtx.SelectedShifts {
				shiftNameMap[shift.ID] = shift.Name
			}
		}
		// 如果上下文中找不到，尝试从服务获取（作为回退）
		if len(shiftNameMap) == 0 {
			allShifts, err := service.ListShifts(ctx, sess.OrgID, "")
			if err == nil {
				for _, shift := range allShifts {
					shiftNameMap[shift.ID] = shift.Name
				}
			}
		}

		// 批量处理每个临时需求
		addedCount := 0
		for _, needItem := range addNeedPayload.Needs {
			// 验证必填字段
			if needItem.StaffID == "" || needItem.Description == "" {
				logger.Warn("Skipping invalid temporary need item", "staffID", needItem.StaffID, "description", needItem.Description)
				continue
			}

			// 获取人员姓名
			staffName := staffNameMap[needItem.StaffID]
			if staffName == "" {
				staffName = needItem.StaffID
			}

			// 获取班次名称
			targetShiftName := shiftNameMap[needItem.TargetShiftID]

			// 解析目标日期（已经是 []string 格式，直接使用）
			targetDates := needItem.TargetDates
			if targetDates == nil {
				targetDates = []string{}
			}

			// 获取请求类型（默认 prefer）
			requestType := needItem.RequestType
			if requestType == "" {
				requestType = "prefer"
			}

			// 获取优先级（默认 5）
			priority := needItem.Priority
			if priority <= 0 {
				priority = 5
			}

			// 创建临时需求
			newNeed := &PersonalNeed{
				StaffID:         needItem.StaffID,
				StaffName:       staffName,
				NeedType:        "temporary",
				RequestType:     requestType,
				TargetShiftID:   needItem.TargetShiftID,
				TargetShiftName: targetShiftName,
				TargetDates:     targetDates,
				Description:     needItem.Description,
				Priority:        priority,
				Source:          "user",
				Confirmed:       true,
			}

			// 添加到上下文
			createCtx.AddPersonalNeed(newNeed)
			addedCount++
		}

		logger.Info("Added temporary personal needs", "total", len(addNeedPayload.Needs), "added", addedCount)

		// 保存上下文
		if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		// 重新显示个人需求确认界面（显示当前需求数量，提供预览、确认、修改等选项）
		// 不发送单独的确认消息，直接返回需求确认界面
		logger.Info("Returning to personal needs phase after adding temporary needs")
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	// 如果执行到这里，说明有临时需求数据但处理失败，不应该继续
	logger.Warn("Unexpected code path: addNeedPayload exists but was not processed")
	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// actModifyPersonalNeeds 修改个人需求（添加临时需求）
func actModifyPersonalNeeds(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Modifying personal needs", "sessionID", sess.ID)

	// 加载上下文（包含已加载的人员列表）
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从班次关联的分组中获取人员（而不是使用 ListStaff）
	// 构建人员选项
	staffOptions := make([]session.FieldOption, 0)
	service, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !serviceOk {
		return fmt.Errorf("rosteringService not found")
	}

	// 从上下文中获取人员列表（如果已加载）
	if createCtx.StaffList != nil && len(createCtx.StaffList) > 0 {
		for _, staff := range createCtx.StaffList {
			staffOptions = append(staffOptions, session.FieldOption{
				Label: staff.Name,
				Value: staff.ID,
				Extra: map[string]any{
					"id": staff.ID,
				},
			})
		}
	} else if createCtx.SelectedShifts != nil && len(createCtx.SelectedShifts) > 0 {
		// 如果上下文中没有人员列表，从班次关联的分组中获取
		// 使用 map 去重，但保持 SQL 返回的顺序（已按 employee_id 排序）
		allStaffMap := make(map[string]*d_model.Employee)
		allStaffOrder := make([]*d_model.Employee, 0) // 保持顺序的切片
		for _, shift := range createCtx.SelectedShifts {
			members, err := service.GetShiftGroupMembers(ctx, shift.ID)
			if err != nil {
				logger.Warn("Failed to get members for shift", "shiftID", shift.ID, "error", err)
				continue
			}
			// GetShiftGroupMembers 已按 employee_id 排序，直接按顺序添加
			for _, m := range members {
				if _, exists := allStaffMap[m.ID]; !exists {
					allStaffMap[m.ID] = m
					allStaffOrder = append(allStaffOrder, m)
				}
			}
		}
		// 使用保持顺序的切片（SQL 已按 employee_id 排序）
		for _, staff := range allStaffOrder {
			staffOptions = append(staffOptions, session.FieldOption{
				Label: staff.Name,
				Value: staff.ID,
				Extra: map[string]any{
					"id": staff.ID,
				},
			})
		}
		logger.Info("Loaded staff from shift groups", "count", len(staffOptions))
	} else {
		logger.Warn("No shifts selected, cannot load staff from groups")
	}

	// 获取服务（用于获取班次列表，service 已在上面获取）

	// 只使用用户已确认的班次（createCtx.SelectedShifts），而不是所有班次
	// 这样用户取消的班次不会出现在选择列表中
	shiftOptions := make([]session.FieldOption, 0)
	if createCtx.SelectedShifts != nil && len(createCtx.SelectedShifts) > 0 {
		for _, shift := range createCtx.SelectedShifts {
			// 只包含已启用的班次
			if !shift.IsActive {
				continue
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
					"type":      shift.Type,
					"startTime": shift.StartTime,
					"endTime":   shift.EndTime,
					"color":     shift.Color,
				},
			})
		}
	}

	// 构建请求类型选项
	requestTypeOptions := []session.FieldOption{
		{Label: "偏好（prefer）", Value: "prefer", Description: "尽量满足，但不强制"},
		{Label: "必须（must）", Value: "must", Description: "必须满足，否则排班失败"},
		{Label: "回避（avoid）", Value: "avoid", Description: "尽量避免安排"},
	}

	// 获取已有的临时需求列表（用于显示）
	existingTemporaryNeeds := make([]map[string]any, 0)
	for staffID, needs := range createCtx.PersonalNeeds {
		for _, need := range needs {
			if need.NeedType == "temporary" {
				// 构建已添加的临时需求项
				needItem := map[string]any{
					"staffId":       staffID,
					"requestType":   need.RequestType,
					"targetShiftId": need.TargetShiftID,
					"targetDates":   need.TargetDates,
					"description":   need.Description,
					"priority":      need.Priority,
				}
				existingTemporaryNeeds = append(existingTemporaryNeeds, needItem)
			}
		}
	}

	// 构建工作流操作按钮（放在工作流 meta 上，以便在离开节点时清空）
	// 使用 table-form 类型字段：上半部分表格显示已有需求，下半部分表单用于新增/编辑
	workflowActions := []session.WorkflowAction{
		{
			ID:    "manage_temporary_needs",
			Type:  session.ActionTypeWorkflow,
			Label: "管理临时需求",
			Event: session.WorkflowEvent(CreateV2EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:     "needs",
					Label:    "临时需求列表",
					Type:     session.FieldTypeTableForm,
					Required: true, // 必须至少有一条临时需求
					Extra: map[string]any{
						"tableColumns": []map[string]any{
							{"prop": "staffName", "label": "人员", "width": 120},
							{"prop": "requestType", "label": "请求类型", "width": 100},
							{"prop": "targetShiftName", "label": "目标班次", "width": 120},
							{"prop": "targetDates", "label": "目标日期", "width": 150},
							{"prop": "description", "label": "需求描述", "minWidth": 180},
							{"prop": "priority", "label": "优先级", "width": 80},
						},
						"initialItems": existingTemporaryNeeds, // 预填充已有需求
						"formFields": []map[string]any{
							{
								"name":        "staffId",
								"label":       "选择人员",
								"type":        "select",
								"required":    true,
								"options":     staffOptions,
								"placeholder": "请选择人员",
								"span":        12, // 一行占12列（一行两个字段）
							},
							{
								"name":         "requestType",
								"label":        "请求类型",
								"type":         "select",
								"required":     true,
								"options":      requestTypeOptions,
								"placeholder":  "请选择请求类型",
								"defaultValue": "prefer",
								"span":         12, // 一行占12列（一行两个字段）
							},
							{
								"name":        "targetShiftId",
								"label":       "目标班次",
								"type":        "select",
								"required":    false,
								"options":     shiftOptions,
								"placeholder": "选择班次（可选，留空表示任意班次）",
								"span":        12, // 一行占12列（一行两个字段）
							},
							{
								"name":        "targetDates",
								"label":       "目标日期",
								"type":        "date",
								"required":    false,
								"placeholder": "请选择日期（可选，可多选，留空表示整个周期）",
								"span":        12, // 一行占12列（一行两个字段）
								"extra": map[string]any{
									"multiple":  true,
									"startDate": createCtx.StartDate, // 排班周期开始日期
									"endDate":   createCtx.EndDate,   // 排班周期结束日期
								},
							},
							{
								"name":        "description",
								"label":       "需求描述",
								"type":        "textarea",
								"required":    true,
								"placeholder": "请输入需求描述，例如：希望12月15日上早班",
								"span":        24, // 占满整行
							},
							{
								"name":         "priority",
								"label":        "优先级",
								"type":         "number",
								"required":     true,
								"placeholder":  "1-10，数字越小优先级越高",
								"defaultValue": 5,
								"span":         24, // 占满整行（单独一项）
							},
						},
					},
				},
			},
		},
		{
			ID:    "back_to_confirm",
			Type:  session.ActionTypeWorkflow,
			Label: "返回需求确认",
			Event: session.WorkflowEvent(CreateV2EventSkipPhase), // 使用 SkipPhase 事件返回到确认阶段
			Style: session.ActionStyleSecondary,
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 构建消息内容，显示已有临时需求数量
	message := "📝 **管理临时需求**\n\n"
	if len(existingTemporaryNeeds) > 0 {
		message += fmt.Sprintf("当前已有 **%d** 条临时需求。\n\n", len(existingTemporaryNeeds))
	} else {
		message += "当前暂无临时需求。\n\n"
	}
	message += "您可以添加、编辑或删除临时需求，这些需求将作为排班约束条件。\n\n"
	message += "**注意**：确认前请确保所有需求项都已完整填写，且至少添加一条临时需求。"

	// 发送消息（不包含工作流操作按钮）
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type": "modifyPersonalNeeds",
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send modify personal needs message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actReturnToPersonalNeedsConfirm 返回到个人需求确认界面（从修改需求界面返回）
func actReturnToPersonalNeedsConfirm(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Returning to personal needs confirmation", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 直接返回到个人需求确认界面
	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 3: 固定班次处理
// ============================================================

// startFixedShiftPhase 开始固定班次阶段
// 注意：固定班次不是独立的班次类型，而是班次的固定人员配置
func startFixedShiftPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting fixed shift phase", "sessionID", sess.ID)

	// 禁用用户输入（固定班次处理过程中不允许输入）
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 获取 rosteringService（包含固定人员配置功能）
	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		logger.Warn("CreateV2: RosteringService not available, skipping phase")
		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
			"⏭️ 跳过固定班次"); err != nil {
			logger.Warn("Failed to send message", "error", err)
		}
		return wctx.Send(ctx, CreateV2EventFixedShiftConfirmed, nil)
	}

	// 收集所有班次ID
	shiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range createCtx.SelectedShifts {
		shiftIDs = append(shiftIDs, shift.ID)
		shiftMap[shift.ID] = shift
	}

	// 批量计算所有班次的固定排班
	allFixedSchedules, err := rosteringService.CalculateMultipleFixedSchedules(
		ctx,
		shiftIDs,
		createCtx.StartDate,
		createCtx.EndDate,
	)
	if err != nil {
		logger.Error("CreateV2: Failed to calculate fixed schedules", "error", err)
		return fmt.Errorf("计算固定排班失败: %w", err)
	}

	if len(allFixedSchedules) == 0 {
		// 没有固定班次配置，跳过此阶段
		logger.Info("CreateV2: No fixed assignments found, skipping phase")

		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
			"⏭️ 跳过固定班次"); err != nil {
			logger.Warn("Failed to send message", "error", err)
		}

		// 直接触发确认事件，进入下一阶段
		return wctx.Send(ctx, CreateV2EventFixedShiftConfirmed, nil)
	}

	// 转换为 ShiftScheduleDraft 格式并保存
	fixedDrafts := make(map[string]*d_model.ShiftScheduleDraft)
	totalSlots := 0
	shiftsWithFixed := make([]struct {
		ID   string
		Name string
	}, 0)

	for shiftID, schedule := range allFixedSchedules {
		if len(schedule) == 0 {
			continue
		}

		fixedDrafts[shiftID] = &d_model.ShiftScheduleDraft{
			Schedule: schedule,
		}

		// 统计槽位数
		for _, staffIDs := range schedule {
			totalSlots += len(staffIDs)
		}

		// 记录有固定配置的班次信息
		if shift, ok := shiftMap[shiftID]; ok && shift != nil {
			shiftsWithFixed = append(shiftsWithFixed, struct {
				ID   string
				Name string
			}{
				ID:   shiftID,
				Name: shift.Name,
			})
		}
	}

	// 保存到上下文
	if createCtx.FixedShiftResults == nil {
		createCtx.FixedShiftResults = &PhaseResult{
			PhaseName:      PhaseFixedShift,
			ShiftType:      ShiftTypeFixed,
			ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
			CompletedCount: 0,
			SkippedCount:   0,
			FailedCount:    0,
			StartTime:      time.Now().Format(time.RFC3339),
		}
	}
	createCtx.FixedShiftResults.ScheduleDrafts = fixedDrafts
	createCtx.FixedShiftResults.CompletedCount = len(fixedDrafts)

	// 更新已占位信息
	for shiftID, draft := range fixedDrafts {
		MergeOccupiedSlots(createCtx.OccupiedSlots, draft, shiftID)
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 格式化消息展示给用户
	message := "🔒 **固定班次自动填充完成**\n\n"
	message += fmt.Sprintf("共处理 **%d** 个班次，填充了 **%d** 个固定排班槽位：\n\n", len(fixedDrafts), totalSlots)

	for i, shiftInfo := range shiftsWithFixed {
		if i < 10 { // 最多显示10个
			schedule, ok := allFixedSchedules[shiftInfo.ID]
			if !ok {
				continue
			}
			daysCount := len(schedule)
			message += fmt.Sprintf("  • %s: %d 天有固定人员\n", shiftInfo.Name, daysCount)
		}
	}

	if len(shiftsWithFixed) > 10 {
		message += fmt.Sprintf("  • ... 还有 %d 个班次\n", len(shiftsWithFixed)-10)
	}

	message += "\n✅ 这些固定人员已自动排入班表，不再参与AI排班。"

	// 构建固定班次排班数据（用于查看详情）
	// 获取员工列表用于转换员工ID为姓名
	// 注意：使用 AllStaffList（所有员工）而不是 StaffList（班次关联员工），确保包含固定排班人员
	staffNames := make(map[string]string)
	staffListForMapping := createCtx.AllStaffList
	if len(staffListForMapping) == 0 {
		// 如果没有 AllStaffList，回退到 StaffList
		staffListForMapping = createCtx.StaffList
	}
	for _, staff := range staffListForMapping {
		staffNames[staff.ID] = staff.Name
	}

	// 构建包含所有固定班次的排班数据
	fixedScheduleData := make(map[string]any)
	shiftsData := make(map[string]any)
	shiftInfoList := make([]map[string]any, 0) // 用于前端显示班次信息
	for shiftID, draft := range fixedDrafts {
		if shift, ok := shiftMap[shiftID]; ok && shift != nil {
			days := make(map[string]any)
			// 获取该班次的人数需求配置
			shiftRequirements := createCtx.StaffRequirements[shiftID]
			if shiftRequirements == nil {
				shiftRequirements = make(map[string]int)
			}

			for date, staffIDs := range draft.Schedule {
				names := make([]string, 0, len(staffIDs))
				for _, id := range staffIDs {
					if name, ok := staffNames[id]; ok {
						names = append(names, name)
					} else {
						names = append(names, id)
					}
				}

				// 从 StaffRequirements 获取该日期的要求人数
				requiredCount := 0
				if req, ok := shiftRequirements[date]; ok {
					requiredCount = req
				}

				days[date] = map[string]any{
					"staff":         names,
					"staffIds":      staffIDs,
					"requiredCount": requiredCount,
					"actualCount":   len(staffIDs),
				}
			}
			shiftsData[shiftID] = map[string]any{
				"shiftId":  shift.ID,
				"priority": shift.SchedulingPriority,
				"days":     days,
			}
			// 保存班次信息供前端使用
			shiftInfoList = append(shiftInfoList, map[string]any{
				"id":   shift.ID,
				"name": shift.Name,
			})
		}
	}
	fixedScheduleData["shifts"] = shiftsData
	fixedScheduleData["startDate"] = createCtx.StartDate
	fixedScheduleData["endDate"] = createCtx.EndDate
	fixedScheduleData["shiftInfoList"] = shiftInfoList // 添加班次信息列表

	// 构建查询按钮（附加到消息上）
	queryAction := session.WorkflowAction{
		ID:      "view_fixed_shifts_detail",
		Type:    session.ActionTypeQuery,
		Label:   "📊 查看详情",
		Payload: fixedScheduleData,
		Style:   session.ActionStyleSuccess, // 绿色图标
	}

	// 发送消息并附加查询按钮
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: []session.WorkflowAction{queryAction},
		Metadata: map[string]any{
			"type":         "fixedShiftSchedule",
			"scheduleData": fixedScheduleData,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		logger.Warn("Failed to send message with query action", "error", err)
	}

	// 构建确认界面（只包含工作流按钮）
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认",
			Event: session.WorkflowEvent(CreateV2EventFixedShiftConfirmed),
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
		"请确认固定班次排班：", workflowActions)
}

// actOnFixedShiftConfirmed 处理固定班次确认
func actOnFixedShiftConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Fixed shifts confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 确保固定班次结果已保存（在 startFixedShiftPhase 中已保存 ScheduleDrafts）
	// 只需要更新确认时间和完成状态
	if createCtx.FixedShiftResults == nil {
		// 如果结果不存在，创建新的（理论上不应该发生）
		createCtx.FixedShiftResults = &PhaseResult{
			PhaseName:      PhaseFixedShift,
			ShiftType:      ShiftTypeFixed,
			ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
			CompletedCount: 0,
			StartTime:      time.Now().Format(time.RFC3339),
		}
	}
	// 更新结束时间
	createCtx.FixedShiftResults.EndTime = time.Now().Format(time.RFC3339)

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 清空工作流 action（进入自动处理阶段）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 开始特殊班次阶段
	return startShiftPhase(ctx, wctx, createCtx, PhaseSpecialShift, ShiftTypeSpecial, CreateV2StateSpecialShift)
}

// actOnPersonalNeedsConfirmedFromFixedShift 在固定班次阶段处理个人需求确认事件
// 这种情况发生在用户从固定班次阶段返回修改需求后再确认
func actOnPersonalNeedsConfirmedFromFixedShift(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Personal needs confirmed from fixed shift phase", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 处理可能的临时需求数据（与 actOnPersonalNeedsConfirmed 相同的逻辑）
	var addNeedPayload *AddTemporaryNeedPayload
	if payload != nil {
		if p, ok := payload.(*AddTemporaryNeedPayload); ok {
			addNeedPayload = p
		} else if p, ok := payload.(AddTemporaryNeedPayload); ok {
			addNeedPayload = &p
		} else if payloadMap, ok := payload.(map[string]any); ok && len(payloadMap) > 0 {
			jsonBytes, err := json.Marshal(payloadMap)
			if err == nil {
				var p AddTemporaryNeedPayload
				if err := json.Unmarshal(jsonBytes, &p); err == nil {
					addNeedPayload = &p
				}
			}
		}
	}

	// 如果有临时需求数据，处理它们
	if addNeedPayload != nil && len(addNeedPayload.Needs) > 0 {
		logger.Info("Processing temporary needs from fixed shift phase", "count", len(addNeedPayload.Needs))

		// 添加临时需求（与 actOnPersonalNeedsConfirmed 相同的逻辑）
		service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
		if !ok {
			return fmt.Errorf("rosteringService not found")
		}

		staffNameMap := make(map[string]string)
		if len(createCtx.StaffList) > 0 {
			for _, staff := range createCtx.StaffList {
				staffNameMap[staff.ID] = staff.Name
			}
		}
		if len(staffNameMap) == 0 {
			staffResult, err := service.ListStaff(ctx, d_model.StaffListFilter{OrgID: sess.OrgID})
			if err == nil && staffResult != nil {
				for _, staff := range staffResult.Items {
					staffNameMap[staff.UserID] = staff.Name
				}
			}
		}

		shiftNameMap := make(map[string]string)
		if createCtx.SelectedShifts != nil {
			for _, shift := range createCtx.SelectedShifts {
				shiftNameMap[shift.ID] = shift.Name
			}
		}

		for _, needItem := range addNeedPayload.Needs {
			if needItem.StaffID == "" || needItem.Description == "" {
				continue
			}

			staffName := staffNameMap[needItem.StaffID]
			if staffName == "" {
				staffName = needItem.StaffID
			}

			newNeed := &PersonalNeed{
				StaffID:         needItem.StaffID,
				StaffName:       staffName,
				NeedType:        "temporary",
				RequestType:     needItem.RequestType,
				TargetShiftID:   needItem.TargetShiftID,
				TargetShiftName: shiftNameMap[needItem.TargetShiftID],
				TargetDates:     needItem.TargetDates,
				Description:     needItem.Description,
				Priority:        needItem.Priority,
				Source:          "user",
				Confirmed:       true,
			}
			createCtx.AddPersonalNeed(newNeed)
		}
	}

	// 标记个人需求已确认
	createCtx.PersonalNeedsConfirmed = true

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送确认消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"✅ 个人需求已确认，继续处理固定班次..."); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 清空工作流 action（进入自动处理阶段）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 重新开始固定班次处理
	return startFixedShiftPhase(ctx, wctx, createCtx)
}

// actSkipFixedShift 跳过固定班次阶段
func actSkipFixedShift(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Skipping fixed shift phase", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
		"⏭️ 跳过固定班次"); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 开始特殊班次阶段
	return startShiftPhase(ctx, wctx, createCtx, PhaseSpecialShift, ShiftTypeSpecial, CreateV2StateSpecialShift)
}

// ============================================================
// 通用：班次排班阶段处理（特殊/普通/科研班次）
// ============================================================

// startShiftPhase 开始班次排班阶段（通用）
func startShiftPhase(
	ctx context.Context,
	wctx engine.Context,
	createCtx *CreateV2Context,
	phaseName string,
	shiftType string,
	targetState engine.State,
) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Starting shift phase",
		"phase", phaseName,
		"shiftType", shiftType,
		"sessionID", sess.ID)

	// 清空工作流 action（进入自动处理阶段），并禁用用户输入
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}
	// 禁用用户输入（排班过程中不允许输入）
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 获取该类型的班次列表
	shifts := GetShiftsByType(createCtx.ClassifiedShifts, shiftType)

	if len(shifts) == 0 {
		// 没有该类型班次，跳过
		logger.Info("CreateV2: No shifts of this type, skipping phase",
			"shiftType", shiftType)

		// 设置当前阶段名称（用于后续阶段完成处理）
		createCtx.CurrentPhase = phaseName
		if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		// 直接触发阶段完成事件
		return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
	}

	// 按优先级排序
	shifts = SortShiftsBySchedulingPriority(shifts)

	// 初始化阶段上下文
	createCtx.ResetPhaseProgress(phaseName, shifts)

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送阶段开始消息
	phaseNameCN := getPhaseNameCN(phaseName)
	message := fmt.Sprintf("🔄 开始排%s", phaseNameCN)

	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 启动第一个班次的排班（调用 Core 子工作流）
	return spawnCurrentShift(ctx, wctx, createCtx)
}

// spawnCurrentShift 启动当前班次的排班子工作流
func spawnCurrentShift(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 检查是否还有班次需要处理
	if createCtx.IsPhaseComplete() {
		// 阶段完成
		logger.Info("CreateV2: Phase complete", "phase", createCtx.CurrentPhase)
		return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
	}

	// 获取当前班次
	currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]

	logger.Info("CreateV2: Spawning shift scheduling",
		"shiftID", currentShift.ID,
		"shiftName", currentShift.Name,
		"index", createCtx.CurrentShiftIndex,
		"total", createCtx.TotalShiftsInPhase)

	// 发送进度消息
	message := fmt.Sprintf("📌 [%d/%d] %s",
		createCtx.CurrentShiftIndex+1,
		createCtx.TotalShiftsInPhase,
		currentShift.Name)

	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 准备 ShiftSchedulingContext
	shiftCtx := prepareShiftSchedulingContext(createCtx, currentShift)

	// 保存到 session data
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "shift_scheduling_context", shiftCtx); err != nil {
		return fmt.Errorf("failed to save shift context: %w", err)
	}

	// 调用 Core 子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor, cannot spawn sub-workflow")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowSchedulingCore,
		Input:        nil, // Core 从 session 读取 ShiftSchedulingContext
		OnComplete:   CreateV2EventShiftCompleted,
		OnError:      CreateV2EventSubFailed,
		Timeout:      10 * 60 * 1e9, // 10 分钟超时 (纳秒)
		SnapshotKeys: []string{
			KeyCreateV2Context,
			"shift_scheduling_context",
		},
	}

	logger.Info("CreateV2: Spawning Core sub-workflow",
		"shiftID", currentShift.ID,
		"shiftName", currentShift.Name,
	)

	return actor.SpawnSubWorkflow(ctx, config)
}

// prepareShiftSchedulingContext 准备班次排班上下文
func prepareShiftSchedulingContext(createCtx *CreateV2Context, shift *d_model.Shift) *d_model.ShiftSchedulingContext {
	// 获取该班次的人数需求
	staffReqs := createCtx.StaffRequirements[shift.ID]
	if staffReqs == nil {
		staffReqs = make(map[string]int)
	}

	// 构建 ShiftSchedulingContext
	shiftCtx := d_model.NewShiftSchedulingContext(
		shift,
		createCtx.StartDate,
		createCtx.EndDate,
		string(WorkflowScheduleCreateV2),
	)

	// 填充详细数据
	// 注意：不过滤人员列表，将所有人员传递给 AI
	// AI 会根据 OccupiedSlots 和 ExistingScheduleMarks 按日期判断每个人员的可用性
	// 一个人在某个日期被占用，只意味着他在那一天不可用，其他日期仍然可用
	shiftCtx.StaffList = createCtx.StaffList
	shiftCtx.AllStaffList = createCtx.AllStaffList // 所有员工列表（用于姓名映射，确保显示正确的姓名而不是UUID）
	shiftCtx.StaffLeaves = createCtx.StaffLeaves
	shiftCtx.GlobalRules = createCtx.Rules         // TODO: 分离全局规则和班次规则
	shiftCtx.ShiftRules = make([]*d_model.Rule, 0) // TODO: 从 Rules 中筛选班次规则
	shiftCtx.ExistingScheduleMarks = createCtx.ExistingScheduleMarks

	// 传递完整需求给AI（包含固定人员）
	// 固定人员信息也会传递给AI，让AI直接生成包含固定人员的完整排班
	shiftCtx.StaffRequirements = staffReqs

	// 提取固定人员信息（如果当前班次有固定人员配置）
	fixedAssignments := make(map[string][]string) // date -> staffIds
	if createCtx.FixedShiftResults != nil && createCtx.FixedShiftResults.ScheduleDrafts != nil {
		if fixedDraft, ok := createCtx.FixedShiftResults.ScheduleDrafts[shift.ID]; ok {
			if fixedDraft != nil && fixedDraft.Schedule != nil {
				for date, staffIDs := range fixedDraft.Schedule {
					// 只提取在排班周期内的日期
					if date >= createCtx.StartDate && date <= createCtx.EndDate {
						fixedAssignments[date] = append([]string{}, staffIDs...)
					}
				}
			}
		}
	}
	shiftCtx.FixedShiftAssignments = fixedAssignments

	// 提取临时需求（从 PersonalNeeds 中，只提取临时需求，NeedType == "temporary"）
	// 临时需求应该被考虑，确保不会安排已出差、请假或有事的员工
	if createCtx.PersonalNeeds != nil {
		temporaryNeeds := make([]*d_model.PersonalNeed, 0)
		for _, needs := range createCtx.PersonalNeeds {
			for _, need := range needs {
				// 只提取临时需求
				if need != nil && need.NeedType == "temporary" {
					// 转换为 d_model.PersonalNeed
					modelNeed := &d_model.PersonalNeed{
						StaffID:         need.StaffID,
						StaffName:       need.StaffName,
						NeedType:        need.NeedType,
						RequestType:     need.RequestType,
						TargetShiftID:   need.TargetShiftID,
						TargetShiftName: need.TargetShiftName,
						TargetDates:     need.TargetDates,
						Description:     need.Description,
						Priority:        need.Priority,
						RuleID:          need.RuleID,
						Source:          need.Source,
						Confirmed:       need.Confirmed,
					}
					temporaryNeeds = append(temporaryNeeds, modelNeed)
				}
			}
		}
		shiftCtx.TemporaryNeeds = temporaryNeeds
	}

	return shiftCtx
}

// ============================================================
// 班次完成和阶段完成处理
// ============================================================

// actOnShiftCompleted 处理单个班次完成
func actOnShiftCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Shift completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取班次排班结果（从 Core 子工作流返回的 ShiftSchedulingContext）
	shiftCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, "shift_scheduling_context")
	if err == nil && found {
		// 解析 ShiftSchedulingContext
		var shiftCtx d_model.ShiftSchedulingContext
		dataBytes, err := json.Marshal(shiftCtxData)
		if err == nil {
			if err := json.Unmarshal(dataBytes, &shiftCtx); err == nil && shiftCtx.ResultDraft != nil {
				// 获取当前班次
				if createCtx.CurrentShiftIndex < len(createCtx.PhaseShiftList) {
					currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]
					shiftID := currentShift.ID

					// 从 ResultDraft 提取排班结果
					// ResultDraft 是 ShiftScheduleDraft，直接使用
					scheduleDraft := shiftCtx.ResultDraft
					if scheduleDraft == nil || scheduleDraft.Schedule == nil || len(scheduleDraft.Schedule) == 0 {
						// 如果没有结果，跳过
						logger.Warn("No schedule result for shift", "shiftID", shiftID)
					} else {
						// 保存到对应阶段的 PhaseResult
						var phaseResult *PhaseResult
						switch createCtx.CurrentPhase {
						case PhaseSpecialShift:
							if createCtx.SpecialShiftResults == nil {
								createCtx.SpecialShiftResults = &PhaseResult{
									PhaseName:      PhaseSpecialShift,
									ShiftType:      ShiftTypeSpecial,
									ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
									StartTime:      time.Now().Format(time.RFC3339),
								}
							}
							phaseResult = createCtx.SpecialShiftResults
						case PhaseNormalShift:
							if createCtx.NormalShiftResults == nil {
								createCtx.NormalShiftResults = &PhaseResult{
									PhaseName:      PhaseNormalShift,
									ShiftType:      ShiftTypeNormal,
									ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
									StartTime:      time.Now().Format(time.RFC3339),
								}
							}
							phaseResult = createCtx.NormalShiftResults
						case PhaseResearchShift:
							if createCtx.ResearchShiftResults == nil {
								createCtx.ResearchShiftResults = &PhaseResult{
									PhaseName:      PhaseResearchShift,
									ShiftType:      ShiftTypeResearch,
									ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
									StartTime:      time.Now().Format(time.RFC3339),
								}
							}
							phaseResult = createCtx.ResearchShiftResults
						}

						if phaseResult != nil {
							if phaseResult.ScheduleDrafts == nil {
								phaseResult.ScheduleDrafts = make(map[string]*d_model.ShiftScheduleDraft)
							}

							// AI已经生成了包含固定人员的完整排班，直接使用
							// 无需合并逻辑，因为AI已经知道固定人员并包含在结果中
							logger.Info("CreateV2: Using AI schedule result directly (includes fixed staff)",
								"shiftID", shiftID,
								"scheduleDates", len(scheduleDraft.Schedule))

							// 验证排班结果
							fixedAssignments := make(map[string][]string)
							if createCtx.FixedShiftResults != nil && createCtx.FixedShiftResults.ScheduleDrafts != nil {
								if fixedDraft, ok := createCtx.FixedShiftResults.ScheduleDrafts[shiftID]; ok {
									if fixedDraft != nil && fixedDraft.Schedule != nil {
										for date, staffIDs := range fixedDraft.Schedule {
											if date >= createCtx.StartDate && date <= createCtx.EndDate {
												fixedAssignments[date] = append([]string{}, staffIDs...)
											}
										}
									}
								}
							}
							staffReqs := createCtx.StaffRequirements[shiftID]

							isValid, issues := validateScheduleResult(scheduleDraft, fixedAssignments, staffReqs, shiftID, createCtx.AllStaffList)

							// 记录是否执行了自动修正（在外部作用域声明，以便在后续代码中使用）
							var hasAutoCorrection bool
							var adjustResult *d_model.AdjustScheduleResult
							var originalIssues []string

							// 如果验证失败，尝试自动修正
							if !isValid {
								// 保存原始问题列表
								originalIssues = make([]string, len(issues))
								copy(originalIssues, issues)

								logger.Warn("CreateV2: Schedule validation failed, attempting auto-correction",
									"shiftID", shiftID,
									"issues", issues)

								// 准备修正所需的数据
								shiftInfo := &d_model.ShiftInfo{
									ShiftID:   shiftID,
									ShiftName: currentShift.Name,
									StartDate: createCtx.StartDate,
									EndDate:   createCtx.EndDate,
								}
								staffList := d_model.NewStaffInfoListFromEmployees(createCtx.StaffList)
								// 转换规则
								ruleInfos := d_model.NewRuleInfoListFromRules(createCtx.Rules)
								// 转换ExistingScheduleMarks从map[string]map[string][]*ShiftMark到map[string]map[string]bool
								existingMarksBool := make(map[string]map[string]bool)
								for staffID, dateMarks := range createCtx.ExistingScheduleMarks {
									if dateMarks != nil {
										existingMarksBool[staffID] = make(map[string]bool)
										for date := range dateMarks {
											existingMarksBool[staffID][date] = true
										}
									}
								}

								// 调用自动修正
								var err error
								adjustResult, err = autoCorrectSchedule(
									ctx,
									wctx,
									scheduleDraft,
									issues,
									fixedAssignments,
									staffReqs,
									shiftInfo,
									staffList,
									ruleInfos,
									createCtx.AllStaffList,
									existingMarksBool,
								)

								if err != nil {
									logger.Error("CreateV2: Auto-correction failed", "error", err)
									// 修正失败，标记为失败
									phaseResult.FailedCount++
									phaseResult.ScheduleDrafts[shiftID] = scheduleDraft // 保存原始结果
									// 发送失败消息
									return handleShiftFailed(ctx, wctx, createCtx, shiftID, currentShift.Name, issues, scheduleDraft, "自动修正失败")
								}

								// 使用修正后的结果
								scheduleDraft = adjustResult.Draft
								hasAutoCorrection = true

								// 记录修正后的排班详情（用于调试）
								logger.Info("CreateV2: Auto-correction completed, re-validating",
									"shiftID", shiftID,
									"summary", adjustResult.Summary,
									"correctedDraftDates", len(scheduleDraft.Schedule))

								// 记录每个日期的人数（用于调试）
								for date, reqCount := range staffReqs {
									if reqCount > 0 {
										actualCount := 0
										if staffIDs, exists := scheduleDraft.Schedule[date]; exists {
											actualCount = len(staffIDs)
										}
										logger.Info("CreateV2: Corrected schedule date check",
											"date", date,
											"required", reqCount,
											"actual", actualCount,
											"match", actualCount == reqCount)
									}
								}

								// 再次验证修正后的结果
								isValid, issues = validateScheduleResult(scheduleDraft, fixedAssignments, staffReqs, shiftID, createCtx.AllStaffList)
								if !isValid {
									logger.Warn("CreateV2: Schedule still invalid after auto-correction",
										"shiftID", shiftID,
										"issues", issues)
									// 修正后仍失败，标记为失败
									phaseResult.FailedCount++
									phaseResult.ScheduleDrafts[shiftID] = scheduleDraft // 保存修正后的结果
									// 发送失败消息
									return handleShiftFailed(ctx, wctx, createCtx, shiftID, currentShift.Name, issues, scheduleDraft, "修正后仍不满足要求")
								}

								logger.Info("CreateV2: Auto-correction successful, schedule is now valid",
									"shiftID", shiftID)

								// 发送自动修正成功的消息
								var correctionMsg strings.Builder
								correctionMsg.WriteString(fmt.Sprintf("### ✅ 班次【%s】排班已自动修正\n\n", currentShift.Name))
								correctionMsg.WriteString("**发现的问题**：\n")
								for i, issue := range originalIssues {
									correctionMsg.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
								}
								correctionMsg.WriteString("\n**自动修正说明**：\n")
								correctionMsg.WriteString(adjustResult.Summary)
								if len(adjustResult.Changes) > 0 {
									correctionMsg.WriteString("\n\n**主要变化**：\n")
									for i, change := range adjustResult.Changes {
										if i >= 10 { // 最多显示10个变化
											correctionMsg.WriteString(fmt.Sprintf("\n... 还有 %d 个变化\n", len(adjustResult.Changes)-10))
											break
										}
										correctionMsg.WriteString(fmt.Sprintf("%d. %s\n", i+1, formatAdjustChange(&change, createCtx.AllStaffList)))
									}
								}
								correctionMsg.WriteString("\n**修正结果**：排班已修正并通过验证，可以继续。")

								if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, correctionMsg.String()); err != nil {
									logger.Warn("Failed to send auto-correction success message", "error", err)
								}
							} else {
								// 如果验证通过且没有执行自动修正，发送验证通过消息
								validationMsg := fmt.Sprintf("### ✅ 班次【%s】排班验证通过\n\n排班结果符合要求，可以继续。", currentShift.Name)
								if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, validationMsg); err != nil {
									logger.Warn("Failed to send validation success message", "error", err)
								}
							}

							// 保存AI排班结果（已包含固定人员，且已验证）
							phaseResult.ScheduleDrafts[shiftID] = scheduleDraft

							// 更新已占位信息
							MergeOccupiedSlots(createCtx.OccupiedSlots, scheduleDraft, shiftID)

							// 更新现有排班标记（用于时段冲突检查）
							// 合并所有阶段的 ScheduleDrafts 来构建标记
							allDrafts := make(map[string]*d_model.ShiftScheduleDraft)
							if createCtx.FixedShiftResults != nil && createCtx.FixedShiftResults.ScheduleDrafts != nil {
								for k, v := range createCtx.FixedShiftResults.ScheduleDrafts {
									allDrafts[k] = v
								}
							}
							if createCtx.SpecialShiftResults != nil && createCtx.SpecialShiftResults.ScheduleDrafts != nil {
								for k, v := range createCtx.SpecialShiftResults.ScheduleDrafts {
									allDrafts[k] = v
								}
							}
							if createCtx.NormalShiftResults != nil && createCtx.NormalShiftResults.ScheduleDrafts != nil {
								for k, v := range createCtx.NormalShiftResults.ScheduleDrafts {
									allDrafts[k] = v
								}
							}
							if createCtx.ResearchShiftResults != nil && createCtx.ResearchShiftResults.ScheduleDrafts != nil {
								for k, v := range createCtx.ResearchShiftResults.ScheduleDrafts {
									allDrafts[k] = v
								}
							}
							// 添加当前班次的结果
							allDrafts[shiftID] = scheduleDraft

							// 构建班次 map
							shiftMap := make(map[string]*d_model.Shift)
							for _, shift := range createCtx.SelectedShifts {
								shiftMap[shift.ID] = shift
							}

							// 重新构建所有标记
							createCtx.ExistingScheduleMarks = BuildExistingScheduleMarks(allDrafts, shiftMap)

							// 如果执行了自动修正，保存修正信息到独立的 session data 中（用于前端显示）
							if hasAutoCorrection && adjustResult != nil {
								autoCorrectionInfo := map[string]any{
									"summary": adjustResult.Summary,
									"changes": adjustResult.Changes,
									"issues":  originalIssues, // 原始问题列表
									"shiftID": shiftID,
								}
								// 保存到独立的 session data key
								if _, err := wctx.SessionService().SetData(ctx, sess.ID, fmt.Sprintf("auto_correction_%s", shiftID), autoCorrectionInfo); err != nil {
									logger.Warn("Failed to save auto-correction info to session", "error", err)
								}
							}
						}
					}
				}
			}
		}
	}

	// 增加完成计数
	createCtx.CompletedShiftCount++

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 检查连续排班配置
	rosteringService, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	continuousScheduling := false // 默认关闭连续排班（更安全）
	if serviceOk {
		continuousSetting, err := rosteringService.GetSystemSetting(ctx, sess.OrgID, "continuous_scheduling")
		if err != nil {
			// 记录详细错误信息以便排查
			logger.Error("Failed to get continuous_scheduling setting",
				"error", err,
				"orgID", sess.OrgID,
				"serviceAvailable", true)
			// 如果获取失败，使用默认值（false，非连续排班）
			continuousScheduling = false
		} else {
			continuousScheduling = (continuousSetting == "true")
			logger.Info("Got continuous_scheduling setting",
				"value", continuousSetting,
				"enabled", continuousScheduling,
				"orgID", sess.OrgID)
		}
	} else {
		logger.Error("Rostering service not available in service registry",
			"orgID", sess.OrgID,
			"serviceKey", engine.ServiceKeyRostering)
		continuousScheduling = false
	}

	// 如果关闭连续排班，进入审核状态
	if !continuousScheduling {
		logger.Info("CreateV2: Continuous scheduling disabled, entering review state")
		// 发送事件进入审核状态（通过状态转换）
		// 注意：enterShiftReviewState 会在审核状态的 Act 中被调用
		// 这里只发送事件触发状态转换
		return wctx.Send(ctx, CreateV2EventEnterShiftReview, nil)
	}

	// 连续排班开启，自动继续（通过 AfterAct 触发）
	return nil
}

// actSpawnNextShiftOrComplete 启动下一个班次或完成阶段（AfterAct）
func actSpawnNextShiftOrComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()

	// 检查 context 是否已取消（子工作流超时后可能发生）
	// 如果已取消，使用 context.Background() 继续执行
	workCtx := ctx
	select {
	case <-ctx.Done():
		logger.Warn("CreateV2: Context cancelled in AfterAct, using background context",
			"error", ctx.Err())
		workCtx = context.Background()
	default:
		// Context 正常，继续使用
	}

	createCtx, err := loadCreateV2Context(workCtx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 检查连续排班配置（在启动下一个班次前）
	sess := wctx.Session()
	rosteringService, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	continuousScheduling := false // 默认关闭连续排班（更安全）
	if serviceOk {
		continuousSetting, err := rosteringService.GetSystemSetting(workCtx, sess.OrgID, "continuous_scheduling")
		if err != nil {
			// 记录详细错误信息以便排查
			logger.Error("Failed to get continuous_scheduling setting in AfterAct",
				"error", err,
				"orgID", sess.OrgID,
				"serviceAvailable", true)
			// 如果获取失败，使用默认值（false，非连续排班）
			continuousScheduling = false
		} else {
			continuousScheduling = (continuousSetting == "true")
			logger.Info("Got continuous_scheduling setting in AfterAct",
				"value", continuousSetting,
				"enabled", continuousScheduling,
				"orgID", sess.OrgID)
		}
	} else {
		logger.Error("Rostering service not available in service registry (AfterAct)",
			"orgID", sess.OrgID,
			"serviceKey", engine.ServiceKeyRostering)
		continuousScheduling = false
	}

	// 如果关闭连续排班，不自动启动下一个班次（应该在审核状态）
	if !continuousScheduling {
		logger.Info("CreateV2: Continuous scheduling disabled, skipping auto-spawn")
		return nil
	}

	// 移动到下一个班次
	createCtx.IncrementPhaseProgress()

	if err := saveCreateV2Context(workCtx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 检查是否还有班次
	if createCtx.IsPhaseComplete() {
		// 阶段完成
		logger.Info("CreateV2: Phase complete", "phase", createCtx.CurrentPhase)
		return wctx.Send(workCtx, CreateV2EventShiftPhaseComplete, nil)
	}

	// 启动下一个班次
	return spawnCurrentShift(workCtx, wctx, createCtx)
}

// actOnShiftFailed 处理班次排班失败
func actOnShiftFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Warn("CreateV2: Shift failed", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 增加失败计数
	createCtx.FailedShiftCount++

	// 标记当前班次为跳过
	if createCtx.CurrentShiftIndex < len(createCtx.PhaseShiftList) {
		currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]
		logger.Warn("CreateV2: Skipping failed shift",
			"shiftID", currentShift.ID,
			"shiftName", currentShift.Name)

		// 发送提示消息
		message := fmt.Sprintf("⚠️ 班次 %s 排班失败，已跳过。", currentShift.Name)
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send message", "error", err)
		}
	}

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// actOnPhaseComplete 处理阶段完成
func actOnPhaseComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	phaseName := createCtx.CurrentPhase
	logger.Info("CreateV2: Phase completed", "phase", phaseName, "sessionID", sess.ID)

	// 如果 phaseName 为空，尝试从 session 的 workflow meta 中获取当前状态
	if phaseName == "" {
		sess := wctx.Session()
		if sess != nil && sess.WorkflowMeta != nil {
			// 从 workflow meta 的 Phase 字段获取当前状态
			currentState := engine.State(sess.WorkflowMeta.Phase)
			switch currentState {
			case CreateV2StateSpecialShift:
				phaseName = PhaseSpecialShift
			case CreateV2StateNormalShift:
				phaseName = PhaseNormalShift
			case CreateV2StateResearchShift:
				phaseName = PhaseResearchShift
			default:
				logger.Warn("CreateV2: Cannot determine phase from workflow meta", "state", currentState)
				return fmt.Errorf("unknown phase: empty phase name in context and cannot infer from state: %s", currentState)
			}
			// 更新上下文中的阶段名称
			createCtx.CurrentPhase = phaseName
		} else {
			logger.Warn("CreateV2: Phase name is empty and cannot access session workflow meta")
			return fmt.Errorf("unknown phase: empty phase name in context")
		}
	}

	// 保存阶段结果
	phaseResult := &PhaseResult{
		PhaseName:      phaseName,
		CompletedCount: createCtx.CompletedShiftCount,
		SkippedCount:   createCtx.SkippedShiftCount,
		FailedCount:    createCtx.FailedShiftCount,
		EndTime:        time.Now().Format(time.RFC3339),
	}

	// 根据阶段保存结果
	switch phaseName {
	case PhaseSpecialShift:
		createCtx.SpecialShiftResults = phaseResult
	case PhaseNormalShift:
		createCtx.NormalShiftResults = phaseResult
	case PhaseResearchShift:
		createCtx.ResearchShiftResults = phaseResult
	}

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送阶段完成消息
	phaseNameCN := getPhaseNameCN(phaseName)
	message := fmt.Sprintf("✅ %s完成", phaseNameCN)

	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	// 根据当前阶段确定下一阶段
	return startNextPhase(ctx, wctx, createCtx, phaseName)
}

// startNextPhase 启动下一阶段
func startNextPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context, currentPhase string) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 清空工作流 action（进入自动处理阶段）
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	switch currentPhase {
	case PhaseSpecialShift:
		return startShiftPhase(ctx, wctx, createCtx, PhaseNormalShift, ShiftTypeNormal, CreateV2StateNormalShift)
	case PhaseNormalShift:
		return startShiftPhase(ctx, wctx, createCtx, PhaseResearchShift, ShiftTypeResearch, CreateV2StateResearchShift)
	case PhaseResearchShift:
		return startFillShiftPhase(ctx, wctx, createCtx)
	default:
		return fmt.Errorf("unknown phase: %s", currentPhase)
	}
}

// actSkipPhase 跳过当前阶段
func actSkipPhase(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	phaseName := createCtx.CurrentPhase
	logger.Info("CreateV2: Skipping phase", "phase", phaseName)

	phaseNameCN := getPhaseNameCN(phaseName)
	message := fmt.Sprintf("⏭️ 已跳过%s。", phaseNameCN)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return startNextPhase(ctx, wctx, createCtx, phaseName)
}

// ============================================================
// 后续阶段实现将在下一部分继续...
// ============================================================

// actOnFillShiftComplete 处理填充班次完成
func actOnFillShiftComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Fill shift phase completed, preparing to save", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 1. 合并所有阶段的 ScheduleDrafts
	allDrafts := make(map[string]*d_model.ShiftScheduleDraft)
	phaseResults := []*PhaseResult{
		createCtx.FixedShiftResults,
		createCtx.SpecialShiftResults,
		createCtx.NormalShiftResults,
		createCtx.ResearchShiftResults,
		createCtx.FillShiftResults,
	}

	for _, phaseResult := range phaseResults {
		if phaseResult != nil && phaseResult.ScheduleDrafts != nil {
			for shiftID, draft := range phaseResult.ScheduleDrafts {
				allDrafts[shiftID] = draft
			}
		}
	}

	if len(allDrafts) == 0 {
		return fmt.Errorf("no schedule drafts to save")
	}

	// 2. 转换为 ScheduleDraft 格式（参考 convertShiftResultsToScheduleDraft）
	scheduleDraft := convertCreateV2DraftsToScheduleDraft(createCtx, allDrafts)

	// 3. 保存到 ScheduleCreateContext.DraftSchedule
	scheduleCtx := common.GetOrCreateScheduleContext(sess)
	scheduleCtx.DraftSchedule = scheduleDraft
	scheduleCtx.StartDate = createCtx.StartDate
	scheduleCtx.EndDate = createCtx.EndDate
	scheduleCtx.SelectedShifts = createCtx.SelectedShifts

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save schedule context: %w", err)
	}

	logger.Info("CreateV2: Converted drafts to ScheduleDraft",
		"shiftCount", len(scheduleDraft.Shifts),
		"staffCount", len(scheduleDraft.StaffStats))

	// 4. 启动 ConfirmSave 子工作流（在 AfterAct 中执行）
	// 注意：这里不直接启动，而是通过状态转换后的 AfterAct 启动
	return nil
}

// actSpawnConfirmSaveWorkflow 启动 ConfirmSave 子工作流（AfterAct）
func actSpawnConfirmSaveWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Spawning ConfirmSave sub-workflow", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 构建 ConfirmSave 子工作流输入
	inputMap := map[string]any{
		"source_type":   "create_v2",
		"start_date":    createCtx.StartDate,
		"end_date":      createCtx.EndDate,
		"total_shifts":  len(createCtx.SelectedShifts),
		"skipped_count": createCtx.SkippedShiftCount,
	}

	// 启动确认保存子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor, cannot spawn sub-workflow")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowConfirmSave,
		Input:        inputMap,
		OnComplete:   CreateV2EventSaveCompleted,
		OnError:      CreateV2EventSubFailed,
		Timeout:      0, // 无超时，等待用户确认
	}

	logger.Info("CreateV2: Spawning ConfirmSave sub-workflow")
	return actor.SpawnSubWorkflow(ctx, config)
}

// startFillShiftPhase 开始填充班次阶段
func startFillShiftPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	// TODO: 实现填充班次阶段
	// 临时：直接触发阶段完成
	return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
}

// actModifyFillShifts 修改填充班次
func actModifyFillShifts(ctx context.Context, wctx engine.Context, payload any) error {
	// TODO: 实现
	return nil
}

// actSkipFillShift 跳过填充班次
func actSkipFillShift(ctx context.Context, wctx engine.Context, payload any) error {
	// TODO: 实现
	return nil
}

// actOnSaveCompleted 处理保存完成
func actOnSaveCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Save completed", "sessionID", sess.ID)

	// 解析子工作流结果
	result, ok := payload.(*engine.SubWorkflowResult)
	if !ok {
		return fmt.Errorf("invalid payload type for save completed event")
	}

	// 从子工作流结果中获取保存结果
	var savedCount, failedCount int
	if result.Output != nil {
		if output, ok := result.Output["output"]; ok {
			if outputMap, ok := output.(map[string]any); ok {
				if u, ok := outputMap["savedCount"].(int); ok {
					savedCount = u
				}
				if f, ok := outputMap["failedCount"].(int); ok {
					failedCount = f
				}
			} else if confirmOutput, ok := output.(*ConfirmSaveOutput); ok {
				savedCount = confirmOutput.SavedCount
				failedCount = confirmOutput.FailedCount
			}
		}
	}

	// 发送完成消息
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		logger.Warn("Failed to load context for completion message", "error", err)
	} else {
		successMsg := fmt.Sprintf("### ✅ 排班保存成功\n\n"+
			"**排班周期**：%s 至 %s\n"+
			"**涉及班次**：%d 个\n"+
			"**排班记录**：%d 条（成功：%d，失败：%d）\n\n"+
			"您可以在排班日历中查看详情。",
			createCtx.StartDate, createCtx.EndDate,
			len(createCtx.SelectedShifts), savedCount+failedCount, savedCount, failedCount)

		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, successMsg); err != nil {
			logger.Warn("Failed to send success message", "error", err)
		}
	}

	logger.Info("CreateV2: Schedule saved successfully",
		"savedCount", savedCount,
		"failedCount", failedCount)

	return nil
}

// actModifyBeforeSave 保存前修改
func actModifyBeforeSave(ctx context.Context, wctx engine.Context, payload any) error {
	// TODO: 实现
	return nil
}

// actUserCancel 用户取消
func actUserCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: User cancelled", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"❌ 排班已取消。"); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return nil
}

// actOnSubCancelled 子工作流取消
func actOnSubCancelled(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Warn("CreateV2: Subworkflow cancelled", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"⚠️ 子流程已取消，排班终止。"); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return nil
}

// actOnSubFailed 子工作流失败
func actOnSubFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("CreateV2: Subworkflow failed", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"❌ 子流程执行失败，排班终止。"); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return nil
}

// actHandleError 处理错误
func actHandleError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("CreateV2: Error occurred", "sessionID", sess.ID, "payload", payload)

	errorMsg := "系统错误，排班失败。"
	if payload != nil {
		errorMsg = fmt.Sprintf("系统错误：%v", payload)
	}

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); err != nil {
		logger.Warn("Failed to send message", "error", err)
	}

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// loadCreateV2Context 从 session 加载工作流上下文
func loadCreateV2Context(ctx context.Context, wctx engine.Context) (*CreateV2Context, error) {
	sess := wctx.Session()

	data, found, err := wctx.SessionService().GetData(ctx, sess.ID, KeyCreateV2Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get context data: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("context not found")
	}

	var createCtx CreateV2Context
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &createCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &createCtx, nil
}

// saveCreateV2Context 保存工作流上下文到 session
func saveCreateV2Context(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	sess := wctx.Session()
	_, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCreateV2Context, createCtx)
	return err
}

// populateInfoFromSubWorkflow 从服务直接获取数据
func populateInfoFromSubWorkflow(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 尝试从 session 获取时间范围（如果上下文还没有）
	if createCtx.StartDate == "" || createCtx.EndDate == "" {
		if data, found, err := wctx.SessionService().GetData(ctx, sess.ID, "schedule_period"); err == nil && found {
			if period, ok := data.(map[string]any); ok {
				if startDate, ok := period["startDate"].(string); ok {
					createCtx.StartDate = startDate
				}
				if endDate, ok := period["endDate"].(string); ok {
					createCtx.EndDate = endDate
				}
			}
		}
	}

	// 提取已选班次ID（如果已有）
	shiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		shiftIDs = append(shiftIDs, shift.ID)
	}

	// 使用公共方法一键加载基础信息
	basicCtx, err := common.LoadScheduleBasicContext(
		ctx,
		wctx,
		sess.OrgID,
		createCtx.StartDate,
		createCtx.EndDate,
		shiftIDs, // 如果为空，则加载所有激活的班次
	)
	if err != nil {
		return fmt.Errorf("failed to load basic context: %w", err)
	}

	// 填充到 createCtx
	createCtx.StartDate = basicCtx.StartDate
	createCtx.EndDate = basicCtx.EndDate
	createCtx.SelectedShifts = basicCtx.SelectedShifts
	createCtx.StaffList = basicCtx.StaffList       // 班次关联的员工（用于AI排班）
	createCtx.AllStaffList = basicCtx.AllStaffList // 所有员工（用于信息检索）
	createCtx.StaffLeaves = basicCtx.StaffLeaves
	createCtx.StaffRequirements = basicCtx.StaffRequirements
	createCtx.Rules = basicCtx.Rules

	logger.Info("Info collection completed",
		"startDate", createCtx.StartDate,
		"endDate", createCtx.EndDate,
		"shifts", len(createCtx.SelectedShifts),
		"staff", len(createCtx.StaffList),
		"rules", len(createCtx.Rules))

	return nil
}

// BoolPtr 返回 bool 指针
func BoolPtr(b bool) *bool {
	return &b
}

// ============================================================
// 人数配置相关辅助函数
// ============================================================

// buildStaffCountFields 构建人员数量配置表单字段
func buildStaffCountFields(createCtx *CreateV2Context) []session.WorkflowActionField {
	fields := make([]session.WorkflowActionField, 0)

	for _, shift := range createCtx.SelectedShifts {
		// 构建默认值（map[date]count 格式）
		defaultValue := make(map[string]int)
		if reqs, ok := createCtx.StaffRequirements[shift.ID]; ok {
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
						"startDate":  createCtx.StartDate,
						"endDate":    createCtx.EndDate,
					},
				},
			},
		})
	}

	return fields
}

// parseStaffCountPayload 解析人数配置 payload
func parseStaffCountPayload(payloadMap map[string]any, createCtx *CreateV2Context) error {
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

	// 确保 StaffRequirements 已初始化
	if createCtx.StaffRequirements == nil {
		createCtx.StaffRequirements = make(map[string]map[string]int)
	}

	// 解析日期范围
	startDate, startErr := time.Parse("2006-01-02", createCtx.StartDate)
	endDate, endErr := time.Parse("2006-01-02", createCtx.EndDate)

	for _, shift := range createCtx.SelectedShifts {
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
				createCtx.StaffRequirements[shift.ID] = shiftReqs
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
				createCtx.StaffRequirements[shift.ID] = shiftReqs
			}
		}
	}

	return nil
}

// getPhaseNameCN 获取阶段中文名称
func getPhaseNameCN(phase string) string {
	phaseNames := map[string]string{
		PhaseInfoCollect:   "信息收集",
		PhasePersonalNeed:  "个人需求",
		PhaseFixedShift:    "固定班次",
		PhaseSpecialShift:  "特殊班次",
		PhaseNormalShift:   "普通班次",
		PhaseResearchShift: "科研班次",
		PhaseFillShift:     "填充班次",
		PhaseConfirmSave:   "确认保存",
	}

	if name, ok := phaseNames[phase]; ok {
		return name
	}
	return phase
}

// ============================================================
// 班次审核相关函数（非连续排班模式）
// ============================================================

// actEnterShiftReviewState 进入班次审核状态（状态转换的 Act）
func actEnterShiftReviewState(ctx context.Context, wctx engine.Context, payload any) error {
	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	return enterShiftReviewState(ctx, wctx, createCtx)
}

// enterShiftReviewState 进入班次审核状态（内部实现）
func enterShiftReviewState(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取当前班次信息
	if createCtx.CurrentShiftIndex >= len(createCtx.PhaseShiftList) {
		return fmt.Errorf("invalid shift index")
	}
	currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]
	shiftID := currentShift.ID

	// 获取排班结果（优先从 phaseResult 中获取合并后的结果，包含固定人员）
	var scheduleDraft *d_model.ShiftScheduleDraft
	var adjustChanges []d_model.AdjustScheduleChange

	// 首先尝试从 phaseResult 中获取合并后的结果（包含固定人员）
	var phaseResult *PhaseResult
	switch createCtx.CurrentPhase {
	case PhaseSpecialShift:
		phaseResult = createCtx.SpecialShiftResults
	case PhaseNormalShift:
		phaseResult = createCtx.NormalShiftResults
	case PhaseResearchShift:
		phaseResult = createCtx.ResearchShiftResults
	}

	if phaseResult != nil && phaseResult.ScheduleDrafts != nil {
		if mergedDraft, ok := phaseResult.ScheduleDrafts[shiftID]; ok && mergedDraft != nil {
			scheduleDraft = mergedDraft
			logger.Info("CreateV2: Using merged draft from phaseResult (includes fixed staff)", "shiftID", shiftID)
		}
	}

	// 如果 phaseResult 中没有，则从 shift_scheduling_context 中获取（用于调整场景）
	if scheduleDraft == nil {
		shiftCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, "shift_scheduling_context")
		if err == nil && found {
			var shiftCtx d_model.ShiftSchedulingContext
			dataBytes, err := json.Marshal(shiftCtxData)
			if err == nil {
				if err := json.Unmarshal(dataBytes, &shiftCtx); err == nil && shiftCtx.ResultDraft != nil {
					scheduleDraft = shiftCtx.ResultDraft
					logger.Info("CreateV2: Using draft from shift_scheduling_context (adjustment scenario)", "shiftID", shiftID)
				}
			}
		}
	}

	// 获取调整变化信息（如果有）
	adjustMetadataData, found, err := wctx.SessionService().GetData(ctx, sess.ID, "shift_scheduling_adjust_metadata")
	if err == nil && found {
		if metadataMap, ok := adjustMetadataData.(map[string]any); ok {
			if changesRaw, ok := metadataMap["adjust_changes"]; ok {
				if changes, ok := changesRaw.([]d_model.AdjustScheduleChange); ok {
					adjustChanges = changes
				} else if changesArr, ok := changesRaw.([]any); ok {
					// 尝试从 []any 转换
					adjustChanges = make([]d_model.AdjustScheduleChange, 0, len(changesArr))
					for _, item := range changesArr {
						if changeMap, ok := item.(map[string]any); ok {
							change := d_model.AdjustScheduleChange{}
							if date, ok := changeMap["date"].(string); ok {
								change.Date = date
							}
							if added, ok := changeMap["added"].([]any); ok {
								change.Added = make([]string, 0, len(added))
								for _, id := range added {
									if idStr, ok := id.(string); ok {
										change.Added = append(change.Added, idStr)
									}
								}
							}
							if removed, ok := changeMap["removed"].([]any); ok {
								change.Removed = make([]string, 0, len(removed))
								for _, id := range removed {
									if idStr, ok := id.(string); ok {
										change.Removed = append(change.Removed, idStr)
									}
								}
							}
							adjustChanges = append(adjustChanges, change)
						}
					}
				}
			}
		}
	}

	// 构建变化映射（用于快速查找）
	dateChanged := make(map[string]bool)
	staffAdded := make(map[string]bool)   // date -> staffID -> true
	staffRemoved := make(map[string]bool) // date -> staffID -> true
	for _, change := range adjustChanges {
		dateChanged[change.Date] = true
		for _, id := range change.Added {
			staffAdded[change.Date+"_"+id] = true
		}
		for _, id := range change.Removed {
			staffRemoved[change.Date+"_"+id] = true
		}
	}

	// 构建审核消息
	message := fmt.Sprintf("✅ 班次「%s」排班完成\n\n", currentShift.Name)
	if scheduleDraft != nil && len(scheduleDraft.Schedule) > 0 {
		message += "排班结果：\n"
		for date, staffIDs := range scheduleDraft.Schedule {
			message += fmt.Sprintf("- %s: %d人\n", date, len(staffIDs))
		}
		message += "\n"
	}
	message += "请选择下一步操作："

	// 构建排班详情数据（用于查看详情按钮）
	var scheduleDetailPayload map[string]any
	if scheduleDraft != nil && len(scheduleDraft.Schedule) > 0 {
		// 构建员工ID到姓名的映射
		staffNameMap := make(map[string]string)
		for _, staff := range createCtx.AllStaffList {
			staffNameMap[staff.ID] = staff.Name
		}

		// 构建排班详情数据（格式与固定班次类似，但只有一个班次）
		daysData := make(map[string]map[string]any)
		for date, staffIDs := range scheduleDraft.Schedule {
			staffNames := make([]string, 0, len(staffIDs))
			staffFlags := make([]map[string]any, 0, len(staffIDs)) // 用于标记新增/移除
			for _, staffID := range staffIDs {
				if name, ok := staffNameMap[staffID]; ok {
					staffNames = append(staffNames, name)
				} else {
					staffNames = append(staffNames, staffID) // 如果找不到姓名，使用ID
				}
				// 标记是否为新增的人员
				isAdded := staffAdded[date+"_"+staffID]
				staffFlags = append(staffFlags, map[string]any{
					"isAdded": isAdded,
				})
			}

			// 获取该日期的人数需求
			requiredCount := 0
			if shiftReqs := createCtx.StaffRequirements[currentShift.ID]; shiftReqs != nil {
				if count, ok := shiftReqs[date]; ok {
					requiredCount = count
				}
			}

			daysData[date] = map[string]any{
				"staff":         staffNames,
				"staffIds":      staffIDs,
				"staffFlags":    staffFlags,
				"requiredCount": requiredCount,
				"actualCount":   len(staffIDs),
				"isChanged":     dateChanged[date], // 标记该日期是否有变化
			}
		}

		// 添加被移除的人员信息（从 adjustChanges 中提取）
		for _, change := range adjustChanges {
			if len(change.Removed) > 0 {
				removedNames := make([]string, 0, len(change.Removed))
				for _, id := range change.Removed {
					if name, ok := staffNameMap[id]; ok {
						removedNames = append(removedNames, name)
					} else {
						removedNames = append(removedNames, id) // 如果找不到姓名，使用ID
					}
				}
				if dayData, ok := daysData[change.Date]; ok {
					dayData["removedStaff"] = removedNames
					dayData["removedStaffIds"] = change.Removed
				} else {
					// 如果该日期不在当前排班中（可能被完全清空），也要显示移除的人员
					daysData[change.Date] = map[string]any{
						"staff":           []string{},
						"staffIds":        []string{},
						"staffFlags":      []map[string]any{},
						"removedStaff":    removedNames,
						"removedStaffIds": change.Removed,
						"requiredCount": func() int {
							if shiftReqs := createCtx.StaffRequirements[currentShift.ID]; shiftReqs != nil {
								if count, ok := shiftReqs[change.Date]; ok {
									return count
								}
							}
							return 0
						}(),
						"actualCount": 0,
						"isChanged":   true,
					}
				}
			}
		}

		scheduleDetailPayload = map[string]any{
			"shiftId":   currentShift.ID,
			"shiftName": currentShift.Name,
			"startDate": createCtx.StartDate,
			"endDate":   createCtx.EndDate,
			"schedule": map[string]any{
				"shiftId":  currentShift.ID,
				"priority": 0, // 普通班次优先级为0
				"days":     daysData,
			},
		}
	}

	// 添加操作按钮
	actions := []session.WorkflowAction{
		{
			ID:      "confirm_continue",
			Label:   "确认继续",
			Type:    session.ActionTypeWorkflow,
			Style:   session.ActionStylePrimary,
			Event:   session.WorkflowEvent(CreateV2EventShiftReviewConfirmed),
			Payload: nil,
		},
		{
			ID:      "adjust",
			Label:   "提出调整需求",
			Type:    session.ActionTypeWorkflow,
			Style:   session.ActionStyleSecondary,
			Event:   session.WorkflowEvent(CreateV2EventShiftReviewAdjust),
			Payload: nil,
		},
	}

	// 如果有排班结果，添加"查看详情"按钮
	var queryAction session.WorkflowAction
	if scheduleDetailPayload != nil {
		queryAction = session.WorkflowAction{
			ID:      fmt.Sprintf("view_shift_schedule_%s", currentShift.ID),
			Type:    session.ActionTypeQuery,
			Label:   "📊 查看详情",
			Payload: scheduleDetailPayload,
			Style:   session.ActionStyleSuccess, // 绿色图标
		}
	}

	// 发送消息
	if queryAction.ID != "" {
		// 如果有查看详情按钮，附加到消息上
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: message,
			Actions: []session.WorkflowAction{queryAction},
			Metadata: map[string]any{
				"type":      "shift_review",
				"shiftId":   currentShift.ID,
				"shiftName": currentShift.Name,
			},
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			logger.Warn("Failed to send review message with detail button", "error", err)
		}
	} else {
		// 如果没有查看详情按钮，只发送消息
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send review message", "error", err)
		}
	}

	// 设置工作流动作，并禁用用户输入
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "请审核当前班次排班结果", actions); err != nil {
		return fmt.Errorf("failed to set workflow actions: %w", err)
	}
	// 禁用用户输入（审核状态不允许输入）
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	return nil
}

// actOnShiftReviewConfirmed 处理用户确认继续
func actOnShiftReviewConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Shift review confirmed, continuing to next shift")

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 清除工作流动作
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 移动到下一个班次
	createCtx.IncrementPhaseProgress()

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 检查是否还有班次
	if createCtx.IsPhaseComplete() {
		// 阶段完成
		logger.Info("CreateV2: Phase complete", "phase", createCtx.CurrentPhase)
		return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
	}

	// 根据当前阶段，转换到相应的状态，然后再启动子工作流
	// 这样当子工作流完成时，父工作流处于正确的状态，可以处理 CreateV2EventShiftCompleted 事件
	var targetState engine.State
	switch createCtx.CurrentPhase {
	case PhaseSpecialShift:
		targetState = CreateV2StateSpecialShift
	case PhaseNormalShift:
		targetState = CreateV2StateNormalShift
	case PhaseResearchShift:
		targetState = CreateV2StateResearchShift
	default:
		logger.Error("CreateV2: Unknown phase when confirming shift review",
			"phase", createCtx.CurrentPhase)
		return fmt.Errorf("unknown phase: %s", createCtx.CurrentPhase)
	}

	// 先转换状态，然后再启动子工作流
	// 注意：这里不能直接调用 wctx.Send，因为状态转换已经在定义中处理了
	// 我们需要手动更新状态，然后启动子工作流
	// 但是，由于状态转换定义中已经指定了 To: CreateV2StateSpecialShift（硬编码），
	// 我们需要在 AfterAct 中处理状态转换，或者修改状态转换定义

	// 实际上，更好的方法是在状态转换定义中支持动态转换
	// 但为了快速修复，我们可以在启动子工作流之前，确保状态正确
	// 由于状态转换已经在定义中处理，我们只需要启动子工作流
	// 但是，如果状态转换定义中的 To 状态不正确，我们需要手动转换

	// 临时解决方案：在启动子工作流之前，手动更新状态
	// 这样当子工作流完成时，父工作流处于正确的状态
	// 注意：状态转换定义中硬编码了 To: CreateV2StateSpecialShift，但实际应该根据当前阶段动态决定
	// 我们使用 UpdateWorkflowMeta 来覆盖状态转换定义中的硬编码状态
	if _, err := wctx.SessionService().UpdateWorkflowMeta(ctx, sess.ID, func(meta *session.WorkflowMeta) error {
		meta.Phase = session.WorkflowState(targetState)
		return nil
	}); err != nil {
		logger.Warn("Failed to update workflow state", "error", err, "targetState", targetState)
		// 继续执行，不返回错误
	}

	// 启动下一个班次
	// 注意：状态转换会在 AfterAct 中根据当前阶段动态处理
	return spawnCurrentShift(ctx, wctx, createCtx)
}

// actOnShiftReviewConfirmedAfterAct 在状态转换后根据当前阶段动态调整状态
func actOnShiftReviewConfirmedAfterAct(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 根据当前阶段，转换到相应的状态
	// 这样当子工作流完成时，父工作流处于正确的状态，可以处理 CreateV2EventShiftCompleted 事件
	var targetState engine.State
	switch createCtx.CurrentPhase {
	case PhaseSpecialShift:
		targetState = CreateV2StateSpecialShift
	case PhaseNormalShift:
		targetState = CreateV2StateNormalShift
	case PhaseResearchShift:
		targetState = CreateV2StateResearchShift
	default:
		logger.Error("CreateV2: Unknown phase when confirming shift review",
			"phase", createCtx.CurrentPhase)
		return fmt.Errorf("unknown phase: %s", createCtx.CurrentPhase)
	}

	// 获取当前状态（从 Session 的 WorkflowMeta 中获取）
	currentState := engine.State(sess.WorkflowMeta.Phase)

	// 如果当前状态不是目标状态，更新状态
	if currentState != targetState {
		logger.Info("CreateV2: Updating workflow state after shift review confirmed",
			"currentState", currentState,
			"targetState", targetState,
			"phase", createCtx.CurrentPhase)

		// 使用 UpdateWorkflowMeta 来更新 Session 的状态
		if _, err := wctx.SessionService().UpdateWorkflowMeta(ctx, sess.ID, func(meta *session.WorkflowMeta) error {
			meta.Phase = session.WorkflowState(targetState)
			return nil
		}); err != nil {
			logger.Warn("Failed to update workflow state", "error", err, "targetState", targetState)
			// 继续执行，不返回错误
		}

		// 注意：Actor 的内部状态（a.state）已经在状态转换时更新为 CreateV2StateSpecialShift（硬编码）
		// 但我们需要将其更新为目标状态，以便子工作流完成时能正确处理 CreateV2EventShiftCompleted 事件
		// 由于 Actor 的状态是私有的，我们无法直接更新
		// 但是，子工作流完成时，会从 Session 读取状态，所以这里更新 Session 应该足够
		// 如果子工作流立即完成，它会从 Session 读取正确的状态
		// 如果子工作流稍后完成，Actor 的状态会在下次事件处理时从 Session 同步
		// 为了确保立即生效，我们需要确保 Actor 的状态也更新
		// 但是，由于 Actor 的状态是私有的，我们无法直接更新
		// 实际上，子工作流完成时，会调用 ReturnToParent，它会从 Session 读取状态
		// 所以这里更新 Session 应该足够
	}

	return nil
}

// actOnShiftReviewAdjust 处理用户提出调整需求
func actOnShiftReviewAdjust(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: User requested adjustment")

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送提示消息，等待用户输入
	message := "请描述您的调整需求，我将根据您的需求重新排班。\n\n例如：\n- 调整某日期的人员安排\n- 增加或减少某天的人数\n- 替换特定人员"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send adjustment message", "error", err)
	}

	// 清除工作流动作，等待用户输入
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "等待调整需求", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}
	// 启用用户输入（等待调整状态允许输入）
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 不直接调用 spawnCurrentShift，等待用户输入消息
	return nil
}

// actOnUserAdjustmentMessage 处理用户发送的调整需求消息
func actOnUserAdjustmentMessage(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Processing user adjustment message")

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从 payload 获取用户消息
	var userMessage string
	if payload != nil {
		if payloadMap, ok := payload.(map[string]any); ok {
			if msg, ok := payloadMap["message"].(string); ok {
				userMessage = msg
			} else if msg, ok := payloadMap["content"].(string); ok {
				userMessage = msg
			}
		} else if msg, ok := payload.(string); ok {
			userMessage = msg
		}
	}

	// 如果 payload 中没有，尝试从会话消息获取最后一条用户消息
	if userMessage == "" {
		if len(sess.Messages) > 0 {
			for i := len(sess.Messages) - 1; i >= 0; i-- {
				if sess.Messages[i].Role == session.RoleUser {
					userMessage = sess.Messages[i].Content
					break
				}
			}
		}
	}

	if userMessage == "" {
		logger.Warn("No user message found in payload or session")
		errorMsg := "未收到您的调整需求，请重新输入。"
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); err != nil {
			logger.Warn("Failed to send error message", "error", err)
		}
		// 保持在等待状态，不转换
		return nil
	}

	// 添加用户消息到 session（确保消息显示在界面上）
	// 注意：如果消息已经存在，AddUserMessage 不会重复添加
	if _, err := wctx.SessionService().AddUserMessage(ctx, sess.ID, userMessage); err != nil {
		logger.Warn("Failed to add user message to session", "error", err)
		// 继续处理，不返回错误
	}

	// 首先提取临时需求（在调整排班之前单独识别）
	logger.Info("CreateV2: Extracting temporary needs from user message")
	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	var temporaryNeeds []*d_model.PersonalNeed
	if ok {
		temporaryNeeds, err = aiService.ExtractTemporaryNeeds(
			ctx,
			userMessage,
			createCtx.AllStaffList,
			createCtx.StartDate,
			createCtx.EndDate,
			sess.Messages,
		)
		if err != nil {
			logger.Warn("CreateV2: Failed to extract temporary needs", "error", err)
			// 继续处理，不返回错误（临时需求提取失败不影响调整流程）
			temporaryNeeds = []*d_model.PersonalNeed{}
		} else {
			logger.Info("CreateV2: Extracted temporary needs",
				"count", len(temporaryNeeds))
		}
	} else {
		logger.Warn("CreateV2: AI service not available for extracting temporary needs")
		temporaryNeeds = []*d_model.PersonalNeed{}
	}

	// 发送确认消息（不再进行意图分析，直接交给AI处理）
	confirmMsg := fmt.Sprintf("✅ 已收到您的调整需求：\n\n%s\n\n正在根据您的需求调整排班...", userMessage)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, confirmMsg); err != nil {
		logger.Warn("Failed to send confirmation message", "error", err)
	}

	// 禁用用户输入（处理调整需求后，重新进入排班状态）
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 获取当前班次信息
	if createCtx.CurrentShiftIndex >= len(createCtx.PhaseShiftList) {
		return fmt.Errorf("current shift index out of range")
	}
	currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]

	// 获取当前班次的上次排班结果
	var originalDraft *d_model.ShiftScheduleDraft
	var phaseResult *PhaseResult
	switch createCtx.CurrentPhase {
	case PhaseSpecialShift:
		phaseResult = createCtx.SpecialShiftResults
	case PhaseNormalShift:
		phaseResult = createCtx.NormalShiftResults
	case PhaseResearchShift:
		phaseResult = createCtx.ResearchShiftResults
	}

	if phaseResult != nil && phaseResult.ScheduleDrafts != nil {
		if draft, ok := phaseResult.ScheduleDrafts[currentShift.ID]; ok {
			originalDraft = draft
		}
	}

	// 如果没有上次结果，创建一个空的草案
	if originalDraft == nil {
		originalDraft = d_model.NewShiftScheduleDraft()
		logger.Info("No previous draft found, using empty draft", "shiftID", currentShift.ID)
	}

	// 构建调整上下文
	// 重要：在调整排班时，需要排除当前班次的占用，因为要重排当前班次
	// 创建一个临时的 OccupiedSlots，排除当前班次的占用
	adjustedOccupiedSlots := make(map[string]map[string]string)
	for staffID, staffDates := range createCtx.OccupiedSlots {
		adjustedStaffDates := make(map[string]string)
		for date, shiftID := range staffDates {
			// 排除当前班次的占用
			if shiftID != currentShift.ID {
				adjustedStaffDates[date] = shiftID
			}
		}
		if len(adjustedStaffDates) > 0 {
			adjustedOccupiedSlots[staffID] = adjustedStaffDates
		}
	}

	// 检查原始人员列表是否为空
	if len(createCtx.StaffList) == 0 {
		logger.Error("CreateV2: StaffList is empty when building adjustment context",
			"shiftID", currentShift.ID,
			"shiftName", currentShift.Name,
			"orgID", sess.OrgID)
		return fmt.Errorf("staff list is empty, cannot proceed with adjustment")
	}

	// 注意：不过滤人员列表，将所有人员传递给 AI
	// AI 会根据 adjustedOccupiedSlots 和 adjustedScheduleMarks 按日期判断每个人员的可用性
	// 一个人在某个日期被占用，只意味着他在那一天不可用，其他日期仍然可用
	// 例如：一个人周一被占用，不代表他周二、周三也不可用

	// 排除当前班次的 ExistingScheduleMarks（用于时段冲突检查）
	// 因为要重排当前班次，所以不应该将当前班次的标记作为冲突
	adjustedScheduleMarks := make(map[string]map[string][]*d_model.ShiftMark)
	for staffID, staffDates := range createCtx.ExistingScheduleMarks {
		adjustedStaffDates := make(map[string][]*d_model.ShiftMark)
		for date, marks := range staffDates {
			adjustedMarks := make([]*d_model.ShiftMark, 0)
			for _, mark := range marks {
				// 排除当前班次的标记
				if mark.ShiftID != currentShift.ID {
					adjustedMarks = append(adjustedMarks, mark)
				}
			}
			if len(adjustedMarks) > 0 {
				adjustedStaffDates[date] = adjustedMarks
			}
		}
		if len(adjustedStaffDates) > 0 {
			adjustedScheduleMarks[staffID] = adjustedStaffDates
		}
	}

	// 提取固定排班信息（从 FixedShiftResults 中）
	// 重要：只提取与当前调整的班次相关的固定排班人员
	// FixedShiftResults.ScheduleDrafts 的 key 是 shift_id，只提取当前班次的固定排班
	fixedAssignments := make(map[string][]string) // date -> staffIds
	if createCtx.FixedShiftResults != nil && createCtx.FixedShiftResults.ScheduleDrafts != nil {
		// 只提取当前调整班次的固定排班（如果存在）
		if draft, ok := createCtx.FixedShiftResults.ScheduleDrafts[currentShift.ID]; ok {
			if draft != nil && draft.Schedule != nil {
				for date, staffIDs := range draft.Schedule {
					// 只提取在排班周期内的日期
					if date >= createCtx.StartDate && date <= createCtx.EndDate {
						fixedAssignments[date] = append([]string{}, staffIDs...)
					}
				}
			}
		}
	}

	logger.Info("CreateV2: Building adjustment context",
		"shiftID", currentShift.ID,
		"shiftName", currentShift.Name,
		"totalStaff", len(createCtx.StaffList),
		"originalOccupiedCount", len(createCtx.OccupiedSlots),
		"adjustedOccupiedCount", len(adjustedOccupiedSlots),
		"originalMarksCount", len(createCtx.ExistingScheduleMarks),
		"adjustedMarksCount", len(adjustedScheduleMarks),
		"fixedAssignmentsDates", len(fixedAssignments))

	// 构建调整上下文（不再进行意图分析，直接交给AI处理）
	adjustCtx := &adjust.AdjustV2Context{
		UserMessage:           userMessage,
		ShiftID:               currentShift.ID,
		ShiftName:             currentShift.Name,
		StartDate:             createCtx.StartDate,
		EndDate:               createCtx.EndDate,
		OriginalDraft:         originalDraft,
		StaffList:             createCtx.StaffList,    // 使用完整人员列表，AI 会根据 OccupiedSlots 按日期判断可用性
		AllStaffList:          createCtx.AllStaffList, // 所有员工列表（用于姓名映射，确保显示正确的姓名而不是UUID）
		StaffRequirements:     createCtx.StaffRequirements[currentShift.ID],
		StaffLeaves:           createCtx.StaffLeaves,
		Rules:                 createCtx.Rules,
		ExistingScheduleMarks: adjustedScheduleMarks, // 使用排除当前班次后的标记
		FixedShiftAssignments: fixedAssignments,      // 固定排班人员信息
		TemporaryNeeds:        temporaryNeeds,        // 在用户输入时已提取的临时需求
	}

	// 如果 StaffRequirements 为 nil，初始化为空 map
	if adjustCtx.StaffRequirements == nil {
		adjustCtx.StaffRequirements = make(map[string]int)
	}

	// 保存调整上下文到 session
	if err := adjust.SaveAdjustV2Context(ctx, wctx, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 调用调整子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor, cannot spawn sub-workflow")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowScheduleAdjustV2,
		Input:        nil, // Adjust 从 session 读取 adjust_v2_context
		OnComplete:   CreateV2EventShiftAdjusted,
		OnError:      CreateV2EventSubFailed,
		Timeout:      10 * 60 * 1e9, // 10 分钟超时
		SnapshotKeys: []string{
			KeyCreateV2Context,
			adjust.DataKeyAdjustV2Context,
		},
	}

	logger.Info("CreateV2: Spawning Adjust V2 sub-workflow",
		"shiftID", currentShift.ID,
		"shiftName", currentShift.Name,
	)

	return actor.SpawnSubWorkflow(ctx, config)
}

// ============================================================
// 排班结果验证和自动修正
// ============================================================

// validateScheduleResult 验证排班结果
// 检查：1) 固定人员是否都在结果中 2) 每日人数是否满足需求 3) 是否有重复人员
func validateScheduleResult(
	scheduleDraft *d_model.ShiftScheduleDraft,
	fixedShiftAssignments map[string][]string,
	staffRequirements map[string]int,
	shiftID string,
	allStaffList []*d_model.Employee,
) (isValid bool, issues []string) {
	issues = make([]string, 0)

	// 构建姓名映射
	nameMapping := make(map[string]string) // staffID -> staffName
	for _, staff := range allStaffList {
		if staff != nil {
			nameMapping[staff.ID] = staff.Name
		}
	}

	// 验证1：检查固定人员是否都在排班结果中
	if len(fixedShiftAssignments) > 0 {
		for date, fixedStaffIDs := range fixedShiftAssignments {
			if scheduleStaffIDs, exists := scheduleDraft.Schedule[date]; exists {
				// 检查每个固定人员是否都在结果中
				scheduleStaffMap := make(map[string]bool)
				for _, id := range scheduleStaffIDs {
					scheduleStaffMap[id] = true
				}
				for _, fixedID := range fixedStaffIDs {
					if !scheduleStaffMap[fixedID] {
						staffName := fixedID
						if name, ok := nameMapping[fixedID]; ok {
							staffName = name
						}
						issues = append(issues, fmt.Sprintf("%s: 缺少固定人员 %s", date, staffName))
					}
				}
			} else {
				// 该日期没有排班，但需要固定人员
				for _, fixedID := range fixedStaffIDs {
					staffName := fixedID
					if name, ok := nameMapping[fixedID]; ok {
						staffName = name
					}
					issues = append(issues, fmt.Sprintf("%s: 缺少固定人员 %s（该日期未排班）", date, staffName))
				}
			}
		}
	}

	// 验证2：检查每日人数是否满足需求
	for date, reqCount := range staffRequirements {
		if reqCount > 0 { // 只检查需要排班的日期
			actualCount := 0
			if scheduleStaffIDs, exists := scheduleDraft.Schedule[date]; exists {
				actualCount = len(scheduleStaffIDs)
			}
			if actualCount != reqCount {
				issues = append(issues, fmt.Sprintf("%s: 人数不匹配，需求%d人，实际%d人", date, reqCount, actualCount))
			}
		}
	}

	// 验证3：检查重复人员
	for date, staffIDs := range scheduleDraft.Schedule {
		staffCount := make(map[string]int) // staffID -> count
		for _, id := range staffIDs {
			staffCount[id]++
		}
		for staffID, count := range staffCount {
			if count > 1 {
				staffName := staffID
				if name, ok := nameMapping[staffID]; ok {
					staffName = name
				}
				issues = append(issues, fmt.Sprintf("%s: 人员 %s 重复排班（出现%d次）", date, staffName, count))
			}
		}
	}

	isValid = len(issues) == 0
	return isValid, issues
}

// autoCorrectSchedule 自动修正排班结果
func autoCorrectSchedule(
	ctx context.Context,
	wctx engine.Context,
	scheduleDraft *d_model.ShiftScheduleDraft,
	issues []string,
	fixedShiftAssignments map[string][]string,
	staffRequirements map[string]int,
	shiftInfo *d_model.ShiftInfo,
	staffList []*d_model.StaffInfoForAI,
	rules []*d_model.RuleInfo,
	allStaffList []*d_model.Employee,
	existingScheduleMarks map[string]map[string]bool,
) (*d_model.AdjustScheduleResult, error) {
	logger := wctx.Logger()

	// 构建修正需求描述
	var sb strings.Builder
	sb.WriteString("请修正以下问题：\n")
	for i, issue := range issues {
		sb.WriteString(fmt.Sprintf("%d) %s\n", i+1, issue))
	}

	correctionRequirement := sb.String()
	logger.Info("CreateV2: Auto-correcting schedule", "requirement", correctionRequirement)

	// 获取AI服务
	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		return nil, fmt.Errorf("AI service not available")
	}

	// 调用AI修正
	adjustResult, err := aiService.AdjustShiftSchedule(
		ctx,
		correctionRequirement,
		scheduleDraft,
		shiftInfo,
		staffList,
		allStaffList,
		rules, // rules已经是RuleInfo类型
		staffRequirements,
		existingScheduleMarks,
		fixedShiftAssignments,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-correct schedule: %w", err)
	}

	logger.Info("CreateV2: Auto-correction completed",
		"summary", adjustResult.Summary,
		"changesCount", len(adjustResult.Changes))

	return adjustResult, nil
}

// handleShiftFailed 处理班次排班失败
func handleShiftFailed(
	ctx context.Context,
	wctx engine.Context,
	createCtx *CreateV2Context,
	shiftID string,
	shiftName string,
	issues []string,
	scheduleDraft *d_model.ShiftScheduleDraft,
	reason string,
) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Warn("CreateV2: Shift scheduling failed",
		"shiftID", shiftID,
		"shiftName", shiftName,
		"reason", reason,
		"issues", issues)

	// 保存失败信息到 session，供 actOnEnterShiftFailedState 使用
	failedInfo := map[string]any{
		"shiftID":   shiftID,
		"shiftName": shiftName,
		"issues":    issues,
		"reason":    reason,
	}
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "shift_failed_info", failedInfo); err != nil {
		logger.Warn("Failed to save shift failed info", "error", err)
	}

	// 构建失败消息
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("❌ **班次【%s】排班失败**\n\n", shiftName))
	sb.WriteString(fmt.Sprintf("**失败原因**：%s\n\n", reason))
	sb.WriteString("**问题列表**：\n")
	for i, issue := range issues {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
	}
	sb.WriteString("\n**说明**：已尝试自动修正，但仍无法满足要求。修正后的排班结果已保存，您可以查看详情。\n\n")
	sb.WriteString("请选择：")

	// 发送失败消息
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: sb.String(),
		Actions: nil,
		Metadata: map[string]any{
			"type":      "shift_failed",
			"shiftID":   shiftID,
			"shiftName": shiftName,
			"issues":    issues,
			"reason":    reason,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send shift failed message", "error", err)
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 转换到失败状态（按钮将在 actOnEnterShiftFailedState 中设置）
	return wctx.Send(ctx, CreateV2EventShiftFailed, nil)
}

// actOnEnterShiftFailedState 进入班次失败状态时设置按钮
func actOnEnterShiftFailedState(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Entering shift failed state", "sessionID", sess.ID)

	// 从 session 中读取失败信息
	var shiftName string
	if failedInfoRaw, ok := sess.Data["shift_failed_info"]; ok {
		if failedInfo, ok := failedInfoRaw.(map[string]any); ok {
			if name, ok := failedInfo["shiftName"].(string); ok {
				shiftName = name
			}
		}
	}

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			ID:    "continue_shift",
			Type:  session.ActionTypeWorkflow,
			Label: "继续排班",
			Event: session.WorkflowEvent(CreateV2EventShiftFailedContinue),
			Style: session.ActionStylePrimary,
		},
		{
			ID:    "cancel_scheduling",
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV2EventShiftFailedCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 设置工作流 meta（包含工作流操作按钮）
	description := "班次排班失败，请选择操作"
	if shiftName != "" {
		description = fmt.Sprintf("班次【%s】排班失败，请选择操作", shiftName)
	}
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, description, workflowActions); err != nil {
		logger.Warn("Failed to set workflow actions", "error", err)
		return fmt.Errorf("failed to set workflow actions: %w", err)
	}

	return nil
}

// actOnShiftFailedContinue 用户选择继续排班
func actOnShiftFailedContinue(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: User chose to continue after shift failure", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 清除工作流动作
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 移动到下一个班次
	createCtx.IncrementPhaseProgress()

	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 检查是否还有班次
	if createCtx.IsPhaseComplete() {
		// 阶段完成
		logger.Info("CreateV2: Phase complete", "phase", createCtx.CurrentPhase)
		return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
	}

	// 继续下一个班次
	return spawnCurrentShift(ctx, wctx, createCtx)
}

// actOnShiftFailedCancel 用户选择取消排班
func actOnShiftFailedCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: User chose to cancel after shift failure", "sessionID", sess.ID)

	// 清除工作流动作
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", nil); err != nil {
		logger.Warn("Failed to clear workflow actions", "error", err)
	}

	// 发送取消消息
	cancelMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: "已取消排班流程。",
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, cancelMsg); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	// 转换到取消状态
	return wctx.Send(ctx, engine.EventCancel, nil)
}

// formatAdjustChange 格式化调整变化说明
func formatAdjustChange(change *d_model.AdjustScheduleChange, allStaffList []*d_model.Employee) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: ", change.Date))

	// 构建姓名映射
	nameMap := make(map[string]string)
	for _, staff := range allStaffList {
		if staff != nil {
			nameMap[staff.ID] = staff.Name
		}
	}

	// 转换ID为姓名
	convertIDsToNames := func(ids []string) []string {
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := nameMap[id]; ok {
				names = append(names, name)
			} else {
				names = append(names, id) // 如果找不到，使用ID
			}
		}
		return names
	}

	if len(change.Added) > 0 {
		names := convertIDsToNames(change.Added)
		sb.WriteString(fmt.Sprintf("新增 %d 人（%s）", len(change.Added), strings.Join(names, ", ")))
	}
	if len(change.Removed) > 0 {
		if sb.Len() > len(change.Date)+2 { // 如果已经有内容，添加分隔符
			sb.WriteString("; ")
		}
		names := convertIDsToNames(change.Removed)
		sb.WriteString(fmt.Sprintf("移除 %d 人（%s）", len(change.Removed), strings.Join(names, ", ")))
	}

	return sb.String()
}

// convertCreateV2DraftsToScheduleDraft 将 CreateV2 的 ScheduleDrafts 转换为 ScheduleDraft 格式
// 参考 schedule/create/actions.go 中的 convertShiftResultsToScheduleDraft
func convertCreateV2DraftsToScheduleDraft(
	createCtx *CreateV2Context,
	allDrafts map[string]*d_model.ShiftScheduleDraft,
) *d_model.ScheduleDraft {
	draft := &d_model.ScheduleDraft{
		StartDate:  createCtx.StartDate,
		EndDate:    createCtx.EndDate,
		Shifts:     make(map[string]*d_model.ShiftDraft),
		StaffStats: make(map[string]*d_model.StaffStats),
		Conflicts:  make([]*d_model.ScheduleConflict, 0),
	}

	// 构建班次名称和优先级映射
	shiftNameMap := make(map[string]string)
	shiftPriorityMap := make(map[string]int)
	for _, shift := range createCtx.SelectedShifts {
		shiftNameMap[shift.ID] = shift.Name
		shiftPriorityMap[shift.ID] = shift.Priority
	}

	// 构建人员名称映射（使用 AllStaffList）
	staffNameMap := make(map[string]string)
	for _, staff := range createCtx.AllStaffList {
		if staff != nil {
			staffNameMap[staff.ID] = staff.Name
		}
	}

	// 转换每个班次的结果
	for shiftID, shiftDraft := range allDrafts {
		if shiftDraft == nil || shiftDraft.Schedule == nil {
			continue
		}

		// 创建 ShiftDraft
		sd := &d_model.ShiftDraft{
			ShiftID:  shiftID,
			Priority: shiftPriorityMap[shiftID],
			Days:     make(map[string]*d_model.DayShift),
		}

		// 转换每天的排班
		for date, staffIDs := range shiftDraft.Schedule {
			// 获取人员姓名
			staffNames := make([]string, 0, len(staffIDs))
			for _, staffID := range staffIDs {
				if name, ok := staffNameMap[staffID]; ok {
					staffNames = append(staffNames, name)
				} else {
					staffNames = append(staffNames, staffID) // 回退使用 ID
				}
			}

			// 获取需求人数
			requiredCount := 0
			if dateReqs, ok := createCtx.StaffRequirements[shiftID]; ok {
				if count, ok := dateReqs[date]; ok {
					requiredCount = count
				}
			}

			sd.Days[date] = &d_model.DayShift{
				Staff:         staffNames,
				StaffIDs:      staffIDs,
				RequiredCount: requiredCount,
				ActualCount:   len(staffIDs),
			}

			// 统计人员工作信息
			for _, staffID := range staffIDs {
				if _, exists := draft.StaffStats[staffID]; !exists {
					draft.StaffStats[staffID] = &d_model.StaffStats{
						StaffID:   staffID,
						StaffName: staffNameMap[staffID],
						WorkDays:  0,
						Shifts:    make([]string, 0),
					}
				}
				stats := draft.StaffStats[staffID]
				stats.WorkDays++
				stats.Shifts = append(stats.Shifts, shiftNameMap[shiftID])
			}
		}

		draft.Shifts[shiftID] = sd
	}

	return draft
}
