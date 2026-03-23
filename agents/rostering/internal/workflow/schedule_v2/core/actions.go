package core

import (
	"context"
	"fmt"
	"strings"

	"jusha/agent/rostering/internal/workflow/schedule_v2/utils"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

// ============================================================
// 核心子工作流 Actions
// ============================================================

const (
	session_data_key_validate_result = "core_validate_result"
)

/********************** 1.验证数据 ****************************/
func actValidate(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	sessionID := sess.ID

	logger.Info("Core: Validating necessary data", "sessionID", sessionID)

	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	// 开始逐条验证
	// 首先结果置为false
	wctx.SessionService().SetData(ctx, sessionID, session_data_key_validate_result, false)
	var svc d_service.IRosteringService
	var ok bool
	if svc, ok = engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering); !ok {
		logger.Error("rosteringService not found in services registry")
		return fmt.Errorf("rosteringService not found in services registry")
	}

	// 验证时间
	startDate := shiftCtx.StartDate
	endDate := shiftCtx.EndDate
	{
		err := ValidateDateRange(startDate, endDate)
		if err != nil {
			startDate, endDate, err = GetDefaultNextWeekRange()
			if err != nil {
				return fmt.Errorf("failed to get default week range: %w", err)
			}
			// 需要收集日期信息
			actions := utils.BuildPeriodActions(
				startDate,
				endDate,
				CoreEventUserConfirmed,
				CoreEventUserCancelled,
			)
			confirmMessage := "请提供一个有效的排班周期（开始日期应早于或等于结束日期）"
			return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, confirmMessage, actions)
		}
	}

	// 验证班次
	if shiftCtx.Shift == nil {
		var actions []session.WorkflowAction
		result, err := svc.QuerySchedules(ctx, d_model.ScheduleQueryFilter{
			OrgID:     sess.OrgID,
			StartDate: startDate,
			EndDate:   endDate,
			PageSize:  1000, // 获取所有排班
		})
		if err != nil {
			logger.Error("Failed to query schedules", "error", err)
		}
		shiftScheduleCount := make(map[string]int)
		for _, schedule := range result.Schedules {
			shiftScheduleCount[schedule.ShiftID]++
		}
		shifts, err := svc.ListShifts(ctx, sess.OrgID, "")
		if err != nil {
			logger.Error("Failed to load shifts from SDK", "error", err)
		}
		// 组织用户会话
		actions = utils.BuildShiftSelectActions(
			shifts,
			nil,
			CoreEventUserConfirmed,
			CoreEventUserCancelled,
		)
		// 缺少班次信息，需要补充班次信息
		confirmMessage := "请选择您需要生成的班次？"
		// 用户需要选择正确班次
		return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, confirmMessage, actions)
	}

	// 验证人数
	if len(shiftCtx.StaffRequirements) <= 0 {
		// 人数不合法，提示用户输入
		actions := utils.BuildStaffCountActions(
			1,
			100,
			CoreEventUserConfirmed,
			CoreEventUserCancelled,
		)
		confirmMessage := "请选择班次需要的排班人数"
		return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, confirmMessage, actions)
	}

	wctx.SessionService().SetData(ctx, sessionID, session_data_key_validate_result, true)
	logger.Info("Core: Data validation succeeded")
	return nil
}

func actAfterValidate(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()

	// 如果 Act 返回了错误，这里不会被调用
	// 错误情况会由工作流引擎自动触发 CoreEventError
	validatePassed, ok, err := wctx.SessionService().GetData(ctx, wctx.Session().ID, session_data_key_validate_result)
	if err != nil {
		logger.Error("Core: Failed to get validation result from session data", "error", err)
		return err
	}
	if ok && validatePassed.(bool) {
		logger.Info("Core: Data validated, transitioning to plan generation state")
		// 触发 TodoGenerated 事件以转换到 GeneratingTodo 状态
		return wctx.Send(ctx, CoreEventTodoGenerated, nil)
	}

	return nil
}

