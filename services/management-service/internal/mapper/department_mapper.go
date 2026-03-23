package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// DepartmentEntityToModel 将实体转换为领域模型
func DepartmentEntityToModel(e *entity.DepartmentEntity) *model.Department {
	if e == nil {
		return nil
	}

	return &model.Department{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Code:        e.Code,
		Name:        e.Name,
		ParentID:    e.ParentID,
		Level:       e.Level,
		Path:        e.Path,
		Description: e.Description,
		ManagerID:   e.ManagerID,
		SortOrder:   e.SortOrder,
		IsActive:    e.IsActive,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   e.DeletedAt,
	}
}

// DepartmentModelToEntity 将领域模型转换为实体
func DepartmentModelToEntity(m *model.Department) *entity.DepartmentEntity {
	if m == nil {
		return nil
	}

	return &entity.DepartmentEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Code:        m.Code,
		Name:        m.Name,
		ParentID:    m.ParentID,
		Level:       m.Level,
		Path:        m.Path,
		Description: m.Description,
		ManagerID:   m.ManagerID,
		SortOrder:   m.SortOrder,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
	}
}

// DepartmentEntitiesToModels 批量转换实体到领域模型
func DepartmentEntitiesToModels(entities []*entity.DepartmentEntity) []*model.Department {
	if entities == nil {
		return nil
	}

	models := make([]*model.Department, 0, len(entities))
	for _, e := range entities {
		models = append(models, DepartmentEntityToModel(e))
	}

	return models
}
