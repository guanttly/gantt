package model

import (
	"time"
)

// SchedulingRule 排班规则领域模型
type SchedulingRule struct {
	ID             string     `json:"id"`
	OrgID          string     `json:"orgId"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	RuleType       RuleType   `json:"ruleType"`
	ApplyScope     ApplyScope `json:"applyScope"`
	TimeScope      TimeScope  `json:"timeScope"`
	TimeOffsetDays *int       `json:"timeOffsetDays,omitempty"` // V4.1新增：时间偏移天数，正数表示向未来偏移，负数表示向历史偏移
	RuleData       string     `json:"ruleData"`                 // 规则说明文本，例如"隔周上夜班"、"不能连续超过2天"等

	// 数值型规则参数（根据规则类型使用）
	MaxCount       *int `json:"maxCount,omitempty"`       // 最大次数：用于 maxCount 类型，如"每周最多3次"
	ConsecutiveMax *int `json:"consecutiveMax,omitempty"` // 连续最大天数：用于 maxCount 类型，如"连续不超过2天"
	IntervalDays   *int `json:"intervalDays,omitempty"`   // 间隔天数：用于 periodic 类型，如"间隔7天"、"隔周=14天"
	MinRestDays    *int `json:"minRestDays,omitempty"`    // 最少休息天数：用于休息规则，如"夜班后至少休息1天"

	Priority  int        `json:"priority"`
	IsActive  bool       `json:"isActive"`
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// 关联信息（从关联表加载）
	Associations []RuleAssociation `json:"associations,omitempty"`

	// V4新增字段：规则分类
	Category       string `json:"category,omitempty"`       // 规则分类: constraint/preference/dependency
	SubCategory    string `json:"subCategory,omitempty"`    // 规则子分类: forbid/must/limit/prefer/suggest/source/resource/order
	OriginalRuleID string `json:"originalRuleId,omitempty"` // 原始规则ID（如果是从语义化规则解析出来的）

	// SourceType 规则来源类型
	// manual: 手动创建 / llm_parsed: LLM 解析 / migrated: V3 迁移
	// @deprecated V3: 迁移完成后 migrated 统一为 manual
	SourceType string `json:"sourceType,omitempty"`

	// ParseConfidence LLM 解析置信度 (0.0-1.0)
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`

	// Version 规则版本号（V3=空或"v3", V4="v4"）
	// @deprecated V3: 全量迁移完成后此字段固定为"v4"，可移除版本判断逻辑
	Version string `json:"version,omitempty"`

	// V4.1新增：适用范围
	ApplyScopes []RuleApplyScope `json:"applyScopes,omitempty"` // 适用范围列表
}

// RelationRole 关系角色常量（存储在 RuleAssociation.Role 中）
const (
	RelationRoleSubject = "subject" // 主体：触发规则的班次（如：排了A就不能排B中的A）
	RelationRoleObject  = "object"  // 客体：被约束的班次（如：排了A就不能排B中的B）
	RelationRoleTarget  = "target"  // 目标：单一班次规则的目标班次（如：夜班最多3次中的夜班）
)

