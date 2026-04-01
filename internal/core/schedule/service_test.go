package schedule

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/leave"
	"gantt-saas/internal/core/rule"
	step "gantt-saas/internal/core/schedule/step"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type stubScheduleGroupProvider struct {
	members map[string][]string
}

func (s stubScheduleGroupProvider) GetMemberEmployeeIDs(_ context.Context, groupID string) ([]string, error) {
	return s.members[groupID], nil
}

type scheduleGenerateTestEnv struct {
	db       *gorm.DB
	ctx      context.Context
	repo     *Repository
	ruleSvc  *rule.Service
	shiftSvc *shift.Service
	svc      *Service
}

var scheduleGenerateSchemaStatements = []string{
	`CREATE TABLE schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		group_id TEXT,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		status TEXT NOT NULL,
		pipeline_type TEXT NOT NULL,
		config TEXT,
		created_by TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		org_node_id TEXT NOT NULL
	)`,
	`CREATE TABLE schedule_assignments (
		id TEXT PRIMARY KEY,
		schedule_id TEXT NOT NULL,
		employee_id TEXT NOT NULL,
		shift_id TEXT NOT NULL,
		date TEXT NOT NULL,
		source TEXT NOT NULL,
		created_at DATETIME,
		org_node_id TEXT NOT NULL
	)`,
	`CREATE TABLE leaves (
		id TEXT PRIMARY KEY,
		employee_id TEXT NOT NULL,
		leave_type TEXT NOT NULL,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		reason TEXT,
		status TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		org_node_id TEXT NOT NULL
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
		name TEXT NOT NULL,
		code TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'regular',
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		duration INTEGER NOT NULL,
		is_cross_day BOOLEAN NOT NULL DEFAULT FALSE,
		color TEXT,
		priority INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		description TEXT,
		metadata TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`,
	`CREATE TABLE shift_dependencies (
		id TEXT PRIMARY KEY,
		org_node_id TEXT NOT NULL,
		shift_id TEXT NOT NULL,
		depends_on_id TEXT NOT NULL,
		dependency_type TEXT NOT NULL,
		created_at DATETIME
	)`,
	`CREATE TABLE shift_groups (
		id TEXT PRIMARY KEY,
		shift_id TEXT NOT NULL,
		group_id TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		notes TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		org_node_id TEXT NOT NULL
	)`,
	`CREATE TABLE shift_weekly_staff (
		id TEXT PRIMARY KEY,
		shift_id TEXT NOT NULL,
		weekday INTEGER NOT NULL,
		staff_count INTEGER NOT NULL DEFAULT 0,
		is_custom BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME,
		updated_at DATETIME,
		org_node_id TEXT NOT NULL
	)`,
	`CREATE TABLE fixed_assignments (
		id TEXT PRIMARY KEY,
		shift_id TEXT NOT NULL,
		employee_id TEXT NOT NULL,
		pattern_type TEXT NOT NULL,
		weekdays TEXT,
		week_pattern TEXT,
		monthdays TEXT,
		specific_dates TEXT,
		start_date TEXT,
		end_date TEXT,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at DATETIME,
		updated_at DATETIME,
		org_node_id TEXT NOT NULL
	)`,
	`CREATE TABLE employees (
		id TEXT PRIMARY KEY,
		org_node_id TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active'
	)`,
	`CREATE TABLE employee_groups (
		id TEXT PRIMARY KEY,
		org_node_id TEXT NOT NULL,
		name TEXT NOT NULL
	)`,
}

func newScheduleGenerateTestEnv(t *testing.T, seedStatements []string, groupMembers map[string][]string) *scheduleGenerateTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	for _, statement := range scheduleGenerateSchemaStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}
	for _, statement := range seedStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("插入测试数据失败: %v", err)
		}
	}

	ctx := tenant.WithOrgNode(context.Background(), "dept-001", "/org-root/dept-001")
	repo := NewRepository(db)
	ruleSvc := rule.NewService(rule.NewRepository(db), tenant.NewRepository(db))
	shiftSvc := shift.NewService(shift.NewRepository(db))
	svc := NewService(
		repo,
		ruleSvc,
		shiftSvc,
		employee.NewRepository(db),
		leave.NewRepository(db),
		zap.NewNop(),
	)
	if groupMembers != nil {
		svc.SetGroupMemberProvider(stubScheduleGroupProvider{members: groupMembers})
	}

	return &scheduleGenerateTestEnv{
		db:       db,
		ctx:      ctx,
		repo:     repo,
		ruleSvc:  ruleSvc,
		shiftSvc: shiftSvc,
		svc:      svc,
	}
}

