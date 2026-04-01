package rule

import (
	"context"
	"strings"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 规则数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建规则仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建规则。
func (r *Repository) Create(ctx context.Context, rule *Rule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据 ID 查询规则。
func (r *Repository) GetByID(ctx context.Context, id string) (*Rule, error) {
	var rule Rule
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetByIDAnyScope 根据 ID 查询规则，跳过租户范围限制。
func (r *Repository) GetByIDAnyScope(ctx context.Context, id string) (*Rule, error) {
	var rule Rule
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Update 更新规则。
func (r *Repository) Update(ctx context.Context, rule *Rule) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(rule).Error
}

// Delete 删除规则（硬删除，级联删除关联）。
func (r *Repository) Delete(ctx context.Context, id string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		Delete(&Rule{}).Error
}

// List 查询当前节点的规则列表。
func (r *Repository) List(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("disabled = ?", false).
		Order("category ASC, priority ASC").
		Find(&rules).Error
	return rules, err
}

// ListByNodeID 查询指定节点的规则列表（不使用 tenant scope，用于继承计算）。
func (r *Repository) ListByNodeID(ctx context.Context, nodeID string) ([]Rule, error) {
	var rules []Rule
	err := r.db.WithContext(ctx).
		Where("org_node_id = ? AND is_enabled = ? AND disabled = ?", nodeID, true, false).
		Order("category ASC, priority ASC").
		Find(&rules).Error
	return rules, err
}

// ListByNodeIDs 批量查询多个节点的规则列表（用于继承计算优化）。
func (r *Repository) ListByNodeIDs(ctx context.Context, nodeIDs []string) ([]Rule, error) {
	var rules []Rule
	err := r.db.WithContext(ctx).
		Where("org_node_id IN ? AND is_enabled = ?", nodeIDs, true).
		Order("category ASC, priority ASC").
		Find(&rules).Error
	return rules, err
}

// GetByOverrideRuleID 查询覆盖某规则的子规则。
func (r *Repository) GetByOverrideRuleID(ctx context.Context, overrideRuleID string) ([]Rule, error) {
	var rules []Rule
	err := r.db.WithContext(ctx).
		Where("override_rule_id = ?", overrideRuleID).
		Find(&rules).Error
	return rules, err
}

// GetByNodeAndOverrideRuleID 查询某节点下对指定上级规则的本级覆盖或禁用记录。
func (r *Repository) GetByNodeAndOverrideRuleID(ctx context.Context, nodeID, overrideRuleID string) (*Rule, error) {
	var rule Rule
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("org_node_id = ? AND override_rule_id = ?", nodeID, overrideRuleID).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListDisabledByNodeID 查询某节点创建的禁用继承标记。
func (r *Repository) ListDisabledByNodeID(ctx context.Context, nodeID string) ([]Rule, error) {
	var rules []Rule
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("org_node_id = ? AND disabled = ?", nodeID, true).
		Order("category ASC, priority ASC, created_at ASC").
		Find(&rules).Error
	return rules, err
}

// ── 规则关联 ──────────────────────────────

// CreateAssociation 创建规则关联。
func (r *Repository) CreateAssociation(ctx context.Context, assoc *RuleAssociation) error {
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(assoc).Error
}

// DeleteAssociationsByRule 删除某规则的所有关联。
func (r *Repository) DeleteAssociationsByRule(ctx context.Context, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("rule_id = ?", ruleID).
		Delete(&RuleAssociation{}).Error
}

// ListAssociationsByRule 查询某规则的关联列表。
func (r *Repository) ListAssociationsByRule(ctx context.Context, ruleID string) ([]RuleAssociation, error) {
	var assocs []RuleAssociation
	err := r.db.WithContext(ctx).
		Where("rule_id = ?", ruleID).
		Find(&assocs).Error
	for i := range assocs {
		assocs[i].NormalizeAliases()
	}
	return assocs, err
}

// ListAssociationsByTarget 查询关联到指定目标的规则 ID 列表。
func (r *Repository) ListAssociationsByTarget(ctx context.Context, targetType, targetID string) ([]RuleAssociation, error) {
	var assocs []RuleAssociation
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Find(&assocs).Error
	return assocs, err
}

// BatchCreateAssociations 批量创建规则关联。
func (r *Repository) BatchCreateAssociations(ctx context.Context, assocs []RuleAssociation) error {
	if len(assocs) == 0 {
		return nil
	}
	for i := range assocs {
		if assocs[i].ID == "" {
			assocs[i].ID = uuid.New().String()
		}
	}
	return r.db.WithContext(ctx).Create(&assocs).Error
}

// ReplaceAssociationsByRule 替换规则关联。
func (r *Repository) ReplaceAssociationsByRule(ctx context.Context, ruleID string, assocs []RuleAssociation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := &Repository{db: tx}
		if err := repo.DeleteAssociationsByRule(ctx, ruleID); err != nil {
			return err
		}
		if len(assocs) == 0 {
			return nil
		}
		return repo.BatchCreateAssociations(ctx, assocs)
	})
}

// ListAssociationsByRuleIDs 按规则批量查询关联。
func (r *Repository) ListAssociationsByRuleIDs(ctx context.Context, ruleIDs []string) ([]RuleAssociation, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var assocs []RuleAssociation
	err := r.db.WithContext(ctx).
		Where("rule_id IN ?", ruleIDs).
		Order("created_at ASC").
		Find(&assocs).Error
	for i := range assocs {
		assocs[i].NormalizeAliases()
	}
	return assocs, err
}

// CreateApplyScopes 批量创建规则适用范围。
func (r *Repository) CreateApplyScopes(ctx context.Context, scopes []RuleApplyScope) error {
	if len(scopes) == 0 {
		return nil
	}
	for i := range scopes {
		if scopes[i].ID == "" {
			scopes[i].ID = uuid.New().String()
		}
	}
	return r.db.WithContext(ctx).Create(&scopes).Error
}

// DeleteApplyScopesByRule 删除某规则的适用范围。
func (r *Repository) DeleteApplyScopesByRule(ctx context.Context, ruleID string) error {
	return r.db.WithContext(ctx).
		Where("rule_id = ?", ruleID).
		Delete(&RuleApplyScope{}).Error
}

// ReplaceApplyScopesByRule 替换规则适用范围。
func (r *Repository) ReplaceApplyScopesByRule(ctx context.Context, ruleID string, scopes []RuleApplyScope) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := &Repository{db: tx}
		if err := repo.DeleteApplyScopesByRule(ctx, ruleID); err != nil {
			return err
		}
		if len(scopes) == 0 {
			return nil
		}
		return repo.CreateApplyScopes(ctx, scopes)
	})
}

