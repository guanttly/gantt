package executor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// buildUnavailableStaffMap 从负向需求构建不可用人员清单
func (e *ProgressiveTaskExecutor) buildUnavailableStaffMap(
	personalNeeds map[string][]*d_model.PersonalNeed,
	targetDates []string,
	startDate, endDate string,
) *d_model.UnavailableStaffMap {
	unavailableMap := d_model.NewUnavailableStaffMap()

	if len(personalNeeds) == 0 {
		return unavailableMap
	}

	// 遍历所有个人需求
	for staffID, needs := range personalNeeds {
		for _, need := range needs {
			if need == nil {
				continue
			}

			// 识别负向需求：
			// 1. RequestType为"avoid"
			// 2. 或RequestType为"prefer"/"must"但未指定TargetShiftID（表示休息或回避）
			isNegative := need.RequestType == "avoid" ||
				((need.RequestType == "prefer" || need.RequestType == "must") && need.TargetShiftID == "")

			if !isNegative {
				continue
			}

			// 确定不可用日期
			var unavailableDates []string
			if len(need.TargetDates) == 0 {
				// 如果TargetDates为空，表示整个周期都不可用
				// 需要生成整个周期的日期列表
				if len(targetDates) > 0 {
					// 使用目标日期列表
					unavailableDates = targetDates
				} else {
					// 如果没有目标日期，使用整个周期（需要传入startDate和endDate）
					// 这里简化处理：如果targetDates为空，暂时不处理
					// 实际使用时应该传入周期日期列表
					continue
				}
			} else {
				// 如果TargetDates有值，只在这些日期不可用
				unavailableDates = need.TargetDates
			}

			// 添加不可用记录
			unavailableMap.AddUnavailableDates(staffID, unavailableDates)
		}
	}

	return unavailableMap
}

// buildAvailableStaffList 构建可用人员列表
func (e *ProgressiveTaskExecutor) buildAvailableStaffList(
	staffList []*d_model.Employee,
	currentDraft *d_model.ShiftScheduleDraft,
	occupiedSlots map[string]map[string]string,
	unavailableStaffMap *d_model.UnavailableStaffMap,
	targetDates []string,
) []*d_model.StaffInfoForAI {
	result := make([]*d_model.StaffInfoForAI, 0, len(staffList))

	for _, staff := range staffList {
		if staff == nil {
			continue
		}

		// 检查该人员是否在目标日期中不可用
		if unavailableStaffMap != nil && unavailableStaffMap.IsUnavailableInAnyDate(staff.ID, targetDates) {
			// 该人员在目标日期中不可用，跳过
			e.logger.Debug("Excluding unavailable staff from available list",
				"staffID", staff.ID,
				"staffName", staff.Name,
				"targetDates", targetDates)
			continue
		}

		info := &d_model.StaffInfoForAI{
			ID:   staff.ID,
			Name: staff.Name,
		}

		// 提取分组名称
		if len(staff.Groups) > 0 {
			info.Groups = make([]string, 0, len(staff.Groups))
			for _, g := range staff.Groups {
				if g != nil {
					info.Groups = append(info.Groups, g.Name)
				}
			}
		}

		// 添加已排班标记（从 currentDraft 和 occupiedSlots 中提取）
		if currentDraft != nil && currentDraft.Schedule != nil {
			scheduledShifts := make(map[string][]*d_model.ShiftMark)
			for date, staffIDs := range currentDraft.Schedule {
				for _, id := range staffIDs {
					if id == staff.ID {
						// 该人员在该日期已被排班
						if scheduledShifts[date] == nil {
							scheduledShifts[date] = make([]*d_model.ShiftMark, 0)
						}
						scheduledShifts[date] = append(scheduledShifts[date], &d_model.ShiftMark{
							ShiftID:   "", // 当前班次ID，需要从上下文中获取
							StartTime: "",
							EndTime:   "",
						})
					}
				}
			}
			if len(scheduledShifts) > 0 {
				info.ScheduledShifts = scheduledShifts
			}
		}

		// 添加占位信息（从 occupiedSlots 中提取）
		if occupiedSlots != nil {
			if dates, ok := occupiedSlots[staff.ID]; ok {
				if info.ScheduledShifts == nil {
					info.ScheduledShifts = make(map[string][]*d_model.ShiftMark)
				}
				for date, shiftID := range dates {
					if info.ScheduledShifts[date] == nil {
						info.ScheduledShifts[date] = make([]*d_model.ShiftMark, 0)
					}
					info.ScheduledShifts[date] = append(info.ScheduledShifts[date], &d_model.ShiftMark{
						ShiftID:   shiftID,
						StartTime: "",
						EndTime:   "",
					})
				}
			}
		}

		result = append(result, info)
	}

	return result
}

