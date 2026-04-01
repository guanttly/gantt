package rule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRuleNotFound           = errors.New("规则不存在")
	ErrInvalidRuleType        = errors.New("无效的规则类型")
	ErrInvalidCategory        = errors.New("无效的规则分类")
	ErrInvalidSubType         = errors.New("无效的规则子类型")
	ErrInvalidConfig          = errors.New("无效的规则配置")
	ErrRuleNameDup            = errors.New("同节点下规则名称已存在")
	ErrOverrideNotSupported   = errors.New("规则继承与覆盖能力已下线，请直接在当前科室维护规则")
	ErrNotDeptNode            = errors.New("只有科室级（department）节点可以管理排班规则")
	ErrShiftReferenceNotFound = errors.New("存在无法映射的班次短代号或名称")
)

var validRuleTypes = map[string]bool{
	RuleTypeExclusive:        true,
	RuleTypeCombinable:       true,
	RuleTypeRequiredTogether: true,
	RuleTypePeriodic:         true,
	RuleTypeMaxCount:         true,
	RuleTypeForbiddenDay:     true,
	RuleTypePreferred:        true,
	RuleTypeSource:           true,
	RuleTypeOrder:            true,
	RuleTypeMinRestDays:      true,
}

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

type CreateInput struct {
	Name            string            `json:"name"`
	Type            string            `json:"type,omitempty"`
	RuleType        string            `json:"rule_type,omitempty"`
	Category        string            `json:"category"`
	SubType         string            `json:"sub_type,omitempty"`
	SubCategory     string            `json:"sub_category,omitempty"`
	ApplyScope      string            `json:"apply_scope,omitempty"`
	TimeScope       string            `json:"time_scope,omitempty"`
	TimeOffsetDays  *int              `json:"time_offset_days,omitempty"`
	RuleData        *string           `json:"rule_data,omitempty"`
	Config          json.RawMessage   `json:"config"`
	Priority        int               `json:"priority"`
	Enabled         *bool             `json:"enabled,omitempty"`
	IsEnabled       *bool             `json:"is_enabled,omitempty"`
	SourceType      string            `json:"source_type,omitempty"`
	ParseConfidence *float64          `json:"parse_confidence,omitempty"`
	Version         string            `json:"version,omitempty"`
	OverrideRuleID  *string           `json:"override_rule_id,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Associations    []AssocInput      `json:"associations,omitempty"`
	ApplyScopes     []ApplyScopeInput `json:"apply_scopes,omitempty"`
}

type UpdateInput struct {
	Name            *string            `json:"name,omitempty"`
	Type            *string            `json:"type,omitempty"`
	RuleType        *string            `json:"rule_type,omitempty"`
	Category        *string            `json:"category,omitempty"`
	SubType         *string            `json:"sub_type,omitempty"`
	SubCategory     *string            `json:"sub_category,omitempty"`
	ApplyScope      *string            `json:"apply_scope,omitempty"`
	TimeScope       *string            `json:"time_scope,omitempty"`
	TimeOffsetDays  *int               `json:"time_offset_days,omitempty"`
	RuleData        *string            `json:"rule_data,omitempty"`
	Config          *json.RawMessage   `json:"config,omitempty"`
	Priority        *int               `json:"priority,omitempty"`
	Enabled         *bool              `json:"enabled,omitempty"`
	IsEnabled       *bool              `json:"is_enabled,omitempty"`
	SourceType      *string            `json:"source_type,omitempty"`
	ParseConfidence *float64           `json:"parse_confidence,omitempty"`
	Version         *string            `json:"version,omitempty"`
	Description     *string            `json:"description,omitempty"`
	Associations    *[]AssocInput      `json:"associations,omitempty"`
	ApplyScopes     *[]ApplyScopeInput `json:"apply_scopes,omitempty"`
}

type AssocInput struct {
	TargetType      string `json:"target_type,omitempty"`
	TargetID        string `json:"target_id,omitempty"`
	AssociationType string `json:"association_type,omitempty"`
	AssociationID   string `json:"association_id,omitempty"`
	Role            string `json:"role,omitempty"`
}

type ApplyScopeInput struct {
	ScopeType string  `json:"scope_type"`
	ScopeID   *string `json:"scope_id,omitempty"`
	ScopeName *string `json:"scope_name,omitempty"`
}

type ParsedRuleInput struct {
	CreateInput
	SubjectShifts  []string `json:"subject_shifts,omitempty"`
	ObjectShifts   []string `json:"object_shifts,omitempty"`
	TargetShifts   []string `json:"target_shifts,omitempty"`
	ScopeType      string   `json:"scope_type,omitempty"`
	ScopeEmployees []string `json:"scope_employees,omitempty"`
	ScopeGroups    []string `json:"scope_groups,omitempty"`
}

type BatchCreateInput struct {
	ParsedRules  []ParsedRuleInput `json:"parsed_rules"`
	Dependencies []RuleDependency  `json:"dependencies,omitempty"`
	Conflicts    []RuleConflict    `json:"conflicts,omitempty"`
}

type ValidateInput struct {
	EmployeeID string `json:"employee_id"`
	ShiftID    string `json:"shift_id"`
	Date       string `json:"date"`
}

type Service struct {
	repo            *Repository
	nodeRepo        *tenant.Repository
	orgNodeResolver OrgNodeTypeChecker
}

type OrgNodeTypeChecker interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
}

type normalizedCreateInput struct {
	Name            string
	RuleType        string
	Category        string
	SubType         string
	ApplyScope      string
	TimeScope       string
	TimeOffsetDays  *int
	RuleData        string
	Config          json.RawMessage
	Priority        int
	Enabled         bool
	SourceType      string
	ParseConfidence *float64
	Version         string
	OverrideRuleID  *string
	Description     *string
	Associations    []RuleAssociation
	ApplyScopes     []RuleApplyScope
}

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

func (s *Service) Create(ctx context.Context, input CreateInput) (*Rule, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	return s.createWithRepo(ctx, s.repo, input)
}

func (s *Service) createWithRepo(ctx context.Context, repo *Repository, input CreateInput) (*Rule, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return nil, err
	}
	if normalized.OverrideRuleID != nil && *normalized.OverrideRuleID != "" {
		return nil, ErrOverrideNotSupported
	}

	rule := &Rule{
		ID:              uuid.New().String(),
		Name:            normalized.Name,
		Type:            normalized.RuleType,
		Category:        normalized.Category,
		SubType:         normalized.SubType,
		ApplyScope:      normalized.ApplyScope,
		TimeScope:       normalized.TimeScope,
		TimeOffsetDays:  normalized.TimeOffsetDays,
		RuleData:        normalized.RuleData,
		Config:          normalized.Config,
		Priority:        normalized.Priority,
		IsEnabled:       normalized.Enabled,
		SourceType:      normalized.SourceType,
		ParseConfidence: normalized.ParseConfidence,
		Version:         normalized.Version,
		Description:     normalized.Description,
		TenantModel:     tenant.TenantModel{OrgNodeID: orgNodeID},
	}

	if err := repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("创建规则失败: %w", err)
	}

	if len(normalized.Associations) > 0 {
		for i := range normalized.Associations {
			normalized.Associations[i].RuleID = rule.ID
			normalized.Associations[i].TenantModel = tenant.TenantModel{OrgNodeID: orgNodeID}
		}
		if err := repo.BatchCreateAssociations(ctx, normalized.Associations); err != nil {
			return nil, fmt.Errorf("创建规则关联失败: %w", err)
		}
		rule.Associations = normalized.Associations
	}

	if len(normalized.ApplyScopes) > 0 {
		for i := range normalized.ApplyScopes {
			normalized.ApplyScopes[i].RuleID = rule.ID
			normalized.ApplyScopes[i].TenantModel = tenant.TenantModel{OrgNodeID: orgNodeID}
		}
		if err := repo.CreateApplyScopes(ctx, normalized.ApplyScopes); err != nil {
			return nil, fmt.Errorf("创建规则适用范围失败: %w", err)
		}
		rule.ApplyScopes = normalized.ApplyScopes
	}

	rule.NormalizeAliases()
	return rule, nil
}

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
	if err := s.hydrateRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

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
		rule.Name = strings.TrimSpace(*input.Name)
	}
	if input.Type != nil && strings.TrimSpace(*input.Type) != "" {
		rule.Type = strings.TrimSpace(*input.Type)
	}
	if input.RuleType != nil && strings.TrimSpace(*input.RuleType) != "" {
		rule.Type = strings.TrimSpace(*input.RuleType)
	}
	if input.Category != nil {
		rule.Category = strings.TrimSpace(*input.Category)
	}
	if input.SubType != nil && strings.TrimSpace(*input.SubType) != "" {
		rule.SubType = strings.TrimSpace(*input.SubType)
	}
	if input.SubCategory != nil && strings.TrimSpace(*input.SubCategory) != "" {
		rule.SubType = strings.TrimSpace(*input.SubCategory)
	}
	if input.ApplyScope != nil {
		rule.ApplyScope = strings.TrimSpace(*input.ApplyScope)
	}
	if input.TimeScope != nil {
		rule.TimeScope = strings.TrimSpace(*input.TimeScope)
	}
	if input.TimeOffsetDays != nil {
		rule.TimeOffsetDays = input.TimeOffsetDays
	}
	if input.RuleData != nil {
		rule.RuleData = strings.TrimSpace(*input.RuleData)
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
	if input.Enabled != nil {
		rule.IsEnabled = *input.Enabled
	}
	if input.IsEnabled != nil {
		rule.IsEnabled = *input.IsEnabled
	}
	if input.SourceType != nil {
		rule.SourceType = strings.TrimSpace(*input.SourceType)
	}
	if input.ParseConfidence != nil {
		rule.ParseConfidence = input.ParseConfidence
	}
	if input.Version != nil {
		rule.Version = strings.TrimSpace(*input.Version)
	}
	if input.Description != nil {
		rule.Description = input.Description
	}
	if rule.Type == "" {
		rule.Type = inferRuleTypeFromSubType(rule.SubType)
	}
	if !validRuleTypes[rule.Type] {
		return nil, ErrInvalidRuleType
	}
	if !validCategories[rule.Category] {
		return nil, ErrInvalidCategory
	}
	if !validSubTypes[rule.SubType] {
		return nil, ErrInvalidSubType
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("更新规则失败: %w", err)
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if input.Associations != nil {
		assocs := normalizeAssocInputs(*input.Associations, rule.ID, orgNodeID)
		if err := s.repo.ReplaceAssociationsByRule(ctx, rule.ID, assocs); err != nil {
			return nil, fmt.Errorf("更新规则关联失败: %w", err)
		}
		rule.Associations = assocs
	}
	if input.ApplyScopes != nil {
		scopes := normalizeApplyScopeInputs(*input.ApplyScopes, rule.ID, orgNodeID)
		if err := s.repo.ReplaceApplyScopesByRule(ctx, rule.ID, scopes); err != nil {
			return nil, fmt.Errorf("更新规则适用范围失败: %w", err)
		}
		rule.ApplyScopes = scopes
	}
	if err := s.hydrateRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *Service) BatchCreateParsed(ctx context.Context, input BatchCreateInput) ([]Rule, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}
	if len(input.ParsedRules) == 0 {
		return nil, fmt.Errorf("parsed_rules 不能为空")
	}

	var saved []Rule
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &Repository{db: tx}
		shiftRefs := collectShiftRefs(input.ParsedRules)
		shiftCodeMap, err := txRepo.ResolveShiftIDsByCodes(ctx, shiftRefs)
		if err != nil {
			return err
		}
		shiftNameMap, err := txRepo.ResolveShiftIDsByNames(ctx, shiftRefs)
		if err != nil {
			return err
		}
		shiftIDMap, err := txRepo.ResolveShiftIDsByIDs(ctx, shiftRefs)
		if err != nil {
			return err
		}
		employeeRefs := collectEmployeeRefs(input.ParsedRules)
		employeeNameMap, err := txRepo.ResolveEmployeeIDsByNames(ctx, employeeRefs)
		if err != nil {
			return err
		}
		employeeIDMap, err := txRepo.ResolveEmployeeIDsByIDs(ctx, employeeRefs)
		if err != nil {
			return err
		}
		groupRefs := collectGroupRefs(input.ParsedRules)
		groupNameMap, err := txRepo.ResolveGroupIDsByNames(ctx, groupRefs)
		if err != nil {
			return err
		}
		groupIDMap, err := txRepo.ResolveGroupIDsByIDs(ctx, groupRefs)
		if err != nil {
			return err
		}
		ruleNameToID := make(map[string]string, len(input.ParsedRules))
		saved = make([]Rule, 0, len(input.ParsedRules))
		for _, parsed := range input.ParsedRules {
			normalizedConfig, unresolvedShiftRefs, err := normalizeParsedRuleConfig(parsed.Config, shiftCodeMap, shiftNameMap, shiftIDMap)
			if err != nil {
				return err
			}
			if len(unresolvedShiftRefs) > 0 {
				return fmt.Errorf("%w: %s", ErrShiftReferenceNotFound, strings.Join(unresolvedShiftRefs, "、"))
			}
			parsed.CreateInput.Config = normalizedConfig
			assocs, unresolvedAssocRefs := buildAssocInputs(parsed, shiftCodeMap, shiftNameMap, shiftIDMap, employeeNameMap, employeeIDMap, groupNameMap, groupIDMap)
			if len(unresolvedAssocRefs) > 0 {
				return fmt.Errorf("%w: %s", ErrShiftReferenceNotFound, strings.Join(unresolvedAssocRefs, "、"))
			}
			parsed.CreateInput.Associations = assocs
			parsed.CreateInput.ApplyScopes = buildApplyScopeInputs(parsed, employeeNameMap, employeeIDMap, groupNameMap, groupIDMap)
			rule, err := s.createWithRepo(ctx, txRepo, parsed.CreateInput)
			if err != nil {
				return err
			}
			ruleNameToID[strings.TrimSpace(rule.Name)] = rule.ID
			saved = append(saved, *rule)
		}
		if err := txRepo.BatchCreateDependencies(ctx, buildRuleDependencies(input.Dependencies, ruleNameToID, orgNodeID)); err != nil {
			return fmt.Errorf("保存规则依赖失败: %w", err)
		}
		if err := txRepo.BatchCreateConflicts(ctx, buildRuleConflicts(input.Conflicts, ruleNameToID, orgNodeID)); err != nil {
			return fmt.Errorf("保存规则冲突失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range saved {
		if err := s.hydrateRule(ctx, &saved[i]); err != nil {
			return nil, err
		}
	}
	return saved, nil
}

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
		if err := s.hydrateRule(ctx, &r); err != nil {
			return nil, err
		}
		result = append(result, RuleWithSource{Rule: r, SourceNode: "本级", IsInherited: false, IsOverridable: false})
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

func (s *Service) GetAssociations(ctx context.Context, ruleID string) ([]RuleAssociation, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListAssociationsByRule(ctx, ruleID)
}

func (s *Service) DisableInherited(ctx context.Context, ruleID, reason, actorUserID string) (*RuleWithSource, error) {
	return nil, ErrOverrideNotSupported
}

func (s *Service) RestoreInheritance(ctx context.Context, ruleID string) error {
	return ErrOverrideNotSupported
}

func (s *Service) hydrateRule(ctx context.Context, rule *Rule) error {
	assocs, err := s.repo.ListAssociationsByRule(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("查询规则关联失败: %w", err)
	}
	scopes, err := s.repo.ListApplyScopesByRule(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("查询规则适用范围失败: %w", err)
	}
	deps, err := s.repo.ListDependenciesByRuleIDs(ctx, []string{rule.ID})
	if err != nil {
		return fmt.Errorf("查询规则依赖失败: %w", err)
	}
	confs, err := s.repo.ListConflictsByRuleIDs(ctx, []string{rule.ID})
	if err != nil {
		return fmt.Errorf("查询规则冲突失败: %w", err)
	}
	rule.Associations = assocs
	rule.ApplyScopes = scopes
	rule.Dependencies = filterDependenciesForRule(rule.ID, deps)
	rule.Conflicts = filterConflictsForRule(rule.ID, confs)
	rule.NormalizeAliases()
	return nil
}

func normalizeCreateInput(input CreateInput) (*normalizedCreateInput, error) {
	ruleType := firstNonEmpty(input.RuleType, input.Type)
	if ruleType == "" {
		ruleType = inferRuleType(input.SubType, input.SubCategory, input.Config)
	}
	if !validRuleTypes[ruleType] {
		return nil, ErrInvalidRuleType
	}
	subType := firstNonEmpty(input.SubType, input.SubCategory, inferSubType(ruleType))
	if !validSubTypes[subType] {
		return nil, ErrInvalidSubType
	}
	category := firstNonEmpty(input.Category, inferCategory(ruleType))
	if !validCategories[category] {
		return nil, ErrInvalidCategory
	}
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return nil, ErrInvalidConfig
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	applyScope := firstNonEmpty(input.ApplyScope, ApplyScopeGlobal)
	timeScope := firstNonEmpty(input.TimeScope, TimeScopeSameDay)
	sourceType := firstNonEmpty(input.SourceType, SourceTypeManual)
	version := firstNonEmpty(input.Version, "v4")
	ruleData := ""
	if input.RuleData != nil {
		ruleData = strings.TrimSpace(*input.RuleData)
	}
	if ruleData == "" && input.Description != nil {
		ruleData = strings.TrimSpace(*input.Description)
	}
	return &normalizedCreateInput{
		Name:            strings.TrimSpace(input.Name),
		RuleType:        ruleType,
		Category:        category,
		SubType:         subType,
		ApplyScope:      applyScope,
		TimeScope:       timeScope,
		TimeOffsetDays:  input.TimeOffsetDays,
		RuleData:        ruleData,
		Config:          input.Config,
		Priority:        input.Priority,
		Enabled:         enabled,
		SourceType:      sourceType,
		ParseConfidence: input.ParseConfidence,
		Version:         version,
		OverrideRuleID:  input.OverrideRuleID,
		Description:     input.Description,
		Associations:    normalizeAssocInputs(input.Associations, "", ""),
		ApplyScopes:     normalizeApplyScopeInputs(input.ApplyScopes, "", ""),
	}, nil
}

func normalizeAssocInputs(inputs []AssocInput, ruleID, orgNodeID string) []RuleAssociation {
	assocs := make([]RuleAssociation, 0, len(inputs))
	for _, input := range inputs {
		targetType := firstNonEmpty(input.AssociationType, input.TargetType)
		targetID := firstNonEmpty(input.AssociationID, input.TargetID)
		if targetType == "" || targetID == "" {
			continue
		}
		assoc := RuleAssociation{RuleID: ruleID, TargetType: targetType, TargetID: targetID, Role: firstNonEmpty(input.Role, AssociationRoleTarget), TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID}}
		assoc.NormalizeAliases()
		assocs = append(assocs, assoc)
	}
	return assocs
}

func normalizeApplyScopeInputs(inputs []ApplyScopeInput, ruleID, orgNodeID string) []RuleApplyScope {
	scopes := make([]RuleApplyScope, 0, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.ScopeType) == "" {
			continue
		}
		scopes = append(scopes, RuleApplyScope{RuleID: ruleID, ScopeType: strings.TrimSpace(input.ScopeType), ScopeID: input.ScopeID, ScopeName: input.ScopeName, TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID}})
	}
	return scopes
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func inferRuleType(subType, subCategory string, config json.RawMessage) string {
	if inferred := inferRuleTypeFromSubType(firstNonEmpty(subType, subCategory)); inferred != "" {
		return inferred
	}
	var payload struct {
		Type string `json:"type"`
	}
	if len(config) > 0 && json.Unmarshal(config, &payload) == nil {
		switch strings.TrimSpace(payload.Type) {
		case "exclusive_shifts":
			return RuleTypeExclusive
		case "max_count":
			return RuleTypeMaxCount
		case "min_rest":
			return RuleTypeMinRestDays
		case "required_together":
			return RuleTypeRequiredTogether
		case "prefer_employee":
			return RuleTypePreferred
		case "staff_source":
			return RuleTypeSource
		case "execution_order":
			return RuleTypeOrder
		}
	}
	return ""
}

func inferSubType(ruleType string) string {
	switch ruleType {
	case RuleTypeExclusive, RuleTypeForbiddenDay:
		return SubTypeForbid
	case RuleTypeMaxCount, RuleTypePeriodic:
		return SubTypeLimit
	case RuleTypeRequiredTogether:
		return SubTypeMust
	case RuleTypePreferred:
		return SubTypePrefer
	case RuleTypeCombinable:
		return SubTypeCombinable
	case RuleTypeSource:
		return SubTypeSource
	case RuleTypeOrder:
		return SubTypeOrder
	case RuleTypeMinRestDays:
		return SubTypeMinRest
	default:
		return ""
	}
}

func inferCategory(ruleType string) string {
	switch ruleType {
	case RuleTypePreferred:
		return CategoryPreference
	case RuleTypeSource, RuleTypeOrder:
		return CategoryDependency
	default:
		return CategoryConstraint
	}
}

func collectShiftRefs(rules []ParsedRuleInput) []string {
	result := make([]string, 0)
	for _, rule := range rules {
		result = append(result, rule.SubjectShifts...)
		result = append(result, rule.ObjectShifts...)
		result = append(result, rule.TargetShifts...)
		result = append(result, collectConfigShiftRefs(rule.Config)...)
	}
	return result
}

func collectEmployeeRefs(rules []ParsedRuleInput) []string {
	result := make([]string, 0)
	for _, rule := range rules {
		result = append(result, rule.ScopeEmployees...)
	}
	return result
}

func collectGroupRefs(rules []ParsedRuleInput) []string {
	result := make([]string, 0)
	for _, rule := range rules {
		result = append(result, rule.ScopeGroups...)
	}
	return result
}

func buildAssocInputs(rule ParsedRuleInput, shiftCodeMap, shiftNameMap, shiftIDMap, employeeNameMap, employeeIDMap, groupNameMap, groupIDMap map[string]string) ([]AssocInput, []string) {
	assocs := append([]AssocInput{}, rule.Associations...)
	unresolved := make([]string, 0)
	for _, ref := range rule.SubjectShifts {
		resolvedID := resolveReferenceID(ref, shiftIDMap, shiftCodeMap, shiftNameMap)
		if resolvedID == "" {
			if trimmed := strings.TrimSpace(ref); trimmed != "" {
				unresolved = append(unresolved, trimmed)
			}
			continue
		}
		assocs = append(assocs, AssocInput{AssociationType: TargetTypeShift, AssociationID: resolvedID, Role: AssociationRoleSubject})
	}
	for _, ref := range rule.ObjectShifts {
		resolvedID := resolveReferenceID(ref, shiftIDMap, shiftCodeMap, shiftNameMap)
		if resolvedID == "" {
			if trimmed := strings.TrimSpace(ref); trimmed != "" {
				unresolved = append(unresolved, trimmed)
			}
			continue
		}
		assocs = append(assocs, AssocInput{AssociationType: TargetTypeShift, AssociationID: resolvedID, Role: AssociationRoleObject})
	}
	for _, ref := range rule.TargetShifts {
		resolvedID := resolveReferenceID(ref, shiftIDMap, shiftCodeMap, shiftNameMap)
		if resolvedID == "" {
			if trimmed := strings.TrimSpace(ref); trimmed != "" {
				unresolved = append(unresolved, trimmed)
			}
			continue
		}
		assocs = append(assocs, AssocInput{AssociationType: TargetTypeShift, AssociationID: resolvedID, Role: AssociationRoleTarget})
	}
	for i := range assocs {
		assocType := firstNonEmpty(assocs[i].AssociationType, assocs[i].TargetType)
		assocRef := firstNonEmpty(assocs[i].AssociationID, assocs[i].TargetID)
		switch assocType {
		case TargetTypeShift:
			if assocs[i].AssociationID == "" {
				assocs[i].AssociationID = resolveReferenceID(assocRef, shiftIDMap, shiftCodeMap, shiftNameMap)
			}
		case TargetTypeEmployee:
			if assocs[i].AssociationID == "" {
				assocs[i].AssociationID = resolveReferenceID(assocRef, employeeIDMap, employeeNameMap)
			}
		case TargetTypeGroup:
			if assocs[i].AssociationID == "" {
				assocs[i].AssociationID = resolveReferenceID(assocRef, groupIDMap, groupNameMap)
			}
		}
	}
	return assocs, uniqueStrings(unresolved)
}

func buildApplyScopeInputs(rule ParsedRuleInput, employeeNameMap, employeeIDMap, groupNameMap, groupIDMap map[string]string) []ApplyScopeInput {
	if len(rule.ApplyScopes) > 0 {
		return rule.ApplyScopes
	}
	if strings.TrimSpace(rule.ScopeType) == "" {
		return nil
	}
	if rule.ScopeType == ScopeTypeAll {
		return []ApplyScopeInput{{ScopeType: ScopeTypeAll}}
	}
	result := make([]ApplyScopeInput, 0)
	for _, ref := range rule.ScopeEmployees {
		if id := resolveReferenceID(ref, employeeIDMap, employeeNameMap); id != "" {
			idCopy := id
			nameCopy := strings.TrimSpace(ref)
			result = append(result, ApplyScopeInput{ScopeType: rule.ScopeType, ScopeID: &idCopy, ScopeName: &nameCopy})
		}
	}
	for _, ref := range rule.ScopeGroups {
		if id := resolveReferenceID(ref, groupIDMap, groupNameMap); id != "" {
			idCopy := id
			nameCopy := strings.TrimSpace(ref)
			result = append(result, ApplyScopeInput{ScopeType: rule.ScopeType, ScopeID: &idCopy, ScopeName: &nameCopy})
		}
	}
	return result
}

func collectConfigShiftRefs(config json.RawMessage) []string {
	if len(config) == 0 || !json.Valid(config) {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(config, &payload); err != nil {
		return nil
	}
	result := make([]string, 0)
	for _, key := range []string{"shift_id", "target_shift_id", "source_shift_id", "before_shift_id", "after_shift_id"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	if values, ok := payload["shift_ids"].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
	}
	return result
}

func normalizeParsedRuleConfig(config json.RawMessage, shiftCodeMap, shiftNameMap, shiftIDMap map[string]string) (json.RawMessage, []string, error) {
	if len(config) == 0 {
		return config, nil, nil
	}
	if !json.Valid(config) {
		return nil, nil, ErrInvalidConfig
	}
	var payload map[string]any
	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, nil, ErrInvalidConfig
	}

	unresolved := make([]string, 0)
	for _, key := range []string{"shift_id", "target_shift_id", "source_shift_id", "before_shift_id", "after_shift_id"} {
		value, ok := payload[key].(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if resolvedID := resolveReferenceID(trimmed, shiftIDMap, shiftCodeMap, shiftNameMap); resolvedID != "" {
			payload[key] = resolvedID
			continue
		}
		unresolved = append(unresolved, trimmed)
	}
	if values, ok := payload["shift_ids"].([]any); ok {
		resolvedValues := make([]string, 0, len(values))
		for _, value := range values {
			text, ok := value.(string)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				continue
			}
			if resolvedID := resolveReferenceID(trimmed, shiftIDMap, shiftCodeMap, shiftNameMap); resolvedID != "" {
				resolvedValues = append(resolvedValues, resolvedID)
				continue
			}
			unresolved = append(unresolved, trimmed)
		}
		payload["shift_ids"] = resolvedValues
	}
	if len(unresolved) > 0 {
		unresolved = uniqueStrings(unresolved)
		sort.Strings(unresolved)
		return nil, unresolved, nil
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, ErrInvalidConfig
	}
	return normalized, nil, nil
}

func resolveReferenceID(ref string, maps ...map[string]string) string {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return ""
	}
	for _, current := range maps {
		if current == nil {
			continue
		}
		if id := strings.TrimSpace(current[trimmed]); id != "" {
			return id
		}
	}
	return ""
}

func buildRuleDependencies(inputs []RuleDependency, ruleNameToID map[string]string, orgNodeID string) []RuleDependency {
	result := make([]RuleDependency, 0, len(inputs))
	for _, input := range inputs {
		depRuleID := ruleNameToID[strings.TrimSpace(input.DependentRuleName)]
		onRuleID := ruleNameToID[strings.TrimSpace(input.DependentOnName)]
		if depRuleID == "" || onRuleID == "" || depRuleID == onRuleID {
			continue
		}
		result = append(result, RuleDependency{DependentRuleID: depRuleID, DependentOnRuleID: onRuleID, DependentRuleName: input.DependentRuleName, DependentOnName: input.DependentOnName, DependencyType: input.DependencyType, Description: input.Description, TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID}})
	}
	return result
}

func buildRuleConflicts(inputs []RuleConflict, ruleNameToID map[string]string, orgNodeID string) []RuleConflict {
	result := make([]RuleConflict, 0, len(inputs))
	for _, input := range inputs {
		ruleID1 := ruleNameToID[strings.TrimSpace(input.RuleName1)]
		ruleID2 := ruleNameToID[strings.TrimSpace(input.RuleName2)]
		if ruleID1 == "" || ruleID2 == "" || ruleID1 == ruleID2 {
			continue
		}
		result = append(result, RuleConflict{RuleID1: ruleID1, RuleID2: ruleID2, RuleName1: input.RuleName1, RuleName2: input.RuleName2, ConflictType: input.ConflictType, Description: input.Description, ResolutionPriority: input.ResolutionPriority, TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID}})
	}
	return result
}

func filterDependenciesForRule(ruleID string, deps []RuleDependency) []RuleDependency {
	result := make([]RuleDependency, 0)
	for _, dep := range deps {
		if dep.DependentRuleID == ruleID || dep.DependentOnRuleID == ruleID {
			result = append(result, dep)
		}
	}
	return result
}

func filterConflictsForRule(ruleID string, conflicts []RuleConflict) []RuleConflict {
	result := make([]RuleConflict, 0)
	for _, conflict := range conflicts {
		if conflict.RuleID1 == ruleID || conflict.RuleID2 == ruleID {
			result = append(result, conflict)
		}
	}
	return result
}
