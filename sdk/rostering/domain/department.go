package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// IDepartmentService 部门管理接口
type IDepartmentService interface {
	// CreateDepartment 创建部门
	CreateDepartment(ctx context.Context, req *model.CreateDepartmentRequest) (string, error)

	// UpdateDepartment 更新部门
	UpdateDepartment(ctx context.Context, id string, req *model.UpdateDepartmentRequest) error

	// ListDepartments 获取部门列表
	ListDepartments(ctx context.Context, orgID string, page, pageSize int) (*model.ListDepartmentsResponse, error)
}