// extractTemporaryNeedsFromTask 从任务中提取临时需求
// 根据任务类型过滤相关的个人需求：
// - 正向需求任务：只提取RequestType为prefer/must且指定了TargetShiftID的需求
// - 特殊班次填充/剩余填充任务：只提取RequestType为avoid或未指定TargetShiftID的需求（负向需求）
func (e *ProgressiveTaskExecutor) extractTemporaryNeedsFromTask(task *d_model.ProgressiveTask) []*d_model.PersonalNeed {
	if e.taskContext == nil || e.taskContext.PersonalNeeds == nil {
		return make([]*d_model.PersonalNeed, 0)
	}

	// 判断任务类型（通过标题和描述推断）
	title := strings.ToLower(task.Title)
	desc := strings.ToLower(task.Description)

	isPositiveNeedTask := strings.Contains(title, "正向需求") || strings.Contains(desc, "正向需求")
	isNegativeNeedTask := strings.Contains(title, "负向需求") || strings.Contains(desc, "负向需求") ||
		strings.Contains(title, "特殊班次") || strings.Contains(title, "剩余") ||
		strings.Contains(desc, "避开") || strings.Contains(desc, "回避")

	result := make([]*d_model.PersonalNeed, 0)

	// 遍历所有个人需求，根据任务类型过滤
	for _, needs := range e.taskContext.PersonalNeeds {
		for _, need := range needs {
			if need == nil {
				continue
			}

			// 判断需求是正向还是负向
			isPositive := (need.RequestType == "prefer" || need.RequestType == "must") && need.TargetShiftID != ""

			// 根据任务类型决定是否包含该需求
			if isPositiveNeedTask && isPositive {
				// 正向需求任务：只包含正向需求
				result = append(result, need)
			} else if isNegativeNeedTask && !isPositive {
				// 特殊班次填充/剩余填充任务：只包含负向需求
				result = append(result, need)
			}
		}
	}

	return result
}

