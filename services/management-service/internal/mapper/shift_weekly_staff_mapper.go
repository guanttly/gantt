package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ShiftWeeklyStaffEntityToModel 将班次周默认人数实体转换为领域模型
func ShiftWeeklyStaffEntityToModel(e *entity.ShiftWeeklyStaffEntity) *model.ShiftWeeklyStaff {
	if e == nil {
		return nil
	}
	return &model.ShiftWeeklyStaff{
		ID:         e.ID,
		ShiftID:    e.ShiftID,
		Weekday:    e.Weekday,
		StaffCount: e.StaffCount,
	}
}

// ShiftWeeklyStaffModelToEntity 将班次周默认人数领域模型转换为实体
func ShiftWeeklyStaffModelToEntity(m *model.ShiftWeeklyStaff) *entity.ShiftWeeklyStaffEntity {
	if m == nil {
		return nil
	}
	return &entity.ShiftWeeklyStaffEntity{
		ID:         m.ID,
		ShiftID:    m.ShiftID,
		Weekday:    m.Weekday,
		StaffCount: m.StaffCount,
	}
}

// ShiftWeeklyStaffEntitiesToModels 批量转换实体为领域模型
func ShiftWeeklyStaffEntitiesToModels(entities []*entity.ShiftWeeklyStaffEntity) []*model.ShiftWeeklyStaff {
	if entities == nil {
		return nil
	}
	models := make([]*model.ShiftWeeklyStaff, len(entities))
	for i, e := range entities {
		models[i] = ShiftWeeklyStaffEntityToModel(e)
	}
	return models
}
