package create

import (
	"context"
	"fmt"
	"strings"

	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/workflow/common"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// V4 动作实现 - 阶段 0~5（初始化 → 个人需求确认）
// ============================================================

// ============================================================
// 阶段 0: 初始化 - 启动信息收集
// ============================================================

// actStartInfoCollect 启动信息收集
func actStartInfoCollect(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting workflow", "sessionID", sess.ID)

	createCtx := NewCreateV4Context()
	createCtx.OrgID = sess.OrgID
	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
		"🚀 开始创建排班方案（V4确定性引擎）"); err != nil {
		logger.Warn("Failed to send welcome message", "error", err)
	}

	return wctx.Send(ctx, CreateV4EventInfoCollected, nil)
}

// ============================================================
// 阶段 1: 信息收集完成处理
// ============================================================

// actOnInfoCollected 处理信息收集完成事件
func actOnInfoCollected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Info collection completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	if err := populateInfoFromService(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to populate info: %w", err)
	}

	personalNeeds := ExtractPersonalNeeds(createCtx.Rules, createCtx.AllStaff)
	createCtx.PersonalNeeds = personalNeeds

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return startPeriodConfirmPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 2: 确认排班时间
// ============================================================

// startPeriodConfirmPhase 开始时间确认阶段
func startPeriodConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting period confirmation phase", "sessionID", sess.ID)

	dateRangeDisplay := common.FormatDateRangeForDisplay(createCtx.StartDate, createCtx.EndDate)
	message := fmt.Sprintf("📅 **请确认排班时间范围**\n\n当前时间范围：**%s**\n\n", dateRangeDisplay)
	message += "如需修改，请点击「修改时间」按钮。"

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认时间",
			Event: session.WorkflowEvent(CreateV4EventPeriodConfirmed),
			Style: session.ActionStylePrimary,
			Payload: serializePayload(&PeriodConfirmPayload{
				StartDate: createCtx.StartDate,
				EndDate:   createCtx.EndDate,
			}),
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actOnPeriodConfirmed 处理排班时间确认
func actOnPeriodConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Period confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			if startDate, ok := payloadMap["startDate"].(string); ok {
				createCtx.StartDate = startDate
			}
			if endDate, ok := payloadMap["endDate"].(string); ok {
				createCtx.EndDate = endDate
			}
		}
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return startShiftsConfirmPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 3: 确认班次选择
// ============================================================