// getTaskShifts 获取任务相关的班次
func (e *ProgressiveTaskExecutor) getTaskShifts(task *d_model.ProgressiveTask, shifts []*d_model.Shift) []*d_model.Shift {
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

// splitTaskByShifts 将多班次任务拆分成单班次子任务
// 如果任务只涉及一个班次或没有指定班次，直接返回原任务
// 如果涉及多个班次，拆分成多个单班次子任务，每个子任务保留原任务的其他信息
// 拆分后按 SchedulingPriority 排序，确保被依赖的班次先执行
func (e *ProgressiveTaskExecutor) splitTaskByShifts(
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
) []*d_model.ProgressiveTask {
	// 获取任务相关的班次
	taskShifts := e.getTaskShifts(task, shifts)

	// 如果只有一个班次或没有指定班次，直接返回原任务
	if len(taskShifts) <= 1 {
		return []*d_model.ProgressiveTask{task}
	}

	// 按 SchedulingPriority 升序排序（数字越小越先执行，0为最高优先级）
	sort.SliceStable(taskShifts, func(i, j int) bool {
		return taskShifts[i].SchedulingPriority < taskShifts[j].SchedulingPriority
	})

	// 拆分成多个单班次子任务
	subTasks := make([]*d_model.ProgressiveTask, 0, len(taskShifts))
	for _, shift := range taskShifts {
		subTask := &d_model.ProgressiveTask{
			ID:           fmt.Sprintf("%s_shift_%s", task.ID, shift.ID),
			Order:        task.Order, // 保持原顺序
			Title:        task.Title,
			Description:  task.Description,
			TargetShifts: []string{shift.ID}, // 只包含当前班次
			TargetDates:  task.TargetDates,
			TargetStaff:  task.TargetStaff,
			RuleIDs:      task.RuleIDs,
			Priority:     task.Priority,
			Status:       task.Status,
		}
		subTasks = append(subTasks, subTask)
		e.logger.Debug("Created sub-task",
			"subTaskID", subTask.ID,
			"shiftID", shift.ID,
			"shiftName", shift.Name)
	}

	return subTasks
}

// splitTaskByShiftsWithSpecs 使用解析出的班次规格创建子任务
// 为每个班次创建独立的子任务，使用解析出的任务说明
func (e *ProgressiveTaskExecutor) splitTaskByShiftsWithSpecs(
	task *d_model.ProgressiveTask,
	shiftSpecs []ShiftTaskSpec,
) []*d_model.ProgressiveTask {
	// 如果只有一个班次或没有班次，直接返回原任务
	if len(shiftSpecs) <= 1 {
		if len(shiftSpecs) == 1 {
			// 更新任务的TargetShifts和Description
			updatedTask := *task
			updatedTask.TargetShifts = []string{shiftSpecs[0].ShiftID}
			updatedTask.Description = shiftSpecs[0].Description
			return []*d_model.ProgressiveTask{&updatedTask}
		}
		return []*d_model.ProgressiveTask{task}
	}

	// ============================================================
	// 【依赖排序】：根据班次间依赖关系调整执行顺序
	// 规则：下夜班依赖前一天的夜班数据，所以夜班应该先执行，下夜班后执行
	// ============================================================
	sortedSpecs := e.sortShiftSpecsByDependency(shiftSpecs)

	// 拆分成多个单班次子任务
	subTasks := make([]*d_model.ProgressiveTask, 0, len(sortedSpecs))
	for _, spec := range sortedSpecs {
		subTask := &d_model.ProgressiveTask{
			ID:           fmt.Sprintf("%s_shift_%s", task.ID, spec.ShiftID),
			Order:        task.Order, // 保持原顺序
			Title:        task.Title,
			Description:  spec.Description,       // 使用解析出的班次专门任务说明
			TargetShifts: []string{spec.ShiftID}, // 只包含当前班次
			TargetDates:  task.TargetDates,
			TargetStaff:  task.TargetStaff,
			RuleIDs:      task.RuleIDs,
			Priority:     task.Priority,
			Status:       task.Status,
		}
		subTasks = append(subTasks, subTask)
	}

	return subTasks
}

// sortShiftSpecsByDependency 根据班次的排班优先级(SchedulingPriority)排序
// 优先级数字越小越先执行，确保被依赖的班次先完成排班
// 例如：上半夜班(优先级5) 先于 下夜班(优先级6)，保证下夜班能获取到上半夜班的排班数据
func (e *ProgressiveTaskExecutor) sortShiftSpecsByDependency(specs []ShiftTaskSpec) []ShiftTaskSpec {
	if len(specs) <= 1 {
		return specs
	}

	// 获取班次优先级信息
	shiftPriorityMap := make(map[string]int) // shiftID -> SchedulingPriority
	if e.taskContext != nil && len(e.taskContext.Shifts) > 0 {
		for _, shift := range e.taskContext.Shifts {
			if shift != nil {
				shiftPriorityMap[shift.ID] = shift.SchedulingPriority
			}
		}
	}

	// 按 SchedulingPriority 升序排序（数字越小越先执行，0为最高优先级）
	result := make([]ShiftTaskSpec, len(specs))
	copy(result, specs)
	sort.SliceStable(result, func(i, j int) bool {
		return shiftPriorityMap[result[i].ShiftID] < shiftPriorityMap[result[j].ShiftID]
	})

	return result
}

// copyShiftScheduleDraft 复制排班草案
func (e *ProgressiveTaskExecutor) copyShiftScheduleDraft(
	source *d_model.ShiftScheduleDraft,
) *d_model.ShiftScheduleDraft {
	draft := d_model.NewShiftScheduleDraft()
	if source != nil && source.Schedule != nil {
		draft.Schedule = make(map[string][]string)
		for date, staffIDs := range source.Schedule {
			draft.Schedule[date] = make([]string, len(staffIDs))
			copy(draft.Schedule[date], staffIDs)
		}
	}
	return draft
}

// mergeFixedShiftAssignments 合并所有班次的固定排班数据
// 输入: []d_model.CtxFixedShiftAssignment
// 输出: date -> staffIDs（所有班次合并，去重）
// 用于传递给校验器、ExecuteTodoTask 等需要合并格式的函数
func (e *ProgressiveTaskExecutor) mergeFixedShiftAssignments(
	fixedShiftAssignments []d_model.CtxFixedShiftAssignment,
) map[string][]string {
	result := make(map[string][]string)
	for _, assignment := range fixedShiftAssignments {
		// 合并同一天的固定排班人员（去重）
		existingStaffIDs := result[assignment.Date]
		staffIDMap := make(map[string]bool)
		for _, id := range existingStaffIDs {
			staffIDMap[id] = true
		}
		for _, id := range assignment.StaffIDs {
			if !staffIDMap[id] {
				result[assignment.Date] = append(result[assignment.Date], id)
				staffIDMap[id] = true
			}
		}
	}
	return result
}

// mergeSubTaskResult 合并子任务结果到最终草案
// 增强合并逻辑：包含冲突检测和人数上限检查
func (e *ProgressiveTaskExecutor) mergeSubTaskResult(
	finalDraft *d_model.ShiftScheduleDraft,
	subTaskDraft *d_model.ShiftScheduleDraft,
	shiftID string,
	staffRequirements map[string]int,
) error {
	if subTaskDraft == nil || subTaskDraft.Schedule == nil {
		return nil
	}

	conflictCount := 0
	exceedLimitCount := 0

	for date, staffIDs := range subTaskDraft.Schedule {
		// 检查人数上限
		if maxCount, ok := staffRequirements[date]; ok {
			existingCount := len(finalDraft.Schedule[date])
			if existingCount+len(staffIDs) > maxCount {
				exceedLimitCount++
				e.logger.Warn("Merged result exceeds staff requirement limit",
					"date", date,
					"existing", existingCount,
					"new", len(staffIDs),
					"total", existingCount+len(staffIDs),
					"max", maxCount,
					"shiftID", shiftID,
					"exceedBy", existingCount+len(staffIDs)-maxCount)
				// 记录警告，但不阻止合并（由上层逻辑决定是否截断）
				// 注意：这里允许超过上限，因为可能是多个子任务的结果合并，实际业务逻辑可能需要截断
			}
		}

		// 合并人员（去重）
		existingStaffIDs := finalDraft.Schedule[date]
		staffIDMap := make(map[string]bool)
		for _, id := range existingStaffIDs {
			staffIDMap[id] = true
		}

		conflictStaffIDs := make([]string, 0)
		for _, id := range staffIDs {
			if !staffIDMap[id] {
				if finalDraft.Schedule == nil {
					finalDraft.Schedule = make(map[string][]string)
				}
				finalDraft.Schedule[date] = append(finalDraft.Schedule[date], id)
				staffIDMap[id] = true
			} else {
				// 检测冲突：同一人员在同一天被安排多次（可能是多个班次）
				conflictCount++
				conflictStaffIDs = append(conflictStaffIDs, id)
				e.logger.Warn("Staff conflict detected: same staff scheduled multiple times on same date",
					"date", date,
					"staffID", id,
					"shiftID", shiftID,
					"reason", "可能是在多个班次中被安排，或子任务结果重复")
			}
		}

		// 如果有冲突，记录警告但继续合并
		if len(conflictStaffIDs) > 0 {
			e.logger.Warn("Staff conflicts detected during merge",
				"date", date,
				"conflictCount", len(conflictStaffIDs),
				"shiftID", shiftID)
		}
	}

	return nil
}

// getTaskRules 获取任务相关的规则
func (e *ProgressiveTaskExecutor) getTaskRules(task *d_model.ProgressiveTask, rules []*d_model.Rule) []*d_model.Rule {
	if len(task.RuleIDs) == 0 {
		return rules
	}

	ruleMap := make(map[string]*d_model.Rule)
	for _, rule := range rules {
		ruleMap[rule.ID] = rule
	}

	result := make([]*d_model.Rule, 0)
	for _, ruleID := range task.RuleIDs {
		if rule, ok := ruleMap[ruleID]; ok {
			result = append(result, rule)
		}
	}

	return result
}

// buildTaskStaffRequirements 构建任务相关的每日人数需求
func (e *ProgressiveTaskExecutor) buildTaskStaffRequirements(task *d_model.ProgressiveTask, staffRequirements map[string]map[string]int) map[string]int {
	result := make(map[string]int)

	// 如果任务指定了日期，只包含这些日期
	if len(task.TargetDates) > 0 {
		for _, date := range task.TargetDates {
			// 查找该日期在所有班次中的需求
			for shiftID, dates := range staffRequirements {
				if count, ok := dates[date]; ok {
					// 如果任务指定了班次，只包含这些班次
					if len(task.TargetShifts) == 0 || contains(task.TargetShifts, shiftID) {
						result[date] += count
					}
				}
			}
		}
	} else {
		// 如果没有指定日期，包含所有日期
		for shiftID, dates := range staffRequirements {
			// 如果任务指定了班次，只包含这些班次
			if len(task.TargetShifts) == 0 || contains(task.TargetShifts, shiftID) {
				for date, count := range dates {
					result[date] += count
				}
			}
		}
	}

	return result
}

// contains 检查字符串切片是否包含指定值
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// getWeekdayName 根据日期字符串获取中文星期名称
// dateStr 格式为 "YYYY-MM-DD"
func getWeekdayName(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	weekdayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return weekdayNames[t.Weekday()]
}

// formatDateWithWeekday 格式化日期并附带星期信息
// 返回格式如 "2026-02-09(周一)"
func formatDateWithWeekday(dateStr string) string {
	weekday := getWeekdayName(dateStr)
	if weekday == "" {
		return dateStr
	}
	return fmt.Sprintf("%s(%s)", dateStr, weekday)
}

// getPreviousDate 获取前一天的日期
// 返回格式如 "2026-02-08"
func getPreviousDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	previousDay := t.AddDate(0, 0, -1)
	return previousDay.Format("2006-01-02")
}

