package repository

import (
	"context"
	"jusha/agent/rostering/domain/model"
)

// ShiftTypeRepository 班次类型仓储接口
type ShiftTypeRepository interface {
	// Create 创建班次类型
	Create(ctx context.Context, shiftType *model.ShiftType) error

	// Update 更新班次类型
	Update(ctx context.Context, id string, shiftType *model.ShiftType) error

	// Delete 软删除班次类型
	Delete(ctx context.Context, id string) error

	// GetByID 根据ID获取班次类型
	GetByID(ctx context.Context, id string) (*model.ShiftType, error)

	// GetByCode 根据编码获取班次类型
	GetByCode(ctx context.Context, orgID, code string) (*model.ShiftType, error)

	// List 查询班次类型列表
	List(ctx context.Context, req *model.ListShiftTypesRequest) ([]*model.ShiftType, int, error)

	// ListByOrgID 获取组织的所有班次类型（包括系统类型）
	ListByOrgID(ctx context.Context, orgID string, includeSystem bool) ([]*model.ShiftType, error)

	// ListByWorkflowPhase 根据工作流阶段获取班次类型
	ListByWorkflowPhase(ctx context.Context, orgID, phase string) ([]*model.ShiftType, error)

	// ListByPriority 按优先级排序获取班次类型
	ListByPriority(ctx context.Context, orgID string, ascending bool) ([]*model.ShiftType, error)

	// GetStats 获取班次类型统计信息
	GetStats(ctx context.Context, orgID string) ([]*model.ShiftTypeStats, error)

	// BatchGetByIDs 批量获取班次类型
	BatchGetByIDs(ctx context.Context, ids []string) (map[string]*model.ShiftType, error)

	// Exists 检查班次类型是否存在
	Exists(ctx context.Context, orgID, code string) (bool, error)

	// CanDelete 检查是否可以删除（是否有关联班次）
	CanDelete(ctx context.Context, id string) (bool, error)
}
