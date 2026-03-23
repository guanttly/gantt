package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// IRuleService 规则管理接口
type IRuleService interface {
	// CreateRule 创建规则
	CreateRule(ctx context.Context, req model.CreateRuleRequest) (string, error)

	// UpdateRule 更新规则
	UpdateRule(ctx context.Context, req model.UpdateRuleRequest) error

	// ListRules 获取规则列表
	ListRules(ctx context.Context, req model.ListRulesRequest) (*model.Page[*model.Rule], error)

	// GetRule 获取规则详情
	GetRule(ctx context.Context, orgID, ruleID string) (*model.Rule, error)

	// DeleteRule 删除规则
	DeleteRule(ctx context.Context, orgID, ruleID string) error

	// AddRuleAssociations 添加规则关联
	AddRuleAssociations(ctx context.Context, req model.AddRuleAssociationsRequest) error

	// GetRulesForEmployee 获取员工的所有生效规则
	GetRulesForEmployee(ctx context.Context, orgID, employeeID, date string) ([]*model.Rule, error)

	// GetRulesForGroup 获取分组的所有生效规则
	GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error)

	// GetRulesForShift 获取班次的所有生效规则
	GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)
	// 批量查询规则
	GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error)
	GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error)
	GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error)
}
