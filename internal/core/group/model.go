package group

import (
"time"

"gantt-saas/internal/tenant"
)

// EmployeeGroup 员工分组模型。
type EmployeeGroup struct {
ID          string    `gorm:"primaryKey;size:64" json:"id"`
Name        string    `gorm:"size:64;not null" json:"name"`
Description *string   `gorm:"size:256" json:"description"`
CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (EmployeeGroup) TableName() string {
return "employee_groups"
}

// GroupMember 分组成员模型。
type GroupMember struct {
ID         string    `gorm:"primaryKey;size:64" json:"id"`
GroupID    string    `gorm:"size:64;not null" json:"group_id"`
EmployeeID string    `gorm:"size:64;not null" json:"employee_id"`
CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (GroupMember) TableName() string {
return "group_members"
}
