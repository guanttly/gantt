package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// TimePeriodServiceImpl 时间段服务实现
type TimePeriodServiceImpl struct {
	timePeriodRepo repository.ITimePeriodRepository
	logger         logging.ILogger
}

// NewTimePeriodService 创建时间段服务实例
func NewTimePeriodService(
	timePeriodRepo repository.ITimePeriodRepository,
	logger logging.ILogger,
) domain_service.ITimePeriodService {
	return &TimePeriodServiceImpl{
		timePeriodRepo: timePeriodRepo,
		logger:         logger,
	}
}

// CreateTimePeriod 创建时间段
func (s *TimePeriodServiceImpl) CreateTimePeriod(ctx context.Context, timePeriod *model.TimePeriod) error {
	// 生成ID
	if timePeriod.ID == "" {
		timePeriod.ID = uuid.New().String()
	}

	// 设置默认值
	timePeriod.IsActive = true
	timePeriod.CreatedAt = time.Now()
	timePeriod.UpdatedAt = time.Now()

	// 检查编码是否重复
	existing, err := s.timePeriodRepo.GetByCode(ctx, timePeriod.OrgID, timePeriod.Code)
	if err == nil && existing != nil {
		return fmt.Errorf("时间段编码 %s 已存在", timePeriod.Code)
	}

	if err := s.timePeriodRepo.Create(ctx, timePeriod); err != nil {
		s.logger.Error("Failed to create time period", "error", err)
		return fmt.Errorf("创建时间段失败: %w", err)
	}

	s.logger.Info("Time period created", "id", timePeriod.ID, "name", timePeriod.Name)
	return nil
}

// UpdateTimePeriod 更新时间段信息
func (s *TimePeriodServiceImpl) UpdateTimePeriod(ctx context.Context, timePeriod *model.TimePeriod) error {
	// 检查时间段是否存在
	existing, err := s.timePeriodRepo.GetByID(ctx, timePeriod.OrgID, timePeriod.ID)
	if err != nil {
		return fmt.Errorf("时间段不存在: %w", err)
	}

	// 如果修改了编码，检查新编码是否重复
	if timePeriod.Code != existing.Code {
		existingByCode, err := s.timePeriodRepo.GetByCode(ctx, timePeriod.OrgID, timePeriod.Code)
		if err == nil && existingByCode != nil && existingByCode.ID != timePeriod.ID {
			return fmt.Errorf("时间段编码 %s 已存在", timePeriod.Code)
		}
	}

	timePeriod.UpdatedAt = time.Now()

	if err := s.timePeriodRepo.Update(ctx, timePeriod); err != nil {
		s.logger.Error("Failed to update time period", "error", err)
		return fmt.Errorf("更新时间段失败: %w", err)
	}

	s.logger.Info("Time period updated", "id", timePeriod.ID, "name", timePeriod.Name)
	return nil
}

// DeleteTimePeriod 删除时间段
func (s *TimePeriodServiceImpl) DeleteTimePeriod(ctx context.Context, orgID, timePeriodID string) error {
	// 检查时间段是否存在
	_, err := s.timePeriodRepo.GetByID(ctx, orgID, timePeriodID)
	if err != nil {
		return fmt.Errorf("时间段不存在: %w", err)
	}

	// TODO: 检查是否有关联的检查量数据或计算规则

	if err := s.timePeriodRepo.Delete(ctx, orgID, timePeriodID); err != nil {
		s.logger.Error("Failed to delete time period", "error", err)
		return fmt.Errorf("删除时间段失败: %w", err)
	}

	s.logger.Info("Time period deleted", "id", timePeriodID)
	return nil
}

// GetTimePeriod 获取时间段详情
func (s *TimePeriodServiceImpl) GetTimePeriod(ctx context.Context, orgID, timePeriodID string) (*model.TimePeriod, error) {
	timePeriod, err := s.timePeriodRepo.GetByID(ctx, orgID, timePeriodID)
	if err != nil {
		return nil, fmt.Errorf("获取时间段失败: %w", err)
	}
	return timePeriod, nil
}

// GetTimePeriodByCode 根据编码获取时间段
func (s *TimePeriodServiceImpl) GetTimePeriodByCode(ctx context.Context, orgID, code string) (*model.TimePeriod, error) {
	timePeriod, err := s.timePeriodRepo.GetByCode(ctx, orgID, code)
	if err != nil {
		return nil, fmt.Errorf("获取时间段失败: %w", err)
	}
	return timePeriod, nil
}

// GetTimePeriodByName 根据名称获取时间段
func (s *TimePeriodServiceImpl) GetTimePeriodByName(ctx context.Context, orgID, name string) (*model.TimePeriod, error) {
	timePeriod, err := s.timePeriodRepo.GetByName(ctx, orgID, name)
	if err != nil {
		return nil, fmt.Errorf("获取时间段失败: %w", err)
	}
	return timePeriod, nil
}

// ListTimePeriods 查询时间段列表
func (s *TimePeriodServiceImpl) ListTimePeriods(ctx context.Context, filter *model.TimePeriodFilter) (*model.TimePeriodListResult, error) {
	// 设置默认分页
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := s.timePeriodRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list time periods", "error", err)
		return nil, fmt.Errorf("查询时间段列表失败: %w", err)
	}
	return result, nil
}

// GetActiveTimePeriods 获取所有启用的时间段
func (s *TimePeriodServiceImpl) GetActiveTimePeriods(ctx context.Context, orgID string) ([]*model.TimePeriod, error) {
	timePeriods, err := s.timePeriodRepo.GetActiveTimePeriods(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to get active time periods", "error", err)
		return nil, fmt.Errorf("获取启用时间段失败: %w", err)
	}
	return timePeriods, nil
}

// ToggleTimePeriodStatus 切换时间段启用/禁用状态
func (s *TimePeriodServiceImpl) ToggleTimePeriodStatus(ctx context.Context, orgID, timePeriodID string, isActive bool) error {
	timePeriod, err := s.timePeriodRepo.GetByID(ctx, orgID, timePeriodID)
	if err != nil {
		return fmt.Errorf("时间段不存在: %w", err)
	}

	timePeriod.IsActive = isActive
	timePeriod.UpdatedAt = time.Now()

	if err := s.timePeriodRepo.Update(ctx, timePeriod); err != nil {
		s.logger.Error("Failed to toggle time period status", "error", err)
		return fmt.Errorf("切换时间段状态失败: %w", err)
	}

	s.logger.Info("Time period status toggled", "id", timePeriodID, "isActive", isActive)
	return nil
}
