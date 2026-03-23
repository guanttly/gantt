package model

import "time"

// PatternType 固定人员配置模式类型
type PatternType string

const (
	PatternTypeWeekly   PatternType = "weekly"   // 按周重复
	PatternTypeMonthly  PatternType = "monthly"  // 按月重复
	PatternTypeSpecific PatternType = "specific" // 指定日期
)

// WeekPattern 周重复模式
type WeekPattern string

const (
	WeekPatternEvery WeekPattern = "every" // 每周
	WeekPatternOdd   WeekPattern = "odd"   // 奇数周
	WeekPatternEven  WeekPattern = "even"  // 偶数周
)

// ShiftFixedAssignment 班次固定人员配置
type ShiftFixedAssignment struct {
	ID        string    `json:"id"`
	ShiftID   string    `json:"shiftId"`
	StaffID   string    `json:"staffId"`
	StaffName string    `json:"staffName,omitempty"` // 冗余字段，方便显示

	PatternType   PatternType `json:"patternType"`
	Weekdays      []int       `json:"weekdays,omitempty"`      // [1,3,5] = 周一、三、五 (1=周一, 7=周日)
	WeekPattern   WeekPattern `json:"weekPattern,omitempty"`   // every=每周, odd=奇数周, even=偶数周
	Monthdays     []int       `json:"monthdays,omitempty"`     // [1,15,30] = 每月1号、15号、30号
	SpecificDates []string    `json:"specificDates,omitempty"` // ["2025-01-01", "2025-01-05"]

	StartDate *time.Time `json:"startDate,omitempty"` // 生效开始日期
	EndDate   *time.Time `json:"endDate,omitempty"`   // 生效结束日期（NULL表示永久生效）
	IsActive  bool       `json:"isActive"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// CreateShiftFixedAssignmentRequest 创建固定人员配置请求
type CreateShiftFixedAssignmentRequest struct {
	ShiftID       string      `json:"shiftId" binding:"required"`
	StaffID       string      `json:"staffId" binding:"required"`
	PatternType   PatternType `json:"patternType" binding:"required,oneof=weekly monthly specific"`
	Weekdays      []int       `json:"weekdays"`
	WeekPattern   WeekPattern `json:"weekPattern" binding:"omitempty,oneof=every odd even"`
	Monthdays     []int       `json:"monthdays"`
	SpecificDates []string    `json:"specificDates"`
	StartDate     *time.Time  `json:"startDate"`
	EndDate       *time.Time  `json:"endDate"`
}

// UpdateShiftFixedAssignmentRequest 更新固定人员配置请求
type UpdateShiftFixedAssignmentRequest struct {
	PatternType   PatternType `json:"patternType" binding:"oneof=weekly monthly specific"`
	Weekdays      []int       `json:"weekdays"`
	WeekPattern   WeekPattern `json:"weekPattern" binding:"omitempty,oneof=every odd even"`
	Monthdays     []int       `json:"monthdays"`
	SpecificDates []string    `json:"specificDates"`
	StartDate     *time.Time  `json:"startDate"`
	EndDate       *time.Time  `json:"endDate"`
	IsActive      bool        `json:"isActive"`
}

// BatchCreateShiftFixedAssignmentsRequest 批量创建固定人员配置请求
type BatchCreateShiftFixedAssignmentsRequest struct {
	ShiftID     string                                `json:"shiftId" binding:"required"`
	Assignments []CreateShiftFixedAssignmentRequest   `json:"assignments" binding:"required"`
}

// ListShiftFixedAssignmentsRequest 查询固定人员配置列表请求
type ListShiftFixedAssignmentsRequest struct {
	ShiftID     string
	StaffID     string
	PatternType PatternType
	IsActive    *bool
	StartDate   *time.Time // 查询在此日期范围内生效的配置
	EndDate     *time.Time
}

// ShiftFixedAssignmentWithStaff 带人员信息的固定配置
type ShiftFixedAssignmentWithStaff struct {
	ShiftFixedAssignment
	StaffName   string `json:"staffName"`
	StaffCode   string `json:"staffCode"`
	DepartmentName string `json:"departmentName,omitempty"`
}

// CalculatedScheduleDate 计算出的排班日期
type CalculatedScheduleDate struct {
	Date     string   `json:"date"`     // 2025-01-01
	StaffIDs []string `json:"staffIds"` // 该日期固定的人员ID列表
}

// CalculatedScheduleResult 固定排班计算结果
type CalculatedScheduleResult struct {
	ShiftID   string                    `json:"shiftId"`
	ShiftName string                    `json:"shiftName,omitempty"`
	Dates     []CalculatedScheduleDate  `json:"dates"`
	TotalDays int                       `json:"totalDays"` // 总天数
}

