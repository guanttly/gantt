package repository

import (
	"context"
	"fmt"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// DepartmentRepository 部门仓储实现
type DepartmentRepository struct {
	db *gorm.DB
}

// NewDepartmentRepository 创建部门仓储实例
func NewDepartmentRepository(db *gorm.DB) repository.IDepartmentRepository {
	return &DepartmentRepository{db: db}
}

// Create 创建部门
func (r *DepartmentRepository) Create(ctx context.Context, department *model.Department) error {
	departmentEntity := mapper.DepartmentModelToEntity(department)
	return r.db.WithContext(ctx).Create(departmentEntity).Error
}

// Update 更新部门信息
func (r *DepartmentRepository) Update(ctx context.Context, department *model.Department) error {
	departmentEntity := mapper.DepartmentModelToEntity(department)
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", department.OrgID, department.ID).
		Updates(departmentEntity).Error
}

// Delete 删除部门（软删除）
func (r *DepartmentRepository) Delete(ctx context.Context, orgID, departmentID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, departmentID).
		Delete(&entity.DepartmentEntity{}).Error
}

// GetByID 根据ID获取部门
func (r *DepartmentRepository) GetByID(ctx context.Context, orgID, departmentID string) (*model.Department, error) {
	var departmentEntity entity.DepartmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, departmentID).
		First(&departmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntityToModel(&departmentEntity), nil
}

// GetByCode 根据编码获取部门
func (r *DepartmentRepository) GetByCode(ctx context.Context, orgID, code string) (*model.Department, error) {
	var departmentEntity entity.DepartmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&departmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntityToModel(&departmentEntity), nil
}

// List 查询部门列表
func (r *DepartmentRepository) List(ctx context.Context, filter *model.DepartmentFilter) (*model.DepartmentListResult, error) {
	query := r.db.WithContext(ctx).Model(&entity.DepartmentEntity{}).
		Where("org_id = ?", filter.OrgID)

	// ParentID过滤
	if filter.ParentID != nil {
		if *filter.ParentID == "" {
			// 查询顶级部门
			query = query.Where("parent_id IS NULL OR parent_id = ''")
		} else {
			query = query.Where("parent_id = ?", *filter.ParentID)
		}
	}

	// 关键词搜索
	if filter.Keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?",
			"%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	// 启用状态过滤
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// 排序
	query = query.Order("level ASC, sort_order ASC, created_at DESC")

	var departmentEntities []*entity.DepartmentEntity
	if err := query.Find(&departmentEntities).Error; err != nil {
		return nil, err
	}

	return &model.DepartmentListResult{
		Items:    mapper.DepartmentEntitiesToModels(departmentEntities),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查部门是否存在
func (r *DepartmentRepository) Exists(ctx context.Context, orgID, departmentID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.DepartmentEntity{}).
		Where("org_id = ? AND id = ?", orgID, departmentID).
		Count(&count).Error
	return count > 0, err
}

// GetChildren 获取直接子部门
func (r *DepartmentRepository) GetChildren(ctx context.Context, orgID, parentID string) ([]*model.Department, error) {
	var departmentEntities []*entity.DepartmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND parent_id = ?", orgID, parentID).
		Order("sort_order ASC, created_at DESC").
		Find(&departmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntitiesToModels(departmentEntities), nil
}

// GetDescendants 获取所有后代部门
func (r *DepartmentRepository) GetDescendants(ctx context.Context, orgID, departmentID string) ([]*model.Department, error) {
	// 先获取父部门以获取其路径
	parent, err := r.GetByID(ctx, orgID, departmentID)
	if err != nil {
		return nil, err
	}

	var departmentEntities []*entity.DepartmentEntity
	err = r.db.WithContext(ctx).
		Where("org_id = ? AND path LIKE ?", orgID, parent.Path+"/%").
		Order("level ASC, sort_order ASC, created_at DESC").
		Find(&departmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntitiesToModels(departmentEntities), nil
}

// GetTreeByOrgID 获取组织的完整部门树
func (r *DepartmentRepository) GetTreeByOrgID(ctx context.Context, orgID string) ([]*model.Department, error) {
	var departmentEntities []*entity.DepartmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("level ASC, sort_order ASC, created_at DESC").
		Find(&departmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntitiesToModels(departmentEntities), nil
}

// GetActiveDepartments 获取所有启用的部门
func (r *DepartmentRepository) GetActiveDepartments(ctx context.Context, orgID string) ([]*model.Department, error) {
	var departmentEntities []*entity.DepartmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("level ASC, sort_order ASC, created_at DESC").
		Find(&departmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.DepartmentEntitiesToModels(departmentEntities), nil
}

// UpdateSortOrder 更新排序
func (r *DepartmentRepository) UpdateSortOrder(ctx context.Context, orgID, departmentID string, sortOrder int) error {
	return r.db.WithContext(ctx).
		Model(&entity.DepartmentEntity{}).
		Where("org_id = ? AND id = ?", orgID, departmentID).
		Update("sort_order", sortOrder).Error
}

// CountEmployees 统计部门员工数量
func (r *DepartmentRepository) CountEmployees(ctx context.Context, orgID, departmentID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("employees").
		Where("org_id = ? AND department_id = ? AND deleted_at IS NULL", orgID, departmentID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count employees: %w", err)
	}
	return int(count), nil
}
