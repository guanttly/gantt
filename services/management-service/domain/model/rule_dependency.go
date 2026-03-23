package model

import "time"

// RuleDependency 规则依赖关系
type RuleDependency struct {
	ID                string    `json:"id"`
	OrgID             string    `json:"orgId"`
	DependentRuleID   string    `json:"dependentRuleId"`   // 被依赖的规则ID（需要先执行）
	DependentOnRuleID string    `json:"dependentOnRuleId"` // 依赖的规则ID（后执行）
	DependencyType    string    `json:"dependencyType"`     // 依赖类型: time/source/resource/order
	Description       string    `json:"description"`        // 依赖关系描述
	CreatedAt         time.Time `json:"createdAt"`
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
	ID                 string    `json:"id"`
	OrgID              string    `json:"orgId"`
	RuleID1            string    `json:"ruleId1"`            // 冲突的规则1
	RuleID2            string    `json:"ruleId2"`            // 冲突的规则2
	ConflictType       string    `json:"conflictType"`       // 冲突类型: exclusive/resource/time/frequency
	Description        string    `json:"description"`       // 冲突描述
	ResolutionPriority int       `json:"resolutionPriority"` // 解决优先级（数字越小越优先）
	CreatedAt          time.Time `json:"createdAt"`
}

// ShiftDependency 班次依赖关系
type ShiftDependency struct {
	ID                 string    `json:"id"`
	OrgID              string    `json:"orgId"`
	DependentShiftID   string    `json:"dependentShiftId"`   // 被依赖的班次ID（需要先排）
	DependentOnShiftID string    `json:"dependentOnShiftId"` // 依赖的班次ID（后排）
	DependencyType     string    `json:"dependencyType"`     // 依赖类型: time/source/resource
	RuleID             string    `json:"ruleId"`             // 产生此依赖关系的规则ID
	Description        string    `json:"description"`        // 依赖关系描述
	CreatedAt          time.Time `json:"createdAt"`
}
