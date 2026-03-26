package group

import (
	"context"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockAppRoleSyncer struct {
	synced  []string
	revoked []string
	cleaned []string
}

func (m *mockAppRoleSyncer) SyncRolesForGroupMember(_ context.Context, groupID, employeeID, _ string) error {
	m.synced = append(m.synced, groupID+":"+employeeID)
	return nil
}

func (m *mockAppRoleSyncer) RevokeRolesForGroupMember(_ context.Context, groupID, employeeID string) error {
	m.revoked = append(m.revoked, groupID+":"+employeeID)
	return nil
}

func (m *mockAppRoleSyncer) CleanupGroup(_ context.Context, groupID string) error {
	m.cleaned = append(m.cleaned, groupID)
	return nil
}

func setupGroupService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	statements := []string{
		`CREATE TABLE employee_groups (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL, description TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE group_members (id TEXT PRIMARY KEY, group_id TEXT NOT NULL, employee_id TEXT NOT NULL, org_node_id TEXT NOT NULL, created_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}
	node := tenant.OrgNode{ID: "dept-001", NodeType: tenant.NodeTypeDepartment, Name: "心内科", Code: "dept-001", Path: "/dept-001", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	return NewService(NewRepository(db)), db, node
}

func TestService_AppRoleSyncHooks(t *testing.T) {
	svc, db, node := setupGroupService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	syncer := &mockAppRoleSyncer{}
	svc.SetAppRoleSyncer(syncer)

	grp := EmployeeGroup{ID: "grp-001", Name: "A组", TenantModel: tenant.TenantModel{OrgNodeID: node.ID}}
	if err := db.Create(&grp).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}

	if _, err := svc.AddMember(ctx, grp.ID, "emp-001", "user-001"); err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if len(syncer.synced) != 1 {
		t.Fatalf("len(synced) = %d, want 1", len(syncer.synced))
	}

	if err := svc.RemoveMember(ctx, grp.ID, "emp-001"); err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}
	if len(syncer.revoked) != 1 {
		t.Fatalf("len(revoked) = %d, want 1", len(syncer.revoked))
	}

	if err := svc.Delete(ctx, grp.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(syncer.cleaned) != 1 {
		t.Fatalf("len(cleaned) = %d, want 1", len(syncer.cleaned))
	}
}
