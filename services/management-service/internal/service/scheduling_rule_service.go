package service

import (
	"context"
	"fmt"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

// SchedulingRuleServiceImpl 排班规则服务实现
type SchedulingRuleServiceImpl struct {
	ruleRepo     repository.ISchedulingRuleRepository
	employeeRepo repository.IEmployeeRepository
	shiftRepo    repository.IShiftRepository
	groupRepo    repository.IGroupRepository
	scopeRepo    repository.IRuleApplyScopeRepository
	logger       logging.ILogger
}

// NewSchedulingRuleService 创建排班规则服务实例
func NewSchedulingRuleService(
	ruleRepo repository.ISchedulingRuleRepository,
	employeeRepo repository.IEmployeeRepository,
	shiftRepo repository.IShiftRepository,
	groupRepo repository.IGroupRepository,
	scopeRepo repository.IRuleApplyScopeRepository,
	logger logging.ILogger,
) service.ISchedulingRuleService {
	return &SchedulingRuleServiceImpl{
		ruleRepo:     ruleRepo,
		employeeRepo: employeeRepo,
		shiftRepo:    shiftRepo,
		groupRepo:    groupRepo,
		scopeRepo:    scopeRepo,
		logger:       logger,
	}
}

// loadApplyScopesForRules 为一组规则批量加载适用范围（V4.1）
func (s *SchedulingRuleServiceImpl) loadApplyScopesForRules(ctx context.Context, orgID string, rules []*model.SchedulingRule) {
	for _, rule := range rules {
		applyScopes, err := s.scopeRepo.GetByRuleID(ctx, orgID, rule.ID)
		if err != nil {
			s.logger.Warn("Failed to load apply scopes for rule", "ruleId", rule.ID, "error", err)
		} else {
			rule.ApplyScopes = applyScopes
		}
	}
}

// loadApplyScopesForRulesMap 为 map 结构的规则批量加载适用范围（V4.1）
func (s *SchedulingRuleServiceImpl) loadApplyScopesForRulesMap(ctx context.Context, orgID string, rulesMap map[string][]*model.SchedulingRule) {
	loaded := make(map[string]bool) // 避免同一规则重复加载
	for _, rules := range rulesMap {
		for _, rule := range rules {
			if loaded[rule.ID] {
				continue
			}
			loaded[rule.ID] = true
			applyScopes, err := s.scopeRepo.GetByRuleID(ctx, orgID, rule.ID)
			if err != nil {
				s.logger.Warn("Failed to load apply scopes for rule", "ruleId", rule.ID, "error", err)
			} else {
				rule.ApplyScopes = applyScopes
			}
		}
	}
}

// CreateRule 创建排班规则
func (s *SchedulingRuleServiceImpl) CreateRule(ctx context.Context, rule *model.SchedulingRule) error {
	if rule.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	// 生成ID
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// 设置默认值
	if rule.Priority == 0 {
		rule.Priority = 0
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// 验证规则
	if err := s.ValidateRule(ctx, rule); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	// 检查冲突
	conflicts, err := s.CheckRuleConflicts(ctx, rule.OrgID, rule)
	if err != nil {
		return fmt.Errorf("check conflicts failed: %w", err)
	}
	if len(conflicts) > 0 {
		s.logger.Warn("Rule has conflicts", "ruleId", rule.ID, "conflicts", conflicts)
		// 注意：这里只是警告，不阻止创建
	}

	// 创建规则
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("Failed to create rule", "error", err)
		return fmt.Errorf("create rule: %w", err)
	}

	// 如果是特定范围规则且有关联，创建关联（兼容旧逻辑）
	if rule.ApplyScope == model.ApplyScopeSpecific && len(rule.Associations) > 0 {
		if err := s.ruleRepo.AddAssociations(ctx, rule.OrgID, rule.ID, rule.Associations); err != nil {
			s.logger.Error("Failed to add associations", "error", err)
			return fmt.Errorf("add associations: %w", err)
		}
	}

	// V4.1新增：保存适用范围（如果有员工/分组范围）
	if len(rule.ApplyScopes) > 0 {
		var scopeAssociations []model.RuleAssociation
		for _, scope := range rule.ApplyScopes {
			if scope.ScopeType == model.ScopeTypeAll {
				continue // 全局不需要关联
			}
			var assocType model.AssociationType
			switch scope.ScopeType {
			case model.ScopeTypeEmployee, model.ScopeTypeExcludeEmployee:
				assocType = model.AssociationTypeEmployee
			case model.ScopeTypeGroup, model.ScopeTypeExcludeGroup:
				assocType = model.AssociationTypeGroup
			default:
				continue
			}
			scopeAssociations = append(scopeAssociations, model.RuleAssociation{
				AssociationType: assocType,
				AssociationID:   scope.ScopeID,
				Role:            scope.ScopeType,
			})
		}
		if len(scopeAssociations) > 0 {
			if err := s.ruleRepo.AddAssociations(ctx, rule.OrgID, rule.ID, scopeAssociations); err != nil {
				s.logger.Error("Failed to add scope associations", "error", err)
				return fmt.Errorf("add scope associations: %w", err)
			}
		}
	}

	s.logger.Info("Rule created successfully", "ruleId", rule.ID, "name", rule.Name)
	return nil
}

// UpdateRule 更新排班规则
func (s *SchedulingRuleServiceImpl) UpdateRule(ctx context.Context, rule *model.SchedulingRule) error {
	if rule.OrgID == "" || rule.ID == "" {
		return fmt.Errorf("orgId and ruleId are required")
	}

	// 检查规则是否存在
	exists, err := s.ruleRepo.Exists(ctx, rule.OrgID, rule.ID)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("rule not found")
	}

	// 验证规则
	if err := s.ValidateRule(ctx, rule); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	rule.UpdatedAt = time.Now()

	// 更新规则
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		s.logger.Error("Failed to update rule", "error", err)
		return fmt.Errorf("update rule: %w", err)
	}

	// 构建最新关联列表：班次关联 + 适用范围关联，直接覆盖旧数据
	var newAssociations []model.RuleAssociation

	// 1. 班次关联（shift）
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift {
			newAssociations = append(newAssociations, assoc)
		}
	}

	// 2. 适用范围关联（employee / group）
	for _, scope := range rule.ApplyScopes {
		if scope.ScopeType == model.ScopeTypeAll {
			continue
		}
		var assocType model.AssociationType
		switch scope.ScopeType {
		case model.ScopeTypeEmployee, model.ScopeTypeExcludeEmployee:
			assocType = model.AssociationTypeEmployee
		case model.ScopeTypeGroup, model.ScopeTypeExcludeGroup:
			assocType = model.AssociationTypeGroup
		default:
			continue
		}
		newAssociations = append(newAssociations, model.RuleAssociation{
			AssociationType: assocType,
			AssociationID:   scope.ScopeID,
			Role:            scope.ScopeType,
		})
	}

	// 先清空再写入，保证幂等
	if err := s.UpdateRuleAssociations(ctx, rule.OrgID, rule.ID, newAssociations); err != nil {
		s.logger.Error("Failed to update rule associations", "error", err)
		return fmt.Errorf("update rule associations: %w", err)
	}

	s.logger.Info("Rule updated successfully", "ruleId", rule.ID)
	return nil
}

