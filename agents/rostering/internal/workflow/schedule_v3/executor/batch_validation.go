package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// 分批校验执行器
// 将大量人员分成多个小批次（每批5人）进行LLM校验
// 有效控制单次LLM调用的任务粒度，避免超出处理能力
// ============================================================

// validateInBatches 分批校验入口
// 将待校验人员按批次大小分组，逐批调用LLM校验
// 所有批次完成后返回合并的校验结果
func (e *ProgressiveTaskExecutor) validateInBatches(
	ctx context.Context,
	shift *d_model.Shift,
	targetDates []string,
	staffList []*d_model.Employee,
	draft *d_model.ShiftScheduleDraft,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
	config *BatchConfig,
) (*BatchValidationContext, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch config: %w", err)
	}

	// 收集需要校验的人员（在draft中有排班的人员）
	staffToValidate := e.collectStaffToValidate(draft, staffList)
	if len(staffToValidate) == 0 {
		e.logger.Info("No staff to validate in batch mode",
			"shiftID", shift.ID,
			"shiftName", shift.Name)
		return NewBatchValidationContext(shift.ID, shift.Name, 0, 0), nil
	}

	// 分组
	batches := e.splitStaffIntoBatches(staffToValidate, config.BatchSize)
	totalBatches := len(batches)

	e.logger.Info("Starting batch validation",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"totalStaff", len(staffToValidate),
		"batchSize", config.BatchSize,
		"totalBatches", totalBatches)

	// 创建上下文
	batchCtx := NewBatchValidationContext(shift.ID, shift.Name, len(staffToValidate), totalBatches)
	startTime := time.Now()

	// 逐批校验
	for i, batch := range batches {
		batchCtx.CurrentBatch = i + 1

		// 发送进度通知
		if config.EnableProgressNotify {
			e.notifyBatchValidationProgress(batchCtx, batch, "processing")
		}

		// 执行单批校验
		result := e.validateSingleBatch(ctx, shift, targetDates, batch, draft, rules, occupiedSlots, allShifts, config)
		batchCtx.AddBatchResult(result)

		e.logger.Info("Batch validation completed",
			"shiftID", shift.ID,
			"batchIndex", i+1,
			"totalBatches", totalBatches,
			"batchStaffCount", len(batch),
			"passed", result.Passed,
			"violationCount", len(result.Violations),
			"executionTime", result.ExecutionTime)

		// 检查是否需要中止
		if !result.Passed && !config.ContinueOnError {
			e.logger.Warn("Batch validation failed, stopping due to continueOnError=false",
				"shiftID", shift.ID,
				"batchIndex", i+1)
			break
		}
	}

	batchCtx.TotalTime = time.Since(startTime).Seconds()

	// 发送完成通知
	if config.EnableProgressNotify {
		status := "completed"
		if !batchCtx.AllPassed {
			status = "failed"
		}
		e.notifyBatchValidationProgress(batchCtx, nil, status)
	}

	e.logger.Info("Batch validation finished",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"totalBatches", totalBatches,
		"allPassed", batchCtx.AllPassed,
		"totalViolations", len(batchCtx.AllViolations),
		"totalTime", batchCtx.TotalTime)

	return batchCtx, nil
}

