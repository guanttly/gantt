package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
	"jusha/mcp/pkg/logging"
)

// validateSingleShift 校验单个班次的排班结果
// 【占位信息格式统一】已移除 map 格式的 occupiedSlots 参数，统一使用 taskContext.OccupiedSlots（强类型数组）
func (e *ProgressiveTaskExecutor) validateSingleShift(
	ctx context.Context,
	shift *d_model.Shift,
	draft *d_model.ShiftScheduleDraft,
	rules []*d_model.Rule,
	shiftStaffRequirements map[string]int,
	staffList []*d_model.Employee,
	allShifts []*d_model.Shift, // 新增：所有班次信息，用于真正的时间重叠检查
) *d_model.RuleValidationResult {
	e.logger.Debug("Validating single shift",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"draftDates", len(draft.Schedule))

	result := &d_model.RuleValidationResult{
		Passed:               true,
		StaffCountIssues:     make([]*d_model.ValidationIssue, 0),
		ShiftRuleIssues:      make([]*d_model.ValidationIssue, 0),
		RuleComplianceIssues: make([]*d_model.ValidationIssue, 0),
	}

	// 构建班次ID到班次信息的映射
	shiftMap := make(map[string]*d_model.Shift)
	for _, s := range allShifts {
		shiftMap[s.ID] = s
	}

	// 1. 人数校验
	// 首先收集所有需要校验的日期（来自需求和草案的并集）
	allDates := make(map[string]bool)
	for date := range shiftStaffRequirements {
		allDates[date] = true
	}
	for date := range draft.Schedule {
		allDates[date] = true
	}

	// 【渐进式校验】中途只检查超配，允许人数不足（渐进填充）
	// 人数严格等于的校验放在所有任务完成后的最终校验阶段
	for date := range allDates {
		requiredCount, hasRequirement := shiftStaffRequirements[date]
		if !hasRequirement {
			continue
		}

		// 【关键修复】draft.Schedule 已经是合并后的完整结果（包含固定排班+之前的动态排班+本次LLM输出）
		// 不应该再次累加 alreadyScheduledCount，否则会导致重复计算
		actualCount := 0
		if staffIDs, ok := draft.Schedule[date]; ok {
			actualCount = len(staffIDs)
		}

		// 【渐进式校验核心逻辑】
		// - 超配（actualCount > requiredCount）：立即失败，触发重试
		// - 不足（actualCount < requiredCount）：仅记录信息，允许继续（渐进填充）
		// - 刚好（actualCount == requiredCount）：通过
		if actualCount > requiredCount {
			// 超配：严重错误，必须立即重试
			result.Passed = false
			result.StaffCountIssues = append(result.StaffCountIssues, &d_model.ValidationIssue{
				RuleID:         "staff_count_overflow",
				Type:           "staff_count",
				Severity:       "critical",
				Description:    fmt.Sprintf("%s %s: 超配！需要%d人，实际安排%d人，超出%d人", date, shift.Name, requiredCount, actualCount, actualCount-requiredCount),
				AffectedDates:  []string{date},
				AffectedShifts: []string{shift.ID},
			})
		}
		// 人数不足的情况不再标记为失败，由最终校验处理
	}

	// 2. 时间冲突校验（使用真正的时间重叠检查）
	// 【占位信息格式统一】使用 taskContext.OccupiedSlots（强类型数组）
	for date, staffIDs := range draft.Schedule {
		for _, staffID := range staffIDs {
			// 检查 taskContext.OccupiedSlots 中是否已占位
			if e.taskContext != nil {
				slot := d_model.FindOccupiedSlot(e.taskContext.OccupiedSlots, staffID, date)
				if slot != nil {
					// 如果是同一个班次，不算冲突
					if slot.ShiftID != shift.ID {
						// 获取已占用班次的信息
						existingShift := shiftMap[slot.ShiftID]

						// 检查时间是否真的重叠（超过1小时）
						hasRealOverlap := false // 如果找不到班次信息，不认为有冲突
						if existingShift != nil {
							// 使用真正的时间重叠检查（只有重叠超过1小时才算冲突）
							hasRealOverlap = utils.CheckTimeOverlap(shift, existingShift)
						}

						if hasRealOverlap {
							result.Passed = false
							existingShiftName := slot.ShiftID
							if existingShift != nil {
								existingShiftName = existingShift.Name
							}
							result.ShiftRuleIssues = append(result.ShiftRuleIssues, &d_model.ValidationIssue{
								RuleID:         "time_conflict",
								Type:           "time_conflict",
								Severity:       "error",
								Description:    fmt.Sprintf("%s: 人员与班次[%s]时间冲突", date, existingShiftName),
								AffectedDates:  []string{date},
								AffectedStaff:  []string{staffID},
								AffectedShifts: []string{shift.ID, slot.ShiftID},
							})
						}
					}
				}
			}
		}
	}

	// 3. 规则合规校验（调用规则级校验器）
	if e.ruleValidator != nil && len(rules) > 0 {
		// 构建临时的ShiftScheduleDraft用于校验（注意：ValidateAll需要ShiftScheduleDraft类型）
		tempScheduleDraft := &d_model.ShiftScheduleDraft{
			Schedule:     draft.Schedule, // 直接使用已有的schedule（date -> staffIDs）
			UpdatedStaff: make(map[string]bool),
		}
		// 填充UpdatedStaff
		for _, staffIDs := range draft.Schedule {
			for _, staffID := range staffIDs {
				tempScheduleDraft.UpdatedStaff[staffID] = true
			}
		}

		// 调用规则校验器
		allShifts := []*d_model.Shift{shift}
		var allStaffList []*d_model.Employee
		if e.taskContext != nil && e.taskContext.AllStaff != nil {
			allStaffList = e.taskContext.AllStaff
		} else {
			allStaffList = staffList
		}

		// 【占位信息格式统一】调用ValidateAll进行综合校验（使用强类型数组）
		var occupiedSlotsForValidation map[string]map[string]string
		if e.taskContext != nil {
			occupiedSlotsForValidation = d_model.ConvertOccupiedSlotsToMap(e.taskContext.OccupiedSlots)
		}
		ruleValidationResult, err := e.ruleValidator.ValidateAll(
			ctx,
			tempScheduleDraft,
			map[string]int{}, // staffRequirements - 使用空map，因为我们已经单独做了人数校验
			allShifts,
			rules,
			allStaffList,
			occupiedSlotsForValidation, // 传入占位信息（转换为map格式以兼容旧接口）
			map[string][]string{},      // fixedShiftAssignments - 空map表示没有固定排班
		)
		if err != nil {
			e.logger.Error("Rule validation failed with error", "error", err)
			result.Passed = false
			result.Summary = fmt.Sprintf("规则校验执行失败: %v", err)
			return result
		}

		if ruleValidationResult != nil && !ruleValidationResult.Passed {
			result.Passed = false
			// 复制所有类型的校验问题
			result.StaffCountIssues = append(result.StaffCountIssues, ruleValidationResult.StaffCountIssues...)
			result.ShiftRuleIssues = append(result.ShiftRuleIssues, ruleValidationResult.ShiftRuleIssues...)
			result.RuleComplianceIssues = append(result.RuleComplianceIssues, ruleValidationResult.RuleComplianceIssues...)
		}
	}

	// 构建总结
	if !result.Passed {
		summaryParts := make([]string, 0)
		if len(result.StaffCountIssues) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("人数问题%d个", len(result.StaffCountIssues)))
		}
		if len(result.ShiftRuleIssues) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("时间冲突%d个", len(result.ShiftRuleIssues)))
		}
		if len(result.RuleComplianceIssues) > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("规则违反%d个", len(result.RuleComplianceIssues)))
		}
		result.Summary = fmt.Sprintf("校验失败：%s", strings.Join(summaryParts, "，"))
	} else {
		result.Summary = "校验通过"
	}

	e.logger.Info("Single shift validation completed",
		"shiftID", shift.ID,
		"passed", result.Passed,
		"summary", result.Summary)

	return result
}

