package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// ILeaveService 请假服务接口
type ILeaveService interface {
	Create(ctx context.Context, req *model.CreateLeaveRequest) (*model.Leave, error)
	GetList(ctx context.Context, req *model.ListLeavesRequest) (*model.ListLeavesResponse, error)
	Get(ctx context.Context, id string) (*model.Leave, error)
	Update(ctx context.Context, id string, req *model.UpdateLeaveRequest) (*model.Leave, error)
	Delete(ctx context.Context, id string) error
	GetBalance(ctx context.Context, employeeID string) ([]*model.LeaveBalance, error)
}
