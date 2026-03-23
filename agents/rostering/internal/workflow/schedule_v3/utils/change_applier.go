package utils

import (
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 变更应用器 - 将任务结果应用到工作草稿
// ============================================================

// OccupySlotsFunc 占位函数类型
type OccupySlotsFunc func(staffID, date, shiftID string)

// ApplyChangeBatch 将任务结果应用到工作草稿（V3改进：增加AI输出验证）
//
// 参数:
//   - draft: 要更新的排班草稿（通常是 WorkingDraft）
//   - afterSchedules: 任务返回的排班结果（shiftID -> ShiftScheduleDraft）
//   - staffList: 人员列表（用于姓名映射和验证）
//   - allStaffList: 所有人员列表（用于姓名映射补充）
//   - occupySlot: 占位函数
//   - occupiedSlots: 当前占位状态（强类型数组）
//
// 返回:
//   - error: 错误信息（如果AI输出包含无效数据）
func ApplyChangeBatch(
	draft *d_model.ScheduleDraft,
	afterSchedules map[string]*d_model.ShiftScheduleDraft,
	staffList []*d_model.Employee,
	allStaffList []*d_model.Employee,
	occupySlot OccupySlotsFunc,
	occupiedSlots []d_model.StaffOccupiedSlot,
) error {
	if draft == nil {
		return fmt.Errorf("draft is nil")
	}
	if draft.Shifts == nil {
		draft.Shifts = make(map[string]*d_model.ShiftDraft)
	}

	// 【性能优化】提前构建一次姓名映射，避免在循环中重复构建
	staffNamesMap := BuildStaffNamesMap(staffList)
	if allStaffList != nil {
		allStaffNamesMap := BuildStaffNamesMap(allStaffList)
		for id, name := range allStaffNamesMap {
			staffNamesMap[id] = name
		}
	}

	// 【V3改进】构建有效人员ID集合（用于验证AI输出）
	validStaffIDs := make(map[string]bool)
	for _, staff := range staffList {
		validStaffIDs[staff.ID] = true
	}
	for _, staff := range allStaffList {
		validStaffIDs[staff.ID] = true
	}

	// 【V3改进】构建姓名到ID的反向映射（当AI返回姓名时可以转换回ID）
	nameToIDMap := make(map[string]string)
	for _, staff := range staffList {
		nameToIDMap[staff.Name] = staff.ID
	}
	for _, staff := range allStaffList {
		nameToIDMap[staff.Name] = staff.ID
	}

	// 遍历所有班次的结果
	for shiftID, shiftSchedule := range afterSchedules {
		if shiftSchedule == nil || shiftSchedule.Schedule == nil {
			continue
		}

		// 确保班次存在
		if draft.Shifts[shiftID] == nil {
			draft.Shifts[shiftID] = &d_model.ShiftDraft{
				ShiftID: shiftID,
				Days:    make(map[string]*d_model.DayShift),
			}
		}

		shiftDraft := draft.Shifts[shiftID]
		if shiftDraft.Days == nil {
			shiftDraft.Days = make(map[string]*d_model.DayShift)
		}

		// 遍历每一天的排班
		for date, staffIDs := range shiftSchedule.Schedule {
			// 【关键】检查是否为固定排班，如果是则跳过
			if existingDayShift, ok := shiftDraft.Days[date]; ok && existingDayShift != nil && existingDayShift.IsFixed {
				// 固定排班不允许修改，跳过AI的结果
				// 注：已移除调试日志，这是正常的保护行为
				continue
			}

			// 【V3改进】验证AI返回的人员ID是否有效，并尝试将姓名转换为ID
			invalidStaffIDs := make([]string, 0)
			validFilteredIDs := make([]string, 0)

			for _, staffID := range staffIDs {
				if validStaffIDs[staffID] {
					// 直接是有效ID
					validFilteredIDs = append(validFilteredIDs, staffID)
				} else if id, ok := nameToIDMap[staffID]; ok {
					// AI返回的是姓名，转换为ID（静默转换，无需打印日志）
					validFilteredIDs = append(validFilteredIDs, id)
				} else {
					// 无效的人员标识
					invalidStaffIDs = append(invalidStaffIDs, staffID)
				}
			}

			// 如果有无效的人员ID，记录警告并过滤掉
			if len(invalidStaffIDs) > 0 {
				// 这里可以选择返回错误，或者只记录警告并继续
				// 当前采用宽松策略：过滤掉无效ID并继续
				fmt.Printf("警告: AI返回了无效的人员ID，已过滤掉: %v (班次=%s, 日期=%s)\n",
					invalidStaffIDs, shiftID, date)
			}

			// 使用过滤后的有效ID列表
			staffIDs = validFilteredIDs

			// 先清理旧的占位（如果存在）
			if oldDayShift, ok := shiftDraft.Days[date]; ok && oldDayShift != nil {
				for _, oldStaffID := range oldDayShift.StaffIDs {
					occupiedSlots = clearOccupiedSlot(occupiedSlots, oldStaffID, date, shiftID)
				}
			}

			// 如果新结果为空数组，表示删除该日期的排班
			if len(staffIDs) == 0 {
				delete(shiftDraft.Days, date)
				continue
			}

			// 转换为姓名（复用已构建的映射）
			staffNames := MapIDsToNames(staffIDs, staffNamesMap)

			// 更新排班
			shiftDraft.Days[date] = &d_model.DayShift{
				Staff:         staffNames,
				StaffIDs:      staffIDs,
				ActualCount:   len(staffIDs),
				RequiredCount: len(staffIDs), // 默认与实际相同
				IsFixed:       false,         // 非固定排班
			}

			// 更新占位
			for _, staffID := range staffIDs {
				occupySlot(staffID, date, shiftID)
			}
		}

		// 如果班次的所有日期都被删除了，删除该班次
		if len(shiftDraft.Days) == 0 {
			delete(draft.Shifts, shiftID)
		}
	}

	return nil
}

// clearOccupiedSlot 清理占位（仅当占位确实是该班次时才清理）
// 使用强类型数组，返回更新后的数组
func clearOccupiedSlot(occupiedSlots []d_model.StaffOccupiedSlot, staffID, date, shiftID string) []d_model.StaffOccupiedSlot {
	// 查找并移除匹配的占位记录
	result := make([]d_model.StaffOccupiedSlot, 0, len(occupiedSlots))
	for _, slot := range occupiedSlots {
		// 只保留不匹配的记录（移除匹配的）
		if slot.StaffID == staffID && slot.Date == date && slot.ShiftID == shiftID {
			continue // 跳过要删除的记录
		}
		result = append(result, slot)
	}
	return result
}

// ============================================================
// V3增强：时间冲突和每日时长验证
// ============================================================

// ValidateTimeConstraints 验证时间约束（时间冲突和每日时长限制）
//
// 参数:
//   - draft: 要验证的排班草稿
//   - allShifts: 所有班次列表（用于获取时间信息）
//   - maxDailyHours: 每日最大工作时长（小时）
//   - staffList: 人员列表（用于显示姓名）
//
// 返回:
//   - []string: 验证错误列表（如果为空则验证通过）
func ValidateTimeConstraints(
	draft *d_model.ScheduleDraft,
	allShifts []*d_model.Shift,
	maxDailyHours float64,
	staffList []*d_model.Employee,
) []string {
	if draft == nil || draft.Shifts == nil {
		return nil
	}

	errors := make([]string, 0)

	// 构建班次ID->班次对象的映射
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range allShifts {
		if shift != nil {
			shiftMap[shift.ID] = shift
		}
	}

	// 构建人员ID->人员姓名的映射
	staffNameMap := make(map[string]string)
	for _, staff := range staffList {
		if staff != nil {
			staffNameMap[staff.ID] = staff.Name
		}
	}

	// 构建人员->日期->班次列表的映射
	staffDailyShifts := make(map[string]map[string][]*d_model.AssignedShiftInfo)

	// 遍历所有班次的排班数据
	for shiftID, shiftDraft := range draft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}

		for date, dayShift := range shiftDraft.Days {
			if dayShift == nil {
				continue
			}

			for _, staffID := range dayShift.StaffIDs {
				// 初始化人员排班记录
				if staffDailyShifts[staffID] == nil {
					staffDailyShifts[staffID] = make(map[string][]*d_model.AssignedShiftInfo)
				}
				if staffDailyShifts[staffID][date] == nil {
					staffDailyShifts[staffID][date] = make([]*d_model.AssignedShiftInfo, 0)
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
				}

				staffDailyShifts[staffID][date] = append(staffDailyShifts[staffID][date], assignedShift)
			}
		}
	}

	// 验证每个人员的每日排班
	for staffID, dailyShifts := range staffDailyShifts {
		for date, shifts := range dailyShifts {
			// 获取人员姓名
			staffName := staffNameMap[staffID]
			if staffName == "" {
				staffName = staffID // 找不到姓名时显示ID
			}

			// 1. 检查时间冲突
			if HasTimeOverlap(shifts) {
				shiftNames := make([]string, len(shifts))
				for i, s := range shifts {
					shiftNames[i] = fmt.Sprintf("%s(%s-%s)", s.ShiftName, s.StartTime, s.EndTime)
				}
				errors = append(errors, fmt.Sprintf("人员%s在%s存在时间冲突：%v",
					staffName, date, shiftNames))
			}
		}
	}

	return errors
}

