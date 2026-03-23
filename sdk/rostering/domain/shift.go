package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// IShiftService 班次管理接口
type IShiftService interface {
	// CreateShift 创建班次
	CreateShift(ctx context.Context, req *model.CreateShiftRequest) (string, error)

	// UpdateShift 更新班次信息
	UpdateShift(ctx context.Context, id string, req *model.UpdateShiftRequest) error

	// ListShifts 获取班次列表
	ListShifts(ctx context.Context, req *model.ListShiftsRequest) (*model.Page[*model.Shift], error)

	// SetShiftGroups 设置班次关联的分组
	SetShiftGroups(ctx context.Context, req *model.SetShiftGroupsRequest) error

	// AddShiftGroup 添加班次分组
	AddShiftGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error

	// RemoveShiftGroup 移除班次分组
	RemoveShiftGroup(ctx context.Context, shiftID, groupID string) error

	// GetShiftGroups 获取班次分组
	GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error)

	// GetShiftGroupMembers 获取班次分组成员
	GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error)

	// GetWeeklyStaffConfig 获取班次周人数配置
	GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error)

	// SetWeeklyStaffConfig 设置班次周人数配置
	SetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string, config []model.WeekdayStaffConfig) error

	// CalculateStaffing 计算班次推荐人数
	CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error)
}