// extractJSON 从LLM响应中提取JSON内容
// 处理 ```json ... ``` 格式的代码块
func extractJSON(raw string) string {
	jsonStr := raw
	if strings.Contains(raw, "```") {
		lines := strings.Split(raw, "\n")
		var jsonLines []string
		inCodeBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}
			if inCodeBlock || (!strings.HasPrefix(trimmed, "```") && len(trimmed) > 0) {
				jsonLines = append(jsonLines, line)
			}
		}
		jsonStr = strings.Join(jsonLines, "\n")
	}
	return strings.TrimSpace(jsonStr)
}

// isDateInPersonalNeed 检查日期是否在个人需求的目标日期范围内
func isDateInPersonalNeed(targetDate string, need *d_model.PersonalNeed) bool {
	if need == nil {
		return false
	}
	// 如果 TargetDates 为空，表示整个周期都生效
	if len(need.TargetDates) == 0 {
		return true
	}
	// 检查日期是否在目标日期列表中
	for _, date := range need.TargetDates {
		if date == targetDate {
			return true
		}
	}
	return false
}

// extractDatesFromText 从文本中提取日期（YYYY-MM-DD格式）
func (e *ProgressiveTaskExecutor) extractDatesFromText(text string) []string {
	var dates []string
	// 简单匹配 YYYY-MM-DD 格式
	for i := 0; i <= len(text)-10; i++ {
		if text[i] >= '2' && text[i] <= '2' && // 2xxx年
			text[i+4] == '-' && text[i+7] == '-' {
			dateCandidate := text[i : i+10]
			// 验证格式
			if len(dateCandidate) == 10 {
				year := dateCandidate[0:4]
				month := dateCandidate[5:7]
				day := dateCandidate[8:10]
				if isNumeric(year) && isNumeric(month) && isNumeric(day) {
					dates = append(dates, dateCandidate)
				}
			}
		}
	}
	return dates
}

