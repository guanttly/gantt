package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// 分批修复执行器（LLM4修复流程）
// 在所有批次校验完成后，对发现的违规问题进行分批修复
// 每批处理涉及的5个人员，通过多轮迭代直到全部修复或达到最大轮次
// ============================================================

// executeFixInBatches 分批执行LLM4修复
// 所有校验批次完成后调用，对累积的违规问题进行修复
func (e *ProgressiveTaskExecutor) executeFixInBatches(
	ctx context.Context,
	shift *d_model.Shift,
	workingDraft *d_model.ScheduleDraft,
	violations []*d_model.ValidationIssue,
	staffList []*d_model.Employee,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
	config *BatchConfig,
) (*BatchFixContext, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}

	if len(violations) == 0 {
		e.logger.Info("No violations to fix")
		return NewBatchFixContext(config.MaxFixRounds, nil), nil
	}

	e.logger.Info("Starting batch fix process (LLM4)",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"totalViolations", len(violations),
		"maxFixRounds", config.MaxFixRounds)

	// 创建修复上下文
	fixCtx := NewBatchFixContext(config.MaxFixRounds, violations)
	startTime := time.Now()

	// 多轮修复迭代
	for fixCtx.ShouldContinue() {
		fixCtx.CurrentRound++

		// 执行单轮修复
		roundResult := e.executeSingleFixRound(
			ctx, shift, workingDraft, fixCtx.CurrentViolations,
			staffList, rules, occupiedSlots, allShifts, config,
		)
		fixCtx.FixResults = append(fixCtx.FixResults, roundResult)

		// 更新当前违规列表（重新校验后的结果）
		if roundResult.ViolationsAfter == 0 {
			fixCtx.CurrentViolations = nil
			fixCtx.AllFixed = true
		} else {
			// 重新收集未解决的违规
			fixCtx.CurrentViolations = e.revalidateAfterFix(
				ctx, shift, workingDraft, staffList, rules, occupiedSlots, allShifts,
			)
		}

		// 检查是否没有进展（避免死循环）
		if roundResult.FixedCount == 0 && roundResult.ViolationsAfter > 0 {
			e.logger.Warn("No progress in fix round, may need manual intervention",
				"shiftID", shift.ID,
				"round", fixCtx.CurrentRound)
			// 继续尝试下一轮，但达到最大轮次后会自动停止
		}
	}

	fixCtx.TotalTime = time.Since(startTime).Seconds()

	e.logger.Info("Batch fix process completed",
		"shiftID", shift.ID,
		"totalRounds", fixCtx.CurrentRound,
		"allFixed", fixCtx.AllFixed,
		"initialViolations", fixCtx.InitialViolations,
		"remainingViolations", len(fixCtx.CurrentViolations),
		"totalTime", fixCtx.TotalTime)

	return fixCtx, nil
}

