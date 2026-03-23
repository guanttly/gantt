package model

import "time"

// Rule 规则领域模型（对应management-service的SchedulingRule）
type Rule struct {
	ID             string `json:"id"`
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RuleType       string `json:"ruleType"`
	ApplyScope     string `json:"applyScope,omitempty"`
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

	// V4新增字段
	Category        string   `json:"category,omitempty"`        // constraint, preference, dependency
	SubCategory     string   `json:"subCategory,omitempty"`     // forbid, must, limit, prefer, suggest, source, resource, order
	OriginalRuleID  string   `json:"originalRuleId,omitempty"`  // 原始规则ID（用于迁移）
	SourceType      string   `json:"sourceType,omitempty"`      // manual, llm_parsed, migrated
	ParseConfidence *float64 `json:"parseConfidence,omitempty"` // LLM 解析置信度 (0.0-1.0)
	Version         string   `json:"version,omitempty"`         // v3, v4

	// 关联信息（从关联表加载）
	Associations []RuleAssociation `json:"associations,omitempty"`

	// V4.1新增字段
	ApplyScopes []ApplyScope `json:"applyScopes,omitempty"` // 适用范围
}

// RuleAssociation 规则关联
type RuleAssociation struct {
	ID              string `json:"id,omitempty"`
	RuleID          string `json:"ruleId"`
	AssociationType string `json:"associationType"` // employee, shift, group
	AssociationID   string `json:"associationId"`
	Role            string `json:"role,omitempty"` // V4新增：关联角色 subject/object/target
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RuleType       string `json:"ruleType"`
	ApplyScope     string `json:"applyScope,omitempty"`
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
	ApplyScope     string `json:"applyScope,omitempty"`
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
	OrgID      string
	RuleType   string
	ApplyScope string
	TimeScope  string
	IsActive   *bool
	Keyword    string
	Status     string
	Page       int
	PageSize   int
	// V4新增筛选字段
	Category    string
	SubCategory string
	SourceType  string
	Version     string
}

// ListRulesResponse 规则列表响应
type ListRulesResponse struct {
	Rules      []*Rule `json:"rules"`
	TotalCount int     `json:"totalCount"`
}

// ApplyScope 适用范围 (V4.1)
type ApplyScope struct {
	ScopeType string `json:"scopeType"` // all, employee, group, exclude_employee, exclude_group
	ScopeID   string `json:"scopeId,omitempty"`
	ScopeName string `json:"scopeName,omitempty"`
}

// CreateRuleWithRelationsRequest 创建带关系的规则请求 (V4.1)
type CreateRuleWithRelationsRequest struct {
	CreateRuleRequest
	Associations []RuleAssociation `json:"associations,omitempty"`
	ApplyScopes  []ApplyScope      `json:"applyScopes,omitempty"`
}

// UpdateRuleWithRelationsRequest 更新带关系的规则请求 (V4.1)
type UpdateRuleWithRelationsRequest struct {
	UpdateRuleRequest
	Associations []RuleAssociation `json:"associations,omitempty"`
	ApplyScopes  []ApplyScope      `json:"applyScopes,omitempty"`
}
