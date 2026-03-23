package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ScanTypeEntityToModel 将检查类型实体转换为领域模型
func ScanTypeEntityToModel(e *entity.ScanTypeEntity) *model.ScanType {
	if e == nil {
		return nil
	}
	return &model.ScanType{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Code:        e.Code,
		Name:        e.Name,
		Description: e.Description,
		IsActive:    e.IsActive,
		SortOrder:   e.SortOrder,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   e.DeletedAt,
	}
}

// ScanTypeModelToEntity 将检查类型领域模型转换为实体
func ScanTypeModelToEntity(m *model.ScanType) *entity.ScanTypeEntity {
	if m == nil {
		return nil
	}
	return &entity.ScanTypeEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		IsActive:    m.IsActive,
		SortOrder:   m.SortOrder,
		DeletedAt:   m.DeletedAt,
		// 注意：不设置 CreatedAt 和 UpdatedAt，让 GORM 通过 autoCreateTime/autoUpdateTime 自动处理
	}
}

// ScanTypeEntitiesToModels 批量转换检查类型实体为领域模型
func ScanTypeEntitiesToModels(entities []*entity.ScanTypeEntity) []*model.ScanType {
	if entities == nil {
		return nil
	}
	models := make([]*model.ScanType, len(entities))
	for i, e := range entities {
		models[i] = ScanTypeEntityToModel(e)
	}
	return models
}
