package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ModalityRoomWeeklyVolumeEntityToModel 将周检查量实体转换为领域模型
func ModalityRoomWeeklyVolumeEntityToModel(e *entity.ModalityRoomWeeklyVolumeEntity) *model.ModalityRoomWeeklyVolume {
	if e == nil {
		return nil
	}
	return &model.ModalityRoomWeeklyVolume{
		ID:             e.ID,
		OrgID:          e.OrgID,
		ModalityRoomID: e.ModalityRoomID,
		Weekday:        e.Weekday,
		TimePeriodID:   e.TimePeriodID,
		ScanTypeID:     e.ScanTypeID,
		Volume:         e.Volume,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

// ModalityRoomWeeklyVolumeModelToEntity 将周检查量领域模型转换为实体
func ModalityRoomWeeklyVolumeModelToEntity(m *model.ModalityRoomWeeklyVolume) *entity.ModalityRoomWeeklyVolumeEntity {
	if m == nil {
		return nil
	}
	return &entity.ModalityRoomWeeklyVolumeEntity{
		ID:             m.ID,
		OrgID:          m.OrgID,
		ModalityRoomID: m.ModalityRoomID,
		Weekday:        m.Weekday,
		TimePeriodID:   m.TimePeriodID,
		ScanTypeID:     m.ScanTypeID,
		Volume:         m.Volume,
		// 注意：不设置 CreatedAt 和 UpdatedAt，让 GORM 通过 autoCreateTime/autoUpdateTime 自动处理
	}
}

// ModalityRoomWeeklyVolumeEntitiesToModels 批量转换周检查量实体为领域模型
func ModalityRoomWeeklyVolumeEntitiesToModels(entities []*entity.ModalityRoomWeeklyVolumeEntity) []*model.ModalityRoomWeeklyVolume {
	if entities == nil {
		return nil
	}
	models := make([]*model.ModalityRoomWeeklyVolume, len(entities))
	for i, e := range entities {
		models[i] = ModalityRoomWeeklyVolumeEntityToModel(e)
	}
	return models
}
