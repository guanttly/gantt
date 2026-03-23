package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"
)

// RuleDependencyRepository 规则依赖关系仓储实现
type RuleDependencyRepository struct {
	db *gorm.DB
}

// NewRuleDependencyRepository 创建规则依赖关系仓储实例
func NewRuleDependencyRepository(db *gorm.DB) repository.IRuleDependencyRepository {
	return &RuleDependencyRepository{db: db}
}

// Create 创建规则依赖关系
func (r *RuleDependencyRepository) Create(ctx context.Context, dependency *model.RuleDependency) error {
	if dependency.ID == "" {
		dependency.ID = uuid.New().String()
	}
	if dependency.CreatedAt.IsZero() {
		dependency.CreatedAt = time.Now()
	}

	entity := &entity.RuleDependencyEntity{
		ID:                dependency.ID,
		OrgID:             dependency.OrgID,
		DependentRuleID:   dependency.DependentRuleID,
		DependentOnRuleID: dependency.DependentOnRuleID,
		DependencyType:    dependency.DependencyType,
		Description:       dependency.Description,
		CreatedAt:         dependency.CreatedAt,
	}

	return r.db.WithContext(ctx).Create(entity).Error
}

// Delete 删除规则依赖关系
func (r *RuleDependencyRepository) Delete(ctx context.Context, orgID, dependencyID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, dependencyID).
		Delete(&entity.RuleDependencyEntity{}).Error
}

// GetByOrgID 获取组织的所有规则依赖关系
func (r *RuleDependencyRepository) GetByOrgID(ctx context.Context, orgID string) ([]*model.RuleDependency, error) {
	var entities []*entity.RuleDependencyEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.RuleDependency, len(entities))
	for i, e := range entities {
		result[i] = mapper.RuleDependencyEntityToModel(e)
	}
	return result, nil
}

// GetByRuleID 获取某个规则的所有依赖关系
func (r *RuleDependencyRepository) GetByRuleID(ctx context.Context, orgID, ruleID string) ([]*model.RuleDependency, error) {
	var entities []*entity.RuleDependencyEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ? AND (dependent_rule_id = ? OR dependent_on_rule_id = ?)", orgID, ruleID, ruleID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.RuleDependency, len(entities))
	for i, e := range entities {
		result[i] = mapper.RuleDependencyEntityToModel(e)
	}
	return result, nil
}

// BatchCreate 批量创建规则依赖关系
func (r *RuleDependencyRepository) BatchCreate(ctx context.Context, dependencies []*model.RuleDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	entities := make([]*entity.RuleDependencyEntity, len(dependencies))
	for i, dep := range dependencies {
		if dep.ID == "" {
			dep.ID = uuid.New().String()
		}
		if dep.CreatedAt.IsZero() {
			dep.CreatedAt = time.Now()
		}
		entities[i] = &entity.RuleDependencyEntity{
			ID:                dep.ID,
			OrgID:             dep.OrgID,
			DependentRuleID:   dep.DependentRuleID,
			DependentOnRuleID: dep.DependentOnRuleID,
			DependencyType:    dep.DependencyType,
			Description:       dep.Description,
			CreatedAt:         dep.CreatedAt,
		}
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// RuleConflictRepository 规则冲突关系仓储实现
type RuleConflictRepository struct {
	db *gorm.DB
}

// NewRuleConflictRepository 创建规则冲突关系仓储实例
func NewRuleConflictRepository(db *gorm.DB) repository.IRuleConflictRepository {
	return &RuleConflictRepository{db: db}
}

// Create 创建规则冲突关系
func (r *RuleConflictRepository) Create(ctx context.Context, conflict *model.RuleConflict) error {
	if conflict.ID == "" {
		conflict.ID = uuid.New().String()
	}
	if conflict.CreatedAt.IsZero() {
		conflict.CreatedAt = time.Now()
	}

	entity := &entity.RuleConflictEntity{
		ID:                 conflict.ID,
		OrgID:              conflict.OrgID,
		RuleID1:            conflict.RuleID1,
		RuleID2:            conflict.RuleID2,
		ConflictType:       conflict.ConflictType,
		Description:        conflict.Description,
		ResolutionPriority: conflict.ResolutionPriority,
		CreatedAt:          conflict.CreatedAt,
	}

	return r.db.WithContext(ctx).Create(entity).Error
}

// Delete 删除规则冲突关系
func (r *RuleConflictRepository) Delete(ctx context.Context, orgID, conflictID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, conflictID).
		Delete(&entity.RuleConflictEntity{}).Error
}

// GetByOrgID 获取组织的所有规则冲突关系
func (r *RuleConflictRepository) GetByOrgID(ctx context.Context, orgID string) ([]*model.RuleConflict, error) {
	var entities []*entity.RuleConflictEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.RuleConflict, len(entities))
	for i, e := range entities {
		result[i] = mapper.RuleConflictEntityToModel(e)
	}
	return result, nil
}