// cleanShiftOccupiedSlots 清理本班次在 taskContext.OccupiedSlots 中的占位记录
// 【占位信息格式统一】已改为使用强类型数组
// 用于重试前清理，避免自我冲突
func (e *ProgressiveTaskExecutor) cleanShiftOccupiedSlots(
	shift *d_model.Shift,
	targetDates []string,
	draft *d_model.ShiftScheduleDraft,
) {
	e.logger.Debug("Cleaning shift occupied slots",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"targetDatesCount", len(targetDates))

	cleanedCount := 0

	// 【占位信息格式统一】清理 taskContext.OccupiedSlots 中本班次的占位
	if e.taskContext != nil {
		result := make([]d_model.StaffOccupiedSlot, 0, len(e.taskContext.OccupiedSlots))
		for _, slot := range e.taskContext.OccupiedSlots {
			// 只保留不是本班次 || 不在目标日期范围内的占位
			if slot.ShiftID != shift.ID || (len(targetDates) > 0 && !contains(targetDates, slot.Date)) {
				result = append(result, slot)
			} else {
				cleanedCount++
				e.logger.Debug("Cleaned occupied slot",
					"staffID", slot.StaffID,
					"date", slot.Date,
					"shiftID", slot.ShiftID)
			}
		}
		e.taskContext.OccupiedSlots = result
	}

	// 清理draft中的数据
	if draft != nil && draft.Schedule != nil {
		for date := range draft.Schedule {
			if len(targetDates) == 0 || contains(targetDates, date) {
				delete(draft.Schedule, date)
			}
		}
	}
}

// analyzeShiftFailureWithAI 使用AI分析班次失败原因
func (e *ProgressiveTaskExecutor) analyzeShiftFailureWithAI(
	ctx context.Context,
	shift *d_model.Shift,
	task *d_model.ProgressiveTask,
	validationResult *d_model.RuleValidationResult,
	retryHistory []string,
) string {
	var prompt strings.Builder

	prompt.WriteString("你是一个排班失败分析专家。请分析以下班次排班失败的原因，并提供改进建议。\n\n")

	prompt.WriteString(fmt.Sprintf("**班次信息**：%s (%s-%s)\n", shift.Name, shift.StartTime, shift.EndTime))
	prompt.WriteString(fmt.Sprintf("**任务描述**：%s\n\n", task.Description))

	prompt.WriteString("**校验失败详情**：\n")
	if validationResult != nil {
		prompt.WriteString(fmt.Sprintf("- 总结：%s\n", validationResult.Summary))
		if len(validationResult.StaffCountIssues) > 0 {
			prompt.WriteString("\n人数问题：\n")
			for _, issue := range validationResult.StaffCountIssues {
				prompt.WriteString(fmt.Sprintf("  - %s\n", issue.Description))
			}
		}
		if len(validationResult.ShiftRuleIssues) > 0 {
			prompt.WriteString("\n时间冲突：\n")
			for _, issue := range validationResult.ShiftRuleIssues {
				prompt.WriteString(fmt.Sprintf("  - %s\n", issue.Description))
			}
		}
		if len(validationResult.RuleComplianceIssues) > 0 {
			prompt.WriteString("\n规则违反：\n")
			for i, issue := range validationResult.RuleComplianceIssues {
				if i >= 5 {
					prompt.WriteString(fmt.Sprintf("  ... 还有%d个规则问题\n", len(validationResult.RuleComplianceIssues)-5))
					break
				}
				prompt.WriteString(fmt.Sprintf("  - %s\n", issue.Description))
			}
		}
	}

	if len(retryHistory) > 0 {
		prompt.WriteString("\n**历史失败记录**：\n")
		for i, history := range retryHistory {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, history))
		}
	}

	prompt.WriteString("\n**请提供**：\n")
	prompt.WriteString("1. 失败的根本原因分析（简要，1-2句话）\n")
	prompt.WriteString("2. 具体的改进建议（针对性的操作建议，如\"减少人数\"、\"更换人员\"、\"调整日期范围\"等）\n")
	prompt.WriteString("\n请用简洁的语言回复，重点突出可操作的建议。\n")

	systemPrompt := "你是一个排班失败分析专家。请简要分析问题并提供可操作的改进建议。"
	userPrompt := prompt.String()

	// 调用LLM
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录到调试文件
	e.logLLMDebug(task.Title, logging.LLMCallFailureAnalysis, shift.Name, "", systemPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Failure analysis LLM call failed", "shiftID", shift.ID, "error", err)
		return "AI分析失败，请根据校验结果手动调整"
	}

	recommendation := strings.TrimSpace(resp.Content)
	return recommendation
}

// formatFailureToSemantic 将校验失败转为简要语义化描述
func (e *ProgressiveTaskExecutor) formatFailureToSemantic(
	draft *d_model.ShiftScheduleDraft,
	validationResult *d_model.RuleValidationResult,
	shiftName string,
) string {
	var issues []string

	// 人数问题
	if len(validationResult.StaffCountIssues) > 0 {
		for _, issue := range validationResult.StaffCountIssues {
			issues = append(issues, issue.Description)
		}
	}

	// 时间冲突 - 包含具体详情
	if len(validationResult.ShiftRuleIssues) > 0 {
		var conflictDetails []string
		for i, issue := range validationResult.ShiftRuleIssues {
			if i >= 3 { // 最多显示3个，避免prompt过长
				conflictDetails = append(conflictDetails, fmt.Sprintf("等共%d个时间冲突", len(validationResult.ShiftRuleIssues)))
				break
			}
			// 提取关键信息：描述中通常包含日期和员工信息
			conflictDetails = append(conflictDetails, issue.Description)
		}
		issues = append(issues, fmt.Sprintf("时间冲突：%s", strings.Join(conflictDetails, "；")))
	}

	// 规则违反（只显示前3个）
	if len(validationResult.RuleComplianceIssues) > 0 {
		count := len(validationResult.RuleComplianceIssues)
		if count <= 3 {
			for _, issue := range validationResult.RuleComplianceIssues {
				issues = append(issues, issue.Description)
			}
		} else {
			for i := 0; i < 3; i++ {
				issues = append(issues, validationResult.RuleComplianceIssues[i].Description)
			}
			issues = append(issues, fmt.Sprintf("等%d个规则问题", count))
		}
	}

	if len(issues) == 0 {
		return "未知错误"
	}

	return strings.Join(issues, "；")
}