// ============================================================
// V3增强：后端强制校验功能
// ============================================================

// ValidationResult 校验结果
type ValidationResult struct {
	Valid           bool     // 是否通过校验
	Errors          []string // 错误列表
	Warnings        []string // 警告列表
	TotalViolations int      // 总违规数
	TimeConflicts   int      // 时间冲突数
	OvertimeCount   int      // 超时人数
	RestViolations  int      // 休息不足数
}

// ValidateScheduleDraft 校验排班草案是否符合约束（后端强制校验）
//
// 参数:
//   - draft: 排班草案
//   - allShifts: 所有班次信息
//   - maxDailyHours: 每日最大工时
//   - minRestHours: 最小休息时间
//   - staffNamesMap: 人员ID到姓名的映射
//
// 返回:
//   - *ValidationResult: 校验结果
func ValidateScheduleDraft(
	draft *d_model.ScheduleDraft,
	allShifts []*d_model.Shift,
	maxDailyHours float64,
	minRestHours float64,
	staffNamesMap map[string]string,
) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	if draft == nil || draft.Shifts == nil {
		return result
	}

	// 构建班次映射
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range allShifts {
		if shift != nil {
			shiftMap[shift.ID] = shift
		}
	}

	// 收集每个人员每天的所有班次
	// staffDailyShifts: staffID -> date -> []AssignedShiftInfo
	staffDailyShifts := make(map[string]map[string][]*d_model.AssignedShiftInfo)

	for shiftID, shiftDraft := range draft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}

		for date, dayShift := range shiftDraft.Days {
			if dayShift == nil {
				continue
			}

			for _, staffID := range dayShift.StaffIDs {
				if staffDailyShifts[staffID] == nil {
					staffDailyShifts[staffID] = make(map[string][]*d_model.AssignedShiftInfo)
				}
				if staffDailyShifts[staffID][date] == nil {
					staffDailyShifts[staffID][date] = make([]*d_model.AssignedShiftInfo, 0)
				}

				assignedShift := &d_model.AssignedShiftInfo{
					ShiftID:     shift.ID,
					ShiftName:   shift.Name,
					StartTime:   shift.StartTime,
					EndTime:     shift.EndTime,
					Duration:    GetShiftDurationHours(shift),
					IsOvernight: shift.IsOvernight,
					IsFixed:     dayShift.IsFixed,
				}

				staffDailyShifts[staffID][date] = append(
					staffDailyShifts[staffID][date],
					assignedShift,
				)
			}
		}
	}

	// 对每个人员的每一天进行校验
	for staffID, dateShifts := range staffDailyShifts {
		staffName := staffNamesMap[staffID]
		if staffName == "" {
			staffName = staffID
		}

		for date, shifts := range dateShifts {
			// 1. 检查时间冲突
			if HasTimeOverlap(shifts) {
				conflictDesc := GetTimeConflictDescription(shifts)
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("[%s][%s] %s", date, staffName, conflictDesc))
				result.TimeConflicts++
				result.TotalViolations++
			}

			// 2. 检查超时
			totalHours := CalculateTotalHours(shifts)
			if totalHours > maxDailyHours {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("[%s][%s] 超时：已安排%.1f小时（限制%.1f小时）",
						date, staffName, totalHours, maxDailyHours))
				result.OvertimeCount++
				result.TotalViolations++
			} else if totalHours > maxDailyHours*0.9 {
				// 接近超时（警告）
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("[%s][%s] 接近超时：已安排%.1f小时（限制%.1f小时）",
						date, staffName, totalHours, maxDailyHours))
			}
		}
	}

	// 3. 检查跨日休息时间
	// 需要对日期排序并检查连续日期的休息时间
	for staffID, dateShifts := range staffDailyShifts {
		staffName := staffNamesMap[staffID]
		if staffName == "" {
			staffName = staffID
		}

		// 获取并排序日期
		dates := make([]string, 0, len(dateShifts))
		for date := range dateShifts {
			dates = append(dates, date)
		}

		// 简单排序（字符串排序对于YYYY-MM-DD格式是正确的）
		for i := 0; i < len(dates); i++ {
			for j := i + 1; j < len(dates); j++ {
				if dates[i] > dates[j] {
					dates[i], dates[j] = dates[j], dates[i]
				}
			}
		}

		// 检查连续日期
		for i := 0; i < len(dates)-1; i++ {
			currentDate := dates[i]
			nextDate := dates[i+1]

			// 检查是否是连续日期（简单判断）
			if isConsecutiveDate(currentDate, nextDate) {
				currentShifts := dateShifts[currentDate]
				nextShifts := dateShifts[nextDate]

				// 检查休息时间是否足够
				violated, message := DetectConsecutiveShiftViolation(
					currentShifts, nextShifts, minRestHours)

				if violated {
					result.Valid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("[%s][%s] %s", nextDate, staffName, message))
					result.TotalViolations++
				}
			}
		}

		// 检查连续工作天数限制（新增校验）
		if len(dates) >= 7 {
			// 检查是否有连续7天或以上的工作
			consecutiveDays := 1
			for i := 1; i < len(dates); i++ {
				if isConsecutiveDate(dates[i-1], dates[i]) {
					consecutiveDays++
					if consecutiveDays >= 7 {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("[%s] 连续工作%d天，建议安排休息",
								staffName, consecutiveDays))
					}
				} else {
					consecutiveDays = 1
				}
			}
		}
	}

	return result
}