// RuleApplyScope 规则适用范围（V4.1新增）
type RuleApplyScope struct {
	ID        string    `json:"id,omitempty"`
	RuleID    string    `json:"ruleId,omitempty"`
	ScopeType string    `json:"scopeType"`           // all/employee/group/exclude_employee/exclude_group
	ScopeID   string    `json:"scopeId,omitempty"`   // 范围对象ID
	ScopeName string    `json:"scopeName,omitempty"` // 范围对象名称
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// ScopeType 范围类型常量
const (
	ScopeTypeAll             = "all"              // 全局：对所有员工生效
	ScopeTypeEmployee        = "employee"         // 员工：仅对指定员工生效
	ScopeTypeGroup           = "group"            // 分组：仅对指定分组的员工生效
	ScopeTypeExcludeEmployee = "exclude_employee" // 排除员工：对除指定员工外的所有人生效
	ScopeTypeExcludeGroup    = "exclude_group"    // 排除分组：对除指定分组外的所有人生效
)

// RuleAssociation 规则关联信息
type RuleAssociation struct {
	ID              string          `json:"id"`
	RuleID          string          `json:"ruleId"`
	AssociationType AssociationType `json:"associationType"`
	AssociationID   string          `json:"associationId"`
	Role            string          `json:"role,omitempty"` // V4新增：关联角色 target/source/reference
	CreatedAt       time.Time       `json:"createdAt"`
}

// RuleType 规则类型
type RuleType string

const (
	// ===== 班次约束规则类型（通常 ApplyScope = global）=====
	RuleTypeExclusive        RuleType = "exclusive"         // 排他：排了A就不能排B
	RuleTypeCombinable       RuleType = "combinable"        // 可合并：A和B可以同时排同一人
	RuleTypeRequiredTogether RuleType = "required_together" // 必须同时：排了A必须排B

	// ===== 员工约束规则类型（通常 ApplyScope = specific）=====
	RuleTypePeriodic     RuleType = "periodic"      // 周期性：隔N周/隔N天
	RuleTypeMaxCount     RuleType = "maxCount"      // 最大次数：每周/每月最多N次
	RuleTypeForbiddenDay RuleType = "forbidden_day" // 禁止日期：某些日期不排班
	RuleTypePreferred    RuleType = "preferred"     // 偏好：优先/避免某些班次或日期
)

// ApplyScope 作用范围
type ApplyScope string

const (
	ApplyScopeGlobal   ApplyScope = "global"   // 全局规则（不需要关联表）
	ApplyScopeSpecific ApplyScope = "specific" // 针对特定对象（需要通过关联表关联）
)

// TimeScope 时间范围
type TimeScope string

const (
	TimeScopeSameDay   TimeScope = "same_day"   // 同一天
	TimeScopeSameWeek  TimeScope = "same_week"  // 同一周
	TimeScopeSameMonth TimeScope = "same_month" // 同一月
	TimeScopeCustom    TimeScope = "custom"     // 自定义（根据规则数据判断）
)

// AssociationType 关联类型
type AssociationType string

const (
	AssociationTypeEmployee AssociationType = "employee" // 员工
	AssociationTypeShift    AssociationType = "shift"    // 班次
	AssociationTypeGroup    AssociationType = "group"    // 分组
)

// IsValid 判断规则在指定日期是否有效
func (r *SchedulingRule) IsValid(date time.Time) bool {
	if !r.IsActive {
		return false
	}

	if r.ValidFrom != nil && date.Before(*r.ValidFrom) {
		return false
	}

	if r.ValidTo != nil && date.After(*r.ValidTo) {
		return false
	}

	return true
}

// IsGlobal 判断是否为全局规则
func (r *SchedulingRule) IsGlobal() bool {
	return r.ApplyScope == ApplyScopeGlobal
}

// HasAssociations 判断规则是否有关联对象
func (r *SchedulingRule) HasAssociations() bool {
	return len(r.Associations) > 0
}

// GetAssociatedEmployeeIDs 获取关联的员工ID列表
func (r *SchedulingRule) GetAssociatedEmployeeIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeEmployee {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// GetAssociatedShiftIDs 获取关联的班次ID列表
func (r *SchedulingRule) GetAssociatedShiftIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeShift {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// ============================================================================
// V4.1 辅助方法（基于 Associations + Role）
// ============================================================================

// GetSubjectShiftIDs 获取主体班次ID列表（Role=subject 的班次关联）
func (r *SchedulingRule) GetSubjectShiftIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeShift && assoc.Role == RelationRoleSubject {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// GetObjectShiftIDs 获取客体班次ID列表（Role=object 的班次关联）
func (r *SchedulingRule) GetObjectShiftIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeShift && assoc.Role == RelationRoleObject {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// GetTargetShiftIDs 获取目标班次ID列表（Role=target 的班次关联）
func (r *SchedulingRule) GetTargetShiftIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeShift && assoc.Role == RelationRoleTarget {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// GetAllRelatedShiftIDs 获取所有相关班次ID列表
func (r *SchedulingRule) GetAllRelatedShiftIDs() []string {
	var ids []string
	for _, assoc := range r.Associations {
		if assoc.AssociationType == AssociationTypeShift {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// IsApplicableToEmployee 判断规则是否适用于指定员工（V4.1）
func (r *SchedulingRule) IsApplicableToEmployee(employeeID string, employeeGroupIDs []string) bool {
	if len(r.ApplyScopes) == 0 {
		return true // 没有范围限定，默认全局生效
	}

	hasInclude := false
	isExcluded := false

	for _, scope := range r.ApplyScopes {
		switch scope.ScopeType {
		case ScopeTypeAll:
			hasInclude = true
		case ScopeTypeEmployee:
			if scope.ScopeID == employeeID {
				hasInclude = true
			}
		case ScopeTypeGroup:
			for _, gid := range employeeGroupIDs {
				if scope.ScopeID == gid {
					hasInclude = true
					break
				}
			}
		case ScopeTypeExcludeEmployee:
			if scope.ScopeID == employeeID {
				isExcluded = true
			}
		case ScopeTypeExcludeGroup:
			for _, gid := range employeeGroupIDs {
				if scope.ScopeID == gid {
					isExcluded = true
					break
				}
			}
		}
	}

	return hasInclude && !isExcluded
}

// IsBinaryRelation 判断是否为二元关系规则（V4.1）
func (r *SchedulingRule) IsBinaryRelation() bool {
	return r.RuleType == RuleTypeExclusive ||
		r.RuleType == RuleTypeCombinable ||
		r.RuleType == RuleTypeRequiredTogether
}

// RequiresShiftRelation 判断该规则类型是否需要班次关系（V4.1）
func (r *SchedulingRule) RequiresShiftRelation() bool {
	switch r.RuleType {
	case RuleTypeExclusive, RuleTypeCombinable, RuleTypeRequiredTogether:
		return true // 需要主体和客体
	case RuleTypePeriodic, RuleTypeMaxCount:
		return true // 需要目标班次
	default:
		return false
	}
}

// ValidateShiftAssociations 验证班次关联完整性（基于 Associations + Role）
func (r *SchedulingRule) ValidateShiftAssociations() error {
	// 先统一校验 Role 枚举值合法性
	for _, assoc := range r.Associations {
		if assoc.Role != "" &&
			assoc.Role != RelationRoleSubject &&
			assoc.Role != RelationRoleObject &&
			assoc.Role != RelationRoleTarget &&
			assoc.Role != AssociationRoleSource &&
			assoc.Role != AssociationRoleReference {
			return &ValidationError{
				Field:   "associations.role",
				Message: "非法的关联角色: " + assoc.Role,
			}
		}
	}

	switch r.RuleType {
	case RuleTypeExclusive, RuleTypeCombinable, RuleTypeRequiredTogether:
		// 二元关系规则：需要至少1个主体和1个客体
		hasSubject := false
		hasObject := false
		for _, assoc := range r.Associations {
			if assoc.AssociationType == AssociationTypeShift {
				if assoc.Role == RelationRoleSubject {
					hasSubject = true
				}
				if assoc.Role == RelationRoleObject {
					hasObject = true
				}
			}
		}
		if !hasSubject || !hasObject {
			return &ValidationError{
				Field:   "associations",
				Message: "排他/组合/必须同时规则需要至少一个主体班次和一个客体班次",
			}
		}
	case RuleTypePeriodic, RuleTypeMaxCount:
		// 单班次规则：需要至少1个目标班次
		hasTarget := false
		for _, assoc := range r.Associations {
			if assoc.AssociationType == AssociationTypeShift && assoc.Role == RelationRoleTarget {
				hasTarget = true
				break
			}
		}
		if !hasTarget {
			return &ValidationError{
				Field:   "associations",
				Message: "周期性/最大次数规则需要至少一个目标班次",
			}
		}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// SchedulingRuleFilter 排班规则查询过滤器
type SchedulingRuleFilter struct {
	OrgID      string
	RuleType   *RuleType   // 规则类型
	ApplyScope *ApplyScope // 作用范围
	TimeScope  *TimeScope  // 时间范围
	IsActive   *bool       // 是否启用
	Keyword    string      // 按名称、描述模糊搜索
	// V4新增筛选字段
	TimeOffsetDays *int    // 时间偏移天数
	Category       *string // 规则分类: constraint/preference/dependency
	SubCategory    *string // 规则子分类
	SourceType     *string // 规则来源类型: manual/llm_parsed/migrated
	Version        *string // 规则版本: v3/v4
	Page           int
	PageSize       int
}

// SchedulingRuleListResult 排班规则列表结果
type SchedulingRuleListResult struct {
	Items    []*SchedulingRule `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// RuleCategory 规则分类常量
const (
	CategoryConstraint = "constraint" // 约束型规则
	CategoryPreference = "preference" // 偏好型规则
	CategoryDependency = "dependency" // 依赖型规则
)

// RuleSubCategory 规则子分类常量
const (
	SubCategoryForbid   = "forbid"   // 禁止型
	SubCategoryMust     = "must"     // 必须型
	SubCategoryLimit    = "limit"    // 限制型
	SubCategoryPrefer   = "prefer"   // 优先型
	SubCategorySuggest  = "suggest"  // 建议型
	SubCategorySource   = "source"   // 来源依赖
	SubCategoryResource = "resource" // 资源预留
	SubCategoryOrder    = "order"    // 顺序依赖
)

// AssociationRole 关联角色常量
const (
	AssociationRoleTarget    = "target"    // 被约束的对象（规则作用目标）
	AssociationRoleSource    = "source"    // 数据来源（依赖型规则的前置对象）
	AssociationRoleReference = "reference" // 引用对象（规则中提到但不直接约束的对象）
)