// DeleteRule 删除排班规则
func (s *SchedulingRuleServiceImpl) DeleteRule(ctx context.Context, orgID, ruleID string) error {
	if orgID == "" || ruleID == "" {
		return fmt.Errorf("orgId and ruleId are required")
	}

	// 先清空关联
	if err := s.ruleRepo.ClearAssociations(ctx, orgID, ruleID); err != nil {
		s.logger.Error("Failed to clear associations", "error", err)
		return fmt.Errorf("clear associations: %w", err)
	}

	// 删除适用范围
	if err := s.scopeRepo.DeleteByRuleID(ctx, orgID, ruleID); err != nil {
		s.logger.Error("Failed to delete apply scopes", "error", err)
		return fmt.Errorf("delete apply scopes: %w", err)
	}

	// 删除规则
	if err := s.ruleRepo.Delete(ctx, orgID, ruleID); err != nil {
		s.logger.Error("Failed to delete rule", "error", err)
		return fmt.Errorf("delete rule: %w", err)
	}

	s.logger.Info("Rule deleted successfully", "ruleId", ruleID)
	return nil
}

// GetRule 获取规则详情
func (s *SchedulingRuleServiceImpl) GetRule(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error) {
	if orgID == "" || ruleID == "" {
		return nil, fmt.Errorf("orgId and ruleId are required")
	}

	rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
	if err != nil {
		s.logger.Error("Failed to get rule", "error", err)
		return nil, fmt.Errorf("get rule: %w", err)
	}

	return rule, nil
}

