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

// SchedulingRepository 排班仓储实现
type SchedulingRepository struct {
	db *gorm.DB
}

// NewSchedulingRepository 创建排班仓储实例
func NewSchedulingRepository(db *gorm.DB) repository.ISchedulingRepository {
	return &SchedulingRepository{db: db}
}

// CreateAssignment 创建排班分配
func (r *SchedulingRepository) CreateAssignment(ctx context.Context, assignment *model.ShiftAssignment) error {
	assignmentEntity := mapper.ShiftAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).Create(assignmentEntity).Error
}

// BatchCreateAssignments 批量创建排班分配
func (r *SchedulingRepository) BatchCreateAssignments(ctx context.Context, assignments []*model.ShiftAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	assignmentEntities := make([]*entity.ShiftAssignmentEntity, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentEntities = append(assignmentEntities, mapper.ShiftAssignmentModelToEntity(assignment))
	}

	return r.db.WithContext(ctx).Create(assignmentEntities).Error
}

// UpdateAssignment 更新排班分配
func (r *SchedulingRepository) UpdateAssignment(ctx context.Context, assignment *model.ShiftAssignment) error {
	assignmentEntity := mapper.ShiftAssignmentModelToEntity(assignment)
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", assignment.OrgID, assignment.ID).
		Updates(assignmentEntity).Error
}

// DeleteAssignment 删除排班分配（通过ID）
func (r *SchedulingRepository) DeleteAssignment(ctx context.Context, orgID, assignmentID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, assignmentID).
		Delete(&entity.ShiftAssignmentEntity{}).Error
}

// DeleteByEmployeeAndDate 删除员工在指定日期的排班
func (r *SchedulingRepository) DeleteByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND date = ?", orgID, employeeID, date).
		Delete(&entity.ShiftAssignmentEntity{}).Error
}

// GetAssignmentByID 根据ID获取排班分配
func (r *SchedulingRepository) GetAssignmentByID(ctx context.Context, orgID, assignmentID string) (*model.ShiftAssignment, error) {
	var assignmentEntity entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, assignmentID).
		First(&assignmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntityToModel(&assignmentEntity), nil
}

// GetAssignmentByEmployeeAndDate 获取员工在指定日期的排班
func (r *SchedulingRepository) GetAssignmentByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) (*model.ShiftAssignment, error) {
	var assignmentEntity entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND date = ?", orgID, employeeID, date).
		First(&assignmentEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntityToModel(&assignmentEntity), nil
}

// ListAssignmentsByEmployee 查询员工的排班列表
func (r *SchedulingRepository) ListAssignmentsByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
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

// ListAssignmentsByDateRange 查询日期范围内的所有排班
func (r *SchedulingRepository) ListAssignmentsByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	var assignmentEntities []*entity.ShiftAssignmentEntity
	
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND date BETWEEN ? AND ?", orgID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("date ASC, employee_id ASC").
		Find(&assignmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntitiesToModels(assignmentEntities), nil
}

// ListAssignmentsByShift 查询特定班次的排班列表
func (r *SchedulingRepository) ListAssignmentsByShift(ctx context.Context, orgID, shiftID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	var assignmentEntities []*entity.ShiftAssignmentEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND shift_id = ? AND date BETWEEN ? AND ?", orgID, shiftID, startDate, endDate).
		Order("date ASC, employee_id ASC").
		Find(&assignmentEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftAssignmentEntitiesToModels(assignmentEntities), nil
}

// CountAssignmentsByDate 统计指定日期的排班数量
func (r *SchedulingRepository) CountAssignmentsByDate(ctx context.Context, orgID string, date time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ShiftAssignmentEntity{}).
		Where("org_id = ? AND date = ?", orgID, date).
		Count(&count).Error
	return count, err
}

// CountAssignmentsByShift 统计指定班次在日期范围内的排班数量
func (r *SchedulingRepository) CountAssignmentsByShift(ctx context.Context, orgID, shiftID string, startDate, endDate time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ShiftAssignmentEntity{}).
		Where("org_id = ? AND shift_id = ? AND date BETWEEN ? AND ?", orgID, shiftID, startDate, endDate).
		Count(&count).Error
	return count, err
}
