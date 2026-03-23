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

// analyzeDayRules 分析当日规则（并行调用LLM1+LLM2+LLM3）
func (e *ProgressiveTaskExecutor) analyzeDayRules(
	ctx context.Context,
	targetShift *d_model.Shift,
	targetDate string,
	allRules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed,
	candidateStaff []*d_model.Employee,
	relatedShiftsSchedule map[string]map[string][]string,
	shiftFixedAssignments map[string][]string, // 当前班次本周的固定排班
	currentDraft *d_model.ShiftScheduleDraft, // 【新增】当前班次已完成的动态排班
) (*DayRuleAnalysis, error) {
	startTime := time.Now()

	// 使用 channel 收集结果
	type personalNeedsResult struct {
		analysis *PersonalNeedsAnalysis
		err      error
	}
	type rulesResult struct {
		analysis *RulesAnalysis
		err      error
	}
	type ruleConflictResult struct {
		analysis *RuleConflictAnalysis
		err      error
	}

	personalNeedsChan := make(chan personalNeedsResult, 1)
	rulesChan := make(chan rulesResult, 1)
	ruleConflictChan := make(chan ruleConflictResult, 1)

	// 并行调用 LLM1: 人员过滤
	go func() {
		analysis, err := e.filterPersonalNeeds(ctx, targetDate, personalNeeds, candidateStaff)
		personalNeedsChan <- personalNeedsResult{analysis: analysis, err: err}
	}()

	// 并行调用 LLM2: 规则过滤
	go func() {
		analysis, err := e.filterRelevantRules(ctx, targetShift, allRules)
		rulesChan <- rulesResult{analysis: analysis, err: err}
	}()

	// 并行调用 LLM3: 规则冲突人员过滤
	go func() {
		analysis, err := e.filterRuleConflictStaff(ctx, targetShift, targetDate, candidateStaff, shiftFixedAssignments, currentDraft, allRules, relatedShiftsSchedule)
		ruleConflictChan <- ruleConflictResult{analysis: analysis, err: err}
	}()

	// 等待三个结果
	personalNeedsRes := <-personalNeedsChan
	rulesRes := <-rulesChan
	ruleConflictRes := <-ruleConflictChan
	_ = startTime // 避免未使用警告

	// 合并结果到 DayRuleAnalysis
	analysis := &DayRuleAnalysis{
		RelevantRules:         []RelevantRule{},
		RelevantPersonalNeeds: []RelevantPersonalNeed{},
		Summary:               "",
	}

	// 处理规则过滤结果
	if rulesRes.err != nil {
		e.logger.Warn("LLM2 rule filter failed, using all rules", "error", rulesRes.err)
	} else if rulesRes.analysis != nil {
		for _, rule := range rulesRes.analysis.RelevantRules {
			analysis.RelevantRules = append(analysis.RelevantRules, RelevantRule{
				RuleName:   rule.RuleName,
				Reason:     "通过LLM2规则过滤",
				Constraint: rule.Constraint,
			})
		}
	}

	// 处理人员过滤结果（LLM1）
	if personalNeedsRes.err != nil {
		e.logger.Warn("LLM1 staff filter failed", "error", personalNeedsRes.err)
	} else if personalNeedsRes.analysis != nil {
		for _, staff := range personalNeedsRes.analysis.UnavailableStaff {
			analysis.RelevantPersonalNeeds = append(analysis.RelevantPersonalNeeds, RelevantPersonalNeed{
				StaffName:   staff.StaffName,
				StaffID:     staff.StaffID,
				NeedType:    "不可用",
				Description: staff.Reason,
			})
		}
	}

	// 处理规则冲突人员结果（LLM3）
	if ruleConflictRes.err != nil {
		e.logger.Warn("LLM3 rule conflict filter failed", "error", ruleConflictRes.err)
	} else if ruleConflictRes.analysis != nil && len(ruleConflictRes.analysis.ConflictStaff) > 0 {
		for _, staff := range ruleConflictRes.analysis.ConflictStaff {
			// 检查是否已经在不可用列表中（避免重复）
			alreadyExists := false
			for _, existing := range analysis.RelevantPersonalNeeds {
				if existing.StaffName == staff.StaffName {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				analysis.RelevantPersonalNeeds = append(analysis.RelevantPersonalNeeds, RelevantPersonalNeed{
					StaffName:   staff.StaffName,
					StaffID:     staff.StaffID,
					NeedType:    "规则冲突",
					Description: fmt.Sprintf("%s: %s", staff.ConflictRule, staff.Reason),
				})
			}
		}
	}

	// 构建简洁的总结
	var summaryParts []string
	if len(analysis.RelevantRules) > 0 {
		ruleNames := make([]string, 0, len(analysis.RelevantRules))
		for _, r := range analysis.RelevantRules {
			ruleNames = append(ruleNames, r.RuleName)
		}
		summaryParts = append(summaryParts, fmt.Sprintf("需遵守规则：%s", strings.Join(ruleNames, "、")))
	}
	if len(analysis.RelevantPersonalNeeds) > 0 {
		staffNames := make([]string, 0, len(analysis.RelevantPersonalNeeds))
		for _, s := range analysis.RelevantPersonalNeeds {
			staffNames = append(staffNames, s.StaffName)
		}
		summaryParts = append(summaryParts, fmt.Sprintf("不可用人员：%s", strings.Join(staffNames, "、")))
	}
	if len(summaryParts) > 0 {
		analysis.Summary = strings.Join(summaryParts, "；")
	} else {
		analysis.Summary = "无特殊约束"
	}

	return analysis, nil
}

// ============================================================
// LLM1: 人员可用性过滤（专门判断当日不可用人员）
// ============================================================

// filterPersonalNeeds 使用LLM过滤当日不可用的人员
func (e *ProgressiveTaskExecutor) filterPersonalNeeds(
	ctx context.Context,
	targetDate string,
	personalNeeds map[string][]*d_model.PersonalNeed,
	candidateStaff []*d_model.Employee,
) (*PersonalNeedsAnalysis, error) {
	sysPrompt := e.buildPersonalNeedsFilterSystemPrompt()
	userPrompt := e.buildPersonalNeedsFilterUserPrompt(targetDate, personalNeeds, candidateStaff)

	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录LLM调试日志
	e.logLLMDebug("personal_needs_filter", logging.LLMCallPersonalNeedsFilter, "", targetDate, sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("LLM1 staff filter call failed", "date", targetDate, "error", err)
		return nil, fmt.Errorf("人员过滤LLM调用失败: %w", err)
	}

	// 解析响应
	analysis, err := e.parsePersonalNeedsResponse(resp.Content)
	if err != nil {
		e.logger.Warn("LLM1 response parse failed", "date", targetDate, "error", err)
		return nil, nil
	}

	return analysis, nil
}

// buildPersonalNeedsFilterSystemPrompt LLM1系统提示词
func (e *ProgressiveTaskExecutor) buildPersonalNeedsFilterSystemPrompt() string {
	return `你是人员可用性判断专家。

任务：判断哪些人在指定日期不可用。

输出JSON格式：
{
  "unavailableStaff": [
    {"staffName": "姓名", "staffId": "ID", "reason": "不可用原因"}
  ]
}

规则：
1. 只输出在【指定日期】确实不可用的人
2. 请假日期必须包含指定日期才算不可用
3. 不要输出日期不匹配的人员
4. 如果没有人不可用，输出空数组`
}

// buildPersonalNeedsFilterUserPrompt LLM1用户提示词
func (e *ProgressiveTaskExecutor) buildPersonalNeedsFilterUserPrompt(
	targetDate string,
	personalNeeds map[string][]*d_model.PersonalNeed,
	candidateStaff []*d_model.Employee,
) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("# 判断日期：%s\n\n", formatDateWithWeekday(targetDate)))

	prompt.WriteString("# 候选人员\n\n")
	staffNames := make([]string, 0, len(candidateStaff))
	for _, staff := range candidateStaff {
		staffNames = append(staffNames, staff.Name)
	}
	prompt.WriteString(fmt.Sprintf("共%d人：%s\n\n", len(candidateStaff), strings.Join(staffNames, ", ")))

	prompt.WriteString("# 个人需求（所有日期）\n\n")
	hasNeeds := false
	for staffID, needs := range personalNeeds {
		if len(needs) > 0 {
			hasNeeds = true
			for _, need := range needs {
				staffName := need.StaffName
				if staffName == "" && e.taskContext != nil && e.taskContext.StaffIDToName != nil {
					staffName = e.taskContext.StaffIDToName[staffID]
				}
				if staffName == "" {
					if e.taskContext != nil {
						staffName = e.taskContext.GetStaffName(staffID) // 禁止UUID泄漏
					} else {
						staffName = staffID
					}
				}
				datesInfo := "整个周期"
				if len(need.TargetDates) > 0 {
					if len(need.TargetDates) <= 3 {
						datesInfo = strings.Join(need.TargetDates, ", ")
					} else {
						datesInfo = fmt.Sprintf("%s 等%d天", strings.Join(need.TargetDates[:3], ", "), len(need.TargetDates))
					}
				}
				prompt.WriteString(fmt.Sprintf("- %s: %s（%s）\n", staffName, need.NeedType, datesInfo))
			}
		}
	}
	if !hasNeeds {
		prompt.WriteString("无\n")
	}

	prompt.WriteString(fmt.Sprintf("\n# 任务\n\n请判断在【%s】这天，哪些候选人员不可用。\n", formatDateWithWeekday(targetDate)))

	return prompt.String()
}

