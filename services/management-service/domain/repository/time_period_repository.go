package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// ITimePeriodRepository 时间段仓储接口
type ITimePeriodRepository interface {
	// Create 创建时间段
	Create(ctx context.Context, timePeriod *model.TimePeriod) error

	// Update 更新时间段信息
	Update(ctx context.Context, timePeriod *model.TimePeriod) error

	// Delete 删除时间段（软删除）
	Delete(ctx context.Context, orgID, timePeriodID string) error

	// GetByID 根据ID获取时间段
	GetByID(ctx context.Context, orgID, timePeriodID string) (*model.TimePeriod, error)

	// GetByCode 根据编码获取时间段
	GetByCode(ctx context.Context, orgID, code string) (*model.TimePeriod, error)

	// GetByName 根据名称获取时间段
	GetByName(ctx context.Context, orgID, name string) (*model.TimePeriod, error)

	// List 查询时间段列表
	List(ctx context.Context, filter *model.TimePeriodFilter) (*model.TimePeriodListResult, error)

	// Exists 检查时间段是否存在
	Exists(ctx context.Context, orgID, timePeriodID string) (bool, error)

	// GetActiveTimePeriods 获取所有启用的时间段
	GetActiveTimePeriods(ctx context.Context, orgID string) ([]*model.TimePeriod, error)
}
