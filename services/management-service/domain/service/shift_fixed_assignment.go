package service

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IShiftFixedAssignmentService 班次固定人员配置领域服务接口
type IShiftFixedAssignmentService interface {
	// Create 创建固定人员配置
	Create(ctx context.Context, assignment *model.ShiftFixedAssignment) error

	// BatchCreate 批量创建固定人员配置
	BatchCreate(ctx context.Context, shiftID string, assignments []*model.ShiftFixedAssignment) error

	// Update 更新固定人员配置
	Update(ctx context.Context, id string, assignment *model.ShiftFixedAssignment) error

	// Delete 删除固定人员配置
	Delete(ctx context.Context, id string) error

	// GetByID 获取固定人员配置详情
	GetByID(ctx context.Context, id string) (*model.ShiftFixedAssignment, error)

	// ListByShiftID 获取班次的所有固定人员配置
	ListByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error)

	// ListByStaffID 获取人员的所有固定班次配置
	ListByStaffID(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error)

	// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
	CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error)

	// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
	CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)
}

