package service

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IDepartmentService 部门管理领域服务接口
type IDepartmentService interface {
	// CreateDepartment 创建部门
	CreateDepartment(ctx context.Context, department *model.Department) error

	// UpdateDepartment 更新部门信息
	UpdateDepartment(ctx context.Context, department *model.Department) error

	// DeleteDepartment 删除部门
	DeleteDepartment(ctx context.Context, orgID, departmentID string) error

	// GetDepartment 获取部门详情
	GetDepartment(ctx context.Context, orgID, departmentID string) (*model.Department, error)

	// ListDepartments 查询部门列表
	ListDepartments(ctx context.Context, filter *model.DepartmentFilter) (*model.DepartmentListResult, error)

	// GetDepartmentTree 获取部门树
	GetDepartmentTree(ctx context.Context, orgID string) ([]*model.DepartmentTree, error)

	// GetActiveDepartments 获取所有启用的部门
	GetActiveDepartments(ctx context.Context, orgID string) ([]*model.Department, error)

	// UpdateSortOrder 更新排序
	UpdateSortOrder(ctx context.Context, orgID, departmentID string, sortOrder int) error

	// MoveDepartment 移动部门到新的父部门下
	MoveDepartment(ctx context.Context, orgID, departmentID string, newParentID *string) error
}
