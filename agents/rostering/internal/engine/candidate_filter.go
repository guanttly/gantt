package engine

import (
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
	"strconv"
	"strings"
	"time"
)

// CandidateFilter 候选人过滤器（替代 LLM-1）
type CandidateFilter struct {
	logger logging.ILogger
}

// NewCandidateFilter 创建候选人过滤器
func NewCandidateFilter(logger logging.ILogger) *CandidateFilter {
	return &CandidateFilter{logger: logger}
}

// Filter 过滤候选人
func (f *CandidateFilter) Filter(
	allStaff []*model.Staff,
	personalNeeds []*model.PersonalNeed,
	fixedAssignments []*model.CtxFixedShiftAssignment,
	currentDraft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
	allShifts []*model.Shift,
	targetShift *model.Shift,
) ([]*model.Staff, map[string]string) {
	candidates := make([]*model.Staff, 0)
	exclusionReasons := make(map[string]string)

	// 构建班次ID到班次信息的映射（用于时间重叠检查）
	shiftMap := make(map[string]*model.Shift)
	for _, shift := range allShifts {
		shiftMap[shift.ID] = shift
	}
	// 如果目标班次不在allShifts中，添加它
	if targetShift != nil {
		shiftMap[targetShift.ID] = targetShift
	}

	for _, staff := range allStaff {
		// 1. 检查请假
		if f.hasLeaveOnDate(staff.ID, personalNeeds, date) {
			exclusionReasons[staff.ID] = "该日期请假"
			continue
		}

		// 2. 检查固定排班（已排其他班次）
		if f.hasFixedAssignmentOnDate(staff.ID, fixedAssignments, shiftID, date) {
			exclusionReasons[staff.ID] = "该日期已有固定排班"
			continue
		}

		// 3. 检查员工是否已在同一天被安排到时间重叠的其他班次（基于 CurrentDraft）
		if conflictShiftID := f.findTimeOverlappingShiftOnDate(staff.ID, currentDraft, shiftID, date, shiftMap, targetShift); conflictShiftID != "" {
			conflictShiftName := conflictShiftID
			if s, ok := shiftMap[conflictShiftID]; ok && s != nil && s.Name != "" {
				conflictShiftName = s.Name + "(" + conflictShiftID[:8] + ")"
			}
			exclusionReasons[staff.ID] = "该日期已安排时间重叠的其他班次: " + conflictShiftName
			continue
		}

		candidates = append(candidates, staff)
	}

	return candidates, exclusionReasons
}

// hasLeaveOnDate 检查员工在指定日期是否请假（仅处理 avoid 类型）
// 注意：must 类型为"强制排班"诉求，不属于请假，不在此处处理
func (f *CandidateFilter) hasLeaveOnDate(
	staffID string,
	personalNeeds []*model.PersonalNeed,
	date time.Time,
) bool {
	dateStr := date.Format("2006-01-02")
	for _, need := range personalNeeds {
		if need.StaffID != staffID {
			continue
		}
		// 只处理 avoid（回避/请假）类型，must 为强制排班不应在此排除
		if need.RequestType != "avoid" {
			continue
		}
		// 如果指定了 TargetDates，只在那些日期上生效
		if len(need.TargetDates) > 0 {
			for _, targetDate := range need.TargetDates {
				if targetDate == dateStr {
					return true
				}
			}
		}
		// 没有指定 TargetDates 时：
		// - 如果有 TargetShiftID，表示只回避特定班次，不代表整体请假，不在此处排除
		//   （由 isForbiddenOnShift 处理）
		// - 如果也没有 TargetShiftID，表示整个周期都回避所有班次，视为整期请假
		if need.TargetShiftID == "" {
			return true
		}
	}
	return false
}

// hasFixedAssignmentOnDate 检查员工在指定日期是否有固定排班
func (f *CandidateFilter) hasFixedAssignmentOnDate(
	staffID string,
	fixedAssignments []*model.CtxFixedShiftAssignment,
	shiftID string,
	date time.Time,
) bool {
	dateStr := date.Format("2006-01-02")
	for _, assignment := range fixedAssignments {
		if assignment.Date != dateStr {
			continue
		}
		// 检查员工是否在该日期的固定排班中
		for _, staffIDInAssignment := range assignment.StaffIDs {
			if staffIDInAssignment == staffID {
				// 如果已排其他班次，则排除
				if assignment.ShiftID != shiftID {
					return true
				}
			}
		}
	}
	return false
}

