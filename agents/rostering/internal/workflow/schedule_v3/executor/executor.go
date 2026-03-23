package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"

	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// NewProgressiveTaskExecutor 创建渐进式任务执行器
func NewProgressiveTaskExecutor(
	logger logging.ILogger,
	schedulingAIService ISchedulingAIService,
	ruleValidator d_service.IRuleLevelValidator,
	rosteringService d_service.IRosteringService,
	aiFactory *ai.AIProviderFactory,
	configurator config.IRosteringConfigurator,
) IProgressiveTaskExecutor {
	return &ProgressiveTaskExecutor{
		logger:              logger.With("component", "ProgressiveTaskExecutor"),
		schedulingAIService: schedulingAIService,
		ruleValidator:       ruleValidator,
		rosteringService:    rosteringService,
		aiFactory:           aiFactory,
		configurator:        configurator,
	}
}

// SetProgressCallback 设置进度回调函数
func (e *ProgressiveTaskExecutor) SetProgressCallback(callback ProgressCallback) {
	e.progressCallback = callback
}

// notifyProgress 发送进度通知（内部方法）
func (e *ProgressiveTaskExecutor) notifyProgress(info *ShiftProgressInfo) {
	if e.progressCallback != nil {
		e.progressCallback(info)
	}
}

// ExecuteTask 执行渐进式任务（新接口，推荐使用）
// 使用强类型 taskContext，内部统一使用 []StaffOccupiedSlot 数组
func (e *ProgressiveTaskExecutor) ExecuteTask(
	ctx context.Context,
	taskCtx *utils.CoreV3TaskContext,
) (*d_model.TaskResult, error) {
	// 设置任务上下文
	e.taskContext = taskCtx

	startTime := time.Now()
	e.logger.Info("Executing progressive task (V3)",
		"taskID", taskCtx.Task.ID,
		"taskTitle", taskCtx.Task.Title)

	result := &d_model.TaskResult{
		TaskID:  taskCtx.Task.ID,
		Success: false,
	}

	// 确保 currentDraft 不为 nil
	if taskCtx.CurrentDraft == nil {
		taskCtx.CurrentDraft = d_model.NewShiftScheduleDraft()
	}

	// 根据任务类型执行不同的逻辑
	var err error
	task := taskCtx.Task
	shifts := taskCtx.Shifts
	rules := taskCtx.Rules
	staffList := taskCtx.CandidateStaff
	staffRequirements := d_model.ConvertRequirementsToMap(taskCtx.StaffRequirements)

	if e.aiFactory != nil && (task.Type == "" || task.Type == "ai") {
		// AI 执行任务 - 直接使用 taskContext.OccupiedSlots（强类型数组）
		shiftSchedules, _, execErr := e.executeAITaskV3(ctx, task, shifts, rules, staffList, staffRequirements)
		if execErr != nil {
			err = execErr
		} else {
			result.ShiftSchedules = shiftSchedules
			e.logger.Info("AI task completed",
				"taskID", task.ID,
				"shiftCount", len(shiftSchedules))
		}
	} else if task.Type == "fill" || e.aiFactory == nil {
		// 使用填充逻辑（显式指定或AI不可用时回退）
		scheduleDraft, execErr := e.executeRemainingFillTaskV3(ctx, task, shifts, rules, staffList, staffRequirements, taskCtx.CurrentDraft)
		if execErr != nil {
			err = execErr
		} else if scheduleDraft != nil && len(scheduleDraft.Schedule) > 0 {
			// 转换为 ShiftSchedules 格式
			targetShiftID := ""
			if len(task.TargetShifts) > 0 {
				targetShiftID = task.TargetShifts[0]
			} else if len(shifts) > 0 {
				targetShiftID = shifts[0].ID
			}
			if result.ShiftSchedules == nil {
				result.ShiftSchedules = make(map[string]*d_model.ShiftScheduleDraft)
			}
			result.ShiftSchedules[targetShiftID] = scheduleDraft
		}
	} else {
		err = fmt.Errorf("unsupported task type: %s", task.Type)
	}

	result.ExecutionTime = time.Since(startTime).Seconds()

	if err != nil {
		result.Error = err.Error()
		e.logger.Error("Progressive task failed",
			"taskID", task.ID,
			"error", err,
			"executionTime", result.ExecutionTime)
		return result, err
	}

	result.Success = true
	e.logger.Info("Progressive task completed successfully",
		"taskID", task.ID,
		"executionTime", result.ExecutionTime)

	return result, nil
}

