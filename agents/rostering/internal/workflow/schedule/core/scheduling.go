// Package core 提供排班核心逻辑，可被 Create 和 Adjust 工作流复用
package core

// import (
// 	"context"
// 	"fmt"
// 	"sort"
// 	"strings"

// 	"jusha/mcp/pkg/workflow/engine"
// 	"jusha/mcp/pkg/workflow/session"

// 	d_model "jusha/agent/rostering/domain/model"
// 	d_service "jusha/agent/rostering/domain/service"
// )

// // ============================================================
// // ShiftSchedulingContext 辅助函数
// // ============================================================

// // GetShiftSchedulingContext 从 session 获取共享排班上下文
// func GetShiftSchedulingContext(sess *session.Session) (*d_model.ShiftSchedulingContext, error) {
// 	if ctx, ok := sess.Data[d_model.DataKeyShiftSchedulingContext]; ok {
// 		if shiftCtx, ok := ctx.(*d_model.ShiftSchedulingContext); ok {
// 			return shiftCtx, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("shift scheduling context not found in session")
// }

// // SaveShiftSchedulingContext 保存共享排班上下文到 session
// func SaveShiftSchedulingContext(ctx context.Context, wctx engine.Context, shiftCtx *d_model.ShiftSchedulingContext) error {
// 	sess := wctx.Session()
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
// 		return fmt.Errorf("failed to save shift scheduling context: %w", err)
// 	}
// 	return nil
// }

// // ClearShiftSchedulingContext 清除共享排班上下文
// func ClearShiftSchedulingContext(ctx context.Context, wctx engine.Context) error {
// 	sess := wctx.Session()
// 	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, nil); err != nil {
// 		return fmt.Errorf("failed to clear shift scheduling context: %w", err)
// 	}
// 	return nil
// }

// // ============================================================
// // 三阶段排班核心函数
// // ============================================================

// // GenerateShiftTodoPlan 阶段1：为班次生成Todo计划
// func GenerateShiftTodoPlan(ctx context.Context, wctx engine.Context) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	shift := shiftCtx.Shift
// 	logger.Info("Generating Todo plan for shift", "shiftName", shift.Name, "source", shiftCtx.SourceWorkflow)

// 	// 发送阶段开始消息
// 	startMsg := fmt.Sprintf("### 📋 班次【%s】排班计划生成\n\n正在分析班次规则、人员情况和历史排班...", shift.Name)
// 	if shiftCtx.ProgressCallback != nil {
// 		shiftCtx.ProgressCallback(startMsg)
// 	} else {
// 		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, startMsg); err != nil {
// 			logger.Warn("Failed to send start message", "error", err)
// 		}
// 	}

// 	// 准备数据
// 	staffList := d_model.NewStaffInfoListFromEmployees(shiftCtx.StaffList)
// 	allRules := combineRulesAsRuleInfo(shiftCtx.GlobalRules, shiftCtx.ShiftRules)
// 	shiftInfo := d_model.NewShiftInfoFromContext(shiftCtx)

// 	// 调用AI生成Todo计划
// 	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
// 	todoPlanResult, err := schedulingAIService.GenerateShiftTodoPlan(ctx, shiftInfo, staffList, allRules, shiftCtx.StaffRequirements, nil)
// 	if err != nil {
// 		logger.Error("Failed to generate todo plan", "error", err)
// 		return fmt.Errorf("failed to generate todo plan: %w", err)
// 	}

// 	// 解析并保存Todo计划
// 	todoPlan := parseTodoPlanFromResult(todoPlanResult, shift)
// 	shiftCtx.TodoPlan = todoPlan
// 	shiftCtx.CurrentTodoIndex = 0

// 	// 保存Context
// 	if err := SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return err
// 	}

// 	// 发送Todo计划消息
// 	planMsg := formatTodoPlanMarkdownFromResult(shift.Name, todoPlan, todoPlanResult)
// 	if shiftCtx.ProgressCallback != nil {
// 		shiftCtx.ProgressCallback(planMsg)
// 	} else {
// 		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, planMsg); err != nil {
// 			logger.Warn("Failed to send todo plan message", "error", err)
// 		}
// 	}

