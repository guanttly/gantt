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

// ============================================================
// 分批排班执行器
// 将大量排班需求分成多个小批次（每批5人）进行LLM决策
// 有效控制单次LLM调用的任务粒度，避免超出处理能力
// ============================================================

// scheduleInBatches 分批排班入口
// 将当日排班需求按批次大小分组，逐批调用LLM进行人员选择
// 所有批次完成后返回合并的排班结果
func (e *ProgressiveTaskExecutor) scheduleInBatches(
	ctx context.Context,
	shift *d_model.Shift,
	targetDate string,
	requiredCount int,
	candidates []*d_model.Employee,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
	config *BatchConfig,
) (*BatchSchedulingContext, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch config: %w", err)
	}

	// 过滤已被占用的候选人（当日已有排班的人员）
	availableCandidates := e.filterAvailableCandidates(candidates, targetDate, shift.ID, occupiedSlots)

	e.logger.Info("Starting batch scheduling",
		"shiftID", shift.ID,
		"shiftName", shift.Name,
		"targetDate", targetDate,
		"requiredCount", requiredCount,
		"totalCandidates", len(candidates),
		"availableCandidates", len(availableCandidates),
		"batchSize", config.BatchSize)

	// 创建上下文
	batchCtx := NewBatchSchedulingContext(shift.ID, shift.Name, targetDate, requiredCount, availableCandidates)
	startTime := time.Now()

	// 计算实际批次数
	totalBatches := (requiredCount + config.BatchSize - 1) / config.BatchSize
	batchCtx.TotalBatches = totalBatches

	// 逐批排班
	for batchIndex := 0; batchIndex < totalBatches && len(batchCtx.ScheduledStaff) < requiredCount; batchIndex++ {
		batchCtx.CurrentBatch = batchIndex + 1

		// 计算本批需要安排的人数
		remaining := requiredCount - len(batchCtx.ScheduledStaff)
		batchSize := batchMin(config.BatchSize, remaining)

		// 发送进度通知
		if config.EnableProgressNotify {
			e.notifyBatchSchedulingProgress(batchCtx, "processing")
		}

		// 检查是否还有足够的候选人
		if len(batchCtx.RemainingCandidates) < batchSize {
			e.logger.Warn("Not enough candidates for batch scheduling",
				"shiftID", shift.ID,
				"targetDate", targetDate,
				"batchIndex", batchIndex+1,
				"requiredForBatch", batchSize,
				"availableCandidates", len(batchCtx.RemainingCandidates))
			batchSize = len(batchCtx.RemainingCandidates)
			if batchSize == 0 {
				break
			}
		}

		// 执行单批排班
		result := e.scheduleSingleBatch(
			ctx, shift, targetDate, batchSize, batchCtx,
			rules, personalNeeds, fixedAssignments, config,
		)
		batchCtx.AddBatchResult(result)

		// 更新剩余候选人（移除已安排的）
		batchCtx.RemainingCandidates = e.removeScheduledFromCandidates(
			batchCtx.RemainingCandidates,
			result.ScheduledIDs,
		)

		// 更新占位信息
		for _, staffID := range result.ScheduledIDs {
			*occupiedSlots = d_model.AddOccupiedSlotIfNotExists(*occupiedSlots, d_model.StaffOccupiedSlot{
				StaffID:   staffID,
				Date:      targetDate,
				ShiftID:   shift.ID,
				ShiftName: shift.Name,
				Source:    "draft",
			})
		}

		e.logger.Info("Batch scheduling completed",
			"shiftID", shift.ID,
			"targetDate", targetDate,
			"batchIndex", batchIndex+1,
			"totalBatches", totalBatches,
			"requestedCount", batchSize,
			"scheduledCount", len(result.ScheduledIDs),
			"totalScheduled", len(batchCtx.ScheduledStaff),
			"executionTime", result.ExecutionTime)

		// 检查是否需要中止（单批失败）
		if result.ErrorMessage != "" && !config.ContinueOnError {
			batchCtx.ErrorMessage = result.ErrorMessage
			break
		}
	}

	batchCtx.TotalTime = time.Since(startTime).Seconds()
	batchCtx.Completed = len(batchCtx.ScheduledStaff) >= requiredCount

	// 发送完成通知
	if config.EnableProgressNotify {
		status := "completed"
		if !batchCtx.Completed {
			status = "partial"
		}
		e.notifyBatchSchedulingProgress(batchCtx, status)
	}

	e.logger.Info("Batch scheduling finished",
		"shiftID", shift.ID,
		"targetDate", targetDate,
		"required", requiredCount,
		"scheduled", len(batchCtx.ScheduledStaff),
		"completed", batchCtx.Completed,
		"totalBatches", len(batchCtx.BatchResults),
		"totalTime", batchCtx.TotalTime)

	return batchCtx, nil
}

