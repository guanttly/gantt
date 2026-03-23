package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IShiftWeeklyStaffRepository 班次周默认人数仓储接口
type IShiftWeeklyStaffRepository interface {
	// Create 创建周默认人数记录
	Create(ctx context.Context, weeklyStaff *model.ShiftWeeklyStaff) error

	// Update 更新周默认人数记录
	Update(ctx context.Context, weeklyStaff *model.ShiftWeeklyStaff) error

	// Delete 删除周默认人数记录
	Delete(ctx context.Context, id uint64) error

	// GetByShiftAndWeekday 根据班次ID和周几获取记录
	GetByShiftAndWeekday(ctx context.Context, shiftID string, weekday int) (*model.ShiftWeeklyStaff, error)

	// GetByShiftID 获取班次的所有周默认人数配置
	GetByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftWeeklyStaff, error)

	// GetByShiftIDs 批量获取多个班次的周默认人数配置
	GetByShiftIDs(ctx context.Context, shiftIDs []string) (map[string][]*model.ShiftWeeklyStaff, error)

	// BatchUpsert 批量创建或更新周默认人数
	BatchUpsert(ctx context.Context, shiftID string, weeklyStaffs []*model.ShiftWeeklyStaff) error

	// DeleteByShiftID 删除班次的所有周默认人数配置
	DeleteByShiftID(ctx context.Context, shiftID string) error
}
