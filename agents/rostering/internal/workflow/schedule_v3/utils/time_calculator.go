package utils

import (
	"fmt"
	"strconv"
	"strings"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 时间计算工具（V3 LLM增强：支持班次时间冲突检测）
// ============================================================

// timeToMinutes 将HH:MM格式的时间转换为从00:00开始的分钟数
// 例如: "08:30" -> 510, "14:00" -> 840
// 支持HH:MM和HH:MM:SS格式
func timeToMinutes(timeStr string) int {
	if timeStr == "" {
		return 0
	}

	// 支持HH:MM和HH:MM:SS两种格式
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return 0
	}

	hours, err1 := strconv.Atoi(parts[0])
	minutes, err2 := strconv.Atoi(parts[1])

	if err1 != nil || err2 != nil {
		return 0
	}

	// 校验时间范围
	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return 0
	}

	return hours*60 + minutes
}

// CheckTimeOverlap 检查两个班次是否有时间重叠（超过1小时）
// 简化逻辑：直接比较时间段，重叠超过60分钟才算冲突
// 这样可以处理边界情况，如夜班(20:00-24:00)和下夜班(00:00-08:00)不算冲突
//
// 参数:
//   - shift1: 第一个班次
//   - shift2: 第二个班次
//
// 返回:
//   - bool: true表示有时间重叠超过1小时，false表示无重叠或重叠不足1小时
func CheckTimeOverlap(shift1, shift2 *d_model.Shift) bool {
	if shift1 == nil || shift2 == nil {
		return false
	}

	s1 := timeToMinutes(shift1.StartTime)
	e1 := timeToMinutes(shift1.EndTime)
	s2 := timeToMinutes(shift2.StartTime)
	e2 := timeToMinutes(shift2.EndTime)

	// 处理结束时间小于开始时间的情况（如24:00表示为00:00）
	if e1 == 0 {
		e1 = 24 * 60 // 00:00 当作 24:00
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

	// 重叠时间 = max(0, overlapEnd - overlapStart)
	overlap := overlapEnd - overlapStart
	if overlap < 0 {
		overlap = 0
	}

	// 重叠超过60分钟才算冲突
	return overlap > 60
}

// HasTimeOverlap 检查班次数组中是否存在时间重叠
// 用于检测一个人员当天是否被安排了冲突的班次
//
// 参数:
//   - shifts: 已分配的班次列表
//
// 返回:
//   - bool: true表示存在时间重叠
func HasTimeOverlap(shifts []*d_model.AssignedShiftInfo) bool {
	if len(shifts) <= 1 {
		return false
	}

	// 两两比较
	for i := 0; i < len(shifts)-1; i++ {
		for j := i + 1; j < len(shifts); j++ {
			if checkAssignedShiftOverlap(shifts[i], shifts[j]) {
				return true
			}
		}
	}

	return false
}

// checkAssignedShiftOverlap 检查两个已分配班次是否有时间重叠
func checkAssignedShiftOverlap(shift1, shift2 *d_model.AssignedShiftInfo) bool {
	if shift1 == nil || shift2 == nil {
		return false
	}

	s1 := timeToMinutes(shift1.StartTime)
	e1 := timeToMinutes(shift1.EndTime)
	s2 := timeToMinutes(shift2.StartTime)
	e2 := timeToMinutes(shift2.EndTime)

	// 处理跨夜
	if shift1.IsOvernight {
		e1 += 24 * 60
	}
	if shift2.IsOvernight {
		e2 += 24 * 60
	}

	return !(e1 <= s2 || e2 <= s1)
}

// CalculateRestHours 计算两个班次之间的休息时长
// 用于检查是否满足最小休息时长要求
//
// 参数:
//   - shift1End: 第一个班次的结束时间 (HH:MM)
//   - shift2Start: 第二个班次的开始时间 (HH:MM)
//   - shift1IsOvernight: 第一个班次是否跨夜
//
// 返回:
//   - float64: 休息时长（小时）
func CalculateRestHours(shift1End, shift2Start string, shift1IsOvernight bool) float64 {
	e1 := timeToMinutes(shift1End)
	s2 := timeToMinutes(shift2Start)

	// 如果第一个班次跨夜，其结束时间在第二天
	if shift1IsOvernight {
		e1 += 24 * 60
	}

	// 如果s2 < e1，说明第二个班次在第二天或更晚
	if s2 < e1 {
		s2 += 24 * 60
	}

	restMinutes := s2 - e1
	if restMinutes < 0 {
		restMinutes = 0
	}

	return float64(restMinutes) / 60.0
}

// GetShiftDurationHours 获取班次时长（小时）
// 如果班次有Duration字段则使用，否则根据StartTime和EndTime计算
//
// 参数:
//   - shift: 班次对象
//
// 返回:
//   - float64: 时长（小时）
func GetShiftDurationHours(shift *d_model.Shift) float64 {
	if shift == nil {
		return 0
	}

	// 优先使用Duration字段（更准确，考虑了休息时间等）
	if shift.Duration > 0 {
		return float64(shift.Duration) / 60.0
	}

	// 降级：根据StartTime和EndTime计算
	start := timeToMinutes(shift.StartTime)
	end := timeToMinutes(shift.EndTime)

	if shift.IsOvernight {
		end += 24 * 60
	}

	duration := end - start
	if duration < 0 {
		duration = 0
	}

	return float64(duration) / 60.0
}

// DetectConsecutiveShiftViolation 检测跨日连班违规
// 检查前一天的班次和当天的班次之间是否有足够的休息时间
//
// 参数:
//   - previousDayShifts: 前一天的班次列表
//   - currentDayShifts: 当天的班次列表
//   - minRestHours: 最小休息时长（小时）
//
// 返回:
//   - violated: 是否违规
//   - message: 违规描述信息
func DetectConsecutiveShiftViolation(
	previousDayShifts []*d_model.AssignedShiftInfo,
	currentDayShifts []*d_model.AssignedShiftInfo,
	minRestHours float64,
) (violated bool, message string) {
	if len(previousDayShifts) == 0 || len(currentDayShifts) == 0 {
		return false, ""
	}

	// 找到前一天最晚结束的班次
	var latestShift *d_model.AssignedShiftInfo
	latestEndMinutes := -1

	for _, shift := range previousDayShifts {
		if shift == nil {
			continue
		}

		endMinutes := timeToMinutes(shift.EndTime)

		// 跨夜班的结束时间在次日，需要加24小时
		if shift.IsOvernight {
			endMinutes += 24 * 60
		}

		if endMinutes > latestEndMinutes {
			latestEndMinutes = endMinutes
			latestShift = shift
		}
	}

	// 找到当天最早开始的班次
	var earliestShift *d_model.AssignedShiftInfo
	earliestStartMinutes := 24 * 60 // 初始化为一天的最大分钟数

	for _, shift := range currentDayShifts {
		if shift == nil {
			continue
		}

		startMinutes := timeToMinutes(shift.StartTime)

		if startMinutes < earliestStartMinutes {
			earliestStartMinutes = startMinutes
			earliestShift = shift
		}
	}

	if latestShift == nil || earliestShift == nil {
		return false, ""
	}

	// 计算休息时长
	restHours := CalculateRestHours(latestShift.EndTime, earliestShift.StartTime, latestShift.IsOvernight)

	// 检查是否违规
	if restHours < minRestHours {
		message = fmt.Sprintf("前一天%s结束于%s，当天%s开始于%s，休息仅%.1f小时（要求≥%.1f小时）",
			latestShift.ShiftName, latestShift.EndTime,
			earliestShift.ShiftName, earliestShift.StartTime,
			restHours, minRestHours)
		return true, message
	}

	return false, ""
}

// CalculateTotalHours 计算班次列表的总工时（小时）
func CalculateTotalHours(shifts []*d_model.AssignedShiftInfo) float64 {
	total := 0.0
	for _, shift := range shifts {
		if shift != nil {
			total += shift.Duration
		}
	}
	return total
}

// GetTimeConflictDescription 获取时间冲突的详细描述
// 用于生成友好的错误消息
func GetTimeConflictDescription(shifts []*d_model.AssignedShiftInfo) string {
	if len(shifts) <= 1 {
		return ""
	}

	conflicts := []string{}

	// 两两比较找出所有冲突
	for i := 0; i < len(shifts)-1; i++ {
		for j := i + 1; j < len(shifts); j++ {
			if checkAssignedShiftOverlap(shifts[i], shifts[j]) {
				conflict := fmt.Sprintf("%s(%s-%s) 与 %s(%s-%s)",
					shifts[i].ShiftName, shifts[i].StartTime, shifts[i].EndTime,
					shifts[j].ShiftName, shifts[j].StartTime, shifts[j].EndTime)
				conflicts = append(conflicts, conflict)
			}
		}
	}

	if len(conflicts) > 0 {
		return "时间冲突: " + strings.Join(conflicts, "; ")
	}

	return ""
}
