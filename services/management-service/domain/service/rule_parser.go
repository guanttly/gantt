package service

import (
	"context"
	"jusha/gantt/service/management/domain/model"
	"time"
)

// IRuleParserService 规则解析服务接口
type IRuleParserService interface {
	// ParseRule 解析语义化规则
	ParseRule(ctx context.Context, req *ParseRuleRequest) (*ParseRuleResponse, error)

	// BatchParse 批量解析规则
	BatchParse(ctx context.Context, req *BatchParseRequest) (*BatchParseResponse, error)

	// SaveParsedRules 保存解析后的规则
	SaveParsedRules(ctx context.Context, orgID string, parsedRules []*ParsedRule, dependencies []*RuleDependency, conflicts []*RuleConflict) ([]*model.SchedulingRule, error)
}

// ParseRuleRequest 规则解析请求
type ParseRuleRequest struct {
	OrgID      string   `json:"orgId"`
	RuleText   string   `json:"ruleText"`   // 自然语言规则描述（设计字段）
	ShiftNames []string `json:"shiftNames"` // 当前组织的班次名称列表（设计字段）
	GroupNames []string `json:"groupNames"` // 当前组织的分组名称列表（设计字段）
	// 向后兼容字段（@deprecated V3: 迁移完成后可移除）
	Name            string     `json:"name,omitempty"`
	RuleDescription string     `json:"ruleDescription,omitempty"` // 兼容旧字段，优先使用 ruleText
	ApplyScope      string     `json:"applyScope,omitempty"`
	Priority        int        `json:"priority,omitempty"`
	ValidFrom       *time.Time `json:"validFrom,omitempty"`
	ValidTo         *time.Time `json:"validTo,omitempty"`
}

// BatchParseRequest 批量解析请求（一段文字含多条规则）
type BatchParseRequest struct {
	OrgID      string   `json:"orgId"`
	RuleTexts  []string `json:"ruleTexts"`  // 多段规则描述
	ShiftNames []string `json:"shiftNames"` // 当前组织的班次名称列表
	GroupNames []string `json:"groupNames"` // 当前组织的分组名称列表
}

// ParseRuleResponse 规则解析响应
type ParseRuleResponse struct {
	OriginalRule string            `json:"originalRule"` // 原始规则描述
	ParsedRules  []*ParsedRule     `json:"parsedRules"`  // 解析后的规则列表
	Dependencies []*RuleDependency `json:"dependencies"` // 识别出的依赖关系
	Conflicts    []*RuleConflict   `json:"conflicts"`    // 识别出的冲突关系
	Reasoning    string            `json:"reasoning"`    // 解析说明
	// V4 设计字段
	BackTranslation string `json:"backTranslation,omitempty"` // 回译文本（LLM 将结构化结果翻译回自然语言）
}

// BatchParseResponse 批量解析响应
type BatchParseResponse struct {
	Results []*ParseRuleResponse `json:"results"`          // 每个规则文本的解析结果
	Errors  []*ParseError        `json:"errors,omitempty"` // 解析失败的规则
}

// ParseError 解析错误
type ParseError struct {
	RuleText string `json:"ruleText"` // 失败的规则文本
	Error    string `json:"error"`    // 错误信息
}

// ParsedRule 解析后的规则
type ParsedRule struct {
	Name        string           `json:"name"`
	Category    string           `json:"category"`    // constraint/preference/dependency
	SubCategory string           `json:"subCategory"` // forbid/must/limit/prefer/suggest/source/resource/order
	RuleType    model.RuleType   `json:"ruleType"`
	ApplyScope  model.ApplyScope `json:"applyScope"`
	TimeScope   model.TimeScope  `json:"timeScope"`
	Description string           `json:"description"`
	RuleData    string           `json:"ruleData"`

	// 数值型参数
	MaxCount       *int `json:"maxCount,omitempty"`
	ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
	IntervalDays   *int `json:"intervalDays,omitempty"`
	MinRestDays    *int `json:"minRestDays,omitempty"`
	TimeOffsetDays *int `json:"timeOffsetDays,omitempty"` // 跨日依赖天数偏移

	Priority  int        `json:"priority"`
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	ValidTo   *time.Time `json:"validTo,omitempty"`

	// 关联信息（@deprecated V3: 保留向后兼容）
	Associations []model.RuleAssociation `json:"associations,omitempty"`

	// V4.1新增：结构化的班次关系
	SubjectShifts []string `json:"subjectShifts,omitempty"` // 主体班次名称列表
	ObjectShifts  []string `json:"objectShifts,omitempty"`  // 客体班次名称列表
	TargetShifts  []string `json:"targetShifts,omitempty"`  // 目标班次名称列表（单目标规则）

	// V4.1新增：适用范围
	ScopeType      string   `json:"scopeType,omitempty"`      // all/employee/group/exclude_employee/exclude_group
	ScopeEmployees []string `json:"scopeEmployees,omitempty"` // 员工名称列表
	ScopeGroups    []string `json:"scopeGroups,omitempty"`    // 分组名称列表

	// 依赖和冲突关系（解析时为空，保存后填充）
	Dependencies []string `json:"dependencies,omitempty"` // 依赖的其他规则ID
	Conflicts    []string `json:"conflicts,omitempty"`    // 冲突的其他规则ID
}

// RuleDependency 规则依赖关系
type RuleDependency struct {
	DependentRuleName   string `json:"dependentRuleName"`   // 被依赖的规则（需要先执行）
	DependentOnRuleName string `json:"dependentOnRuleName"` // 依赖的规则（后执行）
	DependencyType      string `json:"dependencyType"`      // time/source/resource/order
	Description         string `json:"description"`
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
	RuleName1    string `json:"ruleName1"`
	RuleName2    string `json:"ruleName2"`
	ConflictType string `json:"conflictType"` // exclusive/resource/time/frequency
	Description  string `json:"description"`
}
