package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ShiftFixedAssignmentEntity 班次固定人员配置数据库实体（对应shift_fixed_assignments表）
type ShiftFixedAssignmentEntity struct {
	ID            string     `gorm:"primaryKey;type:varchar(64)"`
	ShiftID       string     `gorm:"index;type:varchar(64);not null"`        // 班次ID
	StaffID       string     `gorm:"index;type:varchar(64);not null"`        // 人员ID (EmployeeID)
	PatternType   string     `gorm:"type:varchar(32);not null"`              // 模式类型: weekly, monthly, specific
	Weekdays      IntArray   `gorm:"type:json"`                              // 周模式: 星期几 (1-7)
	WeekPattern   string     `gorm:"type:varchar(32)"`                       // 周模式: every, odd, even
	Monthdays     IntArray   `gorm:"type:json"`                              // 月模式: 每月几号 (1-31)
	SpecificDates StringArray `gorm:"type:json"`                             // 指定日期模式: 日期列表
	StartDate     *time.Time `gorm:"type:date"`                              // 生效开始日期
	EndDate       *time.Time `gorm:"type:date"`                              // 生效结束日期
	IsActive      bool       `gorm:"default:true"`                           // 是否启用
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (ShiftFixedAssignmentEntity) TableName() string {
	return "shift_fixed_assignments"
}

// IntArray 自定义类型用于存储整数数组
type IntArray []int

// Scan 实现 sql.Scanner 接口
func (a *IntArray) Scan(value interface{}) error {
	if value == nil {
		*a = []int{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, a)
}

// Value 实现 driver.Valuer 接口
func (a IntArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// StringArray 自定义类型用于存储字符串数组
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, a)
}

// Value 实现 driver.Valuer 接口
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

