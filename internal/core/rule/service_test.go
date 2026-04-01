package rule

import (
	"context"
	"errors"
	"testing"

	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockOrgNodeResolver struct {
	nodes map[string]tenant.OrgNode
}

func (m *mockOrgNodeResolver) GetByID(_ context.Context, id string) (*tenant.OrgNode, error) {
	node, ok := m.nodes[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &node, nil
}

func setupRuleService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode, tenant.OrgNode, tenant.OrgNode) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	statements := []string{
		`CREATE TABLE org_nodes (
			id TEXT PRIMARY KEY,
			parent_id TEXT,
			node_type TEXT NOT NULL,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			contact_name TEXT,
			contact_phone TEXT,
			path TEXT NOT NULL,
			depth INTEGER NOT NULL DEFAULT 0,
			is_login_point BOOLEAN NOT NULL DEFAULT FALSE,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE rules (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			name TEXT NOT NULL,
			rule_type TEXT,
			category TEXT NOT NULL,
			sub_type TEXT NOT NULL,
			apply_scope TEXT NOT NULL DEFAULT 'global',
			time_scope TEXT NOT NULL DEFAULT 'same_day',
			time_offset_days INTEGER,
			rule_data TEXT,
			config TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			disabled BOOLEAN NOT NULL DEFAULT FALSE,
			disabled_by TEXT,
			disabled_at DATETIME,
			disabled_reason TEXT,
			override_rule_id TEXT,
			source_type TEXT,
			parse_confidence REAL,
			version TEXT,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE rule_associations (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			role TEXT,
			org_node_id TEXT NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE rule_apply_scopes (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT,
			scope_name TEXT,
			org_node_id TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE rule_dependencies (
			id TEXT PRIMARY KEY,
			dependent_rule_id TEXT NOT NULL,
			dependent_on_rule_id TEXT NOT NULL,
			dependency_type TEXT NOT NULL,
			description TEXT,
			org_node_id TEXT NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE rule_conflicts (
			id TEXT PRIMARY KEY,
			rule_id_1 TEXT NOT NULL,
			rule_id_2 TEXT NOT NULL,
			conflict_type TEXT NOT NULL,
			description TEXT,
			resolution_priority INTEGER,
			org_node_id TEXT NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE shifts (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			code TEXT,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE employees (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE employee_groups (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			name TEXT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}

	root := tenant.OrgNode{ID: "platform-root-id", NodeType: tenant.NodeTypeOrganization, Name: "平台管理", Code: "platform-root", Path: "/platform-root-id", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	orgParentID := root.ID
	org := tenant.OrgNode{ID: "org-001", ParentID: &orgParentID, NodeType: tenant.NodeTypeOrganization, Name: "测试机构", Code: "org-001", Path: "/platform-root-id/org-001", Depth: 1, IsLoginPoint: true, Status: tenant.StatusActive}
	deptParentID := org.ID
	dept := tenant.OrgNode{ID: "dept-001", ParentID: &deptParentID, NodeType: tenant.NodeTypeDepartment, Name: "急诊科", Code: "dept-001", Path: "/platform-root-id/org-001/dept-001", Depth: 2, IsLoginPoint: true, Status: tenant.StatusActive}

	if err := db.Create(&[]tenant.OrgNode{root, org, dept}).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}

	svc := NewService(NewRepository(db), tenant.NewRepository(db))
	svc.SetOrgNodeResolver(&mockOrgNodeResolver{nodes: map[string]tenant.OrgNode{root.ID: root, org.ID: org, dept.ID: dept}})
	return svc, db, root, org, dept
}

func TestService_CreateAndListLocalRules(t *testing.T) {
	svc, _, _, _, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)

	created, err := svc.Create(ctx, CreateInput{
		Name:     "科室夜班上限",
		Category: CategoryConstraint,
		SubType:  SubTypeLimit,
		Config:   []byte(`{"type":"max_count","shift_id":"night","max":2,"period":"week"}`),
		Priority: 20,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OrgNodeID != dept.ID {
		t.Fatalf("created.org_node_id = %q, want %q", created.OrgNodeID, dept.ID)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != created.ID {
		t.Fatalf("items[0].id = %q, want %q", items[0].ID, created.ID)
	}
	if items[0].IsInherited {
		t.Fatal("科室规则不应标记为继承")
	}

	effective, err := svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 1 || effective.Rules[0].ID != created.ID {
		t.Fatal("生效规则应仅包含当前科室启用规则")
	}
	if effective.SourceMap[created.ID] != "本级" {
		t.Fatalf("sourceMap[%q] = %q, want %q", created.ID, effective.SourceMap[created.ID], "本级")
	}
}

func TestService_RejectOverrideAndNonDepartmentNode(t *testing.T) {
	svc, db, _, org, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	baseRule := Rule{ID: "rule-org-003", Name: "机构连班限制", Type: RuleTypeMaxCount, Category: CategoryConstraint, SubType: SubTypeLimit, ApplyScope: ApplyScopeGlobal, TimeScope: TimeScopeSameWeek, Config: []byte(`{"type":"max_count","shift_id":"day","max":5,"period":"week"}`), Priority: 8, IsEnabled: true, SourceType: SourceTypeManual, Version: "v4", TenantModel: tenant.TenantModel{OrgNodeID: org.ID}}
	if err := db.Create(&baseRule).Error; err != nil {
		t.Fatalf("创建上级规则失败: %v", err)
	}

	if _, err := svc.Create(ctx, CreateInput{Name: "科室连班限制", Category: CategoryConstraint, SubType: SubTypeLimit, Config: []byte(`{"type":"max_count","shift_id":"day","max":3,"period":"week"}`), Priority: 9, OverrideRuleID: &baseRule.ID}); !errors.Is(err, ErrOverrideNotSupported) {
		t.Fatalf("Create(override) error = %v, want %v", err, ErrOverrideNotSupported)
	}

	orgCtx := tenant.WithOrgNode(context.Background(), org.ID, org.Path)
	if _, err := svc.List(orgCtx); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("List(org) error = %v, want %v", err, ErrNotDeptNode)
	}
	if _, err := svc.ListEffective(orgCtx); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("ListEffective(org) error = %v, want %v", err, ErrNotDeptNode)
	}
}

func TestService_BatchCreateParsed_PersistsHydratedRelations(t *testing.T) {
	svc, db, _, _, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)

	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-day', '` + dept.ID + `', 'DAY', 'Day')`,
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-night', '` + dept.ID + `', 'NIGHT', 'Night')`,
		`INSERT INTO employees (id, org_node_id, name) VALUES ('emp-alice', '` + dept.ID + `', 'Alice')`,
		`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-a', '` + dept.ID + `', 'TeamA')`,
	}
	for _, statement := range seedStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("插入批量规则测试数据失败: %v", err)
		}
	}

	saved, err := svc.BatchCreateParsed(ctx, BatchCreateInput{
		ParsedRules: []ParsedRuleInput{
			{
				CreateInput: CreateInput{
					Name:       "Night Source",
					RuleType:   RuleTypeSource,
					Category:   CategoryDependency,
					SubType:    SubTypeSource,
					ApplyScope: ApplyScopeSpecific,
					TimeScope:  TimeScopeSameDay,
					Config:     []byte(`{"type":"staff_source","target_shift_id":"NIGHT","source_shift_id":"DAY"}`),
					Priority:   10,
					SourceType: SourceTypeLLMParsed,
					Version:    "v4",
				},
				SubjectShifts: []string{"NIGHT"},
				ObjectShifts:  []string{"DAY"},
				ScopeType:     ScopeTypeGroup,
				ScopeGroups:   []string{"TeamA"},
			},
			{
				CreateInput: CreateInput{
					Name:       "Alice Prefers Night",
					RuleType:   RuleTypePreferred,
					Category:   CategoryPreference,
					SubType:    SubTypePrefer,
					ApplyScope: ApplyScopeSpecific,
					TimeScope:  TimeScopeSameDay,
					Config:     []byte(`{"type":"prefer_employee","employee_id":"emp-alice","shift_id":"shift-night","weight":80}`),
					Priority:   20,
					SourceType: SourceTypeLLMParsed,
					Version:    "v4",
				},
				TargetShifts:   []string{"NIGHT"},
				ScopeType:      ScopeTypeEmployee,
				ScopeEmployees: []string{"Alice"},
			},
		},
		Dependencies: []RuleDependency{{
			DependentRuleName: "Alice Prefers Night",
			DependentOnName:   "Night Source",
			DependencyType:    "source",
			Description:       "preference depends on source staffing",
		}},
		Conflicts: []RuleConflict{{
			RuleName1:    "Night Source",
			RuleName2:    "Alice Prefers Night",
			ConflictType: "exclusive",
			Description:  "test conflict",
		}},
	})
	if err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("len(saved) = %d, want 2", len(saved))
	}

	rulesByName := make(map[string]Rule, len(saved))
	for _, current := range saved {
		rulesByName[current.Name] = current
	}

	sourceRule, ok := rulesByName["Night Source"]
	if !ok {
		t.Fatal("missing saved source rule")
	}
	if sourceRule.Type != RuleTypeSource || sourceRule.RuleType != RuleTypeSource {
		t.Fatalf("source rule type aliases mismatch, got type=%q rule_type=%q", sourceRule.Type, sourceRule.RuleType)
	}
	if !sourceRule.Enabled || !sourceRule.IsEnabled {
		t.Fatal("source rule should be enabled")
	}
	if len(sourceRule.Associations) != 2 {
		t.Fatalf("len(sourceRule.Associations) = %d, want 2", len(sourceRule.Associations))
	}
	if string(sourceRule.Config) != `{"source_shift_id":"shift-day","target_shift_id":"shift-night","type":"staff_source"}` {
		t.Fatalf("sourceRule.Config = %s, want normalized shift ids", string(sourceRule.Config))
	}
	if len(sourceRule.ApplyScopes) != 1 || sourceRule.ApplyScopes[0].ScopeID == nil || *sourceRule.ApplyScopes[0].ScopeID != "grp-a" {
		t.Fatalf("source rule apply scopes = %+v, want TeamA group scope", sourceRule.ApplyScopes)
	}
	if len(sourceRule.Dependencies) != 1 {
		t.Fatalf("len(sourceRule.Dependencies) = %d, want 1", len(sourceRule.Dependencies))
	}
	if len(sourceRule.Conflicts) != 1 {
		t.Fatalf("len(sourceRule.Conflicts) = %d, want 1", len(sourceRule.Conflicts))
	}

	preferRule, ok := rulesByName["Alice Prefers Night"]
	if !ok {
		t.Fatal("missing saved preference rule")
	}
	if preferRule.Type != RuleTypePreferred || preferRule.RuleType != RuleTypePreferred {
		t.Fatalf("prefer rule type aliases mismatch, got type=%q rule_type=%q", preferRule.Type, preferRule.RuleType)
	}
	if len(preferRule.Associations) != 1 {
		t.Fatalf("len(preferRule.Associations) = %d, want 1", len(preferRule.Associations))
	}
	if preferRule.Associations[0].TargetID != "shift-night" || preferRule.Associations[0].Role != AssociationRoleTarget {
		t.Fatalf("prefer rule association = %+v, want target shift-night", preferRule.Associations[0])
	}
	if string(preferRule.Config) != `{"employee_id":"emp-alice","shift_id":"shift-night","type":"prefer_employee","weight":80}` {
		t.Fatalf("preferRule.Config = %s, want normalized shift id", string(preferRule.Config))
	}
	if len(preferRule.ApplyScopes) != 1 || preferRule.ApplyScopes[0].ScopeID == nil || *preferRule.ApplyScopes[0].ScopeID != "emp-alice" {
		t.Fatalf("prefer rule apply scopes = %+v, want Alice employee scope", preferRule.ApplyScopes)
	}
	if len(preferRule.Dependencies) != 1 {
		t.Fatalf("len(preferRule.Dependencies) = %d, want 1", len(preferRule.Dependencies))
	}
	if preferRule.Dependencies[0].DependentRuleID != preferRule.ID || preferRule.Dependencies[0].DependentOnRuleID != sourceRule.ID {
		t.Fatalf("prefer rule dependency = %+v, want dependent=%q dependent_on=%q", preferRule.Dependencies[0], preferRule.ID, sourceRule.ID)
	}
	if len(preferRule.Conflicts) != 1 {
		t.Fatalf("len(preferRule.Conflicts) = %d, want 1", len(preferRule.Conflicts))
	}
	if preferRule.Conflicts[0].RuleID1 != sourceRule.ID || preferRule.Conflicts[0].RuleID2 != preferRule.ID {
		t.Fatalf("prefer rule conflict = %+v, want rule_id_1=%q rule_id_2=%q", preferRule.Conflicts[0], sourceRule.ID, preferRule.ID)
	}

	loaded, err := svc.GetByID(ctx, preferRule.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(loaded.ApplyScopes) != 1 || loaded.ApplyScopes[0].ScopeID == nil || *loaded.ApplyScopes[0].ScopeID != "emp-alice" {
		t.Fatalf("loaded apply scopes = %+v, want hydrated employee scope", loaded.ApplyScopes)
	}
	if len(loaded.Dependencies) != 1 || loaded.Dependencies[0].DependentOnRuleID != sourceRule.ID {
		t.Fatalf("loaded dependencies = %+v, want hydrated dependency on source rule", loaded.Dependencies)
	}
	if len(loaded.Conflicts) != 1 || loaded.Conflicts[0].RuleID1 != sourceRule.ID {
		t.Fatalf("loaded conflicts = %+v, want hydrated conflict with source rule", loaded.Conflicts)
	}
}

