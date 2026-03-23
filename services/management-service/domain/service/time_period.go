package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// ITimePeriodService 时间段管理服务接口
type ITimePeriodService interface {
	// CreateTimePeriod 创建时间段
	CreateTimePeriod(ctx context.Context, timePeriod *model.TimePeriod) error

	// UpdateTimePeriod 更新时间段信息
	UpdateTimePeriod(ctx context.Context, timePeriod *model.TimePeriod) error

	// DeleteTimePeriod 删除时间段
	DeleteTimePeriod(ctx context.Context, orgID, timePeriodID string) error

	// GetTimePeriod 获取时间段详情
	GetTimePeriod(ctx context.Context, orgID, timePeriodID string) (*model.TimePeriod, error)

	// GetTimePeriodByCode 根据编码获取时间段
	GetTimePeriodByCode(ctx context.Context, orgID, code string) (*model.TimePeriod, error)

	// GetTimePeriodByName 根据名称获取时间段
	GetTimePeriodByName(ctx context.Context, orgID, name string) (*model.TimePeriod, error)

	// ListTimePeriods 查询时间段列表
	ListTimePeriods(ctx context.Context, filter *model.TimePeriodFilter) (*model.TimePeriodListResult, error)

	// GetActiveTimePeriods 获取所有启用的时间段
	GetActiveTimePeriods(ctx context.Context, orgID string) ([]*model.TimePeriod, error)

	// ToggleTimePeriodStatus 切换时间段启用/禁用状态
	ToggleTimePeriodStatus(ctx context.Context, orgID, timePeriodID string, isActive bool) error
}
