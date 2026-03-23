package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// ISchedulingService 排班服务接口
type ISchedulingService interface {
	BatchAssign(ctx context.Context, req *model.BatchAssignRequest) error
	GetByDateRange(ctx context.Context, req *model.GetScheduleByDateRangeRequest) (*model.ScheduleResponse, error)
	GetSummary(ctx context.Context, req *model.GetScheduleSummaryRequest) (*model.ScheduleSummaryResponse, error)
	Delete(ctx context.Context, id string) error
}
