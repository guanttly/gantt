package common

import (
	"fmt"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// getDefaultNextWeekRange 获取默认的下周排班周期（周一到周日）
func GetDefaultNextWeekRange() (string, string, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysUntilNextMonday := 8 - weekday
	nextMonday := now.AddDate(0, 0, daysUntilNextMonday)
	nextSunday := nextMonday.AddDate(0, 0, 6) // 周一+6天=周日

	return nextMonday.Format("2006-01-02"), nextSunday.Format("2006-01-02"), nil
}

// formatDateRangeForDisplay 格式化日期范围用于显示
func FormatDateRangeForDisplay(startDate, endDate string) string {
	start, err1 := time.Parse("2006-01-02", startDate)
	end, err2 := time.Parse("2006-01-02", endDate)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s 至 %s", startDate, endDate)
	}

	// 格式化为中文日期
	return fmt.Sprintf("%d年%d月%d日 至 %d年%d月%d日",
		start.Year(), start.Month(), start.Day(),
		end.Year(), end.Month(), end.Day())
}

// ============================================================
// 原有验证函数
// ============================================================

// validateDateRange 验证日期范围
func ValidateDateRange(startDate, endDate string) error {
	// 解析日期
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid end date format: %w", err)
	}

	// 验证规则
	today := time.Now().Truncate(24 * time.Hour)
	minDate := today.AddDate(0, 0, -30) // 今天 - 30天
	maxDate := start.AddDate(0, 0, 90)  // 开始日期 + 90天

	if start.Before(minDate) {
		return fmt.Errorf("start date must be >= today - 30 days")
	}
	if end.After(maxDate) {
		return fmt.Errorf("end date must be <= start date + 90 days")
	}
	if start.After(end) {
		return fmt.Errorf("start date must be <= end date")
	}

	return nil
}

// generateDateRange 生成日期范围数组（包含起止日期）
func GenerateDateRange(startDate, endDate string) []string {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return []string{}
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return []string{}
	}

	dates := make([]string, 0)
	current := start
	for !current.After(end) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}
	return dates
}

// DateWithTime 日期及其时间信息
type DateWithTime struct {
	DateStr string    // 日期字符串 "2006-01-02"
	Time    time.Time // 完整时间对象
}

// GenerateDateRangeWithTime 生成日期范围数组（包含时间对象，用于获取星期几等信息）
func GenerateDateRangeWithTime(startDate, endDate string) []DateWithTime {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return []DateWithTime{}
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return []DateWithTime{}
	}

	dates := make([]DateWithTime, 0)
	current := start
	for !current.After(end) {
		dates = append(dates, DateWithTime{
			DateStr: current.Format("2006-01-02"),
			Time:    current,
		})
		current = current.AddDate(0, 0, 1)
	}
	return dates
}

// getOrCreateScheduleContext 获取或创建排班上下文
func GetOrCreateScheduleContext(sess *session.Session) *d_model.ScheduleCreateContext {
	if ctx, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
		if scheduleCtx, ok := ctx.(*d_model.ScheduleCreateContext); ok {
			return scheduleCtx
		}
	}

	// 创建新的上下文
	return &d_model.ScheduleCreateContext{
		ShiftStaffRequirements: make(map[string]map[string]int),
		ShiftRules:             make(map[string][]*d_model.Rule),
		GroupRules:             make(map[string][]*d_model.Rule),
		EmployeeRules:          make(map[string][]*d_model.Rule),
		ScheduledStaffSet:      make(map[string]bool),                            // 保留兼容
		StaffScheduleMarks:     make(map[string]map[string][]*d_model.ShiftMark), // 新增：人员已排班标记
		AISummaries:            []string{},
		ShiftTodoPlans:         make(map[string]*d_model.ShiftTodoPlan), // 新增：三阶段排班
		TodoExecutionLogs:      []string{},                              // 新增：执行日志
		ValidationAttempts:     make(map[string]int),                    // 新增：校验次数（每个班次独立计数）
		ShiftStaffIDs:          make(map[string][]string),               // 添加缺失的字段
		StaffLeaves:            make(map[string][]*d_model.LeaveRecord), // 添加缺失的字段
	}
}

