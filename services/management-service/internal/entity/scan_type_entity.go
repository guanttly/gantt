package entity

import (
	"time"
)

// ScanTypeEntity 检查类型数据库实体（对应scan_types表）
// 放射科检查类型，如平扫、增强等
type ScanTypeEntity struct {
	ID          string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string     `gorm:"uniqueIndex:idx_scan_type_org_code;type:varchar(64);not null"`
	Code        string     `gorm:"uniqueIndex:idx_scan_type_org_code;type:varchar(64);not null"` // 类型编码
	Name        string     `gorm:"type:varchar(128);not null"`                                   // 类型名称
	Description string     `gorm:"type:text"`                                                    // 类型说明
	IsActive    bool       `gorm:"default:true"`                                                 // 是否启用
	SortOrder   int        `gorm:"default:0"`                                                    // 排序序号
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (ScanTypeEntity) TableName() string {
	return "scan_types"
}
