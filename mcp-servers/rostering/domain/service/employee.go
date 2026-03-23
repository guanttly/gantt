package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IEmployeeService 员工服务接口
type IEmployeeService interface {
	Create(ctx context.Context, req *model.CreateEmployeeRequest) (*model.Employee, error)
	GetList(ctx context.Context, req *model.ListEmployeesRequest) (*model.ListEmployeesResponse, error)
	Get(ctx context.Context, id string) (*model.Employee, error)
	Update(ctx context.Context, id string, req *model.UpdateEmployeeRequest) (*model.Employee, error)
	Delete(ctx context.Context, id string) error
}
