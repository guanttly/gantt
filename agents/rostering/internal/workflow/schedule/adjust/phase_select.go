package adjust

import (
	"context"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ========== 阶段2: 选择班次 ==========

// actScheduleAdjustSelectShift 处理班次选择逻辑
// 根据情况决定是否需要用户选择，结果通过 AfterAct 触发
func actScheduleAdjustSelectShift(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	// 清空之前的选择，确保干净状态
	adjustCtx.SelectedShiftID = ""

	logger.Info("Selecting shift for adjustment",
		"availableShiftsCount", len(adjustCtx.AvailableShifts),
		"sourceType", adjustCtx.SourceType)

	// 1. 检查 payload 中是否有班次选择
	if payload != nil {
		var req struct {
			ShiftID string `json:"shiftId,omitempty"`
		}
		if err := ParsePayload(payload, &req); err == nil && req.ShiftID != "" {
			adjustCtx.SelectedShiftID = req.ShiftID
			adjustCtx.AddLog(fmt.Sprintf("已选择班次: %s", req.ShiftID))

			if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
				return fmt.Errorf("failed to save adjust context: %w", err)
			}
			return nil // AfterAct 会触发 EventAdjustShiftSelected
		}
	}

	// 2. 检查 intent 中是否包含班次信息
	if intentRaw, ok := sess.Data["intent"]; ok {
		if intent, ok := intentRaw.(*session.Intent); ok {
			if shiftName, ok := intent.Entities["shiftName"].(string); ok && shiftName != "" {
				// 尝试匹配班次
				for _, shift := range adjustCtx.AvailableShifts {
					if shift.Name == shiftName {
						adjustCtx.SelectedShiftID = shift.ID
						adjustCtx.AddLog(fmt.Sprintf("从意图中识别班次: %s", shiftName))

						if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
							return fmt.Errorf("failed to save adjust context: %w", err)
						}
						return nil // AfterAct 会触发 EventAdjustShiftSelected
					}
				}
			}
		}
	}

	// 3. 如果只有一个班次，自动选择
	if len(adjustCtx.AvailableShifts) == 1 {
		adjustCtx.SelectedShiftID = adjustCtx.AvailableShifts[0].ID
		adjustCtx.AddLog(fmt.Sprintf("自动选择唯一班次: %s", adjustCtx.AvailableShifts[0].Name))

		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil // AfterAct 会触发 EventAdjustShiftSelected
	}

	// 4. 需要用户选择班次，保持状态等待用户交互
	// AfterAct 会触发 EventAdjustNeedShiftSelection
	return nil
}

// actScheduleAdjustAfterSelectShift 在选择班次后触发下一步
func actScheduleAdjustAfterSelectShift(ctx context.Context, wctx engine.Context, payload any) error {
	sess := wctx.Session()
	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.SelectedShiftID != "" {
		return wctx.Send(ctx, EventAdjustShiftSelected, nil)
	}
	return wctx.Send(ctx, EventAdjustNeedShiftSelection, nil)
}

// actScheduleAdjustPromptShiftSelection 提示用户选择班次
func actScheduleAdjustPromptShiftSelection(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Prompting user to select shift",
		"shiftCount", len(adjustCtx.AvailableShifts),
		"currentState", sess.WorkflowMeta.Phase)

	// 构建班次选择消息
	actions := make([]session.WorkflowAction, 0, len(adjustCtx.AvailableShifts)+1)

	for _, shift := range adjustCtx.AvailableShifts {
		// 获取该班次的排班数量
		scheduleCount := 0
		if adjustCtx.ShiftScheduleCounts != nil {
			scheduleCount = adjustCtx.ShiftScheduleCounts[shift.ID]
		}

		// 根据是否有排班显示不同的标签
		var label string
		var style session.WorkflowActionStyle
		if scheduleCount > 0 {
			label = fmt.Sprintf("🏢 %s（%d条排班）", shift.Name, scheduleCount)
			style = session.ActionStylePrimary
		} else {
			label = fmt.Sprintf("🏢 %s（无排班）", shift.Name)
			style = session.ActionStyleSecondary
		}

		actions = append(actions, session.WorkflowAction{
			ID:    fmt.Sprintf("shift_%s", shift.ID),
			Event: session.WorkflowEvent(EventAdjustShiftSelected),
			Label: label,
			Type:  session.ActionTypeWorkflow,
			Style: style,
			Payload: map[string]any{
				"shiftId": shift.ID,
			},
		})
	}

	// 添加取消按钮
	actions = append(actions, session.WorkflowAction{
		ID:    "cancel",
		Event: session.WorkflowEvent(EventAdjustUserCancelled),
		Label: "❌ 取消",
		Type:  session.ActionTypeWorkflow,
		Style: session.ActionStyleDanger,
	})

	// 使用 SetWorkflowMetaWithActions 同时设置消息和按钮
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
		"🏢 请选择要调整的班次：", actions); err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	return nil
}

// actScheduleAdjustOnShiftSelected 班次选中后的处理
func actScheduleAdjustOnShiftSelected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	// 从 payload 中获取班次 ID
	if payload != nil {
		var req struct {
			ShiftID string `json:"shiftId,omitempty"`
		}
		if err := ParsePayload(payload, &req); err == nil && req.ShiftID != "" {
			adjustCtx.SelectedShiftID = req.ShiftID
		}
	}

	if adjustCtx.SelectedShiftID == "" {
		return fmt.Errorf("no shift selected")
	}

	// 获取班次名称
	var shiftName string
	for _, shift := range adjustCtx.AvailableShifts {
		if shift.ID == adjustCtx.SelectedShiftID {
			shiftName = shift.Name
			break
		}
	}

	adjustCtx.AddLog(fmt.Sprintf("已选择班次: %s", shiftName))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 发送确认消息
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: fmt.Sprintf("✅ 已选择班次「%s」，请告诉我您想做什么调整？", shiftName),
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	logger.Info("Shift selected", "shiftID", adjustCtx.SelectedShiftID, "shiftName", shiftName)

	return nil
}

// actScheduleAdjustAfterShiftSelected 班次选中后触发意图收集
func actScheduleAdjustAfterShiftSelected(ctx context.Context, wctx engine.Context, payload any) error {
	// 触发意图收集界面
	return actScheduleAdjustCollectIntent(ctx, wctx, payload)
}
