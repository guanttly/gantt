package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"
)

// SchedulingRuleRepository 排班规则仓储实现
type SchedulingRuleRepository struct {
	db *gorm.DB
}

// NewSchedulingRuleRepository 创建排班规则仓储实例
func NewSchedulingRuleRepository(db *gorm.DB) repository.ISchedulingRuleRepository {
	return &SchedulingRuleRepository{db: db}
}

// Create 创建排班规则
func (r *SchedulingRuleRepository) Create(ctx context.Context, rule *model.SchedulingRule) error {
	ruleEntity := mapper.SchedulingRuleModelToEntity(rule)
	if err := r.db.WithContext(ctx).Create(ruleEntity).Error; err != nil {
		return err
	}
	rule.ID = ruleEntity.ID
	return nil
}

// Update 更新排班规则
func (r *SchedulingRuleRepository) Update(ctx context.Context, rule *model.SchedulingRule) error {
	ruleEntity := mapper.SchedulingRuleModelToEntity(rule)
	return r.db.WithContext(ctx).
		Model(&entity.SchedulingRuleEntity{}).
		Where("org_id = ? AND id = ?", rule.OrgID, rule.ID).
		Omit("created_at").
		Select("*").
		Updates(ruleEntity).Error
}

// Delete 删除排班规则（软删除）
func (r *SchedulingRuleRepository) Delete(ctx context.Context, orgID, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, ruleID).
		Delete(&entity.SchedulingRuleEntity{}).Error
}

// GetByID 根据ID获取规则
func (r *SchedulingRuleRepository) GetByID(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error) {
	var ruleEntity entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, ruleID).
		First(&ruleEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	rule := mapper.SchedulingRuleEntityToModel(&ruleEntity)

	// 加载关联信息
	associations, err := r.GetAssociations(ctx, orgID, ruleID)
	if err != nil {
		return nil, err
	}
	rule.Associations = associations

	return rule, nil
}

