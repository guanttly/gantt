// Package create 提供子工作流调用示例
// 本文件展示如何从 Create 工作流调用 Core 子工作流
package create

// import (
// 	"context"
// 	"fmt"

// 	"jusha/mcp/pkg/workflow/engine"

// 	d_model "jusha/agent/rostering/domain/model"

// 	. "jusha/agent/rostering/internal/workflow/common"
// 	"jusha/agent/rostering/internal/workflow/schedule/core"
// 	. "jusha/agent/rostering/internal/workflow/state/schedule"
// )

// // ============================================================
// // 子工作流调用示例
// // 以下代码展示如何将直接函数调用改为子工作流调用
// // ============================================================

// // actScheduleCreateSpawnCoreSubWorkflow 启动核心子工作流
// // 这是重构后的版本，使用子工作流替代直接调用
// func actScheduleCreateSpawnCoreSubWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()
// 	scheduleCtx := GetOrCreateScheduleContext(sess)

// 	// 检查是否还有未处理的班次
// 	if scheduleCtx.CurrentShiftIndex >= len(scheduleCtx.SelectedShifts) {
// 		logger.Info("All shifts processed")
// 		return wctx.Send(ctx, EventAllShiftsComplete, nil)
// 	}

// 	// 获取当前班次
// 	currentShift := scheduleCtx.SelectedShifts[scheduleCtx.CurrentShiftIndex]
// 	logger.Info("Spawning core sub-workflow for shift",
// 		"shiftIndex", scheduleCtx.CurrentShiftIndex+1,
// 		"shiftName", currentShift.Name,
// 	)

// 	// ========== 准备子工作流输入 ==========
// 	// 1. 构建 ShiftSchedulingContext
// 	shiftCtx := d_model.NewShiftSchedulingContext(currentShift, scheduleCtx.StartDate, scheduleCtx.EndDate, "create")

// 	// 2. 设置人员列表
// 	if ids, ok := scheduleCtx.ShiftStaffIDs[currentShift.ID]; ok {
// 		staffMap := make(map[string]*d_model.Employee)
// 		for _, s := range scheduleCtx.StaffList {
// 			staffMap[s.ID] = s
// 		}
// 		for _, id := range ids {
// 			if s, exists := staffMap[id]; exists {
// 				shiftCtx.StaffList = append(shiftCtx.StaffList, s)
// 			}
// 		}
// 	}

// 	// 3. 设置人数需求
// 	if requirements, ok := scheduleCtx.ShiftStaffRequirements[currentShift.ID]; ok {
// 		shiftCtx.StaffRequirements = requirements
// 	}

// 	// 4. 设置请假记录
// 	shiftCtx.StaffLeaves = scheduleCtx.StaffLeaves

// 	// 5. 设置规则
// 	shiftCtx.GlobalRules = scheduleCtx.GlobalRules
// 	if shiftRules, ok := scheduleCtx.ShiftRules[currentShift.ID]; ok {
// 		shiftCtx.ShiftRules = shiftRules
// 	}

// 	// 6. 设置已有排班标记
// 	shiftCtx.ExistingScheduleMarks = scheduleCtx.StaffScheduleMarks

// 	// 7. 设置进度回调（用于子工作流向父工作流推送消息）
// 	shiftCtx.ProgressCallback = func(msg string) {
// 		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, msg); err != nil {
// 			logger.Warn("Failed to send progress message from sub-workflow", "error", err)
// 		}
// 	}

// 	// 8. 保存共享排班上下文（子工作流会读取）
// 	if err := core.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return fmt.Errorf("failed to save shift scheduling context: %w", err)
// 	}

// 	// ========== 调用子工作流 ==========
// 	// 检查 Actor 是否支持子工作流
// 	actor, ok := wctx.(*engine.Actor)
// 	if !ok {
// 		// 降级：直接调用函数（保持向后兼容）
// 		logger.Warn("Actor does not support sub-workflow, falling back to direct call")
// 		return fallbackDirectCall(ctx, wctx)
// 	}

// 	// 构建子工作流配置
// 	config := engine.SubWorkflowConfig{
// 		WorkflowName: WorkflowSchedulingCore,
// 		Input: map[string]any{
// 			"shift_id":    currentShift.ID,
// 			"shift_name":  currentShift.Name,
// 			"shift_index": scheduleCtx.CurrentShiftIndex,
// 		},
// 		OnComplete: EventShiftProcessed, // 子工作流成功后触发的事件
// 		OnError:    EventAIFailed,       // 子工作流失败后触发的事件
// 		Timeout:    0,                   // 无限等待（排班可能需要较长时间）
// 		SnapshotKeys: []string{
// 			d_model.DataKeyShiftSchedulingContext,
// 			d_model.DataKeyScheduleCreateContext,
// 		},
// 	}

// 	// 启动子工作流
// 	if err := actor.SpawnSubWorkflow(ctx, config); err != nil {
// 		logger.Error("Failed to spawn core sub-workflow", "error", err)
// 		// 降级处理
// 		return fallbackDirectCall(ctx, wctx)
// 	}

// 	logger.Info("Core sub-workflow spawned successfully",
// 		"shiftName", currentShift.Name,
// 		"depth", actor.GetWorkflowDepth(),
// 	)

// 	return nil
// }

