package model

import (
	"testing"
)

// ============================================================
// StaffOccupiedSlot 测试
// ============================================================

func TestFindOccupiedSlot(t *testing.T) {
	slots := []StaffOccupiedSlot{
		{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
		{StaffID: "staff2", Date: "2026-01-01", ShiftID: "shift2"},
		{StaffID: "staff1", Date: "2026-01-02", ShiftID: "shift3"},
	}

	tests := []struct {
		name     string
		staffID  string
		date     string
		expected *StaffOccupiedSlot
	}{
		{
			name:     "找到存在的占位",
			staffID:  "staff1",
			date:     "2026-01-01",
			expected: &slots[0],
		},
		{
			name:     "找到另一个日期的占位",
			staffID:  "staff1",
			date:     "2026-01-02",
			expected: &slots[2],
		},
		{
			name:     "未找到-staffID不存在",
			staffID:  "staff999",
			date:     "2026-01-01",
			expected: nil,
		},
		{
			name:     "未找到-date不存在",
			staffID:  "staff1",
			date:     "2026-01-99",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindOccupiedSlot(slots, tt.staffID, tt.date)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("期望 nil，得到 %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("期望 %+v，得到 nil", tt.expected)
				} else if result.StaffID != tt.expected.StaffID || result.Date != tt.expected.Date {
					t.Errorf("期望 %+v，得到 %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestFindOccupiedSlot_EmptySlice(t *testing.T) {
	result := FindOccupiedSlot([]StaffOccupiedSlot{}, "staff1", "2026-01-01")
	if result != nil {
		t.Errorf("空切片应返回 nil，得到 %+v", result)
	}
}

func TestIsStaffOccupiedOnDate(t *testing.T) {
	slots := []StaffOccupiedSlot{
		{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
		{StaffID: "staff2", Date: "2026-01-01", ShiftID: "shift2"},
	}

	tests := []struct {
		name     string
		staffID  string
		date     string
		expected bool
	}{
		{
			name:     "已占位",
			staffID:  "staff1",
			date:     "2026-01-01",
			expected: true,
		},
		{
			name:     "未占位",
			staffID:  "staff1",
			date:     "2026-01-02",
			expected: false,
		},
		{
			name:     "空切片-未占位",
			staffID:  "staff999",
			date:     "2026-01-01",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStaffOccupiedOnDate(slots, tt.staffID, tt.date)
			if result != tt.expected {
				t.Errorf("期望 %v，得到 %v", tt.expected, result)
			}
		})
	}
}

func TestFilterOccupiedByStaff(t *testing.T) {
	slots := []StaffOccupiedSlot{
		{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
		{StaffID: "staff2", Date: "2026-01-01", ShiftID: "shift2"},
		{StaffID: "staff1", Date: "2026-01-02", ShiftID: "shift3"},
		{StaffID: "staff1", Date: "2026-01-03", ShiftID: "shift1"},
	}

	tests := []struct {
		name          string
		staffID       string
		expectedCount int
	}{
		{
			name:          "过滤staff1-应有3条",
			staffID:       "staff1",
			expectedCount: 3,
		},
		{
			name:          "过滤staff2-应有1条",
			staffID:       "staff2",
			expectedCount: 1,
		},
		{
			name:          "过滤不存在的staff-应有0条",
			staffID:       "staff999",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterOccupiedByStaff(slots, tt.staffID)
			if len(result) != tt.expectedCount {
				t.Errorf("期望 %d 条记录，得到 %d 条", tt.expectedCount, len(result))
			}
			// 验证所有记录的 StaffID 都匹配
			for _, slot := range result {
				if slot.StaffID != tt.staffID {
					t.Errorf("过滤结果包含错误的 StaffID: %s", slot.StaffID)
				}
			}
		})
	}
}

func TestAddOccupiedSlotIfNotExists(t *testing.T) {
	tests := []struct {
		name          string
		initial       []StaffOccupiedSlot
		newSlot       StaffOccupiedSlot
		expectedCount int
		shouldAdd     bool
	}{
		{
			name:    "添加到空切片",
			initial: []StaffOccupiedSlot{},
			newSlot: StaffOccupiedSlot{
				StaffID: "staff1",
				Date:    "2026-01-01",
				ShiftID: "shift1",
			},
			expectedCount: 1,
			shouldAdd:     true,
		},
		{
			name: "添加新记录-不冲突",
			initial: []StaffOccupiedSlot{
				{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
			},
			newSlot: StaffOccupiedSlot{
				StaffID: "staff1",
				Date:    "2026-01-02",
				ShiftID: "shift2",
			},
			expectedCount: 2,
			shouldAdd:     true,
		},
		{
			name: "重复添加-相同staffID和date",
			initial: []StaffOccupiedSlot{
				{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
			},
			newSlot: StaffOccupiedSlot{
				StaffID: "staff1",
				Date:    "2026-01-01",
				ShiftID: "shift2", // 不同班次，但日期冲突
			},
			expectedCount: 1,
			shouldAdd:     false,
		},
		{
			name: "不同人员同一天-允许添加",
			initial: []StaffOccupiedSlot{
				{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
			},
			newSlot: StaffOccupiedSlot{
				StaffID: "staff2",
				Date:    "2026-01-01",
				ShiftID: "shift1",
			},
			expectedCount: 2,
			shouldAdd:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddOccupiedSlotIfNotExists(tt.initial, tt.newSlot)
			if len(result) != tt.expectedCount {
				t.Errorf("期望 %d 条记录，得到 %d 条", tt.expectedCount, len(result))
			}
			// 验证是否添加了新记录
			found := FindOccupiedSlot(result, tt.newSlot.StaffID, tt.newSlot.Date)
			if tt.shouldAdd && found == nil {
				t.Errorf("应该添加记录但未找到")
			}
		})
	}
}

// ============================================================
// ShiftDateRequirement 测试
// ============================================================

func TestFindRequirement(t *testing.T) {
	reqs := []ShiftDateRequirement{
		{ShiftID: "shift1", Date: "2026-01-01", Count: 5},
		{ShiftID: "shift1", Date: "2026-01-02", Count: 3},
		{ShiftID: "shift2", Date: "2026-01-01", Count: 4},
	}

	tests := []struct {
		name     string
		shiftID  string
		date     string
		expected *ShiftDateRequirement
	}{
		{
			name:     "找到需求",
			shiftID:  "shift1",
			date:     "2026-01-01",
			expected: &reqs[0],
		},
		{
			name:     "找到另一个日期",
			shiftID:  "shift1",
			date:     "2026-01-02",
			expected: &reqs[1],
		},
		{
			name:     "未找到-shiftID不存在",
			shiftID:  "shift999",
			date:     "2026-01-01",
			expected: nil,
		},
		{
			name:     "未找到-date不存在",
			shiftID:  "shift1",
			date:     "2026-01-99",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindRequirement(reqs, tt.shiftID, tt.date)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("期望 nil，得到 %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("期望 %+v，得到 nil", tt.expected)
				} else if result.ShiftID != tt.expected.ShiftID || result.Date != tt.expected.Date || result.Count != tt.expected.Count {
					t.Errorf("期望 %+v，得到 %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestFilterRequirementsByShift(t *testing.T) {
	reqs := []ShiftDateRequirement{
		{ShiftID: "shift1", Date: "2026-01-01", Count: 5},
		{ShiftID: "shift1", Date: "2026-01-02", Count: 3},
		{ShiftID: "shift2", Date: "2026-01-01", Count: 4},
		{ShiftID: "shift1", Date: "2026-01-03", Count: 2},
	}

	tests := []struct {
		name          string
		shiftID       string
		expectedCount int
	}{
		{
			name:          "过滤shift1-应有3条",
			shiftID:       "shift1",
			expectedCount: 3,
		},
		{
			name:          "过滤shift2-应有1条",
			shiftID:       "shift2",
			expectedCount: 1,
		},
		{
			name:          "过滤不存在的shift-应有0条",
			shiftID:       "shift999",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterRequirementsByShift(reqs, tt.shiftID)
			if len(result) != tt.expectedCount {
				t.Errorf("期望 %d 条记录，得到 %d 条", tt.expectedCount, len(result))
			}
			// 验证所有记录的 ShiftID 都匹配
			for _, req := range result {
				if req.ShiftID != tt.shiftID {
					t.Errorf("过滤结果包含错误的 ShiftID: %s", req.ShiftID)
				}
			}
		})
	}
}

// ============================================================
// CtxFixedShiftAssignment 测试
// ============================================================

func TestFindFixedAssignment(t *testing.T) {
	assigns := []CtxFixedShiftAssignment{
		{ShiftID: "shift1", Date: "2026-01-01", StaffIDs: []string{"staff1", "staff2"}},
		{ShiftID: "shift1", Date: "2026-01-02", StaffIDs: []string{"staff3"}},
		{ShiftID: "shift2", Date: "2026-01-01", StaffIDs: []string{"staff4", "staff5"}},
	}

	tests := []struct {
		name            string
		shiftID         string
		date            string
		expectedStaffID []string
	}{
		{
			name:            "找到固定排班",
			shiftID:         "shift1",
			date:            "2026-01-01",
			expectedStaffID: []string{"staff1", "staff2"},
		},
		{
			name:            "找到另一个日期",
			shiftID:         "shift1",
			date:            "2026-01-02",
			expectedStaffID: []string{"staff3"},
		},
		{
			name:            "未找到",
			shiftID:         "shift999",
			date:            "2026-01-01",
			expectedStaffID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFixedAssignment(assigns, tt.shiftID, tt.date)
			if tt.expectedStaffID == nil {
				if result != nil {
					t.Errorf("期望 nil，得到 %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("期望 %+v，得到 nil", tt.expectedStaffID)
				} else if len(result) != len(tt.expectedStaffID) {
					t.Errorf("期望 %d 个人员，得到 %d 个", len(tt.expectedStaffID), len(result))
				}
			}
		})
	}
}

func TestFindFixedAssignmentStruct(t *testing.T) {
	assigns := []CtxFixedShiftAssignment{
		{ShiftID: "shift1", Date: "2026-01-01", StaffIDs: []string{"staff1", "staff2"}},
		{ShiftID: "shift2", Date: "2026-01-01", StaffIDs: []string{"staff3"}},
	}

	tests := []struct {
		name      string
		shiftID   string
		date      string
		expectNil bool
	}{
		{
			name:      "找到结构体",
			shiftID:   "shift1",
			date:      "2026-01-01",
			expectNil: false,
		},
		{
			name:      "未找到",
			shiftID:   "shift999",
			date:      "2026-01-01",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFixedAssignmentStruct(assigns, tt.shiftID, tt.date)
			if tt.expectNil {
				if result != nil {
					t.Errorf("期望 nil，得到 %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("期望非 nil，得到 nil")
				}
			}
		})
	}
}

// ============================================================
// 类型转换测试
// ============================================================

func TestConvertOccupiedSlotsToMap(t *testing.T) {
	slots := []StaffOccupiedSlot{
		{StaffID: "staff1", Date: "2026-01-01", ShiftID: "shift1"},
		{StaffID: "staff1", Date: "2026-01-02", ShiftID: "shift2"},
		{StaffID: "staff2", Date: "2026-01-01", ShiftID: "shift3"},
	}

	result := ConvertOccupiedSlotsToMap(slots)

	// 验证结构
	if len(result) != 2 {
		t.Errorf("期望 2 个人员，得到 %d 个", len(result))
	}

	// 验证 staff1 的数据
	if staff1Map, ok := result["staff1"]; ok {
		if len(staff1Map) != 2 {
			t.Errorf("staff1 应有 2 条记录，得到 %d 条", len(staff1Map))
		}
		if staff1Map["2026-01-01"] != "shift1" {
			t.Errorf("staff1/2026-01-01 应为 shift1，得到 %s", staff1Map["2026-01-01"])
		}
	} else {
		t.Error("未找到 staff1")
	}

	// 验证 staff2 的数据
	if staff2Map, ok := result["staff2"]; ok {
		if len(staff2Map) != 1 {
			t.Errorf("staff2 应有 1 条记录，得到 %d 条", len(staff2Map))
		}
	} else {
		t.Error("未找到 staff2")
	}
}

func TestConvertRequirementsToMap(t *testing.T) {
	reqs := []ShiftDateRequirement{
		{ShiftID: "shift1", Date: "2026-01-01", Count: 5},
		{ShiftID: "shift1", Date: "2026-01-02", Count: 3},
		{ShiftID: "shift2", Date: "2026-01-01", Count: 4},
	}

	result := ConvertRequirementsToMap(reqs)

	// 验证结构
	if len(result) != 2 {
		t.Errorf("期望 2 个班次，得到 %d 个", len(result))
	}

	// 验证 shift1 的数据
	if shift1Map, ok := result["shift1"]; ok {
		if len(shift1Map) != 2 {
			t.Errorf("shift1 应有 2 条记录，得到 %d 条", len(shift1Map))
		}
		if shift1Map["2026-01-01"] != 5 {
			t.Errorf("shift1/2026-01-01 应为 5，得到 %d", shift1Map["2026-01-01"])
		}
		if shift1Map["2026-01-02"] != 3 {
			t.Errorf("shift1/2026-01-02 应为 3，得到 %d", shift1Map["2026-01-02"])
		}
	} else {
		t.Error("未找到 shift1")
	}
}

func TestConvertFixedAssignmentsToMap(t *testing.T) {
	assigns := []CtxFixedShiftAssignment{
		{ShiftID: "shift1", Date: "2026-01-01", StaffIDs: []string{"staff1", "staff2"}},
		{ShiftID: "shift2", Date: "2026-01-01", StaffIDs: []string{"staff3"}},
		{ShiftID: "shift1", Date: "2026-01-02", StaffIDs: []string{"staff4"}},
	}

	result := ConvertFixedAssignmentsToMap(assigns)

	// 验证结构
	if len(result) != 2 {
		t.Errorf("期望 2 个日期，得到 %d 个", len(result))
	}

	// 验证 2026-01-01 的数据（合并了 shift1 和 shift2）
	if staffIDs, ok := result["2026-01-01"]; ok {
		if len(staffIDs) != 3 {
			t.Errorf("2026-01-01 应有 3 个人员，得到 %d 个", len(staffIDs))
		}
	} else {
		t.Error("未找到 2026-01-01")
	}

	// 验证 2026-01-02 的数据
	if staffIDs, ok := result["2026-01-02"]; ok {
		if len(staffIDs) != 1 {
			t.Errorf("2026-01-02 应有 1 个人员，得到 %d 个", len(staffIDs))
		}
	} else {
		t.Error("未找到 2026-01-02")
	}
}

// ============================================================
// 空输入测试（边界情况）
// ============================================================

func TestEmptyInputs(t *testing.T) {
	// 空切片应返回空结果，不应 panic
	t.Run("FindOccupiedSlot-空切片", func(t *testing.T) {
		result := FindOccupiedSlot([]StaffOccupiedSlot{}, "staff1", "2026-01-01")
		if result != nil {
			t.Error("空切片应返回 nil")
		}
	})

	t.Run("FilterOccupiedByStaff-空切片", func(t *testing.T) {
		result := FilterOccupiedByStaff([]StaffOccupiedSlot{}, "staff1")
		if len(result) != 0 {
			t.Error("空切片应返回空切片")
		}
	})

	t.Run("ConvertOccupiedSlotsToMap-空切片", func(t *testing.T) {
		result := ConvertOccupiedSlotsToMap([]StaffOccupiedSlot{})
		if len(result) != 0 {
			t.Error("空切片应返回空 map")
		}
	})
}
