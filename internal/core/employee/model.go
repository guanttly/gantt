package employee

import (
	"time"

	"gantt-saas/internal/tenant"
)

const (
	StatusActive            = "active"
	StatusInactive          = "inactive"
	SchedulingRoleEmployee  = "employee"
	SchedulingRoleScheduler = "scheduler"
)

// Employee 员工模型。
type Employee struct {
	ID                 string    `gorm:"primaryKey;size:64" json:"id"`
	Name               string    `gorm:"size:64;not null" json:"name"`
	EmployeeNo         *string   `gorm:"size:32" json:"employee_no"`
	Phone              *string   `gorm:"size:20" json:"phone"`
	Email              *string   `gorm:"size:128" json:"email"`
	Position           *string   `gorm:"size:64" json:"position"`
	Category           *string   `gorm:"size:32" json:"category"`
	SchedulingRole     string    `gorm:"size:16;not null;default:employee" json:"scheduling_role"`
	AppPasswordHash    *string   `gorm:"size:256" json:"-"`
	AppMustResetPwd    bool      `gorm:"not null;default:true" json:"app_must_reset_pwd"`
	Status             string    `gorm:"size:16;not null;default:active" json:"status"`
	HireDate           *string   `gorm:"size:10" json:"hire_date"`
	AppDefaultPassword *string   `gorm:"-" json:"app_default_password,omitempty"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	tenant.TenantModel
}

// EmployeeResponse 员工 API 响应（带组织路径信息）。
type EmployeeResponse struct {
	Employee
	OrgNodeName        string `json:"org_node_name"`
	OrgNodePathDisplay string `json:"org_node_path_display"`
	OrgNodeType        string `json:"org_node_type"`
}

// TableName 指定表名。
func (Employee) TableName() string {
	return "employees"
}

// IsActive 员工是否在职。
func (e *Employee) IsActive() bool {
	return e.Status == StatusActive
}