// isNumeric 检查字符串是否为数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isShiftTimeOverlap 检查两个班次的时间是否重叠（超过1小时）
// 用于判断同一天内两个班次是否存在时间冲突
func (e *ProgressiveTaskExecutor) isShiftTimeOverlap(shift1, shift2 *d_model.Shift) bool {
	if shift1 == nil || shift2 == nil {
		return false
	}

	// 解析时间（格式 HH:MM）
	parseTime := func(timeStr string) int {
		parts := strings.Split(timeStr, ":")
		if len(parts) != 2 {
			return 0
		}
		hour := 0
		min := 0
		fmt.Sscanf(parts[0], "%d", &hour)
		fmt.Sscanf(parts[1], "%d", &min)
		return hour*60 + min
	}

	start1 := parseTime(shift1.StartTime)
	end1 := parseTime(shift1.EndTime)
	start2 := parseTime(shift2.StartTime)
	end2 := parseTime(shift2.EndTime)

	// 处理结束时间为00:00的情况（当作24:00）
	if end1 == 0 {
		end1 = 24 * 60
	}
	if end2 == 0 {
		end2 = 24 * 60
	}

	// 计算重叠时间
	overlapStart := start1
	if start2 > start1 {
		overlapStart = start2
	}
	overlapEnd := end1
	if end2 < end1 {
		overlapEnd = end2
	}
	overlap := overlapEnd - overlapStart
	if overlap < 0 {
		overlap = 0
	}

	// 重叠超过60分钟才算冲突
	return overlap > 60
}

