package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"

	domainRepo "jusha/gantt/service/management/domain/repository"
)

// ============================================================================
// RuleApplyScopeRepository 规则适用范围仓储实现
// ============================================================================

type RuleApplyScopeRepository struct {
	db *gorm.DB
}

// NewRuleApplyScopeRepository 创建规则适用范围仓储
func NewRuleApplyScopeRepository(db *gorm.DB) domainRepo.IRuleApplyScopeRepository {
	return &RuleApplyScopeRepository{db: db}
}

// Create 创建适用范围
func (r *RuleApplyScopeRepository) Create(ctx context.Context, scope *model.RuleApplyScope) error {
	entity := &entity.RuleApplyScopeEntity{
		ID:        uuid.New().String(),
		OrgID:     scope.RuleID, // 需要从上下文获取orgID
		RuleID:    scope.RuleID,
		ScopeType: scope.ScopeType,
		ScopeID:   scope.ScopeID,
		ScopeName: scope.ScopeName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(entity).Error
}

// BatchCreate 批量创建适用范围
func (r *RuleApplyScopeRepository) BatchCreate(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error {
	if len(scopes) == 0 {
		return nil
	}

	entities := make([]*entity.RuleApplyScopeEntity, len(scopes))
	now := time.Now()
	for i, s := range scopes {
		entities[i] = &entity.RuleApplyScopeEntity{
			ID:        uuid.New().String(),
			OrgID:     orgID,
			RuleID:    ruleID,
			ScopeType: s.ScopeType,
			ScopeID:   s.ScopeID,
			ScopeName: s.ScopeName,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// Delete 删除适用范围
func (r *RuleApplyScopeRepository) Delete(ctx context.Context, orgID, scopeID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, scopeID).
		Delete(&entity.RuleApplyScopeEntity{}).Error
}

// DeleteByRuleID 删除规则的所有适用范围
func (r *RuleApplyScopeRepository) DeleteByRuleID(ctx context.Context, orgID, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ?", orgID, ruleID).
		Delete(&entity.RuleApplyScopeEntity{}).Error
}

// GetByRuleID 获取规则的所有适用范围
func (r *RuleApplyScopeRepository) GetByRuleID(ctx context.Context, orgID, ruleID string) ([]model.RuleApplyScope, error) {
	var entities []entity.RuleApplyScopeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ?", orgID, ruleID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	scopes := make([]model.RuleApplyScope, len(entities))
	for i, e := range entities {
		scopes[i] = model.RuleApplyScope{
			ID:        e.ID,
			RuleID:    e.RuleID,
			ScopeType: e.ScopeType,
			ScopeID:   e.ScopeID,
			ScopeName: e.ScopeName,
			CreatedAt: e.CreatedAt,
		}
	}
	return scopes, nil
}

// GetByEmployeeID 获取适用于指定员工的规则ID列表
func (r *RuleApplyScopeRepository) GetByEmployeeID(ctx context.Context, orgID, employeeID string) ([]string, error) {
	var ruleIDs []string
	err := r.db.WithContext(ctx).
		Model(&entity.RuleApplyScopeEntity{}).
		Where("org_id = ? AND scope_type = ? AND scope_id = ?", orgID, model.ScopeTypeEmployee, employeeID).
		Distinct("rule_id").
		Pluck("rule_id", &ruleIDs).Error
	return ruleIDs, err
}

// GetByGroupID 获取适用于指定分组的规则ID列表
func (r *RuleApplyScopeRepository) GetByGroupID(ctx context.Context, orgID, groupID string) ([]string, error) {
	var ruleIDs []string
	err := r.db.WithContext(ctx).
		Model(&entity.RuleApplyScopeEntity{}).
		Where("org_id = ? AND scope_type = ? AND scope_id = ?", orgID, model.ScopeTypeGroup, groupID).
		Distinct("rule_id").
		Pluck("rule_id", &ruleIDs).Error
	return ruleIDs, err
}

// GetGlobalRuleIDs 获取全局规则ID列表
func (r *RuleApplyScopeRepository) GetGlobalRuleIDs(ctx context.Context, orgID string) ([]string, error) {
	var ruleIDs []string
	err := r.db.WithContext(ctx).
		Model(&entity.RuleApplyScopeEntity{}).
		Where("org_id = ? AND scope_type = ?", orgID, model.ScopeTypeAll).
		Distinct("rule_id").
		Pluck("rule_id", &ruleIDs).Error
	return ruleIDs, err
}

// ReplaceScopes 替换规则的所有适用范围（先删后增）
func (r *RuleApplyScopeRepository) ReplaceScopes(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除旧范围
		if err := tx.Where("org_id = ? AND rule_id = ?", orgID, ruleID).
			Delete(&entity.RuleApplyScopeEntity{}).Error; err != nil {
			return err
		}

		// 再创建新范围
		if len(scopes) == 0 {
			return nil
		}

		entities := make([]*entity.RuleApplyScopeEntity, len(scopes))
		now := time.Now()
		for i, s := range scopes {
			entities[i] = &entity.RuleApplyScopeEntity{
				ID:        uuid.New().String(),
				OrgID:     orgID,
				RuleID:    ruleID,
				ScopeType: s.ScopeType,
				ScopeID:   s.ScopeID,
				ScopeName: s.ScopeName,
				CreatedAt: now,
				UpdatedAt: now,
			}
		}

		return tx.CreateInBatches(entities, 100).Error
	})
}

// IsRuleApplicableToEmployee 判断规则是否适用于指定员工
func (r *RuleApplyScopeRepository) IsRuleApplicableToEmployee(ctx context.Context, orgID, ruleID, employeeID string, employeeGroupIDs []string) (bool, error) {
	scopes, err := r.GetByRuleID(ctx, orgID, ruleID)
	if err != nil {
		return false, err
	}

	if len(scopes) == 0 {
		return true, nil // 没有范围限定，默认全局生效
	}

	hasInclude := false
	isExcluded := false

	for _, scope := range scopes {
		switch scope.ScopeType {
		case model.ScopeTypeAll:
			hasInclude = true
		case model.ScopeTypeEmployee:
			if scope.ScopeID == employeeID {
				hasInclude = true
			}
		case model.ScopeTypeGroup:
			for _, gid := range employeeGroupIDs {
				if scope.ScopeID == gid {
					hasInclude = true
					break
				}
			}
		case model.ScopeTypeExcludeEmployee:
			if scope.ScopeID == employeeID {
				isExcluded = true
			}
		case model.ScopeTypeExcludeGroup:
			for _, gid := range employeeGroupIDs {
				if scope.ScopeID == gid {
					isExcluded = true
					break
				}
			}
		}
	}

	return hasInclude && !isExcluded, nil
}
