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

// ========== 阶段1: 初始化与解析来源 ==========

// actScheduleAdjustStart 启动排班调整工作流
// 职责：分析调整请求来源，确定是从会话草案还是历史排班调整
func actScheduleAdjustStart(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	if sess == nil {
		logger.Error("session is nil in actScheduleAdjustStart")
		return fmt.Errorf("session not found")
	}

	logger.Info("Starting schedule adjust workflow", "sessionID", sess.ID)

	// 1. 初始化调整上下文
	adjustCtx := d_model.NewScheduleAdjustContext()

	// 2. 尝试从 payload 获取参数
	var startDate, endDate string
	var hasExplicitDates bool

	if payload != nil {
		var req struct {
			StartDate   string `json:"startDate,omitempty"`
			EndDate     string `json:"endDate,omitempty"`
			ShiftID     string `json:"shiftId,omitempty"`
			FromSession bool   `json:"fromSession,omitempty"` // 是否从当前会话草案调整
		}
		if err := ParsePayload(payload, &req); err == nil {
			if req.StartDate != "" && req.EndDate != "" {
				startDate = req.StartDate
				endDate = req.EndDate
				hasExplicitDates = true
			}
			if req.ShiftID != "" {
				adjustCtx.SelectedShiftID = req.ShiftID
			}
			if req.FromSession {
				adjustCtx.SourceType = d_model.AdjustSourceSessionDraft
			}
		}
	}

	// 3. 尝试从 Intent.Entities 提取日期范围
	if !hasExplicitDates {
		if intentRaw, ok := sess.Data["intent"]; ok {
			if intent, ok := intentRaw.(*session.Intent); ok {
				logger.Debug("Extracting parameters from intent", "entities", intent.Entities)

				// 提取日期范围
				if dateRange, ok := intent.Entities["dateRange"].(string); ok && dateRange != "" {
					if s, e, err := ParseDateRange(dateRange); err == nil {
						startDate = s
						endDate = e
						hasExplicitDates = true
						logger.Info("Parsed date range from intent", "dateRange", dateRange, "start", s, "end", e)
					}
				}

				// 分别提取 startDate 和 endDate
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
			}
		}
	}

	// 4. 检查是否有当前会话的排班草案
	if adjustCtx.SourceType == "" {
		if createCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
			if createCtx, ok := createCtxRaw.(*d_model.ScheduleCreateContext); ok && createCtx.DraftSchedule != nil {
				// 存在会话草案
				adjustCtx.SourceType = d_model.AdjustSourceSessionDraft
				adjustCtx.OriginalDraft = createCtx.DraftSchedule
				adjustCtx.CurrentDraft = cloneScheduleDraft(createCtx.DraftSchedule)
				adjustCtx.StartDate = createCtx.StartDate
				adjustCtx.EndDate = createCtx.EndDate
				adjustCtx.AvailableShifts = createCtx.SelectedShifts
				adjustCtx.StaffList = createCtx.StaffList
				logger.Info("Found session draft", "startDate", adjustCtx.StartDate, "endDate", adjustCtx.EndDate)
			}
		}
	}

	// 5. 如果有明确日期但没有会话草案，则从历史排班加载
	if hasExplicitDates && adjustCtx.SourceType != d_model.AdjustSourceSessionDraft {
		adjustCtx.SourceType = d_model.AdjustSourceDateRange
		adjustCtx.StartDate = startDate
		adjustCtx.EndDate = endDate
	}

	// 6. 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 7. 记录来源类型（AfterAct 会根据来源类型决定下一步）
	if adjustCtx.SourceType == d_model.AdjustSourceSessionDraft {
		adjustCtx.AddLog("从当前会话草案开始调整")
	} else if adjustCtx.SourceType == d_model.AdjustSourceDateRange {
		adjustCtx.AddLog(fmt.Sprintf("从历史排班加载 %s 至 %s", startDate, endDate))
	}

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterStart 在启动后根据来源类型触发下一步
func actScheduleAdjustAfterStart(ctx context.Context, wctx engine.Context, payload any) error {
	sess := wctx.Session()
	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.SourceType == d_model.AdjustSourceSessionDraft || adjustCtx.SourceType == d_model.AdjustSourceDateRange {
		return wctx.Send(ctx, EventAdjustSourceResolved, nil)
	}
	return wctx.Send(ctx, EventAdjustNeedDateRange, nil)
}