func TestService_ParseBatchResultToEffectiveRules(t *testing.T) {
	svc, db, _, _, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)

	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-day', '` + dept.ID + `', 'DAY', 'Day')`,
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-night', '` + dept.ID + `', 'NIGHT', 'Night')`,
		`INSERT INTO employees (id, org_node_id, name) VALUES ('emp-alice', '` + dept.ID + `', 'Alice')`,
		`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-a', '` + dept.ID + `', 'TeamA')`,
	}
	for _, statement := range seedStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("插入端到端规则测试数据失败: %v", err)
		}
	}

	parser := ruleparse.NewParser(nil, zap.NewNop())
	parsed, err := parser.ParseBatchFromContent(`
以下是解析结果：
{
	"rules": [
		{
			"name": "Night Source",
			"rule_type": "source",
			"category": "dependency",
			"sub_type": "source",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 10,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "night depends on day",
			"config": {"type":"staff_source","target_shift_id":"NIGHT","source_shift_id":"DAY"},
			"subject_shifts": ["NIGHT"],
			"object_shifts": ["DAY"],
			"scope_type": "group",
			"scope_groups": ["TeamA"]
		},
		{
			"name": "Alice Prefers Night",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 20,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "alice prefers night",
			"config": {"type":"prefer_employee","employee_id":"emp-alice","shift_id":"shift-night","weight":80},
			"target_shifts": ["NIGHT"],
			"scope_type": "employee",
			"scope_employees": ["Alice"]
		}
	],
	"dependencies": [{
		"dependent_rule_name": "Alice Prefers Night",
		"dependent_on_rule_name": "Night Source",
		"dependency_type": "source",
		"description": "prefer night requires night source"
	}],
	"conflicts": [{
		"rule_name_1": "Night Source",
		"rule_name_2": "Alice Prefers Night",
		"conflict_type": "exclusive",
		"description": "test conflict"
	}],
	"reasoning": "ok"
}
解析完成。
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}

	saved, err := svc.BatchCreateParsed(ctx, batchCreateInputFromParseResult(parsed))
	if err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("len(saved) = %d, want 2", len(saved))
	}

	effective, err := svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 2 {
		t.Fatalf("len(effective.Rules) = %d, want 2", len(effective.Rules))
	}

	rulesByName := make(map[string]Rule, len(effective.Rules))
	for _, current := range effective.Rules {
		rulesByName[current.Name] = current
		if effective.SourceMap[current.ID] != "本级" {
			t.Fatalf("sourceMap[%q] = %q, want %q", current.ID, effective.SourceMap[current.ID], "本级")
		}
	}

	sourceRule, ok := rulesByName["Night Source"]
	if !ok {
		t.Fatal("effective rules missing Night Source")
	}
	preferRule, ok := rulesByName["Alice Prefers Night"]
	if !ok {
		t.Fatal("effective rules missing Alice Prefers Night")
	}
	if len(sourceRule.ApplyScopes) != 1 || sourceRule.ApplyScopes[0].ScopeID == nil || *sourceRule.ApplyScopes[0].ScopeID != "grp-a" {
		t.Fatalf("sourceRule.ApplyScopes = %+v, want TeamA group scope", sourceRule.ApplyScopes)
	}
	if len(preferRule.ApplyScopes) != 1 || preferRule.ApplyScopes[0].ScopeID == nil || *preferRule.ApplyScopes[0].ScopeID != "emp-alice" {
		t.Fatalf("preferRule.ApplyScopes = %+v, want Alice employee scope", preferRule.ApplyScopes)
	}
	if len(preferRule.Dependencies) != 1 || preferRule.Dependencies[0].DependentOnRuleID != sourceRule.ID {
		t.Fatalf("preferRule.Dependencies = %+v, want dependency on source rule", preferRule.Dependencies)
	}

	ordered := OrderRulesForExecution(effective.Rules)
	if len(ordered) != 2 {
		t.Fatalf("len(ordered) = %d, want 2", len(ordered))
	}
	if ordered[0].Name != "Night Source" || ordered[1].Name != "Alice Prefers Night" {
		t.Fatalf("ordered names = [%s %s], want [Night Source Alice Prefers Night]", ordered[0].Name, ordered[1].Name)
	}
}

func TestService_BatchCreateParsed_RejectsUnknownShiftReference(t *testing.T) {
	svc, db, _, _, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)

	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-day', '` + dept.ID + `', 'DAY', 'Day')`,
	}
	for _, statement := range seedStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("插入班次测试数据失败: %v", err)
		}
	}

	_, err := svc.BatchCreateParsed(ctx, BatchCreateInput{
		ParsedRules: []ParsedRuleInput{{
			CreateInput: CreateInput{
				Name:       "Broken Source",
				RuleType:   RuleTypeSource,
				Category:   CategoryDependency,
				SubType:    SubTypeSource,
				ApplyScope: ApplyScopeGlobal,
				TimeScope:  TimeScopeSameDay,
				Config:     []byte(`{"type":"staff_source","target_shift_id":"NIGHT","source_shift_id":"UNKNOWN"}`),
				Priority:   10,
			},
			SubjectShifts: []string{"NIGHT"},
			ObjectShifts:  []string{"UNKNOWN"},
		}},
	})
	if !errors.Is(err, ErrShiftReferenceNotFound) {
		t.Fatalf("BatchCreateParsed() error = %v, want %v", err, ErrShiftReferenceNotFound)
	}
}