// ListRules 查询规则列表
func (s *SchedulingRuleServiceImpl) ListRules(ctx context.Context, filter *model.SchedulingRuleFilter) (*model.SchedulingRuleListResult, error) {
	if filter == nil || filter.OrgID == "" {
		return nil, fmt.Errorf("filter with orgId is required")
	}

	// 设置默认分页
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list rules", "error", err)
		return nil, fmt.Errorf("list rules: %w", err)
	}

	// 为每个规则加载适用范围（V4.1）
	s.loadApplyScopesForRules(ctx, filter.OrgID, result.Items)

	return result, nil
}

// GetActiveRules 获取所有启用的规则
func (s *SchedulingRuleServiceImpl) GetActiveRules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	rules, err := s.ruleRepo.GetActiveRules(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to get active rules", "error", err)
		return nil, fmt.Errorf("get active rules: %w", err)
	}

	return rules, nil
}

// ToggleRuleStatus 切换规则启用状态
func (s *SchedulingRuleServiceImpl) ToggleRuleStatus(ctx context.Context, orgID, ruleID string, isActive bool) error {
	if orgID == "" || ruleID == "" {
		return fmt.Errorf("orgId and ruleId are required")
	}

	rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
	if err != nil {
		return fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("rule not found")
	}

	rule.IsActive = isActive
	rule.UpdatedAt = time.Now()

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		s.logger.Error("Failed to toggle rule status", "error", err)
		return fmt.Errorf("toggle status: %w", err)
	}

	s.logger.Info("Rule status toggled", "ruleId", ruleID, "isActive", isActive)
	return nil
}

// GetRuleAssociations 获取规则的所有关联 (V4.1: 保留用于内部查询)
func (s *SchedulingRuleServiceImpl) GetRuleAssociations(ctx context.Context, orgID, ruleID string) ([]model.RuleAssociation, error) {
	if orgID == "" || ruleID == "" {
		return nil, fmt.Errorf("orgId and ruleId are required")
	}

	associations, err := s.ruleRepo.GetAssociations(ctx, orgID, ruleID)
	if err != nil {
		s.logger.Error("Failed to get associations", "error", err)
		return nil, fmt.Errorf("get associations: %w", err)
	}

	return associations, nil
}

// GetRulesForEmployee 获取某个员工相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.SchedulingRule, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}

	rules, err := s.ruleRepo.GetRulesForEmployee(ctx, orgID, employeeID)
	if err != nil {
		s.logger.Error("Failed to get rules for employee", "error", err)
		return nil, fmt.Errorf("get rules for employee: %w", err)
	}

	s.loadApplyScopesForRules(ctx, orgID, rules)
	return rules, nil
}

// GetRulesForShift 获取某个班次相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error) {
	if orgID == "" || shiftID == "" {
		return nil, fmt.Errorf("orgId and shiftId are required")
	}

	rules, err := s.ruleRepo.GetRulesForShift(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to get rules for shift", "error", err)
		return nil, fmt.Errorf("get rules for shift: %w", err)
	}

	s.loadApplyScopesForRules(ctx, orgID, rules)
	return rules, nil
}

// GetRulesForGroup 获取某个分组相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.SchedulingRule, error) {
	if orgID == "" || groupID == "" {
		return nil, fmt.Errorf("orgId and groupId are required")
	}

	rules, err := s.ruleRepo.GetRulesForGroup(ctx, orgID, groupID)
	if err != nil {
		s.logger.Error("Failed to get rules for group", "error", err)
		return nil, fmt.Errorf("get rules for group: %w", err)
	}

	s.loadApplyScopesForRules(ctx, orgID, rules)
	return rules, nil
}

// GetRulesForEmployees 批量获取多个员工相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if len(employeeIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	rules, err := s.ruleRepo.GetRulesForEmployees(ctx, orgID, employeeIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for employees", "error", err)
		return nil, fmt.Errorf("get rules for employees: %w", err)
	}

	s.loadApplyScopesForRulesMap(ctx, orgID, rules)
	return rules, nil
}

// GetRulesForShifts 批量获取多个班次相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if len(shiftIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	rules, err := s.ruleRepo.GetRulesForShifts(ctx, orgID, shiftIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for shifts", "error", err)
		return nil, fmt.Errorf("get rules for shifts: %w", err)
	}

	s.loadApplyScopesForRulesMap(ctx, orgID, rules)
	return rules, nil
}