/*********************** 2.生成 Todo ****************************/
func actTodoGenerated(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Generating Todo schedule", "sessionID", sess.ID)

	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	shift := shiftCtx.Shift
	logger.Info("Generating Todo plan for shift", "shiftName", shift.Name, "source", shiftCtx.SourceWorkflow)

	// 发送阶段开始消息
	startMsg := fmt.Sprintf("📋 生成排班计划：%s", shift.Name)
	if shiftCtx.ProgressCallback != nil {
		shiftCtx.ProgressCallback(startMsg)
	} else {
		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, startMsg); err != nil {
			logger.Warn("Failed to send start message", "error", err)
		}
	}

	// 准备数据
	staffList := d_model.NewStaffInfoListFromEmployees(shiftCtx.StaffList)
	allRules := combineRulesAsRuleInfo(shiftCtx.GlobalRules, shiftCtx.ShiftRules)
	shiftInfo := d_model.NewShiftInfoFromContext(shiftCtx)

	// 调用AI生成Todo计划
	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	todoPlanResult, err := schedulingAIService.GenerateShiftTodoPlan(ctx, shiftInfo, staffList, allRules, shiftCtx.StaffRequirements, nil, shiftCtx.FixedShiftAssignments, shiftCtx.TemporaryNeeds)
	if err != nil {
		logger.Error("Failed to generate todo plan", "error", err)
		return fmt.Errorf("failed to generate todo plan: %w", err)
	}

	// 解析并保存Todo计划
	todoPlan := parseTodoPlanFromResult(todoPlanResult, shift)
	shiftCtx.TodoPlan = todoPlan
	shiftCtx.CurrentTodoIndex = 0

	// 保存Context
	if err := utils.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
		return err
	}

	// 发送Todo计划消息
	planMsg := formatTodoPlanMarkdownFromResult(todoPlan, todoPlanResult)
	if shiftCtx.ProgressCallback != nil {
		shiftCtx.ProgressCallback(planMsg)
	} else {
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, planMsg); err != nil {
			logger.Warn("Failed to send todo plan message", "error", err)
		}
	}

	logger.Info("Todo plan generated", "shiftName", shift.Name, "todoCount", len(todoPlan.TodoList))
	return nil
}

func actAfterTodoGenerated(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	logger.Info("Core: Transitioning to execute Todo state", "sessionID", wctx.Session().ID)

	// 触发 ExecuteTodo 事件以转换到 ExecutingTodo 状态
	return wctx.Send(ctx, CoreEventExecuteTodo, nil)
}

/*********************** 2.1 调整计划 ****************************/
func actTodoAdjust(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Adjusting generated Todo schedule", "sessionID", sess.ID)

	// TODO: 实现调整 Todo 计划的逻辑
	// 目前暂时重新生成 Todo 计划
	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	// 重置 Todo 计划，重新生成
	shiftCtx.TodoPlan = nil
	shiftCtx.CurrentTodoIndex = 0

	// 保存Context
	if err := utils.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
		return err
	}

	// 触发重新生成 Todo 计划
	logger.Info("Core: Resetting Todo plan, will regenerate")
	return wctx.Send(ctx, CoreEventTodoGenerated, nil)
}

func actAfterTodoAdjust(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	logger.Info("Core: Transitioning to execute Todo state after adjustment", "sessionID", wctx.Session().ID)

	// 触发 ExecuteTodo 事件以转换到 ExecutingTodo 状态
	return wctx.Send(ctx, CoreEventExecuteTodo, nil)
}

