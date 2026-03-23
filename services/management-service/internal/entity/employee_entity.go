package entity

import (
	"time"
)

// EmployeeEntity 员工数据库实体（对应employees表）
type EmployeeEntity struct {
	ID           string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID        string     `gorm:"index;type:varchar(64);not null"`
	EmployeeID   string     `gorm:"uniqueIndex;type:varchar(64);not null"` // 工号
	UserID       string     `gorm:"index;type:varchar(64)"`                // 关联用户ID
	Name         string     `gorm:"type:varchar(128);not null"`
	Phone        string     `gorm:"type:varchar(32)"`
	Email        string     `gorm:"type:varchar(128)"`
	DepartmentID string     `gorm:"index;type:varchar(64)"` // 关联部门ID（关联到departments表）
	Position     string     `gorm:"type:varchar(128)"`      // 职位
	Role         string     `gorm:"type:varchar(64)"`       // 角色（权限相关）
	Status       string     `gorm:"type:varchar(32);default:'active'"`
	HireDate     *time.Time // 入职日期
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
	DeletedAt    *time.Time `gorm:"index"` // 软删除时间戳（GORM标准软删除字段）
}

// TableName 表名
func (EmployeeEntity) TableName() string {
	return "employees"
}