// executeSingleFixRound 执行单轮修复
// 将违规问题按涉及人员分组，每批5人进行LLM修复决策
func (e *ProgressiveTaskExecutor) executeSingleFixRound(
	ctx context.Context,
	shift *d_model.Shift,
	workingDraft *d_model.ScheduleDraft,
	violations []*d_model.ValidationIssue,
	staffList []*d_model.Employee,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
	config *BatchConfig,
) *BatchFixResult {
	startTime := time.Now()
	result := &BatchFixResult{
		Round:            0, // 将在外层设置
		ViolationsBefore: len(violations),
		ModifiedStaff:    make([]string, 0),
		Changes:          make([]*ScheduleChange, 0),
	}

	// 按涉及人员分组违规问题
	staffViolations := e.groupViolationsByStaff(violations)

	// 收集所有涉及的人员ID
	affectedStaffIDs := make([]string, 0, len(staffViolations))
	for staffID := range staffViolations {
		affectedStaffIDs = append(affectedStaffIDs, staffID)
	}

	// 按严重程度排序（优先处理严重问题涉及的人员）
	sort.Slice(affectedStaffIDs, func(i, j int) bool {
		viI := staffViolations[affectedStaffIDs[i]]
		viJ := staffViolations[affectedStaffIDs[j]]
		return getMaxSeverity(viI) > getMaxSeverity(viJ)
	})

	// 分批处理
	batchSize := config.BatchSize
	for i := 0; i < len(affectedStaffIDs); i += batchSize {
		end := i + batchSize
		if end > len(affectedStaffIDs) {
			end = len(affectedStaffIDs)
		}
		batchStaffIDs := affectedStaffIDs[i:end]

		// 收集该批人员的违规问题
		batchViolations := make([]*d_model.ValidationIssue, 0)
		for _, staffID := range batchStaffIDs {
			batchViolations = append(batchViolations, staffViolations[staffID]...)
		}

		// 执行单批修复
		changes := e.fixSingleBatch(
			ctx, shift, workingDraft, batchStaffIDs, batchViolations,
			staffList, rules, occupiedSlots, allShifts,
		)

		result.Changes = append(result.Changes, changes...)
		result.ModifiedStaff = append(result.ModifiedStaff, batchStaffIDs...)
	}

	result.FixedCount = len(result.Changes)
	result.ExecutionTime = time.Since(startTime).Seconds()

	// 重新校验获取修复后的违规数
	remainingViolations := e.revalidateAfterFix(
		ctx, shift, workingDraft, staffList, rules, occupiedSlots, allShifts,
	)
	result.ViolationsAfter = len(remainingViolations)

	return result
}

// fixSingleBatch 修复单批人员的违规问题
func (e *ProgressiveTaskExecutor) fixSingleBatch(
	ctx context.Context,
	shift *d_model.Shift,
	workingDraft *d_model.ScheduleDraft,
	batchStaffIDs []string,
	violations []*d_model.ValidationIssue,
	staffList []*d_model.Employee,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
) []*ScheduleChange {
	// 构建人员ID到信息的映射
	staffMap := make(map[string]*d_model.Employee)
	for _, staff := range staffList {
		staffMap[staff.ID] = staff
	}

	// 获取批次人员的详细信息
	batchStaff := make([]*d_model.Employee, 0)
	batchStaffNames := make([]string, 0)
	for _, staffID := range batchStaffIDs {
		if staff, ok := staffMap[staffID]; ok {
			batchStaff = append(batchStaff, staff)
			batchStaffNames = append(batchStaffNames, staff.Name)
		}
	}

	// 构建修复Prompt
	sysPrompt := e.buildBatchFixSystemPrompt()
	userPrompt := e.buildBatchFixUserPrompt(
		shift, batchStaff, violations, workingDraft,
		staffList, rules, occupiedSlots, allShifts,
	)

	// 调用LLM
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录LLM调试日志
	e.logLLMDebug("batch_fix", logging.LLMCallBatchFix, shift.Name, "", sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Batch fix LLM call failed",
			"shiftID", shift.ID,
			"batchStaff", batchStaffNames,
			"error", err)
		return nil
	}

	// 解析修复建议
	changes, parseErr := e.parseBatchFixResponse(resp.Content, batchStaff, staffList)
	if parseErr != nil {
		e.logger.Error("Failed to parse batch fix response",
			"shiftID", shift.ID,
			"error", parseErr)
		return nil
	}

	// 应用修复变更到WorkingDraft
	appliedChanges := e.applyFixChanges(workingDraft, shift.ID, changes, occupiedSlots)

	return appliedChanges
}

// groupViolationsByStaff 按涉及人员分组违规问题
func (e *ProgressiveTaskExecutor) groupViolationsByStaff(
	violations []*d_model.ValidationIssue,
) map[string][]*d_model.ValidationIssue {
	result := make(map[string][]*d_model.ValidationIssue)
	for _, v := range violations {
		for _, staffID := range v.AffectedStaff {
			result[staffID] = append(result[staffID], v)
		}
	}
	return result
}

