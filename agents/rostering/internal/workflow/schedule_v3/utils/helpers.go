package utils

import (
	"encoding/json"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 辅助函数 - 用于变更计算和应用
// ============================================================

// BuildStaffNamesMap 构建人员ID到姓名的映射
func BuildStaffNamesMap(staffList []*d_model.Employee) map[string]string {
	namesMap := make(map[string]string)
	for _, staff := range staffList {
		namesMap[staff.ID] = staff.Name
	}
	return namesMap
}

// BuildShiftNamesMap 构建班次ID到名称的映射
func BuildShiftNamesMap(shifts []*d_model.Shift) map[string]string {
	namesMap := make(map[string]string)
	for _, shift := range shifts {
		namesMap[shift.ID] = shift.Name
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
			names = append(names, id) // 降级处理：找不到名称则使用ID
		}
	}
	return names
}

// DeepCopyScheduleDraft 深拷贝 ScheduleDraft（使用JSON序列化）
func DeepCopyScheduleDraft(src *d_model.ScheduleDraft) (*d_model.ScheduleDraft, error) {
	if src == nil {
		return nil, nil
	}

	// 序列化
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	// 反序列化
	var dst d_model.ScheduleDraft
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}

	return &dst, nil
}

// SlicesEqual 判断两个字符串切片是否相等（不考虑顺序）
func SlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// 构建计数 map
	countA := make(map[string]int)
	for _, item := range a {
		countA[item]++
	}

	countB := make(map[string]int)
	for _, item := range b {
		countB[item]++
	}

	// 比较
	if len(countA) != len(countB) {
		return false
	}

	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}

// ============================================================
// AI输入数据转换函数（V3改进：map转强类型结构体）
// ============================================================

// ConvertStaffRequirementsToDailyList 将嵌套map转换为DailyRequirement数组
// 参数：
//   - requirements: map[shiftID]map[date]count 格式的人员需求
//   - shifts: 班次列表（用于获取班次名称）
//
// 返回：DailyRequirement数组，适合AI输入和序列化
func ConvertStaffRequirementsToDailyList(
	requirements map[string]map[string]int,
	shifts []*d_model.Shift,
) []*d_model.DailyRequirement {
	shiftNamesMap := BuildShiftNamesMap(shifts)
	result := make([]*d_model.DailyRequirement, 0)

	for shiftID, dateMap := range requirements {
		shiftName := shiftNamesMap[shiftID]
		if shiftName == "" {
			shiftName = shiftID // 降级：使用ID
		}

		for date, count := range dateMap {
			result = append(result, &d_model.DailyRequirement{
				ShiftID:       shiftID,
				ShiftName:     shiftName,
				Date:          date,
				RequiredCount: count,
			})
		}
	}

	return result
}

// ConvertFixedAssignmentsForAI 将CtxFixedShiftAssignment转换为AI输入格式
// 参数：
//   - assignments: CtxFixedShiftAssignment列表
//   - shifts: 班次列表（用于获取班次名称）
//   - staffList: 人员列表（用于获取人员姓名）
//
// 返回：FixedAssignmentForAI数组，适合AI输入
func ConvertFixedAssignmentsForAI(
	assignments []d_model.CtxFixedShiftAssignment,
	shifts []*d_model.Shift,
	staffList []*d_model.Employee,
) []*d_model.FixedAssignmentForAI {
	shiftNamesMap := BuildShiftNamesMap(shifts)
	staffNamesMap := BuildStaffNamesMap(staffList)
	result := make([]*d_model.FixedAssignmentForAI, 0, len(assignments))

	for _, assignment := range assignments {
		shiftName := shiftNamesMap[assignment.ShiftID]
		if shiftName == "" {
			shiftName = assignment.ShiftID
		}

		staffNames := MapIDsToNames(assignment.StaffIDs, staffNamesMap)

		result = append(result, &d_model.FixedAssignmentForAI{
			Date:            assignment.Date,
			ShiftID:         assignment.ShiftID,
			ShiftName:       shiftName,
			StaffIDs:        assignment.StaffIDs,
			StaffNames:      staffNames,
			IsFixedSchedule: true, // 标识为固定排班
		})
	}

	return result
}
