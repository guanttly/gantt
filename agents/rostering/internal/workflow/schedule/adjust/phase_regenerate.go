package adjust

// import (
// 	"context"
// 	"fmt"

// 	"jusha/mcp/pkg/workflow/engine"

// 	d_model "jusha/agent/rostering/domain/model"
// 	d_service "jusha/agent/rostering/domain/service"

// 	"jusha/agent/rostering/internal/workflow/schedule/core"
// )

// // ========== 重排班次阶段 ==========

// // actScheduleAdjustPrepareRegenerate 准备重排班次
// // 检查数据、构建 ShiftSchedulingContext
// func actScheduleAdjustPrepareRegenerate(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	adjustCtx, err := GetScheduleAdjustContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	shiftID := adjustCtx.SelectedShiftID
// 	if shiftID == "" {
// 		return fmt.Errorf("no shift selected for regeneration")
// 	}

// 	// 获取班次信息
// 	var targetShift *d_model.Shift
// 	for _, shift := range adjustCtx.AvailableShifts {
// 		if shift.ID == shiftID {
// 			targetShift = shift
// 			break
// 		}
// 	}
// 	if targetShift == nil {
// 		return fmt.Errorf("shift not found: %s", shiftID)
// 	}

// 	logger.Info("Preparing regeneration for shift",
// 		"shiftID", shiftID,
// 		"shiftName", targetShift.Name,
// 		"hasCurrentDraft", adjustCtx.CurrentDraft != nil)

// 	// 创建共享排班上下文
// 	shiftCtx := d_model.NewShiftSchedulingContext(targetShift, adjustCtx.StartDate, adjustCtx.EndDate, "adjust")

// 	// 保存原班次快照（用于差异对比）并清空当前班次数据
// 	if adjustCtx.CurrentDraft != nil && adjustCtx.CurrentDraft.Shifts != nil {
// 		if originalShiftDraft, ok := adjustCtx.CurrentDraft.Shifts[shiftID]; ok {
// 			// 1. 保存快照
// 			adjustCtx.RegenerateOriginalShift = cloneShiftDraft(originalShiftDraft)

// 			// 2. 从原始快照提取人数需求
// 			if originalShiftDraft.Days != nil {
// 				for date, dayShift := range originalShiftDraft.Days {
// 					if dayShift.RequiredCount > 0 {
// 						shiftCtx.StaffRequirements[date] = dayShift.RequiredCount
// 					}
// 				}
// 			}

// 			// 3. 清空当前班次的排班数据，让Core子工作流在干净环境下重新排班
// 			delete(adjustCtx.CurrentDraft.Shifts, shiftID)
// 			logger.Info("Cleared existing shift data for regeneration", "shiftID", shiftID)
// 		}
// 	}

// 	// 获取班次可用人员（仅该班次绑定的人员）
// 	if len(adjustCtx.ShiftStaffIDs) > 0 {
// 		if staffIDs, ok := adjustCtx.ShiftStaffIDs[shiftID]; ok && len(staffIDs) > 0 {
// 			// 根据ID过滤人员
// 			staffMap := make(map[string]*d_model.Employee)
// 			for _, s := range adjustCtx.StaffList {
// 				staffMap[s.ID] = s
// 			}
// 			for _, id := range staffIDs {
// 				if staff, exists := staffMap[id]; exists {
// 					shiftCtx.StaffList = append(shiftCtx.StaffList, staff)
// 				}
// 			}
// 		}
// 	}
// 	// 如果没有班次绑定人员，使用全部人员
// 	if len(shiftCtx.StaffList) == 0 {
// 		shiftCtx.StaffList = adjustCtx.StaffList
// 	}

// 	// 构建其他班次的已排班标记（避免时段冲突）
// 	// 注意：当前班次的数据已被清空，只会包含其他班次的排班标记
// 	if adjustCtx.CurrentDraft != nil {
// 		shiftCtx.ExistingScheduleMarks = core.BuildExistingScheduleMarks(adjustCtx.CurrentDraft, shiftID, adjustCtx.AvailableShifts)
// 	}

// 	// 获取规则（优先复用缓存，否则从 SDK 查询）
// 	if len(adjustCtx.GlobalRules) > 0 {
// 		shiftCtx.GlobalRules = adjustCtx.GlobalRules
// 	}
// 	if len(adjustCtx.ShiftRules) > 0 {
// 		if rules, ok := adjustCtx.ShiftRules[shiftID]; ok {
// 			shiftCtx.ShiftRules = rules
// 		}
// 	}

// 	// 如果没有规则缓存，从 SDK 查询
// 	if len(shiftCtx.GlobalRules) == 0 || len(shiftCtx.ShiftRules) == 0 {
// 		if svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); ok {
// 			// 查询全局规则
// 			if len(shiftCtx.GlobalRules) == 0 {
// 				globalRules, err := svc.ListRules(ctx, d_model.ListRulesRequest{
// 					OrgID:      sess.OrgID,
// 					ApplyScope: "global",
// 				})
// 				if err != nil {
// 					logger.Warn("Failed to load global rules", "error", err)
// 				} else {
// 					shiftCtx.GlobalRules = globalRules
// 					adjustCtx.GlobalRules = globalRules // 缓存到 adjustCtx
// 				}
// 			}
// 			// 查询班次规则
// 			if len(shiftCtx.ShiftRules) == 0 {
// 				shiftRules, err := svc.GetRulesForShift(ctx, sess.OrgID, shiftID)
// 				if err != nil {
// 					logger.Warn("Failed to load shift rules", "error", err)
// 				} else {
// 					shiftCtx.ShiftRules = shiftRules
// 					if adjustCtx.ShiftRules == nil {
// 						adjustCtx.ShiftRules = make(map[string][]*d_model.Rule)
// 					}
// 					adjustCtx.ShiftRules[shiftID] = shiftRules // 缓存到 adjustCtx
// 				}
// 			}
// 		}
// 	}

// 	// 保存共享排班上下文
// 	if err := core.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return err
// 	}

// 	// 保存调整上下文
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
// 		return fmt.Errorf("failed to save adjust context: %w", err)
// 	}

// 	// 发送开始消息
// 	msg := fmt.Sprintf("🔄 准备重新排班：**%s**\n\n- 日期范围：%s 至 %s\n- 可用人员：%d 人",
// 		targetShift.Name, adjustCtx.StartDate, adjustCtx.EndDate, len(shiftCtx.StaffList))
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, msg); err != nil {
// 		logger.Warn("Failed to send start message", "error", err)
// 	}

// 	logger.Info("Regeneration prepared",
// 		"shiftName", targetShift.Name,
// 		"staffCount", len(shiftCtx.StaffList))

// 	return nil
// }

// // ========== 辅助函数 ==========

// // cloneShiftDraft 克隆班次草案
// func cloneShiftDraft(src *d_model.ShiftDraft) *d_model.ShiftDraft {
// 	if src == nil {
// 		return nil
// 	}
// 	dst := &d_model.ShiftDraft{
// 		ShiftID:  src.ShiftID,
// 		Priority: src.Priority,
// 		Days:     make(map[string]*d_model.DayShift),
// 	}
// 	for date, dayShift := range src.Days {
// 		dst.Days[date] = &d_model.DayShift{
// 			Staff:         append([]string{}, dayShift.Staff...),
// 			StaffIDs:      append([]string{}, dayShift.StaffIDs...),
// 			RequiredCount: dayShift.RequiredCount,
// 			ActualCount:   dayShift.ActualCount,
// 		}
// 	}
// 	return dst
// }
