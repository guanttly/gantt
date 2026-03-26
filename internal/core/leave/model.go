package leave

import (
"time"

"gantt-saas/internal/tenant"
)

const (
StatusPending  = "pending"
StatusApproved = "approved"
StatusRejected = "rejected"
)

// Leave 请假模型。
type Leave struct {
ID         string    `gorm:"primaryKey;size:64" json:"id"`
EmployeeID string    `gorm:"size:64;not null" json:"employee_id"`
LeaveType  string    `gorm:"size:32;not null" json:"leave_type"`
StartDate  string    `gorm:"size:10;not null" json:"start_date"`
EndDate    string    `gorm:"size:10;not null" json:"end_date"`
Reason     *string   `gorm:"size:256" json:"reason"`
Status     string    `gorm:"size:16;not null;default:pending" json:"status"`
CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (Leave) TableName() string {
return "leaves"
}

// IsPending 是否待审批。
func (l *Leave) IsPending() bool {
return l.Status == StatusPending
}
