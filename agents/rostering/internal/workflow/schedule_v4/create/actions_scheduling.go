package create

import (
	"context"
	"fmt"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	rule_engine "jusha/agent/rostering/internal/engine"
	"jusha/agent/rostering/internal/workflow/schedule_v4/executor"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 阶段 6: 规则组织（V4核心新增）
// ============================================================

// actOnRulesOrganized 规则组织完成
func actOnRulesOrganized(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting rule organization", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送进度消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"⚙️ 正在组织规则和分析依赖关系..."); err != nil {
		logger.Warn("Failed to send progress message", "error", err)
	}

	// 合并常态规则和临时规则
	allRules := make([]*d_model.Rule, 0, len(createCtx.Rules)+len(createCtx.TemporaryRules))
	allRules = append(allRules, createCtx.Rules...)
	allRules = append(allRules, createCtx.TemporaryRules...)

	// 创建规则组织器
	ruleOrganizer := executor.NewRuleOrganizer(logger, nil) // nil 表示不从数据库加载依赖

	// 执行规则组织（使用合并后的规则）
	ruleOrg, err := ruleOrganizer.OrganizeRules(
		ctx,
		createCtx.OrgID,
		allRules,
		createCtx.SelectedShifts,
		nil, nil, nil, // 从规则中自动分析依赖关系
	)
	if err != nil {
		logger.Error("Failed to organize rules", "error", err)
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
			fmt.Sprintf("❌ 规则组织失败：%v", err)); err != nil {
			logger.Warn("Failed to send error message", "error", err)
		}
		return fmt.Errorf("failed to organize rules: %w", err)
	}

	createCtx.RuleOrganization = ruleOrg

	// 构建班次名称映射
	shiftNameMap := make(map[string]string)
	for _, shift := range createCtx.SelectedShifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	// 发送规则组织结果（包含临时规则信息）
	orgMessage := buildRuleOrganizationMessageWithTemporary(ruleOrg, len(createCtx.TemporaryRules), shiftNameMap)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, orgMessage); err != nil {
		logger.Warn("Failed to send organization message", "error", err)
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 进入排班执行阶段
	return wctx.Send(ctx, CreateV4EventSchedulingComplete, nil)
}

// ============================================================
// 阶段 7: 排班执行（使用确定性规则引擎）
// ============================================================

