package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IFixedAssignmentService 固定人员配置服务接口
type IFixedAssignmentService interface {
	// ListFixedAssignmentsByShift 获取班次的所有固定人员配置
	ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error)

	// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
	CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error)

	// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
	CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)
}

