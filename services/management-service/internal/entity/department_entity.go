package entity

import (
	"time"
)

// DepartmentEntity 部门数据库实体
type DepartmentEntity struct {
	ID          string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string     `gorm:"index;type:varchar(64);not null"`                    // 组织ID
	Code        string     `gorm:"uniqueIndex:idx_org_code;type:varchar(64);not null"` // 部门编码
	Name        string     `gorm:"type:varchar(128);not null"`                         // 部门名称
	ParentID    *string    `gorm:"index;type:varchar(64)"`                             // 父部门ID
	Level       int        `gorm:"not null;default:1"`                                 // 层级
	Path        string     `gorm:"type:varchar(512)"`                                  // 部门路径
	Description string     `gorm:"type:text"`                                          // 部门描述
	ManagerID   *string    `gorm:"type:varchar(64)"`                                   // 部门经理员工ID
	SortOrder   int        `gorm:"not null;default:0"`                                 // 排序
	IsActive    bool       `gorm:"not null;default:true"`                              // 是否启用
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (DepartmentEntity) TableName() string {
	return "departments"
}
