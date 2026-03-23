package executor

import (
	"context"
	"fmt"
	"strings"

	d_model "jusha/agent/rostering/domain/model"
)

// ValidateFinalSchedule 执行最终严格人数校验
// 在所有任务完成后调用，检查所有班次的人数是否严格等于需求
// 如果存在缺员，自动生成 LLM 补充任务
// maxFillRounds: 最大补充轮次（默认2次）
// 注意：只校验 shifts 参数中包含的班次，忽略 staffRequirements 中多余的班次需求
func (e *ProgressiveTaskExecutor) ValidateFinalSchedule(
	ctx context.Context,
	workingDraft *d_model.ScheduleDraft,
	staffRequirements map[string]map[string]int, // shiftID -> date -> requiredCount
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	maxFillRounds int,
) *FinalValidationResult {
	e.logger.Info("Validating final schedule (strict mode)",
		"shiftCount", len(shifts),
		"requirementShiftCount", len(staffRequirements),
		"maxFillRounds", maxFillRounds)

	if maxFillRounds <= 0 {
		maxFillRounds = 2 // 默认最大补充2轮
	}

	result := &FinalValidationResult{
		Passed:          true,
		ShortageDetails: make([]*ShortageDetail, 0),
		SupplementTasks: make([]*d_model.ProgressiveTask, 0),
	}

	// 构建班次ID到名称的映射（同时作为选中班次的白名单）
	shiftNameMap := make(map[string]string)
	for _, shift := range shifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	// 【关键修复】只检查 SelectedShifts 中包含的班次，忽略 staffRequirements 中多余的班次
	// 原问题：staffRequirements 可能包含所有 10 个班次的需求，但 SelectedShifts 只有用户选择的 5 个
	// 这会导致最终校验报告未参与排班的班次缺员，生成错误的补充任务
	skippedShifts := 0
	for shiftID, dateRequirements := range staffRequirements {
		shiftName, isSelected := shiftNameMap[shiftID]
		if !isSelected {
			// 该班次不在选中班次列表中，跳过校验
			skippedShifts++
			continue
		}
		if shiftName == "" {
			shiftName = shiftID
		}

		for date, requiredCount := range dateRequirements {
			if requiredCount <= 0 {
				continue
			}

			// 获取实际排班人数
			actualCount := 0
			if workingDraft != nil && workingDraft.Shifts != nil {
				if shiftDraft, exists := workingDraft.Shifts[shiftID]; exists && shiftDraft != nil {
					if dayShift, dayExists := shiftDraft.Days[date]; dayExists && dayShift != nil {
						actualCount = len(dayShift.StaffIDs)
					}
				}
			}

			if actualCount < requiredCount {
				// 发现缺员
				result.Passed = false
				shortage := &ShortageDetail{
					ShiftID:       shiftID,
					ShiftName:     shiftName,
					Date:          date,
					RequiredCount: requiredCount,
					ActualCount:   actualCount,
					ShortageCount: requiredCount - actualCount,
				}
				result.ShortageDetails = append(result.ShortageDetails, shortage)
			} else if actualCount > requiredCount {
				// 超配（不应该发生，但记录警告）
				e.logger.Warn("Final validation found overflow",
					"shiftID", shiftID,
					"shiftName", shiftName,
					"date", date,
					"required", requiredCount,
					"actual", actualCount)
			}
		}
	}

	if skippedShifts > 0 {
		e.logger.Info("Final validation skipped non-selected shifts",
			"skippedShifts", skippedShifts,
			"selectedShifts", len(shifts),
			"totalRequirementShifts", len(staffRequirements))
	}

	if result.Passed {
		result.Summary = "最终校验通过：所有班次人数严格满足需求"
		e.logger.Info("Final schedule validation passed")
		return result
	}

	// 汇总缺员信息
	totalShortage := 0
	for _, shortage := range result.ShortageDetails {
		totalShortage += shortage.ShortageCount
	}
	result.Summary = fmt.Sprintf("最终校验失败：发现 %d 处缺员，共缺 %d 人", len(result.ShortageDetails), totalShortage)

	e.logger.Warn("Final schedule validation failed",
		"shortageCount", len(result.ShortageDetails),
		"totalShortage", totalShortage)

	// 生成 LLM 补充任务
	supplementTasks := e.generateSupplementTasks(ctx, result.ShortageDetails, shifts, rules, staffList)
	result.SupplementTasks = supplementTasks

	if len(supplementTasks) == 0 {
		result.NeedsManualIntervention = true
		result.Summary += "（无法自动生成补充任务，需人工介入）"
	}

	return result
}

