package rule

import (
	"context"
	"errors"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
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
			category TEXT NOT NULL,
			sub_type TEXT NOT NULL,
			config TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			disabled BOOLEAN NOT NULL DEFAULT FALSE,
			disabled_by TEXT,
			disabled_at DATETIME,
			disabled_reason TEXT,
			override_rule_id TEXT,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME
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
	baseRule := Rule{ID: "rule-org-003", Name: "机构连班限制", Category: CategoryConstraint, SubType: SubTypeLimit, Config: []byte(`{"type":"max_count","shift_id":"day","max":5,"period":"week"}`), Priority: 8, IsEnabled: true, TenantModel: tenant.TenantModel{OrgNodeID: org.ID}}
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
