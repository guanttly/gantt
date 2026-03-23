package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/logging"

	domain_service "jusha/gantt/service/management/domain/service"
)

// SystemSettingServiceImpl 系统设置服务实现
type SystemSettingServiceImpl struct {
	repo   repository.ISystemSettingRepository
	logger logging.ILogger
}

// NewSystemSettingService 创建系统设置服务
func NewSystemSettingService(repo repository.ISystemSettingRepository, logger logging.ILogger) domain_service.ISystemSettingService {
	return &SystemSettingServiceImpl{
		repo:   repo,
		logger: logger.With("service", "SystemSettingService"),
	}
}

// GetSetting 获取系统设置（如果不存在返回默认值）
func (s *SystemSettingServiceImpl) GetSetting(ctx context.Context, orgID, key string) (string, error) {
	setting, err := s.repo.GetByKey(ctx, orgID, key)
	if err != nil {
		// 数据库错误时，尝试返回默认值（更健壮）
		s.logger.Warn("Failed to get setting from database, trying default value", "orgID", orgID, "key", key, "error", err)
		defaultValue := model.GetSystemSettingDefaultValue(key)
		if defaultValue != "" {
			s.logger.Info("Using default value for setting", "key", key, "defaultValue", defaultValue)
			return defaultValue, nil
		}
		// 如果没有默认值，返回错误
		return "", fmt.Errorf("get setting: %w", err)
	}

	// 如果设置不存在，返回默认值
	if setting == nil {
		defaultValue := model.GetSystemSettingDefaultValue(key)
		if defaultValue == "" {
			return "", fmt.Errorf("setting key '%s' not found and no default value", key)
		}
		s.logger.Debug("Setting not found, using default value", "key", key, "defaultValue", defaultValue)
		return defaultValue, nil
	}

	return setting.Value, nil
}

// SetSetting 设置系统设置
func (s *SystemSettingServiceImpl) SetSetting(ctx context.Context, orgID, key, value, description string) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	// 先检查是否已存在（为了保留现有的 description）
	existing, err := s.repo.GetByKey(ctx, orgID, key)
	if err != nil && err.Error() != "" {
		// 如果查询出错但不是记录不存在（比如表不存在），记录警告但继续
		// repository.Set 会处理创建或更新
		s.logger.Warn("Failed to check existing setting, repository will handle upsert", "error", err, "key", key)
	}

	setting := &model.SystemSetting{
		OrgID:       orgID,
		Key:         key,
		Value:       value,
		Description: description,
	}

	if existing == nil {
		// 创建新设置
		setting.ID = uuid.New().String()
		// CreatedAt 和 UpdatedAt 由 GORM 的 autoCreateTime 和 autoUpdateTime 自动设置
	} else {
		// 更新现有设置，保留原有的 description（如果新 description 为空）
		setting.ID = existing.ID
		if description == "" {
			setting.Description = existing.Description
		}
		// CreatedAt 保持不变，UpdatedAt 由 GORM 的 autoUpdateTime 自动更新
	}

	return s.repo.Set(ctx, setting)
}

// GetAllSettings 获取组织的所有设置
func (s *SystemSettingServiceImpl) GetAllSettings(ctx context.Context, orgID string) ([]*model.SystemSetting, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	settings, err := s.repo.GetAll(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	return settings, nil
}

// DeleteSetting 删除系统设置
func (s *SystemSettingServiceImpl) DeleteSetting(ctx context.Context, orgID, key string) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	return s.repo.Delete(ctx, orgID, key)
}
