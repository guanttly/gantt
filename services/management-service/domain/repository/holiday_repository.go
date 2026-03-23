package repository

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// IHolidayRepository 节假日仓储接口
type IHolidayRepository interface {
	// GetByDate 获取指定日期的节假日配置
	GetByDate(ctx context.Context, orgID string, date time.Time) (*model.Holiday, error)

	// ListByDateRange 获取日期范围内的节假日配置
	ListByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.Holiday, error)

	// ListByYear 获取某年的所有节假日配置
	ListByYear(ctx context.Context, orgID string, year int) ([]*model.Holiday, error)

	// Create 创建节假日配置
	Create(ctx context.Context, holiday *model.Holiday) error

	// BatchCreate 批量创建节假日配置
	BatchCreate(ctx context.Context, holidays []*model.Holiday) error

	// Delete 删除节假日配置
	Delete(ctx context.Context, id uint64) error

	// DeleteByYear 删除某年的所有节假日配置
	DeleteByYear(ctx context.Context, orgID string, year int) error
}