// actOnSchedulingComplete 排班执行
func actOnSchedulingComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting scheduling execution", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	startTime := time.Now()

	// 发送进度消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"🔄 正在执行排班（V4确定性引擎）..."); err != nil {
		logger.Warn("Failed to send progress message", "error", err)
	}

	// 先填充固定排班
	if err := fillFixedShiftSchedules(ctx, wctx, createCtx); err != nil {
		logger.Warn("Failed to fill fixed schedules", "error", err)
	}

	// 获取规则引擎（从 internal/engine 包）
	ruleEngineInstance, ok := engine.GetService[*rule_engine.RuleEngine](wctx, "ruleEngine")
	if !ok {
		// 如果没有规则引擎服务，使用简化的排班逻辑
		logger.Warn("RuleEngine not available, using simplified scheduling")
		if err := executeSimplifiedScheduling(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("simplified scheduling failed: %w", err)
		}
	} else {
		// 使用V4执行器
		v4Executor := executor.NewV4Executor(
			logger,
			ruleEngineInstance,
			executor.NewRuleOrganizer(logger, nil),
		)

		// 合并常态规则和临时规则
		allRules := make([]*d_model.Rule, 0, len(createCtx.Rules)+len(createCtx.TemporaryRules))
		allRules = append(allRules, createCtx.Rules...)
		allRules = append(allRules, createCtx.TemporaryRules...)

		v4Executor.OnConflict = func(conflict *d_model.ScheduleConflict) {
			shiftName := conflict.Shift
			for _, shift := range createCtx.SelectedShifts {
				if shift.ID == conflict.Shift {
					shiftName = shift.Name
					break
				}
			}
			detailSection := ""
			if conflict.Detail != "" {
				detailSection = "\n\n**📊 排班现场快照:**" + conflict.Detail
			}
			msg := fmt.Sprintf("⚠️ **排班发生冲突无法继续:**\n- 日期: %s\n- 班次: %s\n- 问题: %s%s\n\n请检查人员及规则配置后重新发起排班。",
				conflict.Date, shiftName, conflict.Issue, detailSection)
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, msg); err != nil {
				logger.Warn("Failed to send conflict message", "error", err)
			}
		}

		// 准备执行输入
		input := &executor.SchedulingExecutionInput{
			OrgID:             createCtx.OrgID,
			StartDate:         createCtx.StartDate,
			EndDate:           createCtx.EndDate,
			AllStaff:          convertEmployeesToStaff(createCtx.AllStaff),
			ShiftMembersMap:   convertShiftMembersMap(createCtx.ShiftMembersMap),
			Rules:             allRules, // 使用合并后的规则
			Shifts:            createCtx.SelectedShifts,
			PersonalNeeds:     convertPersonalNeeds(createCtx.PersonalNeeds),
			FixedAssignments:  convertFixedAssignments(createCtx.FixedAssignments),
			CurrentDraft:      createCtx.WorkingDraft,
			ShiftRequirements: buildShiftRequirements(createCtx.StaffRequirements),
			RuleOrganization:  createCtx.RuleOrganization,
		}

		result, err := v4Executor.ExecuteScheduling(ctx, input)
		if err != nil {
			logger.Error("V4 scheduling failed", "error", err)

			// 如果产生了明确的冲突（OnConflict已经被触发），则停止排班并报错，不执行回退
			if result != nil && len(result.Schedule.Conflicts) > 0 {
				logger.Warn("V4 scheduling stopped due to unresolvable conflicts")
				// 发送取消事件阻断后续流程进展
				_ = wctx.Send(ctx, CreateV4EventUserCancel, nil)
				return nil // 返回nil以防止抛出全局错误及引起重试按钮
			}

			// 回退到简化排班
			if err := executeSimplifiedScheduling(ctx, wctx, createCtx); err != nil {
				return fmt.Errorf("fallback scheduling failed: %w", err)
			}
		} else {
			// 应用执行结果到工作草稿
			applySchedulingResult(createCtx, result)
			createCtx.LLMCallCount = 1 // V4目标：最少LLM调用
		}
	}

	createCtx.SchedulingDuration = time.Since(startTime).Milliseconds()

	// 发送排班完成消息
	completionMessage := fmt.Sprintf("✅ 排班执行完成！\n\n"+
		"- 耗时：%d ms\n"+
		"- LLM调用次数：%d\n"+
		"- 班次数量：%d\n",
		createCtx.SchedulingDuration,
		createCtx.LLMCallCount,
		len(createCtx.SelectedShifts))

	// 构建规则排班阶段的预览按钮，让用户可以查看当前排班结果
	ruleSchedulePreview := createCtx.BuildFullSchedulePreview()
	previewActions := []session.WorkflowAction{
		{
			ID:      "preview_full_schedule",
			Type:    session.ActionTypeQuery,
			Label:   "📅 预览规则排班结果",
			Payload: ruleSchedulePreview,
			Style:   session.ActionStylePrimary,
		},
		{
			ID:      "view_task_schedule_detail",
			Type:    session.ActionTypeQuery,
			Label:   "📊 查看排班详情",
			Payload: ruleSchedulePreview,
			Style:   session.ActionStyleSuccess,
		},
	}

	previewMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: completionMessage,
		Actions: previewActions,
		Metadata: map[string]any{
			"scheduleDetail": ruleSchedulePreview,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, previewMsg); err != nil {
		logger.Warn("Failed to send rule scheduling preview message", "error", err)
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 直接进入校验阶段（已移除 LLM 调整逻辑）
	return wctx.Send(ctx, CreateV4EventValidationComplete, nil)
}

// ============================================================
// 阶段 8: 确定性校验（只读，不主动修改排班）
// ============================================================

