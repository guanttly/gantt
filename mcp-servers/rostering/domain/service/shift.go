package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IShiftService 班次服务接口
type IShiftService interface {
	Create(ctx context.Context, req *model.CreateShiftRequest) (*model.Shift, error)
	GetList(ctx context.Context, req *model.ListShiftsRequest) (*model.ListShiftsResponse, error)
	Get(ctx context.Context, id string) (*model.Shift, error)
	Update(ctx context.Context, id string, req *model.UpdateShiftRequest) (*model.Shift, error)
	Delete(ctx context.Context, id string) error
	SetGroups(ctx context.Context, shiftID string, req *model.SetShiftGroupsRequest) error
	AddGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error
	RemoveGroup(ctx context.Context, shiftID string, groupID string) error
	GetGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error)
	GetGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error)
	ToggleStatus(ctx context.Context, id string, status string) error
	GetWeeklyStaff(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error)
	SetWeeklyStaff(ctx context.Context, orgID, shiftID string, req *model.SetShiftWeeklyStaffRequest) error
	CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error)
}