// GetRulesForGroups 批量获取多个分组相关的所有规则
func (s *SchedulingRuleServiceImpl) GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if len(groupIDs) == 0 {
		return make(map[string][]*model.SchedulingRule), nil
	}

	rules, err := s.ruleRepo.GetRulesForGroups(ctx, orgID, groupIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for groups", "error", err)
		return nil, fmt.Errorf("get rules for groups: %w", err)
	}

	s.loadApplyScopesForRulesMap(ctx, orgID, rules)
	return rules, nil
}

// UpdateRuleAssociations 更新规则关联（先清空再添加）
func (s *SchedulingRuleServiceImpl) UpdateRuleAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error {
	if orgID == "" || ruleID == "" {
		return fmt.Errorf("orgId and ruleId are required")
	}

	// 清空现有关联
	if err := s.ruleRepo.ClearAssociations(ctx, orgID, ruleID); err != nil {
		return fmt.Errorf("clear associations: %w", err)
	}

	// 添加新关联
	if len(associations) > 0 {
		if err := s.ruleRepo.AddAssociations(ctx, orgID, ruleID, associations); err != nil {
			return fmt.Errorf("add associations: %w", err)
		}
	}

	s.logger.Info("Associations updated", "ruleId", ruleID, "count", len(associations))
	return nil
}

// ==================== 业务验证 ====================

// ValidateRule 验证规则的有效性
func (s *SchedulingRuleServiceImpl) ValidateRule(ctx context.Context, rule *model.SchedulingRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.RuleType == "" {
		return fmt.Errorf("rule type is required")
	}
	if rule.ApplyScope == "" {
		return fmt.Errorf("apply scope is required")
	}
	if rule.TimeScope == "" {
		return fmt.Errorf("time scope is required")
	}

	// 验证数值型参数（仅对使用该字段的规则类型才校验）
	if rule.RuleType == model.RuleTypeMaxCount {
		if rule.MaxCount != nil && *rule.MaxCount <= 0 {
			return fmt.Errorf("maxCount must be positive")
		}
		if rule.ConsecutiveMax != nil && *rule.ConsecutiveMax <= 0 {
			return fmt.Errorf("consecutiveMax must be positive")
		}
		if rule.MinRestDays != nil && *rule.MinRestDays < 0 {
			return fmt.Errorf("minRestDays cannot be negative")
		}
	}
	if rule.RuleType == model.RuleTypePeriodic {
		if rule.IntervalDays != nil && *rule.IntervalDays <= 0 {
			return fmt.Errorf("intervalDays must be positive")
		}
		if rule.MinRestDays != nil && *rule.MinRestDays < 0 {
			return fmt.Errorf("minRestDays cannot be negative")
		}
	}

	// 验证时间范围
	if rule.ValidFrom != nil && rule.ValidTo != nil && rule.ValidFrom.After(*rule.ValidTo) {
		return fmt.Errorf("validFrom must be before validTo")
	}

	// V4.1：验证班次关系（ShiftRelations是新的方式，始终允许）
	// 旧逻辑：全局规则不能有关联（仅针对旧式 employee/group 关联，不包括 shift 关联）
	if rule.ApplyScope == model.ApplyScopeGlobal && len(rule.Associations) > 0 {
		// 检查是否只有班次关联（新V4.1模式允许）
		hasNonShiftAssoc := false
		for _, assoc := range rule.Associations {
			if assoc.AssociationType != model.AssociationTypeShift {
				hasNonShiftAssoc = true
				break
			}
		}
		if hasNonShiftAssoc {
			return fmt.Errorf("global rules cannot have employee/group associations")
		}
	}

	// 特定范围规则必须有关联
	if rule.ApplyScope == model.ApplyScopeSpecific && len(rule.Associations) == 0 && len(rule.ApplyScopes) == 0 {
		s.logger.Warn("Specific rule has no associations", "ruleId", rule.ID)
		// 只警告，不阻止（关联可能稍后添加）
	}

	return nil
}

// CheckRuleConflicts 检查规则冲突
func (s *SchedulingRuleServiceImpl) CheckRuleConflicts(ctx context.Context, orgID string, rule *model.SchedulingRule) ([]string, error) {
	conflicts := []string{}

	// 获取所有启用的规则
	existingRules, err := s.ruleRepo.GetActiveRules(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active rules: %w", err)
	}

	for _, existing := range existingRules {
		// 跳过自己
		if existing.ID == rule.ID {
			continue
		}

		// 检查是否有相同名称的规则
		if existing.Name == rule.Name {
			conflicts = append(conflicts, fmt.Sprintf("规则名称冲突: %s", existing.Name))
		}

		// 检查优先级冲突
		if existing.Priority == rule.Priority && rule.Priority > 0 {
			conflicts = append(conflicts, fmt.Sprintf("优先级冲突: 与规则 %s 优先级相同 (%d)", existing.Name, existing.Priority))
		}

		// 可以添加更多冲突检测逻辑...
	}

	return conflicts, nil
}