// ExecuteShiftTask 执行单个班次任务（V3改进，使用ShiftTaskContext）
// 在单班次级别动态计算候选人员，根据当前占位状态实时过滤
// 注意：此方法需要 workingDraft 参数，因为 ConvertShiftTaskContextToCoreV3TaskContext 需要它
func (e *ProgressiveTaskExecutor) ExecuteShiftTask(
	ctx context.Context,
	shiftCtx *utils.ShiftTaskContext,
	orgID string,
	workingDraft *d_model.ScheduleDraft,
) (*d_model.ShiftScheduleDraft, error) {
	// 将 ShiftTaskContext 转换为 CoreV3TaskContext（适配现有执行器）
	coreCtx := utils.ConvertShiftTaskContextToCoreV3TaskContext(
		shiftCtx,
		orgID,
		workingDraft,
	)

	// 执行任务（只处理单个班次）
	taskResult, err := e.ExecuteTask(ctx, coreCtx)
	if err != nil {
		return nil, err
	}

	// 提取该班次的结果
	if taskResult != nil && len(taskResult.ShiftSchedules) > 0 {
		for _, draft := range taskResult.ShiftSchedules {
			return draft, nil
		}
	}

	return d_model.NewShiftScheduleDraft(), nil
}

// ExecuteProgressiveTask 执行渐进式任务（旧接口，保持兼容）
//
// Deprecated: 此方法使用 map[string]map[string]string 类型的 occupiedSlots，
// 存在数据同步问题。请改用 ExecuteTask 方法，它直接使用 taskContext.OccupiedSlots
// 强类型数组，确保数据一致性。
//
// 迁移指南：
//  1. 构建 CoreV3TaskContext 对象
//  2. 调用 ExecuteTask(ctx, taskCtx) 代替此方法
//  3. occupiedSlots 数据应存储在 taskCtx.OccupiedSlots 中
//
// 此方法将在 v3.2.0 版本后移除。
func (e *ProgressiveTaskExecutor) ExecuteProgressiveTask(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
	occupiedSlots map[string]map[string]string,
	currentDraft *d_model.ShiftScheduleDraft,
) (*d_model.TaskResult, error) {
	startTime := time.Now()
	e.logger.Info("Executing progressive task",
		"taskID", task.ID,
		"taskTitle", task.Title)

	result := &d_model.TaskResult{
		TaskID:  task.ID,
		Success: false,
	}

	// 确保 currentDraft 不为 nil
	if currentDraft == nil {
		currentDraft = d_model.NewShiftScheduleDraft()
	}

	// 根据任务类型执行不同的逻辑
	var err error

	// 【渐进式校验重构】移除废弃的 validation 任务类型
	// validation 类型任务已废弃，规则校验现在通过以下方式实现：
	// 1. 中途校验：validateSingleShift（仅超配检测）
	// 2. 单日排班后校验：validateDayScheduleWithLLM（LLM规则匹配度，校验累积排班结果）
	//    在单日排班完成后立即调用，如果校验失败立即纠正，避免错误蔓延
	// 3. 最终校验：ValidateFinalSchedule（严格人数等于）

	if e.aiFactory != nil && (task.Type == "" || task.Type == "ai") {
		// 【占位信息格式统一】AI 执行任务 - 返回班次维度数据（已移除 map 格式的 occupiedSlots 参数）
		shiftSchedules, _, execErr := e.executeAITask(ctx, task, shifts, rules, staffList, staffRequirements)
		if execErr != nil {
			err = execErr
		} else {
			result.ShiftSchedules = shiftSchedules
			e.logger.Info("AI task completed",
				"taskID", task.ID,
				"shiftCount", len(shiftSchedules))
		}
	} else if task.Type == "fill" || e.aiFactory == nil {
		// 使用填充逻辑（显式指定或AI不可用时回退）
		scheduleDraft, execErr := e.executeRemainingFillTask(ctx, task, shifts, rules, staffList, staffRequirements, occupiedSlots, currentDraft)
		if execErr != nil {
			err = execErr
		} else if scheduleDraft != nil && len(scheduleDraft.Schedule) > 0 {
			// 转换为 ShiftSchedules 格式
			targetShiftID := ""
			if len(task.TargetShifts) > 0 {
				targetShiftID = task.TargetShifts[0]
			} else if len(shifts) > 0 {
				targetShiftID = shifts[0].ID
			}
			if targetShiftID != "" {
				result.ShiftSchedules = map[string]*d_model.ShiftScheduleDraft{
					targetShiftID: scheduleDraft,
				}
			}
		}
	} else {
		// 未知任务类型，返回错误
		err = fmt.Errorf("unsupported task type: %s", task.Type)
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime).Seconds()
		return result, err
	}

	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime).Seconds()
		return result, err
	}

	result.Success = true
	result.ExecutionTime = time.Since(startTime).Seconds()

	// 构建部分成功相关字段
	if len(e.lastFailedShifts) > 0 {
		result.FailedShifts = e.lastFailedShifts
		result.SuccessfulShifts = e.lastSuccessfulShifts
		result.PartiallySucceeded = len(e.lastSuccessfulShifts) > 0

		// 如果所有班次都失败，标记为失败
		if len(e.lastSuccessfulShifts) == 0 {
			result.Success = false
		}

		// 【关键修复】构建失败描述并设置到 Error 字段
		// 这样 actOnEnterTaskFailedState 才能提取并显示失败原因
		var failedShiftNames []string
		var failureReasons []string
		for shiftID, failInfo := range e.lastFailedShifts {
			failedShiftNames = append(failedShiftNames, failInfo.ShiftName)
			if failInfo.FailureSummary != "" {
				failureReasons = append(failureReasons, fmt.Sprintf("%s: %s", failInfo.ShiftName, failInfo.FailureSummary))
			} else if failInfo.LastError != "" {
				failureReasons = append(failureReasons, fmt.Sprintf("%s: %s", failInfo.ShiftName, failInfo.LastError))
			} else {
				failureReasons = append(failureReasons, fmt.Sprintf("%s: 执行失败", failInfo.ShiftName))
			}
			_ = shiftID // 避免未使用变量警告
		}
		if len(failureReasons) > 0 {
			result.Error = fmt.Sprintf("以下班次排班失败：\n%s", strings.Join(failureReasons, "\n"))
		}

		e.logger.Info("Task execution result: partial success",
			"taskID", task.ID,
			"successfulCount", len(e.lastSuccessfulShifts),
			"failedCount", len(e.lastFailedShifts),
			"failedShiftNames", failedShiftNames)

		// 清理临时存储
		e.lastFailedShifts = nil
		e.lastSuccessfulShifts = nil
	}

	e.logger.Info("Progressive task executed successfully",
		"taskID", task.ID,
		"taskTitle", task.Title,
		"shiftSchedulesCount", len(result.ShiftSchedules),
		"executionTime", result.ExecutionTime)

	return result, nil
}

