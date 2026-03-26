package rule

import (
	"encoding/json"
	"time"

	"gantt-saas/internal/tenant"
)

// 规则分类。
const (
	CategoryConstraint = "constraint" // 约束规则
	CategoryPreference = "preference" // 偏好规则
	CategoryDependency = "dependency" // 依赖规则
)

// 规则子类型。
const (
	SubTypeForbid     = "forbid"     // 排他/禁止
	SubTypeLimit      = "limit"      // 数量限制
	SubTypeMust       = "must"       // 必须
	SubTypePrefer     = "prefer"     // 偏好
	SubTypeCombinable = "combinable" // 可组合
	SubTypeSource     = "source"     // 人员来源
	SubTypeOrder      = "order"      // 执行顺序
	SubTypeMinRest    = "min_rest"   // 最小休息
)

// 关联目标类型。
const (
	TargetTypeShift    = "shift"
	TargetTypeGroup    = "group"
	TargetTypeEmployee = "employee"
)

// Rule 规则模型。
type Rule struct {
	ID             string          `gorm:"primaryKey;size:64" json:"id"`
	Name           string          `gorm:"size:128;not null" json:"name"`
	Category       string          `gorm:"size:32;not null" json:"category"`
	SubType        string          `gorm:"size:32;not null" json:"sub_type"`
	Config         json.RawMessage `gorm:"type:json;not null" json:"config"`
	Priority       int             `gorm:"not null;default:0" json:"priority"`
	IsEnabled      bool            `gorm:"not null;default:true" json:"is_enabled"`
	OverrideRuleID *string         `gorm:"size:64" json:"override_rule_id,omitempty"`
	Description    *string         `gorm:"size:512" json:"description,omitempty"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	tenant.TenantModel
}

// TableName 指定表名。
func (Rule) TableName() string { return "rules" }

// RuleAssociation 规则关联模型。
type RuleAssociation struct {
	ID         string `gorm:"primaryKey;size:64" json:"id"`
	RuleID     string `gorm:"size:64;not null" json:"rule_id"`
	TargetType string `gorm:"size:32;not null" json:"target_type"`
	TargetID   string `gorm:"size:64;not null" json:"target_id"`
	tenant.TenantModel
}

// TableName 指定表名。
func (RuleAssociation) TableName() string { return "rule_associations" }

// RuleWithSource 带来源标记的规则（用于 API 响应）。
type RuleWithSource struct {
	Rule
	SourceNode    string `json:"source_node"`    // 来源节点名称
	IsInherited   bool   `json:"is_inherited"`   // 是否继承自上级
	IsOverridable bool   `json:"is_overridable"` // 是否可覆盖
}

// ── config JSON 结构定义 ──────────────────────────────

// ExclusiveShiftsConfig 排他班次配置。
type ExclusiveShiftsConfig struct {
	Type     string   `json:"type"`      // "exclusive_shifts"
	ShiftIDs []string `json:"shift_ids"` // 互斥的班次 ID 列表
	Scope    string   `json:"scope"`     // same_day / consecutive
}

// MaxCountConfig 最大次数配置。
type MaxCountConfig struct {
	Type    string `json:"type"`     // "max_count"
	ShiftID string `json:"shift_id"` // 目标班次 ID
	Max     int    `json:"max"`      // 最大次数
	Period  string `json:"period"`   // week / month
}

// MinRestConfig 最小休息天数配置。
type MinRestConfig struct {
	Type string `json:"type"` // "min_rest"
	Days int    `json:"days"` // 最小休息天数
}

// RequiredTogetherConfig 必须同时配置。
type RequiredTogetherConfig struct {
	Type        string   `json:"type"`         // "required_together"
	EmployeeIDs []string `json:"employee_ids"` // 必须同时排班的员工
	ShiftID     string   `json:"shift_id"`     // 目标班次
}

// PreferEmployeeConfig 偏好配置。
type PreferEmployeeConfig struct {
	Type       string `json:"type"`        // "prefer_employee"
	EmployeeID string `json:"employee_id"` // 偏好的员工
	ShiftID    string `json:"shift_id"`    // 目标班次
	Weight     int    `json:"weight"`      // 偏好权重
}

// StaffSourceConfig 人员来源配置。
type StaffSourceConfig struct {
	Type          string `json:"type"`            // "staff_source"
	TargetShiftID string `json:"target_shift_id"` // 目标班次
	SourceShiftID string `json:"source_shift_id"` // 来源班次
}

// ExecutionOrderConfig 执行顺序配置。
type ExecutionOrderConfig struct {
	Type          string `json:"type"`            // "execution_order"
	BeforeShiftID string `json:"before_shift_id"` // 先执行班次
	AfterShiftID  string `json:"after_shift_id"`  // 后执行班次
}
