package repository

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomVolumeRepository 机房检查量仓储接口
type IModalityRoomVolumeRepository interface {
	// Create 创建检查量记录
	Create(ctx context.Context, volume *model.ModalityRoomVolume) error

	// Update 更新检查量记录
	Update(ctx context.Context, volume *model.ModalityRoomVolume) error

	// Delete 删除检查量记录
	Delete(ctx context.Context, volumeID string) error

	// GetByID 根据ID获取检查量记录
	GetByID(ctx context.Context, volumeID string) (*model.ModalityRoomVolume, error)

	// GetByRoomDatePeriod 根据机房、日期、时间段获取检查量
	GetByRoomDatePeriod(ctx context.Context, orgID, modalityRoomID string, date time.Time, timePeriodID string) (*model.ModalityRoomVolume, error)

	// List 查询检查量列表
	List(ctx context.Context, filter *model.ModalityRoomVolumeFilter) (*model.ModalityRoomVolumeListResult, error)

	// BatchCreate 批量创建检查量记录
	BatchCreate(ctx context.Context, volumes []*model.ModalityRoomVolume) error

	// BatchUpsert 批量创建或更新检查量记录（按机房+日期+时间段去重）
	BatchUpsert(ctx context.Context, volumes []*model.ModalityRoomVolume) error

	// GetLatestWeekVolumes 获取最近一周有数据的检查量
	// 返回指定机房列表和时间段的最近有数据的检查量记录（最多7天）
	GetLatestWeekVolumes(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string) ([]*model.ModalityRoomVolume, error)

	// GetVolumesSummary 获取检查量汇总
	// 按机房汇总指定时间段内的检查量
	GetVolumesSummary(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string, startDate, endDate time.Time) ([]*model.ModalityRoomVolumeSummary, error)
}