// sortShiftsByPriority 按优先级排序班次（从低到高）
func SortShiftsByPriority(shifts []*d_model.Shift) {
	// 使用冒泡排序（简单实现）
	for i := 0; i < len(shifts); i++ {
		for j := i + 1; j < len(shifts); j++ {
			if shifts[i].SchedulingPriority > shifts[j].SchedulingPriority {
				shifts[i], shifts[j] = shifts[j], shifts[i]
			}
		}
	}
}

// initializeStaffRequirements 初始化班次人数需求配置
func InitializeStaffRequirements(ctx *d_model.ScheduleCreateContext) error {
	if ctx.ShiftStaffRequirements == nil {
		ctx.ShiftStaffRequirements = make(map[string]map[string]int)
	}

	// 解析日期范围
	startDate, err := time.Parse("2006-01-02", ctx.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", ctx.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	// 为每个班次的每一天初始化默认人数
	for _, shift := range ctx.SelectedShifts {
		if _, ok := ctx.ShiftStaffRequirements[shift.ID]; !ok {
			ctx.ShiftStaffRequirements[shift.ID] = make(map[string]int)
		}

		// 遍历日期范围
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			// 默认人数为0，后续由工作流在选择班次后自动计算或手动配置
			ctx.ShiftStaffRequirements[shift.ID][dateStr] = 0
		}
	}

	return nil
}

