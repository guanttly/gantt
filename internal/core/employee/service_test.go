package employee

import (
	"context"
	"errors"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockAppRoleCleaner struct {
	cleaned []string
}

func (m *mockAppRoleCleaner) CleanupEmployeeRoles(_ context.Context, employeeID string) error {
	m.cleaned = append(m.cleaned, employeeID)
	return nil
}

func setupEmployeeService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	statements := []string{
		`CREATE TABLE org_nodes (id TEXT PRIMARY KEY, parent_id TEXT, node_type TEXT NOT NULL, name TEXT NOT NULL, code TEXT NOT NULL, contact_name TEXT, contact_phone TEXT, path TEXT NOT NULL, depth INTEGER NOT NULL DEFAULT 0, is_login_point BOOLEAN NOT NULL DEFAULT FALSE, status TEXT NOT NULL DEFAULT 'active', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE employees (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL, employee_no TEXT, phone TEXT, email TEXT, position TEXT, category TEXT, scheduling_role TEXT NOT NULL DEFAULT 'employee', app_password_hash TEXT, app_must_reset_pwd BOOLEAN NOT NULL DEFAULT TRUE, status TEXT NOT NULL DEFAULT 'active', hire_date TEXT, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}
	node := tenant.OrgNode{ID: "dept-001", NodeType: tenant.NodeTypeDepartment, Name: "心内科", Code: "dept-001", Path: "/org-001/dept-001", Depth: 1, IsLoginPoint: true, Status: tenant.StatusActive}
	parent := tenant.OrgNode{ID: "org-001", NodeType: tenant.NodeTypeOrganization, Name: "鼓楼医院", Code: "org-001", Path: "/org-001", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	if err := db.Create(&[]tenant.OrgNode{parent, node}).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}
	return NewService(NewRepository(db)), db, node
}

func TestService_CleanupRolesOnInactivateAndDelete(t *testing.T) {
	svc, db, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	cleaner := &mockAppRoleCleaner{}
	svc.SetAppRoleCleaner(cleaner)

	emp := Employee{ID: "emp-001", Name: "张三", Status: StatusActive, TenantModel: tenant.TenantModel{OrgNodeID: node.ID}}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}

	status := StatusInactive
	updated, err := svc.Update(ctx, emp.ID, UpdateInput{Status: &status})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != StatusInactive {
		t.Fatalf("status = %q, want %q", updated.Status, StatusInactive)
	}
	if len(cleaner.cleaned) != 1 || cleaner.cleaned[0] != emp.ID {
		t.Fatal("停用员工时应清理应用角色")
	}

	if err := svc.Delete(ctx, emp.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(cleaner.cleaned) != 2 || cleaner.cleaned[1] != emp.ID {
		t.Fatal("删除员工时应再次清理应用角色")
	}
	var count int64
	if err := db.Model(&Employee{}).Where("id = ?", emp.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询员工失败: %v", err)
	}
	if count != 0 {
		t.Fatal("员工删除后不应仍然存在")
	}
}

func TestService_NoCleanupOnRegularUpdate(t *testing.T) {
	svc, db, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	cleaner := &mockAppRoleCleaner{}
	svc.SetAppRoleCleaner(cleaner)

	emp := Employee{ID: "emp-002", Name: "李四", Status: StatusActive, TenantModel: tenant.TenantModel{OrgNodeID: node.ID}}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}

	name := "李四-更新"
	if _, err := svc.Update(ctx, emp.ID, UpdateInput{Name: &name}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(cleaner.cleaned) != 0 {
		t.Fatal("普通更新不应触发应用角色清理")
	}
}

func TestService_CreateGeneratesAppCredentials(t *testing.T) {
	svc, _, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	empNo := "E1001"

	emp, err := svc.Create(ctx, CreateInput{Name: "王五", EmployeeNo: &empNo})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if emp.AppDefaultPassword == nil || *emp.AppDefaultPassword != "E1001@App1" {
		t.Fatalf("app_default_password = %v, want E1001@App1", emp.AppDefaultPassword)
	}
	if emp.AppPasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*emp.AppPasswordHash), []byte("E1001@App1")) != nil {
		t.Fatal("app_password_hash 未正确生成")
	}
	if !emp.AppMustResetPwd {
		t.Fatal("新建员工必须强制修改 app 密码")
	}
}

func TestService_Create_AllowsDescendantOrgNode(t *testing.T) {
	svc, _, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/org-001")
	targetNodeID := node.ID

	emp, err := svc.Create(ctx, CreateInput{Name: "王五", OrgNodeID: &targetNodeID})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if emp.OrgNodeID != node.ID {
		t.Fatalf("org_node_id = %q, want %q", emp.OrgNodeID, node.ID)
	}
}

func TestService_Update_AllowsMoveToDescendantAndCleansRoles(t *testing.T) {
	svc, db, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/org-001")
	cleaner := &mockAppRoleCleaner{}
	svc.SetAppRoleCleaner(cleaner)
	orgNodeID := "org-001"
	emp := Employee{ID: "emp-move", Name: "张三", Status: StatusActive, TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID}}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}

	targetNodeID := node.ID
	updated, err := svc.Update(ctx, emp.ID, UpdateInput{OrgNodeID: &targetNodeID})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.OrgNodeID != node.ID {
		t.Fatalf("org_node_id = %q, want %q", updated.OrgNodeID, node.ID)
	}
	if len(cleaner.cleaned) != 1 || cleaner.cleaned[0] != emp.ID {
		t.Fatal("移动员工到其他节点后应清理应用角色")
	}
}

func TestService_Create_RejectsOutOfScopeOrgNode(t *testing.T) {
	svc, db, node := setupEmployeeService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	outNode := tenant.OrgNode{ID: "other-dept", NodeType: tenant.NodeTypeDepartment, Name: "外科", Code: "other-dept", Path: "/other-org/other-dept", Depth: 1, IsLoginPoint: true, Status: tenant.StatusActive}
	otherOrg := tenant.OrgNode{ID: "other-org", NodeType: tenant.NodeTypeOrganization, Name: "其他机构", Code: "other-org", Path: "/other-org", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
	if err := db.Create(&[]tenant.OrgNode{otherOrg, outNode}).Error; err != nil {
		t.Fatalf("创建额外组织节点失败: %v", err)
	}
	targetNodeID := outNode.ID

	_, err := svc.Create(ctx, CreateInput{Name: "越权员工", OrgNodeID: &targetNodeID})
	if err == nil {
		t.Fatal("越权创建员工应返回错误")
	}
	if !errors.Is(err, ErrEmployeeNodeOutOfScope) {
		t.Fatalf("error = %v, want ErrEmployeeNodeOutOfScope", err)
	}
}
