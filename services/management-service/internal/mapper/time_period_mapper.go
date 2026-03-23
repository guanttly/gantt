package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// TimePeriodEntityToModel 将时间段实体转换为领域模型
func TimePeriodEntityToModel(e *entity.TimePeriodEntity) *model.TimePeriod {
	if e == nil {
		return nil
	}
	return &model.TimePeriod{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Code:        e.Code,
		Name:        e.Name,
		StartTime:   e.StartTime,
		EndTime:     e.EndTime,
		IsCrossDay:  e.IsCrossDay,
		Description: e.Description,
		IsActive:    e.IsActive,
		SortOrder:   e.SortOrder,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   e.DeletedAt,
	}
}

// TimePeriodModelToEntity 将时间段领域模型转换为实体
func TimePeriodModelToEntity(m *model.TimePeriod) *entity.TimePeriodEntity {
	if m == nil {
		return nil
	}
	return &entity.TimePeriodEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Code:        m.Code,
		Name:        m.Name,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		IsCrossDay:  m.IsCrossDay,
		Description: m.Description,
		IsActive:    m.IsActive,
		SortOrder:   m.SortOrder,
		DeletedAt:   m.DeletedAt,
		// 注意：不设置 CreatedAt 和 UpdatedAt，让 GORM 通过 autoCreateTime/autoUpdateTime 自动处理
	}
}

// TimePeriodEntitiesToModels 批量转换时间段实体为领域模型
func TimePeriodEntitiesToModels(entities []*entity.TimePeriodEntity) []*model.TimePeriod {
	if entities == nil {
		return nil
	}
	models := make([]*model.TimePeriod, len(entities))
	for i, e := range entities {
		models[i] = TimePeriodEntityToModel(e)
	}
	return models
}