// GetByRuleID 获取某个规则的所有冲突关系
func (r *RuleConflictRepository) GetByRuleID(ctx context.Context, orgID, ruleID string) ([]*model.RuleConflict, error) {
	var entities []*entity.RuleConflictEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ? AND (rule_id_1 = ? OR rule_id_2 = ?)", orgID, ruleID, ruleID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.RuleConflict, len(entities))
	for i, e := range entities {
		result[i] = mapper.RuleConflictEntityToModel(e)
	}
	return result, nil
}

// BatchCreate 批量创建规则冲突关系
func (r *RuleConflictRepository) BatchCreate(ctx context.Context, conflicts []*model.RuleConflict) error {
	if len(conflicts) == 0 {
		return nil
	}

	entities := make([]*entity.RuleConflictEntity, len(conflicts))
	for i, conf := range conflicts {
		if conf.ID == "" {
			conf.ID = uuid.New().String()
		}
		if conf.CreatedAt.IsZero() {
			conf.CreatedAt = time.Now()
		}
		entities[i] = &entity.RuleConflictEntity{
			ID:                 conf.ID,
			OrgID:              conf.OrgID,
			RuleID1:            conf.RuleID1,
			RuleID2:            conf.RuleID2,
			ConflictType:       conf.ConflictType,
			Description:        conf.Description,
			ResolutionPriority: conf.ResolutionPriority,
			CreatedAt:          conf.CreatedAt,
		}
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// ShiftDependencyRepository 班次依赖关系仓储实现
type ShiftDependencyRepository struct {
	db *gorm.DB
}

// NewShiftDependencyRepository 创建班次依赖关系仓储实例
func NewShiftDependencyRepository(db *gorm.DB) repository.IShiftDependencyRepository {
	return &ShiftDependencyRepository{db: db}
}

// Create 创建班次依赖关系
func (r *ShiftDependencyRepository) Create(ctx context.Context, dependency *model.ShiftDependency) error {
	if dependency.ID == "" {
		dependency.ID = uuid.New().String()
	}
	if dependency.CreatedAt.IsZero() {
		dependency.CreatedAt = time.Now()
	}

	entity := &entity.ShiftDependencyEntity{
		ID:                 dependency.ID,
		OrgID:              dependency.OrgID,
		DependentShiftID:   dependency.DependentShiftID,
		DependentOnShiftID: dependency.DependentOnShiftID,
		DependencyType:     dependency.DependencyType,
		RuleID:             dependency.RuleID,
		Description:        dependency.Description,
		CreatedAt:          dependency.CreatedAt,
	}

	return r.db.WithContext(ctx).Create(entity).Error
}

// Delete 删除班次依赖关系
func (r *ShiftDependencyRepository) Delete(ctx context.Context, orgID, dependencyID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, dependencyID).
		Delete(&entity.ShiftDependencyEntity{}).Error
}

// GetByOrgID 获取组织的所有班次依赖关系
func (r *ShiftDependencyRepository) GetByOrgID(ctx context.Context, orgID string) ([]*model.ShiftDependency, error) {
	var entities []*entity.ShiftDependencyEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.ShiftDependency, len(entities))
	for i, e := range entities {
		result[i] = mapper.ShiftDependencyEntityToModel(e)
	}
	return result, nil
}

// GetByShiftID 获取某个班次的所有依赖关系
func (r *ShiftDependencyRepository) GetByShiftID(ctx context.Context, orgID, shiftID string) ([]*model.ShiftDependency, error) {
	var entities []*entity.ShiftDependencyEntity
	if err := r.db.WithContext(ctx).
		Where("org_id = ? AND (dependent_shift_id = ? OR dependent_on_shift_id = ?)", orgID, shiftID, shiftID).
		Find(&entities).Error; err != nil {
		return nil, err
	}

	result := make([]*model.ShiftDependency, len(entities))
	for i, e := range entities {
		result[i] = mapper.ShiftDependencyEntityToModel(e)
	}
	return result, nil
}

// BatchCreate 批量创建班次依赖关系
func (r *ShiftDependencyRepository) BatchCreate(ctx context.Context, dependencies []*model.ShiftDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	entities := make([]*entity.ShiftDependencyEntity, len(dependencies))
	for i, dep := range dependencies {
		if dep.ID == "" {
			dep.ID = uuid.New().String()
		}
		if dep.CreatedAt.IsZero() {
			dep.CreatedAt = time.Now()
		}
		entities[i] = &entity.ShiftDependencyEntity{
			ID:                 dep.ID,
			OrgID:              dep.OrgID,
			DependentShiftID:   dep.DependentShiftID,
			DependentOnShiftID: dep.DependentOnShiftID,
			DependencyType:     dep.DependencyType,
			RuleID:             dep.RuleID,
			Description:        dep.Description,
			CreatedAt:          dep.CreatedAt,
		}
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}
