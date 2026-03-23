package repository

import (
	"context"
	"fmt"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// LeaveRepository 假期记录仓储实现
type LeaveRepository struct {
	db *gorm.DB
}

// NewLeaveRepository 创建假期记录仓储实例
func NewLeaveRepository(db *gorm.DB) repository.ILeaveRepository {
	return &LeaveRepository{db: db}
}

// Create 创建假期记录
func (r *LeaveRepository) Create(ctx context.Context, leave *model.LeaveRecord) error {
	leaveEntity := mapper.LeaveRecordModelToEntity(leave)
	return r.db.WithContext(ctx).Create(leaveEntity).Error
}

// Update 更新假期记录
func (r *LeaveRepository) Update(ctx context.Context, leave *model.LeaveRecord) error {
	leaveEntity := mapper.LeaveRecordModelToEntity(leave)
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", leave.OrgID, leave.ID).
		Updates(leaveEntity).Error
}

// Delete 删除假期记录（软删除）
func (r *LeaveRepository) Delete(ctx context.Context, orgID, leaveID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, leaveID).
		Delete(&entity.LeaveRecordEntity{}).Error
}

// GetByID 根据ID获取假期记录
func (r *LeaveRepository) GetByID(ctx context.Context, orgID, leaveID string) (*model.LeaveRecord, error) {
	var leaveEntity entity.LeaveRecordEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, leaveID).
		First(&leaveEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.LeaveRecordEntityToModel(&leaveEntity), nil
}

// List 查询假期记录列表
func (r *LeaveRepository) List(ctx context.Context, filter *model.LeaveFilter) (*model.LeaveListResult, error) {
	if filter == nil || filter.OrgID == "" {
		return nil, fmt.Errorf("filter and org_id are required")
	}

	query := r.db.WithContext(ctx).Model(&entity.LeaveRecordEntity{}).
		Where("leave_records.org_id = ?", filter.OrgID)

	// 应用过滤条件
	if filter.EmployeeID != nil && *filter.EmployeeID != "" {
		query = query.Where("leave_records.employee_id = ?", *filter.EmployeeID)
	}

	// 按员工姓名或工号搜索
	if filter.Keyword != "" {
		query = query.Joins("LEFT JOIN employees ON employees.id = leave_records.employee_id").
			Where("employees.name LIKE ? OR employees.employee_id LIKE ?",
				"%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	if filter.Type != nil && *filter.Type != "" {
		query = query.Where("leave_records.type = ?", *filter.Type)
	}
	if filter.StartDate != nil {
		query = query.Where("leave_records.start_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("leave_records.end_date <= ?", *filter.EndDate)
	}

	// 统计总数
	var total int64
	countQuery := r.db.WithContext(ctx).Model(&entity.LeaveRecordEntity{}).
		Where("leave_records.org_id = ?", filter.OrgID)

	if filter.EmployeeID != nil && *filter.EmployeeID != "" {
		countQuery = countQuery.Where("leave_records.employee_id = ?", *filter.EmployeeID)
	}
	if filter.Keyword != "" {
		countQuery = countQuery.Joins("LEFT JOIN employees ON employees.id = leave_records.employee_id").
			Where("employees.name LIKE ? OR employees.employee_id LIKE ?",
				"%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}
	if filter.Type != nil && *filter.Type != "" {
		countQuery = countQuery.Where("leave_records.type = ?", *filter.Type)
	}
	if filter.StartDate != nil {
		countQuery = countQuery.Where("leave_records.start_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		countQuery = countQuery.Where("leave_records.end_date <= ?", *filter.EndDate)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var leaveEntities []*entity.LeaveRecordEntity
	offset := (filter.Page - 1) * filter.PageSize
	err := query.Offset(offset).Limit(filter.PageSize).
		Order("leave_records.created_at DESC").
		Find(&leaveEntities).Error
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	leaves := mapper.LeaveRecordEntitiesToModels(leaveEntities)

	return &model.LeaveListResult{
		Items:    leaves,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// ListByEmployee 查询员工的假期记录
func (r *LeaveRepository) ListByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate *time.Time) ([]*model.LeaveRecord, error) {
	query := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ?", orgID, employeeID)

	if startDate != nil {
		query = query.Where("start_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("end_date <= ?", *endDate)
	}

	var leaveEntities []*entity.LeaveRecordEntity
	err := query.Order("start_date DESC").Find(&leaveEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.LeaveRecordEntitiesToModels(leaveEntities), nil
}

// CheckConflict 检查假期冲突
func (r *LeaveRepository) CheckConflict(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time, excludeLeaveID *string) (bool, error) {
	query := r.db.WithContext(ctx).Model(&entity.LeaveRecordEntity{}).
		Where("org_id = ? AND employee_id = ?", orgID, employeeID).
		Where("(start_date <= ? AND end_date >= ?) OR (start_date <= ? AND end_date >= ?) OR (start_date >= ? AND end_date <= ?)",
			endDate, startDate, // 覆盖开始日期
			startDate, endDate, // 覆盖结束日期
			startDate, endDate, // 完全包含
		)

	if excludeLeaveID != nil && *excludeLeaveID != "" {
		query = query.Where("id != ?", *excludeLeaveID)
	}

	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

// GetActiveLeaves 获取员工当前有效的假期
func (r *LeaveRepository) GetActiveLeaves(ctx context.Context, orgID, employeeID string) ([]*model.LeaveRecord, error) {
	var leaveEntities []*entity.LeaveRecordEntity
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ?", orgID, employeeID).
		Where("end_date >= ?", now).
		Order("start_date ASC").
		Find(&leaveEntities).Error

	if err != nil {
		return nil, err
	}
	return mapper.LeaveRecordEntitiesToModels(leaveEntities), nil
}

// GetStatistics 获取假期统计
func (r *LeaveRepository) GetStatistics(ctx context.Context, orgID string, year int) (map[model.LeaveType]int64, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.Local)

	var results []struct {
		Type  model.LeaveType
		Count int64
	}

	err := r.db.WithContext(ctx).Model(&entity.LeaveRecordEntity{}).
		Select("type, COUNT(*) as count").
		Where("org_id = ? AND start_date >= ? AND end_date <= ?", orgID, startDate, endDate).
		Group("type").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	statistics := make(map[model.LeaveType]int64)
	for _, result := range results {
		statistics[result.Type] = result.Count
	}

	return statistics, nil
}
