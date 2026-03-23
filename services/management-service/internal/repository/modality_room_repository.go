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

// ModalityRoomRepository 机房仓储实现
type ModalityRoomRepository struct {
	db *gorm.DB
}

// NewModalityRoomRepository 创建机房仓储实例
func NewModalityRoomRepository(db *gorm.DB) repository.IModalityRoomRepository {
	return &ModalityRoomRepository{db: db}
}

// Create 创建机房
func (r *ModalityRoomRepository) Create(ctx context.Context, modalityRoom *model.ModalityRoom) error {
	modalityRoomEntity := mapper.ModalityRoomModelToEntity(modalityRoom)
	return r.db.WithContext(ctx).Create(modalityRoomEntity).Error
}

// Update 更新机房信息
func (r *ModalityRoomRepository) Update(ctx context.Context, modalityRoom *model.ModalityRoom) error {
	modalityRoomEntity := mapper.ModalityRoomModelToEntity(modalityRoom)
	return r.db.WithContext(ctx).
		Model(&entity.ModalityRoomEntity{}).
		Where("org_id = ? AND id = ?", modalityRoom.OrgID, modalityRoom.ID).
		Omit("created_at").
		Updates(modalityRoomEntity).Error
}

// Delete 删除机房（软删除）
func (r *ModalityRoomRepository) Delete(ctx context.Context, orgID, modalityRoomID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, modalityRoomID).
		Delete(&entity.ModalityRoomEntity{}).Error
}

// GetByID 根据ID获取机房
func (r *ModalityRoomRepository) GetByID(ctx context.Context, orgID, modalityRoomID string) (*model.ModalityRoom, error) {
	var modalityRoomEntity entity.ModalityRoomEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, modalityRoomID).
		First(&modalityRoomEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomEntityToModel(&modalityRoomEntity), nil
}

// GetByCode 根据编码获取机房
func (r *ModalityRoomRepository) GetByCode(ctx context.Context, orgID, code string) (*model.ModalityRoom, error) {
	var modalityRoomEntity entity.ModalityRoomEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&modalityRoomEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomEntityToModel(&modalityRoomEntity), nil
}

// List 查询机房列表
func (r *ModalityRoomRepository) List(ctx context.Context, filter *model.ModalityRoomFilter) (*model.ModalityRoomListResult, error) {
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.ModalityRoomEntity{}).
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
	var modalityRoomEntities []*entity.ModalityRoomEntity
	offset := (filter.Page - 1) * filter.PageSize
	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Order("sort_order ASC, created_at DESC").Find(&modalityRoomEntities).Error; err != nil {
		return nil, err
	}

	return &model.ModalityRoomListResult{
		Items:    mapper.ModalityRoomEntitiesToModels(modalityRoomEntities),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查机房是否存在
func (r *ModalityRoomRepository) Exists(ctx context.Context, orgID, modalityRoomID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ModalityRoomEntity{}).
		Where("org_id = ? AND id = ?", orgID, modalityRoomID).
		Count(&count).Error
	return count > 0, err
}

// BatchGet 批量获取机房
func (r *ModalityRoomRepository) BatchGet(ctx context.Context, orgID string, modalityRoomIDs []string) ([]*model.ModalityRoom, error) {
	var modalityRoomEntities []*entity.ModalityRoomEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id IN ?", orgID, modalityRoomIDs).
		Find(&modalityRoomEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomEntitiesToModels(modalityRoomEntities), nil
}

// GetActiveModalityRooms 获取所有启用的机房
func (r *ModalityRoomRepository) GetActiveModalityRooms(ctx context.Context, orgID string) ([]*model.ModalityRoom, error) {
	var modalityRoomEntities []*entity.ModalityRoomEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("sort_order ASC, created_at DESC").
		Find(&modalityRoomEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ModalityRoomEntitiesToModels(modalityRoomEntities), nil
}
