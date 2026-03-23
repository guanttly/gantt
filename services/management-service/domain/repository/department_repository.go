package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IDepartmentRepository 部门仓储接口
type IDepartmentRepository interface {
	// Create 创建部门
	Create(ctx context.Context, department *model.Department) error

	// Update 更新部门信息
	Update(ctx context.Context, department *model.Department) error

	// Delete 删除部门（软删除）
	Delete(ctx context.Context, orgID, departmentID string) error

	// GetByID 根据ID获取部门
	GetByID(ctx context.Context, orgID, departmentID string) (*model.Department, error)

	// GetByCode 根据编码获取部门
	GetByCode(ctx context.Context, orgID, code string) (*model.Department, error)

	// List 查询部门列表
	List(ctx context.Context, filter *model.DepartmentFilter) (*model.DepartmentListResult, error)

	// Exists 检查部门是否存在
	Exists(ctx context.Context, orgID, departmentID string) (bool, error)

	// GetChildren 获取直接子部门
	GetChildren(ctx context.Context, orgID, parentID string) ([]*model.Department, error)

	// GetDescendants 获取所有后代部门（包括子部门的子部门）
	GetDescendants(ctx context.Context, orgID, departmentID string) ([]*model.Department, error)

	// GetTreeByOrgID 获取组织的完整部门树
	GetTreeByOrgID(ctx context.Context, orgID string) ([]*model.Department, error)

	// GetActiveDepartments 获取所有启用的部门
	GetActiveDepartments(ctx context.Context, orgID string) ([]*model.Department, error)

	// UpdateSortOrder 更新排序
	UpdateSortOrder(ctx context.Context, orgID, departmentID string, sortOrder int) error

	// CountEmployees 统计部门员工数量
	CountEmployees(ctx context.Context, orgID, departmentID string) (int, error)
}
