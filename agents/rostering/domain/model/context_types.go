package model

// ============================================================
// V3排班上下文强类型结构定义
// 用于替代嵌套map结构，提供类型安全和更好的可维护性
// ============================================================

// StaffOccupiedSlot 人员已占位班次
// 替代: map[string]map[string]string (staffID -> date -> shiftID)
type StaffOccupiedSlot struct {
	// StaffID 人员ID
	StaffID string `json:"staffId"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称（冗余字段，便于展示）
	ShiftName string `json:"shiftName,omitempty"`

	// Source 占位来源
	// - "fixed": 固定排班
	// - "draft": 草稿排班
	// - "external": 外部导入
	Source string `json:"source,omitempty"`
}

// StaffScheduleMark 人员排班标记（用于时段冲突检查）
// 替代: map[string]map[string][]*ShiftMark (staffID -> date -> marks)
type StaffScheduleMark struct {
	// StaffID 人员ID
	StaffID string `json:"staffId"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称
	ShiftName string `json:"shiftName,omitempty"`

	// StartTime 开始时间 (HH:MM)
	StartTime string `json:"startTime"`

	// EndTime 结束时间 (HH:MM)
	EndTime string `json:"endTime"`

	// IsNextDay 是否跨天到次日
	IsNextDay bool `json:"isNextDay"`
}

// ShiftDateRequirement 班次日期人员需求
// 替代: map[string]map[string]int (shiftID -> date -> count)
type ShiftDateRequirement struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称（冗余字段，便于展示）
	ShiftName string `json:"shiftName,omitempty"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// Count 需求人数
	Count int `json:"count"`
}

// CtxFixedShiftAssignment 班次固定排班（上下文专用，避免与SDK模型冲突）
// 替代: map[string]map[string][]string (shiftID -> date -> staffIDs)
// 或: []utils.FixedShiftAssignment
type CtxFixedShiftAssignment struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// ShiftName 班次名称（冗余字段，便于展示）
	ShiftName string `json:"shiftName,omitempty"`

	// Date 日期 (YYYY-MM-DD)
	Date string `json:"date"`

	// StaffIDs 固定排班的人员ID列表
	StaffIDs []string `json:"staffIds"`
}

// ============================================================
// 批量操作辅助结构
// ============================================================

// OccupiedSlotBatch 批量占位信息（用于批量操作）
type OccupiedSlotBatch struct {
	// Slots 占位记录列表
	Slots []StaffOccupiedSlot `json:"slots"`

	// Timestamp 批次时间戳
	Timestamp string `json:"timestamp,omitempty"`
}

// RequirementsBatch 批量人员需求（用于批量操作）
type RequirementsBatch struct {
	// Requirements 需求记录列表
	Requirements []ShiftDateRequirement `json:"requirements"`

	// TotalCount 总需求人数
	TotalCount int `json:"totalCount,omitempty"`
}
