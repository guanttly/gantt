package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	domain_repo "jusha/gantt/service/management/domain/repository"
)

// EmployeeRepository 员工仓储实现
type EmployeeRepository struct {
	db *gorm.DB
}

// NewEmployeeRepository 创建员工仓储
func NewEmployeeRepository(db *gorm.DB) domain_repo.IEmployeeRepository {
	return &EmployeeRepository{db: db}
}

// Create 创建员工
func (r *EmployeeRepository) Create(ctx context.Context, employee *model.Employee) error {
	employeeEntity, err := mapper.EmployeeModelToEntity(employee)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(employeeEntity).Error
}

// Update 更新员工信息
func (r *EmployeeRepository) Update(ctx context.Context, employee *model.Employee) error {
	employeeEntity, err := mapper.EmployeeModelToEntity(employee)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(employeeEntity).Error
}

// Delete 删除员工（软删除）
func (r *EmployeeRepository) Delete(ctx context.Context, orgID, employeeID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, employeeID).
		Delete(&entity.EmployeeEntity{}).Error
}

// GetByID 根据ID获取员工
func (r *EmployeeRepository) GetByID(ctx context.Context, orgID, employeeID string) (*model.Employee, error) {
	var employeeEntity entity.EmployeeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, employeeID).
		First(&employeeEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("employee not found")
		}
		return nil, err
	}
	return mapper.EmployeeEntityToModel(&employeeEntity)
}

// GetByEmployeeID 根据工号获取员工
func (r *EmployeeRepository) GetByEmployeeID(ctx context.Context, orgID, employeeNo string) (*model.Employee, error) {
	var employeeEntity entity.EmployeeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ?", orgID, employeeNo).
		First(&employeeEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("employee not found")
		}
		return nil, err
	}
	return mapper.EmployeeEntityToModel(&employeeEntity)
}

// GetByUserID 根据用户ID获取员工
func (r *EmployeeRepository) GetByUserID(ctx context.Context, orgID, userID string) (*model.Employee, error) {
	var employeeEntity entity.EmployeeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&employeeEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("employee not found")
		}
		return nil, err
	}
	return mapper.EmployeeEntityToModel(&employeeEntity)
}

// List 查询员工列表
func (r *EmployeeRepository) List(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error) {
	query := r.db.WithContext(ctx).Model(&entity.EmployeeEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 应用过滤条件
	if filter.Department != "" {
		query = query.Where("department = ?", filter.Department)
	}
	if filter.Position != "" {
		query = query.Where("position = ?", filter.Position)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR employee_id LIKE ? OR phone LIKE ?", keyword, keyword, keyword)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var employeeEntities []*entity.EmployeeEntity
	offset := (filter.Page - 1) * filter.PageSize
	err := query.Offset(offset).Limit(filter.PageSize).
		Order("employee_id ASC").
		Find(&employeeEntities).Error
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	employees, err := mapper.EmployeeEntitiesToModels(employeeEntities)
	if err != nil {
		return nil, err
	}

	return &model.EmployeeListResult{
		Items:    employees,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查员工是否存在
func (r *EmployeeRepository) Exists(ctx context.Context, orgID, employeeID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.EmployeeEntity{}).
		Where("org_id = ? AND id = ?", orgID, employeeID).
		Count(&count).Error
	return count > 0, err
}

// BatchGet 批量获取员工
func (r *EmployeeRepository) BatchGet(ctx context.Context, orgID string, employeeIDs []string) ([]*model.Employee, error) {
	var employeeEntities []*entity.EmployeeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id IN ?", orgID, employeeIDs).
		Order("employee_id ASC").
		Find(&employeeEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.EmployeeEntitiesToModels(employeeEntities)
}

// UpdateStatus 更新员工状态
func (r *EmployeeRepository) UpdateStatus(ctx context.Context, orgID, employeeID string, status model.EmployeeStatus) error {
	return r.db.WithContext(ctx).Model(&entity.EmployeeEntity{}).
		Where("org_id = ? AND id = ?", orgID, employeeID).
		Update("status", status).Error
}
