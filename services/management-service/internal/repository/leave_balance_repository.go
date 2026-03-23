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

// LeaveBalanceRepository 假期余额仓储实现
type LeaveBalanceRepository struct {
	db *gorm.DB
}

// NewLeaveBalanceRepository 创建假期余额仓储实例
func NewLeaveBalanceRepository(db *gorm.DB) repository.ILeaveBalanceRepository {
	return &LeaveBalanceRepository{db: db}
}

// Create 创建假期余额
func (r *LeaveBalanceRepository) Create(ctx context.Context, balance *model.LeaveBalance) error {
	balanceEntity := mapper.LeaveBalanceModelToEntity(balance)
	return r.db.WithContext(ctx).Create(balanceEntity).Error
}

// Update 更新假期余额
func (r *LeaveBalanceRepository) Update(ctx context.Context, balance *model.LeaveBalance) error {
	balanceEntity := mapper.LeaveBalanceModelToEntity(balance)
	return r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND type = ? AND year = ?",
			balance.OrgID, balance.EmployeeID, balance.Type, balance.Year).
		Updates(balanceEntity).Error
}

// GetByEmployeeAndType 获取员工某类型的假期余额
func (r *LeaveBalanceRepository) GetByEmployeeAndType(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int) (*model.LeaveBalance, error) {
	var balanceEntity entity.LeaveBalanceEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND type = ? AND year = ?",
			orgID, employeeID, leaveType, year).
		First(&balanceEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.LeaveBalanceEntityToModel(&balanceEntity), nil
}

// ListByEmployee 获取员工的所有假期余额
func (r *LeaveBalanceRepository) ListByEmployee(ctx context.Context, orgID, employeeID string, year int) ([]*model.LeaveBalance, error) {
	var balanceEntities []*entity.LeaveBalanceEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND employee_id = ? AND year = ?", orgID, employeeID, year).
		Order("type ASC").
		Find(&balanceEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.LeaveBalanceEntitiesToModels(balanceEntities), nil
}

// DeductBalance 扣减假期余额
func (r *LeaveBalanceRepository) DeductBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, days float64) error {
	if days <= 0 {
		return fmt.Errorf("days must be positive")
	}

	return r.db.WithContext(ctx).Model(&entity.LeaveBalanceEntity{}).
		Where("org_id = ? AND employee_id = ? AND type = ? AND year = ?",
			orgID, employeeID, leaveType, year).
		Updates(map[string]interface{}{
			"used":      gorm.Expr("used + ?", days),
			"remaining": gorm.Expr("remaining - ?", days),
		}).Error
}

// AddBalance 增加假期余额
func (r *LeaveBalanceRepository) AddBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, days float64) error {
	if days <= 0 {
		return fmt.Errorf("days must be positive")
	}

	return r.db.WithContext(ctx).Model(&entity.LeaveBalanceEntity{}).
		Where("org_id = ? AND employee_id = ? AND type = ? AND year = ?",
			orgID, employeeID, leaveType, year).
		Updates(map[string]interface{}{
			"used":      gorm.Expr("used - ?", days),
			"remaining": gorm.Expr("remaining + ?", days),
		}).Error
}

// InitializeBalance 初始化员工的假期余额
func (r *LeaveBalanceRepository) InitializeBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, totalDays float64) error {
	if totalDays < 0 {
		return fmt.Errorf("total days cannot be negative")
	}

	// 检查是否已存在
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.LeaveBalanceEntity{}).
		Where("org_id = ? AND employee_id = ? AND type = ? AND year = ?",
			orgID, employeeID, leaveType, year).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("balance already exists")
	}

	// 创建新余额记录
	balance := &model.LeaveBalance{
		OrgID:      orgID,
		EmployeeID: employeeID,
		Type:       leaveType,
		Year:       year,
		Total:      totalDays,
		Used:       0,
		Remaining:  totalDays,
	}

	balanceEntity := mapper.LeaveBalanceModelToEntity(balance)
	return r.db.WithContext(ctx).Create(balanceEntity).Error
}
