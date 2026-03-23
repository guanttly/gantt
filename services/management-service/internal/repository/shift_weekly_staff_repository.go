package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ShiftWeeklyStaffRepository 班次周默认人数仓储实现
type ShiftWeeklyStaffRepository struct {
	db *gorm.DB
}

// NewShiftWeeklyStaffRepository 创建班次周默认人数仓储实例
func NewShiftWeeklyStaffRepository(db *gorm.DB) repository.IShiftWeeklyStaffRepository {
	return &ShiftWeeklyStaffRepository{db: db}
}

// Create 创建周默认人数记录
func (r *ShiftWeeklyStaffRepository) Create(ctx context.Context, weeklyStaff *model.ShiftWeeklyStaff) error {
	weeklyStaffEntity := mapper.ShiftWeeklyStaffModelToEntity(weeklyStaff)
	return r.db.WithContext(ctx).Create(weeklyStaffEntity).Error
}

// Update 更新周默认人数记录
func (r *ShiftWeeklyStaffRepository) Update(ctx context.Context, weeklyStaff *model.ShiftWeeklyStaff) error {
	weeklyStaffEntity := mapper.ShiftWeeklyStaffModelToEntity(weeklyStaff)
	return r.db.WithContext(ctx).
		Model(&entity.ShiftWeeklyStaffEntity{}).
		Where("id = ?", weeklyStaff.ID).
		Select("staff_count").
		Updates(weeklyStaffEntity).Error
}

// Delete 删除周默认人数记录
func (r *ShiftWeeklyStaffRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entity.ShiftWeeklyStaffEntity{}).Error
}

// GetByShiftAndWeekday 根据班次ID和周几获取记录
func (r *ShiftWeeklyStaffRepository) GetByShiftAndWeekday(ctx context.Context, shiftID string, weekday int) (*model.ShiftWeeklyStaff, error) {
	var weeklyStaffEntity entity.ShiftWeeklyStaffEntity
	err := r.db.WithContext(ctx).
		Where("shift_id = ? AND weekday = ?", shiftID, weekday).
		First(&weeklyStaffEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftWeeklyStaffEntityToModel(&weeklyStaffEntity), nil
}

// GetByShiftID 获取班次的所有周默认人数配置
func (r *ShiftWeeklyStaffRepository) GetByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftWeeklyStaff, error) {
	var weeklyStaffEntities []*entity.ShiftWeeklyStaffEntity
	err := r.db.WithContext(ctx).
		Where("shift_id = ?", shiftID).
		Order("weekday ASC").
		Find(&weeklyStaffEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftWeeklyStaffEntitiesToModels(weeklyStaffEntities), nil
}

// GetByShiftIDs 批量获取多个班次的周默认人数配置
func (r *ShiftWeeklyStaffRepository) GetByShiftIDs(ctx context.Context, shiftIDs []string) (map[string][]*model.ShiftWeeklyStaff, error) {
	if len(shiftIDs) == 0 {
		return make(map[string][]*model.ShiftWeeklyStaff), nil
	}

	var weeklyStaffEntities []*entity.ShiftWeeklyStaffEntity
	err := r.db.WithContext(ctx).
		Where("shift_id IN ?", shiftIDs).
		Order("shift_id, weekday ASC").
		Find(&weeklyStaffEntities).Error
	if err != nil {
		return nil, err
	}

	// 按 shiftID 分组
	result := make(map[string][]*model.ShiftWeeklyStaff)
	for _, e := range weeklyStaffEntities {
		m := mapper.ShiftWeeklyStaffEntityToModel(e)
		result[e.ShiftID] = append(result[e.ShiftID], m)
	}
	return result, nil
}

// BatchUpsert 批量创建或更新周默认人数
func (r *ShiftWeeklyStaffRepository) BatchUpsert(ctx context.Context, shiftID string, weeklyStaffs []*model.ShiftWeeklyStaff) error {
	if len(weeklyStaffs) == 0 {
		return nil
	}

	entities := make([]*entity.ShiftWeeklyStaffEntity, len(weeklyStaffs))
	for i, ws := range weeklyStaffs {
		entities[i] = mapper.ShiftWeeklyStaffModelToEntity(ws)
		entities[i].ShiftID = shiftID
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "shift_id"}, {Name: "weekday"}},
		DoUpdates: clause.AssignmentColumns([]string{"staff_count", "updated_at"}),
	}).Create(entities).Error
}

// DeleteByShiftID 删除班次的所有周默认人数配置
func (r *ShiftWeeklyStaffRepository) DeleteByShiftID(ctx context.Context, shiftID string) error {
	return r.db.WithContext(ctx).
		Where("shift_id = ?", shiftID).
		Delete(&entity.ShiftWeeklyStaffEntity{}).Error
}