/*********************** 3.执行 Todo ****************************/
func actTodoExecute(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Executing Todo schedule", "sessionID", sess.ID)

	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	shift := shiftCtx.Shift
	todoPlan := shiftCtx.TodoPlan

	if todoPlan == nil || len(todoPlan.TodoList) == 0 {
		logger.Warn("No todo plan found, skipping execution")
		// 如果没有 Todo，直接完成
		return wctx.Send(ctx, CoreEventTodoComplete, nil)
	}

	logger.Info("Executing todos for shift", "shiftName", shift.Name, "todoCount", len(todoPlan.TodoList))

	// 检查是否所有 Todo 都已执行完成
	if shiftCtx.CurrentTodoIndex >= len(todoPlan.TodoList) {
		logger.Info("All Todo items already completed")
		return wctx.Send(ctx, CoreEventTodoComplete, nil)
	}

	// 初始化排班草案（如果还没有）
	if shiftCtx.ResultDraft == nil {
		shiftCtx.ResultDraft = d_model.NewShiftScheduleDraft()
	}
	currentDraft := shiftCtx.ResultDraft

	// 准备数据
	shiftInfo := d_model.NewShiftInfoFromContext(shiftCtx)
	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)

	// 执行当前 Todo（从 CurrentTodoIndex 开始）
	todoIdx := shiftCtx.CurrentTodoIndex
	todo := todoPlan.TodoList[todoIdx]
	logger.Info("Executing todo", "index", todoIdx+1, "total", len(todoPlan.TodoList), "title", todo.Title)

	// 发送进度消息
	todoMsg := fmt.Sprintf("⚙️ 任务 %d/%d", todoIdx+1, len(todoPlan.TodoList))
	if shiftCtx.ProgressCallback != nil {
		shiftCtx.ProgressCallback(todoMsg)
	} else {
		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, todoMsg); err != nil {
			logger.Warn("Failed to send todo progress message", "error", err)
		}
	}

	// 准备当前可用人员（排除已在当前草案中安排的人员）
	currentAvailableStaff := filterAvailableStaffForShift(shiftCtx.StaffList, currentDraft)
	currentAvailableStaffList := d_model.NewStaffInfoListWithScheduleMarks(currentAvailableStaff, shiftCtx.ExistingScheduleMarks)

	// 执行Todo任务（V2不使用allShifts和workingDraft，传递nil）
	todoResult, err := schedulingAIService.ExecuteTodoTask(ctx, todo, shiftInfo, currentAvailableStaffList, currentDraft, shiftCtx.StaffRequirements, shiftCtx.FixedShiftAssignments, shiftCtx.TemporaryNeeds, shiftCtx.AllStaffList, nil, nil)
	if err != nil {
		logger.Error("Failed to execute todo", "todoId", todo.ID, "error", err)
		todo.Status = "failed"
		todo.Result = fmt.Sprintf("执行失败: %v", err)

		failMsg := fmt.Sprintf("   ❌ **执行失败**: %v\n\n   将继续执行下一个任务...", err)
		if shiftCtx.ProgressCallback != nil {
			shiftCtx.ProgressCallback(failMsg)
		} else {
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, failMsg); err != nil {
				logger.Warn("Failed to send error message", "error", err)
			}
		}

		shiftCtx.TodoExecutionLogs = append(shiftCtx.TodoExecutionLogs,
			fmt.Sprintf("Todo %d 失败: %s", todoIdx+1, err.Error()))
	} else {
		// 合并执行结果（强类型）
		currentDraft.MergeTodoResult(todoResult)

		// 更新Todo状态
		todo.Status = "completed"
		explanation := todoResult.Explanation
		todo.Result = explanation

		// 发送执行结果（仅非进度回调模式）
		if shiftCtx.ProgressCallback == nil && len(currentDraft.Schedule) > 0 {
			todoScheduleData := convertDraftToScheduleData(shift, currentDraft, shiftCtx.StartDate, shiftCtx.EndDate, shiftCtx.AllStaffList, shiftCtx.StaffRequirements)

			contentParts := []string{
				fmt.Sprintf("📋 任务 %d 执行结果 - 当前已安排 %d 天的排班", todoIdx+1, len(currentDraft.Schedule)),
			}
			if explanation != "" {
				contentParts = append(contentParts, fmt.Sprintf("\n\n**说明**: %s", explanation))
			}

			todoScheduleMsg := session.Message{
				Role:    session.RoleAssistant,
				Content: strings.Join(contentParts, ""),
				Actions: []session.WorkflowAction{
					{
						ID:      fmt.Sprintf("view_shift_schedule_%s_todo_%d", shift.ID, todoIdx+1),
						Type:    session.ActionTypeQuery,
						Label:   "📊 查看排班详情",
						Payload: todoScheduleData,
						Style:   session.ActionStyleSuccess,
					},
				},
				Metadata: map[string]any{
					"type":         "shiftSchedule",
					"shiftId":      shift.ID,
					"shiftName":    shift.Name,
					"scheduleData": todoScheduleData,
					"isInterim":    true,
					"todoIndex":    todoIdx + 1,
					"todoTotal":    len(todoPlan.TodoList),
				},
			}
			if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, todoScheduleMsg); err != nil {
				logger.Warn("Failed to send todo schedule data", "error", err)
			}
		} else if shiftCtx.ProgressCallback == nil && explanation != "" {
			explanationMsg := fmt.Sprintf("   **说明**: %s", explanation)
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, explanationMsg); err != nil {
				logger.Warn("Failed to send explanation message", "error", err)
			}
		}

		shiftCtx.TodoExecutionLogs = append(shiftCtx.TodoExecutionLogs,
			fmt.Sprintf("Todo %d 完成: %s", todoIdx+1, todo.Title))
	}

	// 更新当前 Todo 索引
	shiftCtx.CurrentTodoIndex = todoIdx + 1

	// 保存结果草案
	shiftCtx.ResultDraft = currentDraft

	// 保存Context
	if err := utils.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
		return err
	}

	// 检查是否所有 Todo 都已完成
	if shiftCtx.CurrentTodoIndex >= len(todoPlan.TodoList) {
		// 发送阶段完成汇总
		completedCount := 0
		failedCount := 0
		for _, todo := range todoPlan.TodoList {
			switch todo.Status {
			case "completed":
				completedCount++
			case "failed":
				failedCount++
			}
		}
		summaryMsg := "班次排班任务完成"
		if shiftCtx.ProgressCallback != nil {
			shiftCtx.ProgressCallback(summaryMsg)
		} else {
			if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, summaryMsg); err != nil {
				logger.Warn("Failed to send summary message", "error", err)
			}
		}

		// // 发送班次完成的完整排班预览（仅非回调模式）
		// if shiftCtx.ProgressCallback == nil && len(currentDraft.Schedule) > 0 {
		// 	shiftScheduleData := convertDraftToScheduleData(shift, currentDraft, shiftCtx.StartDate, shiftCtx.EndDate, shiftCtx.AllStaffList, shiftCtx.StaffRequirements)
		// 	scheduleDataMsg := session.Message{
		// 		Role:    session.RoleAssistant,
		// 		Content: fmt.Sprintf("📊 班次【%s】排班完成，共排班 %d 天", shift.Name, len(currentDraft.Schedule)),
		// 		Actions: []session.WorkflowAction{
		// 			{
		// 				ID:      fmt.Sprintf("view_shift_schedule_%s_complete", shift.ID),
		// 				Type:    session.ActionTypeQuery,
		// 				Label:   "📊 查看完整排班",
		// 				Payload: shiftScheduleData,
		// 				Style:   session.ActionStyleSuccess,
		// 			},
		// 		},
		// 		Metadata: map[string]any{
		// 			"type":          "shiftSchedule",
		// 			"shiftId":       shift.ID,
		// 			"shiftName":     shift.Name,
		// 			"scheduleData":  shiftScheduleData,
		// 			"isInterim":     true,
		// 			"completeTodos": completedCount,
		// 			"totalTodos":    len(todoPlan.TodoList),
		// 		},
		// 	}
		// 	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, scheduleDataMsg); err != nil {
		// 		logger.Warn("Failed to send shift schedule data", "error", err)
		// 	}
		// }

		logger.Info("Todos execution completed", "shiftName", shift.Name)
		// 所有 Todo 完成，触发完成事件
		return wctx.Send(ctx, CoreEventTodoComplete, nil)
	}

	// 还有更多 Todo 需要执行，继续执行下一个
	return wctx.Send(ctx, CoreEventExecuteTodo, nil)
}

