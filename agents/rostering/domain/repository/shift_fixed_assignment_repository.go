package repository

import (
	"context"
	"jusha/agent/rostering/domain/model"
)

// ShiftFixedAssignmentRepository 班次固定人员配置仓储接口
type ShiftFixedAssignmentRepository interface {
	// Create 创建固定人员配置
	Create(ctx context.Context, assignment *model.ShiftFixedAssignment) error

	// BatchCreate 批量创建固定人员配置
	BatchCreate(ctx context.Context, assignments []*model.ShiftFixedAssignment) error

	// Update 更新固定人员配置
	Update(ctx context.Context, id string, assignment *model.ShiftFixedAssignment) error

	// Delete 软删除固定人员配置
	Delete(ctx context.Context, id string) error

	// GetByID 根据ID获取固定人员配置
	GetByID(ctx context.Context, id string) (*model.ShiftFixedAssignment, error)

	// ListByShiftID 获取班次的所有固定人员配置
	ListByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error)

	// ListByShiftIDs 批量获取多个班次的固定人员配置
	ListByShiftIDs(ctx context.Context, shiftIDs []string) (map[string][]*model.ShiftFixedAssignment, error)

	// ListByStaffID 获取人员的所有固定班次配置
	ListByStaffID(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error)

	// List 查询固定人员配置列表
	List(ctx context.Context, req *model.ListShiftFixedAssignmentsRequest) ([]*model.ShiftFixedAssignment, error)

	// DeleteByShiftID 删除班次的所有固定人员配置（级联删除）
	DeleteByShiftID(ctx context.Context, shiftID string) error

	// DeleteByShiftAndStaff 删除指定班次和人员的配置
	DeleteByShiftAndStaff(ctx context.Context, shiftID, staffID string) error
}