func TestService_GetSelfAssignments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	statements := []string{
		`CREATE TABLE schedules (id TEXT PRIMARY KEY, name TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL, status TEXT NOT NULL, pipeline_type TEXT, config TEXT, created_by TEXT NOT NULL, created_at DATETIME, updated_at DATETIME, org_node_id TEXT NOT NULL)`,
		`CREATE TABLE shifts (id TEXT PRIMARY KEY, name TEXT NOT NULL, code TEXT NOT NULL, start_time TEXT NOT NULL, end_time TEXT NOT NULL, duration INTEGER NOT NULL, is_cross_day BOOLEAN NOT NULL DEFAULT FALSE, color TEXT, priority INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, created_at DATETIME, updated_at DATETIME, org_node_id TEXT NOT NULL)`,
		`CREATE TABLE schedule_assignments (id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, employee_id TEXT NOT NULL, shift_id TEXT NOT NULL, date TEXT NOT NULL, source TEXT NOT NULL, created_at DATETIME, org_node_id TEXT NOT NULL)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}

	ctx := tenant.WithOrgNode(t.Context(), "dept-001", "/org-root/dept-001")
	if err := db.Exec(`INSERT INTO schedules (id, name, start_date, end_date, status, created_by, org_node_id) VALUES ('sch-pub', '已发布排班', '2026-03-24', '2026-03-30', 'published', 'tester', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建已发布排班失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedules (id, name, start_date, end_date, status, created_by, org_node_id) VALUES ('sch-draft', '草稿排班', '2026-03-24', '2026-03-30', 'draft', 'tester', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建草稿排班失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO shifts (id, name, code, start_time, end_time, duration, color, status, org_node_id) VALUES ('shift-day', '白班', 'DAY', '08:00', '16:00', 480, '#409EFF', 'active', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建班次失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-001', 'sch-pub', 'emp-001', 'shift-day', '2026-03-26', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建已发布排班分配失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-002', 'sch-draft', 'emp-001', 'shift-day', '2026-03-27', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建草稿排班分配失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-003', 'sch-pub', 'emp-002', 'shift-day', '2026-03-26', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建其他员工排班分配失败: %v", err)
	}

	svc := NewService(NewRepository(db), nil, nil, nil, nil, zap.NewNop())
	items, err := svc.GetSelfAssignments(ctx, "emp-001", "2026-03-24", "2026-03-30")
	if err != nil {
		t.Fatalf("GetSelfAssignments() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ScheduleID != "sch-pub" {
		t.Fatalf("schedule_id = %s, want sch-pub", items[0].ScheduleID)
	}
	if items[0].ShiftName != "白班" {
		t.Fatalf("shift_name = %s, want 白班", items[0].ShiftName)
	}
}

func TestService_Generate_UsesEffectiveRulesFromBatchParsedRules(t *testing.T) {
	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-day', 'dept-001', 'Day', 'day', 'regular', '08:00', '16:00', 480, 0, '#111111', 1, 'active')`,
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-night', 'dept-001', 'Night', 'night', 'regular', '20:00', '08:00', 720, 1, '#222222', 2, 'active')`,
		`INSERT INTO shift_dependencies (id, org_node_id, shift_id, depends_on_id, dependency_type) VALUES ('dep-night-after-day', 'dept-001', 'shift-night', 'shift-day', 'order')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-alice', 'dept-001', 'Alice', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-bob', 'dept-001', 'Bob', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-charlie', 'dept-001', 'Charlie', 'active')`,
	}
	env := newScheduleGenerateTestEnv(t, seedStatements, nil)
	db := env.db
	ctx := env.ctx
	ruleSvc := env.ruleSvc
	svc := env.svc

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
			"description": "assign Alice and Bob to the day shift",
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
			"description": "night shift requires previous day source",
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
			"description": "alice should win night",
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
		"description": "prefer after source"
	}],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if _, err := ruleSvc.BatchCreateParsed(ctx, scheduleBatchCreateInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	configBytes, err := json.Marshal(step.ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	if err != nil {
		t.Fatalf("Marshal(schedule config) error = %v", err)
	}

	sch, err := svc.Create(ctx, CreateInput{
		Name:         "规则驱动排班",
		StartDate:    "2026-03-22",
		EndDate:      "2026-03-23",
		PipelineType: PipelineDeterministic,
		Config:       configBytes,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := svc.Generate(ctx, sch.ID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.ScheduleID != sch.ID {
		t.Fatalf("result.ScheduleID = %q, want %q", result.ScheduleID, sch.ID)
	}
	if result.Status != StatusReview {
		t.Fatalf("result.Status = %q, want %q", result.Status, StatusReview)
	}
	if result.AssignmentsCount != 3 {
		t.Fatalf("result.AssignmentsCount = %d, want 3", result.AssignmentsCount)
	}
	if result.ViolationsCount != 0 {
		t.Fatalf("result.ViolationsCount = %d, want 0", result.ViolationsCount)
	}

	loaded, err := svc.GetByID(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if loaded.Status != StatusReview {
		t.Fatalf("loaded.Status = %q, want %q", loaded.Status, StatusReview)
	}

	assignments, err := svc.GetAssignments(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}

	dayAssignments := 0
	foundAliceNight := false
	for _, assignment := range assignments {
		if assignment.ShiftID == "shift-day" && assignment.Date == "2026-03-22" {
			dayAssignments++
			if assignment.Source != step.SourceFixed {
				t.Fatalf("day assignment source = %q, want %q", assignment.Source, step.SourceFixed)
			}
		}
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			if assignment.EmployeeID != "emp-alice" {
				t.Fatalf("night assignment employee = %q, want emp-alice", assignment.EmployeeID)
			}
			if assignment.Source != step.SourceFill {
				t.Fatalf("night assignment source = %q, want %q", assignment.Source, step.SourceFill)
			}
			foundAliceNight = true
		}
	}
	if dayAssignments != 2 {
		t.Fatalf("dayAssignments = %d, want 2", dayAssignments)
	}
	if !foundAliceNight {
		t.Fatal("expected generated night assignment for Alice")
	}
	if err := db.Model(&Schedule{}).Where("id = ?", sch.ID).Select("status").Scan(&loaded).Error; err != nil {
		t.Fatalf("scan generated schedule status error = %v", err)
	}
	if loaded.Status != StatusReview {
		t.Fatalf("persisted status = %q, want %q", loaded.Status, StatusReview)
	}
}

func TestService_Generate_GroupScopedPreferenceUsesGroupMembers(t *testing.T) {
	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-day', 'dept-001', 'Day', 'day', 'regular', '08:00', '16:00', 480, 0, '#111111', 1, 'active')`,
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-night', 'dept-001', 'Night', 'night', 'regular', '20:00', '08:00', 720, 1, '#222222', 2, 'active')`,
		`INSERT INTO shift_dependencies (id, org_node_id, shift_id, depends_on_id, dependency_type) VALUES ('dep-night-after-day', 'dept-001', 'shift-night', 'shift-day', 'order')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-alice', 'dept-001', 'Alice', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-bob', 'dept-001', 'Bob', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-charlie', 'dept-001', 'Charlie', 'active')`,
		`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-night', 'dept-001', 'NightTeam')`,
	}
	env := newScheduleGenerateTestEnv(t, seedStatements, map[string][]string{"grp-night": {"emp-bob"}})
	ctx := env.ctx
	ruleSvc := env.ruleSvc
	svc := env.svc

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
			"description": "assign Alice and Bob to the day shift",
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
			"description": "night shift requires previous day source",
			"config": {"type":"staff_source","target_shift_id":"shift-night","source_shift_id":"shift-day"},
			"subject_shifts": ["Night"],
			"object_shifts": ["Day"],
			"scope_type": "all"
		},
		{
			"name": "Night Team Prefers Night",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 20,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "night team should win night shifts",
			"config": {"type":"prefer_employee","employee_id":"emp-bob","shift_id":"shift-night","weight":80},
			"target_shifts": ["Night"],
			"scope_type": "group",
			"scope_groups": ["NightTeam"]
		}
	],
	"dependencies": [{
		"dependent_rule_name": "Night Team Prefers Night",
		"dependent_on_rule_name": "Night Source",
		"dependency_type": "source",
		"description": "prefer after source"
	}],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if _, err := ruleSvc.BatchCreateParsed(ctx, scheduleBatchCreateInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	configBytes, err := json.Marshal(step.ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	if err != nil {
		t.Fatalf("Marshal(schedule config) error = %v", err)
	}

	sch, err := svc.Create(ctx, CreateInput{
		Name:         "分组作用域规则驱动排班",
		StartDate:    "2026-03-22",
		EndDate:      "2026-03-23",
		PipelineType: PipelineDeterministic,
		Config:       configBytes,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := svc.Generate(ctx, sch.ID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Status != StatusReview {
		t.Fatalf("result.Status = %q, want %q", result.Status, StatusReview)
	}
	if result.AssignmentsCount != 3 {
		t.Fatalf("result.AssignmentsCount = %d, want 3", result.AssignmentsCount)
	}
	if result.ViolationsCount != 0 {
		t.Fatalf("result.ViolationsCount = %d, want 0", result.ViolationsCount)
	}

	assignments, err := svc.GetAssignments(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}

	dayAssignments := 0
	foundBobNight := false
	for _, assignment := range assignments {
		if assignment.ShiftID == "shift-day" && assignment.Date == "2026-03-22" {
			dayAssignments++
		}
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			if assignment.EmployeeID != "emp-bob" {
				t.Fatalf("night assignment employee = %q, want emp-bob", assignment.EmployeeID)
			}
			if assignment.Source != step.SourceFill {
				t.Fatalf("night assignment source = %q, want %q", assignment.Source, step.SourceFill)
			}
			foundBobNight = true
		}
	}
	if dayAssignments != 2 {
		t.Fatalf("dayAssignments = %d, want 2", dayAssignments)
	}
	if !foundBobNight {
		t.Fatal("expected generated night assignment for Bob from group-scoped preference")
	}
}

func TestService_Generate_ExcludeGroupPreferenceSkipsExcludedMembers(t *testing.T) {
	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-day', 'dept-001', 'Day', 'day', 'regular', '08:00', '16:00', 480, 0, '#111111', 1, 'active')`,
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-night', 'dept-001', 'Night', 'night', 'regular', '20:00', '08:00', 720, 1, '#222222', 2, 'active')`,
		`INSERT INTO shift_dependencies (id, org_node_id, shift_id, depends_on_id, dependency_type) VALUES ('dep-night-after-day', 'dept-001', 'shift-night', 'shift-day', 'order')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-alice', 'dept-001', 'Alice', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-bob', 'dept-001', 'Bob', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-charlie', 'dept-001', 'Charlie', 'active')`,
		`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-night', 'dept-001', 'NightTeam')`,
	}
	env := newScheduleGenerateTestEnv(t, seedStatements, map[string][]string{"grp-night": {"emp-bob"}})
	ctx := env.ctx
	ruleSvc := env.ruleSvc
	svc := env.svc

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
			"description": "assign Alice and Bob to the day shift",
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
			"description": "night shift requires previous day source",
			"config": {"type":"staff_source","target_shift_id":"shift-night","source_shift_id":"shift-day"},
			"subject_shifts": ["Night"],
			"object_shifts": ["Day"],
			"scope_type": "all"
		},
		{
			"name": "Excluded Night Team Bob Prefers Night",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 20,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "bob prefers night except excluded group",
			"config": {"type":"prefer_employee","employee_id":"emp-bob","shift_id":"shift-night","weight":80},
			"target_shifts": ["Night"],
			"scope_type": "exclude_group",
			"scope_groups": ["NightTeam"]
		},
		{
			"name": "Alice Baseline Night Preference",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 30,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "alice baseline night preference",
			"config": {"type":"prefer_employee","employee_id":"emp-alice","shift_id":"shift-night","weight":40},
			"target_shifts": ["Night"],
			"scope_type": "all"
		}
	],
	"dependencies": [
		{
			"dependent_rule_name": "Excluded Night Team Bob Prefers Night",
			"dependent_on_rule_name": "Night Source",
			"dependency_type": "source",
			"description": "prefer after source"
		},
		{
			"dependent_rule_name": "Alice Baseline Night Preference",
			"dependent_on_rule_name": "Night Source",
			"dependency_type": "source",
			"description": "prefer after source"
		}
	],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if _, err := ruleSvc.BatchCreateParsed(ctx, scheduleBatchCreateInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	configBytes, err := json.Marshal(step.ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	if err != nil {
		t.Fatalf("Marshal(schedule config) error = %v", err)
	}

	sch, err := svc.Create(ctx, CreateInput{
		Name:         "排除分组作用域规则驱动排班",
		StartDate:    "2026-03-22",
		EndDate:      "2026-03-23",
		PipelineType: PipelineDeterministic,
		Config:       configBytes,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := svc.Generate(ctx, sch.ID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Status != StatusReview {
		t.Fatalf("result.Status = %q, want %q", result.Status, StatusReview)
	}
	if result.AssignmentsCount != 3 {
		t.Fatalf("result.AssignmentsCount = %d, want 3", result.AssignmentsCount)
	}
	if result.ViolationsCount != 0 {
		t.Fatalf("result.ViolationsCount = %d, want 0", result.ViolationsCount)
	}

	assignments, err := svc.GetAssignments(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}

	dayAssignments := 0
	foundAliceNight := false
	for _, assignment := range assignments {
		if assignment.ShiftID == "shift-day" && assignment.Date == "2026-03-22" {
			dayAssignments++
		}
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			if assignment.EmployeeID != "emp-alice" {
				t.Fatalf("night assignment employee = %q, want emp-alice", assignment.EmployeeID)
			}
			if assignment.Source != step.SourceFill {
				t.Fatalf("night assignment source = %q, want %q", assignment.Source, step.SourceFill)
			}
			foundAliceNight = true
		}
	}
	if dayAssignments != 2 {
		t.Fatalf("dayAssignments = %d, want 2", dayAssignments)
	}
	if !foundAliceNight {
		t.Fatal("expected generated night assignment for Alice after exclude_group removed Bob preference")
	}
}

func TestService_Generate_ExcludeEmployeePreferenceSkipsExcludedEmployee(t *testing.T) {
	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-day', 'dept-001', 'Day', 'day', 'regular', '08:00', '16:00', 480, 0, '#111111', 1, 'active')`,
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-night', 'dept-001', 'Night', 'night', 'regular', '20:00', '08:00', 720, 1, '#222222', 2, 'active')`,
		`INSERT INTO shift_dependencies (id, org_node_id, shift_id, depends_on_id, dependency_type) VALUES ('dep-night-after-day', 'dept-001', 'shift-night', 'shift-day', 'order')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-alice', 'dept-001', 'Alice', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-bob', 'dept-001', 'Bob', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-charlie', 'dept-001', 'Charlie', 'active')`,
	}
	env := newScheduleGenerateTestEnv(t, seedStatements, nil)
	ctx := env.ctx
	ruleSvc := env.ruleSvc
	svc := env.svc

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
			"description": "assign Alice and Bob to the day shift",
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
			"description": "night shift requires previous day source",
			"config": {"type":"staff_source","target_shift_id":"shift-night","source_shift_id":"shift-day"},
			"subject_shifts": ["Night"],
			"object_shifts": ["Day"],
			"scope_type": "all"
		},
		{
			"name": "Excluded Bob Prefers Night",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 20,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "bob prefers night except excluded employee",
			"config": {"type":"prefer_employee","employee_id":"emp-bob","shift_id":"shift-night","weight":80},
			"target_shifts": ["Night"],
			"scope_type": "exclude_employee",
			"scope_employees": ["Bob"]
		},
		{
			"name": "Alice Baseline Night Preference",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 30,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "alice baseline night preference",
			"config": {"type":"prefer_employee","employee_id":"emp-alice","shift_id":"shift-night","weight":40},
			"target_shifts": ["Night"],
			"scope_type": "all"
		}
	],
	"dependencies": [
		{
			"dependent_rule_name": "Excluded Bob Prefers Night",
			"dependent_on_rule_name": "Night Source",
			"dependency_type": "source",
			"description": "prefer after source"
		},
		{
			"dependent_rule_name": "Alice Baseline Night Preference",
			"dependent_on_rule_name": "Night Source",
			"dependency_type": "source",
			"description": "prefer after source"
		}
	],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if _, err := ruleSvc.BatchCreateParsed(ctx, scheduleBatchCreateInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	configBytes, err := json.Marshal(step.ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	if err != nil {
		t.Fatalf("Marshal(schedule config) error = %v", err)
	}

	sch, err := svc.Create(ctx, CreateInput{
		Name:         "排除员工作用域规则驱动排班",
		StartDate:    "2026-03-22",
		EndDate:      "2026-03-23",
		PipelineType: PipelineDeterministic,
		Config:       configBytes,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := svc.Generate(ctx, sch.ID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Status != StatusReview {
		t.Fatalf("result.Status = %q, want %q", result.Status, StatusReview)
	}
	if result.AssignmentsCount != 3 {
		t.Fatalf("result.AssignmentsCount = %d, want 3", result.AssignmentsCount)
	}
	if result.ViolationsCount != 0 {
		t.Fatalf("result.ViolationsCount = %d, want 0", result.ViolationsCount)
	}

	assignments, err := svc.GetAssignments(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}

	dayAssignments := 0
	foundAliceNight := false
	for _, assignment := range assignments {
		if assignment.ShiftID == "shift-day" && assignment.Date == "2026-03-22" {
			dayAssignments++
		}
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			if assignment.EmployeeID != "emp-alice" {
				t.Fatalf("night assignment employee = %q, want emp-alice", assignment.EmployeeID)
			}
			if assignment.Source != step.SourceFill {
				t.Fatalf("night assignment source = %q, want %q", assignment.Source, step.SourceFill)
			}
			foundAliceNight = true
		}
	}
	if dayAssignments != 2 {
		t.Fatalf("dayAssignments = %d, want 2", dayAssignments)
	}
	if !foundAliceNight {
		t.Fatal("expected generated night assignment for Alice after exclude_employee removed Bob preference")
	}
}

func TestService_Generate_ReturnsCrossGroupConflictViolations(t *testing.T) {
	seedStatements := []string{
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-day', 'dept-001', 'Day', 'day', 'regular', '08:00', '16:00', 480, 0, '#111111', 1, 'active')`,
		`INSERT INTO shifts (id, org_node_id, name, code, type, start_time, end_time, duration, is_cross_day, color, priority, status) VALUES ('shift-night', 'dept-001', 'Night', 'night', 'regular', '20:00', '08:00', 720, 1, '#222222', 2, 'active')`,
		`INSERT INTO shift_dependencies (id, org_node_id, shift_id, depends_on_id, dependency_type) VALUES ('dep-night-after-day', 'dept-001', 'shift-night', 'shift-day', 'order')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-alice', 'dept-001', 'Alice', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-bob', 'dept-001', 'Bob', 'active')`,
		`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-charlie', 'dept-001', 'Charlie', 'active')`,
		`INSERT INTO schedules (id, name, start_date, end_date, status, pipeline_type, config, created_by, org_node_id) VALUES ('sch-existing', '其他排班', '2026-03-22', '2026-03-22', 'review', 'deterministic', '{}', 'tester', 'dept-001')`,
		`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-existing', 'sch-existing', 'emp-bob', 'shift-day', '2026-03-22', 'system', 'dept-001')`,
	}
	env := newScheduleGenerateTestEnv(t, seedStatements, nil)
	ctx := env.ctx
	repo := env.repo
	ruleSvc := env.ruleSvc
	svc := env.svc

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
			"description": "assign Alice and Bob to the day shift",
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
			"description": "night shift requires previous day source",
			"config": {"type":"staff_source","target_shift_id":"shift-night","source_shift_id":"shift-day"},
			"subject_shifts": ["Night"],
			"object_shifts": ["Day"],
			"scope_type": "all"
		},
		{
			"name": "Bob Prefers Night",
			"rule_type": "preferred",
			"category": "preference",
			"sub_type": "prefer",
			"apply_scope": "specific",
			"time_scope": "same_day",
			"priority": 20,
			"source_type": "llm_parsed",
			"version": "v4",
			"description": "bob should win night",
			"config": {"type":"prefer_employee","employee_id":"emp-bob","shift_id":"shift-night","weight":80},
			"target_shifts": ["Night"],
			"scope_type": "employee",
			"scope_employees": ["Bob"]
		}
	],
	"dependencies": [{
		"dependent_rule_name": "Bob Prefers Night",
		"dependent_on_rule_name": "Night Source",
		"dependency_type": "source",
		"description": "prefer after source"
	}],
	"reasoning": "ok"
}
`)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if _, err := ruleSvc.BatchCreateParsed(ctx, scheduleBatchCreateInputFromParseResult(parsed)); err != nil {
		t.Fatalf("BatchCreateParsed() error = %v", err)
	}

	configBytes, err := json.Marshal(step.ScheduleConfig{
		ShiftIDs: []string{"shift-day", "shift-night"},
		Requirements: map[string]map[string]int{
			"shift-day":   {"2026-03-22": 2},
			"shift-night": {"2026-03-23": 1},
		},
	})
	if err != nil {
		t.Fatalf("Marshal(schedule config) error = %v", err)
	}

	existingAssignments, err := repo.FindAssignmentsByEmployeeAndDateRange(ctx, "emp-bob", "2026-03-22", "2026-03-23", "")
	if err != nil {
		t.Fatalf("FindAssignmentsByEmployeeAndDateRange() error = %v", err)
	}
	if len(existingAssignments) != 1 {
		t.Fatalf("len(existingAssignments) = %d, want 1", len(existingAssignments))
	}
	if existingAssignments[0].ShiftID != "shift-day" || existingAssignments[0].Date != "2026-03-22" {
		t.Fatalf("existing assignment = %+v, want shift-day on 2026-03-22", existingAssignments[0])
	}

	sch, err := svc.Create(ctx, CreateInput{
		Name:         "跨组冲突规则驱动排班",
		StartDate:    "2026-03-22",
		EndDate:      "2026-03-23",
		PipelineType: PipelineDeterministic,
		Config:       configBytes,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := svc.Generate(ctx, sch.ID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Status != StatusReview {
		t.Fatalf("result.Status = %q, want %q", result.Status, StatusReview)
	}
	if result.AssignmentsCount != 3 {
		t.Fatalf("result.AssignmentsCount = %d, want 3", result.AssignmentsCount)
	}

	loaded, err := svc.GetByID(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if loaded.Status != StatusReview {
		t.Fatalf("loaded.Status = %q, want %q", loaded.Status, StatusReview)
	}

	assignments, err := svc.GetAssignments(ctx, sch.ID)
	if err != nil {
		t.Fatalf("GetAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}
	if result.ViolationsCount != 1 {
		t.Fatalf("result.ViolationsCount = %d, want 1, assignments = %+v", result.ViolationsCount, assignments)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(result.Violations) = %d, want 1", len(result.Violations))
	}
	if result.Violations[0].EmployeeID != "emp-bob" {
		t.Fatalf("violation employee_id = %q, want emp-bob", result.Violations[0].EmployeeID)
	}
	if result.Violations[0].RuleName != "跨组冲突检测" {
		t.Fatalf("violation rule_name = %q, want 跨组冲突检测", result.Violations[0].RuleName)
	}
	if result.Violations[0].ShiftID != "shift-day" {
		t.Fatalf("violation shift_id = %q, want shift-day", result.Violations[0].ShiftID)
	}
	if result.Violations[0].Date != "2026-03-22" {
		t.Fatalf("violation date = %q, want 2026-03-22", result.Violations[0].Date)
	}
	if !strings.Contains(result.Violations[0].Reason, "时段冲突") {
		t.Fatalf("violation reason = %q, want substring 时段冲突", result.Violations[0].Reason)
	}

	dayAssignments := 0
	foundBobNight := false
	for _, assignment := range assignments {
		if assignment.ShiftID == "shift-day" && assignment.Date == "2026-03-22" {
			dayAssignments++
		}
		if assignment.ShiftID == "shift-night" && assignment.Date == "2026-03-23" {
			if assignment.EmployeeID != "emp-bob" {
				t.Fatalf("night assignment employee = %q, want emp-bob", assignment.EmployeeID)
			}
			foundBobNight = true
		}
	}
	if dayAssignments != 2 {
		t.Fatalf("dayAssignments = %d, want 2", dayAssignments)
	}
	if !foundBobNight {
		t.Fatal("expected generated night assignment for Bob before conflict detection")
	}
}

func scheduleBatchCreateInputFromParseResult(result *ruleparse.ParseBatchResult) rule.BatchCreateInput {
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
