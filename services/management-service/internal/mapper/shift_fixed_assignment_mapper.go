package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ShiftFixedAssignmentEntityToModel 将实体转换为领域模型
func ShiftFixedAssignmentEntityToModel(e *entity.ShiftFixedAssignmentEntity) *model.ShiftFixedAssignment {
	if e == nil {
		return nil
	}

	return &model.ShiftFixedAssignment{
		ID:            e.ID,
		ShiftID:       e.ShiftID,
		StaffID:       e.StaffID,
		PatternType:   e.PatternType,
		Weekdays:      []int(e.Weekdays),
		WeekPattern:   e.WeekPattern,
		Monthdays:     []int(e.Monthdays),
		SpecificDates: []string(e.SpecificDates),
		StartDate:     e.StartDate,
		EndDate:       e.EndDate,
		IsActive:      e.IsActive,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		DeletedAt:     e.DeletedAt,
	}
}

// ShiftFixedAssignmentModelToEntity 将领域模型转换为实体
func ShiftFixedAssignmentModelToEntity(m *model.ShiftFixedAssignment) *entity.ShiftFixedAssignmentEntity {
	if m == nil {
		return nil
	}

	return &entity.ShiftFixedAssignmentEntity{
		ID:            m.ID,
		ShiftID:       m.ShiftID,
		StaffID:       m.StaffID,
		PatternType:   m.PatternType,
		Weekdays:      entity.IntArray(m.Weekdays),
		WeekPattern:   m.WeekPattern,
		Monthdays:     entity.IntArray(m.Monthdays),
		SpecificDates: entity.StringArray(m.SpecificDates),
		StartDate:     m.StartDate,
		EndDate:       m.EndDate,
		IsActive:      m.IsActive,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
}

// ShiftFixedAssignmentEntitiesToModels 批量转换实体到领域模型
func ShiftFixedAssignmentEntitiesToModels(entities []*entity.ShiftFixedAssignmentEntity) []*model.ShiftFixedAssignment {
	if entities == nil {
		return nil
	}

	models := make([]*model.ShiftFixedAssignment, 0, len(entities))
	for _, e := range entities {
		models = append(models, ShiftFixedAssignmentEntityToModel(e))
	}

	return models
}