// List 查询规则列表
func (r *SchedulingRuleRepository) List(ctx context.Context, filter *model.SchedulingRuleFilter) (*model.SchedulingRuleListResult, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.SchedulingRuleEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 应用过滤条件
	if filter.RuleType != nil {
		query = query.Where("rule_type = ?", *filter.RuleType)
	}
	if filter.ApplyScope != nil {
		query = query.Where("apply_scope = ?", *filter.ApplyScope)
	}
	if filter.TimeScope != nil {
		query = query.Where("time_scope = ?", *filter.TimeScope)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	// V4新增筛选字段
	if filter.Category != nil && *filter.Category != "" {
		query = query.Where("category = ?", *filter.Category)
	}
	if filter.SubCategory != nil && *filter.SubCategory != "" {
		query = query.Where("sub_category = ?", *filter.SubCategory)
	}
	if filter.SourceType != nil && *filter.SourceType != "" {
		query = query.Where("source_type = ?", *filter.SourceType)
	}
	if filter.Version != nil && *filter.Version != "" {
		query = query.Where("version = ?", *filter.Version)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var ruleEntities []*entity.SchedulingRuleEntity
	offset := (filter.Page - 1) * filter.PageSize
	err := query.Offset(offset).Limit(filter.PageSize).
		Order("priority DESC, created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 批量加载关联信息
	for _, rule := range rules {
		associations, err := r.GetAssociations(ctx, filter.OrgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = associations
	}

	return &model.SchedulingRuleListResult{
		Items:    rules,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查规则是否存在
func (r *SchedulingRuleRepository) Exists(ctx context.Context, orgID, ruleID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.SchedulingRuleEntity{}).
		Where("org_id = ? AND id = ?", orgID, ruleID).
		Count(&count).Error
	return count > 0, err
}

// GetByApplyScope 根据作用范围查询规则
func (r *SchedulingRuleRepository) GetByApplyScope(ctx context.Context, orgID string, applyScope model.ApplyScope) ([]*model.SchedulingRule, error) {
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND apply_scope = ? AND is_active = ?", orgID, applyScope, true).
		Order("priority DESC, created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		associations, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = associations
	}

	return rules, nil
}

// GetByRuleType 根据规则类型查询规则
func (r *SchedulingRuleRepository) GetByRuleType(ctx context.Context, orgID string, ruleType model.RuleType) ([]*model.SchedulingRule, error) {
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND rule_type = ? AND is_active = ?", orgID, ruleType, true).
		Order("priority DESC, created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		associations, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = associations
	}

	return rules, nil
}

// GetActiveRules 获取所有启用的规则
func (r *SchedulingRuleRepository) GetActiveRules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error) {
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("priority DESC, created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		associations, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = associations
	}

	return rules, nil
}

// ==================== 关联管理 ====================

// AddAssociations 批量添加规则关联
func (r *SchedulingRuleRepository) AddAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error {
	if len(associations) == 0 {
		return nil
	}

	entities := make([]*entity.SchedulingRuleAssociationEntity, len(associations))
	for i, assoc := range associations {
		entities[i] = &entity.SchedulingRuleAssociationEntity{
			ID:              uuid.New().String(), // 生成唯一ID
			OrgID:           orgID,
			RuleID:          ruleID,
			AssociationType: string(assoc.AssociationType),
			AssociationID:   assoc.AssociationID,
			Role:            assoc.Role, // V4 新增字段
		}
	}

	return r.db.WithContext(ctx).Create(&entities).Error
}

// RemoveAssociations 批量删除规则关联（通过关联ID）
func (r *SchedulingRuleRepository) RemoveAssociations(ctx context.Context, orgID, ruleID string, associationIDs []string) error {
	if len(associationIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ? AND id IN ?", orgID, ruleID, associationIDs).
		Delete(&entity.SchedulingRuleAssociationEntity{}).Error
}

// RemoveAssociationByTarget 删除规则关联（通过目标类型和目标ID）
func (r *SchedulingRuleRepository) RemoveAssociationByTarget(ctx context.Context, orgID, ruleID, targetType, targetID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ? AND association_type = ? AND association_id = ?", orgID, ruleID, targetType, targetID).
		Delete(&entity.SchedulingRuleAssociationEntity{}).Error
}

// GetAssociations 获取规则的所有关联
func (r *SchedulingRuleRepository) GetAssociations(ctx context.Context, orgID, ruleID string) ([]model.RuleAssociation, error) {
	var entities []*entity.SchedulingRuleAssociationEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ?", orgID, ruleID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return mapper.SchedulingRuleAssociationEntitiesToModels(entities), nil
}

// GetRulesForEmployee 获取某个员工相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.SchedulingRule, error) {
	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id = ?",
			orgID, true, orgID, model.AssociationTypeEmployee, employeeID).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	if len(ruleEntities) == 0 {
		return []*model.SchedulingRule{}, nil
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
	}

	return rules, nil
}

// GetRulesForShift 获取某个班次相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error) {
	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id = ?",
			orgID, true, orgID, model.AssociationTypeShift, shiftID).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	if len(ruleEntities) == 0 {
		return []*model.SchedulingRule{}, nil
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
	}

	return rules, nil
}

// GetRulesForGroup 获取某个分组相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.SchedulingRule, error) {
	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重
	var ruleEntities []*entity.SchedulingRuleEntity
	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id = ?",
			orgID, true, orgID, model.AssociationTypeGroup, groupID).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&ruleEntities).Error
	if err != nil {
		return nil, err
	}

	if len(ruleEntities) == 0 {
		return []*model.SchedulingRule{}, nil
	}

	rules := mapper.SchedulingRuleEntitiesToModels(ruleEntities)

	// 加载关联信息
	for _, rule := range rules {
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
	}

	return rules, nil
}

