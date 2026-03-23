package entity

import (
	"time"
)

// LeaveRecordEntity 假期记录数据库实体（对应leave_records表）
// 记录已成事实的请假情况，不包含审批流
type LeaveRecordEntity struct {
	ID         string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID      string     `gorm:"index;type:varchar(64);not null"`
	EmployeeID string     `gorm:"index;type:varchar(64);not null"`
	Type       string     `gorm:"type:varchar(32);not null"` // 假期类型
	StartDate  time.Time  `gorm:"type:date;not null;index"`  // 开始日期
	EndDate    time.Time  `gorm:"type:date;not null;index"`  // 结束日期
	Days       float64    `gorm:"not null"`                  // 请假天数
	StartTime  *string    `gorm:"type:varchar(8)"`           // 开始时间（如：09:00，半天假使用）
	EndTime    *string    `gorm:"type:varchar(8)"`           // 结束时间（如：18:00，半天假使用）
	Reason     string     `gorm:"type:text"`                 // 请假原因
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
	DeletedAt  *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (LeaveRecordEntity) TableName() string {
	return "leave_records"
}

// LeaveBalanceEntity 假期余额数据库实体（对应leave_balances表）
type LeaveBalanceEntity struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	OrgID      string    `gorm:"index;type:varchar(64);not null"`
	EmployeeID string    `gorm:"uniqueIndex:idx_employee_type;type:varchar(64);not null"`
	Type       string    `gorm:"uniqueIndex:idx_employee_type;type:varchar(32);not null"`
	Year       int       `gorm:"index"`
	Total      float64   `gorm:""`
	Used       float64   `gorm:""`
	Remaining  float64   `gorm:""`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (LeaveBalanceEntity) TableName() string {
	return "leave_balances"
}

// HolidayEntity 节假日数据库实体（对应holidays表）
type HolidayEntity struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	OrgID       string    `gorm:"index;type:varchar(64)"`
	Name        string    `gorm:"type:varchar(128);not null"`
	Date        time.Time `gorm:"type:date;uniqueIndex:idx_org_date;not null"`
	Type        string    `gorm:"type:varchar(32);not null"`
	Description string    `gorm:"type:text"`
	Year        int       `gorm:"index"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (HolidayEntity) TableName() string {
	return "holidays"
}
