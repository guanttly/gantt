package core

import (
	"context"
	"fmt"
	"strings"

	"jusha/agent/rostering/config"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/wsbridge"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	i_service "jusha/agent/rostering/internal/service"

	"jusha/agent/rostering/internal/workflow/schedule_v3/executor"
	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 核心子工作流 V3 Actions
// ============================================================

// getTaskShifts 获取任务相关的班次（辅助函数）
func getTaskShifts(task *d_model.ProgressiveTask, shifts []*d_model.Shift) []*d_model.Shift {
	if len(task.TargetShifts) == 0 {
		return shifts
	}

	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		shiftMap[shift.ID] = shift
	}

	result := make([]*d_model.Shift, 0)
	for _, shiftID := range task.TargetShifts {
		if shift, ok := shiftMap[shiftID]; ok {
			result = append(result, shift)
		}
	}

	return result
}

// 【P2优化】辅助函数：确保TaskResult.Metadata已初始化
func ensureTaskResultMetadata(taskCtx *utils.CoreV3TaskContext) {
	if taskCtx == nil || taskCtx.TaskResult == nil {
		return
	}
	if taskCtx.TaskResult.Metadata == nil {
		taskCtx.TaskResult.Metadata = make(map[string]any)
	}
}

// onCoreV3Enter 子工作流进入钩子
func onCoreV3Enter(ctx engine.Context, parentWorkflow string) error {
	logger := ctx.Logger()
	sess := ctx.Session()

	logger.Info("CoreV3: Sub-workflow entered",
		"sessionID", sess.ID,
		"parentWorkflow", parentWorkflow)

	// 可以在这里进行初始化操作
	return nil
}

// onCoreV3Exit 子工作流退出钩子
func onCoreV3Exit(ctx engine.Context, success bool) error {
	logger := ctx.Logger()
	sess := ctx.Session()

	logger.Info("CoreV3: Sub-workflow exited",
		"sessionID", sess.ID,
		"success", success)

	// 可以在这里进行清理操作
	return nil
}

// ============================================================
// 阶段 1: 预验证
// ============================================================

// actPreValidate 验证任务数据
func actPreValidate(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Pre-validating task data", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	// 验证任务数据
	if taskCtx.Task == nil {
		return fmt.Errorf("task is nil")
	}
	if len(taskCtx.Shifts) == 0 {
		return fmt.Errorf("no shifts provided")
	}
	if len(taskCtx.CandidateStaff) == 0 {
		return fmt.Errorf("no staff list provided")
	}

	logger.Info("CoreV3: Task data validated",
		"taskID", taskCtx.Task.ID,
		"taskTitle", taskCtx.Task.Title)

	// 验证通过，触发执行事件
	return wctx.Send(ctx, CoreV3EventTaskExecuted, nil)
}

// ============================================================
// 阶段 2: 执行任务
// ============================================================

