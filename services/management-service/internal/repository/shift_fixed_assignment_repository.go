package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// ShiftFixedAssignmentRepository 班次固定人员配置仓储实现
type ShiftFixedAssignmentRepository struct {
	db *gorm.DB
}

// NewShiftFixedAssignmentRepository 创建班次固定人员配置仓储实例
func NewShiftFixedAssignmentRepository(db *gorm.DB) repository.IShiftFixedAssignmentRepository {
	return &ShiftFixedAssignmentRepository{db: db}
}

// Create 创建固定人员配置
func (r *ShiftFixedAssignmentRepository) Create(ctx context.Context, assignment *model.ShiftFixedAssignment) error {
	assignmentEntity := mapper.ShiftFixedAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).Create(assignmentEntity).Error
}

// BatchCreate 批量创建固定人员配置
func (r *ShiftFixedAssignmentRepository) BatchCreate(ctx context.Context, assignments []*model.ShiftFixedAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	entities := make([]*entity.ShiftFixedAssignmentEntity, 0, len(assignments))
	for _, assignment := range assignments {
		entities = append(entities, mapper.ShiftFixedAssignmentModelToEntity(assignment))
	}

	return r.db.WithContext(ctx).Create(&entities).Error
}

// Update 更新固定人员配置
func (r *ShiftFixedAssignmentRepository) Update(ctx context.Context, id string, assignment *model.ShiftFixedAssignment) error {
	assignmentEntity := mapper.ShiftFixedAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).
		Model(&entity.ShiftFixedAssignmentEntity{}).
		Where("id = ?", id).
		Omit("created_at").
		Select("*").
		Updates(assignmentEntity).Error
}

// Delete 软删除固定人员配置
func (r *ShiftFixedAssignmentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entity.ShiftFixedAssignmentEntity{}).Error
}

// GetByID 根据ID获取固定人员配置
func (r *ShiftFixedAssignmentRepository) GetByID(ctx context.Context, id string) (*model.ShiftFixedAssignment, error) {
	var assignmentEntity entity.ShiftFixedAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&assignmentEntity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return mapper.ShiftFixedAssignmentEntityToModel(&assignmentEntity), nil
}

// ListByShiftID 获取班次的所有固定人员配置
func (r *ShiftFixedAssignmentRepository) ListByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error) {
	var entities []*entity.ShiftFixedAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("shift_id = ?", shiftID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return mapper.ShiftFixedAssignmentEntitiesToModels(entities), nil
}

// ListByStaffID 获取人员的所有固定班次配置
func (r *ShiftFixedAssignmentRepository) ListByStaffID(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error) {
	var entities []*entity.ShiftFixedAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("staff_id = ?", staffID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return mapper.ShiftFixedAssignmentEntitiesToModels(entities), nil
}

// ListByShiftIDs 批量获取多个班次的所有固定人员配置
func (r *ShiftFixedAssignmentRepository) ListByShiftIDs(ctx context.Context, shiftIDs []string) (map[string][]*model.ShiftFixedAssignment, error) {
	if len(shiftIDs) == 0 {
		return make(map[string][]*model.ShiftFixedAssignment), nil
	}

	var entities []*entity.ShiftFixedAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("shift_id IN ?", shiftIDs).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	// 按 shiftID 分组
	result := make(map[string][]*model.ShiftFixedAssignment)
	for _, entity := range entities {
		model := mapper.ShiftFixedAssignmentEntityToModel(entity)
		result[entity.ShiftID] = append(result[entity.ShiftID], model)
	}

	// 确保所有 shiftID 都有条目（即使为空数组）
	for _, shiftID := range shiftIDs {
		if _, ok := result[shiftID]; !ok {
			result[shiftID] = []*model.ShiftFixedAssignment{}
		}
	}

	return result, nil
}

// DeleteByShiftID 删除班次的所有固定人员配置（级联删除）
func (r *ShiftFixedAssignmentRepository) DeleteByShiftID(ctx context.Context, shiftID string) error {
	return r.db.WithContext(ctx).
		Where("shift_id = ?", shiftID).
		Delete(&entity.ShiftFixedAssignmentEntity{}).Error
}

// Exists 检查固定人员配置是否存在
func (r *ShiftFixedAssignmentRepository) Exists(ctx context.Context, id string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ShiftFixedAssignmentEntity{}).
		Where("id = ?", id).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

