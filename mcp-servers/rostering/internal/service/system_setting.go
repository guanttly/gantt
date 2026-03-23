package service

import (
	"context"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type systemSettingService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newSystemSettingService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.ISystemSettingService {
	return &systemSettingService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *systemSettingService) GetSetting(ctx context.Context, orgID, key string) (string, error) {
	return s.managementRepo.GetSystemSetting(ctx, orgID, key)
}

func (s *systemSettingService) SetSetting(ctx context.Context, orgID, key, value string) error {
	return s.managementRepo.SetSystemSetting(ctx, orgID, key, value)
}

