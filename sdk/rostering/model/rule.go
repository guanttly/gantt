package model

import "time"

// Rule 规则领域模型（对应management-service的SchedulingRule）
type Rule struct {
	ID             string `json:"id"`
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RuleType       string `json:"ruleType"`
	ApplyScope     string `json:"applyScope,omitempty"` // 人员作用范围(global=所有人生效, specific=特定人员生效)，注意：不代表班次全局！
	TimeScope      string `json:"timeScope,omitempty"`
	TimeOffsetDays *int   `json:"timeOffsetDays,omitempty"`
	RuleData       string `json:"ruleData,omitempty"`

	// 数值型规则参数
	MaxCount       *int `json:"maxCount,omitempty"`
	ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
	IntervalDays   *int `json:"intervalDays,omitempty"`
	MinRestDays    *int `json:"minRestDays,omitempty"`

	Priority  int        `json:"priority"`
	IsActive  bool       `json:"isActive,omitempty"`
	Status    string     `json:"status,omitempty"` // 向后兼容
	Config    string     `json:"config,omitempty"` // 向后兼容
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// 关联信息
	Associations []RuleAssociation `json:"associations,omitempty"`

	// V4.1新增字段：适用范围
	ApplyScopes []RuleApplyScope `json:"applyScopes,omitempty"`

	// V4新增字段：规则分类
	Category       string `json:"category,omitempty"`       // 规则分类: constraint/preference/dependency
	SubCategory    string `json:"subCategory,omitempty"`    // 规则子分类: forbid/must/limit/prefer/suggest/source/resource/order
	OriginalRuleID string `json:"originalRuleId,omitempty"` // 原始规则ID（如果是从语义化规则解析出来的）

	// SourceType 规则来源类型: manual/llm_parsed/migrated
	SourceType string `json:"sourceType,omitempty"`

	// ParseConfidence LLM 解析置信度 (0.0-1.0)
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`

	// Version 规则版本号（V3=空或"v3", V4="v4"）
	Version string `json:"version,omitempty"`
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RuleType       string `json:"ruleType"`
	ApplyScope     string `json:"applyScope,omitempty"` // 人员作用范围(global=所有人生效, specific=特定人员生效)，注意：不代表班次全局！
	TimeScope      string `json:"timeScope,omitempty"`
	TimeOffsetDays *int   `json:"timeOffsetDays,omitempty"`
	RuleData       string `json:"ruleData,omitempty"`

	// 数值型规则参数
	MaxCount       *int `json:"maxCount,omitempty"`
	ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
	IntervalDays   *int `json:"intervalDays,omitempty"`
	MinRestDays    *int `json:"minRestDays,omitempty"`

	Priority  int        `json:"priority"`
	IsActive  bool       `json:"isActive,omitempty"`
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	ValidTo   *time.Time `json:"validTo,omitempty"`

	// V4新增字段
	Category        string   `json:"category,omitempty"`
	SubCategory     string   `json:"subCategory,omitempty"`
	OriginalRuleID  string   `json:"originalRuleId,omitempty"`
	SourceType      string   `json:"sourceType,omitempty"`
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`
	Version         string   `json:"version,omitempty"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RuleType       string `json:"ruleType"`
	ApplyScope     string `json:"applyScope,omitempty"` // 人员作用范围(global=所有人生效, specific=特定人员生效)，注意：不代表班次全局！
	TimeScope      string `json:"timeScope,omitempty"`
	TimeOffsetDays *int   `json:"timeOffsetDays,omitempty"`
	RuleData       string `json:"ruleData,omitempty"`

	// 数值型规则参数
	MaxCount       *int `json:"maxCount,omitempty"`
	ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
	IntervalDays   *int `json:"intervalDays,omitempty"`
	MinRestDays    *int `json:"minRestDays,omitempty"`

	Priority  int        `json:"priority"`
	IsActive  bool       `json:"isActive,omitempty"`
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	ValidTo   *time.Time `json:"validTo,omitempty"`

	// V4新增字段
	Category        string   `json:"category,omitempty"`
	SubCategory     string   `json:"subCategory,omitempty"`
	OriginalRuleID  string   `json:"originalRuleId,omitempty"`
	SourceType      string   `json:"sourceType,omitempty"`
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`
	Version         string   `json:"version,omitempty"`
}

