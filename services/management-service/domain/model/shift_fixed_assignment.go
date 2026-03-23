package model

import (
	"time"
)

// ShiftFixedAssignment 班次固定人员配置领域模型
type ShiftFixedAssignment struct {
	ID            string     `json:"id"`
	ShiftID       string     `json:"shiftId"`       // 班次ID
	StaffID       string     `json:"staffId"`       // 人员ID (EmployeeID)
	PatternType   string     `json:"patternType"`   // 模式类型: weekly, monthly, specific
	Weekdays      []int      `json:"weekdays"`      // 周模式: 星期几 (1-7, 1=周一)
	WeekPattern   string     `json:"weekPattern"`   // 周模式: every, odd, even (每周/奇数周/偶数周)
	Monthdays     []int      `json:"monthdays"`     // 月模式: 每月几号 (1-31)
	SpecificDates []string   `json:"specificDates"` // 指定日期模式: 日期列表 (YYYY-MM-DD)
	StartDate     *time.Time `json:"startDate"`     // 生效开始日期
	EndDate       *time.Time `json:"endDate"`       // 生效结束日期
	IsActive      bool       `json:"isActive"`      // 是否启用
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

// 模式类型常量
const (
	PatternTypeWeekly   = "weekly"   // 按周重复
	PatternTypeMonthly  = "monthly"  // 按月重复
	PatternTypeSpecific = "specific" // 指定日期
)

// 周模式常量
const (
	WeekPatternEvery = "every" // 每周
	WeekPatternOdd   = "odd"   // 奇数周
	WeekPatternEven  = "even"  // 偶数周
)

// ShiftFixedAssignmentFilter 固定人员配置查询过滤器
type ShiftFixedAssignmentFilter struct {
	ShiftID  string
	StaffID  string
	IsActive *bool
}

