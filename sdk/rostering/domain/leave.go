package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// ILeaveService 请假管理接口
type ILeaveService interface {
	// CreateLeave 创建请假记录
	CreateLeave(ctx context.Context, req model.CreateLeaveRequest) (string, error)

	// UpdateLeave 更新请假记录
	UpdateLeave(ctx context.Context, orgID, leaveID string, req model.UpdateLeaveRequest) error

	// ListLeaves 获取请假记录列表
	ListLeaves(ctx context.Context, req model.ListLeavesRequest) (*model.ListLeavesResponse, error)

	// GetLeave 获取请假记录详情
	GetLeave(ctx context.Context, orgID, leaveID string) (*model.Leave, error)

	// DeleteLeave 删除请假记录
	DeleteLeave(ctx context.Context, orgID, leaveID string) error

	// GetLeaveBalance 获取员工假期余额
	GetLeaveBalance(ctx context.Context, orgID, employeeID, leaveType string, year int) (*model.LeaveBalance, error)
}