// parsePersonalNeedsResponse 解析人员过滤响应
func (e *ProgressiveTaskExecutor) parsePersonalNeedsResponse(content string) (*PersonalNeedsAnalysis, error) {
	jsonContent := extractJSON(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("无法提取JSON内容")
	}

	var analysis PersonalNeedsAnalysis
	if err := json.Unmarshal([]byte(jsonContent), &analysis); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return &analysis, nil
}

// ============================================================
// LLM2: 规则过滤（专门判断哪些规则约束当前班次）
// ============================================================

// filterRelevantRules 使用LLM过滤与当前班次相关的规则
func (e *ProgressiveTaskExecutor) filterRelevantRules(
	ctx context.Context,
	targetShift *d_model.Shift,
	allRules []*d_model.Rule,
) (*RulesAnalysis, error) {
	sysPrompt := e.buildRulesFilterSystemPrompt()
	userPrompt := e.buildRulesFilterUserPrompt(targetShift, allRules)

	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录LLM调试日志
	e.logLLMDebug("rules_filter", logging.LLMCallRulesFilter, targetShift.Name, "", sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("LLM2 rule filter call failed", "shiftID", targetShift.ID, "error", err)
		return nil, fmt.Errorf("规则过滤LLM调用失败: %w", err)
	}

	// 解析响应
	analysis, err := e.parseRulesFilterResponse(resp.Content)
	if err != nil {
		e.logger.Warn("LLM2 response parse failed", "shiftID", targetShift.ID, "error", err)
		return nil, nil
	}

	return analysis, nil
}