// // actScheduleCreateOnCoreComplete 核心子工作流完成后的处理
// // 这个 Action 在父工作流收到 EventShiftProcessed 事件时执行
// func actScheduleCreateOnCoreComplete(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()
// 	scheduleCtx := GetOrCreateScheduleContext(sess)

// 	// 获取子工作流结果
// 	result := engine.GetSubWorkflowResult(sess)
// 	if result != nil {
// 		logger.Info("Core sub-workflow completed",
// 			"success", result.Success,
// 			"duration", result.Duration,
// 			"output", result.Output,
// 		)
// 	}

// 	// 获取共享上下文中的结果
// 	shiftCtx, err := core.GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		logger.Warn("ShiftSchedulingContext not found after core completion", "error", err)
// 	}

// 	// 如果有班次结果，合并到 ScheduleCreateContext
// 	currentShift := scheduleCtx.SelectedShifts[scheduleCtx.CurrentShiftIndex]
// 	if shiftCtx != nil {
// 		// 同步 TodoPlan
// 		if shiftCtx.TodoPlan != nil {
// 			if scheduleCtx.ShiftTodoPlans == nil {
// 				scheduleCtx.ShiftTodoPlans = make(map[string]*d_model.ShiftTodoPlan)
// 			}
// 			scheduleCtx.ShiftTodoPlans[currentShift.ID] = shiftCtx.TodoPlan
// 		}

// 		// 合并结果到草案
// 		if shiftCtx.ResultDraft != nil {
// 			if scheduleCtx.DraftSchedule == nil {
// 				scheduleCtx.DraftSchedule = &d_model.ScheduleDraft{
// 					Shifts: make(map[string]*d_model.ShiftDraft),
// 				}
// 			}

// 			// 将 ShiftScheduleDraft 转换为 ShiftDraft 格式
// 			shiftDraft := &d_model.ShiftDraft{
// 				ShiftID: currentShift.ID,
// 				Days:    make(map[string]*d_model.DayShift),
// 			}
// 			for date, staffIDs := range shiftCtx.ResultDraft.Schedule {
// 				shiftDraft.Days[date] = &d_model.DayShift{
// 					StaffIDs:    staffIDs,
// 					ActualCount: len(staffIDs),
// 				}
// 			}
// 			scheduleCtx.DraftSchedule.Shifts[currentShift.ID] = shiftDraft
// 		}
// 	}

// 	// 移动到下一个班次
// 	scheduleCtx.CurrentShiftIndex++

// 	// 保存更新后的上下文
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
// 		return fmt.Errorf("failed to save context: %w", err)
// 	}

// 	// 清理共享排班上下文
// 	if err := core.ClearShiftSchedulingContext(ctx, wctx); err != nil {
// 		logger.Warn("Failed to clear shift scheduling context", "error", err)
// 	}

// 	// 检查是否还有更多班次
// 	if scheduleCtx.CurrentShiftIndex < len(scheduleCtx.SelectedShifts) {
// 		// 继续处理下一个班次
// 		return wctx.Send(ctx, EventShiftProcessed, nil)
// 	}

// 	// 所有班次处理完成
// 	return wctx.Send(ctx, EventAllShiftsComplete, nil)
// }

// // actScheduleCreateOnCoreError 核心子工作流失败后的处理
// func actScheduleCreateOnCoreError(ctx context.Context, wctx engine.Context, payload any) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()
// 	scheduleCtx := GetOrCreateScheduleContext(sess)

// 	// 获取子工作流错误结果
// 	result := engine.GetSubWorkflowResult(sess)
// 	if result != nil {
// 		logger.Error("Core sub-workflow failed",
// 			"error", result.ErrorMsg,
// 			"duration", result.Duration,
// 		)
// 	}

// 	// 当前班次
// 	currentShift := scheduleCtx.SelectedShifts[scheduleCtx.CurrentShiftIndex]

// 	// 发送失败消息
// 	failMsg := fmt.Sprintf("❌ 班次【%s】排班失败: %s", currentShift.Name, result.ErrorMsg)
// 	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, failMsg); err != nil {
// 		logger.Warn("Failed to send error message", "error", err)
// 	}

// 	// 可以选择：
// 	// 1. 跳过当前班次继续处理下一个
// 	// 2. 终止整个工作流

// 	// 这里选择跳过继续
// 	scheduleCtx.CurrentShiftIndex++
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
// 		return fmt.Errorf("failed to save context: %w", err)
// 	}

// 	if scheduleCtx.CurrentShiftIndex < len(scheduleCtx.SelectedShifts) {
// 		return wctx.Send(ctx, EventShiftProcessed, nil) // 继续下一个
// 	}

// 	return wctx.Send(ctx, EventAllShiftsComplete, nil)
// }

// // ============================================================
// // 降级处理
// // ============================================================

// // fallbackDirectCall 降级为直接调用（保持向后兼容）
// func fallbackDirectCall(ctx context.Context, wctx engine.Context) error {
// 	// 这里可以调用原有的直接调用逻辑
// 	// 1. GenerateTodoPlan
// 	// 2. ExecuteTodos
// 	// 3. ValidateResult
// 	// 4. MergeToScheduleContext
// 	return fmt.Errorf("direct call fallback not implemented - use actScheduleCreateGenerateTodoPlan instead")
// }
