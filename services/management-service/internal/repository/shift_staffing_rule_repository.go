package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// ShiftStaffingRuleRepository 班次计算规则仓储实现
type ShiftStaffingRuleRepository struct {
	db *gorm.DB
}

// NewShiftStaffingRuleRepository 创建班次计算规则仓储实例
func NewShiftStaffingRuleRepository(db *gorm.DB) repository.IShiftStaffingRuleRepository {
	return &ShiftStaffingRuleRepository{db: db}
}

// Create 创建规则
func (r *ShiftStaffingRuleRepository) Create(ctx context.Context, rule *model.ShiftStaffingRule) error {
	ruleEntity := mapper.ShiftStaffingRuleModelToEntity(rule)
	return r.db.WithContext(ctx).Create(ruleEntity).Error
}

// Update 更新规则
func (r *ShiftStaffingRuleRepository) Update(ctx context.Context, rule *model.ShiftStaffingRule) error {
	ruleEntity := mapper.ShiftStaffingRuleModelToEntity(rule)
	return r.db.WithContext(ctx).
		Model(&entity.ShiftStaffingRuleEntity{}).
		Where("id = ?", rule.ID).
		Omit("created_at").
		Updates(ruleEntity).Error
}

// Delete 删除规则
func (r *ShiftStaffingRuleRepository) Delete(ctx context.Context, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", ruleID).
		Delete(&entity.ShiftStaffingRuleEntity{}).Error
}

// GetByID 根据ID获取规则
func (r *ShiftStaffingRuleRepository) GetByID(ctx context.Context, ruleID string) (*model.ShiftStaffingRule, error) {
	var ruleEntity entity.ShiftStaffingRuleEntity
	err := r.db.WithContext(ctx).
		Where("id = ?", ruleID).
		First(&ruleEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftStaffingRuleEntityToModel(&ruleEntity), nil
}

// GetByShiftID 根据班次ID获取规则
func (r *ShiftStaffingRuleRepository) GetByShiftID(ctx context.Context, shiftID string) (*model.ShiftStaffingRule, error) {
	var ruleEntity entity.ShiftStaffingRuleEntity
	err := r.db.WithContext(ctx).
		Where("shift_id = ?", shiftID).
		First(&ruleEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftStaffingRuleEntityToModel(&ruleEntity), nil
}

// List 查询所有规则
func (r *ShiftStaffingRuleRepository) List(ctx context.Context, orgID string) ([]*model.ShiftStaffingRule, error) {
	// 通过关联班次表查询
	var ruleEntities []*entity.ShiftStaffingRuleEntity
	err := r.db.WithContext(ctx).
		Joins("JOIN shifts ON shifts.id = shift_staffing_rules.shift_id").
		Where("shifts.org_id = ?", orgID).
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftStaffingRuleEntitiesToModels(ruleEntities), nil
}

// ExistsByShiftID 检查班次是否已配置规则
func (r *ShiftStaffingRuleRepository) ExistsByShiftID(ctx context.Context, shiftID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ShiftStaffingRuleEntity{}).
		Where("shift_id = ?", shiftID).
		Count(&count).Error
	return count > 0, err
}
