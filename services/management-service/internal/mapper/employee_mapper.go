package mapper

import (
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// EmployeeEntityToModel 将实体转换为领域模型
func EmployeeEntityToModel(e *entity.EmployeeEntity) (*model.Employee, error) {
	if e == nil {
		return nil, nil
	}

	employee := &model.Employee{
		ID:           e.ID,
		OrgID:        e.OrgID,
		EmployeeID:   e.EmployeeID,
		UserID:       e.UserID,
		Name:         e.Name,
		Phone:        e.Phone,
		Email:        e.Email,
		DepartmentID: e.DepartmentID,
		Position:     e.Position,
		Role:         e.Role,
		Status:       model.EmployeeStatus(e.Status),
		HireDate:     e.HireDate,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
	}

	return employee, nil
}

// EmployeeModelToEntity 将领域模型转换为实体
func EmployeeModelToEntity(m *model.Employee) (*entity.EmployeeEntity, error) {
	if m == nil {
		return nil, nil
	}

	employee := &entity.EmployeeEntity{
		ID:           m.ID,
		OrgID:        m.OrgID,
		EmployeeID:   m.EmployeeID,
		UserID:       m.UserID,
		Name:         m.Name,
		Phone:        m.Phone,
		Email:        m.Email,
		DepartmentID: m.DepartmentID,
		Position:     m.Position,
		Role:         m.Role,
		Status:       string(m.Status),
		HireDate:     m.HireDate,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}

	return employee, nil
}

// EmployeeEntitiesToModels 批量转换实体到领域模型
func EmployeeEntitiesToModels(entities []*entity.EmployeeEntity) ([]*model.Employee, error) {
	if entities == nil {
		return nil, nil
	}

	models := make([]*model.Employee, 0, len(entities))
	for _, e := range entities {
		m, err := EmployeeEntityToModel(e)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}

	return models, nil
}