// actExecuteTask 执行渐进式任务（使用新的L2/L3结构）
func actExecuteTask(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Executing progressive task with L2/L3 structure", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	// 【V3改进】从 CoreV3TaskContext 重建L2/L3结构
	// 1. 构建L2任务执行上下文
	// 【优化】统一使用 taskExecutionContext.OccupiedSlots 作为唯一数据源
	taskExecutionContext := &utils.TaskExecutionContext{
		AllStaff: taskCtx.AllStaff, ShiftMembersMap: taskCtx.ShiftMembersMap, // 各班次专属人员（L3候选人过滤依据）		Rules:             taskCtx.Rules,
		PersonalNeeds:     taskCtx.PersonalNeeds,
		Task:              taskCtx.Task,
		TargetShifts:      taskCtx.Shifts,        // 从CoreV3TaskContext获取班次
		OccupiedSlots:     taskCtx.OccupiedSlots, // 统一数据源
		StaffRequirements: taskCtx.StaffRequirements,
		FixedAssignments:  taskCtx.FixedAssignments,
		WorkingDraft:      taskCtx.WorkingDraft,
	}

	// 2. 拆分成L3单班次上下文（动态计算候选人员）
	// 注意：SplitIntoShiftContexts 会深拷贝 OccupiedSlots，每个班次有独立快照
	shiftContexts := taskExecutionContext.SplitIntoShiftContexts()

	logger.Info("CoreV3: Split task into shift contexts",
		"taskID", taskCtx.Task.ID,
		"shiftCount", len(shiftContexts))

	if len(shiftContexts) == 0 {
		logger.Warn("No shift contexts to execute")
		// 创建空结果
		taskCtx.TaskResult = &d_model.TaskResult{
			TaskID:  taskCtx.Task.ID,
			Success: true,
		}
		if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
			return fmt.Errorf("failed to save task context: %w", err)
		}
		return wctx.Send(ctx, CoreV3EventValidationComplete, nil)
	}

	// 获取服务
	rosteringService := engine.MustGetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	ruleValidator := i_service.NewRuleLevelValidator(logger)

	// 获取配置器
	var configurator config.IRosteringConfigurator
	var maxDailyHours, minRestHours float64

	if cfg, ok := engine.GetService[config.IRosteringConfigurator](wctx, "configurator"); ok {
		configurator = cfg
		// 从配置读取约束值
		rosteringCfg := configurator.GetConfig()
		maxDailyHours = rosteringCfg.SchedulingConstraints.MaxDailyHours
		minRestHours = rosteringCfg.SchedulingConstraints.MinRestHours
		// 校验配置值的合理性
		if maxDailyHours <= 0 {
			maxDailyHours = config.DefaultMaxDailyHours
			logger.Warn("Invalid maxDailyHours in config, using default", "default", maxDailyHours)
		}
		if minRestHours < 0 {
			minRestHours = config.DefaultMinRestHours
			logger.Warn("Invalid minRestHours in config, using default", "default", minRestHours)
		}
	} else {
		// 配置器不可用，使用默认值
		maxDailyHours = config.DefaultMaxDailyHours
		minRestHours = config.DefaultMinRestHours
		logger.Warn("Configurator not available, using default scheduling constraints",
			"maxDailyHours", maxDailyHours,
			"minRestHours", minRestHours)
	}

	// 从服务注册表获取 AI 工厂
	var aiFactory *ai.AIProviderFactory

	// 尝试获取 AI 工厂
	if factory, ok := engine.GetService[*ai.AIProviderFactory](wctx, engine.ServiceKeyAIFactory); ok {
		aiFactory = factory
	} else {
		logger.Warn("AI factory not available, AI task execution will be limited")
	}

	// 创建任务执行器
	taskExecutorImpl := executor.NewProgressiveTaskExecutor(
		logger,
		schedulingAIService,
		ruleValidator,
		rosteringService,
		aiFactory,
		configurator,
	).(*executor.ProgressiveTaskExecutor)

	// 获取 Bridge 用于实时进度广播
	var bridge wsbridge.IBridge
	if b, ok := engine.GetService[wsbridge.IBridge](wctx, engine.ServiceKeyBridge); ok {
		bridge = b
	}

	// 设置进度回调，将进度信息发送给前端
	taskExecutorImpl.SetProgressCallback(func(info *executor.ShiftProgressInfo) {
		// 1. 广播结构化的 shift_progress 消息（用于进度条组件）
		if bridge != nil {
			progressData := map[string]any{
				"shiftId":        info.ShiftID,
				"shiftName":      info.ShiftName,
				"current":        info.Current,
				"total":          info.Total,
				"status":         info.Status,
				"message":        info.Message,
				"reasoning":      info.Reasoning,
				"previewData":    info.PreviewData,
				"currentDay":     info.CurrentDay,
				"totalDays":      info.TotalDays,
				"currentDate":    info.CurrentDate,
				"completedDates": info.CompletedDates,
				"draftPreview":   info.DraftPreview,
			}
			if err := bridge.BroadcastToSession(sess.ID, "shift_progress", progressData); err != nil {
				logger.Warn("Failed to broadcast shift progress", "error", err, "shiftName", info.ShiftName)
			}
		}

		// 2. 同时发送消息（用于聊天区域显示）
		// 验证失败消息使用助手样式，其他使用系统消息样式
		if info.Status == "day_validated" && strings.Contains(info.Message, "发现错误") {
			// 验证失败消息使用助手样式
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, info.Message); err != nil {
				logger.Warn("Failed to send validation error message", "error", err, "shiftName", info.ShiftName)
			}
		} else {
			if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, info.Message); err != nil {
				logger.Warn("Failed to send progress message", "error", err, "shiftName", info.ShiftName)
			}
		}

		// AI解释使用助手消息样式
		if info.Status == "success" && info.Reasoning != "" {
			reasoningMsg := fmt.Sprintf("💡 [%s] AI解释: %s", info.ShiftName, info.Reasoning)
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, reasoningMsg); err != nil {
				logger.Warn("Failed to send reasoning message", "error", err)
			}
		}
	})

	// 【增强】发送任务开始和进度提示
	if len(shiftContexts) > 1 {
		// 发送初始进度消息（使用系统消息样式）
		subTaskMsg := fmt.Sprintf("⏳ 任务开始执行，共需处理 %d 个班次，请稍候...", len(shiftContexts))
		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, subTaskMsg); err != nil {
			logger.Warn("Failed to send task start message", "error", err)
		}
	}

	// 【V3改进】按顺序执行每个班次（确保占位信息实时更新）
	for i, shiftCtx := range shiftContexts {
		logger.Info("Executing shift",
			"shiftID", shiftCtx.TargetShift.ID,
			"shiftName", shiftCtx.TargetShift.Name,
			"candidateCount", len(shiftCtx.CandidateStaff), // ★ 动态计算的候选人员
			"shiftIndex", i+1,
			"totalShifts", len(shiftContexts))

		// 记录执行前的占位信息数量（用于计算新增占位）
		occupiedCountBefore := len(taskExecutionContext.OccupiedSlots)

		// 将 ShiftTaskContext 转换为 CoreV3TaskContext（适配现有执行器）
		// 【优化】转换时直接使用 shiftCtx.OccupiedSlots，避免重复拷贝
		// 注意：Go中切片赋值是浅拷贝，但为了安全，我们让执行器更新 taskExecutionContext.OccupiedSlots
		coreCtx := utils.ConvertShiftTaskContextToCoreV3TaskContext(
			shiftCtx,
			taskCtx.OrgID,
			taskCtx.WorkingDraft,
		)

		// 【关键优化】让 coreCtx.OccupiedSlots 直接引用 taskExecutionContext.OccupiedSlots
		// 这样执行器更新时，会直接更新统一的数据源
		coreCtx.OccupiedSlots = taskExecutionContext.OccupiedSlots

		// 执行单个班次
		var taskExecutor executor.IProgressiveTaskExecutor = taskExecutorImpl
		shiftResult, err := taskExecutor.ExecuteTask(ctx, coreCtx)
		if err != nil {
			logger.Error("CoreV3: Shift execution failed",
				"shiftID", shiftCtx.TargetShift.ID,
				"error", err)
			// 继续执行其他班次，不中断整个任务
			continue
		}

		// 保存班次结果
		if shiftResult != nil && len(shiftResult.ShiftSchedules) > 0 {
			// 提取该班次的结果
			for _, draft := range shiftResult.ShiftSchedules {
				shiftCtx.Result = draft
				break // 通常只有一个班次的结果
			}
		}

		// 【统一数据源】执行后，需要将 coreCtx.OccupiedSlots 同步回 taskExecutionContext.OccupiedSlots
		// 因为 AddOccupiedSlotIfNotExists 可能返回新切片（如果底层数组扩容）
		taskExecutionContext.OccupiedSlots = coreCtx.OccupiedSlots

		// 计算新增的占位信息
		occupiedCountAfter := len(taskExecutionContext.OccupiedSlots)
		var newSlots []d_model.StaffOccupiedSlot
		if occupiedCountAfter > occupiedCountBefore {
			// 提取新增的占位信息（相对于任务开始时的状态）
			newSlots = taskExecutionContext.OccupiedSlots[occupiedCountBefore:]
		}

		// ★ 关键：更新后续班次的占位信息（影响候选人员计算）
		if i < len(shiftContexts)-1 && len(newSlots) > 0 {
			// 更新后续班次的占位信息（使用统一数据源的新增部分）
			for _, nextShiftCtx := range shiftContexts[i+1:] {
				// 合并新增的占位信息
				nextShiftCtx.OccupiedSlots = utils.MergeOccupiedSlots(nextShiftCtx.OccupiedSlots, newSlots)
				// 重新计算候选人员（因为占位信息已更新）
				nextShiftCtx.CandidateStaff = nextShiftCtx.ComputeCandidateStaff()
				// 重新构建LLM缓存（因为候选人员已更新）
				nextShiftCtx.BuildLLMCache()
			}
		}
	}

	// 合并所有班次结果到任务结果
	if err := taskExecutionContext.MergeShiftResults(shiftContexts); err != nil {
		logger.Error("Failed to merge shift results", "error", err)
		return fmt.Errorf("failed to merge shift results: %w", err)
	}

	// 保存执行结果到 CoreV3TaskContext
	taskCtx.TaskResult = taskExecutionContext.TaskResult
	taskCtx.OccupiedSlots = taskExecutionContext.OccupiedSlots // 更新占位信息

	// 【增强】发送任务完成提示
	if len(shiftContexts) > 1 {
		completeMsg := fmt.Sprintf("✅ %d 个班次处理完成", len(shiftContexts))
		if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, completeMsg); err != nil {
			logger.Warn("Failed to send task complete message", "error", err)
		}
	}

	logger.Info("CoreV3: Saving TaskResult to context",
		"taskID", taskCtx.Task.ID,
		"success", taskCtx.TaskResult.Success,
		"shiftSchedulesCount", len(taskCtx.TaskResult.ShiftSchedules))

	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	logger.Info("CoreV3: Task executed and saved successfully",
		"taskID", taskCtx.Task.ID,
		"success", taskCtx.TaskResult.Success)

	// 触发校验事件
	return wctx.Send(ctx, CoreV3EventValidationComplete, nil)
}