// extractIssuesByDate 从校验结果中提取按日期分组的问题
// 用于重试时注入具体问题点到对应日期的Prompt
func (e *ProgressiveTaskExecutor) extractIssuesByDate(
	validationResult *d_model.RuleValidationResult,
) map[string][]string {
	result := make(map[string][]string) // date -> issues

	if validationResult == nil {
		return result
	}

	// 从人数问题提取日期
	for _, issue := range validationResult.StaffCountIssues {
		for _, date := range issue.AffectedDates {
			result[date] = append(result[date], issue.Description)
		}
		// 如果没有明确日期，尝试从描述中提取
		if len(issue.AffectedDates) == 0 && issue.Description != "" {
			// 简单匹配日期格式 YYYY-MM-DD
			if matches := e.extractDatesFromText(issue.Description); len(matches) > 0 {
				for _, date := range matches {
					result[date] = append(result[date], issue.Description)
				}
			}
		}
	}

	// 从时间冲突问题提取日期
	for _, issue := range validationResult.ShiftRuleIssues {
		for _, date := range issue.AffectedDates {
			result[date] = append(result[date], issue.Description)
		}
		if len(issue.AffectedDates) == 0 && issue.Description != "" {
			if matches := e.extractDatesFromText(issue.Description); len(matches) > 0 {
				for _, date := range matches {
					result[date] = append(result[date], issue.Description)
				}
			}
		}
	}

	// 从规则违反问题提取日期
	for _, issue := range validationResult.RuleComplianceIssues {
		for _, date := range issue.AffectedDates {
			result[date] = append(result[date], issue.Description)
		}
		if len(issue.AffectedDates) == 0 && issue.Description != "" {
			if matches := e.extractDatesFromText(issue.Description); len(matches) > 0 {
				for _, date := range matches {
					result[date] = append(result[date], issue.Description)
				}
			}
		}
	}

	return result
}

// extractTargetRetryInfo 从校验结果中提取需要重排的目标信息
// 返回：需要重排的日期列表和冲突的班次ID列表（如有）
func (e *ProgressiveTaskExecutor) extractTargetRetryInfo(
	validationResult *d_model.RuleValidationResult,
	currentShiftID string,
) (targetDates []string, conflictingShiftIDs []string) {
	if validationResult == nil {
		return nil, nil
	}

	// 使用map去重日期
	dateSet := make(map[string]bool)
	// 记录冲突的班次（优先取互斥规则中的）
	conflictShiftIDSet := make(map[string]bool)

	// 遍历所有校验问题，提取日期和冲突班次
	allIssues := append(validationResult.StaffCountIssues, validationResult.ShiftRuleIssues...)
	allIssues = append(allIssues, validationResult.RuleComplianceIssues...)

	for _, issue := range allIssues {
		if issue == nil {
			continue
		}

		// 提取受影响的日期
		for _, date := range issue.AffectedDates {
			dateSet[date] = true
		}
		// 如果没有明确日期，尝试从描述中提取
		if len(issue.AffectedDates) == 0 && issue.Description != "" {
			if matches := e.extractDatesFromText(issue.Description); len(matches) > 0 {
				for _, date := range matches {
					dateSet[date] = true
				}
			}
		}

		// 提取冲突的班次ID（排除当前班次）
		for _, shiftID := range issue.AffectedShifts {
			if shiftID != currentShiftID && shiftID != "" {
				conflictShiftIDSet[shiftID] = true
			}
		}
	}

	// 转换日期map为有序列表
	for date := range dateSet {
		targetDates = append(targetDates, date)
	}
	sort.Strings(targetDates)

	// 转换冲突班次map为有序列表（支持多个冲突班次）
	for shiftID := range conflictShiftIDSet {
		conflictingShiftIDs = append(conflictingShiftIDs, shiftID)
	}
	sort.Strings(conflictingShiftIDs)

	e.logger.Debug("Extracted target retry info from validation result",
		"currentShiftID", currentShiftID,
		"targetDates", targetDates,
		"conflictingShiftIDs", conflictingShiftIDs,
		"totalIssues", len(allIssues))

	return targetDates, conflictingShiftIDs
}

