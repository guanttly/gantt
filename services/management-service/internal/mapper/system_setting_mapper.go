package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// SystemSettingEntityToModel 将实体转换为领域模型
func SystemSettingEntityToModel(e *entity.SystemSettingEntity) *model.SystemSetting {
	if e == nil {
		return nil
	}

	return &model.SystemSetting{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Key:         e.Key,
		Value:       e.Value,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// SystemSettingModelToEntity 将领域模型转换为实体
func SystemSettingModelToEntity(m *model.SystemSetting) *entity.SystemSettingEntity {
	if m == nil {
		return nil
	}

	return &entity.SystemSettingEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Key:         m.Key,
		Value:       m.Value,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// SystemSettingEntitiesToModels 批量转换实体到领域模型
func SystemSettingEntitiesToModels(entities []*entity.SystemSettingEntity) []*model.SystemSetting {
	if entities == nil {
		return nil
	}

	models := make([]*model.SystemSetting, 0, len(entities))
	for _, e := range entities {
		models = append(models, SystemSettingEntityToModel(e))
	}

	return models
}

