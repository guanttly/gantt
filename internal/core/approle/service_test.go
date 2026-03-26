package approle

import (
	"context"
	"testing"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAppRoleService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode, tenant.OrgNode) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	statements := []string{
		`CREATE TABLE org_nodes (id TEXT PRIMARY KEY, parent_id TEXT, node_type TEXT NOT NULL, name TEXT NOT NULL, code TEXT NOT NULL, contact_name TEXT, contact_phone TEXT, path TEXT NOT NULL, depth INTEGER NOT NULL DEFAULT 0, is_login_point BOOLEAN NOT NULL DEFAULT FALSE, status TEXT NOT NULL DEFAULT 'active', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE platform_users (id TEXT PRIMARY KEY, bound_employee_id TEXT)`,
		`CREATE TABLE employees (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active')`,
		`CREATE TABLE employee_groups (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE group_members (id TEXT PRIMARY KEY, group_id TEXT NOT NULL, employee_id TEXT NOT NULL, org_node_id TEXT NOT NULL)`,
		`CREATE TABLE employee_app_roles (id TEXT PRIMARY KEY, employee_id TEXT NOT NULL, org_node_id TEXT NOT NULL, app_role TEXT NOT NULL, source TEXT NOT NULL, source_group_id TEXT, granted_by TEXT NOT NULL, granted_at DATETIME, expires_at DATETIME)`,
		`CREATE TABLE group_default_app_roles (id TEXT PRIMARY KEY, group_id TEXT NOT NULL, org_node_id TEXT NOT NULL, app_role TEXT NOT NULL, created_by TEXT NOT NULL, created_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}

	root := tenant.OrgNode{ID: "org-root", NodeType: tenant.NodeTypeOrganization, Name: "机构", Code: "root", Path: "/org-root", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	deptParentID := root.ID
	dept := tenant.OrgNode{ID: "dept-001", ParentID: &deptParentID, NodeType: tenant.NodeTypeDepartment, Name: "心内科", Code: "dept-001", Path: "/org-root/dept-001", Depth: 1, IsLoginPoint: true, Status: tenant.StatusActive}
	if err := db.Create(&[]tenant.OrgNode{root, dept}).Error; err != nil {
		t.Fatalf("创建测试节点失败: %v", err)
	}

	return NewService(NewRepository(db), tenant.NewRepository(db)), db, root, dept
}

func TestService_AssignEmployeeRole(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}

	item, err := svc.AssignEmployeeRole(ctx, "emp-001", AssignEmployeeRoleInput{AppRole: RoleScheduler, OrgNodeID: dept.ID}, "platform-user-1")
	if err != nil {
		t.Fatalf("AssignEmployeeRole() error = %v", err)
	}
	if item.AppRole != RoleScheduler {
		t.Fatalf("app_role = %q, want %q", item.AppRole, RoleScheduler)
	}
	if item.Source != SourceManual {
		t.Fatalf("source = %q, want %q", item.Source, SourceManual)
	}

	items, err := svc.ListEmployeeRoles(ctx, "emp-001")
	if err != nil {
		t.Fatalf("ListEmployeeRoles() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestService_AssignGroupDefaultRoleGrantsCurrentMembers(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-001', ?, 'A组')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO group_members (id, group_id, employee_id, org_node_id) VALUES ('gm-001', 'grp-001', 'emp-001', ?)`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试分组成员失败: %v", err)
	}

	item, err := svc.AssignGroupDefaultRole(ctx, "grp-001", AssignGroupDefaultRoleInput{AppRole: RoleLeaveApprover, OrgNodeID: dept.ID}, "platform-user-1")
	if err != nil {
		t.Fatalf("AssignGroupDefaultRole() error = %v", err)
	}
	if item.AppRole != RoleLeaveApprover {
		t.Fatalf("app_role = %q, want %q", item.AppRole, RoleLeaveApprover)
	}

	roles, err := svc.ListEmployeeRoles(ctx, "emp-001")
	if err != nil {
		t.Fatalf("ListEmployeeRoles() error = %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("len(roles) = %d, want 1", len(roles))
	}
	if roles[0].Source != SourceGroup {
		t.Fatalf("source = %q, want %q", roles[0].Source, SourceGroup)
	}
}

