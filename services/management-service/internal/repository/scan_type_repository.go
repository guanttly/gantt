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

// ScanTypeRepository 检查类型仓储实现
type ScanTypeRepository struct {
	db *gorm.DB
}

// NewScanTypeRepository 创建检查类型仓储实例
func NewScanTypeRepository(db *gorm.DB) repository.IScanTypeRepository {
	return &ScanTypeRepository{db: db}
}

// Create 创建检查类型
func (r *ScanTypeRepository) Create(ctx context.Context, scanType *model.ScanType) error {
	scanTypeEntity := mapper.ScanTypeModelToEntity(scanType)
	return r.db.WithContext(ctx).Create(scanTypeEntity).Error
}

// Update 更新检查类型信息
func (r *ScanTypeRepository) Update(ctx context.Context, scanType *model.ScanType) error {
	scanTypeEntity := mapper.ScanTypeModelToEntity(scanType)
	return r.db.WithContext(ctx).
		Model(&entity.ScanTypeEntity{}).
		Where("org_id = ? AND id = ?", scanType.OrgID, scanType.ID).
		Omit("created_at").
		Updates(scanTypeEntity).Error
}

// Delete 删除检查类型（软删除）
func (r *ScanTypeRepository) Delete(ctx context.Context, orgID, scanTypeID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, scanTypeID).
		Delete(&entity.ScanTypeEntity{}).Error
}

// GetByID 根据ID获取检查类型
func (r *ScanTypeRepository) GetByID(ctx context.Context, orgID, scanTypeID string) (*model.ScanType, error) {
	var scanTypeEntity entity.ScanTypeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, scanTypeID).
		First(&scanTypeEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ScanTypeEntityToModel(&scanTypeEntity), nil
}

// GetByCode 根据编码获取检查类型
func (r *ScanTypeRepository) GetByCode(ctx context.Context, orgID, code string) (*model.ScanType, error) {
	var scanTypeEntity entity.ScanTypeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&scanTypeEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ScanTypeEntityToModel(&scanTypeEntity), nil
}

// List 查询检查类型列表
func (r *ScanTypeRepository) List(ctx context.Context, filter *model.ScanTypeFilter) (*model.ScanTypeListResult, error) {
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.ScanTypeEntity{}).
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
	var scanTypeEntities []*entity.ScanTypeEntity
	offset := (filter.Page - 1) * filter.PageSize
	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Order("sort_order ASC, created_at DESC").Find(&scanTypeEntities).Error; err != nil {
		return nil, err
	}

	return &model.ScanTypeListResult{
		Items:    mapper.ScanTypeEntitiesToModels(scanTypeEntities),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查检查类型是否存在
func (r *ScanTypeRepository) Exists(ctx context.Context, orgID, scanTypeID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ScanTypeEntity{}).
		Where("org_id = ? AND id = ?", orgID, scanTypeID).
		Count(&count).Error
	return count > 0, err
}

// GetActiveScanTypes 获取所有启用的检查类型
func (r *ScanTypeRepository) GetActiveScanTypes(ctx context.Context, orgID string) ([]*model.ScanType, error) {
	var scanTypeEntities []*entity.ScanTypeEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("sort_order ASC, created_at DESC").
		Find(&scanTypeEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ScanTypeEntitiesToModels(scanTypeEntities), nil
}
