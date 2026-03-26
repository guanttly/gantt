package schedule

import (
	"encoding/json"
	"time"

	"gantt-saas/internal/tenant"
)

// 排班计划状态。
const (
	StatusDraft      = "draft"
	StatusGenerating = "generating"
	StatusReview     = "review"
	StatusPublished  = "published"
	StatusArchived   = "archived"
)

// 排班 Pipeline 类型。
const (
	PipelineDeterministic = "deterministic"
	PipelineAdjust        = "adjust"
	PipelineAIAssisted    = "ai_assisted"
)

// Schedule 排班计划模型。
type Schedule struct {
	ID           string          `gorm:"primaryKey;size:64" json:"id"`
	Name         string          `gorm:"size:128;not null" json:"name"`
	StartDate    string          `gorm:"size:10;not null" json:"start_date"`
	EndDate      string          `gorm:"size:10;not null" json:"end_date"`
	Status       string          `gorm:"size:16;not null;default:draft" json:"status"`
	PipelineType string          `gorm:"size:32;not null;default:deterministic" json:"pipeline_type"`
	Config       json.RawMessage `gorm:"type:json" json:"config,omitempty"`
	CreatedBy    string          `gorm:"size:64;not null" json:"created_by"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	tenant.TenantModel
}

// TableName 指定表名。
func (Schedule) TableName() string { return "schedules" }

// Assignment 排班分配 DB 模型。
type Assignment struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	ScheduleID string    `gorm:"size:64;not null" json:"schedule_id"`
	EmployeeID string    `gorm:"size:64;not null" json:"employee_id"`
	ShiftID    string    `gorm:"size:64;not null" json:"shift_id"`
	Date       string    `gorm:"size:10;not null" json:"date"`
	Source     string    `gorm:"size:16;not null" json:"source"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	tenant.TenantModel
}

// TableName 指定表名。
func (Assignment) TableName() string { return "schedule_assignments" }

type SelfAssignmentView struct {
	ID           string `json:"id"`
	ScheduleID   string `json:"schedule_id"`
	ScheduleName string `json:"schedule_name"`
	EmployeeID   string `json:"employee_id"`
	ShiftID      string `json:"shift_id"`
	ShiftName    string `json:"shift_name"`
	ShiftColor   string `json:"shift_color"`
	Date         string `json:"date"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Source       string `json:"source"`
	Status       string `json:"status"`
}

// Change 排班变更记录 DB 模型。
type Change struct {
	ID           string          `gorm:"primaryKey;size:64" json:"id"`
	ScheduleID   string          `gorm:"size:64;not null" json:"schedule_id"`
	AssignmentID *string         `gorm:"size:64" json:"assignment_id,omitempty"`
	ChangeType   string          `gorm:"size:16;not null" json:"change_type"`
	BeforeData   json.RawMessage `gorm:"type:json" json:"before_data,omitempty"`
	AfterData    json.RawMessage `gorm:"type:json" json:"after_data,omitempty"`
	Reason       *string         `gorm:"size:256" json:"reason,omitempty"`
	ChangedBy    string          `gorm:"size:64;not null" json:"changed_by"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`

	tenant.TenantModel
}

// TableName 指定表名。
func (Change) TableName() string { return "schedule_changes" }
