package step

import (
	"context"
	"testing"

	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type scheduleRuleNodeResolver struct {
	nodes map[string]tenant.OrgNode
}

func (m *scheduleRuleNodeResolver) GetByID(_ context.Context, id string) (*tenant.OrgNode, error) {
	node, ok := m.nodes[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &node, nil
}

func setupScheduleRuleService(t *testing.T) (*rule.Service, *gorm.DB, tenant.OrgNode) {
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

	svc := rule.NewService(rule.NewRepository(db), tenant.NewRepository(db))
	svc.SetOrgNodeResolver(&scheduleRuleNodeResolver{nodes: map[string]tenant.OrgNode{root.ID: root, org.ID: org, dept.ID: dept}})
	return svc, db, dept
}

func TestPhasePipeline_UsesEffectiveRulesFromBatchParsedRules(t *testing.T) {
	svc, db, dept := setupScheduleRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)

	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-day', '` + dept.ID + `', 'DAY', 'Day')`,
		`INSERT INTO shifts (id, org_node_id, code, name) VALUES ('shift-night', '` + dept.ID + `', 'NIGHT', 'Night')`,
		`INSERT INTO employees (id, org_node_id, name) VALUES ('emp-alice', '` + dept.ID + `', 'Alice')`,
		`INSERT INTO employees (id, org_node_id, name) VALUES ('emp-bob', '` + dept.ID + `', 'Bob')`,
	}
	for _, statement := range seedStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("插入测试主数据失败: %v", err)
		}
	}

	parser := ruleparse.NewParser(nil, zap.NewNop())
	parsed, err := parser.ParseBatchFromContent(`
{
	"rules": [
		{
			"name": "Fixed Day Staff",
			"rule_type": "required_together",
			"category": "constraint",
			"sub_type": "must",
			"apply_scope": "global",
			"time_scope": "same_day",
			"priority": 5,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "assign Alice and Bob to the day shift on the source date",
			"config": {"type":"fixed_schedule","employee_ids":["emp-alice","emp-bob"],"shift_id":"shift-day"},
			"target_shifts": ["Day"],
			"scope_type": "all"
		},
		{
			"name": "Night Source",
			"rule_type": "source",
			"category": "dependency",
			"sub_type": "source",
			"apply_scope": "global",
			"time_scope": "same_day",
			"time_offset_days": -1,
			"priority": 10,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "night shift must come from previous day shift",
			"config": {"type":"staff_source","target_shift_id":"shift-night","source_shift_id":"shift-day"},
			"subject_shifts": ["Night"],
			"object_shifts": ["Day"],
			"scope_type": "all"
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
			"description": "alice should win the night slot",
			"config": {"type":"prefer_employee","employee_id":"emp-alice","shift_id":"shift-night","weight":80},
			"target_shifts": ["Night"],
			"scope_type": "employee",
			"scope_employees": ["Alice"]
		}
	],
	"dependencies": [{
		"dependent_rule_name": "Alice Prefers Night",
		"dependent_on_rule_name": "Night Source",
		"dependency_type": "source",
		"description": "prefer rule evaluated after source rule"
	}],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}

	if _, err := svc.BatchCreateParsed(ctx, scheduleBatchInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	effective, err := svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("ListEffective() error = %v", err)
	}

	state := NewScheduleState("sch-1", dept.ID, "", "2026-03-22", "2026-03-23", "user-1", &ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	state.ShiftOrder = makeShifts("shift-day", "shift-night")
	state.EffectiveRules = rule.OrderRulesForExecution(effective.Rules)
	state.Candidates["shift-night|2026-03-23"] = []string{"emp-alice", "emp-bob", "emp-charlie"}

	if err := (&PhaseZeroStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("PhaseZeroStep.Execute() error = %v", err)
	}
	if got := state.CountAssigned("shift-day", "2026-03-22"); got != 2 {
		t.Fatalf("expected 2 fixed day assignments, got %d", got)
	}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("PhaseOneStep.Execute() error = %v", err)
	}
	if got, want := state.Candidates["shift-night|2026-03-23"], []string{"emp-alice", "emp-bob"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("PhaseTwoStep.Execute() error = %v", err)
	}

	if got := len(state.Assignments); got != 3 {
		t.Fatalf("expected 3 assignments after full phase run, got %d", got)
	}
	if got := state.CountAssigned("shift-night", "2026-03-23"); got != 1 {
		t.Fatalf("expected 1 night assignment, got %d", got)
	}

	var nightAssignment Assignment
	foundNight := false
	for _, assignment := range state.Assignments {
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			nightAssignment = assignment
			foundNight = true
			break
		}
	}
	if !foundNight {
		t.Fatal("missing night assignment after PhaseTwo")
	}
	if nightAssignment.EmployeeID != "emp-alice" {
		t.Fatalf("expected Alice to win preferred night assignment, got %s", nightAssignment.EmployeeID)
	}
	if nightAssignment.Source != SourceFill {
		t.Fatalf("expected night assignment source %q, got %q", SourceFill, nightAssignment.Source)
	}
	if len(state.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(state.Violations))
	}
}

func scheduleBatchInputFromParseResult(result *ruleparse.ParseBatchResult) rule.BatchCreateInput {
	parsedRules := make([]rule.ParsedRuleInput, 0, len(result.Rules))
	for _, current := range result.Rules {
		parsedRules = append(parsedRules, rule.ParsedRuleInput{
			CreateInput: rule.CreateInput{
				Name:            current.Name,
				Type:            current.Type,
				RuleType:        current.RuleType,
				Category:        current.Category,
				SubType:         current.SubType,
				ApplyScope:      current.ApplyScope,
				TimeScope:       current.TimeScope,
				TimeOffsetDays:  current.TimeOffsetDays,
				RuleData:        scheduleStringPtr(current.RuleData),
				Config:          current.Config,
				Priority:        current.Priority,
				SourceType:      current.SourceType,
				ParseConfidence: current.ParseConfidence,
				Version:         current.Version,
				Description:     scheduleStringPtr(current.Description),
			},
			SubjectShifts:  append([]string(nil), current.SubjectShifts...),
			ObjectShifts:   append([]string(nil), current.ObjectShifts...),
			TargetShifts:   append([]string(nil), current.TargetShifts...),
			ScopeType:      current.ScopeType,
			ScopeEmployees: append([]string(nil), current.ScopeEmployees...),
			ScopeGroups:    append([]string(nil), current.ScopeGroups...),
		})
	}

	dependencies := make([]rule.RuleDependency, 0, len(result.Dependencies))
	for _, current := range result.Dependencies {
		dependencies = append(dependencies, rule.RuleDependency{
			DependentRuleName: current.DependentRuleName,
			DependentOnName:   current.DependentOnRuleName,
			DependencyType:    current.DependencyType,
			Description:       current.Description,
		})
	}

	conflicts := make([]rule.RuleConflict, 0, len(result.Conflicts))
	for _, current := range result.Conflicts {
		conflicts = append(conflicts, rule.RuleConflict{
			RuleName1:    current.RuleName1,
			RuleName2:    current.RuleName2,
			ConflictType: current.ConflictType,
			Description:  current.Description,
		})
	}

	return rule.BatchCreateInput{ParsedRules: parsedRules, Dependencies: dependencies, Conflicts: conflicts}
}

func scheduleStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	copy := value
	return &copy
}
