package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IDepartmentService 部门服务接口
type IDepartmentService interface {
	Create(ctx context.Context, req *model.CreateDepartmentRequest) (*model.Department, error)
	GetList(ctx context.Context, orgID string) (*model.ListDepartmentsResponse, error)
	Get(ctx context.Context, id string) (*model.Department, error)
	Update(ctx context.Context, id string, req *model.UpdateDepartmentRequest) (*model.Department, error)
	Delete(ctx context.Context, id string) error
	GetTree(ctx context.Context, orgID string) ([]*model.DepartmentTreeNode, error)
}
