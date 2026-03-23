package entity

import (
	"time"
)

// SystemSettingEntity 系统设置数据库实体（对应system_settings表）
type SystemSettingEntity struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string    `gorm:"index:idx_org_key;type:varchar(64);not null"`
	Key         string    `gorm:"index:idx_org_key;type:varchar(128);not null;column:key"` // key是MySQL保留关键字，在SQL查询中需要用反引号包裹
	Value       string    `gorm:"type:text;not null"`
	Description string    `gorm:"type:varchar(512)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (SystemSettingEntity) TableName() string {
	return "system_settings"
}