// ListApplyScopesByRule 查询某规则的适用范围。
func (r *Repository) ListApplyScopesByRule(ctx context.Context, ruleID string) ([]RuleApplyScope, error) {
	var scopes []RuleApplyScope
	err := r.db.WithContext(ctx).
		Where("rule_id = ?", ruleID).
		Order("created_at ASC").
		Find(&scopes).Error
	return scopes, err
}

// ListApplyScopesByRuleIDs 按规则批量查询适用范围。
func (r *Repository) ListApplyScopesByRuleIDs(ctx context.Context, ruleIDs []string) ([]RuleApplyScope, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var scopes []RuleApplyScope
	err := r.db.WithContext(ctx).
		Where("rule_id IN ?", ruleIDs).
		Order("created_at ASC").
		Find(&scopes).Error
	return scopes, err
}

// BatchCreateDependencies 批量创建规则依赖关系。
func (r *Repository) BatchCreateDependencies(ctx context.Context, deps []RuleDependency) error {
	if len(deps) == 0 {
		return nil
	}
	for i := range deps {
		if deps[i].ID == "" {
			deps[i].ID = uuid.New().String()
		}
	}
	return r.db.WithContext(ctx).Create(&deps).Error
}

// ListDependenciesByRuleIDs 查询与指定规则相关的依赖关系。
func (r *Repository) ListDependenciesByRuleIDs(ctx context.Context, ruleIDs []string) ([]RuleDependency, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var deps []RuleDependency
	err := r.db.WithContext(ctx).
		Where("dependent_rule_id IN ? OR dependent_on_rule_id IN ?", ruleIDs, ruleIDs).
		Order("created_at ASC").
		Find(&deps).Error
	return deps, err
}

