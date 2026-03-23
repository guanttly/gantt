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

	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// isTaskExplicitlyEmpty 判断任务是否显式为空
func (e *ProgressiveTaskExecutor) isTaskExplicitlyEmpty(task *d_model.ProgressiveTask) bool {
	// 检查任务标题中是否包含明确的"无需处理"标记
	// 这些标记通常由任务计划生成阶段添加，明确表示该阶段无需执行
	emptyMarkers := []string{
		"无正向需求",
		"无需求",
		"无需执行",
		"跳过",
		"无需处理",
		"无需安排",
	}

	title := strings.ToLower(task.Title)
	for _, marker := range emptyMarkers {
		if strings.Contains(title, strings.ToLower(marker)) {
			return true
		}
	}

	// 检查任务描述中是否明确说明无需执行
	if task.Description != "" {
		desc := strings.ToLower(task.Description)
		descEmptyMarkers := []string{
			"无需执行任何操作",
			"第一阶段正向需求填充无需执行",
			"当前无任何正向需求",
		}
		for _, marker := range descEmptyMarkers {
			if strings.Contains(desc, strings.ToLower(marker)) {
				return true
			}
		}
	}

	return false
}

// buildTaskParsingPrompt 构建任务解析的LLM提示词
// 用于解析任务描述，识别目标班次并为每个班次生成任务说明
func (e *ProgressiveTaskExecutor) buildTaskParsingPrompt(
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
) string {
	var prompt strings.Builder

	prompt.WriteString("你是一个专业的排班任务解析助手。你的任务是根据任务描述，识别该任务涉及哪些班次，并为每个班次生成专门的任务说明。\n\n")

	prompt.WriteString("**当前任务**：\n")
	prompt.WriteString(fmt.Sprintf("- 任务标题：%s\n", task.Title))
	if task.Description != "" {
		// 确保任务描述中的UUID已替换为shortID（虽然理论上应该已经是shortID，但为了安全起见还是处理一下）
		desc := task.Description
		// 构建ID映射（用于替换可能存在的UUID）
		shiftForwardMappings, _ := utils.BuildShiftIDMappings(shifts)
		// 注意：这里只替换UUID，不替换shortID（因为shortID是期望的格式）
		desc = utils.ReplaceIDsWithShortIDs(desc, shiftForwardMappings)
		prompt.WriteString(fmt.Sprintf("- 任务描述：%s\n", desc))
	}
	if len(task.TargetDates) > 0 {
		prompt.WriteString(fmt.Sprintf("- 目标日期：%s\n", strings.Join(task.TargetDates, ", ")))
	}

	prompt.WriteString("\n**可用班次列表**：\n")
	// 构建班次ID映射（用于替换UUID）
	shiftForwardMappings, _ := utils.BuildShiftIDMappings(shifts)

	for i, shift := range shifts {
		if i >= 20 {
			prompt.WriteString(fmt.Sprintf("... 还有 %d 个班次\n", len(shifts)-20))
			break
		}
		shortID := shiftForwardMappings[shift.ID]
		if shortID == "" {
			shortID = fmt.Sprintf("shift_%d", i+1) // 禁止UUID泄漏给LLM
		}
		prompt.WriteString(fmt.Sprintf("%d. %s (ID: %s, 类型: %s", i+1, shift.Name, shortID, shift.Type))
		if shift.StartTime != "" {
			prompt.WriteString(fmt.Sprintf(", 开始时间: %s", shift.StartTime))
		}
		if shift.EndTime != "" {
			prompt.WriteString(fmt.Sprintf(", 结束时间: %s", shift.EndTime))
		}
		prompt.WriteString(")\n")
	}

	prompt.WriteString("\n**解析要求**：\n")
	prompt.WriteString("1. 仔细阅读任务标题和描述，识别任务涉及哪些班次\n")
	prompt.WriteString("2. 严格从上方【可用班次列表】中按名称或ID匹配，禁止编造、推测或组合出列表中不存在的班次名称\n")
	prompt.WriteString("3. 为每个匹配的班次生成专门的任务说明，说明该班次的具体排班要求和注意事项\n")
	prompt.WriteString("4. 如果任务描述中没有明确指定班次，或者涉及所有班次，则返回空数组\n")
	prompt.WriteString("5. 如果无法确定任务涉及的班次，返回空列表并说明原因\n\n")

	prompt.WriteString("**输出格式**：\n")
	prompt.WriteString("请返回 JSON 格式，包含以下字段：\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"targetShifts\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"shiftId\": \"班次ID（使用简短ID）\",\n")
	prompt.WriteString("      \"shiftName\": \"班次名称\",\n")
	prompt.WriteString("      \"description\": \"该班次的任务说明（详细说明该班次的排班要求和注意事项）\"\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ],\n")
	prompt.WriteString("  \"reasoning\": \"解析思路说明（必须简洁，不超过300字）\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("```\n\n")
	prompt.WriteString("**JSON格式要求**：\n")
	prompt.WriteString("- reasoning字段必须简洁明了，不超过300字\n")
	prompt.WriteString("- 不要在reasoning中包含大段文字或重复内容\n")

	prompt.WriteString("**重要提示**：\n")
	prompt.WriteString("- 每个班次的任务说明应该清晰、具体，帮助后续执行时理解该班次的要求\n")
	prompt.WriteString("- 根据班次的名称和类型属性，匹配任务描述中提到的班次\n")
	prompt.WriteString("- 对于跨天班次（开始时间大于结束时间），需特别注意日期处理\n")
	prompt.WriteString("- 严禁幻觉：shiftId和shiftName必须严格来源于上方【可用班次列表】，不得自行创造不存在的班次名称或ID\n")

	return prompt.String()
}

// parseTaskTargetShifts 解析任务，识别目标班次并为每个班次生成任务说明
// 调用LLM解析任务描述，识别目标班次
func (e *ProgressiveTaskExecutor) parseTaskTargetShifts(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
) ([]ShiftTaskSpec, string, error) {
	// 【预判优化】检查任务标题/描述是否明确表示无需处理
	// 避免 LLM 误解导致不必要的解析和校验失败
	if e.isTaskExplicitlyEmpty(task) {
		reason := "任务标题明确表示无需处理（如'无正向需求'），跳过班次解析"
		e.logger.Info("Task explicitly marked as empty, skipping LLM parsing",
			"taskID", task.ID,
			"taskTitle", task.Title,
			"reason", reason)
		return []ShiftTaskSpec{}, reason, nil
	}

	// 构建解析提示词
	userPrompt := e.buildTaskParsingPrompt(task, shifts)

	// 构建系统提示词
	systemPrompt := "你是一个专业的排班任务解析助手。你的任务是根据任务描述，识别该任务涉及哪些班次，并为每个班次生成专门的任务说明。"

	// 调用LLM解析
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录到调试文件
	e.logLLMDebug(task.Title, logging.LLMCallTaskParsing, "", "", systemPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Task parsing LLM call failed", "taskID", task.ID, "error", err)
		return nil, "", fmt.Errorf("failed to call LLM for task parsing: %w", err)
	}

	// 解析LLM响应
	var parsingResult struct {
		TargetShifts []ShiftTaskSpec `json:"targetShifts"`
		Reasoning    string          `json:"reasoning,omitempty"`
	}

	// 提取JSON部分
	jsonStart := strings.Index(resp.Content, "{")
	jsonEnd := strings.LastIndex(resp.Content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		e.logger.Error("No valid JSON found in LLM response",
			"taskID", task.ID,
			"responsePreview", resp.Content[:min(200, len(resp.Content))])
		return nil, "", fmt.Errorf("AI返回的响应中未找到有效的JSON格式：响应可能不完整或格式错误")
	}

	jsonStr := resp.Content[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &parsingResult); err != nil {
		// 记录详细的错误信息，但不尝试修复
		e.logger.Error("Failed to parse LLM response as JSON",
			"taskID", task.ID,
			"error", err,
			"jsonPreview", jsonStr[:min(500, len(jsonStr))])

		// 返回用户友好的错误信息
		return nil, "", fmt.Errorf("AI返回的JSON格式有误：%v。这通常是由于AI生成了包含未转义特殊字符（如换行符）的内容。请重试任务，或调整提示词让AI返回更简洁的reasoning字段", err)
	}

	// 将简短ID替换回真实ID
	_, shiftReverseMappings := utils.BuildShiftIDMappings(shifts)
	for i := range parsingResult.TargetShifts {
		if realID, ok := shiftReverseMappings[parsingResult.TargetShifts[i].ShiftID]; ok {
			parsingResult.TargetShifts[i].ShiftID = realID
		}
		// 如果找不到映射，说明可能是真实ID，直接使用
	}

	// 验证解析出的班次是否存在
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		shiftMap[shift.ID] = shift
	}

	validSpecs := make([]ShiftTaskSpec, 0)
	for _, spec := range parsingResult.TargetShifts {
		if shift, ok := shiftMap[spec.ShiftID]; ok {
			// 更新班次名称（确保准确）
			spec.ShiftName = shift.Name
			validSpecs = append(validSpecs, spec)
		} else {
			e.logger.Warn("Parsed shift ID not found in available shifts",
				"shiftID", spec.ShiftID,
				"shiftName", spec.ShiftName)
		}
	}

	// 如果LLM返回空列表，说明任务不需要处理，这是合法的，直接返回空列表
	// 只有在格式错误（JSON解析失败等）时才应该报错
	if len(validSpecs) == 0 {
		e.logger.Info("Task parsing returned empty shifts (valid: task needs no processing)",
			"taskID", task.ID,
			"reasoning", parsingResult.Reasoning)
		return validSpecs, parsingResult.Reasoning, nil
	}

	return validSpecs, parsingResult.Reasoning, nil
}

// buildProgressiveDaySystemPrompt 构建渐进式单天排班系统提示词
func (e *ProgressiveTaskExecutor) buildProgressiveDaySystemPrompt() string {
	return `你是排班助手。核心原则：
1. 只为当前日期排班，严格按需求人数（不超配不欠配）
2. 需求已扣除固定排班，需求为0输出空schedule
3. 避免时间冲突，遵守规则约束
4. 输出JSON：{"mode":"add","schedule":{"日期":["staff_id"]},"reasoning":"简洁说明"}
reasoning不超300字

注意：只能从【候选人员】列表中选择，禁止选择列表之外的任何人！`
}

// buildProgressiveDayPromptWithAnalysis 构建渐进式单天排班用户提示词（带规则分析结果）
// 【占位信息格式统一】已移除 map 格式的 occupiedSlots 参数，统一使用 taskContext.OccupiedSlots（强类型数组）
func (e *ProgressiveTaskExecutor) buildProgressiveDayPromptWithAnalysis(
	targetShift *d_model.Shift,
	targetDate string,
	allDates []string,
	completedDates []string,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]int,
	currentDraft *d_model.ShiftScheduleDraft,
	personalNeeds map[string][]*d_model.PersonalNeed,
	fixedAssignments map[string][]string,
	relatedShiftsSchedule map[string]map[string][]string,
	retryContext *d_model.ShiftRetryContext,
	failedDatesIssues map[string][]string,
	ruleAnalysis *DayRuleAnalysis, // 【新增】规则分析结果
) string {
	var prompt strings.Builder

	// 从 context 获取ID映射
	var shiftForwardMappings, staffForwardMappings map[string]string
	var staffIDToName map[string]string

	if e.taskContext != nil {
		shiftForwardMappings = e.taskContext.ShiftForwardMappings
		staffForwardMappings = e.taskContext.StaffForwardMappings
		if len(e.taskContext.AllStaff) > 0 {
			staffIDToName = utils.BuildStaffIDToNameMapping(e.taskContext.AllStaff)
		} else {
			staffIDToName = utils.BuildStaffIDToNameMapping(staffList)
		}
	} else {
		staffForwardMappings, _ = utils.BuildStaffIDMappings(staffList)
		staffIDToName = utils.BuildStaffIDToNameMapping(staffList)
	}

	// ============================================================
	// 第1部分：班次信息
	// ============================================================
	prompt.WriteString("# 班次信息\n\n")
	shortShiftID := ""
	if shiftForwardMappings != nil {
		shortShiftID = shiftForwardMappings[targetShift.ID]
	}
	if shortShiftID == "" {
		shortShiftID = "shift_unknown" // 禁止UUID泄漏给LLM
	}
	duration := utils.GetShiftDurationHours(targetShift)
	prompt.WriteString(fmt.Sprintf("班次：%s (ID:%s) %s-%s %.1fh\n", targetShift.Name, shortShiftID, targetShift.StartTime, targetShift.EndTime, duration))
	prompt.WriteString(fmt.Sprintf("排班日期：%s\n\n", formatDateWithWeekday(targetDate)))

	// ============================================================
	// 第2部分：规则分析总结（来自策划阶段）
	// ============================================================
	if ruleAnalysis != nil && ruleAnalysis.Summary != "" {
		prompt.WriteString("# 当日排班注意事项\n\n")
		prompt.WriteString(fmt.Sprintf("⚠️ %s\n\n", ruleAnalysis.Summary))
	}

	// ============================================================
	// 第3部分：相关规则（已过滤）
	// ============================================================
	prompt.WriteString("# 规则约束\n\n")
	if ruleAnalysis != nil && len(ruleAnalysis.RelevantRules) > 0 {
		for i, rule := range ruleAnalysis.RelevantRules {
			prompt.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, rule.RuleName, rule.Constraint))
		}
	} else if len(rules) > 0 {
		// 如果没有分析结果，使用全部规则
		for i, rule := range rules {
			ruleContent := rule.RuleData
			if ruleContent == "" {
				ruleContent = rule.Description
			}
			prompt.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, rule.Name, ruleContent))
		}
	} else {
		prompt.WriteString("无特定规则\n")
	}
	prompt.WriteString("\n")

	// ============================================================
	// 构建当日不可用人员集合（用于后续从候选人员中排除，不在提示词中显示）
	// ============================================================
	unavailableStaffIDs := make(map[string]bool)
	if ruleAnalysis != nil && len(ruleAnalysis.RelevantPersonalNeeds) > 0 {
		for _, need := range ruleAnalysis.RelevantPersonalNeeds {
			unavailableStaffIDs[need.StaffID] = true
		}
	} else {
		// 如果没有分析结果，从原始个人需求中过滤当日的
		for staffID, needs := range personalNeeds {
			for _, need := range needs {
				if isDateInPersonalNeed(targetDate, need) {
					unavailableStaffIDs[staffID] = true
					break
				}
			}
		}
	}

	// ============================================================
	// 第4部分：当前排班草案
	// ============================================================
	prompt.WriteString("# 当前排班草案\n\n")
	for _, date := range allDates {
		status := ""
		if contains(completedDates, date) {
			status = " ✅"
		} else if date == targetDate {
			status = " ⏳待排"
		} else {
			status = " ⬜待处理"
		}

		staffNames := []string{}
		addedStaffIDs := make(map[string]bool)

		// 先添加固定排班人员
		if fixed, ok := fixedAssignments[date]; ok {
			for _, staffID := range fixed {
				name := staffIDToName[staffID]
				if name == "" {
					if e.taskContext != nil {
						name = e.taskContext.GetStaffName(staffID)
					} else {
						name = staffID
					}
				}
				staffNames = append(staffNames, name+"[固定]")
				addedStaffIDs[staffID] = true
			}
		}

		// 再添加draft中的动态排班人员
		if currentDraft != nil && currentDraft.Schedule != nil {
			for _, staffID := range currentDraft.Schedule[date] {
				if addedStaffIDs[staffID] {
					continue
				}
				name := staffIDToName[staffID]
				if name == "" {
					if e.taskContext != nil {
						name = e.taskContext.GetStaffName(staffID)
					} else {
						name = staffID
					}
				}
				staffNames = append(staffNames, name)
				addedStaffIDs[staffID] = true
			}
		}

		dateWithWeekday := formatDateWithWeekday(date)
		if len(staffNames) > 0 {
			prompt.WriteString(fmt.Sprintf("- %s: %s%s\n", dateWithWeekday, strings.Join(staffNames, ", "), status))
		} else {
			prompt.WriteString(fmt.Sprintf("- %s: (空)%s\n", dateWithWeekday, status))
		}
	}
	prompt.WriteString("\n")

	// ============================================================
	// 第6部分：相关班次排班
	// ============================================================
	if len(relatedShiftsSchedule) > 0 {
		prompt.WriteString("# 相关班次排班\n\n")
		sortedShiftIDs := make([]string, 0, len(relatedShiftsSchedule))
		for shiftID := range relatedShiftsSchedule {
			sortedShiftIDs = append(sortedShiftIDs, shiftID)
		}
		sort.Strings(sortedShiftIDs)

		for _, shiftID := range sortedShiftIDs {
			dateSchedule := relatedShiftsSchedule[shiftID]
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
			sortedDates := make([]string, 0, len(dateSchedule))
			for date := range dateSchedule {
				sortedDates = append(sortedDates, date)
			}
			sort.Strings(sortedDates)
			for _, date := range sortedDates {
				staffNameList := dateSchedule[date]
				if len(staffNameList) > 0 {
					prompt.WriteString(fmt.Sprintf("  - %s: %s\n", formatDateWithWeekday(date), strings.Join(staffNameList, ", ")))
				}
			}
		}
		prompt.WriteString("\n")
	}

	// ============================================================
	// 第7部分：本次任务
	// ============================================================
	prompt.WriteString("# 本次任务\n\n")
	prompt.WriteString(fmt.Sprintf("请为 **%s** 排班\n\n", formatDateWithWeekday(targetDate)))

	dynamicRequired := staffRequirements[targetDate]
	fixedCount := 0
	fixedStaff := []string{}
	if fixed, ok := fixedAssignments[targetDate]; ok {
		fixedCount = len(fixed)
		for _, fid := range fixed {
			name := staffIDToName[fid]
			if name == "" {
				if e.taskContext != nil {
					name = e.taskContext.GetStaffName(fid)
				} else {
					name = fid
				}
			}
			fixedStaff = append(fixedStaff, name)
		}
	}

	if fixedCount > 0 {
		prompt.WriteString(fmt.Sprintf("固定排班（已安排）：%s（共%d人）\n", strings.Join(fixedStaff, ", "), fixedCount))
		prompt.WriteString(fmt.Sprintf("**你需要新增排班：%d人**\n", dynamicRequired))
	} else {
		prompt.WriteString(fmt.Sprintf("**需要排班：%d人**\n", dynamicRequired))
	}
	prompt.WriteString("\n")

	// ============================================================
	// 第7部分：候选人员（已排除当日不可用人员）
	// ============================================================
	prompt.WriteString("# 候选人员\n\n")
	availableStaff := []string{}

	// 构建当日不可用人员集合（同时用ID和姓名匹配）
	unavailableIDs := make(map[string]bool)
	unavailableNames := make(map[string]bool) // 【新增】用姓名匹配（LLM3可能只返回姓名）
	if ruleAnalysis != nil {
		for _, need := range ruleAnalysis.RelevantPersonalNeeds {
			if need.StaffID != "" {
				unavailableIDs[need.StaffID] = true
			}
			if need.StaffName != "" {
				unavailableNames[need.StaffName] = true
			}
		}
	} else {
		// 从原始个人需求中过滤
		for staffID, needs := range personalNeeds {
			for _, need := range needs {
				if isDateInPersonalNeed(targetDate, need) {
					unavailableIDs[staffID] = true
					break
				}
			}
		}
	}

	// ============================================================
	// 【P0修复】排除当日固定排班人员（防止LLM重复选中导致去重后少人）
	// ============================================================
	if fixed, ok := fixedAssignments[targetDate]; ok {
		for _, staffID := range fixed {
			unavailableIDs[staffID] = true
			if staffIDToName != nil {
				if name, ok := staffIDToName[staffID]; ok {
					unavailableNames[name] = true
				}
			}
		}
	}

	// ============================================================
	// 【P0修复】时间冲突人员过滤（硬编码逻辑，不依赖LLM）
	// 检查当日已被占用的人员，如果其已排班次与目标班次时间重叠，则不可用
	// ============================================================
	// 构建班次ID到班次信息的映射（用于所有时间冲突检查）
	var shiftMap map[string]*d_model.Shift
	if e.taskContext != nil && e.taskContext.Shifts != nil {
		shiftMap = make(map[string]*d_model.Shift)
		for _, s := range e.taskContext.Shifts {
			if s != nil {
				shiftMap[s.ID] = s
			}
		}
	}

	// 【占位信息格式统一】检查 taskContext.OccupiedSlots 中的占位情况
	if e.taskContext != nil && shiftMap != nil {
		// 遍历所有占位记录，查找目标日期的占位
		for _, slot := range e.taskContext.OccupiedSlots {
			if slot.Date == targetDate {
				// 该人员在当日已有排班，检查时间是否与目标班次重叠
				existingShift := shiftMap[slot.ShiftID]
				if existingShift != nil && utils.CheckTimeOverlap(targetShift, existingShift) {
					unavailableIDs[slot.StaffID] = true
					// 同时记录姓名（用于匹配）
					if staffIDToName != nil {
						if name, ok := staffIDToName[slot.StaffID]; ok {
							unavailableNames[name] = true
						}
					}
				}
			}
		}
	}

	// 2. 检查当前草案中已排班的人员（防止重复排班）
	if currentDraft != nil && currentDraft.Schedule != nil {
		if existingStaffIDs, exists := currentDraft.Schedule[targetDate]; exists {
			for _, staffID := range existingStaffIDs {
				// 同一个班次不应该有时间冲突，但为了安全起见，如果发现重复排班，应该排除
				// 注意：这里可能需要根据业务逻辑决定是否允许同一班次在同一天排多次
				unavailableIDs[staffID] = true
				if staffIDToName != nil {
					if name, ok := staffIDToName[staffID]; ok {
						unavailableNames[name] = true
					}
				}
			}
		}
	}

	// 3. 检查相关班次排班中的人员（防止跨班次时间冲突）
	if len(relatedShiftsSchedule) > 0 && shiftMap != nil {
		// 遍历所有相关班次
		for relatedShiftID, dateSchedule := range relatedShiftsSchedule {
			if relatedShiftStaffNames, exists := dateSchedule[targetDate]; exists && len(relatedShiftStaffNames) > 0 {
				// 获取相关班次信息
				relatedShift := shiftMap[relatedShiftID]
				if relatedShift != nil {
					// 检查时间是否重叠（使用统一的 CheckTimeOverlap 函数）
					if utils.CheckTimeOverlap(targetShift, relatedShift) {
						// 时间重叠，将相关班次的人员标记为不可用
						for _, staffName := range relatedShiftStaffNames {
							// 通过姓名查找人员ID
							unavailableNames[staffName] = true
							for _, staff := range staffList {
								if staff.Name == staffName {
									unavailableIDs[staff.ID] = true
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// 优先使用 L3 动态候选人员（taskContext.CandidateStaff），回退到任务级 staffList
	candidateStaffList := staffList
	if e.taskContext != nil && len(e.taskContext.CandidateStaff) > 0 {
		candidateStaffList = e.taskContext.CandidateStaff
	}

	// 只添加可用人员（排除不可用人员，同时检查ID和姓名）
	for _, staff := range candidateStaffList {
		if unavailableIDs[staff.ID] || unavailableNames[staff.Name] {
			continue // 跳过不可用人员
		}
		staffName := staff.Name
		shortID := staffForwardMappings[staff.ID]
		if shortID == "" {
			if e.taskContext != nil {
				shortID = e.taskContext.MaskStaffID(staff.ID)
			} else {
				shortID = fmt.Sprintf("staff_%d", len(availableStaff)+1) // 禁止UUID泄漏给LLM
			}
		}
		availableStaff = append(availableStaff, fmt.Sprintf("%s(%s)", staffName, shortID))
	}

	if len(availableStaff) > 0 {
		prompt.WriteString(fmt.Sprintf("可选（%d人）：%s\n", len(availableStaff), strings.Join(availableStaff, ", ")))
		prompt.WriteString("\n**注意**：只能从上述候选人员中选择，禁止选择列表之外的任何人！\n")
	} else {
		prompt.WriteString("无可用人员\n")
	}
	prompt.WriteString("\n")

	// ============================================================
	// 第8部分：重试信息（如果是重试）
	// ============================================================
	if retryContext != nil && retryContext.RetryCount > 0 {
		prompt.WriteString("# ⚠️ 重试信息（请务必参考）\n\n")
		prompt.WriteString(fmt.Sprintf("这是第%d次重试，上次排班校验失败。\n\n", retryContext.RetryCount))

		// 【新增】定向重排模式说明
		if retryContext.RetryOnlyTargetDates && len(retryContext.TargetRetryDates) > 0 {
			prompt.WriteString("**🎯 定向重排模式**：\n")
			prompt.WriteString(fmt.Sprintf("- 仅需重新安排以下日期：%v\n", retryContext.TargetRetryDates))
			if len(retryContext.ConflictingShiftIDs) > 0 {
				// 获取所有冲突班次名称
				conflictShiftNames := make([]string, 0, len(retryContext.ConflictingShiftIDs))
				for _, conflictShiftID := range retryContext.ConflictingShiftIDs {
					shiftName := conflictShiftID
					if e.taskContext != nil && e.taskContext.Shifts != nil {
						for _, s := range e.taskContext.Shifts {
							if s != nil && s.ID == conflictShiftID {
								shiftName = s.Name
								break
							}
						}
					}
					conflictShiftNames = append(conflictShiftNames, shiftName)
				}
				prompt.WriteString(fmt.Sprintf("- 原因：与 [%s] 班次存在人员冲突\n", strings.Join(conflictShiftNames, ", ")))
				prompt.WriteString("- 请选择与这些班次不同的人员，避免同一人在同一天被安排到互斥的班次\n")
			}
			prompt.WriteString("\n")
		}

		// 显示所有失败日期的问题
		if len(failedDatesIssues) > 0 {
			prompt.WriteString("**上次排班的问题点**：\n")
			for date, issues := range failedDatesIssues {
				if len(issues) > 0 {
					prompt.WriteString(fmt.Sprintf("- %s：\n", formatDateWithWeekday(date)))
					for _, issue := range issues {
						prompt.WriteString(fmt.Sprintf("    - %s\n", issue))
					}
				}
			}
			prompt.WriteString("\n")

			// 特别标注当前日期是否有问题
			if issues, ok := failedDatesIssues[targetDate]; ok && len(issues) > 0 {
				prompt.WriteString(fmt.Sprintf("⚠️ 当前日期 %s 有问题，请特别注意！\n\n", formatDateWithWeekday(targetDate)))
			}
		}

		if retryContext.AIRecommendations != "" {
			prompt.WriteString(fmt.Sprintf("**改进建议**：%s\n", retryContext.AIRecommendations))
		}
		prompt.WriteString("\n")
	}

	// ============================================================
	// 第10部分：任务总结
	// ============================================================
	prompt.WriteString(fmt.Sprintf("# 任务：为 %s 排 %d 人\n", formatDateWithWeekday(targetDate), dynamicRequired))

	return prompt.String()
}

// ============================================================
// 关联班次识别提示词构建
// ============================================================

// buildShiftGroupingPrompt 构建关联班次分组识别的LLM提示词
// 用于识别哪些班次需要一起排班（共享规则和人员信息）
// 返回: prompt, shiftForwardMappings, ruleForwardMappings
func (e *ProgressiveTaskExecutor) buildShiftGroupingPrompt(
	shiftSpecs []ShiftTaskSpec,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
) (string, map[string]string, map[string]string) {
	var prompt strings.Builder

	// 构建ID映射表（UUID -> shortID）
	shiftForwardMappings, _ := utils.BuildShiftIDMappings(shifts)
	ruleForwardMappings, _ := utils.BuildRuleIDMappings(rules)

	// 构建班次名称映射（用于关联班次显示）
	shiftNameMap := make(map[string]string)
	for _, s := range shifts {
		shiftNameMap[s.ID] = s.Name
	}

	prompt.WriteString("你是一个专业的排班关联性分析助手。你的任务是分析多个班次之间的关联关系，判断哪些班次需要**一起排班**（作为一个组），哪些可以**独立排班**。\n\n")

	// 第1部分：待分析的班次列表
	prompt.WriteString("# 待分析的班次\n\n")
	for i, spec := range shiftSpecs {
		// 获取shortID
		shortID := shiftForwardMappings[spec.ShiftID]
		if shortID == "" {
			shortID = fmt.Sprintf("shift_%d", i+1)
		}

		// 查找班次详细信息
		var shiftInfo *d_model.Shift
		for _, s := range shifts {
			if s.ID == spec.ShiftID {
				shiftInfo = s
				break
			}
		}
		if shiftInfo != nil {
			prompt.WriteString(fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1, spec.ShiftName, shortID))
			prompt.WriteString(fmt.Sprintf("   - 时间：%s - %s\n", shiftInfo.StartTime, shiftInfo.EndTime))
			prompt.WriteString(fmt.Sprintf("   - 任务说明：%s\n", spec.Description))
		} else {
			prompt.WriteString(fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1, spec.ShiftName, shortID))
			prompt.WriteString(fmt.Sprintf("   - 任务说明：%s\n", spec.Description))
		}
	}
	prompt.WriteString("\n")

	// 第2部分：相关规则
	prompt.WriteString("# 排班规则\n\n")
	if len(rules) == 0 {
		prompt.WriteString("（无规则）\n\n")
	} else {
		// 过滤出与这些班次相关的规则
		shiftIDSet := make(map[string]bool)
		for _, spec := range shiftSpecs {
			shiftIDSet[spec.ShiftID] = true
		}

		relatedRules := make([]*d_model.Rule, 0)
		for _, rule := range rules {
			// 检查规则是否与任意待分析班次相关（通过Associations字段）
			isRelated := false
			for _, assoc := range rule.Associations {
				if assoc.AssociationType == "shift" && shiftIDSet[assoc.AssociationID] {
					isRelated = true
					break
				}
			}
			if isRelated {
				relatedRules = append(relatedRules, rule)
			}
		}

		if len(relatedRules) == 0 {
			prompt.WriteString("（无与这些班次直接相关的规则）\n\n")
		} else {
			for i, rule := range relatedRules {
				// 使用规则shortID
				ruleShortID := ruleForwardMappings[rule.ID]
				if ruleShortID == "" {
					ruleShortID = fmt.Sprintf("rule_%d", i+1)
				}
				prompt.WriteString(fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1, rule.Name, ruleShortID))
				prompt.WriteString(fmt.Sprintf("   - 规则内容：%s\n", rule.Description))
				// 收集关联的班次（使用名称+shortID）
				shiftAssocs := make([]string, 0)
				for _, assoc := range rule.Associations {
					if assoc.AssociationType == "shift" {
						assocShortID := shiftForwardMappings[assoc.AssociationID]
						if assocShortID == "" {
							assocShortID = assoc.AssociationID // 如果不在映射中，保留原ID
						}
						shiftName := shiftNameMap[assoc.AssociationID]
						if shiftName != "" {
							shiftAssocs = append(shiftAssocs, fmt.Sprintf("%s(%s)", shiftName, assocShortID))
						} else {
							shiftAssocs = append(shiftAssocs, assocShortID)
						}
					}
				}
				if len(shiftAssocs) > 0 {
					prompt.WriteString(fmt.Sprintf("   - 关联班次：%v\n", shiftAssocs))
				}
			}
		}
	}
	prompt.WriteString("\n")

	// 第3部分：分组判断标准
	prompt.WriteString("# 分组判断标准\n\n")
	prompt.WriteString("判断两个班次是否需要**一起排班**（放入同一组）的依据：\n\n")
	prompt.WriteString("1. **人员来源依赖**：班次A的人员必须来自班次B（如：下夜班人员必须来自前一日上半夜班）\n")
	prompt.WriteString("2. **人员互斥约束**：班次A和班次B不能安排同一人员（如：审核和报告不能同人）\n")
	prompt.WriteString("3. **人员联动要求**：班次A的人员必须同时在班次B（如：穿刺人员必须同日在报告上）\n")
	prompt.WriteString("4. **资源预留关系**：为班次A排班会影响班次B的可用人员池\n\n")

	prompt.WriteString("**不需要一起排班**的情况：\n")
	prompt.WriteString("- 两个班次没有任何规则关联\n")
	prompt.WriteString("- 两个班次的人员池完全独立\n")
	prompt.WriteString("- 规则只涉及单个班次内部的约束（如个人每周限制）\n\n")

	// 第4部分：输出格式
	prompt.WriteString("# 输出格式\n\n")
	prompt.WriteString("请返回 JSON 格式：\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"groups\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"groupId\": \"group_1\",\n")
	prompt.WriteString("      \"shiftIds\": [\"shift_1\", \"shift_2\"],\n")
	prompt.WriteString("      \"relatedReason\": \"简述为什么这些班次需要一起排班\",\n")
	prompt.WriteString("      \"sharedRuleIds\": [\"rule_1\", \"rule_2\"]\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ],\n")
	prompt.WriteString("  \"reasoning\": \"整体分组决策说明（不超过200字）\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**重要说明**：\n")
	prompt.WriteString("- 每个班次必须且只能属于一个分组\n")
	prompt.WriteString("- 使用上述班次和规则的shortID（如shift_1, rule_1）\n")
	prompt.WriteString("- 如果某班次与其他班次无关联，可以单独成组（shiftIds只有一个元素）\n")
	prompt.WriteString("- 同组班次将共享规则和人员信息，以确保排班一致性\n")
	prompt.WriteString("- 不同组的班次将独立排班，互不影响\n")

	return prompt.String(), shiftForwardMappings, ruleForwardMappings
}

// buildShiftGroupingSystemPrompt 构建关联班次分组识别的系统提示词
func (e *ProgressiveTaskExecutor) buildShiftGroupingSystemPrompt() string {
	return `你是排班关联性分析专家。核心任务：
1. 分析班次之间的规则依赖关系
2. 识别需要一起排班的班次组（共享人员池或规则约束）
3. 将无关联的班次分开，减少提示词冗余
4. 输出简洁的JSON分组结果

判断原则：宁可多合并（保证正确性），不可错分离（导致规则冲突）`
}
