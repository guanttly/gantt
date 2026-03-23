package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomRepository 机房仓储接口
type IModalityRoomRepository interface {
	// Create 创建机房
	Create(ctx context.Context, modalityRoom *model.ModalityRoom) error

	// Update 更新机房信息
	Update(ctx context.Context, modalityRoom *model.ModalityRoom) error

	// Delete 删除机房（软删除）
	Delete(ctx context.Context, orgID, modalityRoomID string) error

	// GetByID 根据ID获取机房
	GetByID(ctx context.Context, orgID, modalityRoomID string) (*model.ModalityRoom, error)

	// GetByCode 根据编码获取机房
	GetByCode(ctx context.Context, orgID, code string) (*model.ModalityRoom, error)

	// List 查询机房列表
	List(ctx context.Context, filter *model.ModalityRoomFilter) (*model.ModalityRoomListResult, error)

	// Exists 检查机房是否存在
	Exists(ctx context.Context, orgID, modalityRoomID string) (bool, error)

	// BatchGet 批量获取机房
	BatchGet(ctx context.Context, orgID string, modalityRoomIDs []string) ([]*model.ModalityRoom, error)

	// GetActiveModalityRooms 获取所有启用的机房
	GetActiveModalityRooms(ctx context.Context, orgID string) ([]*model.ModalityRoom, error)
}
