package rule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRuleNotFound         = errors.New("规则不存在")
	ErrInvalidCategory      = errors.New("无效的规则分类")
	ErrInvalidSubType       = errors.New("无效的规则子类型")
	ErrInvalidConfig        = errors.New("无效的规则配置")
	ErrRuleNameDup          = errors.New("同节点下规则名称已存在")
	ErrOverrideNotSupported = errors.New("规则继承与覆盖能力已下线，请直接在当前科室维护规则")
	ErrNotDeptNode          = errors.New("只有科室级（department）节点可以管理排班规则")
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
	Date       string `json:"date"`
}

// Service 规则业务逻辑层。
type Service struct {
	repo            *Repository
	nodeRepo        *tenant.Repository
	orgNodeResolver OrgNodeTypeChecker
}

type OrgNodeTypeChecker interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
}

// NewService 创建规则服务。
func NewService(repo *Repository, nodeRepo *tenant.Repository) *Service {
	return &Service{repo: repo, nodeRepo: nodeRepo}
}

func (s *Service) SetOrgNodeResolver(resolver OrgNodeTypeChecker) {
	s.orgNodeResolver = resolver
}

func (s *Service) ensureDepartmentNode(ctx context.Context) error {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return fmt.Errorf("缺少组织节点信息")
	}
	if s.orgNodeResolver == nil {
		return nil
	}
	node, err := s.orgNodeResolver.GetByID(ctx, orgNodeID)
	if err != nil {
		return fmt.Errorf("查询组织节点失败: %w", err)
	}
	if !tenant.IsLeafNodeType(node.NodeType) {
		return ErrNotDeptNode
	}
	return nil
}

// Create 创建规则。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Rule, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}
	if !validCategories[input.Category] {
		return nil, ErrInvalidCategory
	}
	if !validSubTypes[input.SubType] {
		return nil, ErrInvalidSubType
	}
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return nil, ErrInvalidConfig
	}
	if input.OverrideRuleID != nil && *input.OverrideRuleID != "" {
		return nil, ErrOverrideNotSupported
	}

	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}

	rule := &Rule{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Category:    input.Category,
		SubType:     input.SubType,
		Config:      input.Config,
		Priority:    input.Priority,
		IsEnabled:   isEnabled,
		Description: input.Description,
		TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID},
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("创建规则失败: %w", err)
	}

	if len(input.Associations) > 0 {
		assocs := make([]RuleAssociation, 0, len(input.Associations))
		for _, a := range input.Associations {
			assocs = append(assocs, RuleAssociation{
				RuleID:      rule.ID,
				TargetType:  a.TargetType,
				TargetID:    a.TargetID,
				TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID},
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
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

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
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

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
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRuleNotFound
		}
		return err
	}

	return s.repo.Delete(ctx, id)
}

// List 查询当前节点的规则列表。
func (s *Service) List(ctx context.Context) ([]RuleWithSource, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	rules, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询规则列表失败: %w", err)
	}

	result := make([]RuleWithSource, 0, len(rules))
	for _, r := range rules {
		result = append(result, RuleWithSource{
			Rule:          r,
			SourceNode:    "本级",
			IsInherited:   false,
			IsOverridable: false,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		if result[i].Priority != result[j].Priority {
			return result[i].Priority < result[j].Priority
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// ListEffective 计算最终生效的规则集。
func (s *Service) ListEffective(ctx context.Context) (*EffectiveRuleSet, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}
	return s.ComputeEffectiveRules(ctx, orgNodeID)
}

// GetAssociations 获取规则的关联列表。
func (s *Service) GetAssociations(ctx context.Context, ruleID string) ([]RuleAssociation, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListAssociationsByRule(ctx, ruleID)
}

// DisableInherited 继承模型已下线。
func (s *Service) DisableInherited(ctx context.Context, ruleID, reason, actorUserID string) (*RuleWithSource, error) {
	return nil, ErrOverrideNotSupported
}

// RestoreInheritance 继承模型已下线。
func (s *Service) RestoreInheritance(ctx context.Context, ruleID string) error {
	return ErrOverrideNotSupported
}