// 	logger.Info("Todo plan generated", "shiftName", shift.Name, "todoCount", len(todoPlan.TodoList))

// 	return nil
// }

// // ExecuteShiftTodos 阶段2：执行所有Todo任务
// func ExecuteShiftTodos(ctx context.Context, wctx engine.Context) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	shift := shiftCtx.Shift
// 	todoPlan := shiftCtx.TodoPlan

// 	if todoPlan == nil || len(todoPlan.TodoList) == 0 {
// 		logger.Warn("No todo plan found, skipping execution")
// 		return nil
// 	}

// 	logger.Info("Executing todos for shift", "shiftName", shift.Name, "todoCount", len(todoPlan.TodoList))

// 	// 初始化排班草案
// 	currentDraft := d_model.NewShiftScheduleDraft()

// 	// 准备数据
// 	shiftInfo := d_model.NewShiftInfoFromContext(shiftCtx)
// 	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)

// 	// 依次执行每个Todo
// 	for todoIdx, todo := range todoPlan.TodoList {
// 		shiftCtx.CurrentTodoIndex = todoIdx
// 		logger.Info("Executing todo", "index", todoIdx+1, "total", len(todoPlan.TodoList), "title", todo.Title)

// 		// 发送进度消息
// 		todoMsg := fmt.Sprintf("⚙️ 执行任务 %d/%d: %s", todoIdx+1, len(todoPlan.TodoList), todo.Title)
// 		if shiftCtx.ProgressCallback != nil {
// 			shiftCtx.ProgressCallback(todoMsg)
// 		} else {
// 			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, todoMsg); err != nil {
// 				logger.Warn("Failed to send todo progress message", "error", err)
// 			}
// 		}

// 		// 准备当前可用人员（排除已在当前草案中安排的人员）
// 		currentAvailableStaff := filterAvailableStaffForShift(shiftCtx.StaffList, currentDraft)
// 		currentAvailableStaffList := d_model.NewStaffInfoListWithScheduleMarks(currentAvailableStaff, shiftCtx.ExistingScheduleMarks)

// 		// 执行Todo任务 - 直接传递 todo 本身
// 		todoResult, err := schedulingAIService.ExecuteTodoTask(ctx, todo, shiftInfo, currentAvailableStaffList, currentDraft, shiftCtx.StaffRequirements)
// 		if err != nil {
// 			logger.Error("Failed to execute todo", "todoId", todo.ID, "error", err)
// 			todo.Status = "failed"
// 			todo.Result = fmt.Sprintf("执行失败: %v", err)

// 			failMsg := fmt.Sprintf("   ❌ **执行失败**: %v\n\n   将继续执行下一个任务...", err)
// 			if shiftCtx.ProgressCallback != nil {
// 				shiftCtx.ProgressCallback(failMsg)
// 			} else {
// 				if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, failMsg); err != nil {
// 					logger.Warn("Failed to send error message", "error", err)
// 				}
// 			}

// 			shiftCtx.TodoExecutionLogs = append(shiftCtx.TodoExecutionLogs,
// 				fmt.Sprintf("Todo %d 失败: %s", todoIdx+1, err.Error()))
// 			continue
// 		}

// 		// 合并执行结果（强类型）
// 		currentDraft.MergeTodoResult(todoResult)

// 		// 更新Todo状态
// 		todo.Status = "completed"
// 		explanation := todoResult.Explanation
// 		todo.Result = explanation

// 		// 日志：记录 todo 执行结果
// 		logger.Info("Todo execution result merged",
// 			"todoIndex", todoIdx+1,
// 			"scheduleCount", len(currentDraft.Schedule),
// 			"todoScheduleCount", len(todoResult.Schedule),
// 			"hasProgressCallback", shiftCtx.ProgressCallback != nil,
// 		)

// 		// 发送执行结果（仅非进度回调模式）
// 		if shiftCtx.ProgressCallback == nil && len(currentDraft.Schedule) > 0 {
// 			todoScheduleData := convertDraftToScheduleData(shift, currentDraft, shiftCtx.StartDate, shiftCtx.EndDate, shiftCtx.StaffList, shiftCtx.StaffRequirements)

