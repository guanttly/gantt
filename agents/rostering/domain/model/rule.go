package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的规则类型
type Rule = sdk_model.Rule
type CreateRuleRequest = sdk_model.CreateRuleRequest
type UpdateRuleRequest = sdk_model.UpdateRuleRequest
type ListRulesRequest = sdk_model.ListRulesRequest
type RuleAssociation = sdk_model.RuleAssociation
type RuleApplyScope = sdk_model.RuleApplyScope

// AssociationType 常量
const (
	AssociationTypeEmployee = sdk_model.AssociationTypeEmployee
	AssociationTypeShift    = sdk_model.AssociationTypeShift
	AssociationTypeGroup    = sdk_model.AssociationTypeGroup
)

// RelationRole 常量
const (
	RelationRoleTarget    = sdk_model.RelationRoleTarget
	RelationRoleSource    = sdk_model.RelationRoleSource
	RelationRoleReference = sdk_model.RelationRoleReference
	RelationRoleSubject   = sdk_model.RelationRoleSubject
	RelationRoleObject    = sdk_model.RelationRoleObject
)

// ScopeType 常量
const (
	ScopeTypeAll             = sdk_model.ScopeTypeAll
	ScopeTypeEmployee        = sdk_model.ScopeTypeEmployee
	ScopeTypeGroup           = sdk_model.ScopeTypeGroup
	ScopeTypeExcludeEmployee = sdk_model.ScopeTypeExcludeEmployee
	ScopeTypeExcludeGroup    = sdk_model.ScopeTypeExcludeGroup
)

// RuleSet 会话中的规则集合（业务模型）
type RuleSet struct {
	EmployeeRules []Rule `json:"employeeRules,omitempty"` // 人员相关规则
	GroupRules    []Rule `json:"groupRules,omitempty"`    // 分组相关规则
	ShiftRules    []Rule `json:"shiftRules,omitempty"`    // 班次相关规则
	GlobalRules   []Rule `json:"globalRules,omitempty"`   // 全局规则
}
