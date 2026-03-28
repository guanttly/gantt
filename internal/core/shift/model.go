package shift

import (
	"encoding/json"
	"time"

	"gantt-saas/internal/tenant"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	DepTypeSource = "source"
	DepTypeOrder  = "order"

	ShiftTypeRegular  = "regular"
	ShiftTypeOvertime = "overtime"
	ShiftTypeOnCall   = "oncall"
	ShiftTypeStandby  = "standby"
)

// Shift 班次模型。
type Shift struct {
	ID                   string          `gorm:"primaryKey;size:64" json:"id"`
	Name                 string          `gorm:"size:64;not null" json:"name"`
	Code                 string          `gorm:"size:16;not null" json:"code"`
	Type                 string          `gorm:"size:32;not null;default:regular" json:"type"`
	StartTime            string          `gorm:"size:8;not null" json:"start_time"`
	EndTime              string          `gorm:"size:8;not null" json:"end_time"`
	Duration             int             `gorm:"not null" json:"duration"`
	IsCrossDay           bool            `gorm:"not null;default:false" json:"is_cross_day"`
	Color                string          `gorm:"size:16;default:#409EFF" json:"color"`
	Priority             int             `gorm:"column:priority;not null;default:0" json:"scheduling_priority"`
	Status               string          `gorm:"size:16;not null;default:active" json:"status"`
	Description          *string         `gorm:"type:text" json:"description,omitempty"`
	Metadata             json.RawMessage `gorm:"type:json" json:"metadata,omitempty"`
	CreatedAt            time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	IsActive             bool            `gorm:"-" json:"is_active"`
	WeeklyStaffSummary   string          `gorm:"-" json:"weekly_staff_summary,omitempty"`
	FixedStaffSummary    string          `gorm:"-" json:"fixed_staff_summary,omitempty"`
	GroupSummary         string          `gorm:"-" json:"group_summary,omitempty"`
	GroupNames           []string        `gorm:"-" json:"group_names,omitempty"`
	FixedAssignmentCount int64           `gorm:"-" json:"fixed_assignment_count,omitempty"`
	GroupCount           int64           `gorm:"-" json:"group_count,omitempty"`

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

// ShiftGroup 班次关联分组。
type ShiftGroup struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	ShiftID   string    `gorm:"size:64;not null;index:idx_shift_groups_shift" json:"shift_id"`
	GroupID   string    `gorm:"size:64;not null;index:idx_shift_groups_group" json:"group_id"`
	Priority  int       `gorm:"not null;default:0" json:"priority"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	Notes     *string   `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	GroupName *string   `gorm:"->;column:group_name" json:"group_name,omitempty"`
	GroupCode *string   `gorm:"->;column:group_code" json:"group_code,omitempty"`

	tenant.TenantModel
}

func (ShiftGroup) TableName() string { return "shift_groups" }

// FixedAssignment 班次固定人员配置。
type FixedAssignment struct {
	ID            string          `gorm:"primaryKey;size:64" json:"id,omitempty"`
	ShiftID       string          `gorm:"size:64;not null;index:idx_fixed_assignments_shift" json:"shift_id,omitempty"`
	EmployeeID    string          `gorm:"column:employee_id;size:64;not null;index:idx_fixed_assignments_employee" json:"staff_id"`
	PatternType   string          `gorm:"size:16;not null" json:"pattern_type"`
	Weekdays      json.RawMessage `gorm:"type:json" json:"weekdays,omitempty"`
	WeekPattern   *string         `gorm:"size:16" json:"week_pattern,omitempty"`
	Monthdays     json.RawMessage `gorm:"type:json" json:"monthdays,omitempty"`
	SpecificDates json.RawMessage `gorm:"type:json" json:"specific_dates,omitempty"`
	StartDate     *string         `gorm:"size:10" json:"start_date,omitempty"`
	EndDate       *string         `gorm:"size:10" json:"end_date,omitempty"`
	IsActive      bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	StaffName     *string         `gorm:"-" json:"staff_name,omitempty"`

	tenant.TenantModel
}

func (FixedAssignment) TableName() string { return "fixed_assignments" }

// ShiftWeeklyStaff 班次周人数配置。
type ShiftWeeklyStaff struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id,omitempty"`
	ShiftID    string    `gorm:"size:64;not null;index:idx_shift_weekly_staff_shift" json:"shift_id,omitempty"`
	Weekday    int       `gorm:"not null" json:"weekday"`
	StaffCount int       `gorm:"not null;default:0" json:"staff_count"`
	IsCustom   bool      `gorm:"not null;default:false" json:"is_custom"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`

	tenant.TenantModel
}

func (ShiftWeeklyStaff) TableName() string { return "shift_weekly_staff" }
