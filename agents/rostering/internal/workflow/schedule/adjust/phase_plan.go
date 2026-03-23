package adjust

import (
	"context"
	"fmt"
	"strings"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ========== 阶段4: 生成调整计划 ==========

// actScheduleAdjustGeneratePlan 生成调整计划
func actScheduleAdjustGeneratePlan(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.ParsedIntent == nil {
		logger.Warn("No parsed intent, cannot generate plan")
		return nil
	}

	logger.Info("Generating adjustment plan", "intentType", adjustCtx.ParsedIntent.Type)

	// TODO: 调用 AI 服务生成详细的调整计划
	// 当前使用简单实现
	plan := generateAdjustmentPlan(adjustCtx)
	adjustCtx.AdjustmentPlan = plan
	adjustCtx.AddLog(fmt.Sprintf("生成调整计划，包含 %d 项变更", len(plan.Changes)))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 如果没有变更，显示提示消息
	if len(plan.Changes) == 0 {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 未能识别出有效的调整方案，请重新描述您的需求。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
	}

	return nil
}

// actScheduleAdjustAfterGeneratePlan 生成计划后触发下一步
func actScheduleAdjustAfterGeneratePlan(ctx context.Context, wctx engine.Context, payload any) error {
	sess := wctx.Session()
	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.AdjustmentPlan == nil || len(adjustCtx.AdjustmentPlan.Changes) == 0 {
		return wctx.Send(ctx, EventAdjustBackToIntent, nil)
	}
	return wctx.Send(ctx, EventAdjustPlanGenerated, nil)
}

// actScheduleAdjustPreviewPlan 预览调整计划
func actScheduleAdjustPreviewPlan(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Previewing adjustment plan")

	plan := adjustCtx.AdjustmentPlan
	if plan == nil || len(plan.Changes) == 0 {
		// 没有计划，直接返回（不应该到这里）
		return nil
	}

	// 构建预览消息
	var sb strings.Builder
	sb.WriteString("📋 **调整计划预览**\n\n")

	for i, change := range plan.Changes {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, formatChange(change)))
	}

	sb.WriteString("\n")

	// 显示影响分析
	if plan.Impact != nil {
		sb.WriteString("**影响分析：**\n")
		if len(plan.Impact.AffectedStaff) > 0 {
			sb.WriteString(fmt.Sprintf("- 涉及员工：%s\n", strings.Join(plan.Impact.AffectedStaff, "、")))
		}
		if len(plan.Impact.Conflicts) > 0 {
			sb.WriteString(fmt.Sprintf("- ⚠️ 潜在冲突：%d 项\n", len(plan.Impact.Conflicts)))
			for _, conflict := range plan.Impact.Conflicts {
				sb.WriteString(fmt.Sprintf("  - %s\n", conflict))
			}
		}
		if len(plan.Impact.Warnings) > 0 {
			sb.WriteString("- ⚠️ 警告：\n")
			for _, warning := range plan.Impact.Warnings {
				sb.WriteString(fmt.Sprintf("  - %s\n", warning))
			}
		}
	}

	// 构建操作按钮
	actions := []session.WorkflowAction{
		{
			ID:    "confirm",
			Event: session.WorkflowEvent(EventAdjustPlanConfirmed),
			Label: "✅ 确认执行",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStylePrimary,
		},
		{
			ID:    "modify",
			Event: session.WorkflowEvent(EventAdjustPlanRejected),
			Label: "✏️ 修改计划",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStyleSecondary,
		},
		{
			ID:    "cancel",
			Event: session.WorkflowEvent(EventAdjustUserCancelled),
			Label: "❌ 取消",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStyleSecondary,
		},
	}

	// 使用 SetWorkflowMetaWithActions 同时设置消息和按钮
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
		sb.String(), actions); err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	return nil
}

// generateAdjustmentPlan 生成调整计划（简单实现）
func generateAdjustmentPlan(adjustCtx *d_model.ScheduleAdjustContext) *d_model.AdjustmentPlan {
	plan := &d_model.AdjustmentPlan{
		Changes: make([]*d_model.ScheduleChange, 0),
		Impact:  &d_model.AdjustmentImpact{},
	}

	intent := adjustCtx.ParsedIntent
	if intent == nil {
		return plan
	}

	// 根据意图类型生成变更
	switch intent.Type {
	case d_model.AdjustIntentSwap:
		// 换班：两人互换
		if intent.Date != "" && intent.StaffA != "" && intent.StaffB != "" {
			plan.Changes = append(plan.Changes,
				&d_model.ScheduleChange{
					ChangeType: "swap",
					Date:       intent.Date,
					ShiftID:    adjustCtx.SelectedShiftID,
					OldStaff:   []string{intent.StaffA},
					NewStaff:   []string{intent.StaffB},
				},
				&d_model.ScheduleChange{
					ChangeType: "swap",
					Date:       intent.Date,
					ShiftID:    adjustCtx.SelectedShiftID,
					OldStaff:   []string{intent.StaffB},
					NewStaff:   []string{intent.StaffA},
				},
			)
			plan.Impact.AffectedStaff = []string{intent.StaffA, intent.StaffB}
		}

	case d_model.AdjustIntentReplace:
		// 替换：一人替换另一人
		if intent.Date != "" && intent.StaffA != "" && intent.StaffB != "" {
			plan.Changes = append(plan.Changes,
				&d_model.ScheduleChange{
					ChangeType: "replace",
					Date:       intent.Date,
					ShiftID:    adjustCtx.SelectedShiftID,
					OldStaff:   []string{intent.StaffA},
					NewStaff:   []string{intent.StaffB},
				},
			)
			plan.Impact.AffectedStaff = []string{intent.StaffA, intent.StaffB}
		}

	case d_model.AdjustIntentAdd:
		// 添加员工
		if intent.Date != "" && intent.StaffA != "" {
			plan.Changes = append(plan.Changes,
				&d_model.ScheduleChange{
					ChangeType: "add",
					Date:       intent.Date,
					ShiftID:    adjustCtx.SelectedShiftID,
					NewStaff:   []string{intent.StaffA},
				},
			)
			plan.Impact.AffectedStaff = []string{intent.StaffA}
		}

	case d_model.AdjustIntentRemove:
		// 移除员工
		if intent.Date != "" && intent.StaffA != "" {
			plan.Changes = append(plan.Changes,
				&d_model.ScheduleChange{
					ChangeType: "remove",
					Date:       intent.Date,
					ShiftID:    adjustCtx.SelectedShiftID,
					OldStaff:   []string{intent.StaffA},
				},
			)
			plan.Impact.AffectedStaff = []string{intent.StaffA}
		}
	}

	return plan
}

// formatChange 格式化变更描述
func formatChange(change *d_model.ScheduleChange) string {
	getFirstStaff := func(staffList []string) string {
		if len(staffList) > 0 {
			return staffList[0]
		}
		return "未知"
	}

	switch change.ChangeType {
	case "swap":
		return fmt.Sprintf("📅 %s：%s 与 %s 换班", change.Date, getFirstStaff(change.OldStaff), getFirstStaff(change.NewStaff))
	case "replace":
		return fmt.Sprintf("📅 %s：%s 替换 %s", change.Date, getFirstStaff(change.NewStaff), getFirstStaff(change.OldStaff))
	case "add":
		return fmt.Sprintf("📅 %s：添加 %s", change.Date, getFirstStaff(change.NewStaff))
	case "remove":
		return fmt.Sprintf("📅 %s：移除 %s", change.Date, getFirstStaff(change.OldStaff))
	default:
		return fmt.Sprintf("📅 %s：%s", change.Date, change.ChangeType)
	}
}