// findTimeOverlappingShiftOnDate 查找员工在同一天已被安排且时间重叠的班次ID
// 返回冲突班次ID（空字符串表示无冲突），只有时间重叠超过1小时才算冲突
func (f *CandidateFilter) findTimeOverlappingShiftOnDate(
	staffID string,
	currentDraft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
	shiftMap map[string]*model.Shift,
	targetShift *model.Shift,
) string {
	if currentDraft == nil || currentDraft.Shifts == nil {
		return ""
	}

	dateStr := date.Format("2006-01-02")

	// 获取目标班次信息
	if targetShift == nil {
		// 如果目标班次信息不可用，回退到旧逻辑（保守处理）
		if f.hasAssignedToOtherShiftOnDateLegacy(staffID, currentDraft, shiftID, date) {
			return "unknown(no-shift-info)"
		}
		return ""
	}

	// 遍历所有班次，检查是否有其他班次已安排该员工且时间重叠
	for otherShiftID, shiftDraft := range currentDraft.Shifts {
		// 跳过当前班次（允许在同一班次重复排班，由其他规则控制）
		if otherShiftID == shiftID {
			continue
		}

		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		dayShift, ok := shiftDraft.Days[dateStr]
		if !ok || dayShift == nil {
			continue
		}

		// 检查该员工是否在该班次的排班中
		hasStaff := false
		for _, sid := range dayShift.StaffIDs {
			if sid == staffID {
				hasStaff = true
				break
			}
		}

		if !hasStaff {
			continue
		}

		// 获取其他班次信息
		otherShift, ok := shiftMap[otherShiftID]
		if !ok || otherShift == nil {
			// 如果无法获取班次信息，保守处理：认为有冲突
			f.logger.Warn("Cannot get shift info for time overlap check, treating as conflict",
				"shiftID", otherShiftID)
			return otherShiftID
		}

		// 检查时间是否真正重叠（超过1小时）
		if f.checkTimeOverlap(targetShift, otherShift) {
			return otherShiftID
		}
		// 时间不重叠，不算冲突（如：下夜班00:00-08:00 和 CT/MRI报告下08:00-14:00）
	}

	return ""
}

// hasAssignedToOtherShiftOnDateLegacy 旧版逻辑（无时间重叠检查，用于回退）
func (f *CandidateFilter) hasAssignedToOtherShiftOnDateLegacy(
	staffID string,
	currentDraft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) bool {
	if currentDraft == nil || currentDraft.Shifts == nil {
		return false
	}

	dateStr := date.Format("2006-01-02")

	// 遍历所有班次，检查是否有其他班次已安排该员工
	for otherShiftID, shiftDraft := range currentDraft.Shifts {
		// 跳过当前班次
		if otherShiftID == shiftID {
			continue
		}

		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		dayShift, ok := shiftDraft.Days[dateStr]
		if !ok || dayShift == nil {
			continue
		}

		// 检查该员工是否在该班次的排班中
		for _, sid := range dayShift.StaffIDs {
			if sid == staffID {
				return true
			}
		}
	}

	return false
}

// checkTimeOverlap 检查两个班次是否有时间重叠（超过1小时）
func (f *CandidateFilter) checkTimeOverlap(shift1, shift2 *model.Shift) bool {
	if shift1 == nil || shift2 == nil {
		return false
	}

	// 解析时间字符串（HH:MM格式）为分钟数
	parseTime := func(timeStr string) int {
		parts := strings.Split(timeStr, ":")
		if len(parts) >= 2 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			return h*60 + m
		}
		return 0
	}

	s1 := parseTime(shift1.StartTime)
	e1 := parseTime(shift1.EndTime)
	s2 := parseTime(shift2.StartTime)
	e2 := parseTime(shift2.EndTime)

	// 处理结束时间为00:00的情况（当作24:00，即跨日班次）
	if e1 == 0 {
		e1 = 24 * 60
	}
	if e2 == 0 {
		e2 = 24 * 60
	}

	// 计算重叠时间
	overlapStart := s1
	if s2 > s1 {
		overlapStart = s2
	}
	overlapEnd := e1
	if e2 < e1 {
		overlapEnd = e2
	}
	overlap := overlapEnd - overlapStart
	if overlap < 0 {
		overlap = 0
	}

	// 重叠超过60分钟才算冲突
	// 例如：下夜班(00:00-08:00) 和 CT/MRI报告下(08:00-14:00) 不重叠（overlap=0）
	return overlap > 60
}
