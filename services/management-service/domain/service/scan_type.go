package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IScanTypeService 检查类型管理服务接口
type IScanTypeService interface {
	// CreateScanType 创建检查类型
	CreateScanType(ctx context.Context, scanType *model.ScanType) error

	// UpdateScanType 更新检查类型信息
	UpdateScanType(ctx context.Context, scanType *model.ScanType) error

	// DeleteScanType 删除检查类型
	DeleteScanType(ctx context.Context, orgID, scanTypeID string) error

	// GetScanType 获取检查类型详情
	GetScanType(ctx context.Context, orgID, scanTypeID string) (*model.ScanType, error)

	// GetScanTypeByCode 根据编码获取检查类型
	GetScanTypeByCode(ctx context.Context, orgID, code string) (*model.ScanType, error)

	// ListScanTypes 查询检查类型列表
	ListScanTypes(ctx context.Context, filter *model.ScanTypeFilter) (*model.ScanTypeListResult, error)

	// GetActiveScanTypes 获取所有启用的检查类型
	GetActiveScanTypes(ctx context.Context, orgID string) ([]*model.ScanType, error)

	// ToggleScanTypeStatus 切换检查类型启用/禁用状态
	ToggleScanTypeStatus(ctx context.Context, orgID, scanTypeID string, isActive bool) error
}