// actScheduleAdjustPromptDateRange 提示用户选择日期范围
func actScheduleAdjustPromptDateRange(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Prompting user to select date range")

	// 构建日期范围选择消息和按钮
	message := "📅 请选择要调整的排班日期范围："
	actions := []session.WorkflowAction{
		{
			ID:    "select_this_week",
			Event: session.WorkflowEvent(EventAdjustDateRangeSelected),
			Label: "📆 本周",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"range": "this_week",
			},
		},
		{
			ID:    "select_next_week",
			Event: session.WorkflowEvent(EventAdjustDateRangeSelected),
			Label: "📆 下周",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"range": "next_week",
			},
		},
		{
			ID:    "select_custom",
			Event: session.WorkflowEvent(EventAdjustDateRangeSelected),
			Type:  session.ActionTypeWorkflow,
			Label: "📝 自定义日期",
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "startDate",
					Label:       "开始日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择开始日期",
				},
				{
					Name:        "endDate",
					Label:       "结束日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择结束日期",
				},
			},
		},
		{
			ID:    "cancel",
			Event: session.WorkflowEvent(EventAdjustUserCancelled),
			Label: "❌ 取消",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStyleDanger,
		},
	}

	// 使用 SetWorkflowMetaWithActions 同时设置消息和按钮，避免重复消息
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, actions); err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	return nil
}

// cloneScheduleDraft 深拷贝排班草案
func cloneScheduleDraft(src *d_model.ScheduleDraft) *d_model.ScheduleDraft {
	if src == nil {
		return nil
	}

	dst := &d_model.ScheduleDraft{
		StartDate: src.StartDate,
		EndDate:   src.EndDate,
		Summary:   src.Summary,
	}

	// 复制 Shifts
	if src.Shifts != nil {
		dst.Shifts = make(map[string]*d_model.ShiftDraft)
		for shiftID, shiftDraft := range src.Shifts {
			newShiftDraft := &d_model.ShiftDraft{
				ShiftID:  shiftDraft.ShiftID,
				Priority: shiftDraft.Priority,
			}
			if shiftDraft.Days != nil {
				newShiftDraft.Days = make(map[string]*d_model.DayShift)
				for date, dayShift := range shiftDraft.Days {
					newDayShift := &d_model.DayShift{
						RequiredCount: dayShift.RequiredCount,
						ActualCount:   dayShift.ActualCount,
					}
					if dayShift.Staff != nil {
						newDayShift.Staff = make([]string, len(dayShift.Staff))
						copy(newDayShift.Staff, dayShift.Staff)
					}
					if dayShift.StaffIDs != nil {
						newDayShift.StaffIDs = make([]string, len(dayShift.StaffIDs))
						copy(newDayShift.StaffIDs, dayShift.StaffIDs)
					}
					newShiftDraft.Days[date] = newDayShift
				}
			}
			dst.Shifts[shiftID] = newShiftDraft
		}
	}

	// 复制 StaffStats
	if src.StaffStats != nil {
		dst.StaffStats = make(map[string]*d_model.StaffStats)
		for staffID, stats := range src.StaffStats {
			newStats := &d_model.StaffStats{
				StaffID:    stats.StaffID,
				StaffName:  stats.StaffName,
				WorkDays:   stats.WorkDays,
				TotalHours: stats.TotalHours,
			}
			if stats.Shifts != nil {
				newStats.Shifts = make([]string, len(stats.Shifts))
				copy(newStats.Shifts, stats.Shifts)
			}
			dst.StaffStats[staffID] = newStats
		}
	}

	// 复制 Conflicts
	if src.Conflicts != nil {
		dst.Conflicts = make([]*d_model.ScheduleConflict, len(src.Conflicts))
		for i, conflict := range src.Conflicts {
			dst.Conflicts[i] = &d_model.ScheduleConflict{
				Date:     conflict.Date,
				Shift:    conflict.Shift,
				Issue:    conflict.Issue,
				Severity: conflict.Severity,
			}
		}
	}

	return dst
}
