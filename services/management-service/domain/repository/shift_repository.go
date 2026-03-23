package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
	"time"
)

// IShiftRepository 班次仓储接口
type IShiftRepository interface {
	// Create 创建班次
	Create(ctx context.Context, shift *model.Shift) error

	// Update 更新班次信息
	Update(ctx context.Context, shift *model.Shift) error

	// Delete 删除班次（软删除）
	Delete(ctx context.Context, orgID, shiftID string) error

	// GetByID 根据ID获取班次
	GetByID(ctx context.Context, orgID, shiftID string) (*model.Shift, error)

	// GetByCode 根据编码获取班次
	GetByCode(ctx context.Context, orgID, code string) (*model.Shift, error)

	// List 查询班次列表
	List(ctx context.Context, filter *model.ShiftFilter) (*model.ShiftListResult, error)

	// Exists 检查班次是否存在
	Exists(ctx context.Context, orgID, shiftID string) (bool, error)

	// BatchGet 批量获取班次
	BatchGet(ctx context.Context, orgID string, shiftIDs []string) ([]*model.Shift, error)

	// GetActiveShifts 获取所有启用的班次
	GetActiveShifts(ctx context.Context, orgID string) ([]*model.Shift, error)
}

// IShiftAssignmentRepository 班次分配仓储接口
type IShiftAssignmentRepository interface {
	// Create 创建班次分配
	Create(ctx context.Context, assignment *model.ShiftAssignment) error

	// Update 更新班次分配
	Update(ctx context.Context, assignment *model.ShiftAssignment) error

	// Delete 删除班次分配
	Delete(ctx context.Context, orgID, assignmentID string) error

	// GetByID 根据ID获取班次分配
	GetByID(ctx context.Context, orgID, assignmentID string) (*model.ShiftAssignment, error)

	// GetByEmployeeAndDate 获取员工在指定日期的班次分配
	GetByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) (*model.ShiftAssignment, error)

	// ListByEmployee 查询员工的班次分配列表
	ListByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// ListByDateRange 查询日期范围内的班次分配
	ListByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// BatchCreate 批量创建班次分配
	BatchCreate(ctx context.Context, assignments []*model.ShiftAssignment) error

	// UpdateStatus 更新班次分配状态
	UpdateStatus(ctx context.Context, orgID, assignmentID, status string) error
}

// IShiftGroupRepository 班次-分组关联仓储接口
type IShiftGroupRepository interface {
	// AddGroupToShift 为班次添加关联分组
	AddGroupToShift(ctx context.Context, shiftGroup *model.ShiftGroup) error

	// RemoveGroupFromShift 从班次移除关联分组
	RemoveGroupFromShift(ctx context.Context, shiftID, groupID string) error

	// GetShiftGroups 获取班次关联的所有分组
	GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error)

	// GetGroupShifts 获取分组关联的所有班次
	GetGroupShifts(ctx context.Context, groupID string) ([]*model.ShiftGroup, error)

	// BatchSetShiftGroups 批量设置班次关联的分组（先删除旧的，再添加新的）
	BatchSetShiftGroups(ctx context.Context, shiftID string, groupIDs []string) error

	// UpdateShiftGroup 更新班次-分组关联信息
	UpdateShiftGroup(ctx context.Context, shiftGroup *model.ShiftGroup) error

	// ExistsShiftGroup 检查班次-分组关联是否存在
	ExistsShiftGroup(ctx context.Context, shiftID, groupID string) (bool, error)

	// GetShiftGroupMembers 获取班次关联的所有分组的成员（去重）
	GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error)
}
