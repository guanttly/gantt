package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
	"time"
)

// ILeaveRepository 假期记录仓储接口
// 记录已成事实的请假情况，不包含审批流
type ILeaveRepository interface {
	// Create 创建假期记录
	Create(ctx context.Context, leave *model.LeaveRecord) error

	// Update 更新假期记录
	Update(ctx context.Context, leave *model.LeaveRecord) error

	// Delete 删除假期记录（软删除）
	Delete(ctx context.Context, orgID, leaveID string) error

	// GetByID 根据ID获取假期记录
	GetByID(ctx context.Context, orgID, leaveID string) (*model.LeaveRecord, error)

	// List 查询假期记录列表
	List(ctx context.Context, filter *model.LeaveFilter) (*model.LeaveListResult, error)

	// ListByEmployee 查询员工的假期记录
	ListByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate *time.Time) ([]*model.LeaveRecord, error)

	// CheckConflict 检查假期是否与现有假期冲突
	CheckConflict(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time, excludeID *string) (bool, error)

	// GetActiveLeaves 获取员工当前有效的假期
	GetActiveLeaves(ctx context.Context, orgID, employeeID string) ([]*model.LeaveRecord, error)
}

// ILeaveBalanceRepository 假期余额仓储接口
type ILeaveBalanceRepository interface {
	// Create 创建假期余额
	Create(ctx context.Context, balance *model.LeaveBalance) error

	// Update 更新假期余额
	Update(ctx context.Context, balance *model.LeaveBalance) error

	// GetByEmployeeAndType 获取员工某类型的假期余额
	GetByEmployeeAndType(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int) (*model.LeaveBalance, error)

	// ListByEmployee 获取员工的所有假期余额
	ListByEmployee(ctx context.Context, orgID, employeeID string, year int) ([]*model.LeaveBalance, error)

	// DeductBalance 扣减假期余额
	DeductBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, days float64) error

	// AddBalance 增加假期余额
	AddBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, days float64) error

	// InitializeBalance 初始化员工的假期余额
	InitializeBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, totalDays float64) error
}