// InitializeStaffRequirementsWithWeeklyConfig 使用周人数配置初始化班次人数需求
// weeklyConfigs: map[shiftID] -> map[weekday(0-6)] -> staffCount
func InitializeStaffRequirementsWithWeeklyConfig(ctx *d_model.ScheduleCreateContext, weeklyConfigs map[string]map[int]int) error {
	if ctx.ShiftStaffRequirements == nil {
		ctx.ShiftStaffRequirements = make(map[string]map[string]int)
	}

	// 解析日期范围
	startDate, err := time.Parse("2006-01-02", ctx.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", ctx.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	// 为每个班次的每一天设置人数（根据星期几从周配置中获取）
	for _, shift := range ctx.SelectedShifts {
		if _, ok := ctx.ShiftStaffRequirements[shift.ID]; !ok {
			ctx.ShiftStaffRequirements[shift.ID] = make(map[string]int)
		}

		// 获取该班次的周配置
		shiftWeeklyConfig := weeklyConfigs[shift.ID]

		// 遍历日期范围（仅添加需要排班的日期，staffCount > 0）
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			weekday := int(d.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday

			// 从周配置获取人数
			staffCount := 0
			if shiftWeeklyConfig != nil {
				if count, ok := shiftWeeklyConfig[weekday]; ok {
					staffCount = count
				}
			}

			// 仅添加需要排班的日期（staffCount > 0）
			if staffCount > 0 {
				ctx.ShiftStaffRequirements[shift.ID][dateStr] = staffCount
			}
		}
	}

	return nil
}

// filterRulesForShift 过滤出真正适用于指定班次和人员的规则
// 检查规则的 Associations 数组：
// 1. 没有任何关联的规则 -> 全局规则，保留
// 2. 有班次关联但不包含当前班次 -> 过滤掉
// 3. 有班次关联且包含当前班次 -> 保留
// 4. 只有人员或分组关联（没有班次关联）-> 保留（人员和分组规则适用于所有班次）
func FilterRulesForShift(rules []*d_model.Rule, shiftID string, staffIDs []string, groupIDs []string) []*d_model.Rule {
	filtered := make([]*d_model.Rule, 0)

	// 构建快速查找集合
	staffIDSet := make(map[string]bool)
	for _, id := range staffIDs {
		staffIDSet[id] = true
	}
	groupIDSet := make(map[string]bool)
	for _, id := range groupIDs {
		groupIDSet[id] = true
	}

	for _, rule := range rules {
		// 如果规则没有 Associations，认为是全局规则，保留
		if len(rule.Associations) == 0 {
			filtered = append(filtered, rule)
			continue
		}

		// 检查 Associations 中的关联类型
		hasShiftAssociation := false
		matchesShift := false
		hasRelevantAssociation := false

		for _, assoc := range rule.Associations {
			switch assoc.AssociationType {
			case "shift":
				hasShiftAssociation = true
				if assoc.AssociationID == shiftID {
					matchesShift = true
				}
			case "employee":
				// 检查是否是当前可用人员
				if staffIDSet[assoc.AssociationID] {
					hasRelevantAssociation = true
				}
			case "group":
				// 检查是否是当前人员所属的分组
				if groupIDSet[assoc.AssociationID] {
					hasRelevantAssociation = true
				}
			}
		}

		// 决策逻辑：
		// - 如果有班次关联，必须匹配当前班次才保留
		// - 如果没有班次关联但有人员/分组关联，只要相关就保留
		shouldKeep := false
		if hasShiftAssociation {
			// 有班次关联，必须匹配
			shouldKeep = matchesShift
		} else {
			// 没有班次关联，只要有相关的人员或分组关联就保留
			shouldKeep = hasRelevantAssociation
		}

		if shouldKeep {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}

// convertRuleToMap 转换规则为 map
func ConvertRuleToMap(rule *d_model.Rule) map[string]any {
	m := map[string]any{
		"id":          rule.ID,
		"name":        rule.Name,
		"ruleType":    rule.RuleType,
		"description": rule.Description,
		"ruleData":    rule.RuleData,
		"applyScope":  rule.ApplyScope,
		"timeScope":   rule.TimeScope,
		"priority":    rule.Priority,
	}

	// 只添加非空的数值参数
	if rule.MaxCount != nil {
		m["maxCount"] = *rule.MaxCount
	}
	if rule.ConsecutiveMax != nil {
		m["consecutiveMax"] = *rule.ConsecutiveMax
	}
	if rule.IntervalDays != nil {
		m["intervalDays"] = *rule.IntervalDays
	}
	if rule.MinRestDays != nil {
		m["minRestDays"] = *rule.MinRestDays
	}

	return m
}

// mergeShiftScheduleToDraft 合并班次排班到总草案
func MergeShiftScheduleToDraft(
	ctx *d_model.ScheduleCreateContext,
	shift *d_model.Shift,
	aiResult map[string]any,
	logger any,
) error {
	if ctx.DraftSchedule == nil {
		return fmt.Errorf("draft schedule not initialized")
	}

	// 解析 AI 输出的 schedule 字段
	schedule, ok := aiResult["schedule"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid AI result: missing schedule field")
	}

	// 创建班次草案
	shiftDraft := &d_model.ShiftDraft{
		ShiftID:  shift.ID,
		Priority: shift.SchedulingPriority,
		Days:     make(map[string]*d_model.DayShift),
	}

	// 遍历每一天的排班
	for dateStr, staffsAny := range schedule {
		staffNames := []string{}
		staffIDs := []string{}
		invalidCount := 0

		// 解析人员列表（支持 []any 和 []string 两种类型）
		var staffIdentifiers []string
		switch staffs := staffsAny.(type) {
		case []any:
			staffIdentifiers = make([]string, 0, len(staffs))
			for _, staffAny := range staffs {
				if sid, ok := staffAny.(string); ok {
					staffIdentifiers = append(staffIdentifiers, sid)
				}
			}
		case []string:
			staffIdentifiers = staffs
		default:
			if logger != nil {
				if l, ok := logger.(interface{ Warn(string, ...any) }); ok {
					l.Warn("Unknown staff list type in schedule",
						"date", dateStr,
						"type", fmt.Sprintf("%T", staffsAny))
				}
			}
			continue
		}

		for _, staffIdentifier := range staffIdentifiers {
			// 验证是否为有效的员工ID（在StaffList中存在）
			found := false
			for _, s := range ctx.StaffList {
				if s.ID == staffIdentifier {
					staffIDs = append(staffIDs, s.ID)
					staffNames = append(staffNames, s.Name)
					found = true
					break
				}
			}

			if !found {
				// 无效的ID，记录警告并忽略
				invalidCount++
				if logger != nil {
					if l, ok := logger.(interface{ Warn(string, ...any) }); ok {
						l.Warn("Invalid staff identifier from AI, ignoring",
							"identifier", staffIdentifier,
							"date", dateStr,
							"shift", shift.Name)
					}
				}
			}
		}

		// 如果有无效数据，记录信息
		if invalidCount > 0 && logger != nil {
			if l, ok := logger.(interface{ Info(string, ...any) }); ok {
				l.Info("Filtered invalid staff identifiers",
					"date", dateStr,
					"shift", shift.Name,
					"invalidCount", invalidCount,
					"validCount", len(staffIDs))
			}
		}

		// 获取需求人数
		requiredCount := 0
		if reqs, ok := ctx.ShiftStaffRequirements[shift.ID]; ok {
			if count, ok := reqs[dateStr]; ok {
				requiredCount = count
			}
		}

		shiftDraft.Days[dateStr] = &d_model.DayShift{
			Staff:         staffNames,
			StaffIDs:      staffIDs,
			RequiredCount: requiredCount,
			ActualCount:   len(staffIDs),
		}
	}

	// 添加到总草案 (使用 ShiftID 作为 Key)
	ctx.DraftSchedule.Shifts[shift.ID] = shiftDraft

	return nil
}

// generateStaffStats 生成人员统计
func GenerateStaffStats(ctx *d_model.ScheduleCreateContext, logger any) {
	if ctx.DraftSchedule == nil {
		return
	}

	staffStats := make(map[string]*d_model.StaffStats)

	// 遍历所有班次的所有天
	for shiftName, shiftDraft := range ctx.DraftSchedule.Shifts {
		for _, dayShift := range shiftDraft.Days {
			for _, staffID := range dayShift.StaffIDs {
				if _, ok := staffStats[staffID]; !ok {
					// 查找人员姓名
					staffName := ""
					for _, s := range ctx.StaffList {
						if s.ID == staffID {
							staffName = s.Name
							break
						}
					}

					staffStats[staffID] = &d_model.StaffStats{
						StaffID:    staffID,
						StaffName:  staffName,
						WorkDays:   0,
						Shifts:     []string{},
						TotalHours: 0,
					}
				}

				stats := staffStats[staffID]
				stats.WorkDays++
				stats.Shifts = append(stats.Shifts, shiftName)
				// TODO: 计算总工作小时数（需要班次时长信息）
				stats.TotalHours += 8 // 假设每班8小时
			}
		}
	}

	ctx.DraftSchedule.StaffStats = staffStats
}

// detectScheduleConflicts 检测排班冲突
func DetectScheduleConflicts(ctx *d_model.ScheduleCreateContext, logger any) {
	if ctx.DraftSchedule == nil {
		return
	}

	conflicts := []*d_model.ScheduleConflict{}

	// 检测人数不足
	for shiftName, shiftDraft := range ctx.DraftSchedule.Shifts {
		for dateStr, dayShift := range shiftDraft.Days {
			if dayShift.ActualCount < dayShift.RequiredCount {
				conflicts = append(conflicts, &d_model.ScheduleConflict{
					Date:     dateStr,
					Shift:    shiftName,
					Issue:    fmt.Sprintf("人数不足：需要%d人，实际%d人", dayShift.RequiredCount, dayShift.ActualCount),
					Severity: "warning",
				})
			}
		}
	}

	// TODO: 检测规则冲突（连续工作天数、休息时间等）

	ctx.DraftSchedule.Conflicts = conflicts
}

// convertDraftToScheduleRequests 将排班草案转换为SDK请求格式
func ConvertDraftToScheduleRequests(
	scheduleCtx *d_model.ScheduleCreateContext,
	orgID string,
	logger logging.ILogger,
) ([]d_model.ScheduleUpsertRequest, error) {
	if scheduleCtx.FinalSchedule == nil {
		return nil, fmt.Errorf("no final schedule found")
	}

	items := []d_model.ScheduleUpsertRequest{}

	// 遍历所有班次（注意：Shifts 的 key 是 ShiftID）
	for shiftID, shiftDraft := range scheduleCtx.FinalSchedule.Shifts {
		// 查找对应的Shift对象，获取ShiftCode等信息
		var shift *d_model.Shift
		for _, s := range scheduleCtx.SelectedShifts {
			if s.ID == shiftID {
				shift = s
				break
			}
		}
		if shift == nil {
			logger.Warn("Shift not found in context", "shiftID", shiftID)
			continue
		}

		// 遍历每一天
		for date, dayShift := range shiftDraft.Days {
			// 为每个员工创建排班记录
			for _, staffID := range dayShift.StaffIDs {
				items = append(items, d_model.ScheduleUpsertRequest{
					UserID:    staffID,
					WorkDate:  date,
					ShiftCode: shift.ID, // 实际传递 ShiftID，字段名虽然叫 ShiftCode
					StartTime: shift.StartTime,
					EndTime:   shift.EndTime,
					OrgID:     orgID,
					Status:    "active", // 默认状态为激活
				})
			}
		}
	}

	logger.Info("Converted schedule draft",
		"shifts", len(scheduleCtx.FinalSchedule.Shifts),
		"totalRecords", len(items))

	return items, nil
}

// buildDateWeekdayMap 构建日期到星期几的映射
// 返回格式: {"2024-01-08": "周一", "2024-01-09": "周二", ...}
func BuildDateWeekdayMap(startDate, endDate string) map[string]string {
	dateWeekdayMap := make(map[string]string)

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return dateWeekdayMap
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return dateWeekdayMap
	}

	// 中文星期几映射
	weekdayNames := map[time.Weekday]string{
		time.Sunday:    "周日",
		time.Monday:    "周一",
		time.Tuesday:   "周二",
		time.Wednesday: "周三",
		time.Thursday:  "周四",
		time.Friday:    "周五",
		time.Saturday:  "周六",
	}

	// 遍历日期范围
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		weekday := d.Weekday()
		dateWeekdayMap[dateStr] = weekdayNames[weekday]
	}

	return dateWeekdayMap
}

// convertDraftForAI 将排班草案转换为AI可理解的格式
func ConvertDraftForAI(ctx *d_model.ScheduleCreateContext) map[string]any {
	if ctx.DraftSchedule == nil || len(ctx.DraftSchedule.Shifts) == 0 {
		return nil
	}

	result := make(map[string]any)

	// 转换班次排班数据
	shifts := make(map[string]any)
	for shiftID, shiftDraft := range ctx.DraftSchedule.Shifts {
		days := make(map[string]any)
		for date, dayShift := range shiftDraft.Days {
			days[date] = map[string]any{
				"staffIds":      dayShift.StaffIDs,
				"staff":         dayShift.Staff,
				"requiredCount": dayShift.RequiredCount,
				"actualCount":   dayShift.ActualCount,
			}
		}
		shifts[shiftID] = map[string]any{
			"shiftId":  shiftDraft.ShiftID,
			"priority": shiftDraft.Priority,
			"days":     days,
		}
	}
	result["shifts"] = shifts

	// 添加班次基本信息（供查找班次名称）
	if len(ctx.SelectedShifts) > 0 {
		allShifts := make([]any, 0, len(ctx.SelectedShifts))
		for _, shift := range ctx.SelectedShifts {
			allShifts = append(allShifts, map[string]any{
				"id":   shift.ID,
				"name": shift.Name,
			})
		}
		result["scheduleInfo"] = map[string]any{
			"allShifts": allShifts,
		}
	}

	return result
}