// 			contentParts := []string{
// 				fmt.Sprintf("📋 任务 %d 执行结果 - 当前已安排 %d 天的排班", todoIdx+1, len(currentDraft.Schedule)),
// 			}
// 			if explanation != "" {
// 				contentParts = append(contentParts, fmt.Sprintf("\n\n**说明**: %s", explanation))
// 			}

// 			todoScheduleMsg := session.Message{
// 				Role:    session.RoleAssistant,
// 				Content: strings.Join(contentParts, ""),
// 				Actions: []session.WorkflowAction{
// 					{
// 						ID:      fmt.Sprintf("view_shift_schedule_%s_todo_%d", shift.ID, todoIdx+1),
// 						Type:    session.ActionTypeQuery,
// 						Label:   "📊 查看排班详情",
// 						Payload: todoScheduleData,
// 						Style:   session.ActionStyleInfo,
// 					},
// 				},
// 				Metadata: map[string]any{
// 					"type":         "shiftSchedule",
// 					"shiftId":      shift.ID,
// 					"shiftName":    shift.Name,
// 					"scheduleData": todoScheduleData,
// 					"isInterim":    true,
// 					"todoIndex":    todoIdx + 1,
// 					"todoTotal":    len(todoPlan.TodoList),
// 				},
// 			}
// 			if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, todoScheduleMsg); err != nil {
// 				logger.Warn("Failed to send todo schedule data", "error", err)
// 			}
// 		} else if shiftCtx.ProgressCallback == nil && explanation != "" {
// 			explanationMsg := fmt.Sprintf("   **说明**: %s", explanation)
// 			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, explanationMsg); err != nil {
// 				logger.Warn("Failed to send explanation message", "error", err)
// 			}
// 		}

// 		shiftCtx.TodoExecutionLogs = append(shiftCtx.TodoExecutionLogs,
// 			fmt.Sprintf("Todo %d 完成: %s", todoIdx+1, todo.Title))
// 	}

// 	// 保存结果草案
// 	shiftCtx.ResultDraft = currentDraft

// 	// 标记所有 Todo 执行完成（设置为总数，表示全部完成，避免死循环）
// 	shiftCtx.CurrentTodoIndex = len(todoPlan.TodoList)

// 	// 保存Context
// 	if err := SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return err
// 	}

// 	// 发送阶段完成汇总
// 	completedCount := 0
// 	failedCount := 0
// 	for _, todo := range todoPlan.TodoList {
// 		switch todo.Status {
// 		case "completed":
// 			completedCount++
// 		case "failed":
// 			failedCount++
// 		}
// 	}
// 	summaryMsg := fmt.Sprintf("\n### ✅ 任务执行完成\n\n- 总计: %d 个任务\n- 成功: %d 个\n- 失败: %d 个",
// 		len(todoPlan.TodoList), completedCount, failedCount)
// 	if shiftCtx.ProgressCallback != nil {
// 		shiftCtx.ProgressCallback(summaryMsg)
// 	} else {
// 		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, summaryMsg); err != nil {
// 			logger.Warn("Failed to send summary message", "error", err)
// 		}
// 	}

// 	// 发送班次完成的完整排班预览（仅非回调模式）
// 	if shiftCtx.ProgressCallback == nil && len(currentDraft.Schedule) > 0 {
// 		shiftScheduleData := convertDraftToScheduleData(shift, currentDraft, shiftCtx.StartDate, shiftCtx.EndDate, shiftCtx.StaffList, shiftCtx.StaffRequirements)
// 		scheduleDataMsg := session.Message{
// 			Role:    session.RoleAssistant,
// 			Content: fmt.Sprintf("📊 班次【%s】排班完成，共排班 %d 天", shift.Name, len(currentDraft.Schedule)),
// 			Actions: []session.WorkflowAction{
// 				{
// 					ID:      fmt.Sprintf("view_shift_schedule_%s_complete", shift.ID),
// 					Type:    session.ActionTypeQuery,
// 					Label:   "📊 查看完整排班",
// 					Payload: shiftScheduleData,
// 					Style:   session.ActionStyleSuccess,
// 				},
// 			},
// 			Metadata: map[string]any{
// 				"type":          "shiftSchedule",
// 				"shiftId":       shift.ID,
// 				"shiftName":     shift.Name,
// 				"scheduleData":  shiftScheduleData,
// 				"isInterim":     true,
// 				"completeTodos": completedCount,
// 				"totalTodos":    len(todoPlan.TodoList),
// 			},
// 		}
// 		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, scheduleDataMsg); err != nil {
// 			logger.Warn("Failed to send shift schedule data", "error", err)
// 		}
// 	}

