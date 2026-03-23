package utils

import (
	"fmt"
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 排班上下文构建器（V3 LLM增强：构建完整的排班上下文信息）
// ============================================================

// BuildSchedulingContext 构建完整的排班上下文（传递给LLM）
//
// 参数:
//   - date: 目标日期
//   - targetShift: 当前处理的班次
//   - requiredCount: 需要的人数
//   - createCtx: 创建工作流上下文
//   - maxDailyHours: 最大每日工作小时数（从配置读取）
//   - minRestHours: 最小休息小时数（从配置读取）
//
// 返回:
//   - *d_model.V3SchedulingContext: 完整的排班上下文
func BuildSchedulingContext(
	date string,
	targetShift *d_model.Shift,
	requiredCount int,
	workingDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
	maxDailyHours float64,
	minRestHours float64,
) *d_model.V3SchedulingContext {

	// 1. 构建人员当前排班状态
	staffSchedules := BuildStaffCurrentSchedules(
		date,
		workingDraft,
		staffList,
		allStaffList,
		allShifts,
	)

	// 2. 构建上下文
	return &d_model.V3SchedulingContext{
		TargetDate:      date,
		TargetShiftID:   targetShift.ID,
		TargetShiftName: targetShift.Name,
		TargetShiftTime: fmt.Sprintf("%s-%s", targetShift.StartTime, targetShift.EndTime),
		RequiredCount:   requiredCount,
		AllShifts:       allShifts,
		StaffSchedules:  staffSchedules,
		MaxDailyHours:   maxDailyHours,
		MinRestHours:    minRestHours,
	}
}

// BuildStaffCurrentSchedules 构建人员当前排班状态
// 从WorkingDraft提取指定日期的人员排班信息，并检测前一天的排班以判断连班休息
//
// 参数:
//   - date: 目标日期
//   - workingDraft: 当前排班草稿
//   - staffList: 班次关联人员列表
//   - allStaffList: 所有人员列表
//   - allShifts: 所有班次列表
//
// 返回:
//   - []*d_model.StaffCurrentSchedule: 人员排班状态列表
func BuildStaffCurrentSchedules(
	date string,
	workingDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) []*d_model.StaffCurrentSchedule {

	// 完善空值校验，防止panic
	if workingDraft == nil || workingDraft.Shifts == nil {
		return make([]*d_model.StaffCurrentSchedule, 0)
	}
	if date == "" {
		return make([]*d_model.StaffCurrentSchedule, 0)
	}
	if len(allShifts) == 0 {
		return make([]*d_model.StaffCurrentSchedule, 0)
	}

	// 1. 构建班次ID->班次对象的映射
	shiftMap := buildShiftMap(allShifts)

	// 2. 构建人员ID->姓名的映射
	staffNamesMap := BuildStaffNamesMap(staffList)
	if allStaffList != nil {
		allStaffNamesMap := BuildStaffNamesMap(allStaffList)
		for id, name := range allStaffNamesMap {
			staffNamesMap[id] = name
		}
	}

	// 3. 计算前一天日期（用于检测跨日连班）
	previousDate := calculatePreviousDate(date)

	// 4. 遍历WorkingDraft，收集每个人员的当天和前一天排班
	staffScheduleMap := make(map[string]*d_model.StaffCurrentSchedule)
	staffPreviousDayShifts := make(map[string][]*d_model.AssignedShiftInfo)

	for shiftID, shiftDraft := range workingDraft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}

		// 处理当天排班
		dayShift := shiftDraft.Days[date]
		if dayShift != nil {
			for i, staffID := range dayShift.StaffIDs {
				// 初始化人员排班记录
				if staffScheduleMap[staffID] == nil {
					staffScheduleMap[staffID] = &d_model.StaffCurrentSchedule{
						StaffID:   staffID,
						StaffName: getStaffName(staffID, staffNamesMap),
						Date:      date,
						Shifts:    make([]*d_model.AssignedShiftInfo, 0),
						Errors:    make([]string, 0),
						Warnings:  make([]string, 0),
					}
				}

				// 添加班次信息
				assignedShift := &d_model.AssignedShiftInfo{
					ShiftID:     shift.ID,
					ShiftName:   shift.Name,
					StartTime:   shift.StartTime,
					EndTime:     shift.EndTime,
					Duration:    GetShiftDurationHours(shift),
					IsOvernight: shift.IsOvernight,
					IsFixed:     dayShift.IsFixed,
					TaskID:      determineTaskID(dayShift, i),
				}

				staffScheduleMap[staffID].Shifts = append(
					staffScheduleMap[staffID].Shifts,
					assignedShift,
				)
				staffScheduleMap[staffID].TotalHours += assignedShift.Duration
			}
		}

		// 处理前一天排班（用于连班检测）
		if previousDate != "" {
			prevDayShift := shiftDraft.Days[previousDate]
			if prevDayShift != nil {
				for _, staffID := range prevDayShift.StaffIDs {
					if staffPreviousDayShifts[staffID] == nil {
						staffPreviousDayShifts[staffID] = make([]*d_model.AssignedShiftInfo, 0)
					}

					prevAssignedShift := &d_model.AssignedShiftInfo{
						ShiftID:     shift.ID,
						ShiftName:   shift.Name,
						StartTime:   shift.StartTime,
						EndTime:     shift.EndTime,
						Duration:    GetShiftDurationHours(shift),
						IsOvernight: shift.IsOvernight,
						IsFixed:     prevDayShift.IsFixed,
					}

					staffPreviousDayShifts[staffID] = append(
						staffPreviousDayShifts[staffID],
						prevAssignedShift,
					)
				}
			}
		}
	}

	// 5. 检测错误和警告
	for staffID, schedule := range staffScheduleMap {
		detectErrorsAndWarnings(schedule, staffPreviousDayShifts[staffID], 12.0, 12.0)
	}

	// 6. 转为数组
	result := make([]*d_model.StaffCurrentSchedule, 0, len(staffScheduleMap))
	for _, schedule := range staffScheduleMap {
		result = append(result, schedule)
	}

	return result
}

