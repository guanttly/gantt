package group

import (
	"context"
	"errors"
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
		`CREATE TABLE employees (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL, employee_no TEXT, position TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}
	node := tenant.OrgNode{ID: "dept-001", NodeType: tenant.NodeTypeDepartment, Name: "心内科", Code: "dept-001", Path: "/dept-001", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	svc := NewService(NewRepository(db))
	svc.SetOrgNodeResolver(&mockOrgNodeResolver{nodes: map[string]tenant.OrgNode{
		node.ID:   node,
		"org-001": {ID: "org-001", NodeType: tenant.NodeTypeOrganization, Name: "鼓楼医院", Code: "org-001", Path: "/org-001", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive},
	}})
	return svc, db, node
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

func TestService_RejectNonDepartmentNode(t *testing.T) {
	svc, _, _ := setupGroupService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/org-001")

	if _, err := svc.Create(ctx, CreateInput{Name: "院级分组"}); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("Create() error = %v, want %v", err, ErrNotDeptNode)
	}

	if _, err := svc.List(ctx, ""); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("List() error = %v, want %v", err, ErrNotDeptNode)
	}
}

func TestService_List_ReturnsGroupsWithMemberCount(t *testing.T) {
	svc, db, node := setupGroupService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)

	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-1', ?, '一组'), ('grp-2', ?, '二组')`, node.ID, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO group_members (id, group_id, employee_id, org_node_id) VALUES ('gm-1', 'grp-1', 'emp-1', ?), ('gm-2', 'grp-1', 'emp-2', ?)`, node.ID, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组成员失败: %v", err)
	}

	items, err := svc.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "grp-1" || items[0].MemberCount != 2 {
		t.Fatalf("first group = %#v, want grp-1 with member_count=2", items[0])
	}
	if items[1].ID != "grp-2" || items[1].MemberCount != 0 {
		t.Fatalf("second group = %#v, want grp-2 with member_count=0", items[1])
	}
}

func TestService_List_FiltersByKeyword(t *testing.T) {
	svc, db, node := setupGroupService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)

	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name, description) VALUES ('grp-1', ?, '固定审核组', '固定班次'), ('grp-2', ?, '审核/报告组', '报告相关'), ('grp-3', ?, '江北夜班', '夜班')`, node.ID, node.ID, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}

	items, err := svc.List(ctx, "审核")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "固定审核组" || items[1].Name != "审核/报告组" {
		t.Fatalf("items = %#v, want keyword filtered groups", items)
	}

	items, err = svc.List(ctx, "报告")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "审核/报告组" {
		t.Fatalf("items = %#v, want only 审核/报告组", items)
	}
}

func TestService_GetMembers_ReturnsJoinedEmployeeFields(t *testing.T) {
	svc, db, node := setupGroupService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)

	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-1', ?, '一组')`, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, employee_no, position, status) VALUES ('emp-1', ?, '张三', 'E001', '护士', 'active')`, node.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO group_members (id, group_id, employee_id, org_node_id, created_at) VALUES ('gm-1', 'grp-1', 'emp-1', ?, CURRENT_TIMESTAMP)`, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组成员失败: %v", err)
	}

	members, err := svc.GetMembers(ctx, "grp-1")
	if err != nil {
		t.Fatalf("GetMembers() error = %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].EmployeeName == nil || *members[0].EmployeeName != "张三" {
		t.Fatalf("employee_name = %v, want 张三", members[0].EmployeeName)
	}
	if members[0].EmployeeNo == nil || *members[0].EmployeeNo != "E001" {
		t.Fatalf("employee_no = %v, want E001", members[0].EmployeeNo)
	}
}