// ============================================================
// 阶段 3: 规则级校验
// ============================================================

// actValidateResult 执行规则级校验
func actValidateResult(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Validating task result", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	// 规则级校验已经在 ExecuteProgressiveTask 中完成
	// 这里只需要检查结果
	if taskCtx.TaskResult == nil {
		return fmt.Errorf("task result is nil")
	}

	if taskCtx.TaskResult.RuleValidationResult != nil {
		if !taskCtx.TaskResult.RuleValidationResult.Passed {
			logger.Warn("CoreV3: Rule validation failed",
				"summary", taskCtx.TaskResult.RuleValidationResult.Summary)
			// 即使校验失败，也继续执行（LLMQC已注释，直接完成）
		}
	}

	// 检查是否为部分成功场景
	if taskCtx.TaskResult.PartiallySucceeded {
		logger.Info("CoreV3: Task partially succeeded, transitioning to partial success state",
			"successfulCount", len(taskCtx.TaskResult.SuccessfulShifts),
			"failedCount", len(taskCtx.TaskResult.FailedShifts))

		// 发送部分成功事件，由状态机转换到 CoreV3StatePartialSuccess 状态
		// 这样用户点击的按钮（RetryFailed/SkipFailed/CancelTask）才能被正确处理
		return wctx.Send(ctx, CoreV3EventPartialSuccess, nil)
	}

	// 检查是否为全部失败场景（所有班次都失败）
	if !taskCtx.TaskResult.Success && len(taskCtx.TaskResult.FailedShifts) > 0 && len(taskCtx.TaskResult.SuccessfulShifts) == 0 {
		logger.Info("CoreV3: All shifts failed, transitioning to partial success state for user decision",
			"failedCount", len(taskCtx.TaskResult.FailedShifts))

		// 即使全部失败，也通过部分成功状态让用户决策（重试/跳过/取消）
		return wctx.Send(ctx, CoreV3EventPartialSuccess, nil)
	}

	// LLMQC已注释，直接触发完成事件
	return wctx.Send(ctx, CoreV3EventCompleted, nil)
}