// ============================================================
// 辅助函数
// ============================================================

// buildShiftMap 构建班次ID到班次对象的映射
func buildShiftMap(shifts []*d_model.Shift) map[string]*d_model.Shift {
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		if shift != nil {
			shiftMap[shift.ID] = shift
		}
	}
	return shiftMap
}

// getStaffName 获取人员姓名
func getStaffName(staffID string, namesMap map[string]string) string {
	if name, ok := namesMap[staffID]; ok {
		return name
	}
	return staffID // 降级：使用ID
}

// determineTaskID 判断班次来源任务ID
func determineTaskID(dayShift *d_model.DayShift, index int) string {
	// 固定排班没有任务ID
	if dayShift.IsFixed {
		return ""
	}
	// TODO: 可以从上下文中获取当前任务ID并记录
	return "task"
}

// detectErrorsAndWarnings 检测排班的错误和警告
// 错误（Errors）: 阻断性问题，必须解决（如时间冲突）
// 警告（Warnings）: 提示性问题，可以容忍（如接近超时）
func detectErrorsAndWarnings(
	schedule *d_model.StaffCurrentSchedule,
	previousDayShifts []*d_model.AssignedShiftInfo,
	maxDailyHours float64,
	minRestHours float64,
) {
	if schedule == nil {
		return
	}

	// 1. 检测当天时间冲突（阻断性错误）
	if len(schedule.Shifts) > 1 {
		conflictDesc := GetTimeConflictDescription(schedule.Shifts)
		if conflictDesc != "" {
			schedule.Errors = append(schedule.Errors, conflictDesc)
		}
	}

	// 2. 检测超时（阻断性错误）
	if schedule.TotalHours > maxDailyHours {
		schedule.Errors = append(schedule.Errors,
			fmt.Sprintf("超时：已安排%.1f小时（超过%.1f小时限制）", schedule.TotalHours, maxDailyHours))
	} else if schedule.TotalHours > maxDailyHours*0.9 {
		// 接近超时（警告）
		schedule.Warnings = append(schedule.Warnings,
			fmt.Sprintf("接近超时：已安排%.1f小时（限制%.1f小时）", schedule.TotalHours, maxDailyHours))
	}

	// 3. 检测跨日连班休息不足（阻断性错误）
	if len(previousDayShifts) > 0 && len(schedule.Shifts) > 0 {
		violated, message := DetectConsecutiveShiftViolation(previousDayShifts, schedule.Shifts, minRestHours)
		if violated {
			schedule.Errors = append(schedule.Errors, message)
		}
	}
}

// calculatePreviousDate 计算前一天日期
// 输入: "2026-01-20"
// 输出: "2026-01-19"
func calculatePreviousDate(date string) string {
	if date == "" {
		return ""
	}

	// 使用time包精确处理日期
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}

	// 减1天
	prevDay := t.AddDate(0, 0, -1)
	return prevDay.Format("2006-01-02")
}
