package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// LeaveRecordEntityToModel 将实体转换为领域模型
func LeaveRecordEntityToModel(e *entity.LeaveRecordEntity) *model.LeaveRecord {
	if e == nil {
		return nil
	}

	return &model.LeaveRecord{
		ID:         e.ID,
		OrgID:      e.OrgID,
		EmployeeID: e.EmployeeID,
		Type:       model.LeaveType(e.Type),
		StartDate:  e.StartDate,
		EndDate:    e.EndDate,
		Days:       e.Days,
		StartTime:  e.StartTime,
		EndTime:    e.EndTime,
		Reason:     e.Reason,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
		DeletedAt:  e.DeletedAt,
	}
}

// LeaveRecordModelToEntity 将领域模型转换为实体
func LeaveRecordModelToEntity(m *model.LeaveRecord) *entity.LeaveRecordEntity {
	if m == nil {
		return nil
	}

	return &entity.LeaveRecordEntity{
		ID:         m.ID,
		OrgID:      m.OrgID,
		EmployeeID: m.EmployeeID,
		Type:       string(m.Type),
		StartDate:  m.StartDate,
		EndDate:    m.EndDate,
		Days:       m.Days,
		StartTime:  m.StartTime,
		EndTime:    m.EndTime,
		Reason:     m.Reason,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

// LeaveRecordEntitiesToModels 批量转换实体到领域模型
func LeaveRecordEntitiesToModels(entities []*entity.LeaveRecordEntity) []*model.LeaveRecord {
	if entities == nil {
		return nil
	}

	models := make([]*model.LeaveRecord, 0, len(entities))
	for _, e := range entities {
		models = append(models, LeaveRecordEntityToModel(e))
	}

	return models
}

// LeaveBalanceEntityToModel 将余额实体转换为领域模型
func LeaveBalanceEntityToModel(e *entity.LeaveBalanceEntity) *model.LeaveBalance {
	if e == nil {
		return nil
	}

	return &model.LeaveBalance{
		ID:         e.ID,
		OrgID:      e.OrgID,
		EmployeeID: e.EmployeeID,
		Type:       model.LeaveType(e.Type),
		Year:       e.Year,
		Total:      e.Total,
		Used:       e.Used,
		Remaining:  e.Remaining,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

// LeaveBalanceModelToEntity 将余额领域模型转换为实体
func LeaveBalanceModelToEntity(m *model.LeaveBalance) *entity.LeaveBalanceEntity {
	if m == nil {
		return nil
	}

	return &entity.LeaveBalanceEntity{
		ID:         m.ID,
		OrgID:      m.OrgID,
		EmployeeID: m.EmployeeID,
		Type:       string(m.Type),
		Year:       m.Year,
		Total:      m.Total,
		Used:       m.Used,
		Remaining:  m.Remaining,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

// LeaveBalanceEntitiesToModels 批量转换余额实体到领域模型
func LeaveBalanceEntitiesToModels(entities []*entity.LeaveBalanceEntity) []*model.LeaveBalance {
	if entities == nil {
		return nil
	}

	models := make([]*model.LeaveBalance, 0, len(entities))
	for _, e := range entities {
		models = append(models, LeaveBalanceEntityToModel(e))
	}

	return models
}

// HolidayEntityToModel 将节假日实体转换为领域模型
func HolidayEntityToModel(e *entity.HolidayEntity) *model.Holiday {
	if e == nil {
		return nil
	}

	return &model.Holiday{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Name:        e.Name,
		Date:        e.Date,
		Type:        model.HolidayType(e.Type),
		Description: e.Description,
		Year:        e.Year,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// HolidayModelToEntity 将节假日领域模型转换为实体
func HolidayModelToEntity(m *model.Holiday) *entity.HolidayEntity {
	if m == nil {
		return nil
	}

	return &entity.HolidayEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Name:        m.Name,
		Date:        m.Date,
		Type:        string(m.Type),
		Description: m.Description,
		Year:        m.Year,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// HolidayEntitiesToModels 批量转换节假日实体到领域模型
func HolidayEntitiesToModels(entities []*entity.HolidayEntity) []*model.Holiday {
	if entities == nil {
		return nil
	}

	models := make([]*model.Holiday, 0, len(entities))
	for _, e := range entities {
		models = append(models, HolidayEntityToModel(e))
	}

	return models
}
