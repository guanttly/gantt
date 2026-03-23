package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// RuleDependencyEntityToModel 将规则依赖关系实体转换为领域模型
func RuleDependencyEntityToModel(e *entity.RuleDependencyEntity) *model.RuleDependency {
	if e == nil {
		return nil
	}

	return &model.RuleDependency{
		ID:                e.ID,
		OrgID:             e.OrgID,
		DependentRuleID:   e.DependentRuleID,
		DependentOnRuleID: e.DependentOnRuleID,
		DependencyType:    e.DependencyType,
		Description:       e.Description,
		CreatedAt:         e.CreatedAt,
	}
}

// RuleDependencyModelToEntity 将规则依赖关系领域模型转换为实体
func RuleDependencyModelToEntity(m *model.RuleDependency) *entity.RuleDependencyEntity {
	if m == nil {
		return nil
	}

	return &entity.RuleDependencyEntity{
		ID:                m.ID,
		OrgID:             m.OrgID,
		DependentRuleID:   m.DependentRuleID,
		DependentOnRuleID: m.DependentOnRuleID,
		DependencyType:    m.DependencyType,
		Description:       m.Description,
		CreatedAt:         m.CreatedAt,
	}
}

// RuleConflictEntityToModel 将规则冲突关系实体转换为领域模型
func RuleConflictEntityToModel(e *entity.RuleConflictEntity) *model.RuleConflict {
	if e == nil {
		return nil
	}

	return &model.RuleConflict{
		ID:                 e.ID,
		OrgID:              e.OrgID,
		RuleID1:            e.RuleID1,
		RuleID2:            e.RuleID2,
		ConflictType:       e.ConflictType,
		Description:        e.Description,
		ResolutionPriority: e.ResolutionPriority,
		CreatedAt:          e.CreatedAt,
	}
}

// RuleConflictModelToEntity 将规则冲突关系领域模型转换为实体
func RuleConflictModelToEntity(m *model.RuleConflict) *entity.RuleConflictEntity {
	if m == nil {
		return nil
	}

	return &entity.RuleConflictEntity{
		ID:                 m.ID,
		OrgID:              m.OrgID,
		RuleID1:            m.RuleID1,
		RuleID2:            m.RuleID2,
		ConflictType:       m.ConflictType,
		Description:        m.Description,
		ResolutionPriority: m.ResolutionPriority,
		CreatedAt:          m.CreatedAt,
	}
}

// ShiftDependencyEntityToModel 将班次依赖关系实体转换为领域模型
func ShiftDependencyEntityToModel(e *entity.ShiftDependencyEntity) *model.ShiftDependency {
	if e == nil {
		return nil
	}

	return &model.ShiftDependency{
		ID:                 e.ID,
		OrgID:              e.OrgID,
		DependentShiftID:   e.DependentShiftID,
		DependentOnShiftID: e.DependentOnShiftID,
		DependencyType:     e.DependencyType,
		RuleID:             e.RuleID,
		Description:        e.Description,
		CreatedAt:          e.CreatedAt,
	}
}

// ShiftDependencyModelToEntity 将班次依赖关系领域模型转换为实体
func ShiftDependencyModelToEntity(m *model.ShiftDependency) *entity.ShiftDependencyEntity {
	if m == nil {
		return nil
	}

	return &entity.ShiftDependencyEntity{
		ID:                 m.ID,
		OrgID:              m.OrgID,
		DependentShiftID:   m.DependentShiftID,
		DependentOnShiftID: m.DependentOnShiftID,
		DependencyType:     m.DependencyType,
		RuleID:             m.RuleID,
		Description:        m.Description,
		CreatedAt:          m.CreatedAt,
	}
}