// BatchCreateConflicts 批量创建规则冲突关系。
func (r *Repository) BatchCreateConflicts(ctx context.Context, conflicts []RuleConflict) error {
	if len(conflicts) == 0 {
		return nil
	}
	for i := range conflicts {
		if conflicts[i].ID == "" {
			conflicts[i].ID = uuid.New().String()
		}
	}
	return r.db.WithContext(ctx).Create(&conflicts).Error
}

// ListConflictsByRuleIDs 查询与指定规则相关的冲突关系。
func (r *Repository) ListConflictsByRuleIDs(ctx context.Context, ruleIDs []string) ([]RuleConflict, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var conflicts []RuleConflict
	err := r.db.WithContext(ctx).
		Where("rule_id_1 IN ? OR rule_id_2 IN ?", ruleIDs, ruleIDs).
		Order("created_at ASC").
		Find(&conflicts).Error
	return conflicts, err
}

// ResolveShiftIDsByNames 根据班次名称解析ID。
func (r *Repository) ResolveShiftIDsByNames(ctx context.Context, names []string) (map[string]string, error) {
	return r.resolveIDsByName(ctx, "shifts", names)
}

// ResolveShiftIDsByCodes 根据班次短代号解析ID。
func (r *Repository) ResolveShiftIDsByCodes(ctx context.Context, codes []string) (map[string]string, error) {
	return r.resolveIDsByColumn(ctx, "shifts", "code", codes)
}

// ResolveShiftIDsByIDs 校验并保留已存在的班次 ID。
func (r *Repository) ResolveShiftIDsByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return r.resolveIDsByColumn(ctx, "shifts", "id", ids)
}

// ResolveEmployeeIDsByNames 根据员工名称解析ID。
func (r *Repository) ResolveEmployeeIDsByNames(ctx context.Context, names []string) (map[string]string, error) {
	return r.resolveIDsByName(ctx, "employees", names)
}

// ResolveEmployeeIDsByIDs 校验并保留已存在的员工 ID。
func (r *Repository) ResolveEmployeeIDsByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return r.resolveIDsByColumn(ctx, "employees", "id", ids)
}

// ResolveGroupIDsByNames 根据分组名称解析ID。
func (r *Repository) ResolveGroupIDsByNames(ctx context.Context, names []string) (map[string]string, error) {
	return r.resolveIDsByName(ctx, "employee_groups", names)
}

// ResolveGroupIDsByIDs 校验并保留已存在的分组 ID。
func (r *Repository) ResolveGroupIDsByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return r.resolveIDsByColumn(ctx, "employee_groups", "id", ids)
}

func (r *Repository) resolveIDsByName(ctx context.Context, table string, names []string) (map[string]string, error) {
	return r.resolveIDsByColumn(ctx, table, "name", names)
}

func (r *Repository) resolveIDsByColumn(ctx context.Context, table, column string, values []string) (map[string]string, error) {
	result := make(map[string]string)
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 {
		return result, nil
	}

	type row struct {
		ID    string
		Value string `gorm:"column:value"`
	}
	var rows []row
	err := tenant.ApplyScopeOnColumn(ctx, r.db.WithContext(ctx).Table(table), table+".org_node_id").
		Select("id, "+column+" as value").
		Where(column+" IN ?", trimmed).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[strings.TrimSpace(item.Value)] = item.ID
	}
	return result, nil
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Rule{}, &RuleAssociation{}, &RuleApplyScope{}, &RuleDependency{}, &RuleConflict{})
}
