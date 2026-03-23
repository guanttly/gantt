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