// ListRulesRequest 查询规则列表请求
type ListRulesRequest struct {
	OrgID      string `json:"orgId"`
	RuleType   string `json:"type"`
	ApplyScope string `json:"applyScope"`
	TimeScope  string `json:"timeScope"`
	IsActive   *bool  `json:"isActive"`
	Keyword    string `json:"keyword"`
	Status     string `json:"status"`
	// V4新增筛选字段
	Category    string `json:"category,omitempty"`    // 规则分类: constraint/preference/dependency
	SubCategory string `json:"subCategory,omitempty"` // 规则子分类
	SourceType  string `json:"sourceType,omitempty"`  // 规则来源类型: manual/llm_parsed/migrated
	Version     string `json:"version,omitempty"`     // 规则版本: v3/v4
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
}

// RuleAssociation 规则关联
//
// 字段说明（V3 vs V4 使用方式不同，注意区分）：
//
// [V3/兼容字段] AssociationType + AssociationID：
//   - 是一对使用 (类型 + ID) 描述关联对象的字段。
//   - AssociationType 可选值: "employee"（员工）、"shift"（班次）、"group"（分组）
//   - AssociationID 是对应类型的实体 ID。
//   - 说明：在 V3 中，规则对班次的限制直接用 AssociationType="shift" + AssociationID=班次ID 表达。
//   - 注意！此字段组**不区分"约束目标"和"来源参考"**，需要配合 Role 字段才能判断语义（见下）。
//
// [V4 新字段] Role：
//   - 对关联对象的语义角色进行细分：
//   - "target"  : 约束目标，该规则对这个实体施加约束（如"禁止排这个班次"）。
//   - "source"  : 数据来源，仅作参考（如"必须从这个班次的已排人员中选"），不对其施加约束。
//   - "reference": 参照对象，仅用于辅助判断，不直接约束。
//   - 重要：AssociationType="shift" 且 Role 为空/非 "target"，不代表该规则对这个班次施加约束！
//
// V4 中班次关联统一通过 Associations（AssociationType="shift"）+ Role（subject/object/target）表达。
//   - V4 规则只使用 Associations，V3 规则也只使用 Associations。

// AssociationType 常量定义
const (
	AssociationTypeEmployee = "employee"
	AssociationTypeShift    = "shift"
	AssociationTypeGroup    = "group"
)

// RelationRole 常量定义
const (
	RelationRoleTarget    = "target"
	RelationRoleSource    = "source"
	RelationRoleReference = "reference"
	RelationRoleSubject   = "subject"
	RelationRoleObject    = "object"
)

type RuleAssociation struct {
	ID              string `json:"id,omitempty"`
	RuleID          string `json:"ruleId"`
	AssociationType string `json:"associationType"` // V3兼容：关联类型，可选 employee/shift/group；V4中需配合 Role 字段判断约束语义
	AssociationID   string `json:"associationId"`   // V3兼容：关联对象的 ID（员工ID/班次ID/分组ID）

	Role string `json:"role,omitempty"` // V4新增：关联角色，"target"=约束目标/"source"=数据来源/"reference"=参照，为空时表示无角色（不可默认视为约束目标！）
}

// RuleApplyScope 规则适用范围（V4.1新增）
type RuleApplyScope struct {
	ScopeType string `json:"scopeType"`           // all/employee/group/exclude_employee/exclude_group
	ScopeID   string `json:"scopeId,omitempty"`   // 范围对象ID
	ScopeName string `json:"scopeName,omitempty"` // 范围对象名称
}

// ScopeType 常量定义
const (
	ScopeTypeAll             = "all"
	ScopeTypeEmployee        = "employee"
	ScopeTypeGroup           = "group"
	ScopeTypeExcludeEmployee = "exclude_employee"
	ScopeTypeExcludeGroup    = "exclude_group"
)

// AddRuleAssociationsRequest 添加规则关联请求
type AddRuleAssociationsRequest struct {
	RuleID       string             `json:"ruleId"`
	Associations []*RuleAssociation `json:"associations"`
}

// GetRuleAssociationsRequest 获取规则关联请求
type GetRuleAssociationsRequest struct {
	RuleID string `json:"ruleId"`
	OrgID  string `json:"orgId"`
}
