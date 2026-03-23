package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	domain_repo "jusha/gantt/service/management/domain/repository"
)

// SystemSettingRepository 系统设置仓储实现
type SystemSettingRepository struct {
	db *gorm.DB
}

// NewSystemSettingRepository 创建系统设置仓储
func NewSystemSettingRepository(db *gorm.DB) domain_repo.ISystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// GetByKey 根据组织ID和键获取设置
func (r *SystemSettingRepository) GetByKey(ctx context.Context, orgID, key string) (*model.SystemSetting, error) {
	var settingEntity entity.SystemSettingEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND `key` = ?", orgID, key).
		First(&settingEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，不返回错误
		}
		return nil, err
	}
	return mapper.SystemSettingEntityToModel(&settingEntity), nil
}

// Set 设置或更新系统设置
func (r *SystemSettingRepository) Set(ctx context.Context, setting *model.SystemSetting) error {
	settingEntity := mapper.SystemSettingModelToEntity(setting)
	if settingEntity == nil {
		return fmt.Errorf("invalid setting")
	}

	// 检查是否已存在
	var existing entity.SystemSettingEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND `key` = ?", setting.OrgID, setting.Key).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		// 依赖 GORM 的 autoCreateTime 和 autoUpdateTime，确保时间字段为零值以便 GORM 自动设置
		if settingEntity.ID == "" {
			settingEntity.ID = uuid.New().String()
		}
		// 清空时间字段，让 GORM 的 autoCreateTime 和 autoUpdateTime 自动设置
		var zeroTime time.Time
		settingEntity.CreatedAt = zeroTime
		settingEntity.UpdatedAt = zeroTime
		return r.db.WithContext(ctx).Create(settingEntity).Error
	} else if err != nil {
		return err
	}

	// 存在，更新记录
	// 使用 Updates 而不是 Save，并 Omit created_at 避免更新创建时间
	settingEntity.ID = existing.ID
	return r.db.WithContext(ctx).
		Model(&entity.SystemSettingEntity{}).
		Where("id = ?", existing.ID).
		Omit("created_at").
		Updates(settingEntity).Error
}

// GetAll 获取组织的所有设置
func (r *SystemSettingRepository) GetAll(ctx context.Context, orgID string) ([]*model.SystemSetting, error) {
	var entities []*entity.SystemSettingEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return mapper.SystemSettingEntitiesToModels(entities), nil
}

// Delete 删除系统设置
func (r *SystemSettingRepository) Delete(ctx context.Context, orgID, key string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND `key` = ?", orgID, key).
		Delete(&entity.SystemSettingEntity{}).Error
}

