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

// ScanTypeServiceImpl 检查类型服务实现
type ScanTypeServiceImpl struct {
	scanTypeRepo repository.IScanTypeRepository
	logger       logging.ILogger
}

// NewScanTypeService 创建检查类型服务实例
func NewScanTypeService(
	scanTypeRepo repository.IScanTypeRepository,
	logger logging.ILogger,
) domain_service.IScanTypeService {
	return &ScanTypeServiceImpl{
		scanTypeRepo: scanTypeRepo,
		logger:       logger,
	}
}

// CreateScanType 创建检查类型
func (s *ScanTypeServiceImpl) CreateScanType(ctx context.Context, scanType *model.ScanType) error {
	// 生成ID
	if scanType.ID == "" {
		scanType.ID = uuid.New().String()
	}

	// 设置默认值
	scanType.IsActive = true
	scanType.CreatedAt = time.Now()
	scanType.UpdatedAt = time.Now()

	// 检查编码是否重复
	existing, err := s.scanTypeRepo.GetByCode(ctx, scanType.OrgID, scanType.Code)
	if err == nil && existing != nil {
		return fmt.Errorf("检查类型编码 %s 已存在", scanType.Code)
	}

	if err := s.scanTypeRepo.Create(ctx, scanType); err != nil {
		s.logger.Error("Failed to create scan type", "error", err)
		return fmt.Errorf("创建检查类型失败: %w", err)
	}

	s.logger.Info("Scan type created", "id", scanType.ID, "name", scanType.Name)
	return nil
}

// UpdateScanType 更新检查类型信息
func (s *ScanTypeServiceImpl) UpdateScanType(ctx context.Context, scanType *model.ScanType) error {
	// 检查检查类型是否存在
	existing, err := s.scanTypeRepo.GetByID(ctx, scanType.OrgID, scanType.ID)
	if err != nil {
		return fmt.Errorf("检查类型不存在: %w", err)
	}

	// 如果修改了编码，检查新编码是否重复
	if scanType.Code != existing.Code {
		existingByCode, err := s.scanTypeRepo.GetByCode(ctx, scanType.OrgID, scanType.Code)
		if err == nil && existingByCode != nil && existingByCode.ID != scanType.ID {
			return fmt.Errorf("检查类型编码 %s 已存在", scanType.Code)
		}
	}

	scanType.UpdatedAt = time.Now()

	if err := s.scanTypeRepo.Update(ctx, scanType); err != nil {
		s.logger.Error("Failed to update scan type", "error", err)
		return fmt.Errorf("更新检查类型失败: %w", err)
	}

	s.logger.Info("Scan type updated", "id", scanType.ID, "name", scanType.Name)
	return nil
}

// DeleteScanType 删除检查类型
func (s *ScanTypeServiceImpl) DeleteScanType(ctx context.Context, orgID, scanTypeID string) error {
	// 检查检查类型是否存在
	_, err := s.scanTypeRepo.GetByID(ctx, orgID, scanTypeID)
	if err != nil {
		return fmt.Errorf("检查类型不存在: %w", err)
	}

	// TODO: 检查是否有关联的检查量数据

	if err := s.scanTypeRepo.Delete(ctx, orgID, scanTypeID); err != nil {
		s.logger.Error("Failed to delete scan type", "error", err)
		return fmt.Errorf("删除检查类型失败: %w", err)
	}

	s.logger.Info("Scan type deleted", "id", scanTypeID)
	return nil
}

// GetScanType 获取检查类型详情
func (s *ScanTypeServiceImpl) GetScanType(ctx context.Context, orgID, scanTypeID string) (*model.ScanType, error) {
	scanType, err := s.scanTypeRepo.GetByID(ctx, orgID, scanTypeID)
	if err != nil {
		return nil, fmt.Errorf("获取检查类型失败: %w", err)
	}
	return scanType, nil
}

// GetScanTypeByCode 根据编码获取检查类型
func (s *ScanTypeServiceImpl) GetScanTypeByCode(ctx context.Context, orgID, code string) (*model.ScanType, error) {
	scanType, err := s.scanTypeRepo.GetByCode(ctx, orgID, code)
	if err != nil {
		return nil, fmt.Errorf("获取检查类型失败: %w", err)
	}
	return scanType, nil
}

// ListScanTypes 查询检查类型列表
func (s *ScanTypeServiceImpl) ListScanTypes(ctx context.Context, filter *model.ScanTypeFilter) (*model.ScanTypeListResult, error) {
	// 设置默认分页参数
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := s.scanTypeRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list scan types", "error", err)
		return nil, fmt.Errorf("查询检查类型列表失败: %w", err)
	}

	return result, nil
}

// GetActiveScanTypes 获取所有启用的检查类型
func (s *ScanTypeServiceImpl) GetActiveScanTypes(ctx context.Context, orgID string) ([]*model.ScanType, error) {
	scanTypes, err := s.scanTypeRepo.GetActiveScanTypes(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to get active scan types", "error", err)
		return nil, fmt.Errorf("获取启用检查类型失败: %w", err)
	}
	return scanTypes, nil
}

// ToggleScanTypeStatus 切换检查类型启用/禁用状态
func (s *ScanTypeServiceImpl) ToggleScanTypeStatus(ctx context.Context, orgID, scanTypeID string, isActive bool) error {
	// 获取现有检查类型
	scanType, err := s.scanTypeRepo.GetByID(ctx, orgID, scanTypeID)
	if err != nil {
		return fmt.Errorf("检查类型不存在: %w", err)
	}

	// 更新状态
	scanType.IsActive = isActive
	scanType.UpdatedAt = time.Now()

	if err := s.scanTypeRepo.Update(ctx, scanType); err != nil {
		s.logger.Error("Failed to toggle scan type status", "error", err)
		return fmt.Errorf("更新检查类型状态失败: %w", err)
	}

	s.logger.Info("Scan type status toggled", "id", scanTypeID, "isActive", isActive)
	return nil
}