// validateTaskRuleMatching 任务间规则匹配度校验（已废弃）
// 注意：此函数已不再使用，校验逻辑已移至单日排班后（validateDayScheduleWithLLM）
// 保留此函数仅用于向后兼容，不应再调用
// Deprecated: 请使用 validateDayScheduleWithLLM 进行单日排班后的校验
func (e *ProgressiveTaskExecutor) validateTaskRuleMatching(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shiftDrafts map[string]*d_model.ShiftScheduleDraft,
	rules []*d_model.Rule,
	shifts []*d_model.Shift,
) *RuleMatchingResult {
	result := &RuleMatchingResult{
		Passed:     true,
		MatchScore: 1.0,
		Issues:     make([]*RuleMatchingIssue, 0),
		NeedsRetry: false,
	}

	// 收集所有激活的规则，交给LLM统一校验（不做硬编码预筛选）
	relevantRules := make([]*d_model.Rule, 0)
	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}
		relevantRules = append(relevantRules, rule)
	}

	if len(relevantRules) == 0 {
		e.logger.Debug("No relevant rules for task matching validation",
			"taskID", task.ID)
		result.LLMAnalysis = "无需校验的规则"
		return result
	}

	// 构建LLM Prompt
	var prompt strings.Builder
	prompt.WriteString("你是一个排班规则匹配度分析专家。请分析以下排班结果是否正确遵守了相关规则。\n\n")

	// 任务信息
	prompt.WriteString("## 任务信息\n")
	prompt.WriteString(fmt.Sprintf("- 任务ID：%s\n", task.ID))
	prompt.WriteString(fmt.Sprintf("- 任务标题：%s\n", task.Title))
	prompt.WriteString(fmt.Sprintf("- 任务描述：%s\n", task.Description))
	if len(task.TargetDates) > 0 {
		prompt.WriteString(fmt.Sprintf("- 目标日期：%s\n", strings.Join(task.TargetDates, ", ")))
	}
	prompt.WriteString("\n")

	// 排班结果
	prompt.WriteString("## 排班结果\n")
	// 构建班次ID到名称的映射
	shiftNameMap := make(map[string]string)
	for _, shift := range shifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	for shiftID, draft := range shiftDrafts {
		shiftName := shiftNameMap[shiftID]
		if shiftName == "" {
			shiftName = e.taskContext.MaskShiftID(shiftID)
		}
		prompt.WriteString(fmt.Sprintf("### 班次：%s\n", shiftName))
		if draft != nil && draft.Schedule != nil {
			for date, staffIDs := range draft.Schedule {
				// 转换为shortID，禁止UUID泄漏给LLM
				maskedIDs := make([]string, 0, len(staffIDs))
				for _, sid := range staffIDs {
					maskedIDs = append(maskedIDs, e.taskContext.MaskStaffID(sid))
				}
				if len(maskedIDs) <= 5 {
					prompt.WriteString(fmt.Sprintf("- %s：%d人 %v\n", date, len(maskedIDs), maskedIDs))
				} else {
					prompt.WriteString(fmt.Sprintf("- %s：%d人 [%s, %s, ... 等]\n", date, len(maskedIDs), maskedIDs[0], maskedIDs[1]))
				}
			}
		} else {
			prompt.WriteString("- 无排班数据\n")
		}
		prompt.WriteString("\n")
	}

	// 已有排班（从WorkingDraft获取）
	if e.taskContext != nil && e.taskContext.WorkingDraft != nil && len(e.taskContext.WorkingDraft.Shifts) > 0 {
		prompt.WriteString("## 已有排班（其他班次）\n")
		for existingShiftID, shiftDraft := range e.taskContext.WorkingDraft.Shifts {
			// 跳过当前任务的班次
			if _, isCurrentShift := shiftDrafts[existingShiftID]; isCurrentShift {
				continue
			}
			existingShiftName := shiftNameMap[existingShiftID]
			if existingShiftName == "" {
				existingShiftName = existingShiftID
			}
			prompt.WriteString(fmt.Sprintf("### %s\n", existingShiftName))
			if shiftDraft != nil && shiftDraft.Days != nil {
				for date, dayShift := range shiftDraft.Days {
					if dayShift != nil && len(dayShift.StaffIDs) > 0 {
						prompt.WriteString(fmt.Sprintf("- %s：%v\n", date, dayShift.Staff))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	// 需要校验的规则
	prompt.WriteString("## 需要校验的规则\n")
	for i, rule := range relevantRules {
		prompt.WriteString(fmt.Sprintf("\n### 规则 %d：%s (优先级: %d)\n", i+1, rule.Name, rule.Priority))
		prompt.WriteString(fmt.Sprintf("- ID：%s\n", e.taskContext.MaskRuleID(rule.ID)))
		prompt.WriteString(fmt.Sprintf("- 类型：%s\n", rule.RuleType))
		prompt.WriteString(fmt.Sprintf("- 说明：%s\n", rule.Description))
		if rule.RuleData != "" {
			prompt.WriteString(fmt.Sprintf("- 规则数据：%s\n", rule.RuleData))
		}
	}

	prompt.WriteString("\n## 分析要求\n")
	prompt.WriteString("请逐条分析每个规则是否被正确遵守，并输出JSON格式的结果：\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"passed\": true/false,\n")
	prompt.WriteString("  \"matchScore\": 0.0-1.0,\n")
	prompt.WriteString("  \"issues\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"ruleId\": \"规则ID\",\n")
	prompt.WriteString("      \"ruleName\": \"规则名称\",\n")
	prompt.WriteString("      \"severity\": \"critical/warning\",\n")
	prompt.WriteString("      \"description\": \"问题描述\",\n")
	prompt.WriteString("      \"suggestion\": \"改进建议\"\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ],\n")
	prompt.WriteString("  \"analysis\": \"整体分析说明\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("```\n")
	prompt.WriteString("\n注意：\n")
	prompt.WriteString("- 如果所有规则都被正确遵守，passed=true，issues为空数组\n")
	prompt.WriteString("- severity=\"critical\" 表示严重问题（违反规则数据中的明确要求），severity=\"warning\" 表示轻微问题\n")
	prompt.WriteString("- 只输出JSON，不要其他内容\n")

	systemPrompt := "你是一个排班规则匹配度分析专家。请严格根据提供的规则数据原文校验排班结果，并以JSON格式输出结果。"
	userPrompt := prompt.String()

	// 调用LLM
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录到调试文件
	e.logLLMDebug(task.Title, logging.LLMCallRuleMatching, "", "", systemPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Rule matching validation LLM call failed", "taskID", task.ID, "error", err)
		// LLM调用失败时，返回通过（不阻塞流程）
		result.LLMAnalysis = fmt.Sprintf("规则匹配校验失败：%v", err)
		return result
	}

	// 解析LLM响应
	respContent := strings.TrimSpace(resp.Content)
	// 提取JSON部分
	jsonStart := strings.Index(respContent, "{")
	jsonEnd := strings.LastIndex(respContent, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		respContent = respContent[jsonStart : jsonEnd+1]
	}

	var llmResult struct {
		Passed     bool    `json:"passed"`
		MatchScore float64 `json:"matchScore"`
		Issues     []struct {
			RuleID        string   `json:"ruleId"`
			RuleName      string   `json:"ruleName"`
			Severity      string   `json:"severity"`
			Description   string   `json:"description"`
			Suggestion    string   `json:"suggestion"`
			AffectedDates []string `json:"affectedDates,omitempty"`
		} `json:"issues"`
		Analysis string `json:"analysis"`
	}

	if err := json.Unmarshal([]byte(respContent), &llmResult); err != nil {
		e.logger.Warn("Failed to parse LLM rule matching response", "taskID", task.ID, "error", err)
		// 解析失败时，返回通过（不阻塞流程）
		result.LLMAnalysis = fmt.Sprintf("规则匹配结果解析失败：%v", err)
		return result
	}

	// 转换结果
	result.Passed = llmResult.Passed
	result.MatchScore = llmResult.MatchScore
	result.LLMAnalysis = llmResult.Analysis

	// 构建规则ID到优先级的映射（同时使用真实UUID和masked shortID作为key）
	rulePriorityMap := make(map[string]int)
	ruleTypeMap := make(map[string]string)
	for _, rule := range relevantRules {
		rulePriorityMap[rule.ID] = rule.Priority
		ruleTypeMap[rule.ID] = rule.RuleType
		// 同时用masked shortID做key，因为LLM返回的是shortID
		if e.taskContext != nil {
			maskedID := e.taskContext.MaskRuleID(rule.ID)
			rulePriorityMap[maskedID] = rule.Priority
			ruleTypeMap[maskedID] = rule.RuleType
		}
	}

	for _, issue := range llmResult.Issues {
		priority := rulePriorityMap[issue.RuleID]
		if priority == 0 {
			priority = 5 // 默认优先级
		}

		matchingIssue := &RuleMatchingIssue{
			RuleID:        issue.RuleID,
			RuleName:      issue.RuleName,
			RuleType:      ruleTypeMap[issue.RuleID],
			Priority:      priority,
			Severity:      issue.Severity,
			Description:   issue.Description,
			AffectedDates: issue.AffectedDates,
			Suggestion:    issue.Suggestion,
		}
		result.Issues = append(result.Issues, matchingIssue)

		// 高优先级规则（priority <= 3）失败需要重试
		if issue.Severity == "critical" && priority <= 3 {
			result.NeedsRetry = true
		}
		// 【P0修复】人数缺员是critical问题，无条件触发重试
		// LLM可能用不存在的ruleID报告人数问题，此时priority会fallback为5
		if issue.Severity == "critical" && rulePriorityMap[issue.RuleID] == 0 {
			result.NeedsRetry = true
		}
	}

	if !result.Passed {
		e.logger.Warn("Task rule matching validation failed",
			"taskID", task.ID,
			"issueCount", len(result.Issues),
			"needsRetry", result.NeedsRetry,
			"analysis", result.LLMAnalysis)
	} else {
		e.logger.Info("Task rule matching validation passed",
			"taskID", task.ID,
			"matchScore", result.MatchScore)
	}

	return result
}

// validateDayScheduleWithLLM 单日排班后LLM校验
// 在单日排班完成后调用，校验到当前日期为止的所有已排班结果（累积数据）
// 如果校验失败，返回校验问题列表
func (e *ProgressiveTaskExecutor) validateDayScheduleWithLLM(
	ctx context.Context,
	targetShift *d_model.Shift,
	targetDate string,
	completedDates []string,
	draft *d_model.ShiftScheduleDraft,
	allShifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	workingDraft *d_model.ScheduleDraft,
	staffRequirements map[string]int, // 新增：人数需求，用于检查人数是否满足
) *RuleMatchingResult {
	result := &RuleMatchingResult{
		Passed:     true,
		MatchScore: 1.0,
		Issues:     make([]*RuleMatchingIssue, 0),
		NeedsRetry: false,
	}

	// 收集所有激活的规则
	relevantRules := make([]*d_model.Rule, 0)
	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}
		relevantRules = append(relevantRules, rule)
	}

	if len(relevantRules) == 0 {
		e.logger.Debug("No relevant rules for day schedule validation",
			"shiftID", targetShift.ID,
			"date", targetDate)
		result.LLMAnalysis = "无需校验的规则"
		return result
	}

	// 构建LLM Prompt
	var prompt strings.Builder
	prompt.WriteString("你是一个排班规则匹配度分析专家。请分析以下排班结果是否正确遵守了相关规则。\n\n")

	// 班次信息
	prompt.WriteString("## 班次信息\n")
	prompt.WriteString(fmt.Sprintf("- 班次ID：%s\n", e.taskContext.MaskShiftID(targetShift.ID)))
	prompt.WriteString(fmt.Sprintf("- 班次名称：%s\n", targetShift.Name))
	prompt.WriteString(fmt.Sprintf("- 班次时间：%s-%s\n", targetShift.StartTime, targetShift.EndTime))
	prompt.WriteString(fmt.Sprintf("- 当前日期：%s\n", targetDate))
	prompt.WriteString(fmt.Sprintf("- 已完成日期：%s\n", strings.Join(completedDates, ", ")))
	prompt.WriteString("\n")

	// 构建人员ID到姓名的映射（用于在prompt中显示姓名）
	staffNamesMap := BuildStaffNamesMap(staffList)

	// 构建UUID到shortID的映射
	uuidToShortID := make(map[string]string)
	if e.taskContext != nil && e.taskContext.StaffReverseMappings != nil {
		for shortID, uuid := range e.taskContext.StaffReverseMappings {
			uuidToShortID[uuid] = shortID
		}
	}

	// 【修复】获取当前班次的固定排班数据
	shiftFixedAssignments := make(map[string][]string)
	if e.taskContext != nil && len(e.taskContext.FixedAssignments) > 0 {
		for _, fa := range e.taskContext.FixedAssignments {
			if fa.ShiftID == targetShift.ID && len(fa.StaffIDs) > 0 {
				shiftFixedAssignments[fa.Date] = fa.StaffIDs
			}
		}
	}

	// 当前班次的排班结果（累积到当前日期，包含固定排班）
	prompt.WriteString("## 当前班次排班结果（累积）\n")
	hasAnySchedule := false
	// 只显示已完成日期的排班
	for _, date := range completedDates {
		// 合并固定排班 + 动态排班
		allStaffIDs := make([]string, 0)
		staffIDSet := make(map[string]bool)
		isFixed := make(map[string]bool)

		// 先加入固定排班
		if fixedIDs, ok := shiftFixedAssignments[date]; ok {
			for _, id := range fixedIDs {
				if !staffIDSet[id] {
					allStaffIDs = append(allStaffIDs, id)
					staffIDSet[id] = true
					isFixed[id] = true
				}
			}
		}

		// 再加入动态排班
		if draft != nil && draft.Schedule != nil {
			if dynamicIDs, ok := draft.Schedule[date]; ok {
				for _, id := range dynamicIDs {
					if !staffIDSet[id] {
						allStaffIDs = append(allStaffIDs, id)
						staffIDSet[id] = true
					}
				}
			}
		}

		// 如果该日期有人员，输出
		if len(allStaffIDs) > 0 {
			hasAnySchedule = true
			// 转换为姓名列表（标注固定排班）
			staffNames := make([]string, 0, len(allStaffIDs))
			displayIDs := make([]string, 0, len(allStaffIDs))
			for _, uuid := range allStaffIDs {
				name := staffNamesMap[uuid]
				if name == "" {
					name = e.taskContext.GetStaffName(uuid)
				}
				if isFixed[uuid] {
					name = name + "[固定]"
				}
				staffNames = append(staffNames, name)

				shortID := uuidToShortID[uuid]
				if shortID == "" {
					shortID = e.taskContext.MaskStaffID(uuid)
				}
				displayIDs = append(displayIDs, shortID)
			}

			if len(allStaffIDs) <= 5 {
				prompt.WriteString(fmt.Sprintf("- %s：%d人 %v (ID: %v)\n", date, len(allStaffIDs), staffNames, displayIDs))
			} else {
				displayCount := 3
				if len(staffNames) < displayCount {
					displayCount = len(staffNames)
				}
				prompt.WriteString(fmt.Sprintf("- %s：%d人 %v 等 (ID: %v)\n", date, len(allStaffIDs), staffNames[:displayCount], displayIDs[:displayCount]))
			}
		}
	}
	if !hasAnySchedule {
		prompt.WriteString("- 无排班数据\n")
	}
	prompt.WriteString("\n")

	// 已有排班（其他班次，从WorkingDraft获取）
	if workingDraft != nil && len(workingDraft.Shifts) > 0 {
		prompt.WriteString("## 已有排班（其他班次）\n")
		// 构建班次ID到名称的映射
		shiftNameMap := make(map[string]string)
		for _, shift := range allShifts {
			shiftNameMap[shift.ID] = shift.Name
		}

		for existingShiftID, shiftDraft := range workingDraft.Shifts {
			// 跳过当前班次
			if existingShiftID == targetShift.ID {
				continue
			}
			existingShiftName := shiftNameMap[existingShiftID]
			if existingShiftName == "" {
				existingShiftName = existingShiftID
			}
			prompt.WriteString(fmt.Sprintf("### %s\n", existingShiftName))
			if shiftDraft != nil && shiftDraft.Days != nil {
				// 只显示已完成日期的排班
				for _, date := range completedDates {
					if dayShift, ok := shiftDraft.Days[date]; ok && dayShift != nil && len(dayShift.StaffIDs) > 0 {
						prompt.WriteString(fmt.Sprintf("- %s：%v\n", date, dayShift.Staff))
					}
				}
			}
			prompt.WriteString("\n")
		}
	}

	// 需要校验的规则
	prompt.WriteString("## 需要校验的规则\n")
	for i, rule := range relevantRules {
		prompt.WriteString(fmt.Sprintf("\n### 规则 %d：%s (优先级: %d)\n", i+1, rule.Name, rule.Priority))
		prompt.WriteString(fmt.Sprintf("- ID：%s\n", e.taskContext.MaskRuleID(rule.ID)))
		prompt.WriteString(fmt.Sprintf("- 类型：%s\n", rule.RuleType))
		prompt.WriteString(fmt.Sprintf("- 说明：%s\n", rule.Description))
		if rule.RuleData != "" {
			prompt.WriteString(fmt.Sprintf("- 规则数据：%s\n", rule.RuleData))
		}
		// 标注规则的作用域：仅约束关联的人员/班次
		if len(rule.Associations) > 0 {
			hasShiftAssoc := false
			hasEmployeeAssoc := false
			var assocEmployeeNames []string
			var assocShiftNames []string
			for _, assoc := range rule.Associations {
				switch assoc.AssociationType {
				case "shift":
					hasShiftAssoc = true
					for _, s := range allShifts {
						if s.ID == assoc.AssociationID {
							assocShiftNames = append(assocShiftNames, s.Name)
							break
						}
					}
				case "employee":
					hasEmployeeAssoc = true
					name := staffNamesMap[assoc.AssociationID]
					if name == "" {
						name = e.taskContext.GetStaffName(assoc.AssociationID)
					}
					if name != "" {
						assocEmployeeNames = append(assocEmployeeNames, name)
					}
				}
			}
			if hasEmployeeAssoc && !hasShiftAssoc {
				prompt.WriteString(fmt.Sprintf("- ⚠️ 作用域：**仅约束以下人员**：%s（只需检查这些人员的排班是否违反此规则，不影响其他人员和其他班次）\n", strings.Join(assocEmployeeNames, "、")))
			} else if hasShiftAssoc && !hasEmployeeAssoc {
				prompt.WriteString(fmt.Sprintf("- ⚠️ 作用域：**仅约束以下班次**：%s\n", strings.Join(assocShiftNames, "、")))
			} else if hasShiftAssoc && hasEmployeeAssoc {
				prompt.WriteString(fmt.Sprintf("- ⚠️ 作用域：约束人员[%s]在班次[%s]中的排班\n", strings.Join(assocEmployeeNames, "、"), strings.Join(assocShiftNames, "、")))
			}
		}
	}

	// 人数需求信息（用于检查人数是否满足）
	// 【修复】需要合并固定排班 + 动态排班的人数
	if len(staffRequirements) > 0 {
		prompt.WriteString("\n## 人数需求\n")
		for _, date := range completedDates {
			if required, ok := staffRequirements[date]; ok {
				// 合并固定排班 + 动态排班人数（去重）
				actualStaffSet := make(map[string]bool)
				// 加入固定排班
				if fixedIDs, exists := shiftFixedAssignments[date]; exists {
					for _, id := range fixedIDs {
						actualStaffSet[id] = true
					}
				}
				// 加入动态排班
				if draft != nil && draft.Schedule != nil {
					if dynamicIDs, exists := draft.Schedule[date]; exists {
						for _, id := range dynamicIDs {
							actualStaffSet[id] = true
						}
					}
				}
				actual := len(actualStaffSet)
				status := "✅"
				if actual < required {
					status = "❌ 缺员"
				} else if actual > required {
					status = "⚠️ 超配"
				}
				prompt.WriteString(fmt.Sprintf("- %s：需要%d人，实际%d人 %s\n", date, required, actual, status))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("\n## 分析要求\n")
	prompt.WriteString("**核心原则：只根据上方明确列出的数据校验。数据中没有的日期视为不存在，涉及该日期的规则直接判定为通过。**\n\n")
	prompt.WriteString("1. 检查每日实际人数是否满足需求，人数不足标记为 critical\n")
	prompt.WriteString("2. 逐条检查每个规则的**规则数据**，判断排班结果是否违反\n")
	prompt.WriteString("3. 只报告能从已提供数据中100%确定违反的问题，不得推测或假设未提供的数据\n")
	prompt.WriteString("\n请只输出JSON：\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\"passed\":true/false,\"matchScore\":0.0-1.0,\"issues\":[{\"ruleId\":\"规则ID\",\"ruleName\":\"规则名称\",\"severity\":\"critical/warning\",\"description\":\"具体哪个日期哪些人员违反\",\"affectedDates\":[\"日期\"]}],\"analysis\":\"整体分析\"}\n")
	prompt.WriteString("```\n")

	systemPrompt := "排班规则校验专家。严格根据提供的规则数据原文逐条校验。只根据提供的数据判断，数据中未出现的日期视为不存在，相关规则直接通过。只输出JSON。"

	// 【详细日志】记录完整的LLM输入（用于检查提示词）
	promptContent := prompt.String()
	// 调用LLM
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallDefault(ctx, systemPrompt, promptContent, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录到调试文件
	e.logLLMDebug("day_validation", logging.LLMCallDayValidation, targetShift.Name, targetDate, systemPrompt, promptContent, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Day schedule validation LLM call failed", "shiftID", targetShift.ID, "date", targetDate, "error", err)
		// LLM调用失败时，返回通过（不阻塞流程）
		result.LLMAnalysis = fmt.Sprintf("单日排班校验失败：%v", err)
		return result
	}

	// 解析LLM响应
	respContent := strings.TrimSpace(resp.Content)
	// 提取JSON部分
	jsonStart := strings.Index(respContent, "{")
	jsonEnd := strings.LastIndex(respContent, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		respContent = respContent[jsonStart : jsonEnd+1]
	}

	var llmResult struct {
		Passed     bool    `json:"passed"`
		MatchScore float64 `json:"matchScore"`
		Issues     []struct {
			RuleID        string   `json:"ruleId"`
			RuleName      string   `json:"ruleName"`
			Severity      string   `json:"severity"`
			Description   string   `json:"description"`
			AffectedDates []string `json:"affectedDates,omitempty"`
		} `json:"issues"`
		Analysis string `json:"analysis"`
	}

	if err := json.Unmarshal([]byte(respContent), &llmResult); err != nil {
		e.logger.Warn("Day schedule validation parse failed", "shiftID", targetShift.ID, "date", targetDate, "error", err)
		// 解析失败时，返回通过（不阻塞流程）
		result.LLMAnalysis = fmt.Sprintf("单日排班校验结果解析失败：%v", err)
		return result
	}

	// 转换结果
	result.Passed = llmResult.Passed
	result.MatchScore = llmResult.MatchScore
	result.LLMAnalysis = llmResult.Analysis

	// 构建规则ID到优先级的映射（同时使用真实UUID和masked shortID作为key）
	rulePriorityMap := make(map[string]int)
	ruleTypeMap := make(map[string]string)
	for _, rule := range relevantRules {
		rulePriorityMap[rule.ID] = rule.Priority
		ruleTypeMap[rule.ID] = rule.RuleType
		// 同时用masked shortID做key，因为LLM返回的是shortID
		if e.taskContext != nil {
			maskedID := e.taskContext.MaskRuleID(rule.ID)
			rulePriorityMap[maskedID] = rule.Priority
			ruleTypeMap[maskedID] = rule.RuleType
		}
	}

	for _, issue := range llmResult.Issues {
		priority := rulePriorityMap[issue.RuleID]
		if priority == 0 {
			priority = 5 // 默认优先级
		}

		matchingIssue := &RuleMatchingIssue{
			RuleID:        issue.RuleID,
			RuleName:      issue.RuleName,
			RuleType:      ruleTypeMap[issue.RuleID],
			Priority:      priority,
			Severity:      issue.Severity,
			Description:   issue.Description,
			AffectedDates: issue.AffectedDates,
			Suggestion:    "", // 不提供建议
		}
		result.Issues = append(result.Issues, matchingIssue)

		// 高优先级规则（priority <= 3）且严重问题需要重试
		if issue.Severity == "critical" && priority <= 3 {
			result.NeedsRetry = true
		}
		// 【P0修复】人数缺员是critical问题，无条件触发重试
		// LLM可能用不存在的ruleID报告人数问题，此时priority会fallback为5
		if issue.Severity == "critical" && rulePriorityMap[issue.RuleID] == 0 {
			result.NeedsRetry = true
		}
	}

	if !result.Passed {
		e.logger.Warn("单日排班LLM校验未通过",
			"shiftID", targetShift.ID,
			"date", targetDate,
			"issueCount", len(result.Issues),
			"needsRetry", result.NeedsRetry)
	}

	return result
}

// adjustScheduleWithLLM4 LLM4: 班次排班调整执行
// 结合LLM3输出的规则冲突人员，对截至当前的排班进行调整
// 输出调整后的排班结果，替换掉冲突人员
func (e *ProgressiveTaskExecutor) adjustScheduleWithLLM4(
	ctx context.Context,
	targetShift *d_model.Shift,
	allDates []string,
	draft *d_model.ShiftScheduleDraft,
	staffList []*d_model.Employee,
	staffRequirements map[string]int,
	conflictStaff map[string][]*RuleConflictStaffInfo, // LLM3输出的冲突人员：date -> ConflictStaff列表
	relevantRules []*d_model.Rule, // 相关排班规则
	relatedShiftsSchedule map[string]map[string][]string, // 相关班次排班：shiftID -> date -> staffNames
) *ScheduleAdjustmentResult {
	result := &ScheduleAdjustmentResult{
		AdjustedSchedule: make(map[string][]string),
		ReplacedStaff:    make(map[string][]string),
		NewStaff:         make(map[string][]string),
	}

	// 如果没有冲突人员，直接返回当前排班
	if len(conflictStaff) == 0 {
		e.logger.Debug("[LLM4] 无冲突人员，无需调整",
			"shiftID", targetShift.ID)
		if draft != nil && draft.Schedule != nil {
			result.AdjustedSchedule = draft.Schedule
		}
		result.Reasoning = "无冲突人员，排班无需调整"
		return result
	}

	// 构建LLM Prompt
	var prompt strings.Builder

	// 班次信息（精简）
	prompt.WriteString(fmt.Sprintf("# 班次：%s（%s-%s）\n", targetShift.Name, targetShift.StartTime, targetShift.EndTime))
	prompt.WriteString(fmt.Sprintf("# 周期：%s 至 %s\n\n", allDates[0], allDates[len(allDates)-1]))

	// 构建UUID到shortID的映射
	uuidToShortID := make(map[string]string)
	if e.taskContext != nil && e.taskContext.StaffReverseMappings != nil {
		for shortID, uuid := range e.taskContext.StaffReverseMappings {
			uuidToShortID[uuid] = shortID
		}
	}

	// 当前排班（使用姓名+shortID格式，便于LLM理解）
	prompt.WriteString("## 当前排班\n")
	if draft != nil && draft.Schedule != nil {
		for _, date := range allDates {
			if staffIDs, ok := draft.Schedule[date]; ok && len(staffIDs) > 0 {
				displayNames := make([]string, len(staffIDs))
				for i, uuid := range staffIDs {
					shortID := ""
					if sid, ok := uuidToShortID[uuid]; ok {
						shortID = sid
					} else {
						shortID = e.taskContext.MaskStaffID(uuid) // 禁止UUID泄漏
					}
					staffName := e.taskContext.GetStaffName(uuid)
					displayNames[i] = fmt.Sprintf("%s(%s)", staffName, shortID)
				}
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", date, strings.Join(displayNames, ", ")))
			}
		}
	}
	prompt.WriteString("\n")

	// 冲突人员（核心信息）
	prompt.WriteString("## 需替换的冲突人员\n")
	for date, conflicts := range conflictStaff {
		if len(conflicts) > 0 {
			for _, conflict := range conflicts {
				conflictStaffID := conflict.StaffID
				if shortID, ok := uuidToShortID[conflict.StaffID]; ok {
					conflictStaffID = shortID
				} else {
					conflictStaffID = e.taskContext.MaskStaffID(conflict.StaffID) // 禁止UUID泄漏
				}
				prompt.WriteString(fmt.Sprintf("- %s: %s(%s) - %s\n", date, conflict.StaffName, conflictStaffID, conflict.Reason))
			}
		}
	}
	prompt.WriteString("\n")

	// 可用人员（精简）
	prompt.WriteString("## 可用替换人员\n")
	staffDisplayList := make([]string, 0, len(staffList))
	for _, staff := range staffList {
		shortID := staff.ID
		if sid, ok := uuidToShortID[staff.ID]; ok {
			shortID = sid
		} else {
			shortID = e.taskContext.MaskStaffID(staff.ID) // 禁止UUID泄漏
		}
		staffDisplayList = append(staffDisplayList, fmt.Sprintf("%s(%s)", staff.Name, shortID))
	}
	if len(staffDisplayList) <= 15 {
		prompt.WriteString(fmt.Sprintf("%s\n", strings.Join(staffDisplayList, ", ")))
	} else {
		prompt.WriteString(fmt.Sprintf("%s 等（共%d人）\n", strings.Join(staffDisplayList[:15], ", "), len(staffDisplayList)))
	}
	prompt.WriteString("\n")

	// 相关班次排班信息（用于避免替换人员产生新冲突）
	if len(relatedShiftsSchedule) > 0 {
		prompt.WriteString("## 相关班次排班\n\n")
		for shiftID, dateSchedule := range relatedShiftsSchedule {
			shiftName := shiftID
			if e.taskContext != nil {
				for _, s := range e.taskContext.Shifts {
					if s.ID == shiftID {
						shiftName = s.Name
						break
					}
				}
			}
			prompt.WriteString(fmt.Sprintf("%s:\n", shiftName))
			// 按日期排序显示
			sortedDates := make([]string, 0, len(dateSchedule))
			for date := range dateSchedule {
				sortedDates = append(sortedDates, date)
			}
			sort.Strings(sortedDates)
			for _, date := range sortedDates {
				staffNames := dateSchedule[date]
				if len(staffNames) > 0 {
					prompt.WriteString(fmt.Sprintf("  - %s: %s\n", date, strings.Join(staffNames, ", ")))
				}
			}
		}
		prompt.WriteString("\n")
	}

	// 相关规则
	if len(relevantRules) > 0 {
		prompt.WriteString("## 相关规则\n\n")
		var staffIDToName map[string]string
		if e.taskContext != nil {
			staffIDToName = e.taskContext.StaffIDToName
		}
		for i, rule := range relevantRules {
			prompt.WriteString(fmt.Sprintf("%d. %s", i+1, rule.Name))
			if rule.Description != "" {
				ruleDesc := rule.Description
				if staffIDToName != nil {
					ruleDesc = utils.ReplaceStaffIDsWithNames(ruleDesc, staffIDToName)
				}
				prompt.WriteString(fmt.Sprintf(": %s", ruleDesc))
			}
			prompt.WriteString("\n")
			if rule.RuleData != "" {
				ruleData := rule.RuleData
				if staffIDToName != nil {
					ruleData = utils.ReplaceStaffIDsWithNames(ruleData, staffIDToName)
				}
				prompt.WriteString(fmt.Sprintf("   规则内容: %s\n", ruleData))
			}
		}
		prompt.WriteString("\n")
	}

	// 人数需求（精简）
	prompt.WriteString("## 人数需求\n")
	for _, date := range allDates {
		if required, ok := staffRequirements[date]; ok {
			prompt.WriteString(fmt.Sprintf("- %s: %d人\n", date, required))
		}
	}
	prompt.WriteString("\n")

	// 任务说明
	prompt.WriteString("## 任务\n")
	prompt.WriteString("替换上述冲突人员，输出调整后的完整排班。\n")
	prompt.WriteString("要求：1)只替换冲突人员，其他不变 2)保持人数满足需求 3)使用shortID 4)替换人员不能违反上述规则\n\n")
	prompt.WriteString("输出JSON：\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\"adjustedSchedule\":{\"日期\":[\"shortID\"]},\"replacedStaff\":{\"日期\":[\"被换下的shortID\"]},\"newStaff\":{\"日期\":[\"新上的shortID\"]},\"reasoning\":\"说明\"}\n")
	prompt.WriteString("```\n")

	systemPrompt := "排班调整专家。根据排班规则和相关班次排班，替换冲突人员，确保替换人员不会产生新的规则冲突。输出完整排班结果。只使用shortID，只输出JSON。"

	promptContent := prompt.String()

	// 调用LLM（使用MAX模型，LLM4排班调整需要更强的推理能力）
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallMax(ctx, systemPrompt, promptContent, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录LLM调试日志
	e.logLLMDebug("schedule_adjust_llm4", logging.LLMCallScheduleAdjust, targetShift.Name, "", systemPrompt, promptContent, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("[LLM4 Call失败] 班次排班调整调用失败",
			"shiftID", targetShift.ID,
			"shiftName", targetShift.Name,
			"duration", llmCallDuration.Seconds(),
			"error", err,
			"errorType", fmt.Sprintf("%T", err))
		// LLM调用失败时，返回原排班（不阻塞流程）
		if draft != nil && draft.Schedule != nil {
			result.AdjustedSchedule = draft.Schedule
		}
		result.Reasoning = fmt.Sprintf("排班调整失败：%v，保持原排班", err)
		return result
	}

	// 解析LLM响应
	responseContent := strings.TrimSpace(resp.Content)
	// 提取JSON部分
	jsonStart := strings.Index(responseContent, "{")
	jsonEnd := strings.LastIndex(responseContent, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		responseContent = responseContent[jsonStart : jsonEnd+1]
	}

	var llmResult struct {
		AdjustedSchedule map[string][]string `json:"adjustedSchedule"`
		ReplacedStaff    map[string][]string `json:"replacedStaff"`
		NewStaff         map[string][]string `json:"newStaff"`
		Reasoning        string              `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(responseContent), &llmResult); err != nil {
		e.logger.Warn("[LLM4解析失败] 班次排班调整结果解析失败",
			"shiftID", targetShift.ID,
			"shiftName", targetShift.Name,
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"responseLength", len(responseContent),
			"response", responseContent)
		// 解析失败时，返回原排班（不阻塞流程）
		if draft != nil && draft.Schedule != nil {
			result.AdjustedSchedule = draft.Schedule
		}
		result.Reasoning = fmt.Sprintf("排班调整结果解析失败：%v，保持原排班", err)
		return result
	}

	// 【转换shortID为UUID】LLM返回的是shortID，需要转换为UUID
	// 统一使用 taskContext.ResolveStaffID 进行ID解析（支持 shortID/中文名/UUID）

	// 转换adjustedSchedule中的shortID为UUID
	adjustedScheduleUUID := make(map[string][]string)
	for date, shortIDs := range llmResult.AdjustedSchedule {
		uuidList := make([]string, 0, len(shortIDs))
		for _, shortID := range shortIDs {
			uuidList = append(uuidList, e.taskContext.ResolveStaffID(shortID))
		}
		adjustedScheduleUUID[date] = uuidList
	}

	// 转换replacedStaff中的shortID为UUID
	replacedStaffUUID := make(map[string][]string)
	for date, shortIDs := range llmResult.ReplacedStaff {
		uuidList := make([]string, 0, len(shortIDs))
		for _, shortID := range shortIDs {
			uuidList = append(uuidList, e.taskContext.ResolveStaffID(shortID))
		}
		replacedStaffUUID[date] = uuidList
	}

	// 转换newStaff中的shortID为UUID
	newStaffUUID := make(map[string][]string)
	for date, shortIDs := range llmResult.NewStaff {
		uuidList := make([]string, 0, len(shortIDs))
		for _, shortID := range shortIDs {
			uuidList = append(uuidList, e.taskContext.ResolveStaffID(shortID))
		}
		newStaffUUID[date] = uuidList
	}

	// 转换结果（使用UUID）
	result.AdjustedSchedule = adjustedScheduleUUID
	result.ReplacedStaff = replacedStaffUUID
	result.NewStaff = newStaffUUID
	result.Reasoning = llmResult.Reasoning

	return result
}