// ============================================================
// 阶段 4: LLMQC校验（已注释，暂时不使用）
// ============================================================

// actLLMQC 执行LLMQC校验（已注释，暂时不使用）
/*
func actLLMQC(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Performing LLMQC validation", "sessionID", sess.ID)

	// 获取任务上下文
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to get task context: %w", err)
	}

	if taskCtx.TaskResult == nil {
		return fmt.Errorf("task result is nil")
	}

	// 如果已有LLMQC结果，直接使用（可能是在任务执行阶段完成的）
	if taskCtx.TaskResult.LLMQCResult != nil {
		logger.Info("CoreV3: LLMQC validation already completed",
			"passed", taskCtx.TaskResult.LLMQCResult.Passed,
			"summary", taskCtx.TaskResult.LLMQCResult.Summary)

		// 保存最终结果
		if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
			return fmt.Errorf("failed to save task context: %w", err)
		}

		// 触发完成事件
		return wctx.Send(ctx, CoreV3EventCompleted, nil)
	}

	// 如果还没有LLMQC结果，执行LLMQC校验
	if len(taskCtx.TaskResult.ShiftSchedules) == 0 {
		logger.Info("CoreV3: No shift schedules for LLMQC validation, skipping")
		if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
			return fmt.Errorf("failed to save task context: %w", err)
		}
		return wctx.Send(ctx, CoreV3EventCompleted, nil)
	}

	// 获取AI服务
	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		logger.Warn("CoreV3: AI service not available for LLMQC validation")
		// AI服务不可用，跳过LLMQC校验
		if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
			return fmt.Errorf("failed to save task context: %w", err)
		}
		return wctx.Send(ctx, CoreV3EventCompleted, nil)
	}

	// 获取任务相关的班次和规则
	taskShifts := taskCtx.Shifts
	if len(taskCtx.Task.TargetShifts) > 0 {
		// 如果任务指定了班次，只验证这些班次
		taskShifts = make([]*d_model.Shift, 0)
		for _, shiftID := range taskCtx.Task.TargetShifts {
			for _, shift := range taskCtx.Shifts {
				if shift.ID == shiftID {
					taskShifts = append(taskShifts, shift)
					break
				}
			}
		}
	}

	taskRules := taskCtx.Rules
	if len(taskCtx.Task.RuleIDs) > 0 {
		// 如果任务指定了规则，只验证这些规则
		taskRules = make([]*d_model.Rule, 0)
		for _, ruleID := range taskCtx.Task.RuleIDs {
			for _, rule := range taskCtx.Rules {
				if rule.ID == ruleID {
					taskRules = append(taskRules, rule)
					break
				}
			}
		}
	}

	// 构建每日人数需求（只包含任务相关的日期）
	taskStaffRequirements := make(map[string]int)
	if len(taskCtx.Task.TargetDates) > 0 && len(taskShifts) > 0 {
		// 使用第一个班次来查找人员需求
		shiftID := taskShifts[0].ID
		for _, date := range taskCtx.Task.TargetDates {
			if dateRequirements, ok := taskCtx.StaffRequirements[shiftID]; ok {
				if count, ok := dateRequirements[date]; ok {
					taskStaffRequirements[date] = count
				}
			}
		}
	}

	// 转换员工列表为AI需要的格式
	staffListForAI := d_model.NewStaffInfoListFromEmployees(taskCtx.StaffList)

	// 转换规则列表为AI需要的格式
	rulesForAI := d_model.NewRuleInfoListFromRules(taskRules)

	// 对每个班次执行LLMQC校验（ValidateAndAdjustShiftSchedule是针对单个班次的）
	// 【P1优化】只在规则级校验失败或配置要求时才触发LLM QC，避免重复校验
	alwaysPerformLLMQC := false // TODO: 从配置文件读取
	shouldPerformLLMQC := alwaysPerformLLMQC ||
		(taskCtx.TaskResult.RuleValidationResult != nil && !taskCtx.TaskResult.RuleValidationResult.Passed) ||
		(taskCtx.TaskResult.RuleValidationResult == nil)

	if !shouldPerformLLMQC {
		logger.Info("CoreV3: Skipping LLMQC validation (rule validation passed)")
		taskCtx.TaskResult.LLMQCResult = &d_model.ValidationResult{
			Passed:  true,
			Summary: "规则级校验通过，跳过LLM QC校验",
		}
		if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
			return fmt.Errorf("failed to save task context: %w", err)
		}
		return wctx.Send(ctx, CoreV3EventCompleted, nil)
	}

	// 简化处理：遍历 ShiftSchedules 中的每个班次执行 LLMQC
	for shiftID, shiftScheduleDraft := range taskCtx.TaskResult.ShiftSchedules {
		if shiftScheduleDraft == nil || len(shiftScheduleDraft.Schedule) == 0 {
			continue
		}

		// 查找班次信息
		var shift *d_model.Shift
		for _, s := range taskShifts {
			if s.ID == shiftID {
				shift = s
				break
			}
		}
		if shift == nil {
			// 在所有班次中查找
			for _, s := range taskCtx.Shifts {
				if s.ID == shiftID {
					shift = s
					break
				}
			}
		}
		if shift == nil {
			logger.Warn("CoreV3: Shift not found for LLMQC",
				"shiftID", shiftID)
			continue
		}

		startDate := taskCtx.Task.TargetDates[0]
		endDate := taskCtx.Task.TargetDates[len(taskCtx.Task.TargetDates)-1]

		shiftInfo := &d_model.ShiftInfo{
			ShiftID:   shift.ID,
			ShiftName: shift.Name,
			StartDate: startDate,
			EndDate:   endDate,
		}

		// 调用AI服务进行LLMQC校验
		llmqcResult, err := aiService.ValidateAndAdjustShiftSchedule(
			ctx,
			shiftScheduleDraft,
			shiftInfo,
			rulesForAI,
			taskStaffRequirements,
			staffListForAI,
			taskCtx.Task,
		)
		if err != nil {
			logger.Warn("CoreV3: LLMQC validation failed",
				"error", err,
				"taskID", taskCtx.Task.ID,
				"shiftID", shiftID)
			taskCtx.TaskResult.LLMQCResult = &d_model.ValidationResult{
				Passed:  false,
				Summary: fmt.Sprintf("LLMQC校验失败：%v", err),
			}
			continue
		}

		taskCtx.TaskResult.LLMQCResult = llmqcResult
		logger.Info("CoreV3: LLMQC validation completed",
			"shiftID", shiftID,
			"passed", llmqcResult.Passed,
			"summary", llmqcResult.Summary)

		// 如果LLMQC建议了调整，自动应用
		if !llmqcResult.Passed && len(llmqcResult.AdjustedSchedule) > 0 {
			totalDates := len(shiftScheduleDraft.Schedule)
			adjustedDates := len(llmqcResult.AdjustedSchedule)
			adjustmentRatio := float64(adjustedDates) / float64(totalDates)
			autoAdjustThreshold := 0.3

			if adjustmentRatio <= autoAdjustThreshold && totalDates > 0 {
				for date, adjustedStaffIDs := range llmqcResult.AdjustedSchedule {
					shiftScheduleDraft.Schedule[date] = adjustedStaffIDs
				}
				logger.Info("CoreV3: LLMQC adjustments auto-applied",
					"shiftID", shiftID,
					"adjustedDates", adjustedDates)
			}
		}
	}

	// 保存最终结果
	if err := SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 触发完成事件
	return wctx.Send(ctx, CoreV3EventCompleted, nil)
}
*/

