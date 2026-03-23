package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IScanTypeRepository 检查类型仓储接口
type IScanTypeRepository interface {
	// Create 创建检查类型
	Create(ctx context.Context, scanType *model.ScanType) error

	// Update 更新检查类型信息
	Update(ctx context.Context, scanType *model.ScanType) error

	// Delete 删除检查类型（软删除）
	Delete(ctx context.Context, orgID, scanTypeID string) error

	// GetByID 根据ID获取检查类型
	GetByID(ctx context.Context, orgID, scanTypeID string) (*model.ScanType, error)

	// GetByCode 根据编码获取检查类型
	GetByCode(ctx context.Context, orgID, code string) (*model.ScanType, error)

	// List 查询检查类型列表
	List(ctx context.Context, filter *model.ScanTypeFilter) (*model.ScanTypeListResult, error)

	// Exists 检查检查类型是否存在
	Exists(ctx context.Context, orgID, scanTypeID string) (bool, error)

	// GetActiveScanTypes 获取所有启用的检查类型
	GetActiveScanTypes(ctx context.Context, orgID string) ([]*model.ScanType, error)
}
