package service

import (
	"context"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type fixedAssignmentService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newFixedAssignmentService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IFixedAssignmentService {
	return &fixedAssignmentService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *fixedAssignmentService) ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error) {
	return s.managementRepo.ListFixedAssignmentsByShift(ctx, shiftID)
}

func (s *fixedAssignmentService) CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error) {
	return s.managementRepo.CalculateFixedSchedule(ctx, shiftID, startDate, endDate)
}

func (s *fixedAssignmentService) CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error) {
	return s.managementRepo.CalculateMultipleFixedSchedules(ctx, shiftIDs, startDate, endDate)
}