// startShiftsConfirmPhase 开始班次确认阶段
func startShiftsConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting shifts confirmation phase", "sessionID", sess.ID)

	var shiftNames []string
	for _, shift := range createCtx.SelectedShifts {
		shiftNames = append(shiftNames, shift.Name)
	}

	message := fmt.Sprintf("📋 **请确认参与排班的班次**\n\n当前选中 **%d** 个班次：\n", len(createCtx.SelectedShifts))
	for i, name := range shiftNames {
		message += fmt.Sprintf("%d. %s\n", i+1, name)
	}
	message += "\n如需修改，请点击「修改班次」按钮。"

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认班次",
			Event: session.WorkflowEvent(CreateV4EventShiftsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actOnShiftsConfirmed 处理班次确认
func actOnShiftsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Shifts confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	return startStaffCountConfirmPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 4: 确认人数配置
// ============================================================

// startStaffCountConfirmPhase 开始人数配置确认阶段
func startStaffCountConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting staff count confirmation phase", "sessionID", sess.ID)

	totalRequirements := 0
	shiftReqMap := make(map[string]int)
	for _, req := range createCtx.StaffRequirements {
		totalRequirements += req.Count
		shiftReqMap[req.ShiftID] += req.Count
	}

	message := "👥 **请确认人数配置**\n\n"
	for _, shift := range createCtx.SelectedShifts {
		count := shiftReqMap[shift.ID]
		message += fmt.Sprintf("- %s：共需 %d 人次\n", shift.Name, count)
	}
	message += fmt.Sprintf("\n**总计**：%d 人次\n", totalRequirements)
	message += "\n如需修改，请在管理页面调整后重新开始。"

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认配置",
			Event: session.WorkflowEvent(CreateV4EventStaffCountConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actOnStaffCountConfirmed 处理人数配置确认
func actOnStaffCountConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Staff count confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 5: 个人需求收集
// ============================================================

// startPersonalNeedsPhase 开始个人需求收集阶段
func startPersonalNeedsPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting personal needs phase", "sessionID", sess.ID)

	message := "📋 **请添加临时需求**\n\n"
	message += "您可以：\n"
	message += "1. 直接开始生成排班计划（跳过临时需求）\n"
	message += "2. 添加临时需求（粘贴文本描述），系统将基于这些需求生成计划\n\n"

	if len(createCtx.PersonalNeeds) > 0 {
		message += buildPersonalNeedsPreviewMessage(createCtx)
	} else {
		message += "当前未解析到任何个人需求。\n"
	}

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "直接生成计划",
			Event: session.WorkflowEvent(CreateV4EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "添加临时需求",
			Event: session.WorkflowEvent(CreateV4EventTemporaryNeedsTextSubmitted),
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "requirementText",
					Label:       "临时需求文本",
					Type:        session.FieldTypeTextarea,
					Required:    true,
					Placeholder: "请粘贴需求文本，例如：\n1. 张三 1月12-14日出差，不能排班\n2. 李四 1月16日希望上早班\n3. 1月20日夜班需增加1人",
					Extra: map[string]any{
						"markdown": true,
						"rows":     8,
					},
				},
			},
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actOnTemporaryNeedsTextSubmitted 处理临时需求文本提交
func actOnTemporaryNeedsTextSubmitted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Temporary needs text submitted", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	var requirementText string
	switch p := payload.(type) {
	case map[string]any:
		if v, ok := p["requirementText"].(string); ok {
			requirementText = v
		} else if v, ok := p["text"].(string); ok {
			requirementText = v
		}
	case string:
		requirementText = p
	}
	requirementText = strings.TrimSpace(requirementText)
	if requirementText == "" {
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "未收到需求文本，请重新填写。"); err != nil {
			logger.Warn("Failed to send empty requirement text message", "error", err)
		}
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	createCtx.TemporaryNeedsText = requirementText

	parsingMsg := "📝 正在解析临时需求..."
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, parsingMsg); err != nil {
		logger.Warn("Failed to send parsing message", "error", err)
	}

	if err := applyTemporaryNeedsText(ctx, wctx, createCtx, requirementText); err != nil {
		errorMsg := fmt.Sprintf("❌ 解析需求失败：%v\n\n请重新填写需求。", err)
		if _, msgErr := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); msgErr != nil {
			logger.Warn("Failed to send error message", "error", msgErr)
		}
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	successMsg := "✅ 需求解析完成！"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, successMsg); err != nil {
		logger.Warn("Failed to send success message", "error", err)
	}

	previewMsg := buildPersonalNeedsPreviewMessage(createCtx)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, previewMsg); err != nil {
		logger.Warn("Failed to send needs preview message", "error", err)
	}

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认需求并继续",
			Event: session.WorkflowEvent(CreateV4EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
		},
	}

	if len(createCtx.TemporaryRules) > 3 {
		workflowActions = append(workflowActions, session.WorkflowAction{
			ID:      "view_temporary_rules",
			Type:    session.ActionTypeQuery,
			Label:   "📋 查看完整规则",
			Style:   session.ActionStyleInfo,
			Payload: buildTemporaryRulesPayload(createCtx),
		})
	}

	workflowActions = append(workflowActions, []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "重新填写需求",
			Event: session.WorkflowEvent(CreateV4EventTemporaryNeedsTextSubmitted),
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "requirementText",
					Label:       "临时需求文本",
					Type:        session.FieldTypeTextarea,
					Required:    true,
					Placeholder: "请粘贴需求文本，例如：\n1. 张三 1月12-14日出差，不能排班\n2. 李四 1月16日希望上早班\n3. 1月20日夜班需增加1人",
					Extra: map[string]any{
						"markdown": true,
						"rows":     8,
					},
				},
			},
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV4EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}...)

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnPersonalNeedsConfirmed 处理个人需求确认
func actOnPersonalNeedsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Personal needs confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	createCtx.PersonalNeedsConfirmed = true
	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return wctx.Send(ctx, CreateV4EventRulesOrganized, nil)
}

