package adjust

import (
	"context"
	"fmt"
	"time"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ========== 阶段5: 执行调整 ==========

// actScheduleAdjustExecute 执行调整计划
func actScheduleAdjustExecute(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Executing adjustment plan")

	plan := adjustCtx.AdjustmentPlan

	// 区分重排场景和普通调整场景
	isRegenerateScenario := plan == nil || len(plan.Changes) == 0

	if isRegenerateScenario {
		// 重排场景：重排结果已经在 actScheduleAdjustOnCoreCompleted 中合并到 CurrentDraft
		// 只需要保存到历史记录（用于撤销）并清理临时数据
		if adjustCtx.RegenerateOriginalShift != nil {
			// 保存原始班次快照到历史（用于撤销）
			adjustCtx.PushHistory(&d_model.AdjustmentRecord{
				Timestamp:     time.Now().Format(time.RFC3339),
				DraftSnapshot: cloneScheduleDraft(adjustCtx.CurrentDraft),
				Changes:       nil, // 重排没有变更列表
			})

			// 清理重排临时数据
			adjustCtx.RegenerateOriginalShift = nil

			adjustCtx.AddLog("确认重排结果")

			// 发送成功消息
			msg := session.Message{
				Role:    session.RoleAssistant,
				Content: "✅ 重排结果已确认。",
			}
			if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
				return fmt.Errorf("failed to add message: %w", err)
			}
		}
	} else {
		// 普通调整场景：应用变更列表
		// 1. 保存当前状态到历史记录
		adjustCtx.PushHistory(&d_model.AdjustmentRecord{
			Timestamp:     time.Now().Format(time.RFC3339),
			DraftSnapshot: cloneScheduleDraft(adjustCtx.CurrentDraft),
			Changes:       plan.Changes,
		})

		// 2. 应用变更到草案
		applyChanges(adjustCtx.CurrentDraft, plan.Changes)

		adjustCtx.AddLog(fmt.Sprintf("执行调整完成，共 %d 项变更", len(plan.Changes)))

		// 发送成功消息
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: fmt.Sprintf("✅ 调整已应用，共执行 %d 项变更。", len(plan.Changes)),
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
	}

	// 3. 清除当前计划
	adjustCtx.AdjustmentPlan = nil
	adjustCtx.ParsedIntent = nil

	// 4. 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterExecute 执行调整后的流转
func actScheduleAdjustAfterExecute(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, EventAdjustExecuted, nil)
}

// actScheduleAdjustUndo 撤销上一次调整
func actScheduleAdjustUndo(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Undoing last adjustment")

	if !adjustCtx.CanUndo() {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 没有可撤销的操作。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		return nil
	}

	// 执行撤销
	record := adjustCtx.Undo()
	if record != nil {
		adjustCtx.CurrentDraft = record
		adjustCtx.AddLog("撤销了上一次调整")
	}

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 发送确认消息
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: "↩️ 已撤销上一次调整。",
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}

// actScheduleAdjustRedo 重做上一次撤销的调整
func actScheduleAdjustRedo(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Redoing last undone adjustment")

	if !adjustCtx.CanRedo() {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 没有可重做的操作。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		return nil
	}

	// 执行重做
	record := adjustCtx.Redo()
	if record != nil {
		// 重新应用变更
		adjustCtx.CurrentDraft = cloneScheduleDraft(record)
		adjustCtx.AddLog("重做了上一次撤销的调整")
	}

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 发送确认消息
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: "↪️ 已重做上一次调整。",
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}

// actScheduleAdjustSave 保存调整结果
func actScheduleAdjustSave(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Saving adjustment result")

	// 1. 同步调整后的草案到创建上下文（如果来源是会话草案）
	if adjustCtx.SourceType == d_model.AdjustSourceSessionDraft {
		if createCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
			if createCtx, ok := createCtxRaw.(*d_model.ScheduleCreateContext); ok {
				createCtx.DraftSchedule = adjustCtx.CurrentDraft
				if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, createCtx); err != nil {
					logger.Error("Failed to save create context", "error", err)
				}
			}
		}
	}

	// 2. TODO: 如果是历史排班，调用服务保存到数据库

	adjustCtx.AddLog("调整结果已保存")

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterSave 保存后的流转
func actScheduleAdjustAfterSave(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, EventAdjustSaved, nil)
}

// actScheduleAdjustComplete 完成调整工作流
func actScheduleAdjustComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Completing adjustment workflow")

	// 构建完成消息
	var historyCount int
	if adjustCtx.History != nil {
		historyCount = len(adjustCtx.History)
	}

	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: fmt.Sprintf("✅ 排班调整完成！\n\n📊 本次调整共执行了 %d 次操作。", historyCount),
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// 清除 WorkflowMeta - 使用 SetWorkflowMeta(nil) 来清除
	if _, err := wctx.SessionService().SetWorkflowMeta(ctx, sess.ID, nil); err != nil {
		logger.Warn("Failed to clear workflow meta", "error", err)
	}

	return nil
}

// applyChanges 将变更应用到草案
func applyChanges(draft *d_model.ScheduleDraft, changes []*d_model.ScheduleChange) {
	if draft == nil || draft.Shifts == nil {
		return
	}

	getFirstStaff := func(staffList []string) string {
		if len(staffList) > 0 {
			return staffList[0]
		}
		return ""
	}

	for _, change := range changes {
		shiftDraft, ok := draft.Shifts[change.ShiftID]
		if !ok || shiftDraft.Days == nil {
			continue
		}

		dayShift, ok := shiftDraft.Days[change.Date]
		if !ok {
			// 如果日期不存在，创建新的
			dayShift = &d_model.DayShift{
				Staff:    make([]string, 0),
				StaffIDs: make([]string, 0),
			}
			shiftDraft.Days[change.Date] = dayShift
		}

		switch change.ChangeType {
		case "add":
			// 添加员工
			newStaff := getFirstStaff(change.NewStaff)
			if newStaff != "" {
				dayShift.StaffIDs = append(dayShift.StaffIDs, newStaff)
				dayShift.ActualCount++
			}

		case "remove":
			// 移除员工
			oldStaff := getFirstStaff(change.OldStaff)
			if oldStaff != "" {
				dayShift.StaffIDs = removeFromSlice(dayShift.StaffIDs, oldStaff)
				if dayShift.ActualCount > 0 {
					dayShift.ActualCount--
				}
			}

		case "replace":
			// 替换员工
			oldStaff := getFirstStaff(change.OldStaff)
			newStaff := getFirstStaff(change.NewStaff)
			if oldStaff != "" && newStaff != "" {
				for i, id := range dayShift.StaffIDs {
					if id == oldStaff {
						dayShift.StaffIDs[i] = newStaff
						break
					}
				}
			}

		case "swap":
			// 换班（两人互换，需要在两个记录中分别处理）
			// 这里只处理单条记录，实际换班逻辑由两条记录组成
			oldStaff := getFirstStaff(change.OldStaff)
			newStaff := getFirstStaff(change.NewStaff)
			if oldStaff != "" && newStaff != "" {
				for i, id := range dayShift.StaffIDs {
					if id == oldStaff {
						dayShift.StaffIDs[i] = newStaff
						break
					}
				}
			}
		}
	}
}

// removeFromSlice 从切片中移除元素
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
