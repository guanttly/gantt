package service

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// ISchedulingRuleService 排班规则服务接口
type ISchedulingRuleService interface {
	// CreateRule 创建排班规则
	CreateRule(ctx context.Context, rule *model.SchedulingRule) error

	// UpdateRule 更新排班规则
	UpdateRule(ctx context.Context, rule *model.SchedulingRule) error

	// DeleteRule 删除排班规则
	DeleteRule(ctx context.Context, orgID, ruleID string) error

	// GetRule 获取规则详情
	GetRule(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error)

	// ListRules 查询规则列表
	ListRules(ctx context.Context, filter *model.SchedulingRuleFilter) (*model.SchedulingRuleListResult, error)

	// GetActiveRules 获取所有启用的规则
	GetActiveRules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)

	// ToggleRuleStatus 切换规则启用状态
	ToggleRuleStatus(ctx context.Context, orgID, ruleID string, isActive bool) error

	// GetRuleAssociations 获取规则的所有关联 (V4.1: 保留用于内部查询)
	GetRuleAssociations(ctx context.Context, orgID, ruleID string) ([]model.RuleAssociation, error)

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

	// UpdateRuleAssociations 更新规则关联（先清空再添加）
	UpdateRuleAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error

	// ==================== 业务验证 ====================

	// ValidateRule 验证规则的有效性
	ValidateRule(ctx context.Context, rule *model.SchedulingRule) error

	// CheckRuleConflicts 检查规则冲突
	CheckRuleConflicts(ctx context.Context, orgID string, rule *model.SchedulingRule) ([]string, error)

	// ==================== V4 新增方法 ====================

	// ListRulesByCategory 按分类获取规则
	ListRulesByCategory(ctx context.Context, orgID, category string) ([]*model.SchedulingRule, error)

	// GetV3Rules 获取所有 V3 规则（待迁移）
	GetV3Rules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)

	// BatchUpdateVersion 批量更新规则版本
	BatchUpdateVersion(ctx context.Context, orgID string, ruleIDs []string, version string) error

	// ==================== V4.1 新增方法：班次关系管理 ====================

	// GetRulesBySubjectShift 获取以指定班次为主体的规则
	GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error)

	// GetRulesByObjectShift 获取以指定班次为客体的规则
	GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error)

	// ==================== V4.1 新增方法：适用范围管理 ====================

	// SetApplyScopes 设置规则的适用范围（替换）
	SetApplyScopes(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error

	// GetApplyScopes 获取规则的适用范围
	GetApplyScopes(ctx context.Context, orgID, ruleID string) ([]model.RuleApplyScope, error)

	// GetRulesForEmployeeWithScope 获取适用于指定员工的所有规则（考虑范围）
	GetRulesForEmployeeWithScope(ctx context.Context, orgID, employeeID string, employeeGroupIDs []string) ([]*model.SchedulingRule, error)

	// IsRuleApplicableToEmployee 判断规则是否适用于指定员工
	IsRuleApplicableToEmployee(ctx context.Context, orgID, ruleID, employeeID string, employeeGroupIDs []string) (bool, error)

	// ==================== V4.1 新增方法：完整规则加载 ====================

	// GetRuleWithRelations 获取规则详情（包含班次关系和适用范围）
	GetRuleWithRelations(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error)

	// GetActiveRulesWithRelations 获取所有启用的规则（包含班次关系和适用范围）
	GetActiveRulesWithRelations(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)
}