// 	logger.Info("Todos execution completed", "shiftName", shift.Name)

// 	return nil
// }

// // ValidateShiftSchedule 阶段3：校验班次排班结果
// func ValidateShiftSchedule(ctx context.Context, wctx engine.Context) error {
// 	logger := wctx.Logger()
// 	sess := wctx.Session()

// 	shiftCtx, err := GetShiftSchedulingContext(sess)
// 	if err != nil {
// 		return err
// 	}

// 	shift := shiftCtx.Shift
// 	logger.Info("Validating shift schedule", "shiftName", shift.Name)

// 	if shiftCtx.ResultDraft == nil {
// 		shiftCtx.ResultDraft = d_model.NewShiftScheduleDraft()
// 	}

// 	// 如果草案为空，发送警告
// 	if len(shiftCtx.ResultDraft.Schedule) == 0 {
// 		warnMsg := "⚠️ **警告**: 当前排班草案为空，可能是 TODO 执行过程中未能生成有效的排班数据。"
// 		if shiftCtx.ProgressCallback != nil {
// 			shiftCtx.ProgressCallback(warnMsg)
// 		} else {
// 			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, warnMsg); err != nil {
// 				logger.Warn("Failed to send warning message", "error", err)
// 			}
// 		}
// 	}

// 	// 发送校验完成消息（AI校验已禁用）
// 	validateMsg := fmt.Sprintf("### ✅ 班次【%s】排班完成\n\n排班数据已生成，跳过AI校验环节。", shift.Name)
// 	if shiftCtx.ProgressCallback != nil {
// 		shiftCtx.ProgressCallback(validateMsg)
// 	} else {
// 		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, validateMsg); err != nil {
// 			logger.Warn("Failed to send validation message", "error", err)
// 		}
// 	}

// 	// 保存Context
// 	if err := SaveShiftSchedulingContext(ctx, wctx, shiftCtx); err != nil {
// 		return err
// 	}

// 	logger.Info("Shift validation completed", "shiftName", shift.Name)

// 	return nil
// }

// // ============================================================
// // BuildExistingScheduleMarks 从草案构建其他班次的排班标记
// // ============================================================

// // BuildExistingScheduleMarks 从 ScheduleDraft 构建其他班次的已排班标记
// func BuildExistingScheduleMarks(draft *d_model.ScheduleDraft, excludeShiftID string, shifts []*d_model.Shift) map[string]map[string][]*d_model.ShiftMark {
// 	marks := make(map[string]map[string][]*d_model.ShiftMark)

// 	if draft == nil || draft.Shifts == nil {
// 		return marks
// 	}

// 	shiftMap := make(map[string]*d_model.Shift)
// 	for _, s := range shifts {
// 		shiftMap[s.ID] = s
// 	}

// 	for shiftID, shiftDraft := range draft.Shifts {
// 		if shiftID == excludeShiftID {
// 			continue
// 		}

// 		shift := shiftMap[shiftID]
// 		if shift == nil {
// 			continue
// 		}

// 		for date, dayShift := range shiftDraft.Days {
// 			for _, staffID := range dayShift.StaffIDs {
// 				if marks[staffID] == nil {
// 					marks[staffID] = make(map[string][]*d_model.ShiftMark)
// 				}
// 				marks[staffID][date] = append(marks[staffID][date], &d_model.ShiftMark{
// 					ShiftID:   shiftID,
// 					ShiftName: shift.Name,
// 					StartTime: shift.StartTime,
// 					EndTime:   shift.EndTime,
// 				})
// 			}
// 		}
// 	}

// 	return marks
// }

// // ============================================================
// // FormatScheduleDiff 格式化排班差异对比
// // ============================================================