// ============================================================
// 完成和失败处理
// ============================================================

// actOnComplete 任务完成处理
func actOnComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Task completed", "sessionID", sess.ID)

	// 子工作流完成，结果已保存在 taskCtx.TaskResult 中
	// 父工作流会通过 OnComplete 回调获取结果

	// 返回父工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		logger.Error("CoreV3: context is not an Actor, cannot return to parent")
		return fmt.Errorf("context is not an Actor")
	}

	result := &engine.SubWorkflowResult{
		Success: true,
		Output:  nil, // 父工作流从 session.Data[KeyCoreV3TaskContext] 读取结果
	}

	logger.Info("CoreV3: Returning to parent workflow", "sessionID", sess.ID)
	return actor.ReturnToParent(ctx, result)
}

// actOnTaskFailed 任务执行失败处理
func actOnTaskFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("CoreV3: Task execution failed", "sessionID", sess.ID)

	// 获取任务上下文，提取错误信息
	taskCtx, err := GetCoreV3TaskContext(ctx, wctx)
	if err == nil && taskCtx != nil && taskCtx.TaskResult != nil {
		errorMsg := "❌ **任务执行失败**\n\n"
		if taskCtx.Task != nil {
			errorMsg += fmt.Sprintf("任务：**%s**\n\n", taskCtx.Task.Title)
		}
		if taskCtx.TaskResult.Error != "" {
			errorMsg += fmt.Sprintf("错误信息：%s\n", taskCtx.TaskResult.Error)
		} else {
			errorMsg += "任务执行过程中出现错误，请查看日志获取详细信息。\n"
		}

		// 尝试发送错误消息（如果失败，只记录警告，不中断流程）
		if _, msgErr := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); msgErr != nil {
			logger.Warn("Failed to send task failed message (MCP tool error)",
				"error", msgErr,
				"originalError", taskCtx.TaskResult.Error)
		}
	}

	return nil
}

// actOnValidationFailed 校验失败处理
func actOnValidationFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Warn("CoreV3: Validation failed", "sessionID", sess.ID)

	// 校验失败，但可以继续（LLMQC可能会调整）
	// 或者标记为失败
	return wctx.Send(ctx, CoreV3EventFailed, nil)
}

// actOnLLMQCFailed LLMQC失败处理（已注释，暂时不使用）
/*
func actOnLLMQCFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Warn("CoreV3: LLMQC validation failed", "sessionID", sess.ID)

	// LLMQC失败，标记为失败
	return wctx.Send(ctx, CoreV3EventFailed, nil)
}
*/

// actRetryTask 重试任务
func actRetryTask(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Retrying task", "sessionID", sess.ID)

	// 重新开始执行流程
	return wctx.Send(ctx, CoreV3EventStart, nil)
}

// actSkipTask 跳过任务
func actSkipTask(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CoreV3: Skipping task", "sessionID", sess.ID)

	// 标记为完成（跳过）
	return wctx.Send(ctx, CoreV3EventCompleted, nil)
}