// GetRulesForEmployees 批量获取多个员工相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.SchedulingRule, error) {
	if len(employeeIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重，并批量查询
	var results []struct {
		entity.SchedulingRuleEntity
		EmployeeID string `gorm:"column:employee_id"`
	}

	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*, scheduling_rule_associations.association_id as employee_id").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id IN ?",
			orgID, true, orgID, model.AssociationTypeEmployee, employeeIDs).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	// 按员工ID分组
	resultMap := make(map[string][]*model.SchedulingRule)
	for _, result := range results {
		rule := mapper.SchedulingRuleEntityToModel(&result.SchedulingRuleEntity)
		// 加载关联信息
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
		resultMap[result.EmployeeID] = append(resultMap[result.EmployeeID], rule)
	}

	// 确保所有员工ID都有条目（即使为空）
	for _, employeeID := range employeeIDs {
		if _, exists := resultMap[employeeID]; !exists {
			resultMap[employeeID] = []*model.SchedulingRule{}
		}
	}

	return resultMap, nil
}

// GetRulesForShifts 批量获取多个班次相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.SchedulingRule, error) {
	if len(shiftIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重，并批量查询
	var results []struct {
		entity.SchedulingRuleEntity
		ShiftID string `gorm:"column:shift_id"`
	}

	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*, scheduling_rule_associations.association_id as shift_id").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id IN ?",
			orgID, true, orgID, model.AssociationTypeShift, shiftIDs).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	// 按班次ID分组
	resultMap := make(map[string][]*model.SchedulingRule)
	for _, result := range results {
		rule := mapper.SchedulingRuleEntityToModel(&result.SchedulingRuleEntity)
		// 加载关联信息
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
		resultMap[result.ShiftID] = append(resultMap[result.ShiftID], rule)
	}

	// 确保所有班次ID都有条目（即使为空）
	for _, shiftID := range shiftIDs {
		if _, exists := resultMap[shiftID]; !exists {
			resultMap[shiftID] = []*model.SchedulingRule{}
		}
	}

	return resultMap, nil
}

// GetRulesForGroups 批量获取多个分组相关的所有规则
func (r *SchedulingRuleRepository) GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.SchedulingRule, error) {
	if len(groupIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	// 使用 JOIN 和 DISTINCT 在 SQL 层面去重，并批量查询
	var results []struct {
		entity.SchedulingRuleEntity
		GroupID string `gorm:"column:group_id"`
	}

	err := r.db.WithContext(ctx).
		Table("scheduling_rules").
		Select("DISTINCT scheduling_rules.*, scheduling_rule_associations.association_id as group_id").
		Joins("INNER JOIN scheduling_rule_associations ON scheduling_rules.id = scheduling_rule_associations.rule_id").
		Where("scheduling_rules.org_id = ? AND scheduling_rules.is_active = ? AND scheduling_rule_associations.org_id = ? AND scheduling_rule_associations.association_type = ? AND scheduling_rule_associations.association_id IN ?",
			orgID, true, orgID, model.AssociationTypeGroup, groupIDs).
		Order("scheduling_rules.priority DESC, scheduling_rules.created_at DESC").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	// 按分组ID分组
	resultMap := make(map[string][]*model.SchedulingRule)
	for _, result := range results {
		rule := mapper.SchedulingRuleEntityToModel(&result.SchedulingRuleEntity)
		// 加载关联信息
		assocs, err := r.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Associations = assocs
		resultMap[result.GroupID] = append(resultMap[result.GroupID], rule)
	}

	// 确保所有分组ID都有条目（即使为空）
	for _, groupID := range groupIDs {
		if _, exists := resultMap[groupID]; !exists {
			resultMap[groupID] = []*model.SchedulingRule{}
		}
	}

	return resultMap, nil
}

// ClearAssociations 清空规则的所有关联
func (r *SchedulingRuleRepository) ClearAssociations(ctx context.Context, orgID, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND rule_id = ?", orgID, ruleID).
		Delete(&entity.SchedulingRuleAssociationEntity{}).Error
}
