package executor

import (
	"context"
	"fmt"
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 分批处理集成器
// 将分批校验、分批排班、分批修复整合为统一的执行流程
// 这是分批处理方案的主入口
// ============================================================

// BatchProcessingResult 分批处理总结果
type BatchProcessingResult struct {
	// 排班结果
	SchedulingContext *BatchSchedulingContext `json:"schedulingContext"`

	// 校验结果
	ValidationContext *BatchValidationContext `json:"validationContext"`

	// 修复结果（如有）
	FixContext *BatchFixContext `json:"fixContext"`

	// 总体状态
	Success      bool    `json:"success"`
	TotalTime    float64 `json:"totalTime"`
	ErrorMessage string  `json:"errorMessage"`

	// 统计信息
	TotalScheduled     int `json:"totalScheduled"`     // 总共排班人次
	TotalViolations    int `json:"totalViolations"`    // 发现的违规数
	FixedViolations    int `json:"fixedViolations"`    // 修复的违规数
	ManualReviewNeeded int `json:"manualReviewNeeded"` // 需人工处理数
}

// ExecuteBatchProcessing 执行完整的分批处理流程
// 1. 分批排班（每批5人）
// 2. 分批校验（每批5人）
// 3. 分批修复（LLM4，如有违规）
func (e *ProgressiveTaskExecutor) ExecuteBatchProcessing(
	ctx context.Context,
	shift *d_model.Shift,
	targetDate string,
	requiredCount int,
	candidates []*d_model.Employee,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
	config *BatchConfig,
) (*BatchProcessingResult, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}

	startTime := time.Now()
	result := &BatchProcessingResult{
		Success: true,
	}

	e.logger.Info("Starting batch processing",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"targetDate", targetDate,
		"requiredCount", requiredCount,
		"candidatesCount", len(candidates),
		"batchSize", config.BatchSize)

	// ============================================================
	// 阶段1：分批排班
	// ============================================================
	e.logger.Info("Phase 1: Starting batch scheduling",
		"shiftID", shift.ID,
		"targetDate", targetDate)

	schedulingCtx, err := e.scheduleInBatches(
		ctx, shift, targetDate, requiredCount, candidates,
		occupiedSlots, rules, personalNeeds, fixedAssignments, config,
	)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("batch scheduling failed: %v", err)
		result.TotalTime = time.Since(startTime).Seconds()
		return result, err
	}
	result.SchedulingContext = schedulingCtx
	result.TotalScheduled = len(schedulingCtx.ScheduledStaff)

	e.logger.Info("Phase 1: Batch scheduling completed",
		"shiftID", shift.ID,
		"targetDate", targetDate,
		"scheduled", len(schedulingCtx.ScheduledStaff),
		"required", requiredCount,
		"batches", len(schedulingCtx.BatchResults))

	// 检查排班是否完成
	if !schedulingCtx.Completed {
		e.logger.Warn("Batch scheduling incomplete",
			"shiftID", shift.ID,
			"targetDate", targetDate,
			"required", requiredCount,
			"scheduled", len(schedulingCtx.ScheduledStaff))
	}

	// ============================================================
	// 阶段2：分批校验
	// ============================================================
	e.logger.Info("Phase 2: Starting batch validation",
		"shiftID", shift.ID,
		"targetDate", targetDate)

	// 构建排班草案用于校验
	draft := &d_model.ShiftScheduleDraft{
		Schedule:     make(map[string][]string),
		UpdatedStaff: make(map[string]bool),
	}
	draft.Schedule[targetDate] = schedulingCtx.ScheduledStaff
	for _, id := range schedulingCtx.ScheduledStaff {
		draft.UpdatedStaff[id] = true
	}

	// 收集待校验的人员
	staffToValidate := make([]*d_model.Employee, 0)
	scheduledSet := make(map[string]bool)
	for _, id := range schedulingCtx.ScheduledStaff {
		scheduledSet[id] = true
	}
	for _, candidate := range candidates {
		if scheduledSet[candidate.ID] {
			staffToValidate = append(staffToValidate, candidate)
		}
	}

	validationCtx, err := e.validateInBatches(
		ctx, shift, []string{targetDate}, staffToValidate, draft,
		rules, occupiedSlots, allShifts, config,
	)
	if err != nil {
		e.logger.Error("Phase 2: Batch validation failed",
			"shiftID", shift.ID,
			"error", err)
		// 校验失败不中止流程，记录错误继续
	}
	result.ValidationContext = validationCtx
	result.TotalViolations = len(validationCtx.AllViolations)

	e.logger.Info("Phase 2: Batch validation completed",
		"shiftID", shift.ID,
		"targetDate", targetDate,
		"allPassed", validationCtx.AllPassed,
		"violations", len(validationCtx.AllViolations))

	// ============================================================
	// 阶段3：分批修复（如有违规）
	// ============================================================
	if len(validationCtx.AllViolations) > 0 {
		e.logger.Info("Phase 3: Starting batch fix (LLM4)",
			"shiftID", shift.ID,
			"violations", len(validationCtx.AllViolations))

		// 获取WorkingDraft
		var workingDraft *d_model.ScheduleDraft
		if e.taskContext != nil {
			workingDraft = e.taskContext.WorkingDraft
		}

		fixCtx, err := e.executeFixInBatches(
			ctx, shift, workingDraft, validationCtx.AllViolations,
			candidates, rules, occupiedSlots, allShifts, config,
		)
		if err != nil {
			e.logger.Error("Phase 3: Batch fix failed",
				"shiftID", shift.ID,
				"error", err)
		}
		result.FixContext = fixCtx

		if fixCtx != nil {
			result.FixedViolations = fixCtx.InitialViolations - len(fixCtx.CurrentViolations)
			result.ManualReviewNeeded = len(fixCtx.CurrentViolations)

			if !fixCtx.AllFixed {
				result.Success = false
				e.logger.Warn("Phase 3: Some violations remain after fix",
					"shiftID", shift.ID,
					"remaining", len(fixCtx.CurrentViolations))
			}
		}

		e.logger.Info("Phase 3: Batch fix completed",
			"shiftID", shift.ID,
			"allFixed", fixCtx.AllFixed,
			"fixedCount", result.FixedViolations,
			"remainingCount", result.ManualReviewNeeded)
	} else {
		e.logger.Debug("Phase 3: Skipped batch fix (no violations)",
			"shiftID", shift.ID)
	}

	result.TotalTime = time.Since(startTime).Seconds()

	e.logger.Info("Batch processing completed",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"targetDate", targetDate,
		"success", result.Success,
		"totalScheduled", result.TotalScheduled,
		"totalViolations", result.TotalViolations,
		"fixedViolations", result.FixedViolations,
		"manualReviewNeeded", result.ManualReviewNeeded,
		"totalTime", result.TotalTime)

	return result, nil
}

