package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomWeeklyVolumeRepository 机房周检查量仓储接口
type IModalityRoomWeeklyVolumeRepository interface {
	// GetByModalityRoomID 获取指定机房的所有周检查量配置
	GetByModalityRoomID(ctx context.Context, orgID, modalityRoomID string) ([]*model.ModalityRoomWeeklyVolume, error)

	// SaveBatch 批量保存周检查量配置（先删除再插入）
	SaveBatch(ctx context.Context, orgID, modalityRoomID string, items []*model.ModalityRoomWeeklyVolume) error

	// DeleteByModalityRoomID 删除指定机房的所有周检查量配置
	DeleteByModalityRoomID(ctx context.Context, orgID, modalityRoomID string) error

	// GetByFilter 按条件查询周检查量
	GetByFilter(ctx context.Context, orgID string, modalityRoomIDs []string, weekdays []int, timePeriodIDs []string) ([]*model.ModalityRoomWeeklyVolume, error)
}
