package service

import "context"

// IRuleOrganizerService 规则组织服务接口
type IRuleOrganizerService interface {
	// OrganizeRules 组织规则
	OrganizeRules(ctx context.Context, orgID string) (*RuleOrganizationResult, error)
}

// RuleOrganizationResult 规则组织结果
type RuleOrganizationResult struct {
	ConstraintRules     []*ClassifiedRuleInfo  `json:"constraintRules"`
	PreferenceRules     []*ClassifiedRuleInfo  `json:"preferenceRules"`
	DependencyRules     []*ClassifiedRuleInfo  `json:"dependencyRules"`
	ShiftDependencies   []*ShiftDependencyInfo `json:"shiftDependencies"`
	RuleDependencies    []*RuleDependencyInfo  `json:"ruleDependencies"`
	RuleConflicts       []*RuleConflictInfo    `json:"ruleConflicts"`
	ShiftExecutionOrder []string               `json:"shiftExecutionOrder"`
	RuleExecutionOrder  []string               `json:"ruleExecutionOrder"`
	Warnings            []*OrganizationWarning `json:"warnings,omitempty"` // 警告信息
}

// OrganizationWarning 组织过程中的警告信息
type OrganizationWarning struct {
	Type       string `json:"type"`    // 警告类型: priority_dependency_conflict
	Message    string `json:"message"` // 警告消息
	ShiftID1   string `json:"shiftId1,omitempty"`
	ShiftName1 string `json:"shiftName1,omitempty"`
	ShiftID2   string `json:"shiftId2,omitempty"`
	ShiftName2 string `json:"shiftName2,omitempty"`
	Priority1  int    `json:"priority1,omitempty"`
	Priority2  int    `json:"priority2,omitempty"`
	Resolution string `json:"resolution"` // 解决方式: use_dependency / use_priority
}

// ClassifiedRuleInfo 分类后的规则信息
type ClassifiedRuleInfo struct {
	RuleID       string   `json:"ruleId"`
	RuleName     string   `json:"ruleName"`
	Category     string   `json:"category"`
	SubCategory  string   `json:"subCategory"`
	RuleType     string   `json:"ruleType"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
	Priority     int      `json:"priority"`
	Description  string   `json:"description"`
}

// ShiftDependencyInfo 班次依赖关系信息
type ShiftDependencyInfo struct {
	DependentShiftID   string `json:"dependentShiftId"`
	DependentOnShiftID string `json:"dependentOnShiftId"`
	DependencyType     string `json:"dependencyType"`
	RuleID             string `json:"ruleId"`
	Description        string `json:"description"`
}

// RuleDependencyInfo 规则依赖关系信息
type RuleDependencyInfo struct {
	DependentRuleID   string `json:"dependentRuleId"`
	DependentOnRuleID string `json:"dependentOnRuleId"`
	DependencyType    string `json:"dependencyType"`
	Description       string `json:"description"`
}

// RuleConflictInfo 规则冲突关系信息
type RuleConflictInfo struct {
	RuleID1            string `json:"ruleId1"`
	RuleID2            string `json:"ruleId2"`
	ConflictType       string `json:"conflictType"`
	Description        string `json:"description"`
	ResolutionPriority int    `json:"resolutionPriority"`
}