// scheduleSingleBatch 执行单批次排班
// 构建针对少量人员的精简Prompt，调用LLM进行人员选择
func (e *ProgressiveTaskExecutor) scheduleSingleBatch(
	ctx context.Context,
	shift *d_model.Shift,
	targetDate string,
	batchSize int,
	batchCtx *BatchSchedulingContext,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
	config *BatchConfig,
) *BatchSchedulingResult {
	startTime := time.Now()
	result := &BatchSchedulingResult{
		BatchIndex:     batchCtx.CurrentBatch,
		RequestedCount: batchSize,
		ScheduledIDs:   make([]string, 0),
		ScheduledNames: make([]string, 0),
	}

	// 构建排班Prompt
	sysPrompt := e.buildBatchSchedulingSystemPrompt()
	userPrompt := e.buildBatchSchedulingUserPrompt(
		shift, targetDate, batchSize, batchCtx,
		rules, personalNeeds, fixedAssignments,
	)

	// 调用LLM（带重试）
	var lastErr error
	for retry := 0; retry <= config.MaxRetryPerBatch; retry++ {
		if retry > 0 {
			result.RetryCount = retry
		}

		llmCallStart := time.Now()
		resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
		llmCallDuration := time.Since(llmCallStart)

		// 记录到调试文件
		e.logLLMDebug(fmt.Sprintf("batch_%d_%s", batchCtx.CurrentBatch, targetDate), logging.LLMCallBatchScheduling, shift.Name, targetDate, sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

		if err != nil {
			lastErr = err
			continue
		}

		// 解析排班结果
		scheduledIDs, scheduledNames, reasoning, parseErr := e.parseBatchSchedulingResponse(
			resp.Content, batchCtx.RemainingCandidates, batchSize,
		)
		if parseErr != nil {
			lastErr = parseErr
			continue
		}

		result.ScheduledIDs = scheduledIDs
		result.ScheduledNames = scheduledNames
		result.Reasoning = reasoning
		result.ExecutionTime = time.Since(startTime).Seconds()

		// 验证返回的人员数量
		if len(scheduledIDs) < batchSize && len(batchCtx.RemainingCandidates) >= batchSize {
			e.logger.Warn("Batch scheduling returned fewer staff than requested",
				"shiftID", shift.ID,
				"targetDate", targetDate,
				"requested", batchSize,
				"returned", len(scheduledIDs))
		}

		return result
	}

	// 重试全部失败
	result.ErrorMessage = fmt.Sprintf("batch scheduling failed after %d retries: %v", config.MaxRetryPerBatch, lastErr)
	result.ExecutionTime = time.Since(startTime).Seconds()

	e.logger.Error("Batch scheduling failed after all retries",
		"shiftID", shift.ID,
		"targetDate", targetDate,
		"batchIndex", batchCtx.CurrentBatch,
		"error", lastErr)

	return result
}

// filterAvailableCandidates 过滤可用候选人（排除当日已有时间冲突排班的人员）
// 【P0修复】不仅检查日期匹配，还要检查班次时间是否重叠
func (e *ProgressiveTaskExecutor) filterAvailableCandidates(
	candidates []*d_model.Employee,
	targetDate string,
	targetShiftID string,
	occupiedSlots *[]d_model.StaffOccupiedSlot,
) []*d_model.Employee {
	// 获取目标班次信息（用于时间冲突检查）
	var targetShift *d_model.Shift
	if e.taskContext != nil && e.taskContext.Shifts != nil {
		for _, s := range e.taskContext.Shifts {
			if s != nil && s.ID == targetShiftID {
				targetShift = s
				break
			}
		}
	}

	// 构建班次ID到班次信息的映射（用于时间冲突检查）
	shiftMap := make(map[string]*d_model.Shift)
	if e.taskContext != nil && e.taskContext.Shifts != nil {
		for _, s := range e.taskContext.Shifts {
			if s != nil {
				shiftMap[s.ID] = s
			}
		}
	}

	available := make([]*d_model.Employee, 0)
	for _, candidate := range candidates {
		// 检查该人员当日是否已有排班
		slot := d_model.FindOccupiedSlot(*occupiedSlots, candidate.ID, targetDate)
		if slot == nil {
			// 当日无排班，可用
			available = append(available, candidate)
			continue
		}

		// 当日有排班，检查时间是否冲突
		// 如果目标班次信息不可用，保守起见排除该候选人
		if targetShift == nil {
			e.logger.Debug("Target shift not found, excluding candidate",
				"candidateID", candidate.ID,
				"candidateName", candidate.Name,
				"targetShiftID", targetShiftID,
				"targetDate", targetDate)
			continue
		}

		// 获取已占用班次信息
		existingShift := shiftMap[slot.ShiftID]
		if existingShift == nil {
			// 已占用班次信息不可用，保守起见排除该候选人
			e.logger.Debug("Existing shift not found, excluding candidate",
				"candidateID", candidate.ID,
				"candidateName", candidate.Name,
				"existingShiftID", slot.ShiftID,
				"targetDate", targetDate)
			continue
		}

		// 检查时间是否重叠
		if utils.CheckTimeOverlap(targetShift, existingShift) {
			// 时间重叠，不可用
			e.logger.Debug("Time conflict detected, excluding candidate",
				"candidateID", candidate.ID,
				"candidateName", candidate.Name,
				"targetShift", targetShift.Name,
				"existingShift", existingShift.Name,
				"targetDate", targetDate)
			continue
		}

		// 时间不重叠，可用（同一天可以有多个不冲突的班次）
		available = append(available, candidate)
	}
	return available
}

// removeScheduledFromCandidates 从候选人列表中移除已安排的人员
func (e *ProgressiveTaskExecutor) removeScheduledFromCandidates(
	candidates []*d_model.Employee,
	scheduledIDs []string,
) []*d_model.Employee {
	scheduledSet := make(map[string]bool)
	for _, id := range scheduledIDs {
		scheduledSet[id] = true
	}

	remaining := make([]*d_model.Employee, 0)
	for _, candidate := range candidates {
		if !scheduledSet[candidate.ID] {
			remaining = append(remaining, candidate)
		}
	}
	return remaining
}

// buildBatchSchedulingSystemPrompt 构建批次排班的系统提示词
func (e *ProgressiveTaskExecutor) buildBatchSchedulingSystemPrompt() string {
	return `你是一个专业的排班决策专家。你的任务是从候选人列表中选择指定数量的员工进行排班。

## 选择原则
1. 严格遵守所有排班规则
2. 考虑员工的个人需求（请假、偏好等）
3. 尽量实现工作负载均衡（优先选择排班较少的员工）
4. 不要选择当日已有冲突排班的员工

## 输出格式（JSON）
{
  "selectedStaff": [
    {"id": "人员ID（如staff_1）", "name": "员工姓名"}
  ],
  "reasoning": "选择理由（简要说明）",
  "warnings": ["如有需要注意的问题"]
}`
}

// buildBatchSchedulingUserPrompt 构建批次排班的用户提示词
func (e *ProgressiveTaskExecutor) buildBatchSchedulingUserPrompt(
	shift *d_model.Shift,
	targetDate string,
	batchSize int,
	batchCtx *BatchSchedulingContext,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
) string {
	var prompt strings.Builder

	prompt.WriteString("## 排班任务\n\n")
	prompt.WriteString(fmt.Sprintf("请从候选人中选择 **%d 名员工** 安排到班次【%s】的 **%s** 排班。\n\n",
		batchSize, shift.Name, targetDate))

	// 排班进度
	prompt.WriteString("## 当前进度\n\n")
	prompt.WriteString(fmt.Sprintf("- 总需求：%d人\n", batchCtx.TotalRequired))
	prompt.WriteString(fmt.Sprintf("- 已安排：%d人\n", len(batchCtx.ScheduledStaff)))
	prompt.WriteString(fmt.Sprintf("- 本批需要：%d人\n", batchSize))
	prompt.WriteString(fmt.Sprintf("- 当前批次：第%d/%d批\n\n", batchCtx.CurrentBatch, batchCtx.TotalBatches))

	// 已安排人员（避免重复）
	if len(batchCtx.ScheduledStaff) > 0 {
		prompt.WriteString("## 已安排人员（请勿重复选择）\n\n")
		// 获取已安排人员姓名
		scheduledNames := make([]string, 0)
		for _, result := range batchCtx.BatchResults {
			scheduledNames = append(scheduledNames, result.ScheduledNames...)
		}
		prompt.WriteString(strings.Join(scheduledNames, "、"))
		prompt.WriteString("\n\n")
	}

	// 固定排班（如有）
	if fixedIDs, ok := fixedAssignments[targetDate]; ok && len(fixedIDs) > 0 {
		prompt.WriteString("## 固定排班人员（已确定，无需选择）\n\n")
		prompt.WriteString(fmt.Sprintf("共%d人已固定排班\n\n", len(fixedIDs)))
	}

	// 候选人列表 - 使用 姓名(staff_N) 格式，禁止UUID泄漏
	prompt.WriteString("## 候选人列表\n\n")
	prompt.WriteString("请从以下候选人中选择：\n\n")

	// 按某种规则排序候选人（可以按排班数量、优先级等）
	sortedCandidates := make([]*d_model.Employee, len(batchCtx.RemainingCandidates))
	copy(sortedCandidates, batchCtx.RemainingCandidates)
	// TODO: 可以添加排序逻辑（如按当月排班天数升序）

	candidateItems := make([]string, 0, len(sortedCandidates))
	for _, candidate := range sortedCandidates {
		shortID := e.maskStaffID(candidate.ID, len(candidateItems)+1)

		// 检查个人需求
		remark := ""
		if needs, ok := personalNeeds[candidate.ID]; ok {
			for _, need := range needs {
				// 检查是否影响当日
				if contains(need.TargetDates, targetDate) {
					remark = fmt.Sprintf("（%s: %s）", need.NeedType, need.Description)
					break
				}
			}
		}

		position := candidate.Position
		if position != "" {
			position = fmt.Sprintf("[%s]", position)
		}

		candidateItems = append(candidateItems, fmt.Sprintf("%s(%s)%s%s", candidate.Name, shortID, position, remark))
	}

	prompt.WriteString(fmt.Sprintf("可选（%d人）：%s\n\n", len(candidateItems), strings.Join(candidateItems, ", ")))
	prompt.WriteString("**注意**：只能从上述候选人员中选择，禁止选择列表之外的任何人！\n\n")

	// 排班规则
	prompt.WriteString("## 需要遵守的规则\n\n")
	if len(rules) == 0 {
		prompt.WriteString("无特定规则限制\n\n")
	} else {
		ruleCount := 0
		for _, rule := range rules {
			if rule == nil || !rule.IsActive {
				continue
			}
			ruleCount++
			if ruleCount > 5 {
				prompt.WriteString(fmt.Sprintf("... 还有%d条规则\n", len(rules)-5))
				break
			}
			prompt.WriteString(fmt.Sprintf("- **%s**：%s\n", rule.Name, rule.Description))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString(fmt.Sprintf("## 请输出选择的%d名员工（JSON格式）\n", batchSize))

	return prompt.String()
}

// parseBatchSchedulingResponse 解析批次排班响应
func (e *ProgressiveTaskExecutor) parseBatchSchedulingResponse(
	response string,
	candidates []*d_model.Employee,
	expectedCount int,
) ([]string, []string, string, error) {
	// 提取JSON部分
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, nil, "", fmt.Errorf("failed to extract JSON from response")
	}

	// 解析JSON
	var result struct {
		SelectedStaff []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"selectedStaff"`
		Reasoning string   `json:"reasoning"`
		Warnings  []string `json:"warnings"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse scheduling response: %w", err)
	}

	// 收集有效的人员ID
	scheduledIDs := make([]string, 0)
	scheduledNames := make([]string, 0)

	for _, selected := range result.SelectedStaff {
		// 统一通过 resolveStaffFromLLM 将 shortID/姓名 定位到真实人员
		found := e.resolveStaffFromLLM(selected.ID, selected.Name, candidates)

		if found != nil {
			scheduledIDs = append(scheduledIDs, found.ID)
			scheduledNames = append(scheduledNames, found.Name)
		} else {
			e.logger.Warn("Selected staff not found in candidates",
				"selectedID", selected.ID,
				"selectedName", selected.Name)
		}
	}

	return scheduledIDs, scheduledNames, result.Reasoning, nil
}

// notifyBatchSchedulingProgress 发送批次排班进度通知
func (e *ProgressiveTaskExecutor) notifyBatchSchedulingProgress(
	batchCtx *BatchSchedulingContext,
	status string,
) {
	var message string
	switch status {
	case "processing":
		message = fmt.Sprintf("%s: 正在安排第%d/%d批（共需%d人，已安排%d人）...",
			batchCtx.TargetDate, batchCtx.CurrentBatch, batchCtx.TotalBatches,
			batchCtx.TotalRequired, len(batchCtx.ScheduledStaff))
	case "completed":
		message = fmt.Sprintf("%s: 排班完成，共安排%d人",
			batchCtx.TargetDate, len(batchCtx.ScheduledStaff))
	case "partial":
		message = fmt.Sprintf("%s: 排班未完成，需要%d人仅安排%d人",
			batchCtx.TargetDate, batchCtx.TotalRequired, len(batchCtx.ScheduledStaff))
	}

	// 获取最近安排的人员姓名
	currentStaffNames := make([]string, 0)
	if len(batchCtx.BatchResults) > 0 {
		lastResult := batchCtx.BatchResults[len(batchCtx.BatchResults)-1]
		currentStaffNames = lastResult.ScheduledNames
	}

	progress := &BatchProgressInfo{
		Type:              "scheduling",
		CurrentBatch:      batchCtx.CurrentBatch,
		TotalBatches:      batchCtx.TotalBatches,
		CurrentStaffNames: currentStaffNames,
		TotalStaff:        batchCtx.TotalRequired,
		ProcessedStaff:    len(batchCtx.ScheduledStaff),
		Progress:          batchCtx.GetProgress(),
		Status:            status,
		Message:           message,
	}

	e.notifyBatchProgress(progress)
}

// ============================================================
// 辅助函数
// ============================================================

// sortCandidatesByWorkload 按工作负载排序候选人（排班少的优先）
func sortCandidatesByWorkload(candidates []*d_model.Employee, occupiedSlots *[]d_model.StaffOccupiedSlot) []*d_model.Employee {
	result := make([]*d_model.Employee, len(candidates))
	copy(result, candidates)

	sort.Slice(result, func(i, j int) bool {
		countI := d_model.CountOccupiedByStaff(*occupiedSlots, result[i].ID)
		countJ := d_model.CountOccupiedByStaff(*occupiedSlots, result[j].ID)
		return countI < countJ
	})

	return result
}

// batchMin 返回两个整数中的较小值
func batchMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
