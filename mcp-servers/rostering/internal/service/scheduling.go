package service

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type schedulingService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newSchedulingService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.ISchedulingService {
	return &schedulingService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *schedulingService) BatchAssign(ctx context.Context, req *model.BatchAssignRequest) error {
	err, _ := s.managementRepo.DoRequest(ctx, "POST", "/scheduling/assignments/batch", req)
	return err
}

func (s *schedulingService) GetByDateRange(ctx context.Context, req *model.GetScheduleByDateRangeRequest) (*model.ScheduleResponse, error) {
	path := fmt.Sprintf("/scheduling/assignments?startDate=%s&endDate=%s&orgId=%s",
		req.StartDate, req.EndDate, req.OrgID)

	err, resp := s.managementRepo.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	return &model.ScheduleResponse{
		Schedules: resp.Data,
	}, nil
}

func (s *schedulingService) GetSummary(ctx context.Context, req *model.GetScheduleSummaryRequest) (*model.ScheduleSummaryResponse, error) {
	path := fmt.Sprintf("/scheduling/summary?startDate=%s&endDate=%s&orgId=%s",
		req.StartDate, req.EndDate, req.OrgID)

	err, resp := s.managementRepo.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	return &model.ScheduleSummaryResponse{
		Summary: resp.Data,
	}, nil
}

func (s *schedulingService) Delete(ctx context.Context, id string) error {
	err, _ := s.managementRepo.DoRequest(ctx, "DELETE", fmt.Sprintf("/scheduling/assignments/%s", id), nil)
	return err
}