// actOnValidationComplete 校验完成（只读校验，不主动修改排班）
func actOnValidationComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Starting validation", "sessionID", sess.ID)

	createCtx, err := loadCreateV4Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送进度消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		"🔍 正在校验排班结果..."); err != nil {
		logger.Warn("Failed to send progress message", "error", err)
	}

	// 执行确定性校验（只读，不修改排班草稿）
	validationResult := validateSchedule(createCtx, logger)

	// 对无法确定性校验的语义规则执行 LLM 辅助校验（只读，不修改排班）
	if len(validationResult.UncheckedRules) > 0 {
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
			fmt.Sprintf("🤖 正在对 %d 条语义规则进行 LLM 辅助校验...", len(validationResult.UncheckedRules))); err != nil {
			logger.Warn("发送 LLM 校验进度消息失败", "error", err)
		}
		performLLMSemanticValidation(ctx, wctx, createCtx, validationResult, logger)
	}

	// 重新构建摘要（包含 LLM 校验结果）
	validationResult.Summary = buildValidationSummary(validationResult)

	createCtx.ValidationResult = validationResult

	// 发送校验结果消息（仅展示修改意见，不主动修改）
	validationMessage := buildValidationMessage(validationResult)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, validationMessage); err != nil {
		logger.Warn("发送校验结果消息失败", "error", err)
	}

	if err := saveCreateV4Context(ctx, wctx, createCtx); err != nil {
		// 保存上下文失败时仅记录警告，不阻止进入预览阶段
		logger.Warn("保存校验上下文失败，继续进入审核阶段", "error", err)
	}

	// 进入审核阶段
	return startReviewPhase(ctx, wctx, createCtx)

}

// ============================================================
// 消息构建：规则组织
// ============================================================

// buildRuleOrganizationMessage 构建规则组织结果消息
func buildRuleOrganizationMessage(ruleOrg *executor.RuleOrganization) string {
	return buildRuleOrganizationMessageWithTemporary(ruleOrg, 0, nil)
}

// buildRuleOrganizationMessageWithTemporary 构建规则组织结果消息（包含临时规则信息）
func buildRuleOrganizationMessageWithTemporary(ruleOrg *executor.RuleOrganization, temporaryRulesCount int, shiftNameMap map[string]string) string {
	if ruleOrg == nil {
		return "⚠️ 规则组织结果为空"
	}

	var message strings.Builder
	message.WriteString("📋 **规则组织结果**\n\n")
	message.WriteString(fmt.Sprintf("- 约束型规则：%d 条\n", len(ruleOrg.ConstraintRules)))
	message.WriteString(fmt.Sprintf("- 偏好型规则：%d 条\n", len(ruleOrg.PreferenceRules)))
	message.WriteString(fmt.Sprintf("- 依赖型规则：%d 条\n", len(ruleOrg.DependencyRules)))

	if temporaryRulesCount > 0 {
		message.WriteString(fmt.Sprintf("- **临时规则**：%d 条 ⏰\n", temporaryRulesCount))
	}

	message.WriteString(fmt.Sprintf("- 班次依赖关系：%d 个\n", len(ruleOrg.ShiftDependencies)))
	message.WriteString(fmt.Sprintf("- 规则冲突关系：%d 个\n", len(ruleOrg.RuleConflicts)))

	if len(ruleOrg.ShiftExecutionOrder) > 0 {
		message.WriteString("\n**班次执行顺序**：")
		// 将班次ID转换为班次名称
		shiftNames := make([]string, 0, len(ruleOrg.ShiftExecutionOrder))
		for _, shiftID := range ruleOrg.ShiftExecutionOrder {
			if shiftNameMap != nil {
				if name, ok := shiftNameMap[shiftID]; ok {
					shiftNames = append(shiftNames, name)
					continue
				}
			}
			// 如果找不到名称，使用短ID
			if len(shiftID) > 8 {
				shiftNames = append(shiftNames, shiftID[:8]+"...")
			} else {
				shiftNames = append(shiftNames, shiftID)
			}
		}
		message.WriteString(strings.Join(shiftNames, " → "))
		message.WriteString("\n")
	}

	// 显示警告信息（如优先级与依赖关系冲突）
	if len(ruleOrg.Warnings) > 0 {
		message.WriteString("\n⚠️ **配置警告**：\n")
		for _, warning := range ruleOrg.Warnings {
			message.WriteString(fmt.Sprintf("- %s\n", warning.Message))
		}
	}

	return message.String()
}
