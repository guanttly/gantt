package service

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// ILeaveService 请假管理领域服务接口
// 记录已成事实的请假情况，不包含审批流
type ILeaveService interface {
	// CreateLeave 创建假期记录
	CreateLeave(ctx context.Context, leave *model.LeaveRecord) error

	// UpdateLeave 更新假期记录
	UpdateLeave(ctx context.Context, leave *model.LeaveRecord) error

	// DeleteLeave 删除假期记录
	DeleteLeave(ctx context.Context, orgID, leaveID string) error

	// GetLeave 获取假期详情
	GetLeave(ctx context.Context, orgID, leaveID string) (*model.LeaveRecord, error)

	// ListLeaves 查询假期记录列表
	ListLeaves(ctx context.Context, filter *model.LeaveFilter) (*model.LeaveListResult, error)

	// GetEmployeeLeaves 获取员工的假期记录
	GetEmployeeLeaves(ctx context.Context, orgID, employeeID string, startDate, endDate *time.Time) ([]*model.LeaveRecord, error)

	// GetLeaveBalance 获取假期余额
	GetLeaveBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int) (*model.LeaveBalance, error)

	// GetEmployeeLeaveBalance 获取员工所有类型的假期余额
	GetEmployeeLeaveBalance(ctx context.Context, orgID, employeeID string, year int) ([]*model.LeaveBalance, error)

	// InitializeLeaveBalance 初始化假期余额
	InitializeLeaveBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, totalDays float64) error

	// CalculateLeaveDays 计算实际请假天数（考虑工作日、节假日、小时级请假）
	// startTime/endTime 格式: HH:MM，用于小时级请假
	CalculateLeaveDays(ctx context.Context, orgID string, startDate, endDate time.Time, startTime, endTime *string) (float64, error)
}
