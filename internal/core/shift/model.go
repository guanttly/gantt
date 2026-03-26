package shift

import (
"time"

"gantt-saas/internal/tenant"
)

const (
StatusActive   = "active"
StatusDisabled = "disabled"

DepTypeSource = "source"
DepTypeOrder  = "order"
)

// Shift 班次模型。
type Shift struct {
ID         string    `gorm:"primaryKey;size:64" json:"id"`
Name       string    `gorm:"size:64;not null" json:"name"`
Code       string    `gorm:"size:16;not null" json:"code"`
StartTime  string    `gorm:"size:8;not null" json:"start_time"`
EndTime    string    `gorm:"size:8;not null" json:"end_time"`
Duration   int       `gorm:"not null" json:"duration"`
IsCrossDay bool      `gorm:"not null;default:false" json:"is_cross_day"`
Color      string    `gorm:"size:16;default:#409EFF" json:"color"`
Priority   int       `gorm:"not null;default:0" json:"priority"`
Status     string    `gorm:"size:16;not null;default:active" json:"status"`
CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (Shift) TableName() string {
return "shifts"
}

// ShiftDependency 班次依赖关系。
type ShiftDependency struct {
ID             string    `gorm:"primaryKey;size:64" json:"id"`
ShiftID        string    `gorm:"size:64;not null" json:"shift_id"`
DependsOnID    string    `gorm:"size:64;not null" json:"depends_on_id"`
DependencyType string    `gorm:"size:16;not null" json:"dependency_type"`
CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

tenant.TenantModel
}

// TableName 指定表名。
func (ShiftDependency) TableName() string {
return "shift_dependencies"
}