// computeCandidateStaffForShift 动态计算该班次的候选人员
// 过滤规则：只排除在所有目标日期都不可用的人员，部分日期请假的人员仍保留（由LLM判断具体日期）
func (e *ProgressiveTaskExecutor) computeCandidateStaffForShift(
	allStaff []*d_model.Employee,
	occupiedSlots map[string]map[string]string, // staffID -> date -> shiftID（不用于过滤，仅用于信息展示）
	personalNeeds map[string][]*d_model.PersonalNeed,
	targetDates []string,
) []*d_model.Employee {
	if len(targetDates) == 0 {
		// 没有指定日期，所有员工都是候选
		return allStaff
	}

	candidates := make([]*d_model.Employee, 0, len(allStaff))

	for _, staff := range allStaff {
		// ⚠️ 不再检查占位！已排班的人员仍然是候选人
		// 原因：
		// 1. 一天可以上多个班次（只要不超工时且时间不冲突）
		// 2. 连班规则需要从已排班人员中选择（如：下夜班必须前一天上夜班）
		// 3. 时间冲突由LLM根据"第六部分：候选人员排班情况"判断

		// 只检查是否在**所有**目标日期都有回避需求
		// 如果只是部分日期请假，仍保留该员工作为候选人，让LLM根据第七部分负向需求判断
		if hasAvoidanceOnAllDates(personalNeeds[staff.ID], targetDates) {
			continue // 所有日期都不可用，跳过
		}

		// 通过过滤，加入候选列表
		candidates = append(candidates, staff)
	}

	return candidates
}

// hasAvoidanceOnDates 检查员工是否在指定日期有回避需求（任意一天有回避即返回true）
func hasAvoidanceOnDates(needs []*d_model.PersonalNeed, dates []string) bool {
	if len(needs) == 0 {
		return false
	}

	for _, need := range needs {
		if need == nil {
			continue
		}

		// 检查是否是回避类型（avoid或must但未指定班次 = 要求休息）
		isAvoidance := need.RequestType == "avoid" ||
			(need.RequestType == "must" && need.TargetShiftID == "")

		if !isAvoidance {
			continue
		}

		// 检查日期是否重叠
		for _, targetDate := range need.TargetDates {
			for _, date := range dates {
				if targetDate == date {
					return true
				}
			}
		}
	}

	return false
}

