package adjust

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

// ========== 加载数据阶段 ==========

// actScheduleAdjustLoadData 加载排班数据
func actScheduleAdjustLoadData(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Loading schedule data", "sourceType", adjustCtx.SourceType)

	switch adjustCtx.SourceType {
	case d_model.AdjustSourceSessionDraft:
		// 已从会话草案加载，直接继续
		if adjustCtx.CurrentDraft != nil {
			return nil
		}
		// 重新从会话加载
		if createCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
			if createCtx, ok := createCtxRaw.(*d_model.ScheduleCreateContext); ok && createCtx.DraftSchedule != nil {
				adjustCtx.OriginalDraft = createCtx.DraftSchedule
				adjustCtx.CurrentDraft = cloneScheduleDraft(createCtx.DraftSchedule)
				adjustCtx.StartDate = createCtx.StartDate
				adjustCtx.EndDate = createCtx.EndDate
				adjustCtx.AvailableShifts = createCtx.SelectedShifts
				adjustCtx.StaffList = createCtx.StaffList

				if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
					return fmt.Errorf("failed to save adjust context: %w", err)
				}

				return nil
			}
		}
		return fmt.Errorf("session draft not found")

	case d_model.AdjustSourceDateRange:

		// 尝试从 create 上下文获取班次和员工信息
		if createCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
			if createCtx, ok := createCtxRaw.(*d_model.ScheduleCreateContext); ok {
				adjustCtx.AvailableShifts = createCtx.SelectedShifts
				adjustCtx.StaffList = createCtx.StaffList
			}
		}

		// 如果没有从 create 上下文获取到班次，则从 SDK 加载
		if len(adjustCtx.AvailableShifts) == 0 {
			if svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); ok {
				shifts, err := svc.ListShifts(ctx, sess.OrgID, "")
				if err != nil {
					logger.Error("Failed to load shifts from SDK", "error", err)
				} else {
					adjustCtx.AvailableShifts = shifts
					logger.Info("Loaded shifts from SDK", "shiftCount", len(shifts))
				}
			} else {
				logger.Error("rosteringService not found in services registry")
			}
		}

		// 如果没有从 create 上下文获取到员工，则从 SDK 加载
		if len(adjustCtx.StaffList) == 0 {
			if svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); ok {
				// 使用 ListStaff 加载所有员工
				staffResult, err := svc.ListStaff(ctx, d_model.StaffListFilter{
					OrgID: sess.OrgID,
				})
				if err != nil {
					logger.Error("Failed to load staff from SDK", "error", err)
				} else if staffResult != nil && len(staffResult.Items) > 0 {
					// 将 Staff 转换为 Employee
					employees := make([]*d_model.Employee, 0, len(staffResult.Items))
					for _, staff := range staffResult.Items {
						if staff != nil {
							employees = append(employees, &d_model.Employee{
								ID:     staff.UserID,
								Name:   staff.Name,
								Groups: staff.Groups,
							})
						}
					}
					adjustCtx.StaffList = employees
					logger.Info("Loaded employees from SDK", "employeeCount", len(employees))
				}
			}
		}

		// 创建空草案（TODO: 实际应从数据库加载历史排班数据）
		adjustCtx.OriginalDraft = &d_model.ScheduleDraft{
			StartDate: adjustCtx.StartDate,
			EndDate:   adjustCtx.EndDate,
			Shifts:    make(map[string]*d_model.ShiftDraft),
		}
		adjustCtx.CurrentDraft = cloneScheduleDraft(adjustCtx.OriginalDraft)

		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown source type: %s", adjustCtx.SourceType)
	}
}

// actScheduleAdjustAfterLoadData 加载数据后的流转
func actScheduleAdjustAfterLoadData(ctx context.Context, wctx engine.Context, payload any) error {
	return wctx.Send(ctx, EventAdjustDataLoaded, nil)
}