func actAfterTodoExecute(ctx context.Context, wctx engine.Context, payload any) error {
	// AfterAct 不需要做任何事情，因为 actTodoExecute 已经处理了状态转换
	if _, err := wctx.SessionService().AddSystemMessage(ctx, wctx.Session().ID, "正在初步校正排班结果"); err != nil {
		logger.Warn("Failed to send summary message", "error", err)
	}
	return nil
}

/************************ 3.1 用户请求 ****************************/
func actUserRequestExecute(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Executing user request on Todo schedule", "sessionID", sess.ID)

	// TODO: 实现执行用户请求的逻辑
	// 目前暂时记录日志，然后继续执行
	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	// 解析用户请求（从 payload 中获取）
	// 这里可以根据实际的 payload 结构来处理用户请求
	// 例如：调整特定日期的排班、替换人员等

	logger.Info("Core: User request received, processing...", "shiftName", shiftCtx.Shift.Name)

	// 保存Context
	if err := utils.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
		return err
	}

	logger.Info("Core: User request processed successfully")
	return nil
}

/************************ 4.完成排班 ****************************/
func actCoreComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Completing scheduling workflow", "sessionID", sess.ID)

	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context", "error", err)
		return err
	}

	shift := shiftCtx.Shift
	logger.Info("Validating shift schedule", "shiftName", shift.Name)

	if shiftCtx.ResultDraft == nil {
		shiftCtx.ResultDraft = d_model.NewShiftScheduleDraft()
	}

	// 如果草案为空，发送警告
	if len(shiftCtx.ResultDraft.Schedule) == 0 {
		warnMsg := "⚠️ **警告**: 当前排班草案为空，可能是 TODO 执行过程中未能生成有效的排班数据。"
		if shiftCtx.ProgressCallback != nil {
			shiftCtx.ProgressCallback(warnMsg)
		} else {
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, warnMsg); err != nil {
				logger.Warn("Failed to send warning message", "error", err)
			}
		}
	}

	// 保存Context
	if err := utils.SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
		return err
	}

	logger.Info("Shift validation completed", "shiftName", shift.Name)
	return nil
}

func actCoreAfterComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Scheduling core sub-workflow completed", "sessionID", sess.ID)

	// 获取排班结果（在 actCoreComplete 中已经保存到 session）
	shiftCtx, err := utils.GetShiftSchedulingContext(sess)
	if err != nil {
		logger.Error("Core: Failed to get shift scheduling context for return", "error", err)
		// 即使获取失败，也尝试返回父工作流（使用空结果）
		return returnToParentWithResult(ctx, wctx, nil, err)
	}

	// 构建返回结果，包含 shift_scheduling_context
	// 父工作流会从 session 的 shift_scheduling_context key 中读取结果
	output := make(map[string]any)
	if shiftCtx != nil {
		if shiftCtx.Shift != nil {
			output["shift_id"] = shiftCtx.Shift.ID
			output["shift_name"] = shiftCtx.Shift.Name
		}
		// 注意：shift_scheduling_context 已经通过 SaveShiftSchedulingContext 保存到 session
		// 父工作流会从 session.Data["shift_scheduling_context"] 读取
	}

	// 返回父工作流
	return returnToParentWithResult(ctx, wctx, output, nil)
}

// returnToParentWithResult 辅助函数：返回父工作流
func returnToParentWithResult(ctx context.Context, wctx engine.Context, output map[string]any, err error) error {
	logger := wctx.Logger()

	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Warn("Context is not an Actor, cannot return to parent workflow")
		if err != nil {
			return err
		}
		return nil
	}

	var result *engine.SubWorkflowResult
	if err != nil {
		result = engine.NewSubWorkflowError(err)
	} else {
		result = engine.NewSubWorkflowResult(output)
	}

	return actor.ReturnToParent(ctx, result)
}

/************************ 全局错误处理 ****************************/
func actCoreHandleError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()
	logger.Info("Core: Handling error in scheduling core sub-workflow", "sessionID", sess.ID)

	var errMsg string
	if err, ok := payload.(error); ok {
		errMsg = err.Error()
	} else if s, ok := payload.(string); ok {
		errMsg = s
	} else {
		errMsg = "unknown error"
	}

	logger.Error("Core: Schedule core failed",
		"sessionID", sess.ID,
		"error", errMsg,
	)

	// 保存错误信息到 session
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "core_error", errMsg); err != nil {
		logger.Warn("Failed to save error to session", "error", err)
	}

	// 获取当前班次信息用于提示
	shiftCtx, _ := utils.GetShiftSchedulingContext(sess)
	shiftName := "当前班次"
	if shiftCtx != nil && shiftCtx.Shift != nil {
		shiftName = shiftCtx.Shift.Name
	}

	// 发送错误消息，包含重试/跳过按钮
	message := fmt.Sprintf("❌ 排班过程中遇到错误：\n\n%s\n\n请选择操作：", errMsg)
	actions := []session.WorkflowAction{
		{
			ID:    "retry",
			Type:  session.ActionTypeWorkflow,
			Label: "🔄 重试",
			Event: session.WorkflowEvent(CoreEventRetry),
			Style: session.ActionStylePrimary,
		},
		{
			ID:    "skip",
			Type:  session.ActionTypeWorkflow,
			Label: fmt.Sprintf("⏭️ 跳过【%s】", shiftName),
			Event: session.WorkflowEvent(CoreEventSkip),
			Style: session.ActionStyleSecondary,
		},
	}
	if _, err := wctx.SessionService().AddAssistantMessageWithActions(ctx, sess.ID, message, actions); err != nil {
		logger.Warn("Failed to send error options message", "error", err)
	}

	return nil // 等待用户选择
}