// buildRulesFilterSystemPrompt LLM2系统提示词
func (e *ProgressiveTaskExecutor) buildRulesFilterSystemPrompt() string {
	return `你是规则分析专家。

任务：判断哪些规则直接约束指定班次的排班。

【核心原则】看规则的"主语"：
- 规则格式通常是："X班人员必须/禁止/限制..."
- X班就是规则的主语，规则只约束X班

【三步判断法】
1. 找出规则中「XXX人员」或「XXX班次」是谁（主语）
2. 主语 == 当前班次 → 保留
3. 主语 != 当前班次 → 排除

【重要区分】
- 规则中"提到"某班次 ≠ 规则"约束"该班次
- 作为数据源被引用 ≠ 被约束

输出JSON格式：
{
  "relevantRules": [
    {"ruleName": "规则名", "constraint": "约束描述"}
  ],
  "excludedRules": ["被排除的规则名1", "被排除的规则名2"]
}`
}

// buildRulesFilterUserPrompt LLM2用户提示词
func (e *ProgressiveTaskExecutor) buildRulesFilterUserPrompt(
	targetShift *d_model.Shift,
	allRules []*d_model.Rule,
) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("# 当前正在排班的班次：【%s】\n\n", targetShift.Name))

	prompt.WriteString("# 系统中的所有规则\n\n")
	if len(allRules) > 0 {
		for i, rule := range allRules {
			ruleContent := rule.RuleData
			if ruleContent == "" {
				ruleContent = rule.Description
			}
			prompt.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, rule.Name, ruleContent))
		}
	} else {
		prompt.WriteString("无规则\n")
	}

	prompt.WriteString(fmt.Sprintf("\n# 任务\n\n请判断哪些规则直接约束【%s】的排班。\n\n", targetShift.Name))
	prompt.WriteString("**判断示例**：\n")
	prompt.WriteString(fmt.Sprintf("- 「%s每人每周最多1次」→ 主语是【%s】→ ✅保留\n", targetShift.Name, targetShift.Name))
	prompt.WriteString(fmt.Sprintf("- 「下夜班人员=前一日%s人员」→ 主语是「下夜班」→ ❌排除\n", targetShift.Name))
	prompt.WriteString(fmt.Sprintf("  （这条规则约束的是下夜班的人选，【%s】只是被引用的数据源）\n", targetShift.Name))

	return prompt.String()
}

