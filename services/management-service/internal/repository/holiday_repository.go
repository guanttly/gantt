package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	domain_repo "jusha/gantt/service/management/domain/repository"
)

// HolidayRepositoryImpl 节假日仓储实现
type HolidayRepositoryImpl struct {
	db *gorm.DB
}

// NewHolidayRepository 创建节假日仓储
func NewHolidayRepository(db *gorm.DB) domain_repo.IHolidayRepository {
	return &HolidayRepositoryImpl{db: db}
}

// GetByDate 获取指定日期的节假日配置
func (r *HolidayRepositoryImpl) GetByDate(ctx context.Context, orgID string, date time.Time) (*model.Holiday, error) {
	var holidayEntity entity.HolidayEntity

	// 先查组织特定配置，再查通用配置
	query := r.db.WithContext(ctx).
		Where("date = ?", date.Format("2006-01-02")).
		Where("org_id = ? OR org_id = ''", orgID).
		Order("org_id DESC"). // 优先返回组织特定配置
		First(&holidayEntity)

	if err := query.Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有配置，返回nil而不是错误
		}
		return nil, fmt.Errorf("query holiday by date: %w", err)
	}

	return mapper.HolidayEntityToModel(&holidayEntity), nil
}

// ListByDateRange 获取日期范围内的节假日配置
func (r *HolidayRepositoryImpl) ListByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.Holiday, error) {
	var holidayEntities []*entity.HolidayEntity

	query := r.db.WithContext(ctx).
		Where("date >= ? AND date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Where("org_id = ? OR org_id = ''", orgID).
		Order("date ASC")

	if err := query.Find(&holidayEntities).Error; err != nil {
		return nil, fmt.Errorf("query holidays by date range: %w", err)
	}

	return mapper.HolidayEntitiesToModels(holidayEntities), nil
}

// ListByYear 获取某年的所有节假日配置
func (r *HolidayRepositoryImpl) ListByYear(ctx context.Context, orgID string, year int) ([]*model.Holiday, error) {
	var holidayEntities []*entity.HolidayEntity

	query := r.db.WithContext(ctx).
		Where("year = ?", year).
		Where("org_id = ? OR org_id = ''", orgID).
		Order("date ASC")

	if err := query.Find(&holidayEntities).Error; err != nil {
		return nil, fmt.Errorf("query holidays by year: %w", err)
	}

	return mapper.HolidayEntitiesToModels(holidayEntities), nil
}

// Create 创建节假日配置
func (r *HolidayRepositoryImpl) Create(ctx context.Context, holiday *model.Holiday) error {
	// 自动设置年份
	if holiday.Year == 0 {
		holiday.Year = holiday.Date.Year()
	}

	holidayEntity := mapper.HolidayModelToEntity(holiday)
	if err := r.db.WithContext(ctx).Create(holidayEntity).Error; err != nil {
		return fmt.Errorf("create holiday: %w", err)
	}

	return nil
}

// BatchCreate 批量创建节假日配置
func (r *HolidayRepositoryImpl) BatchCreate(ctx context.Context, holidays []*model.Holiday) error {
	if len(holidays) == 0 {
		return nil
	}

	// 自动设置年份并转换为实体
	holidayEntities := make([]*entity.HolidayEntity, 0, len(holidays))
	for _, h := range holidays {
		if h.Year == 0 {
			h.Year = h.Date.Year()
		}
		holidayEntities = append(holidayEntities, mapper.HolidayModelToEntity(h))
	}

	if err := r.db.WithContext(ctx).CreateInBatches(holidayEntities, 100).Error; err != nil {
		return fmt.Errorf("batch create holidays: %w", err)
	}

	return nil
}

// Delete 删除节假日配置
func (r *HolidayRepositoryImpl) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&entity.HolidayEntity{}, id).Error; err != nil {
		return fmt.Errorf("delete holiday: %w", err)
	}

	return nil
}

// DeleteByYear 删除某年的所有节假日配置
func (r *HolidayRepositoryImpl) DeleteByYear(ctx context.Context, orgID string, year int) error {
	query := r.db.WithContext(ctx).Where("year = ?", year)

	if orgID != "" {
		query = query.Where("org_id = ?", orgID)
	}

	if err := query.Delete(&entity.HolidayEntity{}).Error; err != nil {
		return fmt.Errorf("delete holidays by year: %w", err)
	}

	return nil
}