// validateSingleBatch 校验单个批次
// 构建针对少量人员的精简Prompt，调用LLM进行规则校验
func (e *ProgressiveTaskExecutor) validateSingleBatch(
	ctx context.Context,
	shift *d_model.Shift,
	targetDates []string,
	batchStaff []*d_model.Employee,
	draft *d_model.ShiftScheduleDraft,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
	config *BatchConfig,
) *BatchValidationResult {
	startTime := time.Now()
	result := &BatchValidationResult{
		StaffIDs:   make([]string, 0, len(batchStaff)),
		StaffNames: make([]string, 0, len(batchStaff)),
		Passed:     true,
		Violations: make([]*d_model.ValidationIssue, 0),
	}

	// 收集人员ID和姓名
	for _, staff := range batchStaff {
		result.StaffIDs = append(result.StaffIDs, staff.ID)
		result.StaffNames = append(result.StaffNames, staff.Name)
	}

	// 构建该批次人员的子草案（只包含这些人的排班）
	batchDraft := e.buildBatchDraft(draft, result.StaffIDs)

	// 构建校验Prompt
	sysPrompt := e.buildBatchValidationSystemPrompt()
	userPrompt := e.buildBatchValidationUserPrompt(shift, targetDates, batchStaff, batchDraft, rules, occupiedSlots, allShifts)

	// 调用LLM
	var lastErr error
	for retry := 0; retry <= config.MaxRetryPerBatch; retry++ {
		if retry > 0 {
			result.RetryCount = retry
		}

		llmCallStart := time.Now()
		resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
		llmCallDuration := time.Since(llmCallStart)

		// 记录到调试文件
		e.logLLMDebug(fmt.Sprintf("batch_validation_%d", retry), logging.LLMCallBatchValidation, shift.Name, "", sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

		if err != nil {
			lastErr = err
			continue
		}

		// 解析校验结果
		violations, parseErr := e.parseBatchValidationResponse(resp.Content, shift, batchStaff)
		if parseErr != nil {
			lastErr = parseErr
			continue
		}

		result.Violations = violations
		result.Passed = len(violations) == 0
		result.ExecutionTime = time.Since(startTime).Seconds()
		return result
	}

	// 重试全部失败
	result.ErrorMessage = fmt.Sprintf("batch validation failed after %d retries: %v", config.MaxRetryPerBatch, lastErr)
	result.Passed = false
	result.ExecutionTime = time.Since(startTime).Seconds()

	e.logger.Error("Batch validation failed after all retries", "shiftID", shift.ID, "error", lastErr)

	return result
}

// collectStaffToValidate 收集需要校验的人员
// 返回在草案中有排班记录的人员列表
func (e *ProgressiveTaskExecutor) collectStaffToValidate(
	draft *d_model.ShiftScheduleDraft,
	staffList []*d_model.Employee,
) []*d_model.Employee {
	if draft == nil || draft.Schedule == nil {
		return nil
	}

	// 收集草案中的所有人员ID
	staffIDSet := make(map[string]bool)
	for _, staffIDs := range draft.Schedule {
		for _, id := range staffIDs {
			staffIDSet[id] = true
		}
	}

	// 从staffList中筛选
	result := make([]*d_model.Employee, 0)
	for _, staff := range staffList {
		if staffIDSet[staff.ID] {
			result = append(result, staff)
		}
	}

	return result
}

// splitStaffIntoBatches 将人员列表按批次大小分组
func (e *ProgressiveTaskExecutor) splitStaffIntoBatches(
	staffList []*d_model.Employee,
	batchSize int,
) [][]*d_model.Employee {
	if len(staffList) == 0 {
		return nil
	}

	batches := make([][]*d_model.Employee, 0)
	for i := 0; i < len(staffList); i += batchSize {
		end := i + batchSize
		if end > len(staffList) {
			end = len(staffList)
		}
		batches = append(batches, staffList[i:end])
	}

	return batches
}

// buildBatchDraft 构建批次子草案（只包含指定人员的排班）
func (e *ProgressiveTaskExecutor) buildBatchDraft(
	draft *d_model.ShiftScheduleDraft,
	staffIDs []string,
) *d_model.ShiftScheduleDraft {
	if draft == nil || draft.Schedule == nil {
		return d_model.NewShiftScheduleDraft()
	}

	staffIDSet := make(map[string]bool)
	for _, id := range staffIDs {
		staffIDSet[id] = true
	}

	batchDraft := d_model.NewShiftScheduleDraft()
	for date, dateStaffIDs := range draft.Schedule {
		filteredIDs := make([]string, 0)
		for _, id := range dateStaffIDs {
			if staffIDSet[id] {
				filteredIDs = append(filteredIDs, id)
			}
		}
		if len(filteredIDs) > 0 {
			batchDraft.Schedule[date] = filteredIDs
		}
	}

	return batchDraft
}

// buildBatchValidationSystemPrompt 构建批次校验的系统提示词
func (e *ProgressiveTaskExecutor) buildBatchValidationSystemPrompt() string {
	return `你是一个专业的排班规则校验专家。你的任务是检查给定员工的排班是否符合所有规则。

## 校验规则
1. 严格按照提供的规则列表进行检查
2. 每个违规问题需要明确指出：
   - 违反的规则名称
   - 涉及的员工
   - 涉及的日期
   - 具体违规描述
3. 如果没有发现违规，返回空的violations数组

## 输出格式（JSON）
{
  "passed": true/false,
  "violations": [
    {
      "ruleId": "规则ID（如rule_1）",
      "ruleName": "规则名称",
      "severity": "critical/error/warning",
      "staffId": "人员ID（如staff_1）",
      "staffName": "员工姓名",
      "dates": ["2026-01-01"],
      "description": "具体违规描述"
    }
  ],
  "summary": "校验结果摘要"
}`
}

// buildBatchValidationUserPrompt 构建批次校验的用户提示词
func (e *ProgressiveTaskExecutor) buildBatchValidationUserPrompt(
	shift *d_model.Shift,
	targetDates []string,
	batchStaff []*d_model.Employee,
	batchDraft *d_model.ShiftScheduleDraft,
	rules []*d_model.Rule,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	allShifts []*d_model.Shift,
) string {
	var prompt strings.Builder

	prompt.WriteString("## 校验任务\n\n")
	prompt.WriteString(fmt.Sprintf("请校验以下 **%d 名员工** 在班次【%s】的排班是否符合规则。\n\n", len(batchStaff), shift.Name))

	// 班次信息
	prompt.WriteString("## 班次信息\n\n")
	prompt.WriteString(fmt.Sprintf("- 班次名称：%s\n", shift.Name))
	prompt.WriteString(fmt.Sprintf("- 班次时间：%s - %s\n", shift.StartTime, shift.EndTime))
	prompt.WriteString(fmt.Sprintf("- 排班日期范围：%s 至 %s\n\n", targetDates[0], targetDates[len(targetDates)-1]))

	// 待校验员工信息 - 使用 姓名(staff_N) 格式，代码层面禁止UUID泄漏
	prompt.WriteString("## 待校验员工\n\n")
	for i, staff := range batchStaff {
		shortID := e.maskStaffID(staff.ID, i+1)

		// 获取该员工的排班日期
		scheduledDates := make([]string, 0)
		if batchDraft != nil && batchDraft.Schedule != nil {
			for date, staffIDs := range batchDraft.Schedule {
				for _, id := range staffIDs {
					if id == staff.ID {
						scheduledDates = append(scheduledDates, date)
						break
					}
				}
			}
		}
		datesStr := strings.Join(scheduledDates, ", ")
		if datesStr == "" {
			datesStr = "无排班"
		}

		prompt.WriteString(fmt.Sprintf("%d. %s(%s) - 排班日期：%s\n", i+1, staff.Name, shortID, datesStr))
	}
	prompt.WriteString("\n")

	// 相关规则
	prompt.WriteString("## 需要校验的规则\n\n")
	if len(rules) == 0 {
		prompt.WriteString("无特定规则\n\n")
	} else {
		for i, rule := range rules {
			if rule == nil || !rule.IsActive {
				continue
			}
			ruleShortID := e.maskRuleID(rule.ID, i+1)
			prompt.WriteString(fmt.Sprintf("### %d. %s\n", i+1, rule.Name))
			prompt.WriteString(fmt.Sprintf("- 规则ID：%s\n", ruleShortID))
			prompt.WriteString(fmt.Sprintf("- 规则类型：%s\n", rule.RuleType))
			prompt.WriteString(fmt.Sprintf("- 规则描述：%s\n", rule.Description))
			if rule.RuleData != "" {
				prompt.WriteString(fmt.Sprintf("- 规则内容：%s\n", rule.RuleData))
			}
			prompt.WriteString("\n")
		}
	}

	// 其他班次排班情况（用于检测时间冲突）
	prompt.WriteString("## 员工在其他班次的排班（用于冲突检测）\n\n")
	hasOtherSchedule := false
	for _, staff := range batchStaff {
		otherSlots := d_model.GetStaffOtherShiftSlots(*occupiedSlots, staff.ID, shift.ID)
		if len(otherSlots) > 0 {
			slotDescs := make([]string, 0, len(otherSlots))
			for _, slot := range otherSlots {
				shiftName := slot.ShiftName
				if shiftName == "" {
					shiftName = "其他班次"
					for _, s := range allShifts {
						if s.ID == slot.ShiftID {
							shiftName = s.Name
							break
						}
					}
				}
				slotDescs = append(slotDescs, fmt.Sprintf("%s(%s)", slot.Date, shiftName))
			}
			prompt.WriteString(fmt.Sprintf("- %s：%s\n", staff.Name, strings.Join(slotDescs, ", ")))
			hasOtherSchedule = true
		}
	}
	if !hasOtherSchedule {
		prompt.WriteString("这些员工暂无其他班次排班记录\n")
	}
	prompt.WriteString("\n")

	prompt.WriteString("## 请输出校验结果（JSON格式）\n")

	return prompt.String()
}

// parseBatchValidationResponse 解析批次校验响应
func (e *ProgressiveTaskExecutor) parseBatchValidationResponse(
	response string,
	shift *d_model.Shift,
	batchStaff []*d_model.Employee,
) ([]*d_model.ValidationIssue, error) {
	// 提取JSON部分
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// 如果没有找到JSON，检查是否表示通过
		if strings.Contains(strings.ToLower(response), "passed") ||
			strings.Contains(response, "无违规") ||
			strings.Contains(response, "全部通过") {
			return nil, nil // 校验通过
		}
		return nil, fmt.Errorf("failed to extract JSON from response")
	}

	// 解析JSON
	var result struct {
		Passed     bool `json:"passed"`
		Violations []struct {
			RuleID      string   `json:"ruleId"`
			RuleName    string   `json:"ruleName"`
			Severity    string   `json:"severity"`
			StaffID     string   `json:"staffId"`
			StaffName   string   `json:"staffName"`
			Dates       []string `json:"dates"`
			Description string   `json:"description"`
		} `json:"violations"`
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse validation response: %w", err)
	}

	if result.Passed || len(result.Violations) == 0 {
		return nil, nil
	}

	// 转换为标准格式，将LLM返回的shortID还原为UUID
	violations := make([]*d_model.ValidationIssue, 0, len(result.Violations))
	for _, v := range result.Violations {
		// 统一通过 resolveStaffID 将 staff_N 还原为 UUID
		resolvedStaffID := e.resolveStaffID(v.StaffID)
		// 如果 shortID 解析失败，尝试通过姓名查找
		if resolvedStaffID == v.StaffID && v.StaffName != "" {
			for _, staff := range batchStaff {
				if staff.Name == v.StaffName {
					resolvedStaffID = staff.ID
					break
				}
			}
		}

		// 统一通过 resolveRuleID 还原 ruleID
		resolvedRuleID := e.resolveRuleID(v.RuleID)

		issue := &d_model.ValidationIssue{
			RuleID:         resolvedRuleID,
			Type:           "rule_compliance",
			Severity:       v.Severity,
			Description:    fmt.Sprintf("[%s] %s: %s", v.RuleName, v.StaffName, v.Description),
			AffectedDates:  v.Dates,
			AffectedStaff:  []string{resolvedStaffID},
			AffectedShifts: []string{shift.ID},
		}
		violations = append(violations, issue)
	}

	return violations, nil
}

