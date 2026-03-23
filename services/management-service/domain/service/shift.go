package service

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// IShiftService 班次管理领域服务接口
type IShiftService interface {
	// CreateShift 创建班次
	CreateShift(ctx context.Context, shift *model.Shift) error

	// UpdateShift 更新班次信息
	UpdateShift(ctx context.Context, shift *model.Shift) error

	// DeleteShift 删除班次
	DeleteShift(ctx context.Context, orgID, shiftID string) error

	// GetShift 获取班次详情
	GetShift(ctx context.Context, orgID, shiftID string) (*model.Shift, error)

	// ListShifts 查询班次列表
	ListShifts(ctx context.Context, filter *model.ShiftFilter) (*model.ShiftListResult, error)

	// GetActiveShifts 获取所有启用的班次
	GetActiveShifts(ctx context.Context, orgID string) ([]*model.Shift, error)

	// AssignShift 分配班次给员工
	AssignShift(ctx context.Context, assignment *model.ShiftAssignment) error

	// GetEmployeeShifts 获取员工的班次安排
	GetEmployeeShifts(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// GetShiftAssignments 获取日期范围内的班次分配
	GetShiftAssignments(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// AddGroupToShift 为班次添加关联分组
	AddGroupToShift(ctx context.Context, shiftID, groupID string, priority int) error

	// RemoveGroupFromShift 从班次移除关联分组
	RemoveGroupFromShift(ctx context.Context, shiftID, groupID string) error

	// SetShiftGroups 批量设置班次的关联分组
	SetShiftGroups(ctx context.Context, shiftID string, groupIDs []string) error

	// GetShiftGroups 获取班次关联的所有分组
	GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error)

	// GetGroupShifts 获取分组关联的所有班次
	GetGroupShifts(ctx context.Context, groupID string) ([]*model.ShiftGroup, error)

	// GetShiftGroupMembers 获取班次关联的所有分组的成员（去重）
	GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error)

	// ToggleShiftStatus 切换班次启用/禁用状态
	ToggleShiftStatus(ctx context.Context, orgID, shiftID string, isActive bool) error
}