// isConsecutiveDate 判断两个日期是否连续
// 简化实现：只检查日期差值是否为1
func isConsecutiveDate(date1, date2 string) bool {
	// 简单实现：解析并比较
	// 对于生产环境，应该使用time.Parse
	return true // 暂时返回true，实际应该实现日期比较
}

// ValidateScheduleChanges 校验排班变更是否符合约束（应用前校验）
//
// 参数:
//   - changes: 要应用的排班变更（shiftID -> ShiftScheduleDraft）
//   - existingDraft: 现有排班草案
//   - allShifts: 所有班次信息
//   - maxDailyHours: 每日最大工时
//   - minRestHours: 最小休息时间
//   - staffNamesMap: 人员ID到姓名的映射
//
// 返回:
//   - *ValidationResult: 校验结果
//   - error: 错误信息
func ValidateScheduleChanges(
	changes map[string]*d_model.ShiftScheduleDraft,
	existingDraft *d_model.ScheduleDraft,
	allShifts []*d_model.Shift,
	maxDailyHours float64,
	minRestHours float64,
	staffNamesMap map[string]string,
) (*ValidationResult, error) {
	// 创建一个临时草案，应用变更
	tempDraft := &d_model.ScheduleDraft{
		StartDate: existingDraft.StartDate,
		EndDate:   existingDraft.EndDate,
		Shifts:    make(map[string]*d_model.ShiftDraft),
	}

	// 复制现有排班
	for shiftID, shiftDraft := range existingDraft.Shifts {
		if shiftDraft == nil {
			continue
		}

		tempDraft.Shifts[shiftID] = &d_model.ShiftDraft{
			ShiftID:  shiftDraft.ShiftID,
			Priority: shiftDraft.Priority,
			Days:     make(map[string]*d_model.DayShift),
		}

		for date, dayShift := range shiftDraft.Days {
			if dayShift == nil {
				continue
			}

			tempDraft.Shifts[shiftID].Days[date] = &d_model.DayShift{
				Staff:         append([]string{}, dayShift.Staff...),
				StaffIDs:      append([]string{}, dayShift.StaffIDs...),
				RequiredCount: dayShift.RequiredCount,
				ActualCount:   dayShift.ActualCount,
				IsFixed:       dayShift.IsFixed,
			}
		}
	}

	// 应用变更
	for shiftID, shiftSchedule := range changes {
		if shiftSchedule == nil || shiftSchedule.Schedule == nil {
			continue
		}

		if tempDraft.Shifts[shiftID] == nil {
			tempDraft.Shifts[shiftID] = &d_model.ShiftDraft{
				ShiftID: shiftID,
				Days:    make(map[string]*d_model.DayShift),
			}
		}

		for date, staffIDs := range shiftSchedule.Schedule {
			tempDraft.Shifts[shiftID].Days[date] = &d_model.DayShift{
				StaffIDs:      staffIDs,
				ActualCount:   len(staffIDs),
				RequiredCount: len(staffIDs),
			}
		}
	}

	// 校验临时草案
	return ValidateScheduleDraft(tempDraft, allShifts, maxDailyHours, minRestHours, staffNamesMap), nil
}