// parseRulesFilterResponse 解析规则过滤响应
func (e *ProgressiveTaskExecutor) parseRulesFilterResponse(content string) (*RulesAnalysis, error) {
	jsonContent := extractJSON(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("无法提取JSON内容")
	}

	var analysis RulesAnalysis
	if err := json.Unmarshal([]byte(jsonContent), &analysis); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return &analysis, nil
}

// ============================================================
// LLM3: 规则冲突人员过滤（基于固定排班+已完成排班+规则判断哪些候选人会违规）
// ============================================================

// filterRuleConflictStaff 使用LLM过滤因规则冲突而不可用的人员
func (e *ProgressiveTaskExecutor) filterRuleConflictStaff(
	ctx context.Context,
	targetShift *d_model.Shift,
	targetDate string,
	candidateStaff []*d_model.Employee,
	shiftFixedAssignments map[string][]string, // 当前班次本周的固定排班
	currentDraft *d_model.ShiftScheduleDraft, // 【新增】当前班次已完成的动态排班
	relevantRules []*d_model.Rule,
	relatedShiftsSchedule map[string]map[string][]string, // 相关班次排班信息
) (*RuleConflictAnalysis, error) {
	// 如果没有规则，跳过
	if len(relevantRules) == 0 {
		return &RuleConflictAnalysis{ConflictStaff: []RuleConflictStaffInfo{}}, nil
	}

	sysPrompt := e.buildRuleConflictFilterSystemPrompt()
	userPrompt := e.buildRuleConflictFilterUserPrompt(targetShift, targetDate, candidateStaff, shiftFixedAssignments, currentDraft, relevantRules, relatedShiftsSchedule)

	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录LLM调试日志
	e.logLLMDebug("rule_conflict_llm3", logging.LLMCallRuleConflict, targetShift.Name, targetDate, sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("LLM3 rule conflict filter call failed", "shiftID", targetShift.ID, "date", targetDate, "error", err)
		return nil, fmt.Errorf("规则冲突人员过滤LLM调用失败: %w", err)
	}

	// 解析响应
	analysis, err := e.parseRuleConflictResponse(resp.Content)
	if err != nil {
		e.logger.Warn("LLM3 response parse failed", "shiftID", targetShift.ID, "date", targetDate, "error", err)
		return nil, nil
	}

	return analysis, nil
}

// buildRuleConflictFilterSystemPrompt LLM3系统提示词
func (e *ProgressiveTaskExecutor) buildRuleConflictFilterSystemPrompt() string {
	return `你是规则冲突分析专家。判断哪些候选人会违反规则约束，需要排除。

检查以下类型的规则冲突（仅当【相关规则】中明确包含对应规则时才检查）：
1. 频次限制：**仅当规则列表中存在明确的频次限制规则（如"每人每周最多N次"）时才检查**。如果规则列表中没有频次限制规则，则所有人在频次方面均无冲突，禁止自行推断任何频率上限。
2. 来源限制（必须来自前一日某班次）：不在前一日相关班次的人不可用
3. 资源预留（当日班次人员需保留给次日）：当日相关班次人员不可用于当日

**极其重要 - 来源限制规则的适用条件**：
- 来源限制规则（如"下夜班人员必须来自前一日的上半夜班"）只有在提供了前一日排班数据时才能适用
- 如果【相关班次排班】中显示"前一日：无排班数据"，则来源限制规则不适用，不得以此排除任何候选人
- 这种情况通常发生在周一（前一日是上周日，可能不在当前排班周期内）

输出JSON：
{
  "conflictStaff": [
    {"staffName":"姓名","staffId":"staff_xx","conflictRule":"规则名","reason":"简洁原因"}
  ]
}

要求：
- 只输出会违反规则的人，无冲突则输出空数组
- staffId必须使用候选列表中的短ID（如staff_11）
- 只根据提供的数据判断，禁止臆测未提供的历史数据
- **极其重要**：禁止自行假设频次限制！"本周已排N次"的信息仅供参考，不代表存在频率上限。除非规则列表中有明确的频次规则（如"每周最多X次"），否则不得以"频次限制"为由排除任何人
- "本周已排班情况"仅用于辅助判断来源限制和资源预留等规则，不能作为频次排除的依据
- **无前日数据时**：如果相关班次显示"无排班数据"，则来源限制规则自动不适用，返回空数组`
}

