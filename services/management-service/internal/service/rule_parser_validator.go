package service

import (
	"context"
	"fmt"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// RuleParserValidator 规则解析验证器（三层验证）
type RuleParserValidator struct {
	logger       logging.ILogger
	ruleRepo     repository.ISchedulingRuleRepository
	employeeRepo repository.IEmployeeRepository
	shiftRepo    repository.IShiftRepository
	groupRepo    repository.IGroupRepository
}

// NewRuleParserValidator 创建验证器
func NewRuleParserValidator(
	logger logging.ILogger,
	ruleRepo repository.ISchedulingRuleRepository,
	employeeRepo repository.IEmployeeRepository,
	shiftRepo repository.IShiftRepository,
	groupRepo repository.IGroupRepository,
) *RuleParserValidator {
	return &RuleParserValidator{
		logger:       logger,
		ruleRepo:     ruleRepo,
		employeeRepo: employeeRepo,
		shiftRepo:    shiftRepo,
		groupRepo:    groupRepo,
	}
}

// ValidateThreeLayers 三层验证
func (v *RuleParserValidator) ValidateThreeLayers(
	ctx context.Context,
	orgID string,
	parsedRules []*service.ParsedRule,
) (*ValidationResult, error) {
	result := &ValidationResult{
		Layer1Errors: make([]string, 0),
		Layer2Errors: make([]string, 0),
		Layer3Errors: make([]string, 0),
		Warnings:     make([]string, 0),
	}

	// 第1层：结构完整性验证
	if err := v.validateLayer1Structure(parsedRules); err != nil {
		result.Layer1Errors = append(result.Layer1Errors, err.Error())
		return result, err
	}

	// 第2层：语义一致性验证
	if err := v.validateLayer2Semantic(ctx, orgID, parsedRules); err != nil {
		result.Layer2Errors = append(result.Layer2Errors, err.Error())
		// 第2层错误不阻断，只记录
	}

	// 第3层：业务合理性验证
	if err := v.validateLayer3Business(ctx, orgID, parsedRules); err != nil {
		result.Layer3Errors = append(result.Layer3Errors, err.Error())
		// 第3层错误不阻断，只记录
	}

	return result, nil
}

// ValidationResult 验证结果
type ValidationResult struct {
	Layer1Errors []string // 结构完整性错误（阻断）
	Layer2Errors []string // 语义一致性错误（警告）
	Layer3Errors []string // 业务合理性错误（警告）
	Warnings     []string // 一般警告
}

// validateLayer1Structure 第1层：结构完整性验证
func (v *RuleParserValidator) validateLayer1Structure(parsedRules []*service.ParsedRule) error {
	for _, rule := range parsedRules {
		if rule.Name == "" {
			return fmt.Errorf("规则名称不能为空")
		}
		if rule.Category == "" {
			return fmt.Errorf("规则分类不能为空: %s", rule.Name)
		}
		if rule.RuleType == "" {
			return fmt.Errorf("规则类型不能为空: %s", rule.Name)
		}
		if rule.ApplyScope == "" {
			return fmt.Errorf("应用范围不能为空: %s", rule.Name)
		}
		if rule.TimeScope == "" {
			return fmt.Errorf("时间范围不能为空: %s", rule.Name)
		}

		// 验证数值参数
		if rule.RuleType == model.RuleTypeMaxCount && rule.MaxCount == nil {
			return fmt.Errorf("maxCount类型规则必须指定MaxCount: %s", rule.Name)
		}
		if rule.RuleType == model.RuleTypePeriodic && rule.IntervalDays == nil {
			return fmt.Errorf("periodic类型规则必须指定IntervalDays: %s", rule.Name)
		}

		// V4.1: 验证应用范围（支持新的班次关系和适用范围方式）
		// 如果是 specific 范围，需要有关联对象、班次关系或适用范围
		if rule.ApplyScope == model.ApplyScopeSpecific {
			hasAssociations := len(rule.Associations) > 0
			hasShiftRelations := len(rule.SubjectShifts) > 0 || len(rule.ObjectShifts) > 0 || len(rule.TargetShifts) > 0
			hasScopeEmployees := len(rule.ScopeEmployees) > 0
			hasScopeGroups := len(rule.ScopeGroups) > 0

			if !hasAssociations && !hasShiftRelations && !hasScopeEmployees && !hasScopeGroups {
				return fmt.Errorf("特定范围规则必须指定关联对象或班次关系: %s", rule.Name)
			}
		}

		// 验证分类一致性
		if !v.isCategoryConsistent(rule.Category, rule.RuleType) {
			return fmt.Errorf("规则分类与类型不一致: %s (category=%s, ruleType=%s)", rule.Name, rule.Category, rule.RuleType)
		}
	}
	return nil
}

// validateLayer2Semantic 第2层：语义一致性验证
func (v *RuleParserValidator) validateLayer2Semantic(
	ctx context.Context,
	orgID string,
	parsedRules []*service.ParsedRule,
) error {
	// 检查规则名称是否重复
	ruleNames := make(map[string]bool)
	for _, rule := range parsedRules {
		if ruleNames[rule.Name] {
			return fmt.Errorf("规则名称重复: %s", rule.Name)
		}
		ruleNames[rule.Name] = true
	}

	// 检查关联对象是否存在
	for _, rule := range parsedRules {
		for _, assoc := range rule.Associations {
			if err := v.validateAssociation(ctx, orgID, assoc); err != nil {
				return fmt.Errorf("关联对象验证失败: %s - %w", rule.Name, err)
			}
		}
	}

	return nil
}

// validateLayer3Business 第3层：业务合理性验证
func (v *RuleParserValidator) validateLayer3Business(
	ctx context.Context,
	orgID string,
	parsedRules []*service.ParsedRule,
) error {
	// 检查数值参数的合理性
	for _, rule := range parsedRules {
		if rule.MaxCount != nil && *rule.MaxCount <= 0 {
			return fmt.Errorf("MaxCount必须大于0: %s", rule.Name)
		}
		if rule.ConsecutiveMax != nil && *rule.ConsecutiveMax <= 0 {
			return fmt.Errorf("ConsecutiveMax必须大于0: %s", rule.Name)
		}
		if rule.IntervalDays != nil && *rule.IntervalDays <= 0 {
			return fmt.Errorf("IntervalDays必须大于0: %s", rule.Name)
		}
		if rule.MinRestDays != nil && *rule.MinRestDays < 0 {
			return fmt.Errorf("MinRestDays不能为负数: %s", rule.Name)
		}
	}

	// 检查时间范围合理性
	for _, rule := range parsedRules {
		if rule.ValidFrom != nil && rule.ValidTo != nil {
			if rule.ValidFrom.After(*rule.ValidTo) {
				return fmt.Errorf("ValidFrom不能晚于ValidTo: %s", rule.Name)
			}
		}
	}

	return nil
}

// validateAssociation 验证关联对象
func (v *RuleParserValidator) validateAssociation(
	ctx context.Context,
	orgID string,
	assoc model.RuleAssociation,
) error {
	switch assoc.AssociationType {
	case model.AssociationTypeEmployee:
		employee, err := v.employeeRepo.GetByID(ctx, orgID, assoc.AssociationID)
		if err != nil || employee == nil {
			return fmt.Errorf("员工不存在: %s", assoc.AssociationID)
		}
	case model.AssociationTypeShift:
		shift, err := v.shiftRepo.GetByID(ctx, orgID, assoc.AssociationID)
		if err != nil || shift == nil {
			return fmt.Errorf("班次不存在: %s", assoc.AssociationID)
		}
	case model.AssociationTypeGroup:
		group, err := v.groupRepo.GetByID(ctx, orgID, assoc.AssociationID)
		if err != nil || group == nil {
			return fmt.Errorf("分组不存在: %s", assoc.AssociationID)
		}
	default:
		return fmt.Errorf("未知的关联类型: %s", assoc.AssociationType)
	}
	return nil
}

// isCategoryConsistent 检查分类一致性
func (v *RuleParserValidator) isCategoryConsistent(category string, ruleType model.RuleType) bool {
	constraintTypes := []model.RuleType{
		model.RuleTypeExclusive,
		model.RuleTypeForbiddenDay,
		model.RuleTypeMaxCount,
		model.RuleTypeRequiredTogether,
		model.RuleTypePeriodic,
	}
	preferenceTypes := []model.RuleType{
		model.RuleTypePreferred,
		model.RuleTypeCombinable,
	}

	if category == "constraint" {
		for _, t := range constraintTypes {
			if ruleType == t {
				return true
			}
		}
		return false
	}
	if category == "preference" {
		for _, t := range preferenceTypes {
			if ruleType == t {
				return true
			}
		}
		return false
	}
	// dependency 类型暂时允许所有规则类型
	return true
}
