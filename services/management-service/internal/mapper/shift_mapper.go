package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ShiftEntityToModel 将实体转换为领域模型
func ShiftEntityToModel(e *entity.ShiftEntity) *model.Shift {
	if e == nil {
		return nil
	}

	return &model.Shift{
		ID:                 e.ID,
		OrgID:              e.OrgID,
		Name:               e.Name,
		Code:               e.Code,
		Type:               model.ShiftType(e.Type),
		Description:        e.Description,
		StartTime:          e.StartTime,
		EndTime:            e.EndTime,
		Duration:           e.Duration,
		IsOvernight:        e.IsOvernight,
		Color:              e.Color,
		Priority:           e.Priority,
		SchedulingPriority: e.SchedulingPriority,
		IsActive:           e.IsActive,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
		DeletedAt:          e.DeletedAt,
	}
}

// ShiftModelToEntity 将领域模型转换为实体
func ShiftModelToEntity(m *model.Shift) *entity.ShiftEntity {
	if m == nil {
		return nil
	}

	return &entity.ShiftEntity{
		ID:                 m.ID,
		OrgID:              m.OrgID,
		Name:               m.Name,
		Code:               m.Code,
		Type:               string(m.Type),
		Description:        m.Description,
		StartTime:          m.StartTime,
		EndTime:            m.EndTime,
		Duration:           m.Duration,
		IsOvernight:        m.IsOvernight,
		Color:              m.Color,
		Priority:           m.Priority,
		SchedulingPriority: m.SchedulingPriority,
		IsActive:           m.IsActive,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
		DeletedAt:          m.DeletedAt,
	}
}

// ShiftEntitiesToModels 批量转换实体到领域模型
func ShiftEntitiesToModels(entities []*entity.ShiftEntity) []*model.Shift {
	if entities == nil {
		return nil
	}

	models := make([]*model.Shift, 0, len(entities))
	for _, e := range entities {
		models = append(models, ShiftEntityToModel(e))
	}

	return models
}

// ShiftAssignmentEntityToModel 将分配实体转换为领域模型
func ShiftAssignmentEntityToModel(e *entity.ShiftAssignmentEntity) *model.ShiftAssignment {
	if e == nil {
		return nil
	}

	return &model.ShiftAssignment{
		ID:         e.ID,
		OrgID:      e.OrgID,
		EmployeeID: e.EmployeeID,
		ShiftID:    e.ShiftID,
		Date:       e.Date,
		Notes:      e.Notes,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
		DeletedAt:  e.DeletedAt,
	}
}

// ShiftAssignmentModelToEntity 将分配领域模型转换为实体
func ShiftAssignmentModelToEntity(m *model.ShiftAssignment) *entity.ShiftAssignmentEntity {
	if m == nil {
		return nil
	}

	return &entity.ShiftAssignmentEntity{
		ID:         m.ID,
		OrgID:      m.OrgID,
		EmployeeID: m.EmployeeID,
		ShiftID:    m.ShiftID,
		Date:       m.Date,
		Notes:      m.Notes,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

// ShiftAssignmentEntitiesToModels 批量转换分配实体到领域模型
func ShiftAssignmentEntitiesToModels(entities []*entity.ShiftAssignmentEntity) []*model.ShiftAssignment {
	if entities == nil {
		return nil
	}

	models := make([]*model.ShiftAssignment, 0, len(entities))
	for _, e := range entities {
		models = append(models, ShiftAssignmentEntityToModel(e))
	}

	return models
}

// ShiftGroupEntityToModel 将班次-分组关联实体转换为领域模型
func ShiftGroupEntityToModel(e *entity.ShiftGroupEntity) *model.ShiftGroup {
	if e == nil {
		return nil
	}

	return &model.ShiftGroup{
		ID:        e.ID,
		ShiftID:   e.ShiftID,
		GroupID:   e.GroupID,
		Priority:  e.Priority,
		IsActive:  e.IsActive,
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// ShiftGroupModelToEntity 将班次-分组关联领域模型转换为实体
func ShiftGroupModelToEntity(m *model.ShiftGroup) *entity.ShiftGroupEntity {
	if m == nil {
		return nil
	}

	return &entity.ShiftGroupEntity{
		ID:        m.ID,
		ShiftID:   m.ShiftID,
		GroupID:   m.GroupID,
		Priority:  m.Priority,
		IsActive:  m.IsActive,
		Notes:     m.Notes,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ShiftGroupEntitiesToModels 批量转换班次-分组关联实体到领域模型
func ShiftGroupEntitiesToModels(entities []*entity.ShiftGroupEntity) []*model.ShiftGroup {
	if entities == nil {
		return nil
	}

	models := make([]*model.ShiftGroup, 0, len(entities))
	for _, e := range entities {
		models = append(models, ShiftGroupEntityToModel(e))
	}

	return models
}
