package rule

import (
	"encoding/json"
	"time"

	"gantt-saas/internal/tenant"
)

// 规则类型。
const (
	RuleTypeExclusive        = "exclusive"
	RuleTypeCombinable       = "combinable"
	RuleTypeRequiredTogether = "required_together"
	RuleTypePeriodic         = "periodic"
	RuleTypeMaxCount         = "maxCount"
	RuleTypeForbiddenDay     = "forbidden_day"
	RuleTypePreferred        = "preferred"
	RuleTypeSource           = "source"
	RuleTypeOrder            = "order"
	RuleTypeMinRestDays      = "min_rest"
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

// 规则作用域。
const (
	ApplyScopeGlobal   = "global"
	ApplyScopeSpecific = "specific"
)

// 规则时间范围。
const (
	TimeScopeSameDay   = "same_day"
	TimeScopeSameWeek  = "same_week"
	TimeScopeSameMonth = "same_month"
	TimeScopeCustom    = "custom"
)

// 规则来源。
const (
	SourceTypeManual    = "manual"
	SourceTypeLLMParsed = "llm_parsed"
	SourceTypeMigrated  = "migrated"
)

// 规则关联角色。
const (
	AssociationRoleSubject   = "subject"
	AssociationRoleObject    = "object"
	AssociationRoleTarget    = "target"
	AssociationRoleSource    = "source"
	AssociationRoleReference = "reference"
)

// 规则适用范围类型。
const (
	ScopeTypeAll             = "all"
	ScopeTypeEmployee        = "employee"
	ScopeTypeGroup           = "group"
	ScopeTypeExcludeEmployee = "exclude_employee"
	ScopeTypeExcludeGroup    = "exclude_group"
)

// 关联目标类型。
const (
	TargetTypeShift    = "shift"
	TargetTypeGroup    = "group"
	TargetTypeEmployee = "employee"
)

// Rule 规则模型。
type Rule struct {
	ID              string            `gorm:"primaryKey;size:64" json:"id"`
	Name            string            `gorm:"size:128;not null" json:"name"`
	Type            string            `gorm:"column:rule_type;size:32;index:idx_rule_type" json:"type"`
	RuleType        string            `gorm:"-" json:"rule_type,omitempty"`
	Category        string            `gorm:"size:32;not null" json:"category"`
	SubType         string            `gorm:"column:sub_type;size:32;not null" json:"sub_type"`
	SubCategory     string            `gorm:"-" json:"sub_category,omitempty"`
	ApplyScope      string            `gorm:"size:32;not null;default:global" json:"apply_scope"`
	TimeScope       string            `gorm:"size:32;not null;default:same_day" json:"time_scope"`
	TimeOffsetDays  *int              `gorm:"type:int" json:"time_offset_days,omitempty"`
	RuleData        string            `gorm:"size:512" json:"rule_data,omitempty"`
	Config          json.RawMessage   `gorm:"type:json;not null" json:"config"`
	Priority        int               `gorm:"not null;default:0" json:"priority"`
	IsEnabled       bool              `gorm:"column:is_enabled;not null;default:true" json:"is_enabled"`
	Enabled         bool              `gorm:"-" json:"enabled"`
	Disabled        bool              `gorm:"not null;default:false;index:idx_rule_disabled" json:"disabled"`
	DisabledBy      *string           `gorm:"size:64" json:"disabled_by,omitempty"`
	DisabledAt      *time.Time        `json:"disabled_at,omitempty"`
	DisabledReason  *string           `gorm:"type:text" json:"disabled_reason,omitempty"`
	OverrideRuleID  *string           `gorm:"size:64" json:"override_rule_id,omitempty"`
	SourceType      string            `gorm:"size:32;default:manual;index:idx_rule_source_type" json:"source_type,omitempty"`
	ParseConfidence *float64          `gorm:"type:decimal(4,3)" json:"parse_confidence,omitempty"`
	Version         string            `gorm:"size:8;default:v4;index:idx_rule_version" json:"version,omitempty"`
	Description     *string           `gorm:"size:512" json:"description,omitempty"`
	CreatedAt       time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	Associations    []RuleAssociation `gorm:"-" json:"associations,omitempty"`
	ApplyScopes     []RuleApplyScope  `gorm:"-" json:"apply_scopes,omitempty"`
	Dependencies    []RuleDependency  `gorm:"-" json:"dependencies,omitempty"`
	Conflicts       []RuleConflict    `gorm:"-" json:"conflicts,omitempty"`
	tenant.TenantModel
}

// TableName 指定表名。
func (Rule) TableName() string { return "rules" }

// RuleAssociation 规则关联模型。
type RuleAssociation struct {
	ID              string    `gorm:"primaryKey;size:64" json:"id"`
	RuleID          string    `gorm:"size:64;not null;index:idx_rule_assoc_rule" json:"rule_id"`
	TargetType      string    `gorm:"size:32;not null" json:"target_type,omitempty"`
	TargetID        string    `gorm:"size:64;not null" json:"target_id,omitempty"`
	AssociationType string    `gorm:"-" json:"association_type,omitempty"`
	AssociationID   string    `gorm:"-" json:"association_id,omitempty"`
	Role            string    `gorm:"size:32;default:target" json:"role,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	tenant.TenantModel
}

// TableName 指定表名。
func (RuleAssociation) TableName() string { return "rule_associations" }

// RuleApplyScope 规则适用范围。
type RuleApplyScope struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	RuleID    string    `gorm:"size:64;not null;index:idx_rule_scope_rule" json:"rule_id"`
	ScopeType string    `gorm:"size:32;not null" json:"scope_type"`
	ScopeID   *string   `gorm:"size:64" json:"scope_id,omitempty"`
	ScopeName *string   `gorm:"size:128" json:"scope_name,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	tenant.TenantModel
}

// TableName 指定表名。
func (RuleApplyScope) TableName() string { return "rule_apply_scopes" }

// RuleDependency 规则依赖关系。
type RuleDependency struct {
	ID                string    `gorm:"primaryKey;size:64" json:"id,omitempty"`
	DependentRuleID   string    `gorm:"size:64;not null;index:idx_rule_dep_rule" json:"dependent_rule_id,omitempty"`
	DependentOnRuleID string    `gorm:"size:64;not null;index:idx_rule_dep_on_rule" json:"dependent_on_rule_id,omitempty"`
	DependentRuleName string    `gorm:"-" json:"dependent_rule_name,omitempty"`
	DependentOnName   string    `gorm:"-" json:"dependent_on_rule_name,omitempty"`
	DependencyType    string    `gorm:"size:32;not null" json:"dependency_type"`
	Description       string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	tenant.TenantModel
}

// TableName 指定表名。
func (RuleDependency) TableName() string { return "rule_dependencies" }

// RuleConflict 规则冲突关系。
type RuleConflict struct {
	ID                 string    `gorm:"primaryKey;size:64" json:"id,omitempty"`
	RuleID1            string    `gorm:"column:rule_id_1;size:64;not null;index:idx_rule_conflict_rule1" json:"rule_id_1,omitempty"`
	RuleID2            string    `gorm:"column:rule_id_2;size:64;not null;index:idx_rule_conflict_rule2" json:"rule_id_2,omitempty"`
	RuleName1          string    `gorm:"-" json:"rule_name_1,omitempty"`
	RuleName2          string    `gorm:"-" json:"rule_name_2,omitempty"`
	ConflictType       string    `gorm:"size:32;not null" json:"conflict_type"`
	Description        string    `gorm:"type:text" json:"description,omitempty"`
	ResolutionPriority int       `gorm:"type:int;default:0" json:"resolution_priority,omitempty"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	tenant.TenantModel
}

// TableName 指定表名。
func (RuleConflict) TableName() string { return "rule_conflicts" }

// RuleWithSource 带来源标记的规则（用于 API 响应）。
type RuleWithSource struct {
	Rule
	SourceNode    string `json:"source_node"`    // 来源节点名称
	IsInherited   bool   `json:"is_inherited"`   // 是否继承自上级
	IsOverridable bool   `json:"is_overridable"` // 是否可覆盖
}

// NormalizeAliases 补齐前后端兼容字段。
func (r *Rule) NormalizeAliases() {
	if r.RuleType == "" {
		r.RuleType = r.Type
	}
	if r.Type == "" {
		r.Type = inferRuleTypeFromSubType(r.SubType)
		r.RuleType = r.Type
	}
	if r.SubCategory == "" {
		r.SubCategory = r.SubType
	}
	r.Enabled = r.IsEnabled
	for i := range r.Associations {
		r.Associations[i].NormalizeAliases()
	}
}

// NormalizeAliases 补齐关联字段别名。
func (a *RuleAssociation) NormalizeAliases() {
	if a.AssociationType == "" {
		a.AssociationType = a.TargetType
	}
	if a.AssociationID == "" {
		a.AssociationID = a.TargetID
	}
}

func inferRuleTypeFromSubType(subType string) string {
	switch subType {
	case SubTypeForbid:
		return RuleTypeExclusive
	case SubTypeLimit:
		return RuleTypeMaxCount
	case SubTypeMust:
		return RuleTypeRequiredTogether
	case SubTypePrefer:
		return RuleTypePreferred
	case SubTypeCombinable:
		return RuleTypeCombinable
	case SubTypeSource:
		return RuleTypeSource
	case SubTypeOrder:
		return RuleTypeOrder
	case SubTypeMinRest:
		return RuleTypeMinRestDays
	default:
		return ""
	}
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
