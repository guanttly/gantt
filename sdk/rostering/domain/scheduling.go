package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// ISchedulingService 排班管理接口
type ISchedulingService interface {
	// BatchAssignSchedule 批量分配排班
	BatchAssignSchedule(ctx context.Context, req model.BatchAssignRequest) error

	// GetScheduleByDateRange 获取指定日期范围的排班
	GetScheduleByDateRange(ctx context.Context, req model.GetScheduleByDateRangeRequest) (*model.ScheduleResponse, error)

	// GetScheduleSummary 获取排班汇总信息
	GetScheduleSummary(ctx context.Context, req model.GetScheduleSummaryRequest) (*model.ScheduleSummaryResponse, error)

	// DeleteSchedule 删除排班记录
	DeleteSchedule(ctx context.Context, orgID, employeeID, date string) error
}
