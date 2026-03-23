package executor

import (
	"context"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
)

// ValidateTaskStaffCount 校验任务相关班次的人数需求（任务级校验）
// 在任务完成后立即调用，检查该任务相关班次的人数是否满足需求
// 如果发现缺员，立即生成补充任务
func (e *ProgressiveTaskExecutor) ValidateTaskStaffCount(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	workingDraft *d_model.ScheduleDraft,
	staffRequirements []d_model.ShiftDateRequirement,
	selectedShifts []*d_model.Shift,
) *TaskValidationResult {
	e.logger.Info("Validating task staff count",
		"taskID", task.ID,
		"taskTitle", task.Title)

	result := &TaskValidationResult{
		Passed:          true,
		ShortageDetails: make([]*ShortageDetail, 0),
	}

	// 构建班次ID到名称的映射
	shiftNameMap := make(map[string]string)
	for _, shift := range selectedShifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	// 过滤任务相关的需求
	taskRequirements := filterRequirementsByTask(staffRequirements, task)

	// 检查每个班次每天的人数
	for _, req := range taskRequirements {
		// 只检查任务相关的班次
		if len(task.TargetShifts) > 0 {
			found := false
			for _, targetShiftID := range task.TargetShifts {
				if targetShiftID == req.ShiftID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 只检查任务相关的日期
		if len(task.TargetDates) > 0 {
			found := false
			for _, targetDate := range task.TargetDates {
				if targetDate == req.Date {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 获取实际排班人数
		actualCount := 0
		if workingDraft != nil && workingDraft.Shifts != nil {
			if shiftDraft, exists := workingDraft.Shifts[req.ShiftID]; exists && shiftDraft != nil {
				if dayShift, dayExists := shiftDraft.Days[req.Date]; dayExists && dayShift != nil {
					actualCount = len(dayShift.StaffIDs)
				}
			}
		}

		if actualCount < req.Count {
			// 发现缺员
			result.Passed = false
			shiftName := shiftNameMap[req.ShiftID]
			if shiftName == "" {
				shiftName = req.ShiftID
			}

			shortage := &ShortageDetail{
				ShiftID:       req.ShiftID,
				ShiftName:     shiftName,
				Date:          req.Date,
				RequiredCount: req.Count,
				ActualCount:   actualCount,
				ShortageCount: req.Count - actualCount,
			}
			result.ShortageDetails = append(result.ShortageDetails, shortage)
		}
	}

	if result.Passed {
		result.Summary = "任务级人数校验通过"
		e.logger.Info("Task staff count validation passed", "taskID", task.ID)
	} else {
		totalShortage := 0
		for _, shortage := range result.ShortageDetails {
			totalShortage += shortage.ShortageCount
		}
		result.Summary = fmt.Sprintf("任务级人数校验失败：发现 %d 处缺员，共缺 %d 人", len(result.ShortageDetails), totalShortage)
		e.logger.Warn("Task staff count validation failed",
			"taskID", task.ID,
			"shortageCount", len(result.ShortageDetails),
			"totalShortage", totalShortage)
	}

	return result
}

// filterRequirementsByTask 过滤任务相关的需求
func filterRequirementsByTask(reqs []d_model.ShiftDateRequirement, task *d_model.ProgressiveTask) []d_model.ShiftDateRequirement {
	if len(task.TargetShifts) == 0 && len(task.TargetDates) == 0 {
		return reqs
	}

	// 构建过滤条件
	targetShifts := make(map[string]bool)
	for _, shiftID := range task.TargetShifts {
		targetShifts[shiftID] = true
	}

	targetDates := make(map[string]bool)
	for _, date := range task.TargetDates {
		targetDates[date] = true
	}

	// 过滤
	result := make([]d_model.ShiftDateRequirement, 0)
	for _, req := range reqs {
		matchShift := len(targetShifts) == 0 || targetShifts[req.ShiftID]
		matchDate := len(targetDates) == 0 || targetDates[req.Date]
		if matchShift && matchDate {
			result = append(result, req)
		}
	}

	return result
}