// ==================== V4 新增方法 ====================

// ListRulesByCategory 按分类获取规则
func (s *SchedulingRuleServiceImpl) ListRulesByCategory(ctx context.Context, orgID, category string) ([]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}

	filter := &model.SchedulingRuleFilter{
		OrgID:    orgID,
		Category: &category,
		Page:     1,
		PageSize: 10000, // 获取所有
	}

	result, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list rules by category", "error", err)
		return nil, fmt.Errorf("list rules by category: %w", err)
	}

	return result.Items, nil
}

// GetV3Rules 获取所有 V3 规则（待迁移）
func (s *SchedulingRuleServiceImpl) GetV3Rules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	filter := &model.SchedulingRuleFilter{
		OrgID:    orgID,
		Version:  stringPtr("v3"),
		Page:     1,
		PageSize: 10000, // 获取所有
	}

	// 也包含 version 为空或空的规则（视为 V3）
	result, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get V3 rules", "error", err)
		return nil, fmt.Errorf("get V3 rules: %w", err)
	}

	// 过滤出 V3 规则（version 为空、"v3" 或未设置）
	v3Rules := make([]*model.SchedulingRule, 0)
	for _, rule := range result.Items {
		if rule.Version == "" || rule.Version == "v3" {
			v3Rules = append(v3Rules, rule)
		}
	}

	return v3Rules, nil
}

// BatchUpdateVersion 批量更新规则版本
func (s *SchedulingRuleServiceImpl) BatchUpdateVersion(ctx context.Context, orgID string, ruleIDs []string, version string) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if len(ruleIDs) == 0 {
		return fmt.Errorf("ruleIDs is required")
	}
	if version == "" {
		return fmt.Errorf("version is required")
	}

	// 逐个更新
	for _, ruleID := range ruleIDs {
		rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
		if err != nil {
			s.logger.Warn("Failed to get rule for version update", "ruleId", ruleID, "error", err)
			continue
		}
		if rule == nil {
			s.logger.Warn("Rule not found for version update", "ruleId", ruleID)
			continue
		}

		rule.Version = version
		rule.UpdatedAt = time.Now()

		if err := s.ruleRepo.Update(ctx, rule); err != nil {
			s.logger.Error("Failed to update rule version", "ruleId", ruleID, "error", err)
			return fmt.Errorf("update rule version: %w", err)
		}
	}

	s.logger.Info("Batch updated rule versions", "count", len(ruleIDs), "version", version)
	return nil
}

// GetRulesBySubjectShift 获取以指定班次为主体的规则
func (s *SchedulingRuleServiceImpl) GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error) {
	// 获取班次相关的所有规则
	rules, err := s.ruleRepo.GetRulesForShift(ctx, orgID, shiftID)
	if err != nil {
		return nil, err
	}

	// 过滤出主体角色的规则
	var result []*model.SchedulingRule
	for _, rule := range rules {
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeShift &&
				assoc.AssociationID == shiftID &&
				assoc.Role == model.RelationRoleSubject {
				result = append(result, rule)
				break
			}
		}
	}

	return result, nil
}

// GetRulesByObjectShift 获取以指定班次为客体的规则
func (s *SchedulingRuleServiceImpl) GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error) {
	// 获取班次相关的所有规则
	rules, err := s.ruleRepo.GetRulesForShift(ctx, orgID, shiftID)
	if err != nil {
		return nil, err
	}

	// 过滤出客体角色的规则
	var result []*model.SchedulingRule
	for _, rule := range rules {
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeShift &&
				assoc.AssociationID == shiftID &&
				assoc.Role == model.RelationRoleObject {
				result = append(result, rule)
				break
			}
		}
	}

	return result, nil
}

// ============================================================================
// V4.1 新增方法实现：适用范围管理
// ============================================================================

// SetApplyScopes 设置规则的适用范围（替换）
func (s *SchedulingRuleServiceImpl) SetApplyScopes(ctx context.Context, orgID, ruleID string, scopes []model.RuleApplyScope) error {
	return s.scopeRepo.ReplaceScopes(ctx, orgID, ruleID, scopes)
}

