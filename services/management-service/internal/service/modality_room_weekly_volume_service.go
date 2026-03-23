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

// ModalityRoomWeeklyVolumeServiceImpl 机房周检查量服务实现
type ModalityRoomWeeklyVolumeServiceImpl struct {
	weeklyVolumeRepo repository.IModalityRoomWeeklyVolumeRepository
	modalityRoomRepo repository.IModalityRoomRepository
	timePeriodRepo   repository.ITimePeriodRepository
	scanTypeRepo     repository.IScanTypeRepository
	logger           logging.ILogger
}

// NewModalityRoomWeeklyVolumeService 创建机房周检查量服务实例
func NewModalityRoomWeeklyVolumeService(
	weeklyVolumeRepo repository.IModalityRoomWeeklyVolumeRepository,
	modalityRoomRepo repository.IModalityRoomRepository,
	timePeriodRepo repository.ITimePeriodRepository,
	scanTypeRepo repository.IScanTypeRepository,
	logger logging.ILogger,
) domain_service.IModalityRoomWeeklyVolumeService {
	return &ModalityRoomWeeklyVolumeServiceImpl{
		weeklyVolumeRepo: weeklyVolumeRepo,
		modalityRoomRepo: modalityRoomRepo,
		timePeriodRepo:   timePeriodRepo,
		scanTypeRepo:     scanTypeRepo,
		logger:           logger,
	}
}

// GetWeeklyVolumes 获取指定机房的周检查量配置
func (s *ModalityRoomWeeklyVolumeServiceImpl) GetWeeklyVolumes(ctx context.Context, orgID, modalityRoomID string) (*model.WeeklyVolumeListResult, error) {
	// 检查机房是否存在
	_, err := s.modalityRoomRepo.GetByID(ctx, orgID, modalityRoomID)
	if err != nil {
		return nil, fmt.Errorf("机房不存在: %w", err)
	}

	// 获取周检查量配置
	items, err := s.weeklyVolumeRepo.GetByModalityRoomID(ctx, orgID, modalityRoomID)
	if err != nil {
		s.logger.Error("Failed to get weekly volumes", "error", err)
		return nil, fmt.Errorf("获取周检查量配置失败: %w", err)
	}

	// 填充冗余字段
	if err := s.fillDisplayNames(ctx, orgID, items); err != nil {
		s.logger.Warn("Failed to fill display names", "error", err)
	}

	return &model.WeeklyVolumeListResult{
		Items: items,
	}, nil
}

// SaveWeeklyVolumes 批量保存周检查量配置
func (s *ModalityRoomWeeklyVolumeServiceImpl) SaveWeeklyVolumes(ctx context.Context, orgID string, req *model.WeeklyVolumeSaveRequest) error {
	// 检查机房是否存在
	_, err := s.modalityRoomRepo.GetByID(ctx, orgID, req.ModalityRoomID)
	if err != nil {
		return fmt.Errorf("机房不存在: %w", err)
	}

	// 转换为领域模型
	now := time.Now()
	models := make([]*model.ModalityRoomWeeklyVolume, 0, len(req.Items))
	for _, item := range req.Items {
		// 跳过检查量为0的记录
		if item.Volume <= 0 {
			continue
		}

		models = append(models, &model.ModalityRoomWeeklyVolume{
			ID:             uuid.New().String(),
			OrgID:          orgID,
			ModalityRoomID: req.ModalityRoomID,
			Weekday:        item.Weekday,
			TimePeriodID:   item.TimePeriodID,
			ScanTypeID:     item.ScanTypeID,
			Volume:         item.Volume,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	// 批量保存
	if err := s.weeklyVolumeRepo.SaveBatch(ctx, orgID, req.ModalityRoomID, models); err != nil {
		s.logger.Error("Failed to save weekly volumes", "error", err)
		return fmt.Errorf("保存周检查量配置失败: %w", err)
	}

	s.logger.Info("Weekly volumes saved", "modalityRoomId", req.ModalityRoomID, "count", len(models))
	return nil
}

// DeleteWeeklyVolumes 删除指定机房的所有周检查量配置
func (s *ModalityRoomWeeklyVolumeServiceImpl) DeleteWeeklyVolumes(ctx context.Context, orgID, modalityRoomID string) error {
	if err := s.weeklyVolumeRepo.DeleteByModalityRoomID(ctx, orgID, modalityRoomID); err != nil {
		s.logger.Error("Failed to delete weekly volumes", "error", err)
		return fmt.Errorf("删除周检查量配置失败: %w", err)
	}

	s.logger.Info("Weekly volumes deleted", "modalityRoomId", modalityRoomID)
	return nil
}

// GetWeeklyVolumesByFilter 按条件查询周检查量（用于计算）
func (s *ModalityRoomWeeklyVolumeServiceImpl) GetWeeklyVolumesByFilter(ctx context.Context, orgID string, modalityRoomIDs []string, weekdays []int, timePeriodIDs []string) ([]*model.ModalityRoomWeeklyVolume, error) {
	items, err := s.weeklyVolumeRepo.GetByFilter(ctx, orgID, modalityRoomIDs, weekdays, timePeriodIDs)
	if err != nil {
		s.logger.Error("Failed to get weekly volumes by filter", "error", err)
		return nil, fmt.Errorf("查询周检查量失败: %w", err)
	}
	return items, nil
}

// fillDisplayNames 填充展示用的名称字段
func (s *ModalityRoomWeeklyVolumeServiceImpl) fillDisplayNames(ctx context.Context, orgID string, items []*model.ModalityRoomWeeklyVolume) error {
	if len(items) == 0 {
		return nil
	}

	// 收集需要查询的ID
	timePeriodIDSet := make(map[string]bool)
	scanTypeIDSet := make(map[string]bool)
	for _, item := range items {
		timePeriodIDSet[item.TimePeriodID] = true
		scanTypeIDSet[item.ScanTypeID] = true
	}

	// 查询时间段名称
	timePeriodNames := make(map[string]string)
	for tpID := range timePeriodIDSet {
		tp, err := s.timePeriodRepo.GetByID(ctx, orgID, tpID)
		if err == nil && tp != nil {
			timePeriodNames[tpID] = tp.Name
		}
	}

	// 查询检查类型名称
	scanTypeNames := make(map[string]string)
	for stID := range scanTypeIDSet {
		st, err := s.scanTypeRepo.GetByID(ctx, orgID, stID)
		if err == nil && st != nil {
			scanTypeNames[stID] = st.Name
		}
	}

	// 填充名称
	for _, item := range items {
		item.TimePeriodName = timePeriodNames[item.TimePeriodID]
		item.ScanTypeName = scanTypeNames[item.ScanTypeID]
	}

	return nil
}
