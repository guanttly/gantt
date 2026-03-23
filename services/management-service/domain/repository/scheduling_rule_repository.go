package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// ISchedulingRuleRepository 排班规则仓储接口
type ISchedulingRuleRepository interface {
	// Create 创建排班规则
	Create(ctx context.Context, rule *model.SchedulingRule) error

	// Update 更新排班规则
	Update(ctx context.Context, rule *model.SchedulingRule) error

	// Delete 删除排班规则（软删除）
	Delete(ctx context.Context, orgID, ruleID string) error

	// GetByID 根据ID获取规则
	GetByID(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error)

	// List 查询规则列表
	List(ctx context.Context, filter *model.SchedulingRuleFilter) (*model.SchedulingRuleListResult, error)

	// Exists 检查规则是否存在
	Exists(ctx context.Context, orgID, ruleID string) (bool, error)

	// GetByApplyScope 根据作用范围查询规则
	GetByApplyScope(ctx context.Context, orgID string, applyScope model.ApplyScope) ([]*model.SchedulingRule, error)

	// GetByRuleType 根据规则类型查询规则
	GetByRuleType(ctx context.Context, orgID string, ruleType model.RuleType) ([]*model.SchedulingRule, error)

	// GetActiveRules 获取所有启用的规则
	GetActiveRules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)

	// ==================== 关联管理 ====================

	// AddAssociations 批量添加规则关联
	AddAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error

	// RemoveAssociations 批量删除规则关联（通过关联ID）
	RemoveAssociations(ctx context.Context, orgID, ruleID string, associationIDs []string) error

	// RemoveAssociationByTarget 删除规则关联（通过目标类型和目标ID）
	RemoveAssociationByTarget(ctx context.Context, orgID, ruleID, targetType, targetID string) error

	// GetAssociations 获取规则的所有关联
	GetAssociations(ctx context.Context, orgID, ruleID string) ([]model.RuleAssociation, error)

	// GetRulesForEmployee 获取某个员工相关的所有规则
	GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.SchedulingRule, error)

	// GetRulesForShift 获取某个班次相关的所有规则
	GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error)

	// GetRulesForGroup 获取某个分组相关的所有规则
	GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.SchedulingRule, error)

	// GetRulesForEmployees 批量获取多个员工相关的所有规则
	GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.SchedulingRule, error)

	// GetRulesForShifts 批量获取多个班次相关的所有规则
	GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.SchedulingRule, error)

	// GetRulesForGroups 批量获取多个分组相关的所有规则
	GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.SchedulingRule, error)

	// ClearAssociations 清空规则的所有关联
	ClearAssociations(ctx context.Context, orgID, ruleID string) error
}

// IRuleApplyScopeRepository 规则适用范围仓储接口（V4.1新增）
type IRuleApplyScopeRepository interface {
	// Create 创建适用范围
	Create(ctx context.Context, scope *model.RuleApplyScope) error

	// BatchCreate 批量创建适用范围
	BatchCreate(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error

	// Delete 删除适用范围
	Delete(ctx context.Context, orgID, scopeID string) error

	// DeleteByRuleID 删除规则的所有适用范围
	DeleteByRuleID(ctx context.Context, orgID, ruleID string) error

	// GetByRuleID 获取规则的所有适用范围
	GetByRuleID(ctx context.Context, orgID, ruleID string) ([]model.RuleApplyScope, error)

	// GetByEmployeeID 获取适用于指定员工的规则ID列表
	GetByEmployeeID(ctx context.Context, orgID, employeeID string) ([]string, error)

	// GetByGroupID 获取适用于指定分组的规则ID列表
	GetByGroupID(ctx context.Context, orgID, groupID string) ([]string, error)

	// GetGlobalRuleIDs 获取全局规则ID列表
	GetGlobalRuleIDs(ctx context.Context, orgID string) ([]string, error)

	// ReplaceScopes 替换规则的所有适用范围（先删后增）
	ReplaceScopes(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error

	// IsRuleApplicableToEmployee 判断规则是否适用于指定员工
	IsRuleApplicableToEmployee(ctx context.Context, orgID, ruleID, employeeID string, employeeGroupIDs []string) (bool, error)
}