// // ScheduleDiffSummary 排班差异汇总
// type ScheduleDiffSummary struct {
// 	AffectedDays  int                  `json:"affectedDays"`
// 	AddedCount    int                  `json:"addedCount"`
// 	RemovedCount  int                  `json:"removedCount"`
// 	ReplacedCount int                  `json:"replacedCount"`
// 	DayChanges    []*DayScheduleChange `json:"dayChanges"`
// }

// // DayScheduleChange 单日排班变更
// type DayScheduleChange struct {
// 	Date     string   `json:"date"`
// 	Added    []string `json:"added"`
// 	Removed  []string `json:"removed"`
// 	Replaced []string `json:"replaced"`
// }

// // FormatScheduleDiff 格式化两个班次草案的差异
// func FormatScheduleDiff(original, current *d_model.ShiftDraft, staffList []*d_model.Employee) *ScheduleDiffSummary {
// 	if original == nil && current == nil {
// 		return nil
// 	}

// 	staffNames := make(map[string]string)
// 	for _, s := range staffList {
// 		staffNames[s.ID] = s.Name
// 	}

// 	summary := &ScheduleDiffSummary{
// 		DayChanges: make([]*DayScheduleChange, 0),
// 	}

// 	allDates := make(map[string]bool)
// 	if original != nil {
// 		for date := range original.Days {
// 			allDates[date] = true
// 		}
// 	}
// 	if current != nil {
// 		for date := range current.Days {
// 			allDates[date] = true
// 		}
// 	}

// 	dates := make([]string, 0, len(allDates))
// 	for date := range allDates {
// 		dates = append(dates, date)
// 	}
// 	sort.Strings(dates)

// 	for _, date := range dates {
// 		var originalStaff, currentStaff []string
// 		if original != nil && original.Days[date] != nil {
// 			originalStaff = original.Days[date].StaffIDs
// 		}
// 		if current != nil && current.Days[date] != nil {
// 			currentStaff = current.Days[date].StaffIDs
// 		}

// 		originalSet := make(map[string]bool)
// 		currentSet := make(map[string]bool)
// 		for _, id := range originalStaff {
// 			originalSet[id] = true
// 		}
// 		for _, id := range currentStaff {
// 			currentSet[id] = true
// 		}

// 		dayChange := &DayScheduleChange{
// 			Date:     date,
// 			Added:    make([]string, 0),
// 			Removed:  make([]string, 0),
// 			Replaced: make([]string, 0),
// 		}

// 		for _, id := range currentStaff {
// 			if !originalSet[id] {
// 				name := staffNames[id]
// 				if name == "" {
// 					name = id
// 				}
// 				dayChange.Added = append(dayChange.Added, name)
// 				summary.AddedCount++
// 			}
// 		}

// 		for _, id := range originalStaff {
// 			if !currentSet[id] {
// 				name := staffNames[id]
// 				if name == "" {
// 					name = id
// 				}
// 				dayChange.Removed = append(dayChange.Removed, name)
// 				summary.RemovedCount++
// 			}
// 		}

// 		if len(dayChange.Added) > 0 || len(dayChange.Removed) > 0 {
// 			summary.DayChanges = append(summary.DayChanges, dayChange)
// 			summary.AffectedDays++
// 		}
// 	}

// 	return summary
// }

// // FormatScheduleDiffMessage 格式化差异对比消息
// func FormatScheduleDiffMessage(diff *ScheduleDiffSummary, shiftName string) string {
// 	if diff == nil || (diff.AddedCount == 0 && diff.RemovedCount == 0) {
// 		return fmt.Sprintf("班次【%s】排班无变化", shiftName)
// 	}

// 	var msg strings.Builder
// 	msg.WriteString(fmt.Sprintf("### 📊 班次【%s】重排对比\n\n", shiftName))
// 	msg.WriteString(fmt.Sprintf("**汇总**: 共调整 %d 天，新增 %d 人次，移除 %d 人次\n\n", diff.AffectedDays, diff.AddedCount, diff.RemovedCount))

// 	msg.WriteString("**变更详情**:\n")
// 	for _, dayChange := range diff.DayChanges {
// 		changes := make([]string, 0)
// 		for _, name := range dayChange.Added {
// 			changes = append(changes, fmt.Sprintf("+%s", name))
// 		}
// 		for _, name := range dayChange.Removed {
// 			changes = append(changes, fmt.Sprintf("-%s", name))
// 		}
// 		msg.WriteString(fmt.Sprintf("- `%s`: %s\n", dayChange.Date, strings.Join(changes, ", ")))
// 	}

