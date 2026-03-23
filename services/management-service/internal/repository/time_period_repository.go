package repository

import (
	"context"
	"fmt"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// TimePeriodRepository 时间段仓储实现
type TimePeriodRepository struct {
	db *gorm.DB
}

// NewTimePeriodRepository 创建时间段仓储实例
func NewTimePeriodRepository(db *gorm.DB) repository.ITimePeriodRepository {
	return &TimePeriodRepository{db: db}
}

// Create 创建时间段
func (r *TimePeriodRepository) Create(ctx context.Context, timePeriod *model.TimePeriod) error {
	timePeriodEntity := mapper.TimePeriodModelToEntity(timePeriod)
	return r.db.WithContext(ctx).Create(timePeriodEntity).Error
}

// Update 更新时间段信息
func (r *TimePeriodRepository) Update(ctx context.Context, timePeriod *model.TimePeriod) error {
	timePeriodEntity := mapper.TimePeriodModelToEntity(timePeriod)
	return r.db.WithContext(ctx).
		Model(&entity.TimePeriodEntity{}).
		Where("org_id = ? AND id = ?", timePeriod.OrgID, timePeriod.ID).
		Omit("created_at").
		Updates(timePeriodEntity).Error
}

// Delete 删除时间段（软删除）
func (r *TimePeriodRepository) Delete(ctx context.Context, orgID, timePeriodID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, timePeriodID).
		Delete(&entity.TimePeriodEntity{}).Error
}

// GetByID 根据ID获取时间段
func (r *TimePeriodRepository) GetByID(ctx context.Context, orgID, timePeriodID string) (*model.TimePeriod, error) {
	var timePeriodEntity entity.TimePeriodEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, timePeriodID).
		First(&timePeriodEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.TimePeriodEntityToModel(&timePeriodEntity), nil
}

// GetByCode 根据编码获取时间段
func (r *TimePeriodRepository) GetByCode(ctx context.Context, orgID, code string) (*model.TimePeriod, error) {
	var timePeriodEntity entity.TimePeriodEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&timePeriodEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.TimePeriodEntityToModel(&timePeriodEntity), nil
}

// GetByName 根据名称获取时间段
func (r *TimePeriodRepository) GetByName(ctx context.Context, orgID, name string) (*model.TimePeriod, error) {
	var timePeriodEntity entity.TimePeriodEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND name = ?", orgID, name).
		First(&timePeriodEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.TimePeriodEntityToModel(&timePeriodEntity), nil
}

// List 查询时间段列表
func (r *TimePeriodRepository) List(ctx context.Context, filter *model.TimePeriodFilter) (*model.TimePeriodListResult, error) {
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.TimePeriodEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 关键词搜索
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", keyword, keyword)
	}

	// 启用状态过滤
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var timePeriodEntities []*entity.TimePeriodEntity
	offset := (filter.Page - 1) * filter.PageSize
	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Order("sort_order ASC, created_at DESC").Find(&timePeriodEntities).Error; err != nil {
		return nil, err
	}

	return &model.TimePeriodListResult{
		Items:    mapper.TimePeriodEntitiesToModels(timePeriodEntities),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查时间段是否存在
func (r *TimePeriodRepository) Exists(ctx context.Context, orgID, timePeriodID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TimePeriodEntity{}).
		Where("org_id = ? AND id = ?", orgID, timePeriodID).
		Count(&count).Error
	return count > 0, err
}

// GetActiveTimePeriods 获取所有启用的时间段
func (r *TimePeriodRepository) GetActiveTimePeriods(ctx context.Context, orgID string) ([]*model.TimePeriod, error) {
	var timePeriodEntities []*entity.TimePeriodEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("sort_order ASC, created_at DESC").
		Find(&timePeriodEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.TimePeriodEntitiesToModels(timePeriodEntities), nil
}