// getMaxSeverity 获取违规列表中的最高严重级别
func getMaxSeverity(violations []*d_model.ValidationIssue) int {
	maxSev := 0
	for _, v := range violations {
		sev := 0
		switch v.Severity {
		case "critical":
			sev = 3
		case "error":
			sev = 2
		case "warning":
			sev = 1
		}
		if sev > maxSev {
			maxSev = sev
		}
	}
	return maxSev
}

// buildBatchFixSystemPrompt 构建批次修复的系统提示词
func (e *ProgressiveTaskExecutor) buildBatchFixSystemPrompt() string {
	return `你是一个排班修复专家。你的任务是修复员工排班中的规则违规问题。

## 修复原则
1. 优先通过替换人员来解决冲突（找可用的替代人选）
2. 如果无法替换，考虑调整排班日期
3. 删除排班是最后手段，尽量避免
4. 确保修复后不产生新的违规
5. 保持排班的总体平衡

## 输出格式（JSON）
{
  "fixes": [
    {
      "type": "replace",  // "replace", "remove", "adjust"
      "staffId": "人员ID（如staff_1）",
      "staffName": "被替换人员姓名",
      "date": "日期",
      "newStaffId": "替换人员ID（如staff_5，replace时需填）",
      "newStaffName": "替换人员姓名（replace时需填）",
      "reason": "修复原因"
    }
  ],
  "reasoning": "总体修复思路",
  "unresolved": ["无法自动修复的问题（需人工介入）"]
}`
}

// buildBatchFixUserPrompt 构建批次修复的用户提示词
func (e *ProgressiveTaskExecutor) buildBatchFixUserPrompt(
	shift *d_model.Shift,
	batchStaff []*d_model.Employee,
	violations []*d_model.ValidationIssue,
	workingDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
) string {
	var prompt strings.Builder

	prompt.WriteString("## 修复任务\n\n")
	prompt.WriteString(fmt.Sprintf("以下 **%d 名员工** 在班次【%s】存在规则违规，请提供修复方案。\n\n",
		len(batchStaff), shift.Name))

	// 班次信息
	prompt.WriteString("## 班次信息\n\n")
	prompt.WriteString(fmt.Sprintf("- 班次名称：%s\n", shift.Name))
	prompt.WriteString(fmt.Sprintf("- 班次时间：%s - %s\n\n", shift.StartTime, shift.EndTime))

	// 违规员工及其问题
	prompt.WriteString("## 违规详情\n\n")
	for i, staff := range batchStaff {
		shortID := e.maskStaffID(staff.ID, i+1)
		prompt.WriteString(fmt.Sprintf("### %d. %s（%s）\n\n", i+1, staff.Name, shortID))

		// 获取该员工当前的排班
		currentSchedule := e.getStaffScheduleForShift(workingDraft, shift.ID, staff.ID)
		if len(currentSchedule) > 0 {
			prompt.WriteString(fmt.Sprintf("**当前排班日期**：%s\n\n", strings.Join(currentSchedule, ", ")))
		}

		// 获取该员工的违规问题
		prompt.WriteString("**违规问题**：\n")
		for _, v := range violations {
			isAffected := false
			for _, affectedID := range v.AffectedStaff {
				if affectedID == staff.ID {
					isAffected = true
					break
				}
			}
			if isAffected {
				prompt.WriteString(fmt.Sprintf("- [%s] %s\n", v.Severity, v.Description))
			}
		}
		prompt.WriteString("\n")
	}

	// 可用的替换人选
	prompt.WriteString("## 可用替换人选\n\n")
	availableStaff := e.findAvailableReplacements(staffList, batchStaff, occupiedSlots, shift.ID)
	if len(availableStaff) == 0 {
		prompt.WriteString("暂无合适的替换人选\n\n")
	} else {
		// 使用 姓名(staff_N) 格式列出可用替换人选
		replacementItems := make([]string, 0, len(availableStaff))
		for _, staff := range availableStaff {
			shortID := e.maskStaffID(staff.ID, len(replacementItems)+100)
			// 计算当月排班天数
			scheduleCount := d_model.CountOccupiedByStaff(*occupiedSlots, staff.ID)
			replacementItems = append(replacementItems, fmt.Sprintf("%s(%s)[已排%d天]", staff.Name, shortID, scheduleCount))
		}
		prompt.WriteString(fmt.Sprintf("可选（%d人）：%s\n\n", len(replacementItems), strings.Join(replacementItems, ", ")))
	}

	// 相关规则
	prompt.WriteString("## 相关规则\n\n")
	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}
		prompt.WriteString(fmt.Sprintf("- **%s**：%s\n", rule.Name, rule.Description))
	}
	prompt.WriteString("\n")

	prompt.WriteString("## 请输出修复方案（JSON格式）\n")

	return prompt.String()
}

