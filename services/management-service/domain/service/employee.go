package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IEmployeeService 员工管理领域服务接口
type IEmployeeService interface {
	// CreateEmployee 创建员工
	CreateEmployee(ctx context.Context, employee *model.Employee) error

	// UpdateEmployee 更新员工信息
	UpdateEmployee(ctx context.Context, employee *model.Employee) error

	// DeleteEmployee 删除员工
	DeleteEmployee(ctx context.Context, orgID, employeeID string) error

	// GetEmployee 获取员工详情
	GetEmployee(ctx context.Context, orgID, employeeID string) (*model.Employee, error)

	// ListEmployees 查询员工列表
	ListEmployees(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error)

	// ListSimpleEmployees 查询简单员工列表（不查询分组信息）
	ListSimpleEmployees(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error)

	// UpdateEmployeeStatus 更新员工状态
	UpdateEmployeeStatus(ctx context.Context, orgID, employeeID string, status model.EmployeeStatus) error

	// BatchGetEmployees 批量获取员工
	BatchGetEmployees(ctx context.Context, orgID string, employeeIDs []string) ([]*model.Employee, error)
}