func TestService_ExpiringAndSummary(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := svc.AssignEmployeeRole(ctx, "emp-001", AssignEmployeeRoleInput{AppRole: RoleScheduler, OrgNodeID: dept.ID, ExpiresAt: &expiresAt}, "platform-user-1")
	if err != nil {
		t.Fatalf("AssignEmployeeRole() error = %v", err)
	}

	summary, err := svc.Summary(ctx)
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if len(summary) != 1 || summary[0].Count != 1 {
		t.Fatal("应用角色汇总结果不符合预期")
	}

	expiring, err := svc.Expiring(ctx, 2)
	if err != nil {
		t.Fatalf("Expiring() error = %v", err)
	}
	if len(expiring) != 1 || expiring[0].EmployeeName != "张三" {
		t.Fatal("即将过期角色列表不符合预期")
	}
}

func TestService_MyRolesAndPermissions(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO platform_users (id, bound_employee_id) VALUES ('user-001', 'emp-001')`).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	_, err := svc.AssignEmployeeRole(ctx, "emp-001", AssignEmployeeRoleInput{AppRole: RoleScheduler, OrgNodeID: dept.ID}, "platform-user-1")
	if err != nil {
		t.Fatalf("AssignEmployeeRole() error = %v", err)
	}

	roles, err := svc.MyRoles(ctx, "user-001")
	if err != nil {
		t.Fatalf("MyRoles() error = %v", err)
	}
	if len(roles.AppRoles) != 2 {
		t.Fatalf("len(app_roles) = %d, want 2", len(roles.AppRoles))
	}

	permissions, err := svc.MyPermissions(ctx, "user-001")
	if err != nil {
		t.Fatalf("MyPermissions() error = %v", err)
	}
	if len(permissions.Permissions) == 0 {
		t.Fatal("permissions 不应为空")
	}

	directRoles, err := svc.MyRoles(ctx, "emp-001")
	if err != nil {
		t.Fatalf("MyRoles(employee token) error = %v", err)
	}
	if directRoles.EmployeeID != "emp-001" {
		t.Fatalf("employee_token employee_id = %s, want emp-001", directRoles.EmployeeID)
	}
}

func TestService_CleanExpiredRoles(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(context.Background(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	now := time.Now()
	expiredAt := now.Add(-1 * time.Hour)
	activeUntil := now.Add(24 * time.Hour)
	if err := db.Create(&EmployeeAppRole{
		ID:         "role-expired",
		EmployeeID: "emp-001",
		OrgNodeID:  dept.ID,
		AppRole:    RoleScheduler,
		Source:     SourceManual,
		GrantedBy:  "platform-user-1",
		ExpiresAt:  &expiredAt,
	}).Error; err != nil {
		t.Fatalf("创建过期角色失败: %v", err)
	}
	if err := db.Create(&EmployeeAppRole{
		ID:         "role-active",
		EmployeeID: "emp-001",
		OrgNodeID:  dept.ID,
		AppRole:    RoleLeaveApprover,
		Source:     SourceManual,
		GrantedBy:  "platform-user-1",
		ExpiresAt:  &activeUntil,
	}).Error; err != nil {
		t.Fatalf("创建未过期角色失败: %v", err)
	}

	rowsAffected, err := svc.CleanExpiredRoles(context.Background())
	if err != nil {
		t.Fatalf("CleanExpiredRoles() error = %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("rowsAffected = %d, want 1", rowsAffected)
	}

	roles, err := svc.ListEmployeeRoles(ctx, "emp-001")
	if err != nil {
		t.Fatalf("ListEmployeeRoles() error = %v", err)
	}
	if len(roles) != 1 || roles[0].ID != "role-active" {
		t.Fatalf("roles after cleanup = %+v, want only role-active", roles)
	}
}
