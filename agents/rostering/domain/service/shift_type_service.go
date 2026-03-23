package service

import (
	"context"
	"errors"
	"fmt"

	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/domain/repository"

	"github.com/google/uuid"
)

var (
	ErrShiftTypeNotFound       = errors.New("班次类型不存在")
	ErrShiftTypeCodeDuplicate  = errors.New("班次类型编码已存在")
	ErrShiftTypeInUse          = errors.New("班次类型正在使用中，无法删除")
	ErrSystemShiftTypeReadonly = errors.New("系统内置班次类型不可修改或删除")
)

// ShiftTypeService 班次类型领域服务
type ShiftTypeService interface {
	// CreateShiftType 创建班次类型
	CreateShiftType(ctx context.Context, req *model.CreateShiftTypeRequest) (*model.ShiftType, error)

	// UpdateShiftType 更新班次类型
	UpdateShiftType(ctx context.Context, id string, req *model.UpdateShiftTypeRequest) (*model.ShiftType, error)

	// DeleteShiftType 删除班次类型
	DeleteShiftType(ctx context.Context, id string) error

	// GetShiftType 获取班次类型详情
	GetShiftType(ctx context.Context, id string) (*model.ShiftType, error)

	// ListShiftTypes 查询班次类型列表
	ListShiftTypes(ctx context.Context, req *model.ListShiftTypesRequest) ([]*model.ShiftType, int, error)

	// GetShiftTypesByPriority 按优先级获取班次类型（用于排班）
	GetShiftTypesByPriority(ctx context.Context, orgID string) ([]*model.ShiftType, error)

	// GetShiftTypeStats 获取班次类型统计
	GetShiftTypeStats(ctx context.Context, orgID string) ([]*model.ShiftTypeStats, error)

	// GetWorkflowPhases 获取工作流阶段列表
	GetWorkflowPhases() []model.WorkflowPhaseInfo
}

// shiftTypeServiceImpl 班次类型服务实现
type shiftTypeServiceImpl struct {
	repo repository.ShiftTypeRepository
}

// NewShiftTypeService 创建班次类型服务实例
func NewShiftTypeService(repo repository.ShiftTypeRepository) ShiftTypeService {
	return &shiftTypeServiceImpl{
		repo: repo,
	}
}

// CreateShiftType 创建班次类型
func (s *shiftTypeServiceImpl) CreateShiftType(ctx context.Context, req *model.CreateShiftTypeRequest) (*model.ShiftType, error) {
	// 检查编码是否已存在
	exists, err := s.repo.Exists(ctx, req.OrgID, req.Code)
	if err != nil {
		return nil, fmt.Errorf("检查班次类型编码失败: %w", err)
	}
	if exists {
		return nil, ErrShiftTypeCodeDuplicate
	}

	// 构建班次类型模型
	shiftType := &model.ShiftType{
		ID:                   uuid.New().String(),
		OrgID:                req.OrgID,
		Code:                 req.Code,
		Name:                 req.Name,
		Description:          req.Description,
		SchedulingPriority:   req.SchedulingPriority,
		WorkflowPhase:        req.WorkflowPhase,
		Color:                req.Color,
		Icon:                 req.Icon,
		SortOrder:            req.SortOrder,
		IsAIScheduling:       req.IsAIScheduling,
		IsFixedSchedule:      req.IsFixedSchedule,
		IsOvertime:           req.IsOvertime,
		RequiresSpecialSkill: req.RequiresSpecialSkill,
		IsActive:             true,
		IsSystem:             false,
	}

	// 创建到数据库
	if err := s.repo.Create(ctx, shiftType); err != nil {
		return nil, fmt.Errorf("创建班次类型失败: %w", err)
	}

	return shiftType, nil
}

// UpdateShiftType 更新班次类型
func (s *shiftTypeServiceImpl) UpdateShiftType(ctx context.Context, id string, req *model.UpdateShiftTypeRequest) (*model.ShiftType, error) {
	// 获取现有班次类型
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取班次类型失败: %w", err)
	}
	if existing == nil {
		return nil, ErrShiftTypeNotFound
	}

	// 系统内置类型不可修改
	if existing.IsSystem {
		return nil, ErrSystemShiftTypeReadonly
	}

	// 更新字段
	existing.Name = req.Name
	existing.Description = req.Description
	existing.SchedulingPriority = req.SchedulingPriority
	existing.WorkflowPhase = req.WorkflowPhase
	existing.Color = req.Color
	existing.Icon = req.Icon
	existing.SortOrder = req.SortOrder
	existing.IsAIScheduling = req.IsAIScheduling
	existing.IsFixedSchedule = req.IsFixedSchedule
	existing.IsOvertime = req.IsOvertime
	existing.RequiresSpecialSkill = req.RequiresSpecialSkill
	existing.IsActive = req.IsActive

	// 保存更新
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, fmt.Errorf("更新班次类型失败: %w", err)
	}

	return existing, nil
}

// DeleteShiftType 删除班次类型
func (s *shiftTypeServiceImpl) DeleteShiftType(ctx context.Context, id string) error {
	// 获取班次类型
	shiftType, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取班次类型失败: %w", err)
	}
	if shiftType == nil {
		return ErrShiftTypeNotFound
	}

	// 系统内置类型不可删除
	if shiftType.IsSystem {
		return ErrSystemShiftTypeReadonly
	}

	// 检查是否可以删除
	canDelete, err := s.repo.CanDelete(ctx, id)
	if err != nil {
		return fmt.Errorf("检查班次类型是否可删除失败: %w", err)
	}
	if !canDelete {
		return ErrShiftTypeInUse
	}

	// 执行删除
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除班次类型失败: %w", err)
	}

	return nil
}

// GetShiftType 获取班次类型详情
func (s *shiftTypeServiceImpl) GetShiftType(ctx context.Context, id string) (*model.ShiftType, error) {
	shiftType, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取班次类型失败: %w", err)
	}
	if shiftType == nil {
		return nil, ErrShiftTypeNotFound
	}
	return shiftType, nil
}

// ListShiftTypes 查询班次类型列表
func (s *shiftTypeServiceImpl) ListShiftTypes(ctx context.Context, req *model.ListShiftTypesRequest) ([]*model.ShiftType, int, error) {
	return s.repo.List(ctx, req)
}

// GetShiftTypesByPriority 按优先级获取班次类型（用于排班）
func (s *shiftTypeServiceImpl) GetShiftTypesByPriority(ctx context.Context, orgID string) ([]*model.ShiftType, error) {
	return s.repo.ListByPriority(ctx, orgID, true)
}

// GetShiftTypeStats 获取班次类型统计
func (s *shiftTypeServiceImpl) GetShiftTypeStats(ctx context.Context, orgID string) ([]*model.ShiftTypeStats, error) {
	return s.repo.GetStats(ctx, orgID)
}

// GetWorkflowPhases 获取工作流阶段列表
func (s *shiftTypeServiceImpl) GetWorkflowPhases() []model.WorkflowPhaseInfo {
	return model.GetWorkflowPhases()
}
