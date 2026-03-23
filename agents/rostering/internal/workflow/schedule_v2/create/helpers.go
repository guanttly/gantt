package create

import (
	"fmt"
	"sort"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 班次分类和排序
// ============================================================

// ClassifyShiftsByType 按类型分类班次
// 使用 Shift.Type 字段进行分类，自动处理前端类型映射，空值默认为 normal
func ClassifyShiftsByType(shifts []*d_model.Shift) map[string][]*d_model.Shift {
	result := make(map[string][]*d_model.Shift)

	for _, shift := range shifts {
		shiftType := shift.Type

		// 空值默认为普通班次
		if shiftType == "" {
			shiftType = ShiftTypeNormal
		} else if mappedType, ok := ShiftTypeFrontendMapping[shiftType]; ok {
			// 如果是前端类型，映射到工作流类型
			shiftType = mappedType
		}

		result[shiftType] = append(result[shiftType], shift)
	}

	return result
}

// SortShiftsBySchedulingPriority 按 SchedulingPriority 排序班次（升序）
// 优先级数字越小，优先级越高
func SortShiftsBySchedulingPriority(shifts []*d_model.Shift) []*d_model.Shift {
	if len(shifts) == 0 {
		return shifts
	}

	sorted := make([]*d_model.Shift, len(shifts))
	copy(sorted, shifts)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].SchedulingPriority < sorted[j].SchedulingPriority
	})

	return sorted
}

// GetShiftsByType 从分类结果中获取指定类型的班次
func GetShiftsByType(classified map[string][]*d_model.Shift, shiftType string) []*d_model.Shift {
	if shifts, ok := classified[shiftType]; ok {
		return shifts
	}
	return []*d_model.Shift{}
}

// ============================================================
// 个人需求提取
// ============================================================

// ExtractPersonalNeeds 从规则中提取个人需求
// 识别规则中与个人排班偏好相关的规则，转换为 PersonalNeed 结构
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
	needs := make(map[string][]*PersonalNeed)

	// 【关键修复】构建候选人员ID集合（用于过滤）
	candidateStaffIDs := make(map[string]bool)
	staffNameMap := make(map[string]string)
	for _, staff := range staffList {
		candidateStaffIDs[staff.ID] = true // ← 添加到候选集合
		staffNameMap[staff.ID] = staff.Name
	}

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		// 检查规则关联，找出员工级别的规则
		for _, assoc := range rule.Associations {
			if assoc.AssociationType != "employee" {
				continue
			}

			staffID := assoc.AssociationID

			// 【关键修复】只提取候选人员的需求，过滤掉非候选人员
			if !candidateStaffIDs[staffID] {
				continue
			}

			// 解析规则内容，提取个人需求
			need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
			if need != nil {
				if needs[need.StaffID] == nil {
					needs[need.StaffID] = make([]*PersonalNeed, 0)
				}
				needs[need.StaffID] = append(needs[need.StaffID], need)
			}
		}
	}

	return needs
}

