package entity

import (
	"time"
)

// TimePeriodEntity 时间段数据库实体（对应time_periods表）
// 用于定义检查量统计的时间段，支持用户自定义配置
type TimePeriodEntity struct {
	ID          string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string     `gorm:"index;type:varchar(64);not null"`
	Code        string     `gorm:"uniqueIndex:idx_time_period_org_code;type:varchar(64);not null"` // 时间段编码
	Name        string     `gorm:"type:varchar(128);not null"`                                     // 时间段名称，如"上午段"、"下午段"
	StartTime   string     `gorm:"type:varchar(8);not null"`                                       // 开始时间：HH:MM，如"20:00"
	EndTime     string     `gorm:"type:varchar(8);not null"`                                       // 结束时间：HH:MM，如"13:59"
	IsCrossDay  bool       `gorm:"default:false"`                                                  // 是否跨日（如前日20:00到当日13:59）
	Description string     `gorm:"type:text"`                                                      // 时间段说明
	IsActive    bool       `gorm:"default:true"`                                                   // 是否启用
	SortOrder   int        `gorm:"default:0"`                                                      // 排序序号
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (TimePeriodEntity) TableName() string {
	return "time_periods"
}
