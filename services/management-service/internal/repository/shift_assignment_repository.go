package repository

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// ShiftAssignmentRepository 班次分配仓储实现
type ShiftAssignmentRepository struct {
	db *gorm.DB
}

// NewShiftAssignmentRepository 创建班次分配仓储实例
func NewShiftAssignmentRepository(db *gorm.DB) repository.IShiftAssignmentRepository {
	return &ShiftAssignmentRepository{db: db}
}

// Create 创建班次分配
func (r *ShiftAssignmentRepository) Create(ctx context.Context, assignment *model.ShiftAssignment) error {
	assignmentEntity := mapper.ShiftAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).Create(assignmentEntity).Error
}

// Update 更新班次分配
func (r *ShiftAssignmentRepository) Update(ctx context.Context, assignment *model.ShiftAssignment) error {
	assignmentEntity := mapper.ShiftAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", assignment.OrgID, assignment.ID).
		Updates(assignmentEntity).Error
}

// Delete 删除班次分配
func (r *ShiftAssignmentRepository) Delete(ctx context.Context, orgID, assignmentID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, assignmentID).
		Delete(&entity.ShiftAssignmentEntity{}).Error
}

// GetByID 根据ID获取班次分配
func (r *ShiftAssignmentRepository) GetByID(ctx context.Context, orgID, assignmentID string) (*model.ShiftAssignment, error) {
	var assignmentEntity entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, assignmentID).
		First(&assignmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntityToModel(&assignmentEntity), nil
}

// GetByEmployeeAndDate 获取员工在指定日期的班次分配
func (r *ShiftAssignmentRepository) GetByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) (*model.ShiftAssignment, error) {
	var assignmentEntity entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND date = ?", orgID, employeeID, date).
		First(&assignmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntityToModel(&assignmentEntity), nil
}

// ListByEmployee 查询员工的班次分配列表
func (r *ShiftAssignmentRepository) ListByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	var assignmentEntities []*entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND date BETWEEN ? AND ?", orgID, employeeID, startDate, endDate).
		Order("date ASC").
		Find(&assignmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntitiesToModels(assignmentEntities), nil
}

// ListByDateRange 查询日期范围内的班次分配
func (r *ShiftAssignmentRepository) ListByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	var assignmentEntities []*entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND date BETWEEN ? AND ?", orgID, startDate, endDate).
		Order("date ASC, employee_id ASC").
		Find(&assignmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntitiesToModels(assignmentEntities), nil
}

// BatchCreate 批量创建班次分配
func (r *ShiftAssignmentRepository) BatchCreate(ctx context.Context, assignments []*model.ShiftAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	// 转换为实体
	assignmentEntities := make([]*entity.ShiftAssignmentEntity, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentEntities = append(assignmentEntities, mapper.ShiftAssignmentModelToEntity(assignment))
	}

	return r.db.WithContext(ctx).Create(&assignmentEntities).Error
}

// UpdateStatus 更新班次分配状态
func (r *ShiftAssignmentRepository) UpdateStatus(ctx context.Context, orgID, assignmentID, status string) error {
	return r.db.WithContext(ctx).Model(&entity.ShiftAssignmentEntity{}).
		Where("org_id = ? AND id = ?", orgID, assignmentID).
		Update("status", status).Error
}