// ============================================================
// 辅助函数
// ============================================================

// combineRulesAsRuleInfo 合并全局规则和班次规则为 RuleInfo 列表
func combineRulesAsRuleInfo(globalRules, shiftRules []*d_model.Rule) []*d_model.RuleInfo {
	allRules := make([]*d_model.Rule, 0, len(globalRules)+len(shiftRules))
	allRules = append(allRules, globalRules...)
	allRules = append(allRules, shiftRules...)
	return d_model.NewRuleInfoListFromRules(allRules)
}

// filterAvailableStaffForShift 过滤出当前可用的人员（排除已在草案中安排的人员）
func filterAvailableStaffForShift(allStaff []*d_model.Employee, currentDraft *d_model.ShiftScheduleDraft) []*d_model.Employee {
	available := make([]*d_model.Employee, 0)
	for _, staff := range allStaff {
		if currentDraft.IsStaffScheduled(staff.ID) {
			continue
		}
		available = append(available, staff)
	}
	return available
}

// parseTodoPlanFromResult 从 TodoPlanResult 解析为 ShiftTodoPlan
func parseTodoPlanFromResult(result *d_model.TodoPlanResult, shift *d_model.Shift) *d_model.ShiftTodoPlan {
	// 使用 TodoPlanResult 内置的转换方法
	return result.ToShiftTodoPlan(shift.ID, shift.Name)
}

// formatTodoPlanMarkdownFromResult 从 TodoPlanResult 格式化 Markdown 消息
func formatTodoPlanMarkdownFromResult(todoPlan *d_model.ShiftTodoPlan, aiResult *d_model.TodoPlanResult) string {
	var msg strings.Builder

	msg.WriteString("### ✨ 排班计划生成完成\n\n")

	if aiResult.Reasoning != "" {
		msg.WriteString(fmt.Sprintf("**AI分析**:\n%s\n\n", aiResult.Reasoning))
	}

	msg.WriteString(fmt.Sprintf("**任务清单**（共%d个任务）:\n\n", len(todoPlan.TodoList)))
	for i, todo := range todoPlan.TodoList {
		msg.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, todo.Title))
		if todo.Description != "" {
			msg.WriteString(fmt.Sprintf("   - %s\n", todo.Description))
		}
	}

	if todoPlan.PlanSummary != "" {
		msg.WriteString(fmt.Sprintf("\n**总体说明**:\n%s\n", todoPlan.PlanSummary))
	}

	return msg.String()
}

// convertDraftToScheduleData 将排班草案转换为前端可用的格式
func convertDraftToScheduleData(shift *d_model.Shift, draft *d_model.ShiftScheduleDraft, startDate, endDate string, staffList []*d_model.Employee, staffRequirements map[string]int) map[string]any {
	staffNames := make(map[string]string)
	for _, s := range staffList {
		staffNames[s.ID] = s.Name
	}

	days := make(map[string]any)
	for date, staffIDs := range draft.Schedule {
		names := make([]string, 0, len(staffIDs))
		for _, id := range staffIDs {
			if name, ok := staffNames[id]; ok {
				names = append(names, name)
			} else {
				names = append(names, id)
			}
		}

		requiredCount := 0
		if req, ok := staffRequirements[date]; ok {
			requiredCount = req
		}

		days[date] = map[string]any{
			"staff":         names,
			"staffIds":      staffIDs,
			"requiredCount": requiredCount,
			"actualCount":   len(staffIDs),
		}
	}

	return map[string]any{
		"shiftId":   shift.ID,
		"shiftName": shift.Name,
		"startDate": startDate,
		"endDate":   endDate,
		"schedule": map[string]any{
			"shiftId":  shift.ID,
			"priority": shift.SchedulingPriority,
			"days":     days,
		},
	}
}