// buildRuleConflictFilterUserPrompt LLM3用户提示词
func (e *ProgressiveTaskExecutor) buildRuleConflictFilterUserPrompt(
	targetShift *d_model.Shift,
	targetDate string,
	candidateStaff []*d_model.Employee,
	shiftFixedAssignments map[string][]string,
	currentDraft *d_model.ShiftScheduleDraft, // 【新增】当前班次已完成的动态排班
	relevantRules []*d_model.Rule,
	relatedShiftsSchedule map[string]map[string][]string,
) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("# 班次：%s\n", targetShift.Name))
	prompt.WriteString(fmt.Sprintf("# 当前排班日期：%s\n\n", formatDateWithWeekday(targetDate)))

	// 获取短ID映射和姓名映射
	var staffForwardMappings map[string]string
	var staffIDToName map[string]string
	if e.taskContext != nil {
		staffForwardMappings = e.taskContext.StaffForwardMappings
		staffIDToName = e.taskContext.StaffIDToName
	}

	// 候选人员（带短ID）
	prompt.WriteString("# 候选人员\n\n")
	staffInfoList := make([]string, 0, len(candidateStaff))
	for _, staff := range candidateStaff {
		shortID := staff.ID
		if staffForwardMappings != nil {
			if sid, ok := staffForwardMappings[staff.ID]; ok {
				shortID = sid
			}
		}
		if shortID == staff.ID && e.taskContext != nil {
			shortID = e.taskContext.MaskStaffID(staff.ID) // 禁止UUID泄漏
		}
		staffInfoList = append(staffInfoList, fmt.Sprintf("%s(%s)", staff.Name, shortID))
	}
	prompt.WriteString(fmt.Sprintf("共%d人：%s\n\n", len(candidateStaff), strings.Join(staffInfoList, ", ")))

	// 本周已排班情况（固定排班 + 已完成的动态排班）
	prompt.WriteString("# 本班次本周已排班情况\n\n")

	// 合并固定排班和动态排班
	allScheduled := make(map[string][]string) // date -> staffNames (with type marker)

	// 添加固定排班
	for date, staffIDs := range shiftFixedAssignments {
		for _, staffID := range staffIDs {
			staffName := staffID
			if staffIDToName != nil {
				if name, ok := staffIDToName[staffID]; ok {
					staffName = name
				}
			}
			if staffName == staffID && e.taskContext != nil {
				staffName = e.taskContext.GetStaffName(staffID) // 禁止UUID泄漏
			}
			allScheduled[date] = append(allScheduled[date], staffName+"[固定]")
		}
	}

	// 添加已完成的动态排班
	if currentDraft != nil && currentDraft.Schedule != nil {
		for date, staffIDs := range currentDraft.Schedule {
			for _, staffID := range staffIDs {
				staffName := staffID
				if staffIDToName != nil {
					if name, ok := staffIDToName[staffID]; ok {
						staffName = name
					}
				}
				if staffName == staffID && e.taskContext != nil {
					staffName = e.taskContext.GetStaffName(staffID) // 禁止UUID泄漏
				}
				allScheduled[date] = append(allScheduled[date], staffName)
			}
		}
	}

	if len(allScheduled) > 0 {
		// 按日期排序显示
		dates := make([]string, 0, len(allScheduled))
		for date := range allScheduled {
			dates = append(dates, date)
		}
		sort.Strings(dates)

		// 构建人员->日期的映射，便于LLM理解
		staffDates := make(map[string][]string)
		for date, staffNames := range allScheduled {
			for _, staffName := range staffNames {
				// 去掉[固定]标记来统计人员
				cleanName := strings.TrimSuffix(staffName, "[固定]")
				staffDates[cleanName] = append(staffDates[cleanName], formatDateWithWeekday(date))
			}
		}

		// 按人员显示已排班日期
		for staffName, scheduledDates := range staffDates {
			sort.Strings(scheduledDates)
			prompt.WriteString(fmt.Sprintf("- %s: 本周已排 %d 次 (%s)\n", staffName, len(scheduledDates), strings.Join(scheduledDates, ", ")))
		}
	} else {
		prompt.WriteString("无\n")
	}
	prompt.WriteString("\n")

	// 【新增】相关班次排班信息
	if len(relatedShiftsSchedule) > 0 {
		prompt.WriteString("# 相关班次排班\n\n")

		// 获取前一日日期
		previousDate := getPreviousDate(targetDate)
		hasPreviousDayData := false

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

			// 显示前一日排班（重要：来源限制规则需要）
			if staffNames, exists := dateSchedule[previousDate]; exists && len(staffNames) > 0 {
				prompt.WriteString(fmt.Sprintf("  - 前一日(%s)：%s\n", formatDateWithWeekday(previousDate), strings.Join(staffNames, ", ")))
				hasPreviousDayData = true
			} else {
				prompt.WriteString(fmt.Sprintf("  - 前一日(%s)：无排班数据\n", formatDateWithWeekday(previousDate)))
			}

			// 显示当日排班（重要：资源预留规则需要）
			if staffNames, exists := dateSchedule[targetDate]; exists && len(staffNames) > 0 {
				prompt.WriteString(fmt.Sprintf("  - 当日(%s)：%s\n", formatDateWithWeekday(targetDate), strings.Join(staffNames, ", ")))
			}
		}

		// 如果没有前一日数据，添加提示
		if !hasPreviousDayData {
			prompt.WriteString("\n⚠️ 注意：所有相关班次均无前一日排班数据，来源限制规则不适用。\n")
		}
		prompt.WriteString("\n")
	}

	// 相关规则
	prompt.WriteString("# 相关规则\n\n")
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

	// 任务
	prompt.WriteString("# 任务\n\n")
	prompt.WriteString(fmt.Sprintf("请判断：在【%s】为【%s】排班时，哪些候选人会因为以下原因不可用：\n", formatDateWithWeekday(targetDate), targetShift.Name))
	prompt.WriteString("1. 频次限制：**仅当上方【相关规则】中存在明确的频次限制规则时才检查此项**。如果规则中没有频次限制，则所有人在此项均无冲突\n")
	prompt.WriteString("2. 来源限制：如果规则要求人员必须来自前一日的某班次，且【相关班次排班】中有前一日数据，不在该班次的人不可用。**如果前一日显示\"无排班数据\"，则来源限制不适用**\n")
	prompt.WriteString("3. 资源预留：如果规则要求本班次人员来自前一日某班次，则当日该班次的人员需保留给次日，不可用于当日\n")
	prompt.WriteString("\n**重要提醒**：\n")
	prompt.WriteString("- 上方【本周已排班情况】仅供参考（用于来源限制和资源预留判断），不代表存在频率上限\n")
	prompt.WriteString("- 禁止自行假设\"每人每周最多N次\"的限制，除非规则列表中有明确规定\n")
	prompt.WriteString("- **如果相关班次的前一日排班显示\"无排班数据\"，来源限制规则不生效，不得排除任何人**\n")
	prompt.WriteString("- 如果没有任何规则会导致冲突，请返回空的conflictStaff数组\n")

	return prompt.String()
}

// parseRuleConflictResponse 解析规则冲突人员过滤响应
func (e *ProgressiveTaskExecutor) parseRuleConflictResponse(content string) (*RuleConflictAnalysis, error) {
	jsonContent := extractJSON(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("无法提取JSON内容")
	}

	var analysis RuleConflictAnalysis
	if err := json.Unmarshal([]byte(jsonContent), &analysis); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return &analysis, nil
}
