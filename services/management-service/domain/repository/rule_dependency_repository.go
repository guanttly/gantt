package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IRuleDependencyRepository 规则依赖关系仓储接口
type IRuleDependencyRepository interface {
	// Create 创建规则依赖关系
	Create(ctx context.Context, dependency *model.RuleDependency) error

	// Delete 删除规则依赖关系
	Delete(ctx context.Context, orgID, dependencyID string) error

	// GetByOrgID 获取组织的所有规则依赖关系
	GetByOrgID(ctx context.Context, orgID string) ([]*model.RuleDependency, error)

	// GetByRuleID 获取某个规则的所有依赖关系
	GetByRuleID(ctx context.Context, orgID, ruleID string) ([]*model.RuleDependency, error)

	// BatchCreate 批量创建规则依赖关系
	BatchCreate(ctx context.Context, dependencies []*model.RuleDependency) error
}

// IRuleConflictRepository 规则冲突关系仓储接口
type IRuleConflictRepository interface {
	// Create 创建规则冲突关系
	Create(ctx context.Context, conflict *model.RuleConflict) error

	// Delete 删除规则冲突关系
	Delete(ctx context.Context, orgID, conflictID string) error

	// GetByOrgID 获取组织的所有规则冲突关系
	GetByOrgID(ctx context.Context, orgID string) ([]*model.RuleConflict, error)

	// GetByRuleID 获取某个规则的所有冲突关系
	GetByRuleID(ctx context.Context, orgID, ruleID string) ([]*model.RuleConflict, error)

	// BatchCreate 批量创建规则冲突关系
	BatchCreate(ctx context.Context, conflicts []*model.RuleConflict) error
}

// IShiftDependencyRepository 班次依赖关系仓储接口
type IShiftDependencyRepository interface {
	// Create 创建班次依赖关系
	Create(ctx context.Context, dependency *model.ShiftDependency) error

	// Delete 删除班次依赖关系
	Delete(ctx context.Context, orgID, dependencyID string) error

	// GetByOrgID 获取组织的所有班次依赖关系
	GetByOrgID(ctx context.Context, orgID string) ([]*model.ShiftDependency, error)

	// GetByShiftID 获取某个班次的所有依赖关系
	GetByShiftID(ctx context.Context, orgID, shiftID string) ([]*model.ShiftDependency, error)

	// BatchCreate 批量创建班次依赖关系
	BatchCreate(ctx context.Context, dependencies []*model.ShiftDependency) error
}
