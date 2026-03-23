package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IEmployeeRepository 员工仓储接口
type IEmployeeRepository interface {
	// Create 创建员工
	Create(ctx context.Context, employee *model.Employee) error

	// Update 更新员工信息
	Update(ctx context.Context, employee *model.Employee) error

	// Delete 删除员工（软删除）
	Delete(ctx context.Context, orgID, employeeID string) error

	// GetByID 根据ID获取员工
	GetByID(ctx context.Context, orgID, employeeID string) (*model.Employee, error)

	// GetByEmployeeID 根据工号获取员工
	GetByEmployeeID(ctx context.Context, orgID, employeeNo string) (*model.Employee, error)

	// GetByUserID 根据用户ID获取员工
	GetByUserID(ctx context.Context, orgID, userID string) (*model.Employee, error)

	// List 查询员工列表
	List(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error)

	// Exists 检查员工是否存在
	Exists(ctx context.Context, orgID, employeeID string) (bool, error)

	// BatchGet 批量获取员工
	BatchGet(ctx context.Context, orgID string, employeeIDs []string) ([]*model.Employee, error)

	// UpdateStatus 更新员工状态
	UpdateStatus(ctx context.Context, orgID, employeeID string, status model.EmployeeStatus) error
}
