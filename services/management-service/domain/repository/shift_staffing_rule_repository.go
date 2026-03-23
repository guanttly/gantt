package repository

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IShiftStaffingRuleRepository 班次排班人数计算规则仓储接口
type IShiftStaffingRuleRepository interface {
	// Create 创建规则
	Create(ctx context.Context, rule *model.ShiftStaffingRule) error

	// Update 更新规则
	Update(ctx context.Context, rule *model.ShiftStaffingRule) error

	// Delete 删除规则
	Delete(ctx context.Context, ruleID string) error

	// GetByID 根据ID获取规则
	GetByID(ctx context.Context, ruleID string) (*model.ShiftStaffingRule, error)

	// GetByShiftID 根据班次ID获取规则（一对一关系）
	GetByShiftID(ctx context.Context, shiftID string) (*model.ShiftStaffingRule, error)

	// List 查询所有规则
	List(ctx context.Context, orgID string) ([]*model.ShiftStaffingRule, error)

	// Exists 检查班次是否已配置规则
	ExistsByShiftID(ctx context.Context, shiftID string) (bool, error)
}
