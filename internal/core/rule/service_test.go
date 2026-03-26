package rule

import (
	"context"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRuleService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode, tenant.OrgNode, tenant.OrgNode) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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

	root := tenant.OrgNode{
		ID:           "platform-root-id",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "平台管理",
		Code:         "platform-root",
		Path:         "/platform-root-id",
		Depth:        0,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	orgParentID := root.ID
	org := tenant.OrgNode{
		ID:           "org-001",
		ParentID:     &orgParentID,
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "测试机构",
		Code:         "org-001",
		Path:         "/platform-root-id/org-001",
		Depth:        1,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	deptParentID := org.ID
	dept := tenant.OrgNode{
		ID:           "dept-001",
		ParentID:     &deptParentID,
		NodeType:     tenant.NodeTypeDepartment,
		Name:         "急诊科",
		Code:         "dept-001",
		Path:         "/platform-root-id/org-001/dept-001",
		Depth:        2,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}

	if err := db.Create(&[]tenant.OrgNode{root, org, dept}).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}

	return NewService(NewRepository(db), tenant.NewRepository(db)), db, root, org, dept
}

func TestService_CreateOverrideFromAncestor(t *testing.T) {
	svc, db, _, org, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	baseRule := Rule{
		ID:        "rule-org-001",
		Name:      "机构夜班上限",
		Category:  CategoryConstraint,
		SubType:   SubTypeLimit,
		Config:    []byte(`{"type":"max_count","shift_id":"night","max":4,"period":"week"}`),
		Priority:  10,
		IsEnabled: true,
		TenantModel: tenant.TenantModel{
			OrgNodeID: org.ID,
		},
	}
	if err := db.Create(&baseRule).Error; err != nil {
		t.Fatalf("创建上级规则失败: %v", err)
	}

	created, err := svc.Create(ctx, CreateInput{
		Name:           "科室夜班上限",
		Category:       CategoryConstraint,
		SubType:        SubTypeLimit,
		Config:         []byte(`{"type":"max_count","shift_id":"night","max":2,"period":"week"}`),
		Priority:       20,
		OverrideRuleID: &baseRule.ID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OverrideRuleID == nil || *created.OverrideRuleID != baseRule.ID {
		t.Fatal("覆盖规则未正确关联上级规则")
	}
	if created.OrgNodeID != dept.ID {
		t.Fatalf("created.org_node_id = %q, want %q", created.OrgNodeID, dept.ID)
	}
}

func TestService_DisableAndRestoreInheritedRule(t *testing.T) {
	svc, db, _, org, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	baseRule := Rule{
		ID:        "rule-org-002",
		Name:      "夜班后休息",
		Category:  CategoryConstraint,
		SubType:   SubTypeMinRest,
		Config:    []byte(`{"type":"min_rest","days":1}`),
		Priority:  5,
		IsEnabled: true,
		TenantModel: tenant.TenantModel{
			OrgNodeID: org.ID,
		},
	}
	if err := db.Create(&baseRule).Error; err != nil {
		t.Fatalf("创建上级规则失败: %v", err)
	}

	disabledView, err := svc.DisableInherited(ctx, baseRule.ID, "急诊科本周临时取消", "platform-user-1")
	if err != nil {
		t.Fatalf("DisableInherited() error = %v", err)
	}
	if !disabledView.Disabled {
		t.Fatal("禁用后的规则视图应标记 disabled=true")
	}
	if disabledView.DisabledReason == nil || *disabledView.DisabledReason != "急诊科本周临时取消" {
		t.Fatal("禁用原因未正确回显")
	}

	effective, err := svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 0 {
		t.Fatalf("len(effective.rules) = %d, want 0", len(effective.Rules))
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if !items[0].Disabled || items[0].ID != baseRule.ID {
		t.Fatal("规则列表应展示被禁用的继承规则")
	}

	if err := svc.RestoreInheritance(ctx, baseRule.ID); err != nil {
		t.Fatalf("RestoreInheritance() error = %v", err)
	}

	effective, err = svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("恢复后 ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 1 || effective.Rules[0].ID != baseRule.ID {
		t.Fatal("恢复继承后应重新使用上级规则")
	}

	items, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("恢复后 List() error = %v", err)
	}
	if len(items) != 1 || items[0].Disabled {
		t.Fatal("恢复继承后规则列表不应再显示为禁用")
	}
}

func TestService_RestoreInheritanceDeletesLocalOverride(t *testing.T) {
	svc, db, _, org, dept := setupRuleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	baseRule := Rule{
		ID:        "rule-org-003",
		Name:      "机构连班限制",
		Category:  CategoryConstraint,
		SubType:   SubTypeLimit,
		Config:    []byte(`{"type":"max_count","shift_id":"day","max":5,"period":"week"}`),
		Priority:  8,
		IsEnabled: true,
		TenantModel: tenant.TenantModel{
			OrgNodeID: org.ID,
		},
	}
	if err := db.Create(&baseRule).Error; err != nil {
		t.Fatalf("创建上级规则失败: %v", err)
	}

	overrideRule, err := svc.Create(ctx, CreateInput{
		Name:           "科室连班限制",
		Category:       CategoryConstraint,
		SubType:        SubTypeLimit,
		Config:         []byte(`{"type":"max_count","shift_id":"day","max":3,"period":"week"}`),
		Priority:       9,
		OverrideRuleID: &baseRule.ID,
	})
	if err != nil {
		t.Fatalf("创建本级覆盖规则失败: %v", err)
	}

	effective, err := svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 1 || effective.Rules[0].ID != overrideRule.ID {
		t.Fatal("生效规则应优先使用本级覆盖")
	}

	if err := svc.RestoreInheritance(ctx, overrideRule.ID); err != nil {
		t.Fatalf("RestoreInheritance() error = %v", err)
	}

	effective, err = svc.ListEffective(ctx)
	if err != nil {
		t.Fatalf("恢复后 ListEffective() error = %v", err)
	}
	if len(effective.Rules) != 1 || effective.Rules[0].ID != baseRule.ID {
		t.Fatal("删除本级覆盖后应恢复为上级规则")
	}
}