// hasAvoidanceOnAllDates 检查员工是否在所有指定日期都有回避需求
// 只有当员工在每一个目标日期都不可用时才返回true
// 如果员工只是部分日期请假，返回false（保留为候选人，让LLM判断具体日期）
func hasAvoidanceOnAllDates(needs []*d_model.PersonalNeed, dates []string) bool {
	if len(needs) == 0 || len(dates) == 0 {
		return false
	}

	// 收集所有回避日期
	avoidanceDates := make(map[string]bool)
	for _, need := range needs {
		if need == nil {
			continue
		}

		// 检查是否是回避类型（avoid或must但未指定班次 = 要求休息）
		isAvoidance := need.RequestType == "avoid" ||
			(need.RequestType == "must" && need.TargetShiftID == "")

		if !isAvoidance {
			continue
		}

		// 收集回避日期
		for _, targetDate := range need.TargetDates {
			avoidanceDates[targetDate] = true
		}
	}

	// 检查是否所有目标日期都在回避日期中
	for _, date := range dates {
		if !avoidanceDates[date] {
			// 有一天可用，员工应保留为候选人
			return false
		}
	}

	// 所有日期都不可用
	return true
}

// filterRulesForShift 过滤与当前班次相关的规则（支持多维度关联）
// 规则关联有三种维度：shift（班次）、employee（员工）、group（分组）
// 过滤逻辑：
//   - 无任何关联 → 全局规则，始终包含
//   - 有班次关联 → 关联到当前班次才包含
//   - 有员工关联 → 关联的员工在当前班次候选人名单中才包含
//   - 混合关联（如同时有班次和员工关联）→ 任一维度匹配即包含
func (e *ProgressiveTaskExecutor) filterRulesForShift(rules []*d_model.Rule, shiftID string, shiftStaffList []*d_model.Employee) []*d_model.Rule {
	if len(rules) == 0 {
		return rules
	}

	// 构建当前班次候选人员ID集合（用于员工关联维度判断）
	staffIDSet := make(map[string]bool)
	for _, staff := range shiftStaffList {
		if staff != nil {
			staffIDSet[staff.ID] = true
		}
	}

	filtered := make([]*d_model.Rule, 0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		// ApplyScope == "global" → 全局规则，始终包含
		if rule.ApplyScope == "global" {
			filtered = append(filtered, rule)
			continue
		}

		// 非全局规则但无关联（数据异常兜底），保守包含
		if len(rule.Associations) == 0 {
			filtered = append(filtered, rule)
			continue
		}

		// 分类统计各维度的关联
		hasShiftAssoc := false
		hasEmployeeAssoc := false
		shiftMatched := false
		employeeMatched := false

		for _, assoc := range rule.Associations {
			switch assoc.AssociationType {
			case "shift":
				hasShiftAssoc = true
				if assoc.AssociationID == shiftID {
					shiftMatched = true
				}
			case "employee":
				hasEmployeeAssoc = true
				if len(staffIDSet) > 0 && staffIDSet[assoc.AssociationID] {
					employeeMatched = true
				}
				// group 关联暂不做额外过滤，保持向后兼容
			}
		}

		// 判断是否包含该规则
		should := false

		// 1. 班次维度匹配
		if shiftMatched {
			should = true
		}

		// 2. 员工维度匹配（仅当传入了 shiftStaffList 时才按员工过滤）
		if hasEmployeeAssoc {
			if len(shiftStaffList) > 0 {
				// 有候选人列表：关联的员工在候选人中才包含
				if employeeMatched {
					should = true
				}
			} else {
				// 未传候选人列表（向后兼容）：保守策略，包含员工规则
				should = true
			}
		}

		// 3. 只有 group 关联（无 shift 和 employee 关联）→ 保持包含（向后兼容）
		if !hasShiftAssoc && !hasEmployeeAssoc {
			should = true
		}

		if should {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BuildStaffNamesMap 构建人员ID到姓名的映射
func BuildStaffNamesMap(staffList []*d_model.Employee) map[string]string {
	namesMap := make(map[string]string)
	for _, staff := range staffList {
		if staff != nil && staff.ID != "" {
			namesMap[staff.ID] = staff.Name
		}
	}
	return namesMap
}

// MapIDsToNames 将ID列表转换为姓名列表
func MapIDsToNames(ids []string, namesMap map[string]string) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := namesMap[id]; ok {
			names = append(names, name)
		} else {
			names = append(names, id) // 如果找不到姓名，使用ID
		}
	}
	return names
}
