package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ModalityRoomEntityToModel 将机房实体转换为领域模型
func ModalityRoomEntityToModel(e *entity.ModalityRoomEntity) *model.ModalityRoom {
	if e == nil {
		return nil
	}
	return &model.ModalityRoom{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Code:        e.Code,
		Name:        e.Name,
		Description: e.Description,
		Location:    e.Location,
		IsActive:    e.IsActive,
		SortOrder:   e.SortOrder,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   e.DeletedAt,
	}
}

// ModalityRoomModelToEntity 将机房领域模型转换为实体
func ModalityRoomModelToEntity(m *model.ModalityRoom) *entity.ModalityRoomEntity {
	if m == nil {
		return nil
	}
	return &entity.ModalityRoomEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		Location:    m.Location,
		IsActive:    m.IsActive,
		SortOrder:   m.SortOrder,
		DeletedAt:   m.DeletedAt,
		// 注意：不设置 CreatedAt 和 UpdatedAt，让 GORM 通过 autoCreateTime/autoUpdateTime 自动处理
	}
}

// ModalityRoomEntitiesToModels 批量转换机房实体为领域模型
func ModalityRoomEntitiesToModels(entities []*entity.ModalityRoomEntity) []*model.ModalityRoom {
	if entities == nil {
		return nil
	}
	models := make([]*model.ModalityRoom, len(entities))
	for i, e := range entities {
		models[i] = ModalityRoomEntityToModel(e)
	}
	return models
}