// executeRemainingFillTask 执行剩余人员填充任务
func (e *ProgressiveTaskExecutor) executeRemainingFillTask(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
	occupiedSlots map[string]map[string]string,
	currentDraft *d_model.ShiftScheduleDraft,
) (*d_model.ShiftScheduleDraft, error) {
	e.logger.Info("Executing remaining fill task", "taskID", task.ID)

	// 创建排班草案（基于当前草案）
	draft := e.copyShiftScheduleDraft(currentDraft)

	// 如果没有班次，直接返回
	if len(shifts) == 0 {
		e.logger.Warn("No shifts provided for remaining fill task")
		return draft, nil
	}

	// 拆分任务（如果涉及多个班次）
	subTasks := e.splitTaskByShifts(task, shifts)

	successCount := 0
	failCount := 0

	// 为每个子任务执行填充
	for _, subTask := range subTasks {
		// 获取子任务的班次（应该只有一个）
		taskShifts := e.getTaskShifts(subTask, shifts)
		if len(taskShifts) == 0 {
			e.logger.Warn("Sub-task has no shifts, skipping", "subTaskID", subTask.ID)
			continue
		}
		shift := taskShifts[0] // 子任务只有一个班次

		// 构建 ShiftInfo
		shiftInfo := &d_model.ShiftInfo{
			ShiftID:   shift.ID,
			ShiftName: shift.Name,
			ShiftCode: shift.Code,
			StartTime: shift.StartTime,
			EndTime:   shift.EndTime,
			Priority:  shift.SchedulingPriority,
		}

		// 获取该班次的人员需求（按日期汇总）
		shiftStaffRequirements := make(map[string]int)
		if reqs, ok := staffRequirements[shift.ID]; ok {
			for date, count := range reqs {
				shiftStaffRequirements[date] = count
			}
		}

		// 【关键修改】计算该班次每日的固定排班人数，并从需求中减去
		// 这样传给 LLM 的是"还需要安排的动态人数"，而不是"总需求人数"
		shiftFixedCounts := make(map[string]int)
		if e.taskContext != nil && len(e.taskContext.FixedAssignments) > 0 {
			for _, fa := range e.taskContext.FixedAssignments {
				if fa.ShiftID == shift.ID {
					shiftFixedCounts[fa.Date] = len(fa.StaffIDs)
				}
			}
		}

		// 【关键修复】获取 WorkingDraft 中已有的动态排班人数
		existingDynamicCounts := make(map[string]int)
		if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
			if shiftDraft, exists := e.taskContext.WorkingDraft.Shifts[shift.ID]; exists && shiftDraft != nil && shiftDraft.Days != nil {
				for date, dayShift := range shiftDraft.Days {
					if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
						existingDynamicCounts[date] = len(dayShift.StaffIDs)
					}
				}
			}
		}

		// 从需求中减去固定排班人数和已有动态排班人数，计算动态需求
		dynamicStaffRequirements := make(map[string]int)
		for date, totalRequired := range shiftStaffRequirements {
			fixedCount := shiftFixedCounts[date]
			existingDynamicCount := existingDynamicCounts[date]
			dynamicRequired := totalRequired - fixedCount - existingDynamicCount
			if dynamicRequired < 0 {
				dynamicRequired = 0 // 如果已超配，动态需求为0
			}
			dynamicStaffRequirements[date] = dynamicRequired
		}

		// 【优化】检查是否所有日期的动态需求都为0
		hasNonZeroRequirement := false
		for _, count := range dynamicStaffRequirements {
			if count > 0 {
				hasNonZeroRequirement = true
				break
			}
		}
		if !hasNonZeroRequirement {
			e.logger.Info("Skipping shift with all zero dynamic requirements (fixed assignments already meet all needs)",
				"shiftID", shift.ID,
				"shiftName", shift.Name,
				"totalDates", len(dynamicStaffRequirements))
			successCount++
			continue
		}

		// 获取目标日期列表（从子任务中获取，如果没有则从需求中提取）
		targetDates := subTask.TargetDates
		if len(targetDates) == 0 {
			// 如果没有指定目标日期，从动态需求中提取需求>0的日期
			for date, count := range dynamicStaffRequirements {
				if count > 0 {
					targetDates = append(targetDates, date)
				}
			}
		}

		// 构建不可用人员清单（从个人需求中提取负向需求）
		var unavailableStaffMap *d_model.UnavailableStaffMap
		if e.taskContext != nil && e.taskContext.PersonalNeeds != nil {
			unavailableStaffMap = e.buildUnavailableStaffMap(
				e.taskContext.PersonalNeeds,
				targetDates,
				"", "", // startDate和endDate暂时为空，因为targetDates已经提供了日期范围
			)
		} else {
			unavailableStaffMap = d_model.NewUnavailableStaffMap()
		}

		// 构建可用人员列表（转换为 StaffInfoForAI，过滤掉不可用人员）
		availableStaff := e.buildAvailableStaffList(staffList, draft, occupiedSlots, unavailableStaffMap, targetDates)

		// 构建固定排班信息（从任务上下文中获取）
		// ExecuteTodoTask 需要 date -> staffIDs 格式，需要合并所有班次
		var fixedShiftAssignmentsMerged map[string][]string
		if e.taskContext != nil && len(e.taskContext.FixedAssignments) > 0 {
			fixedShiftAssignmentsMerged = e.mergeFixedShiftAssignments(e.taskContext.FixedAssignments)
		}
		if fixedShiftAssignmentsMerged == nil {
			fixedShiftAssignmentsMerged = make(map[string][]string)
		}

		// 构建临时需求（从子任务描述中提取，如果有）
		temporaryNeeds := e.extractTemporaryNeedsFromTask(subTask)

		// 构建 SchedulingTodo 对象
		// 注意：SchedulingTodo.Priority 是 string 类型，需要转换
		priorityStr := "normal"
		if subTask.Priority > 0 {
			priorityStr = fmt.Sprintf("%d", subTask.Priority)
		}
		todoTask := &d_model.SchedulingTodo{
			ID:          subTask.ID,
			Title:       subTask.Title,
			Description: subTask.Description,
			Priority:    priorityStr,
		}

		// 调用 AI 执行填充任务（V3增强：传递所有班次和工作草案）
		var allShifts []*d_model.Shift
		var workingDraft *d_model.ScheduleDraft
		if e.taskContext != nil {
			allShifts = shifts // 所有班次列表
			workingDraft = e.taskContext.WorkingDraft
		}

		todoResult, err := e.schedulingAIService.ExecuteTodoTask(
			ctx,
			todoTask,
			shiftInfo,
			availableStaff,
			draft,
			dynamicStaffRequirements,    // 【关键修改】使用减去固定排班后的动态需求
			fixedShiftAssignmentsMerged, // 使用合并后的固定排班数据
			temporaryNeeds,
			staffList,
			allShifts,    // V3新增
			workingDraft, // V3新增
		)
		if err != nil {
			e.logger.Error("Failed to execute sub-task",
				"subTaskID", subTask.ID,
				"shiftID", shift.ID,
				"error", err)
			failCount++
			// 继续处理其他子任务，不中断整个任务
			continue
		}
		successCount++

		// 合并执行结果
		if todoResult != nil && todoResult.Schedule != nil {
			// 转换为 ShiftScheduleDraft 格式
			subTaskDraft := d_model.NewShiftScheduleDraft()
			subTaskDraft.Schedule = todoResult.Schedule

			// 使用增强的合并逻辑
			if err := e.mergeSubTaskResult(draft, subTaskDraft, shift.ID, shiftStaffRequirements); err != nil {
				e.logger.Warn("Failed to merge sub-task result",
					"subTaskID", subTask.ID,
					"error", err)
			}
		}
	}

	e.logger.Info("Remaining fill task completed",
		"taskID", task.ID,
		"subTaskCount", len(subTasks),
		"successCount", successCount,
		"failCount", failCount,
		"totalDates", len(draft.Schedule))

	return draft, nil
}

// executeRemainingFillTaskV3 执行剩余人员填充任务（V3版本，直接使用 taskContext.OccupiedSlots）
// 这是 executeRemainingFillTask 的重构版本，不再需要 occupiedSlots map 参数
func (e *ProgressiveTaskExecutor) executeRemainingFillTaskV3(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
	currentDraft *d_model.ShiftScheduleDraft,
) (*d_model.ShiftScheduleDraft, error) {
	// 从 taskContext 获取 occupiedSlots map（用于兼容旧代码）
	occupiedSlots := d_model.ConvertOccupiedSlotsToMap(e.taskContext.OccupiedSlots)

	// 调用原有实现
	return e.executeRemainingFillTask(ctx, task, shifts, rules, staffList, staffRequirements, occupiedSlots, currentDraft)
}
