package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomWeeklyVolumeService 机房周检查量服务接口
type IModalityRoomWeeklyVolumeService interface {
	// GetWeeklyVolumes 获取指定机房的周检查量配置
	GetWeeklyVolumes(ctx context.Context, orgID, modalityRoomID string) (*model.WeeklyVolumeListResult, error)

	// SaveWeeklyVolumes 批量保存周检查量配置
	SaveWeeklyVolumes(ctx context.Context, orgID string, req *model.WeeklyVolumeSaveRequest) error

	// DeleteWeeklyVolumes 删除指定机房的所有周检查量配置
	DeleteWeeklyVolumes(ctx context.Context, orgID, modalityRoomID string) error

	// GetWeeklyVolumesByFilter 按条件查询周检查量（用于计算）
	GetWeeklyVolumesByFilter(ctx context.Context, orgID string, modalityRoomIDs []string, weekdays []int, timePeriodIDs []string) ([]*model.ModalityRoomWeeklyVolume, error)
}
