package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// SchedulingRuleEntityToModel 将排班规则实体转换为领域模型
func SchedulingRuleEntityToModel(e *entity.SchedulingRuleEntity) *model.SchedulingRule {
	if e == nil {
		return nil
	}

	return &model.SchedulingRule{
		ID:              e.ID,
		OrgID:           e.OrgID,
		Name:            e.Name,
		Description:     e.Description,
		RuleType:        model.RuleType(e.RuleType),
		ApplyScope:      model.ApplyScope(e.ApplyScope),
		TimeScope:       model.TimeScope(e.TimeScope),
		RuleData:        e.RuleData,
		MaxCount:        e.MaxCount,
		ConsecutiveMax:  e.ConsecutiveMax,
		IntervalDays:    e.IntervalDays,
		MinRestDays:     e.MinRestDays,
		TimeOffsetDays:  e.TimeOffsetDays,
		Priority:        e.Priority,
		IsActive:        e.IsActive,
		ValidFrom:       e.ValidFrom,
		ValidTo:         e.ValidTo,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
		DeletedAt:       e.DeletedAt,
		Category:        e.Category,
		SubCategory:     e.SubCategory,
		OriginalRuleID:  e.OriginalRuleID,
		SourceType:      e.SourceType,
		ParseConfidence: e.ParseConfidence,
		Version:         e.Version,
	}
}

// SchedulingRuleModelToEntity 将排班规则领域模型转换为实体
func SchedulingRuleModelToEntity(m *model.SchedulingRule) *entity.SchedulingRuleEntity {
	if m == nil {
		return nil
	}

	return &entity.SchedulingRuleEntity{
		ID:              m.ID,
		OrgID:           m.OrgID,
		Name:            m.Name,
		Description:     m.Description,
		RuleType:        string(m.RuleType),
		ApplyScope:      string(m.ApplyScope),
		TimeScope:       string(m.TimeScope),
		RuleData:        m.RuleData,
		MaxCount:        m.MaxCount,
		ConsecutiveMax:  m.ConsecutiveMax,
		IntervalDays:    m.IntervalDays,
		MinRestDays:     m.MinRestDays,
		TimeOffsetDays:  m.TimeOffsetDays,
		Priority:        m.Priority,
		IsActive:        m.IsActive,
		ValidFrom:       m.ValidFrom,
		ValidTo:         m.ValidTo,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		DeletedAt:       m.DeletedAt,
		Category:        m.Category,
		SubCategory:     m.SubCategory,
		OriginalRuleID:  m.OriginalRuleID,
		SourceType:      m.SourceType,
		ParseConfidence: m.ParseConfidence,
		Version:         m.Version,
	}
}

// SchedulingRuleEntitiesToModels 批量转换排班规则实体为领域模型
func SchedulingRuleEntitiesToModels(entities []*entity.SchedulingRuleEntity) []*model.SchedulingRule {
	if entities == nil {
		return nil
	}

	models := make([]*model.SchedulingRule, 0, len(entities))
	for _, e := range entities {
		models = append(models, SchedulingRuleEntityToModel(e))
	}
	return models
}

// SchedulingRuleAssociationEntityToModel 将规则关联实体转换为领域模型
func SchedulingRuleAssociationEntityToModel(e *entity.SchedulingRuleAssociationEntity) *model.RuleAssociation {
	if e == nil {
		return nil
	}

	return &model.RuleAssociation{
		ID:              e.ID,
		RuleID:          e.RuleID,
		AssociationType: model.AssociationType(e.AssociationType),
		AssociationID:   e.AssociationID,
		Role:            e.Role,
		CreatedAt:       e.CreatedAt,
	}
}

// SchedulingRuleAssociationModelToEntity 将规则关联领域模型转换为实体
func SchedulingRuleAssociationModelToEntity(m *model.RuleAssociation) *entity.SchedulingRuleAssociationEntity {
	if m == nil {
		return nil
	}

	entity := &entity.SchedulingRuleAssociationEntity{
		ID:              m.ID,
		RuleID:          m.RuleID,
		AssociationType: string(m.AssociationType),
		AssociationID:   m.AssociationID,
		Role:            m.Role,
		CreatedAt:       m.CreatedAt,
	}
	// 如果没有指定Role，默认使用target
	if entity.Role == "" {
		entity.Role = "target"
	}
	return entity
}

// SchedulingRuleAssociationEntitiesToModels 批量转换规则关联实体为领域模型
func SchedulingRuleAssociationEntitiesToModels(entities []*entity.SchedulingRuleAssociationEntity) []model.RuleAssociation {
	if entities == nil {
		return nil
	}

	models := make([]model.RuleAssociation, 0, len(entities))
	for _, e := range entities {
		if assoc := SchedulingRuleAssociationEntityToModel(e); assoc != nil {
			models = append(models, *assoc)
		}
	}
	return models
}
