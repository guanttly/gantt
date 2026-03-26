package employee

import (
"time"

"gantt-saas/internal/tenant"
)

const (
StatusActive   = "active"
StatusInactive = "inactive"
)

// Employee 员工模型。
type Employee struct {
ID         string    `gorm:"primaryKey;size:64" json:"id"`
Name       string    `gorm:"size:64;not null" json:"name"`
EmployeeNo *string   `gorm:"size:32" json:"employee_no"`
Phone      *string   `gorm:"size:20" json:"phone"`
Email      *string   `gorm:"size:128" json:"email"`
Position   *string   `gorm:"size:64" json:"position"`
Category   *string   `gorm:"size:32" json:"category"`
Status     string    `gorm:"size:16;not null;default:active" json:"status"`
HireDate   *string   `gorm:"size:10" json:"hire_date"`
CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (Employee) TableName() string {
return "employees"
}

// IsActive 员工是否在职。
func (e *Employee) IsActive() bool {
return e.Status == StatusActive
}
