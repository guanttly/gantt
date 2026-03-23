package model

// ============================================================
// V3排班上下文辅助查询函数
// 提供强类型数组的查询、过滤、转换等操作
// ============================================================

// ============================================================
// StaffOccupiedSlot 查询函数
// ============================================================

// FindOccupiedSlot 查找指定人员在指定日期的占位信息
func FindOccupiedSlot(slots []StaffOccupiedSlot, staffID, date string) *StaffOccupiedSlot {
	for i := range slots {
		if slots[i].StaffID == staffID && slots[i].Date == date {
			return &slots[i]
		}
	}
	return nil
}

// IsStaffOccupiedOnDate 检查人员在指定日期是否已被占位
func IsStaffOccupiedOnDate(slots []StaffOccupiedSlot, staffID, date string) bool {
	return FindOccupiedSlot(slots, staffID, date) != nil
}

// FilterOccupiedByStaff 过滤某员工的所有占位记录
func FilterOccupiedByStaff(slots []StaffOccupiedSlot, staffID string) []StaffOccupiedSlot {
	result := make([]StaffOccupiedSlot, 0)
	for _, slot := range slots {
		if slot.StaffID == staffID {
			result = append(result, slot)
		}
	}
	return result
}

// AddOccupiedSlot 添加占位记录（不检查重复）
func AddOccupiedSlot(slots []StaffOccupiedSlot, slot StaffOccupiedSlot) []StaffOccupiedSlot {
	return append(slots, slot)
}

// AddOccupiedSlotIfNotExists 添加占位记录（检查重复，避免重复占位）
func AddOccupiedSlotIfNotExists(slots []StaffOccupiedSlot, slot StaffOccupiedSlot) []StaffOccupiedSlot {
	// 检查是否已存在
	if FindOccupiedSlot(slots, slot.StaffID, slot.Date) != nil {
		return slots // 已存在，不添加
	}
	return append(slots, slot)
}

// RemoveOccupiedSlot 移除指定人员在指定日期的占位记录
func RemoveOccupiedSlot(slots []StaffOccupiedSlot, staffID, date string) []StaffOccupiedSlot {
	result := make([]StaffOccupiedSlot, 0, len(slots))
	for _, slot := range slots {
		if !(slot.StaffID == staffID && slot.Date == date) {
			result = append(result, slot)
		}
	}
	return result
}

// CountOccupiedByStaff 统计某员工的排班天数
func CountOccupiedByStaff(slots []StaffOccupiedSlot, staffID string) int {
	count := 0
	for _, slot := range slots {
		if slot.StaffID == staffID {
			count++
		}
	}
	return count
}

// GetStaffOtherShiftSlots 获取某员工在其他班次的占位记录（排除指定班次）
func GetStaffOtherShiftSlots(slots []StaffOccupiedSlot, staffID, excludeShiftID string) []StaffOccupiedSlot {
	result := make([]StaffOccupiedSlot, 0)
	for _, slot := range slots {
		if slot.StaffID == staffID && slot.ShiftID != excludeShiftID {
			result = append(result, slot)
		}
	}
	return result
}

// ============================================================
// ShiftDateRequirement 查询函数
// ============================================================

// FindRequirement 查找指定班次在指定日期的人员需求
func FindRequirement(reqs []ShiftDateRequirement, shiftID, date string) *ShiftDateRequirement {
	for i := range reqs {
		if reqs[i].ShiftID == shiftID && reqs[i].Date == date {
			return &reqs[i]
		}
	}
	return nil
}

// FilterRequirementsByShift 过滤某班次的所有需求
func FilterRequirementsByShift(reqs []ShiftDateRequirement, shiftID string) []ShiftDateRequirement {
	result := make([]ShiftDateRequirement, 0)
	for _, req := range reqs {
		if req.ShiftID == shiftID {
			result = append(result, req)
		}
	}
	return result
}

// ============================================================
// CtxFixedShiftAssignment 查询函数
// ============================================================

// FindFixedAssignment 查找指定班次在指定日期的固定排班人员ID列表
func FindFixedAssignment(assigns []CtxFixedShiftAssignment, shiftID, date string) []string {
	for i := range assigns {
		if assigns[i].ShiftID == shiftID && assigns[i].Date == date {
			return assigns[i].StaffIDs
		}
	}
	return nil
}

// FindFixedAssignmentStruct 查找指定班次在指定日期的固定排班结构（返回指针）
func FindFixedAssignmentStruct(assigns []CtxFixedShiftAssignment, shiftID, date string) *CtxFixedShiftAssignment {
	for i := range assigns {
		if assigns[i].ShiftID == shiftID && assigns[i].Date == date {
			return &assigns[i]
		}
	}
	return nil
}

// ============================================================
// 类型转换函数（旧格式 ← 新格式，用于兼容旧代码）
// Deprecated: 这些转换函数仅用于过渡期兼容旧代码。
// 新代码应直接使用强类型数组和对应的查询函数。
// ============================================================

// ConvertOccupiedSlotsToMap 将强类型数组转换回双层map（兼容旧接口）
//
// Deprecated: 此函数仅用于过渡期兼容旧代码。
// 新代码应直接使用 []StaffOccupiedSlot 数组和对应的查询函数。
// 计划在 v3.1.0 版本后移除。
func ConvertOccupiedSlotsToMap(slots []StaffOccupiedSlot) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, slot := range slots {
		if result[slot.StaffID] == nil {
			result[slot.StaffID] = make(map[string]string)
		}
		result[slot.StaffID][slot.Date] = slot.ShiftID
	}
	return result
}

// ConvertRequirementsToMap 将强类型数组转换回双层map（兼容LLM接口）
//
// Deprecated: 此函数仅用于过渡期兼容。
// 新代码应使用 []ShiftDateRequirement 数组和对应的查询函数。
// 或者使用 CoreV3TaskContext.ShiftRequirementsMap 预构建缓存。
func ConvertRequirementsToMap(reqs []ShiftDateRequirement) map[string]map[string]int {
	result := make(map[string]map[string]int)
	for _, req := range reqs {
		if result[req.ShiftID] == nil {
			result[req.ShiftID] = make(map[string]int)
		}
		result[req.ShiftID][req.Date] = req.Count
	}
	return result
}

// ConvertFixedAssignmentsToMap 将强类型数组转换为日期-人员映射（LLM调用时使用）
// 格式: date -> []staffIDs（合并所有班次的固定排班）
//
// Deprecated: 此函数仅用于过渡期兼容。
// 新代码应使用 CoreV3TaskContext.FixedAssignmentsMap 预构建缓存。
func ConvertFixedAssignmentsToMap(assigns []CtxFixedShiftAssignment) map[string][]string {
	result := make(map[string][]string)
	for _, assign := range assigns {
		result[assign.Date] = append(result[assign.Date], assign.StaffIDs...)
	}
	return result
}
