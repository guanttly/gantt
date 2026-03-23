package service

import (
	"context"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type shiftService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newShiftService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IShiftService {
	return &shiftService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *shiftService) Create(ctx context.Context, req *model.CreateShiftRequest) (*model.Shift, error) {
	return s.managementRepo.CreateShift(ctx, req)
}

func (s *shiftService) GetList(ctx context.Context, req *model.ListShiftsRequest) (*model.ListShiftsResponse, error) {
	pageData, err := s.managementRepo.ListShifts(ctx, req)
	if err != nil {
		return nil, err
	}

	return &model.ListShiftsResponse{
		Shifts:     pageData.Items,
		TotalCount: int(pageData.Total),
	}, nil
}

func (s *shiftService) Get(ctx context.Context, id string) (*model.Shift, error) {
	return s.managementRepo.GetShift(ctx, id)
}

func (s *shiftService) Update(ctx context.Context, id string, req *model.UpdateShiftRequest) (*model.Shift, error) {
	return s.managementRepo.UpdateShift(ctx, id, req)
}

func (s *shiftService) Delete(ctx context.Context, id string) error {
	return s.managementRepo.DeleteShift(ctx, id)
}

func (s *shiftService) SetGroups(ctx context.Context, shiftID string, req *model.SetShiftGroupsRequest) error {
	return s.managementRepo.SetShiftGroups(ctx, shiftID, req)
}

func (s *shiftService) AddGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error {
	return s.managementRepo.AddShiftGroup(ctx, shiftID, req)
}

func (s *shiftService) RemoveGroup(ctx context.Context, shiftID string, groupID string) error {
	return s.managementRepo.RemoveShiftGroup(ctx, shiftID, groupID)
}

func (s *shiftService) GetGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error) {
	return s.managementRepo.GetShiftGroups(ctx, shiftID)
}

func (s *shiftService) GetGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error) {
	return s.managementRepo.GetShiftGroupMembers(ctx, shiftID)
}

func (s *shiftService) ToggleStatus(ctx context.Context, id string, status string) error {
	return s.managementRepo.ToggleShiftStatus(ctx, id, status)
}

func (s *shiftService) GetWeeklyStaff(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error) {
	return s.managementRepo.GetShiftWeeklyStaff(ctx, orgID, shiftID)
}

func (s *shiftService) SetWeeklyStaff(ctx context.Context, orgID, shiftID string, req *model.SetShiftWeeklyStaffRequest) error {
	return s.managementRepo.SetShiftWeeklyStaff(ctx, orgID, shiftID, req)
}

func (s *shiftService) CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error) {
	return s.managementRepo.CalculateStaffing(ctx, orgID, shiftID)
}
