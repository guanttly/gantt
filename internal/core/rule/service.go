package rule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRuleNotFound     = errors.New("规则不存在")
	ErrInvalidCategory  = errors.New("无效的规则分类")
	ErrInvalidSubType   = errors.New("无效的规则子类型")
	ErrInvalidConfig    = errors.New("无效的规则配置")
	ErrRuleNameDup      = errors.New("同节点下规则名称已存在")
	ErrOverrideNotFound = errors.New("覆盖的上级规则不存在")
	ErrCannotOverride   = errors.New("只能覆盖上级节点的规则")
)

// 合法的分类和子类型。
var validCategories = map[string]bool{
	CategoryConstraint: true,
	CategoryPreference: true,
	CategoryDependency: true,
}

var validSubTypes = map[string]bool{
	SubTypeForbid:     true,
	SubTypeLimit:      true,
	SubTypeMust:       true,
	SubTypePrefer:     true,
	SubTypeCombinable: true,
	SubTypeSource:     true,
	SubTypeOrder:      true,
	SubTypeMinRest:    true,
}

// CreateInput 创建规则的输入参数。
type CreateInput struct {
	Name           string          `json:"name"`
	Category       string          `json:"category"`
	SubType        string          `json:"sub_type"`
	Config         json.RawMessage `json:"config"`
	Priority       int             `json:"priority"`
	IsEnabled      *bool           `json:"is_enabled,omitempty"`
	OverrideRuleID *string         `json:"override_rule_id,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Associations   []AssocInput    `json:"associations,omitempty"`
}

// UpdateInput 更新规则的输入参数。
type UpdateInput struct {
	Name        *string          `json:"name,omitempty"`
	Config      *json.RawMessage `json:"config,omitempty"`
	Priority    *int             `json:"priority,omitempty"`
	IsEnabled   *bool            `json:"is_enabled,omitempty"`
	Description *string          `json:"description,omitempty"`
}

// AssocInput 规则关联输入。
type AssocInput struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

// ValidateInput 校验请求输入。
type ValidateInput struct {
	EmployeeID string `json:"employee_id"`
	ShiftID    string `json:"shift_id"`
	Date       string `json:"date"` // YYYY-MM-DD
}

// Service 规则业务逻辑层。
type Service struct {
	repo     *Repository
	nodeRepo *tenant.Repository
}

// NewService 创建规则服务。
func NewService(repo *Repository, nodeRepo *tenant.Repository) *Service {
	return &Service{repo: repo, nodeRepo: nodeRepo}
}

// Create 创建规则。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Rule, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 校验分类和子类型
	if !validCategories[input.Category] {
		return nil, ErrInvalidCategory
	}
	if !validSubTypes[input.SubType] {
		return nil, ErrInvalidSubType
	}

	// 校验 config JSON 格式
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return nil, ErrInvalidConfig
	}

	// 如果指定了覆盖规则，校验覆盖规则存在且属于上级节点
	if input.OverrideRuleID != nil && *input.OverrideRuleID != "" {
		overrideRule, err := s.repo.GetByID(ctx, *input.OverrideRuleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrOverrideNotFound
			}
			return nil, fmt.Errorf("查询覆盖规则失败: %w", err)
		}
		// 确保覆盖的规则不在当前节点（只能覆盖上级的规则）
		if overrideRule.OrgNodeID == orgNodeID {
			return nil, ErrCannotOverride
		}
	}

	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}

	rule := &Rule{
		ID:             uuid.New().String(),
		Name:           input.Name,
		Category:       input.Category,
		SubType:        input.SubType,
		Config:         input.Config,
		Priority:       input.Priority,
		IsEnabled:      isEnabled,
		OverrideRuleID: input.OverrideRuleID,
		Description:    input.Description,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("创建规则失败: %w", err)
	}

	// 创建关联
	if len(input.Associations) > 0 {
		assocs := make([]RuleAssociation, 0, len(input.Associations))
		for _, a := range input.Associations {
			assocs = append(assocs, RuleAssociation{
				RuleID:     rule.ID,
				TargetType: a.TargetType,
				TargetID:   a.TargetID,
				TenantModel: tenant.TenantModel{
					OrgNodeID: orgNodeID,
				},
			})
		}
		if err := s.repo.BatchCreateAssociations(ctx, assocs); err != nil {
			return nil, fmt.Errorf("创建规则关联失败: %w", err)
		}
	}

	return rule, nil
}

// GetByID 获取规则详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	return rule, nil
}

// Update 更新规则。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.Config != nil {
		if !json.Valid(*input.Config) {
			return nil, ErrInvalidConfig
		}
		rule.Config = *input.Config
	}
	if input.Priority != nil {
		rule.Priority = *input.Priority
	}
	if input.IsEnabled != nil {
		rule.IsEnabled = *input.IsEnabled
	}
	if input.Description != nil {
		rule.Description = input.Description
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("更新规则失败: %w", err)
	}

	return rule, nil
}

// Delete 删除规则。
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRuleNotFound
		}
		return err
	}

	// 关联通过外键级联删除
	return s.repo.Delete(ctx, id)
}

// List 查询当前节点的规则列表（含继承标记）。
func (s *Service) List(ctx context.Context) ([]RuleWithSource, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 计算生效规则集
	effective, err := s.ComputeEffectiveRules(ctx, orgNodeID)
	if err != nil {
		return nil, err
	}

	result := make([]RuleWithSource, 0, len(effective.Rules))
	for _, r := range effective.Rules {
		rws := RuleWithSource{
			Rule:          r,
			IsInherited:   r.OrgNodeID != orgNodeID,
			IsOverridable: r.OrgNodeID != orgNodeID,
		}
		if name, ok := effective.SourceMap[r.ID]; ok {
			rws.SourceNode = name
		}
		result = append(result, rws)
	}

	return result, nil
}

// ListEffective 计算最终生效的规则集。
func (s *Service) ListEffective(ctx context.Context) (*EffectiveRuleSet, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}
	return s.ComputeEffectiveRules(ctx, orgNodeID)
}

// GetAssociations 获取规则的关联列表。
func (s *Service) GetAssociations(ctx context.Context, ruleID string) ([]RuleAssociation, error) {
	return s.repo.ListAssociationsByRule(ctx, ruleID)
}
