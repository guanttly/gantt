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

// ModalityRoomServiceImpl 机房服务实现
type ModalityRoomServiceImpl struct {
	modalityRoomRepo repository.IModalityRoomRepository
	logger           logging.ILogger
}

// NewModalityRoomService 创建机房服务实例
func NewModalityRoomService(
	modalityRoomRepo repository.IModalityRoomRepository,
	logger logging.ILogger,
) domain_service.IModalityRoomService {
	return &ModalityRoomServiceImpl{
		modalityRoomRepo: modalityRoomRepo,
		logger:           logger,
	}
}

// CreateModalityRoom 创建机房
func (s *ModalityRoomServiceImpl) CreateModalityRoom(ctx context.Context, modalityRoom *model.ModalityRoom) error {
	// 生成ID
	if modalityRoom.ID == "" {
		modalityRoom.ID = uuid.New().String()
	}

	// 设置默认值
	modalityRoom.IsActive = true
	modalityRoom.CreatedAt = time.Now()
	modalityRoom.UpdatedAt = time.Now()

	// 检查编码是否重复
	existing, err := s.modalityRoomRepo.GetByCode(ctx, modalityRoom.OrgID, modalityRoom.Code)
	if err == nil && existing != nil {
		return fmt.Errorf("机房编码 %s 已存在", modalityRoom.Code)
	}

	if err := s.modalityRoomRepo.Create(ctx, modalityRoom); err != nil {
		s.logger.Error("Failed to create modality room", "error", err)
		return fmt.Errorf("创建机房失败: %w", err)
	}

	s.logger.Info("Modality room created", "id", modalityRoom.ID, "name", modalityRoom.Name)
	return nil
}

// UpdateModalityRoom 更新机房信息
func (s *ModalityRoomServiceImpl) UpdateModalityRoom(ctx context.Context, modalityRoom *model.ModalityRoom) error {
	// 检查机房是否存在
	existing, err := s.modalityRoomRepo.GetByID(ctx, modalityRoom.OrgID, modalityRoom.ID)
	if err != nil {
		return fmt.Errorf("机房不存在: %w", err)
	}

	// 如果修改了编码，检查新编码是否重复
	if modalityRoom.Code != existing.Code {
		existingByCode, err := s.modalityRoomRepo.GetByCode(ctx, modalityRoom.OrgID, modalityRoom.Code)
		if err == nil && existingByCode != nil && existingByCode.ID != modalityRoom.ID {
			return fmt.Errorf("机房编码 %s 已存在", modalityRoom.Code)
		}
	}

	modalityRoom.UpdatedAt = time.Now()

	if err := s.modalityRoomRepo.Update(ctx, modalityRoom); err != nil {
		s.logger.Error("Failed to update modality room", "error", err)
		return fmt.Errorf("更新机房失败: %w", err)
	}

	s.logger.Info("Modality room updated", "id", modalityRoom.ID, "name", modalityRoom.Name)
	return nil
}

// DeleteModalityRoom 删除机房
func (s *ModalityRoomServiceImpl) DeleteModalityRoom(ctx context.Context, orgID, modalityRoomID string) error {
	// 检查机房是否存在
	_, err := s.modalityRoomRepo.GetByID(ctx, orgID, modalityRoomID)
	if err != nil {
		return fmt.Errorf("机房不存在: %w", err)
	}

	// TODO: 检查是否有关联的检查量数据或计算规则

	if err := s.modalityRoomRepo.Delete(ctx, orgID, modalityRoomID); err != nil {
		s.logger.Error("Failed to delete modality room", "error", err)
		return fmt.Errorf("删除机房失败: %w", err)
	}

	s.logger.Info("Modality room deleted", "id", modalityRoomID)
	return nil
}

// GetModalityRoom 获取机房详情
func (s *ModalityRoomServiceImpl) GetModalityRoom(ctx context.Context, orgID, modalityRoomID string) (*model.ModalityRoom, error) {
	modalityRoom, err := s.modalityRoomRepo.GetByID(ctx, orgID, modalityRoomID)
	if err != nil {
		return nil, fmt.Errorf("获取机房失败: %w", err)
	}
	return modalityRoom, nil
}

// GetModalityRoomByCode 根据编码获取机房
func (s *ModalityRoomServiceImpl) GetModalityRoomByCode(ctx context.Context, orgID, code string) (*model.ModalityRoom, error) {
	modalityRoom, err := s.modalityRoomRepo.GetByCode(ctx, orgID, code)
	if err != nil {
		return nil, fmt.Errorf("获取机房失败: %w", err)
	}
	return modalityRoom, nil
}

// ListModalityRooms 查询机房列表
func (s *ModalityRoomServiceImpl) ListModalityRooms(ctx context.Context, filter *model.ModalityRoomFilter) (*model.ModalityRoomListResult, error) {
	// 设置默认分页参数
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := s.modalityRoomRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list modality rooms", "error", err)
		return nil, fmt.Errorf("查询机房列表失败: %w", err)
	}

	return result, nil
}

// GetActiveModalityRooms 获取所有启用的机房
func (s *ModalityRoomServiceImpl) GetActiveModalityRooms(ctx context.Context, orgID string) ([]*model.ModalityRoom, error) {
	modalityRooms, err := s.modalityRoomRepo.GetActiveModalityRooms(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to get active modality rooms", "error", err)
		return nil, fmt.Errorf("获取启用机房失败: %w", err)
	}
	return modalityRooms, nil
}

// ToggleModalityRoomStatus 切换机房启用/禁用状态
func (s *ModalityRoomServiceImpl) ToggleModalityRoomStatus(ctx context.Context, orgID, modalityRoomID string, isActive bool) error {
	// 获取现有机房
	modalityRoom, err := s.modalityRoomRepo.GetByID(ctx, orgID, modalityRoomID)
	if err != nil {
		return fmt.Errorf("机房不存在: %w", err)
	}

	// 更新状态
	modalityRoom.IsActive = isActive
	modalityRoom.UpdatedAt = time.Now()

	if err := s.modalityRoomRepo.Update(ctx, modalityRoom); err != nil {
		s.logger.Error("Failed to toggle modality room status", "error", err)
		return fmt.Errorf("更新机房状态失败: %w", err)
	}

	s.logger.Info("Modality room status toggled", "id", modalityRoomID, "isActive", isActive)
	return nil
}