// actScheduleAdjustOnDateRangeSelected 处理日期范围选择
func actScheduleAdjustOnDateRangeSelected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Processing date range selection")

	var startDate, endDate string

	if payload != nil {
		var req struct {
			Range     string `json:"range,omitempty"`
			StartDate string `json:"startDate,omitempty"`
			EndDate   string `json:"endDate,omitempty"`
		}

		if err := ParsePayload(payload, &req); err != nil {
			return fmt.Errorf("failed to parse payload: %w", err)
		}

		// 处理预设范围
		if req.Range != "" {
			now := time.Now()
			switch req.Range {
			case "this_week":
				// 计算本周的开始和结束
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				start := now.AddDate(0, 0, -weekday+1)
				end := start.AddDate(0, 0, 6)
				startDate = start.Format("2006-01-02")
				endDate = end.Format("2006-01-02")

			case "next_week":
				// 计算下周的开始和结束
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				start := now.AddDate(0, 0, -weekday+8)
				end := start.AddDate(0, 0, 6)
				startDate = start.Format("2006-01-02")
				endDate = end.Format("2006-01-02")
			}
		}

		// 使用自定义日期覆盖
		if req.StartDate != "" {
			startDate = req.StartDate
		}
		if req.EndDate != "" {
			endDate = req.EndDate
		}
	}

	if startDate == "" || endDate == "" {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 请选择有效的日期范围。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		// 标记日期无效，需要在 AfterAct 中处理
		adjustCtx.StartDate = ""
		adjustCtx.EndDate = ""
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 更新上下文
	adjustCtx.SourceType = d_model.AdjustSourceDateRange
	adjustCtx.StartDate = startDate
	adjustCtx.EndDate = endDate
	adjustCtx.AddLog(fmt.Sprintf("选择日期范围：%s 至 %s", startDate, endDate))

	// 加载班次数据
	logger.Info("Loading shifts for date range", "startDate", startDate, "endDate", endDate, "orgID", sess.OrgID)

	// 尝试从 create 上下文获取班次和员工信息
	if createCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
		if createCtx, ok := createCtxRaw.(*d_model.ScheduleCreateContext); ok {
			adjustCtx.AvailableShifts = createCtx.SelectedShifts
			adjustCtx.StaffList = createCtx.StaffList
			logger.Info("Loaded shifts from create context", "shiftCount", len(adjustCtx.AvailableShifts))
		}
	}

	// 如果没有从 create 上下文获取到班次，则从 SDK 加载
	if len(adjustCtx.AvailableShifts) == 0 {
		logger.Info("No shifts from create context, loading from SDK")
		if svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); ok {
			shifts, err := svc.ListShifts(ctx, sess.OrgID, "")
			if err != nil {
				logger.Error("Failed to load shifts from SDK", "error", err)
			} else {
				adjustCtx.AvailableShifts = shifts
				logger.Info("Loaded shifts from SDK", "shiftCount", len(shifts))
			}
		} else {
			logger.Error("rosteringService not found in services registry")
		}
	}

	// 查询日期范围内的实际排班数据
	if len(adjustCtx.AvailableShifts) > 0 {
		if svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); ok {
			// 查询排班数据
			filter := d_model.ScheduleQueryFilter{
				OrgID:     sess.OrgID,
				StartDate: startDate,
				EndDate:   endDate,
				PageSize:  1000, // 获取所有排班
			}
			result, err := svc.QuerySchedules(ctx, filter)
			if err != nil {
				logger.Error("Failed to query schedules", "error", err)
			} else {
				logger.Info("Queried schedules for date range", "count", len(result.Schedules))

				// 统计每个班次的排班数量
				shiftScheduleCount := make(map[string]int)
				for _, schedule := range result.Schedules {
					shiftScheduleCount[schedule.ShiftID]++
				}

				// 保存班次排班统计到上下文
				adjustCtx.ShiftScheduleCounts = shiftScheduleCount

				// 保存排班数据到上下文，供后续使用
				adjustCtx.ExistingSchedules = result.Schedules

				logger.Info("Shift schedule counts", "counts", shiftScheduleCount)
			}
		}
	}

	// 创建空草案（TODO: 实际应从数据库加载历史排班数据）
	adjustCtx.OriginalDraft = &d_model.ScheduleDraft{
		StartDate: adjustCtx.StartDate,
		EndDate:   adjustCtx.EndDate,
		Shifts:    make(map[string]*d_model.ShiftDraft),
	}
	adjustCtx.CurrentDraft = cloneScheduleDraft(adjustCtx.OriginalDraft)

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	// 发送确认消息
	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: fmt.Sprintf("📅 已选择日期范围：%s 至 %s，正在加载排班数据...", startDate, endDate),
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterDateRangeSelected 日期范围选择后的流转
func actScheduleAdjustAfterDateRangeSelected(ctx context.Context, wctx engine.Context, payload any) error {
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.StartDate == "" || adjustCtx.EndDate == "" {
		return wctx.Send(ctx, EventAdjustNeedDateRange, nil)
	}

	// 此时状态已经是 StateAdjustLoadingData，应发送 EventAdjustDataLoaded 进入下一阶段
	return wctx.Send(ctx, EventAdjustDataLoaded, nil)
}