// generateSupplementTasks 使用 LLM 生成补充任务
// 按班次拆分，每个缺员班次生成一个子任务
func (e *ProgressiveTaskExecutor) generateSupplementTasks(
	ctx context.Context,
	shortages []*ShortageDetail,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
) []*d_model.ProgressiveTask {
	if len(shortages) == 0 {
		return nil
	}

	e.logger.Info("Generating supplement tasks with LLM",
		"shortageCount", len(shortages))

	// 按班次分组缺员
	shiftShortages := make(map[string][]*ShortageDetail)
	for _, shortage := range shortages {
		shiftShortages[shortage.ShiftID] = append(shiftShortages[shortage.ShiftID], shortage)
	}

	tasks := make([]*d_model.ProgressiveTask, 0)
	taskIndex := 0

	for shiftID, shortageList := range shiftShortages {
		// 收集该班次的缺员日期
		dates := make([]string, 0, len(shortageList))
		totalShortage := 0
		shiftName := ""
		for _, shortage := range shortageList {
			dates = append(dates, shortage.Date)
			totalShortage += shortage.ShortageCount
			shiftName = shortage.ShiftName
		}

		// 构建任务描述
		var description strings.Builder
		description.WriteString(fmt.Sprintf("【补充任务】班次 %s 存在缺员，请补充排班：\n", shiftName))
		description.WriteString("\n**缺员详情**：\n")
		for _, shortage := range shortageList {
			description.WriteString(fmt.Sprintf("- %s：需要 %d 人，当前 %d 人，缺少 %d 人\n",
				shortage.Date, shortage.RequiredCount, shortage.ActualCount, shortage.ShortageCount))
		}
		description.WriteString(fmt.Sprintf("\n共计缺少 %d 人，请从可用人员中补充。\n", totalShortage))
		description.WriteString("\n**注意事项**：\n")
		description.WriteString("1. 遵守所有排班规则（连班规则、互斥规则等）\n")
		description.WriteString("2. 不要选择已在同一天其他冲突班次排班的人员\n")
		description.WriteString("3. 优先选择排班较少的人员实现负载均衡\n")

		// 获取该班次关联的规则ID
		ruleIDs := make([]string, 0)
		for _, rule := range rules {
			if rule == nil || !rule.IsActive {
				continue
			}
			// 检查规则是否关联到该班次
			for _, assoc := range rule.Associations {
				if assoc.AssociationType == "shift" && assoc.AssociationID == shiftID {
					ruleIDs = append(ruleIDs, rule.ID)
					break
				}
			}
		}

		task := &d_model.ProgressiveTask{
			ID:           fmt.Sprintf("supplement_%s_%d", shiftID[:8], taskIndex),
			Title:        fmt.Sprintf("补充任务：%s 缺员补充", shiftName),
			Description:  description.String(),
			Type:         "ai", // 使用AI执行，因为规则复杂
			TargetShifts: []string{shiftID},
			TargetDates:  dates,
			RuleIDs:      ruleIDs,
			Priority:     1, // 高优先级
		}

		tasks = append(tasks, task)
		taskIndex++
	}

	e.logger.Info("Generated supplement tasks",
		"taskCount", len(tasks))

	return tasks
}
