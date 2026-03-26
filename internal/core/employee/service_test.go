package employee

import (
	"context"
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
		`CREATE TABLE employees (id TEXT PRIMARY KEY, org_node_id TEXT NOT NULL, name TEXT NOT NULL, employee_no TEXT, phone TEXT, email TEXT, position TEXT, category TEXT, scheduling_role TEXT NOT NULL DEFAULT 'employee', app_password_hash TEXT, app_must_reset_pwd BOOLEAN NOT NULL DEFAULT TRUE, status TEXT NOT NULL DEFAULT 'active', hire_date TEXT, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}
	node := tenant.OrgNode{ID: "dept-001", NodeType: tenant.NodeTypeDepartment, Name: "心内科", Code: "dept-001", Path: "/dept-001", Depth: 0, IsLoginPoint: true, Status: tenant.StatusActive}
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
