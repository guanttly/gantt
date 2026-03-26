package rule

import (
	"context"

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
		Order("category ASC, priority ASC").
		Find(&rules).Error
	return rules, err
}

// ListByNodeID 查询指定节点的规则列表（不使用 tenant scope，用于继承计算）。
func (r *Repository) ListByNodeID(ctx context.Context, nodeID string) ([]Rule, error) {
	var rules []Rule
	err := r.db.WithContext(ctx).
		Where("org_node_id = ? AND is_enabled = ?", nodeID, true).
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

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Rule{}, &RuleAssociation{})
}
