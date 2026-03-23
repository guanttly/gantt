package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ModalityRoomVolumeEntityToModel 将检查量实体转换为领域模型
func ModalityRoomVolumeEntityToModel(e *entity.ModalityRoomVolumeEntity) *model.ModalityRoomVolume {
	if e == nil {
		return nil
	}
	return &model.ModalityRoomVolume{
		ID:             e.ID,
		OrgID:          e.OrgID,
		ModalityRoomID: e.ModalityRoomID,
		Date:           e.Date,
		TimePeriodID:   e.TimePeriodID,
		ReportVolume:   e.ReportVolume,
		Notes:          e.Notes,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

// ModalityRoomVolumeModelToEntity 将检查量领域模型转换为实体
func ModalityRoomVolumeModelToEntity(m *model.ModalityRoomVolume) *entity.ModalityRoomVolumeEntity {
	if m == nil {
		return nil
	}
	return &entity.ModalityRoomVolumeEntity{
		ID:             m.ID,
		OrgID:          m.OrgID,
		ModalityRoomID: m.ModalityRoomID,
		Date:           m.Date,
		TimePeriodID:   m.TimePeriodID,
		ReportVolume:   m.ReportVolume,
		Notes:          m.Notes,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// ModalityRoomVolumeEntitiesToModels 批量转换检查量实体为领域模型
func ModalityRoomVolumeEntitiesToModels(entities []*entity.ModalityRoomVolumeEntity) []*model.ModalityRoomVolume {
	if entities == nil {
		return nil
	}
	models := make([]*model.ModalityRoomVolume, len(entities))
	for i, e := range entities {
		models[i] = ModalityRoomVolumeEntityToModel(e)
	}
	return models
}