// ExecuteBatchSchedulingForDay 针对单日执行分批排班
// 这是对现有 executeProgressiveShiftScheduling 的分批增强版本
// 当单日需要排班人数 > BatchSize 时自动启用分批模式
func (e *ProgressiveTaskExecutor) ExecuteBatchSchedulingForDay(
	ctx context.Context,
	shift *d_model.Shift,
	targetDate string,
	requiredCount int,
	candidates []*d_model.Employee,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	config *BatchConfig,
) ([]string, string, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}

	// 判断是否需要启用分批模式
	if requiredCount <= config.BatchSize {
		e.logger.Debug("Using single-batch mode (count <= batchSize)",
			"shiftID", shift.ID,
			"targetDate", targetDate,
			"requiredCount", requiredCount,
			"batchSize", config.BatchSize)
		// 人数不多，使用单批模式（直接返回LLM排班结果）
		// 这里调用原有的单日排班逻辑，不做分批
		return nil, "", fmt.Errorf("fallback to original scheduling: count <= batchSize")
	}

	// 执行分批排班
	batchCtx, err := e.scheduleInBatches(
		ctx, shift, targetDate, requiredCount, candidates,
		occupiedSlots, rules, personalNeeds, fixedAssignments, config,
	)
	if err != nil {
		return nil, "", fmt.Errorf("batch scheduling failed: %w", err)
	}

	// 收集推理说明
	var reasoning string
	if len(batchCtx.BatchResults) > 0 {
		reasonings := make([]string, 0)
		for _, result := range batchCtx.BatchResults {
			if result.Reasoning != "" {
				reasonings = append(reasonings, result.Reasoning)
			}
		}
		if len(reasonings) > 0 {
			reasoning = fmt.Sprintf("分批排班完成（共%d批）：%s",
				len(batchCtx.BatchResults),
				reasonings[0]) // 只取第一个批次的理由作为代表
		}
	}

	return batchCtx.ScheduledStaff, reasoning, nil
}

// ShouldUseBatchMode 判断是否应该使用分批模式
// 当人数超过阈值时返回true
func ShouldUseBatchMode(requiredCount int, config *BatchConfig) bool {
	if config == nil {
		config = DefaultBatchConfig()
	}
	return requiredCount > config.BatchSize
}

// GetBatchModeRecommendation 获取分批模式建议
// 返回建议的批次数和每批人数
func GetBatchModeRecommendation(requiredCount int, config *BatchConfig) (int, int) {
	if config == nil {
		config = DefaultBatchConfig()
	}
	if requiredCount <= config.BatchSize {
		return 1, requiredCount
	}
	batches := (requiredCount + config.BatchSize - 1) / config.BatchSize
	return batches, config.BatchSize
}
