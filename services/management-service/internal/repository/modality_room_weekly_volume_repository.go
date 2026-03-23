package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// ModalityRoomWeeklyVolumeRepository 机房周检查量仓储实现
type ModalityRoomWeeklyVolumeRepository struct {
	db *gorm.DB
}

// NewModalityRoomWeeklyVolumeRepository 创建机房周检查量仓储实例
func NewModalityRoomWeeklyVolumeRepository(db *gorm.DB) repository.IModalityRoomWeeklyVolumeRepository {
	return &ModalityRoomWeeklyVolumeRepository{db: db}
}

// GetByModalityRoomID 获取指定机房的所有周检查量配置
func (r *ModalityRoomWeeklyVolumeRepository) GetByModalityRoomID(ctx context.Context, orgID, modalityRoomID string) ([]*model.ModalityRoomWeeklyVolume, error) {
	var entities []*entity.ModalityRoomWeeklyVolumeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND modality_room_id = ?", orgID, modalityRoomID).
		Order("weekday ASC, time_period_id ASC, scan_type_id ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomWeeklyVolumeEntitiesToModels(entities), nil
}

// SaveBatch 批量保存周检查量配置（先删除再插入）
func (r *ModalityRoomWeeklyVolumeRepository) SaveBatch(ctx context.Context, orgID, modalityRoomID string, items []*model.ModalityRoomWeeklyVolume) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该机房的所有配置
		if err := tx.Where("org_id = ? AND modality_room_id = ?", orgID, modalityRoomID).
			Delete(&entity.ModalityRoomWeeklyVolumeEntity{}).Error; err != nil {
			return err
		}

		// 批量插入新配置
		if len(items) > 0 {
			entities := make([]*entity.ModalityRoomWeeklyVolumeEntity, len(items))
			for i, item := range items {
				entities[i] = mapper.ModalityRoomWeeklyVolumeModelToEntity(item)
			}
			if err := tx.CreateInBatches(entities, 100).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteByModalityRoomID 删除指定机房的所有周检查量配置
func (r *ModalityRoomWeeklyVolumeRepository) DeleteByModalityRoomID(ctx context.Context, orgID, modalityRoomID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND modality_room_id = ?", orgID, modalityRoomID).
		Delete(&entity.ModalityRoomWeeklyVolumeEntity{}).Error
}

// GetByFilter 按条件查询周检查量
func (r *ModalityRoomWeeklyVolumeRepository) GetByFilter(ctx context.Context, orgID string, modalityRoomIDs []string, weekdays []int, timePeriodIDs []string) ([]*model.ModalityRoomWeeklyVolume, error) {
	query := r.db.WithContext(ctx).Model(&entity.ModalityRoomWeeklyVolumeEntity{}).
		Where("org_id = ?", orgID)

	if len(modalityRoomIDs) > 0 {
		query = query.Where("modality_room_id IN ?", modalityRoomIDs)
	}

	if len(weekdays) > 0 {
		query = query.Where("weekday IN ?", weekdays)
	}

	if len(timePeriodIDs) > 0 {
		query = query.Where("time_period_id IN ?", timePeriodIDs)
	}

	var entities []*entity.ModalityRoomWeeklyVolumeEntity
	err := query.Order("modality_room_id ASC, weekday ASC, time_period_id ASC, scan_type_id ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomWeeklyVolumeEntitiesToModels(entities), nil
}