// parseRuleToPersonalNeed 解析规则为个人需求
// 根据规则类型和内容，转换为标准化的个人需求结构
func parseRuleToPersonalNeed(rule *d_model.Rule, staffID string, staffNameMap map[string]string) *PersonalNeed {
	if staffID == "" {
		return nil
	}

	staffName := staffNameMap[staffID]
	if staffName == "" {
		staffName = "未知员工"
	}

	// 根据规则类型判断需求类型
	needType := "permanent" // 默认常态化
	// TODO: 根据 rule.TimeScope 或 rule.ValidFrom/ValidTo 判断是否为临时需求
	if rule.ValidFrom != nil && rule.ValidTo != nil {
		needType = "temporary"
	}

	// 根据规则优先级判断请求类型
	requestType := "prefer" // 默认偏好
	if rule.Priority <= 3 {
		requestType = "must" // 高优先级视为必须
	}

	// 规则类型中文名称映射
	ruleTypeNames := map[string]string{
		"max_shifts":         "最大班次数",
		"consecutive_shifts": "连续班次",
		"rest_days":          "休息日",
		"forbidden_pattern":  "禁止模式",
		"preferred_pattern":  "偏好模式",
		"exclusive":          "排他规则",
		"combinable":         "可合并规则",
		"required_together":  "必须同时规则",
		"periodic":           "周期性规则",
		"maxCount":           "最大次数规则",
		"forbidden_day":      "禁止日期规则",
		"preferred":          "偏好规则",
	}

	// 获取规则类型中文名称
	ruleTypeName := ruleTypeNames[rule.RuleType]
	if ruleTypeName == "" {
		ruleTypeName = rule.RuleType // 如果找不到映射，使用原始值
	}

	// 构建需求描述
	// 优先级：Description > RuleData > Name
	description := rule.Description
	if description == "" {
		// 优先使用 RuleData（规则说明文本）
		if rule.RuleData != "" {
			description = rule.RuleData
		} else {
			// 如果 RuleData 也为空，使用规则名称
			description = rule.Name
		}
	}

	// 如果描述中没有包含类型信息，则添加类型中文名称
	if rule.RuleType != "" && !contains(description, ruleTypeName) && !contains(description, rule.RuleType) {
		description = fmt.Sprintf("%s（类型：%s）", description, ruleTypeName)
	}

	need := &PersonalNeed{
		StaffID:     staffID,
		StaffName:   staffName,
		NeedType:    needType,
		RequestType: requestType,
		Description: description,
		Priority:    rule.Priority,
		RuleID:      rule.ID,
		Source:      "rule",
		Confirmed:   false,
	}

	// TODO: 根据规则的 RuleType 和 RuleData 解析具体的班次和日期要求
	// 这部分需要根据实际规则格式进行定制

	return need
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================
// 已占位信息管理
// ============================================================

// BuildOccupiedSlotsMap 构建已占位映射
// 从现有排班结果中提取已占位信息
func BuildOccupiedSlotsMap(scheduleDrafts map[string]*d_model.ShiftScheduleDraft) map[string]map[string]string {
	occupied := make(map[string]map[string]string)

	for shiftID, draft := range scheduleDrafts {
		if draft == nil || draft.Schedule == nil {
			continue
		}

		for date, staffIDs := range draft.Schedule {
			if staffIDs == nil {
				continue
			}

			for _, staffID := range staffIDs {
				if occupied[staffID] == nil {
					occupied[staffID] = make(map[string]string)
				}
				occupied[staffID][date] = shiftID
			}
		}
	}

	return occupied
}

// MergeOccupiedSlots 合并已占位信息
// 将新的排班结果合并到现有的占位映射中
func MergeOccupiedSlots(existing map[string]map[string]string, newDraft *d_model.ShiftScheduleDraft, shiftID string) {
	if newDraft == nil || newDraft.Schedule == nil {
		return
	}

	for date, staffIDs := range newDraft.Schedule {
		if staffIDs == nil {
			continue
		}

		for _, staffID := range staffIDs {
			if existing[staffID] == nil {
				existing[staffID] = make(map[string]string)
			}
			existing[staffID][date] = shiftID
		}
	}
}

// BuildExistingScheduleMarks 构建已有排班标记
// 用于时段冲突检查
func BuildExistingScheduleMarks(scheduleDrafts map[string]*d_model.ShiftScheduleDraft, shifts map[string]*d_model.Shift) map[string]map[string][]*d_model.ShiftMark {
	marks := make(map[string]map[string][]*d_model.ShiftMark)

	for shiftID, draft := range scheduleDrafts {
		if draft == nil || draft.Schedule == nil {
			continue
		}

		shift := shifts[shiftID]
		if shift == nil {
			continue
		}

		for date, staffIDs := range draft.Schedule {
			if staffIDs == nil {
				continue
			}

			mark := &d_model.ShiftMark{
				ShiftID:   shiftID,
				ShiftName: shift.Name,
				StartTime: shift.StartTime,
				EndTime:   shift.EndTime,
			}

			for _, staffID := range staffIDs {
				if marks[staffID] == nil {
					marks[staffID] = make(map[string][]*d_model.ShiftMark)
				}
				if marks[staffID][date] == nil {
					marks[staffID][date] = make([]*d_model.ShiftMark, 0)
				}
				marks[staffID][date] = append(marks[staffID][date], mark)
			}
		}
	}

	return marks
}

// ============================================================
// 人员排班检测
// ============================================================

// DetectUnderScheduledStaff 检测排班不足的人员
// 分析每个人员的排班情况，找出未达标的人员
func DetectUnderScheduledStaff(
	staffList []*d_model.Employee,
	occupiedSlots map[string]map[string]string,
	startDate, endDate string,
	minWorkDays int, // 最小工作天数要求
) []*UnderScheduledStaff {
	result := make([]*UnderScheduledStaff, 0)

	// 计算排班周期的总天数
	totalDays := calculateDaysBetween(startDate, endDate)

	for _, staff := range staffList {
		staffID := staff.ID
		scheduledDays := 0

		// 统计该员工的排班天数
		if staffDates, ok := occupiedSlots[staffID]; ok {
			scheduledDays = len(staffDates)
		}

		// 判断是否未达标
		if scheduledDays < minWorkDays {
			result = append(result, &UnderScheduledStaff{
				StaffID:         staffID,
				StaffName:       staff.Name,
				ScheduledDays:   scheduledDays,
				RequiredDays:    minWorkDays,
				MissingDays:     minWorkDays - scheduledDays,
				TotalPeriodDays: totalDays,
			})
		}
	}

	return result
}

// UnderScheduledStaff 排班不足的人员信息
type UnderScheduledStaff struct {
	StaffID         string `json:"staffId"`
	StaffName       string `json:"staffName"`
	ScheduledDays   int    `json:"scheduledDays"`   // 已排班天数
	RequiredDays    int    `json:"requiredDays"`    // 要求天数
	MissingDays     int    `json:"missingDays"`     // 缺少天数
	TotalPeriodDays int    `json:"totalPeriodDays"` // 周期总天数
}

// calculateDaysBetween 计算两个日期之间的天数（包含起止日期）
func calculateDaysBetween(startDate, endDate string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}

// ============================================================
// 子工作流调用辅助
// ============================================================

// SpawnCoreSubWorkflow 调用 Core 子工作流进行单个班次排班
// 封装子工作流调用逻辑，简化 Actions 代码
func SpawnCoreSubWorkflow(
	wctx engine.Context,
	shift *d_model.Shift,
	shiftCtx *d_model.ShiftSchedulingContext,
	onComplete, onError engine.Event,
) error {
	// TODO: 实现子工作流调用
	// 当前暂时不实现，待引擎支持子工作流功能后补充
	return fmt.Errorf("SpawnCoreSubWorkflow not implemented yet")
}

// ============================================================
// 统计和汇总
// ============================================================

// CalculatePhaseStatistics 计算阶段统计信息
func CalculatePhaseStatistics(phaseResult *PhaseResult) *PhaseStatistics {
	if phaseResult == nil {
		return nil
	}

	totalStaffAssigned := 0
	for _, draft := range phaseResult.ScheduleDrafts {
		if draft == nil || draft.Schedule == nil {
			continue
		}
		for _, staffIDs := range draft.Schedule {
			if staffIDs != nil {
				totalStaffAssigned += len(staffIDs)
			}
		}
	}

	avgTime := 0.0
	totalShifts := phaseResult.CompletedCount + phaseResult.SkippedCount + phaseResult.FailedCount
	if totalShifts > 0 && phaseResult.Duration > 0 {
		avgTime = phaseResult.Duration / float64(totalShifts)
	}

	return &PhaseStatistics{
		PhaseName:           phaseResult.PhaseName,
		TotalShifts:         totalShifts,
		CompletedShifts:     phaseResult.CompletedCount,
		SkippedShifts:       phaseResult.SkippedCount,
		FailedShifts:        phaseResult.FailedCount,
		TotalStaffAssigned:  totalStaffAssigned,
		AverageTimePerShift: avgTime,
	}
}

// MergeScheduleDrafts 合并多个阶段的排班草案为最终草案
func MergeScheduleDrafts(
	startDate, endDate string,
	phaseResults []*PhaseResult,
	staffList []*d_model.Employee,
) *d_model.ScheduleDraft {
	finalDraft := &d_model.ScheduleDraft{
		StartDate:  startDate,
		EndDate:    endDate,
		Shifts:     make(map[string]*d_model.ShiftDraft),
		StaffStats: make(map[string]*d_model.StaffStats),
		Conflicts:  make([]*d_model.ScheduleConflict, 0),
	}

	// 合并所有阶段的班次排班
	for _, phaseResult := range phaseResults {
		if phaseResult == nil || phaseResult.ScheduleDrafts == nil {
			continue
		}

		for shiftID, shiftScheduleDraft := range phaseResult.ScheduleDrafts {
			if shiftScheduleDraft == nil || shiftScheduleDraft.Schedule == nil {
				continue
			}

			// 转换为 ShiftDraft 格式
			// 将 ShiftScheduleDraft.Schedule (map[string][]string) 转换为 ShiftDraft.Days (map[string]*DayShift)
			days := make(map[string]*d_model.DayShift)
			for date, staffIDs := range shiftScheduleDraft.Schedule {
				days[date] = &d_model.DayShift{
					StaffIDs:      staffIDs,
					ActualCount:   len(staffIDs),
					RequiredCount: len(staffIDs), // TODO: 从需求数据获取
				}
			}

			finalDraft.Shifts[shiftID] = &d_model.ShiftDraft{
				ShiftID:  shiftID,
				Priority: 0, // 可以从 Shift 对象获取
				Days:     days,
			}
		}
	}

	// 计算人员统计
	calculateStaffStats(finalDraft, staffList)

	// TODO: 检测冲突
	// detectConflicts(finalDraft)

	return finalDraft
}

// calculateStaffStats 计算人员统计信息
func calculateStaffStats(draft *d_model.ScheduleDraft, staffList []*d_model.Employee) {
	staffWorkDays := make(map[string]int)
	staffShifts := make(map[string][]string)

	// 遍历所有班次和日期，统计每个人员的工作情况
	for shiftID, shiftDraft := range draft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		for date, dayShift := range shiftDraft.Days {
			if dayShift == nil || dayShift.StaffIDs == nil {
				continue
			}

			for _, staffID := range dayShift.StaffIDs {
				staffWorkDays[staffID]++
				if staffShifts[staffID] == nil {
					staffShifts[staffID] = make([]string, 0)
				}
				staffShifts[staffID] = append(staffShifts[staffID], fmt.Sprintf("%s:%s", date, shiftID))
			}
		}
	}

	// 创建 StaffStats 对象
	staffNameMap := make(map[string]string)
	for _, staff := range staffList {
		staffNameMap[staff.ID] = staff.Name
	}

	for staffID, workDays := range staffWorkDays {
		draft.StaffStats[staffID] = &d_model.StaffStats{
			StaffID:   staffID,
			StaffName: staffNameMap[staffID],
			WorkDays:  workDays,
			Shifts:    staffShifts[staffID],
			// TODO: 计算总工作小时数
			TotalHours: 0,
		}
	}
}

// ============================================================
// 日期和时间辅助
// ============================================================

// GenerateDateList 生成日期列表（包含起止日期）
func GenerateDateList(startDate, endDate string) ([]string, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	dates := make([]string, 0)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	return dates, nil
}