// getStaffScheduleForShift 获取员工在指定班次的排班日期
func (e *ProgressiveTaskExecutor) getStaffScheduleForShift(
	workingDraft *d_model.ScheduleDraft,
	shiftID string,
	staffID string,
) []string {
	result := make([]string, 0)
	if workingDraft == nil || workingDraft.Shifts == nil {
		return result
	}
	shiftDraft, exists := workingDraft.Shifts[shiftID]
	if !exists || shiftDraft == nil || shiftDraft.Days == nil {
		return result
	}
	for date, dayShift := range shiftDraft.Days {
		if dayShift == nil {
			continue
		}
		for _, id := range dayShift.StaffIDs {
			if id == staffID {
				result = append(result, date)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}

// findAvailableReplacements 查找可用的替换人选
func (e *ProgressiveTaskExecutor) findAvailableReplacements(
	allStaff []*d_model.Employee,
	excludeStaff []*d_model.Employee,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	shiftID string,
) []*d_model.Employee {
	// 构建排除集合
	excludeSet := make(map[string]bool)
	for _, staff := range excludeStaff {
		excludeSet[staff.ID] = true
	}

	result := make([]*d_model.Employee, 0)
	for _, staff := range allStaff {
		if excludeSet[staff.ID] {
			continue
		}
		// 简单过滤：排班天数较少的优先
		scheduleCount := d_model.CountOccupiedByStaff(*occupiedSlots, staff.ID)
		// 排班不超过20天的可以作为替换人选
		if scheduleCount < 20 {
			result = append(result, staff)
		}
	}

	// 按排班天数排序（少的优先）
	sort.Slice(result, func(i, j int) bool {
		countI := d_model.CountOccupiedByStaff(*occupiedSlots, result[i].ID)
		countJ := d_model.CountOccupiedByStaff(*occupiedSlots, result[j].ID)
		return countI < countJ
	})

	// 限制返回数量
	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

// parseBatchFixResponse 解析批次修复响应
func (e *ProgressiveTaskExecutor) parseBatchFixResponse(
	response string,
	batchStaff []*d_model.Employee,
	allStaff []*d_model.Employee,
) ([]*ScheduleChange, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("failed to extract JSON from fix response")
	}

	var result struct {
		Fixes []struct {
			Type         string `json:"type"`
			StaffID      string `json:"staffId"`
			StaffName    string `json:"staffName"`
			Date         string `json:"date"`
			NewStaffID   string `json:"newStaffId"`
			NewStaffName string `json:"newStaffName"`
			Reason       string `json:"reason"`
		} `json:"fixes"`
		Reasoning  string   `json:"reasoning"`
		Unresolved []string `json:"unresolved"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse fix response: %w", err)
	}

	changes := make([]*ScheduleChange, 0)
	for _, fix := range result.Fixes {
		change := &ScheduleChange{
			Date:       fix.Date,
			ChangeType: fix.Type,
			Reason:     fix.Reason,
		}

		// 统一通过 resolveStaffFromLLM 将 shortID/姓名 定位到真实人员
		if staff := e.resolveStaffFromLLM(fix.StaffID, fix.StaffName, allStaff); staff != nil {
			change.StaffID = staff.ID
			change.StaffName = staff.Name
		}

		// 解析替换人员（如果是replace类型）
		if fix.Type == "replace" {
			if staff := e.resolveStaffFromLLM(fix.NewStaffID, fix.NewStaffName, allStaff); staff != nil {
				change.NewShiftID = staff.ID // 复用字段存储新人员ID
			}
		}

		changes = append(changes, change)
	}

	return changes, nil
}

// applyFixChanges 应用修复变更到WorkingDraft
func (e *ProgressiveTaskExecutor) applyFixChanges(
	workingDraft *d_model.ScheduleDraft,
	shiftID string,
	changes []*ScheduleChange,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
) []*ScheduleChange {
	if workingDraft == nil || workingDraft.Shifts == nil {
		return nil
	}

	shiftDraft, exists := workingDraft.Shifts[shiftID]
	if !exists || shiftDraft == nil {
		return nil
	}

	appliedChanges := make([]*ScheduleChange, 0)

	for _, change := range changes {
		if change.Date == "" || change.StaffID == "" {
			continue
		}

		dayShift, dayExists := shiftDraft.Days[change.Date]
		if !dayExists || dayShift == nil {
			continue
		}

		switch change.ChangeType {
		case "remove":
			// 从排班中移除该人员
			newStaffIDs := make([]string, 0)
			for _, id := range dayShift.StaffIDs {
				if id != change.StaffID {
					newStaffIDs = append(newStaffIDs, id)
				}
			}
			dayShift.StaffIDs = newStaffIDs

			// 更新occupiedSlots
			*occupiedSlots = d_model.RemoveOccupiedSlot(*occupiedSlots, change.StaffID, change.Date)

			appliedChanges = append(appliedChanges, change)

		case "replace":
			newStaffID := change.NewShiftID // 复用字段存储新人员ID
			if newStaffID == "" {
				continue
			}

			// 替换人员
			for i, id := range dayShift.StaffIDs {
				if id == change.StaffID {
					dayShift.StaffIDs[i] = newStaffID
					break
				}
			}

			// 更新occupiedSlots：移除旧人员，添加新人员
			*occupiedSlots = d_model.RemoveOccupiedSlot(*occupiedSlots, change.StaffID, change.Date)
			*occupiedSlots = d_model.AddOccupiedSlotIfNotExists(*occupiedSlots, d_model.StaffOccupiedSlot{
				StaffID: newStaffID,
				Date:    change.Date,
				ShiftID: shiftID,
				Source:  "fix",
			})

			appliedChanges = append(appliedChanges, change)

		case "adjust":
			// 调整日期（需要更复杂的逻辑，暂时记录但不自动处理）
			e.logger.Warn("Fix type 'adjust' requires manual review",
				"shiftID", shiftID,
				"date", change.Date,
				"staffID", change.StaffID,
				"reason", change.Reason)
		}
	}

	return appliedChanges
}

// revalidateAfterFix 修复后重新校验
func (e *ProgressiveTaskExecutor) revalidateAfterFix(
	ctx context.Context,
	shift *d_model.Shift,
	workingDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
) []*d_model.ValidationIssue {
	// 获取该班次的草案
	if workingDraft == nil || workingDraft.Shifts == nil {
		return nil
	}

	shiftScheduleDraft, exists := workingDraft.Shifts[shift.ID]
	if !exists || shiftScheduleDraft == nil {
		return nil
	}

	// 转换为ShiftScheduleDraft格式
	draft := &d_model.ShiftScheduleDraft{
		Schedule:     make(map[string][]string),
		UpdatedStaff: make(map[string]bool),
	}
	for date, dayShift := range shiftScheduleDraft.Days {
		if dayShift != nil && len(dayShift.StaffIDs) > 0 {
			draft.Schedule[date] = dayShift.StaffIDs
			for _, id := range dayShift.StaffIDs {
				draft.UpdatedStaff[id] = true
			}
		}
	}

	// 使用分批校验
	batchCtx, err := e.validateInBatches(
		ctx, shift, nil, staffList, draft, rules, occupiedSlots, allShifts,
		DefaultBatchConfig(),
	)
	if err != nil {
		e.logger.Error("Revalidation failed", "error", err)
		return nil
	}

	return batchCtx.AllViolations
}