// ============================================================
// 消息构建：个人需求预览
// ============================================================

// buildPersonalNeedsPreviewMessage 构建个人需求预览消息
func buildPersonalNeedsPreviewMessage(createCtx *CreateV4Context) string {
	var message strings.Builder
	message.WriteString("📋 **需求预览**\n\n")

	if len(createCtx.TemporaryRules) > 0 {
		message.WriteString("**解析出的临时规则** ⏰\n\n")

		constraintCount := 0
		preferenceCount := 0

		for i, rule := range createCtx.TemporaryRules {
			if i >= 10 {
				message.WriteString(fmt.Sprintf("... 还有 %d 条规则\n", len(createCtx.TemporaryRules)-10))
				break
			}

			icon := "📌"
			if rule.Category == "constraint" {
				constraintCount++
				if rule.SubCategory == "forbid" {
					icon = "🚫"
				} else if rule.SubCategory == "must" {
					icon = "✅"
				}
			} else {
				preferenceCount++
				icon = "💡"
			}

			message.WriteString(fmt.Sprintf("%d. %s **%s**\n", i+1, icon, rule.Name))
			message.WriteString(fmt.Sprintf("   %s\n", rule.Description))

			if len(rule.Associations) > 0 {
				var assocInfo []string
				for _, assoc := range rule.Associations {
					var name string
					if assoc.AssociationType == model.AssociationTypeEmployee {
						for _, staff := range createCtx.AllStaff {
							if staff.ID == assoc.AssociationID {
								name = staff.Name
								break
							}
						}
					} else if assoc.AssociationType == model.AssociationTypeShift {
						for _, shift := range createCtx.SelectedShifts {
							if shift.ID == assoc.AssociationID {
								name = shift.Name
								break
							}
						}
					}
					if name != "" {
						assocInfo = append(assocInfo, fmt.Sprintf("%s: %s", assoc.AssociationType, name))
					}
				}
				if len(assocInfo) > 0 {
					message.WriteString(fmt.Sprintf("   关联: %s\n", strings.Join(assocInfo, ", ")))
				}
			}
			message.WriteString("\n")
		}

		message.WriteString(fmt.Sprintf("**统计**: 约束规则 %d 条, 偏好规则 %d 条\n\n", constraintCount, preferenceCount))
	}

	if len(createCtx.PersonalNeeds) > 0 && len(createCtx.TemporaryRules) == 0 {
		message.WriteString("已解析出以下个人需求：\n\n")

		positiveCount := 0
		negativeCount := 0

		for staffID, needs := range createCtx.PersonalNeeds {
			if len(needs) == 0 {
				continue
			}

			staffName := staffID
			for _, staff := range createCtx.AllStaff {
				if staff.ID == staffID {
					staffName = staff.Name
					break
				}
			}

			message.WriteString(fmt.Sprintf("**%s**：\n", staffName))
			for i, need := range needs {
				isPositive := (need.RequestType == "prefer" || need.RequestType == "must") && need.TargetShiftID != ""
				if isPositive {
					positiveCount++
					message.WriteString(fmt.Sprintf("  %d. ✅ %s", i+1, need.Description))
				} else {
					negativeCount++
					message.WriteString(fmt.Sprintf("  %d. 🚫 %s", i+1, need.Description))
				}
				if len(need.TargetDates) > 0 {
					message.WriteString(fmt.Sprintf(" (日期: %s)", strings.Join(need.TargetDates, ", ")))
				}
				message.WriteString("\n")
			}
			message.WriteString("\n")
		}

		totalNeeds := positiveCount + negativeCount
		message.WriteString(fmt.Sprintf("**共计**：%d 个需求\n\n", totalNeeds))
	}

	if len(createCtx.TemporaryRules) == 0 && len(createCtx.PersonalNeeds) == 0 {
		message.WriteString("未解析到任何临时需求或规则。\n\n")
	}

	message.WriteString("请确认是否继续生成排班计划。")
	return message.String()
}
