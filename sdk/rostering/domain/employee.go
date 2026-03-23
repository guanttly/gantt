package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// IEmployeeService 员工管理接口
type IEmployeeService interface {
	// CreateEmployee 创建员工
	CreateEmployee(ctx context.Context, req *model.CreateEmployeeRequest) (string, error)

	// UpdateEmployee 更新员工信息
	UpdateEmployee(ctx context.Context, id string, req *model.UpdateEmployeeRequest) error

	// ListEmployees 获取员工列表
	ListEmployees(ctx context.Context, req *model.ListEmployeesRequest) (*model.Page[*model.Employee], error)
}
