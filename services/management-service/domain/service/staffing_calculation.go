package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IStaffingCalculationService 排班人数计算服务接口
type IStaffingCalculationService interface {
	// CalculateStaffCount 计算排班人数
	// 根据班次的计算规则，获取最近一周有数据的检查量，计算推荐的排班人数
	// 返回计算预览，包含检查量明细、计算过程、推荐人数
	CalculateStaffCount(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error)

	// ApplyStaffCount 应用排班人数
	// 根据用户确认或调整后的值，更新班次的默认人数
	// 支持更新通用默认值或按周几更新
	ApplyStaffCount(ctx context.Context, orgID string, req *model.ApplyStaffCountRequest) (*model.ApplyStaffCountResult, error)

	// GetShiftStaffingRule 获取班次的计算规则
	GetShiftStaffingRule(ctx context.Context, shiftID string) (*model.ShiftStaffingRule, error)

	// GetStaffingRuleByID 根据规则ID获取计算规则
	GetStaffingRuleByID(ctx context.Context, ruleID string) (*model.ShiftStaffingRule, error)

	// CreateOrUpdateStaffingRule 创建或更新班次计算规则
	CreateOrUpdateStaffingRule(ctx context.Context, rule *model.ShiftStaffingRule) error

	// DeleteStaffingRule 删除班次计算规则
	DeleteStaffingRule(ctx context.Context, ruleID string) error

	// ListStaffingRules 查询所有计算规则
	ListStaffingRules(ctx context.Context, orgID string) ([]*model.ShiftStaffingRule, error)

	// GetWeeklyStaffConfig 获取班次的周默认人数配置
	GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*model.WeeklyStaffConfig, error)

	// SetWeeklyStaffConfig 设置班次的周默认人数配置
	SetWeeklyStaffConfig(ctx context.Context, shiftID string, weeklyConfig []model.WeekdayStaff) error
}