func batchCreateInputFromParseResult(result *ruleparse.ParseBatchResult) BatchCreateInput {
	parsedRules := make([]ParsedRuleInput, 0, len(result.Rules))
	for _, current := range result.Rules {
		parsedRule := ParsedRuleInput{
			CreateInput: CreateInput{
				Name:            current.Name,
				Type:            current.Type,
				RuleType:        current.RuleType,
				Category:        current.Category,
				SubType:         current.SubType,
				ApplyScope:      current.ApplyScope,
				TimeScope:       current.TimeScope,
				TimeOffsetDays:  current.TimeOffsetDays,
				RuleData:        stringPtr(current.RuleData),
				Config:          current.Config,
				Priority:        current.Priority,
				SourceType:      current.SourceType,
				ParseConfidence: current.ParseConfidence,
				Version:         current.Version,
				Description:     stringPtr(current.Description),
			},
			SubjectShifts:  append([]string(nil), current.SubjectShifts...),
			ObjectShifts:   append([]string(nil), current.ObjectShifts...),
			TargetShifts:   append([]string(nil), current.TargetShifts...),
			ScopeType:      current.ScopeType,
			ScopeEmployees: append([]string(nil), current.ScopeEmployees...),
			ScopeGroups:    append([]string(nil), current.ScopeGroups...),
		}
		parsedRules = append(parsedRules, parsedRule)
	}

	dependencies := make([]RuleDependency, 0, len(result.Dependencies))
	for _, current := range result.Dependencies {
		dependencies = append(dependencies, RuleDependency{
			DependentRuleName: current.DependentRuleName,
			DependentOnName:   current.DependentOnRuleName,
			DependencyType:    current.DependencyType,
			Description:       current.Description,
		})
	}

	conflicts := make([]RuleConflict, 0, len(result.Conflicts))
	for _, current := range result.Conflicts {
		conflicts = append(conflicts, RuleConflict{
			RuleName1:    current.RuleName1,
			RuleName2:    current.RuleName2,
			ConflictType: current.ConflictType,
			Description:  current.Description,
		})
	}

	return BatchCreateInput{ParsedRules: parsedRules, Dependencies: dependencies, Conflicts: conflicts}
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	copy := value
	return &copy
}
