package entity

import (
	"time"
)

// ShiftWeeklyStaffEntity 班次周默认人数数据库实体（对应shift_weekly_staff表）
// 支持班次按周一到周日单独配置默认人数
type ShiftWeeklyStaffEntity struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	ShiftID    string    `gorm:"uniqueIndex:idx_shift_weekday;type:varchar(64);not null"` // 班次ID
	Weekday    int       `gorm:"uniqueIndex:idx_shift_weekday;not null"`                  // 周几：0=周日,1=周一,...,6=周六
	StaffCount int       `gorm:"not null;default:1"`                                      // 默认人数
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (ShiftWeeklyStaffEntity) TableName() string {
	return "shift_weekly_staff"
}
