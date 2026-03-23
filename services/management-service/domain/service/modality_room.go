package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomService 机房管理服务接口
type IModalityRoomService interface {
	// CreateModalityRoom 创建机房
	CreateModalityRoom(ctx context.Context, modalityRoom *model.ModalityRoom) error

	// UpdateModalityRoom 更新机房信息
	UpdateModalityRoom(ctx context.Context, modalityRoom *model.ModalityRoom) error

	// DeleteModalityRoom 删除机房
	DeleteModalityRoom(ctx context.Context, orgID, modalityRoomID string) error

	// GetModalityRoom 获取机房详情
	GetModalityRoom(ctx context.Context, orgID, modalityRoomID string) (*model.ModalityRoom, error)

	// GetModalityRoomByCode 根据编码获取机房
	GetModalityRoomByCode(ctx context.Context, orgID, code string) (*model.ModalityRoom, error)

	// ListModalityRooms 查询机房列表
	ListModalityRooms(ctx context.Context, filter *model.ModalityRoomFilter) (*model.ModalityRoomListResult, error)

	// GetActiveModalityRooms 获取所有启用的机房
	GetActiveModalityRooms(ctx context.Context, orgID string) ([]*model.ModalityRoom, error)

	// ToggleModalityRoomStatus 切换机房启用/禁用状态
	ToggleModalityRoomStatus(ctx context.Context, orgID, modalityRoomID string, isActive bool) error
}
