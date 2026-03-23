package service

import (
	"context"

	"jusha/agent/rostering/domain/model"
)

// ============================================================
// 规则级校验器接口（领域服务）
// ============================================================

// IRuleLevelValidator 规则级校验器接口
// 负责对排班草案进行规则级校验，包括人数、班次规则、规则合规性等
type IRuleLevelValidator interface {
	// ValidateStaffCount 人数校验（仅超配检测，用于渐进式中途校验）
	// 检查排班草案中每日的人数是否超过需求（允许不足）
	ValidateStaffCount(ctx context.Context, scheduleDraft *model.ShiftScheduleDraft, staffRequirements map[string]int) (*model.RuleValidationResult, error)

	// ValidateStaffCountStrict 严格人数校验（用于最终校验）
	// 检查排班草案中每日的人数是否严格等于需求
	// 返回详细的缺员信息（班次、日期、缺员数）
	ValidateStaffCountStrict(ctx context.Context, scheduleDraft *model.ShiftScheduleDraft, staffRequirements map[string]int) (*model.RuleValidationResult, error)

	// ValidateShiftRules 班次规则校验
	// 检查排班草案是否符合班次相关规则（如人员冲突、占位冲突等）
	ValidateShiftRules(ctx context.Context, scheduleDraft *model.ShiftScheduleDraft, shifts []*model.Shift, occupiedSlots map[string]map[string]string) (*model.RuleValidationResult, error)

	// ValidateRuleCompliance 规则合规性校验
	// 检查排班草案是否符合业务规则（如最大排班次数、连续工作天数等）
	// fixedShiftAssignments: date -> []staffID，用于标识哪些人员是固定排班，对固定排班人员给予规则豁免
	ValidateRuleCompliance(ctx context.Context, scheduleDraft *model.ShiftScheduleDraft, rules []*model.Rule, staffList []*model.Employee, fixedShiftAssignments map[string][]string) (*model.RuleValidationResult, error)

	// ValidateAll 综合校验（人数、班次、规则合规性）
	// 执行所有校验项并返回综合结果
	// fixedShiftAssignments: date -> []staffID，用于标识哪些人员是固定排班，校验时给予豁免或更宽松的检查
	ValidateAll(ctx context.Context, scheduleDraft *model.ShiftScheduleDraft, staffRequirements map[string]int, shifts []*model.Shift, rules []*model.Rule, staffList []*model.Employee, occupiedSlots map[string]map[string]string, fixedShiftAssignments map[string][]string) (*model.RuleValidationResult, error)
}