// notifyBatchValidationProgress 发送批次校验进度通知
func (e *ProgressiveTaskExecutor) notifyBatchValidationProgress(
	batchCtx *BatchValidationContext,
	currentBatch []*d_model.Employee,
	status string,
) {
	staffNames := make([]string, 0)
	for _, staff := range currentBatch {
		staffNames = append(staffNames, staff.Name)
	}

	var message string
	switch status {
	case "processing":
		message = fmt.Sprintf("正在校验第%d/%d批（%s）...",
			batchCtx.CurrentBatch, batchCtx.TotalBatches, strings.Join(staffNames, "、"))
	case "completed":
		if batchCtx.AllPassed {
			message = fmt.Sprintf("校验完成：%d人全部通过", batchCtx.TotalStaff)
		} else {
			message = fmt.Sprintf("校验完成：%d人中发现%d个问题", batchCtx.TotalStaff, len(batchCtx.AllViolations))
		}
	case "failed":
		message = fmt.Sprintf("校验失败：%d人中发现%d个问题", batchCtx.TotalStaff, len(batchCtx.AllViolations))
	}

	progress := &BatchProgressInfo{
		Type:              "validation",
		CurrentBatch:      batchCtx.CurrentBatch,
		TotalBatches:      batchCtx.TotalBatches,
		CurrentStaffNames: staffNames,
		TotalStaff:        batchCtx.TotalStaff,
		ProcessedStaff:    len(batchCtx.ProcessedStaff),
		Progress:          batchCtx.GetProgress(),
		Status:            status,
		Message:           message,
	}

	e.notifyBatchProgress(progress)
}

// notifyBatchProgress 发送批次进度通知（通用）
func (e *ProgressiveTaskExecutor) notifyBatchProgress(progress *BatchProgressInfo) {
	// 如果有进度回调，调用它
	// TODO: 集成到 progressCallback
}