// 	return msg.String()
// }

// // ============================================================
// // 辅助函数
// // ============================================================

// // combineRulesAsRuleInfo 合并全局规则和班次规则为 RuleInfo 列表
// func combineRulesAsRuleInfo(globalRules, shiftRules []*d_model.Rule) []*d_model.RuleInfo {
// 	allRules := make([]*d_model.Rule, 0, len(globalRules)+len(shiftRules))
// 	allRules = append(allRules, globalRules...)
// 	allRules = append(allRules, shiftRules...)
// 	return d_model.NewRuleInfoListFromRules(allRules)
// }

// func filterAvailableStaffForShift(allStaff []*d_model.Employee, currentDraft *d_model.ShiftScheduleDraft) []*d_model.Employee {
// 	available := make([]*d_model.Employee, 0)
// 	for _, staff := range allStaff {
// 		if currentDraft.IsStaffScheduled(staff.ID) {
// 			continue
// 		}
// 		available = append(available, staff)
// 	}
// 	return available
// }

// // parseTodoPlanFromResult 从 TodoPlanResult 解析为 ShiftTodoPlan
// func parseTodoPlanFromResult(result *d_model.TodoPlanResult, shift *d_model.Shift) *d_model.ShiftTodoPlan {
// 	// 使用 TodoPlanResult 内置的转换方法
// 	return result.ToShiftTodoPlan(shift.ID, shift.Name)
// }

// // formatTodoPlanMarkdownFromResult 从 TodoPlanResult 格式化 Markdown 消息
// func formatTodoPlanMarkdownFromResult(shiftName string, todoPlan *d_model.ShiftTodoPlan, aiResult *d_model.TodoPlanResult) string {
// 	var msg strings.Builder

// 	msg.WriteString("### ✨ 排班计划生成完成\n\n")

// 	if aiResult.Reasoning != "" {
// 		msg.WriteString(fmt.Sprintf("**AI分析**:\n%s\n\n", aiResult.Reasoning))
// 	}

// 	msg.WriteString(fmt.Sprintf("**任务清单**（共%d个任务）:\n\n", len(todoPlan.TodoList)))
// 	for i, todo := range todoPlan.TodoList {
// 		msg.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, todo.Title))
// 		if todo.Description != "" {
// 			msg.WriteString(fmt.Sprintf("   - %s\n", todo.Description))
// 		}
// 	}

// 	if todoPlan.PlanSummary != "" {
// 		msg.WriteString(fmt.Sprintf("\n**总体说明**:\n%s\n", todoPlan.PlanSummary))
// 	}

// 	return msg.String()
// }

// func convertDraftToScheduleData(shift *d_model.Shift, draft *d_model.ShiftScheduleDraft, startDate, endDate string, staffList []*d_model.Employee, staffRequirements map[string]int) map[string]any {
// 	staffNames := make(map[string]string)
// 	for _, s := range staffList {
// 		staffNames[s.ID] = s.Name
// 	}

// 	days := make(map[string]any)
// 	for date, staffIDs := range draft.Schedule {
// 		names := make([]string, 0, len(staffIDs))
// 		for _, id := range staffIDs {
// 			if name, ok := staffNames[id]; ok {
// 				names = append(names, name)
// 			} else {
// 				names = append(names, id)
// 			}
// 		}

// 		requiredCount := 0
// 		if req, ok := staffRequirements[date]; ok {
// 			requiredCount = req
// 		}

// 		days[date] = map[string]any{
// 			"staff":         names,
// 			"staffIds":      staffIDs,
// 			"requiredCount": requiredCount,
// 			"actualCount":   len(staffIDs),
// 		}
// 	}

// 	return map[string]any{
// 		"shiftId":   shift.ID,
// 		"shiftName": shift.Name,
// 		"startDate": startDate,
// 		"endDate":   endDate,
// 		"schedule": map[string]any{
// 			"shiftId":  shift.ID,
// 			"priority": shift.SchedulingPriority,
// 			"days":     days,
// 		},
// 	}
// }