// GetApplyScopes 获取规则的适用范围
func (s *SchedulingRuleServiceImpl) GetApplyScopes(ctx context.Context, orgID, ruleID string) ([]model.RuleApplyScope, error) {
	return s.scopeRepo.GetByRuleID(ctx, orgID, ruleID)
}

// ClearApplyScopes 清除规则的适用范围
func (s *SchedulingRuleServiceImpl) ClearApplyScopes(ctx context.Context, orgID, ruleID string) error {
	return s.scopeRepo.DeleteByRuleID(ctx, orgID, ruleID)
}

// GetRulesForEmployeeWithScope 获取适用于指定员工的所有规则（考虑范围）
func (s *SchedulingRuleServiceImpl) GetRulesForEmployeeWithScope(ctx context.Context, orgID, employeeID string, employeeGroupIDs []string) ([]*model.SchedulingRule, error) {
	// 获取所有启用的规则
	allRules, err := s.GetActiveRules(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active rules: %w", err)
	}

	var applicableRules []*model.SchedulingRule
	for _, rule := range allRules {
		applicable, err := s.IsRuleApplicableToEmployee(ctx, orgID, rule.ID, employeeID, employeeGroupIDs)
		if err != nil {
			s.logger.Warn("Failed to check rule applicability", "ruleId", rule.ID, "error", err)
			continue
		}
		if applicable {
			applicableRules = append(applicableRules, rule)
		}
	}

	return applicableRules, nil
}

// IsRuleApplicableToEmployee 判断规则是否适用于指定员工
func (s *SchedulingRuleServiceImpl) IsRuleApplicableToEmployee(ctx context.Context, orgID, ruleID, employeeID string, employeeGroupIDs []string) (bool, error) {
	rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
	if err != nil {
		return false, fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return false, fmt.Errorf("rule not found")
	}

	// 全局规则对所有人生效
	if rule.ApplyScope == model.ApplyScopeGlobal {
		return true, nil
	}

	// 获取规则关联
	associations, err := s.ruleRepo.GetAssociations(ctx, orgID, ruleID)
	if err != nil {
		return false, fmt.Errorf("get associations: %w", err)
	}

	// 检查是否匹配员工或分组
	for _, assoc := range associations {
		if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == employeeID {
			// 检查是否是排除
			if assoc.Role == model.ScopeTypeExcludeEmployee {
				return false, nil
			}
			return true, nil
		}
		if assoc.AssociationType == model.AssociationTypeGroup {
			for _, gid := range employeeGroupIDs {
				if assoc.AssociationID == gid {
					if assoc.Role == model.ScopeTypeExcludeGroup {
						return false, nil
					}
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// ============================================================================
// V4.1 新增方法实现：完整规则加载
// ============================================================================

// GetRuleWithRelations 获取规则详情（包含关联和适用范围）
func (s *SchedulingRuleServiceImpl) GetRuleWithRelations(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return nil, nil
	}

	// 加载关联信息（包含班次关联）
	associations, err := s.ruleRepo.GetAssociations(ctx, orgID, ruleID)
	if err != nil {
		s.logger.Warn("Failed to load associations", "ruleId", ruleID, "error", err)
	} else {
		rule.Associations = associations
	}

	// 加载适用范围
	applyScopes, err := s.GetApplyScopes(ctx, orgID, ruleID)
	if err != nil {
		s.logger.Warn("Failed to load apply scopes", "ruleId", ruleID, "error", err)
	} else {
		rule.ApplyScopes = applyScopes
	}

	return rule, nil
}

// GetActiveRulesWithRelations 获取所有启用的规则（包含关联和适用范围）
func (s *SchedulingRuleServiceImpl) GetActiveRulesWithRelations(ctx context.Context, orgID string) ([]*model.SchedulingRule, error) {
	rules, err := s.GetActiveRules(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// 为每个规则加载关联和范围
	for _, rule := range rules {
		// 加载关联信息
		associations, err := s.ruleRepo.GetAssociations(ctx, orgID, rule.ID)
		if err != nil {
			s.logger.Warn("Failed to load associations", "ruleId", rule.ID, "error", err)
		} else {
			rule.Associations = associations
		}

		// 加载适用范围
		applyScopes, err := s.GetApplyScopes(ctx, orgID, rule.ID)
		if err != nil {
			s.logger.Warn("Failed to load apply scopes", "ruleId", rule.ID, "error", err)
		} else {
			rule.ApplyScopes = applyScopes
		}
	}

	return rules, nil
}
