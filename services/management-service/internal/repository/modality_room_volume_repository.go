package repository

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ModalityRoomVolumeRepository 检查量仓储实现
type ModalityRoomVolumeRepository struct {
	db *gorm.DB
}

// NewModalityRoomVolumeRepository 创建检查量仓储实例
func NewModalityRoomVolumeRepository(db *gorm.DB) repository.IModalityRoomVolumeRepository {
	return &ModalityRoomVolumeRepository{db: db}
}

// Create 创建检查量记录
func (r *ModalityRoomVolumeRepository) Create(ctx context.Context, volume *model.ModalityRoomVolume) error {
	volumeEntity := mapper.ModalityRoomVolumeModelToEntity(volume)
	return r.db.WithContext(ctx).Create(volumeEntity).Error
}

// Update 更新检查量记录
func (r *ModalityRoomVolumeRepository) Update(ctx context.Context, volume *model.ModalityRoomVolume) error {
	volumeEntity := mapper.ModalityRoomVolumeModelToEntity(volume)
	return r.db.WithContext(ctx).
		Model(&entity.ModalityRoomVolumeEntity{}).
		Where("id = ?", volume.ID).
		Omit("created_at").
		Updates(volumeEntity).Error
}

// Delete 删除检查量记录
func (r *ModalityRoomVolumeRepository) Delete(ctx context.Context, volumeID string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", volumeID).
		Delete(&entity.ModalityRoomVolumeEntity{}).Error
}

// GetByID 根据ID获取检查量记录
func (r *ModalityRoomVolumeRepository) GetByID(ctx context.Context, volumeID string) (*model.ModalityRoomVolume, error) {
	var volumeEntity entity.ModalityRoomVolumeEntity
	err := r.db.WithContext(ctx).
		Where("id = ?", volumeID).
		First(&volumeEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomVolumeEntityToModel(&volumeEntity), nil
}

// GetByRoomDatePeriod 根据机房、日期、时间段获取检查量
func (r *ModalityRoomVolumeRepository) GetByRoomDatePeriod(ctx context.Context, orgID, modalityRoomID string, date time.Time, timePeriodID string) (*model.ModalityRoomVolume, error) {
	var volumeEntity entity.ModalityRoomVolumeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND modality_room_id = ? AND date = ? AND time_period_id = ?", orgID, modalityRoomID, date, timePeriodID).
		First(&volumeEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomVolumeEntityToModel(&volumeEntity), nil
}

// List 查询检查量列表
func (r *ModalityRoomVolumeRepository) List(ctx context.Context, filter *model.ModalityRoomVolumeFilter) (*model.ModalityRoomVolumeListResult, error) {
	query := r.db.WithContext(ctx).Model(&entity.ModalityRoomVolumeEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 机房过滤
	if filter.ModalityRoomID != "" {
		query = query.Where("modality_room_id = ?", filter.ModalityRoomID)
	}
	if len(filter.ModalityRoomIDs) > 0 {
		query = query.Where("modality_room_id IN ?", filter.ModalityRoomIDs)
	}

	// 时间段过滤
	if filter.TimePeriodID != "" {
		query = query.Where("time_period_id = ?", filter.TimePeriodID)
	}

	// 日期范围过滤
	if !filter.StartDate.IsZero() {
		query = query.Where("date >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("date <= ?", filter.EndDate)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var volumeEntities []*entity.ModalityRoomVolumeEntity
	offset := (filter.Page - 1) * filter.PageSize
	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Order("date DESC, modality_room_id ASC").Find(&volumeEntities).Error; err != nil {
		return nil, err
	}

	return &model.ModalityRoomVolumeListResult{
		Items:    mapper.ModalityRoomVolumeEntitiesToModels(volumeEntities),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// BatchCreate 批量创建检查量记录
func (r *ModalityRoomVolumeRepository) BatchCreate(ctx context.Context, volumes []*model.ModalityRoomVolume) error {
	if len(volumes) == 0 {
		return nil
	}

	entities := make([]*entity.ModalityRoomVolumeEntity, len(volumes))
	for i, v := range volumes {
		entities[i] = mapper.ModalityRoomVolumeModelToEntity(v)
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// BatchUpsert 批量创建或更新检查量记录
func (r *ModalityRoomVolumeRepository) BatchUpsert(ctx context.Context, volumes []*model.ModalityRoomVolume) error {
	if len(volumes) == 0 {
		return nil
	}

	entities := make([]*entity.ModalityRoomVolumeEntity, len(volumes))
	for i, v := range volumes {
		entities[i] = mapper.ModalityRoomVolumeModelToEntity(v)
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "modality_room_id"}, {Name: "date"}, {Name: "time_period_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"report_volume", "notes", "updated_at"}),
	}).CreateInBatches(entities, 100).Error
}

// GetLatestWeekVolumes 获取最近一周有数据的检查量
func (r *ModalityRoomVolumeRepository) GetLatestWeekVolumes(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string) ([]*model.ModalityRoomVolume, error) {
	// 先找出最近有数据的日期
	var latestDate time.Time
	err := r.db.WithContext(ctx).Model(&entity.ModalityRoomVolumeEntity{}).
		Select("MAX(date)").
		Where("org_id = ? AND modality_room_id IN ? AND time_period_id = ?", orgID, modalityRoomIDs, timePeriodID).
		Row().Scan(&latestDate)
	if err != nil {
		return nil, err
	}

	if latestDate.IsZero() {
		return []*model.ModalityRoomVolume{}, nil
	}

	// 计算一周前的日期
	startDate := latestDate.AddDate(0, 0, -6)

	// 查询这段时间内的所有数据
	var volumeEntities []*entity.ModalityRoomVolumeEntity
	err = r.db.WithContext(ctx).
		Where("org_id = ? AND modality_room_id IN ? AND time_period_id = ? AND date >= ? AND date <= ?",
			orgID, modalityRoomIDs, timePeriodID, startDate, latestDate).
		Order("date DESC, modality_room_id ASC").
		Find(&volumeEntities).Error
	if err != nil {
		return nil, err
	}

	return mapper.ModalityRoomVolumeEntitiesToModels(volumeEntities), nil
}

// GetVolumesSummary 获取检查量汇总
func (r *ModalityRoomVolumeRepository) GetVolumesSummary(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string, startDate, endDate time.Time) ([]*model.ModalityRoomVolumeSummary, error) {
	type summaryResult struct {
		ModalityRoomID string
		TotalVolume    int
		DataDays       int
	}

	var results []summaryResult
	err := r.db.WithContext(ctx).Model(&entity.ModalityRoomVolumeEntity{}).
		Select("modality_room_id, SUM(report_volume) as total_volume, COUNT(DISTINCT date) as data_days").
		Where("org_id = ? AND modality_room_id IN ? AND time_period_id = ? AND date >= ? AND date <= ?",
			orgID, modalityRoomIDs, timePeriodID, startDate, endDate).
		Group("modality_room_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	summaries := make([]*model.ModalityRoomVolumeSummary, len(results))
	for i, r := range results {
		avgVolume := 0
		if r.DataDays > 0 {
			avgVolume = r.TotalVolume / r.DataDays
		}
		summaries[i] = &model.ModalityRoomVolumeSummary{
			ModalityRoomID: r.ModalityRoomID,
			TotalVolume:    r.TotalVolume,
			DataDays:       r.DataDays,
			AvgVolume:      avgVolume,
		}
	}

	return summaries, nil
}
