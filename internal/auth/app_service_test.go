package auth

import (
	"context"
	"testing"
	"time"

	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAppAuthService(t *testing.T) (*AppService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tenant.OrgNode{}, &employee.Employee{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	repo := NewRepository(db)
	jwtMgr := NewJWTManager(JWTConfig{
		Secret:          "test-app-secret",
		AccessTokenTTL:  2 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "gantt-saas-test:app",
	})
	return NewAppService(repo, jwtMgr), db
}

func seedAppEmployee(t *testing.T, db *gorm.DB, nodeID, loginID, password string, mustReset bool) string {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("密码哈希失败: %v", err)
	}
	if err := db.Create(&tenant.OrgNode{
		ID:       nodeID,
		NodeType: tenant.NodeTypeDepartment,
		Name:     "门诊部",
		Code:     nodeID,
		Path:     "/org/" + nodeID,
		Depth:    1,
		Status:   tenant.StatusActive,
	}).Error; err != nil {
		t.Fatalf("创建节点失败: %v", err)
	}
	empID := "emp-001"
	if err := db.Create(&employee.Employee{
		ID:              empID,
		Name:            "张三",
		EmployeeNo:      stringPtr(loginID),
		Status:          employee.StatusActive,
		SchedulingRole:  employee.SchedulingRoleEmployee,
		AppPasswordHash: stringPtr(string(hashed)),
		AppMustResetPwd: mustReset,
		TenantModel: tenant.TenantModel{
			OrgNodeID: nodeID,
		},
	}).Error; err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}
	return empID
}

func TestAppService_Login_Success(t *testing.T) {
	svc, db := setupAppAuthService(t)
	empID := seedAppEmployee(t, db, "dept-001", "E1001", "Abc12345", true)

	result, err := svc.Login(context.Background(), AppLoginInput{
		LoginID:   "E1001",
		Password:  "Abc12345",
		OrgNodeID: "dept-001",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Employee.ID != empID {
		t.Fatalf("employee.id = %s, want %s", result.Employee.ID, empID)
	}
	if !result.MustResetPwd {
		t.Fatal("must_reset_pwd 应为 true")
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("登录后应返回 access_token 和 refresh_token")
	}
}

func TestAppService_ForceResetPassword_Success(t *testing.T) {
	svc, db := setupAppAuthService(t)
	empID := seedAppEmployee(t, db, "dept-001", "E1001", "Abc12345", true)

	if err := svc.ForceResetPassword(context.Background(), empID, ForceResetPasswordInput{NewPassword: "NewPwd123"}); err != nil {
		t.Fatalf("ForceResetPassword() error = %v", err)
	}

	result, err := svc.Login(context.Background(), AppLoginInput{
		LoginID:   "E1001",
		Password:  "NewPwd123",
		OrgNodeID: "dept-001",
	})
	if err != nil {
		t.Fatalf("重新登录失败: %v", err)
	}
	if result.MustResetPwd {
		t.Fatal("强制改密后 must_reset_pwd 应为 false")
	}
}

func TestAppService_GetMe(t *testing.T) {
	svc, db := setupAppAuthService(t)
	empID := seedAppEmployee(t, db, "dept-001", "E1001", "Abc12345", false)

	me, err := svc.GetMe(context.Background(), &Claims{UserID: empID, OrgNodeID: "dept-001"})
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if me.Employee.ID != empID {
		t.Fatalf("employee.id = %s, want %s", me.Employee.ID, empID)
	}
	if me.CurrentNode.NodeID != "dept-001" {
		t.Fatalf("current_node.node_id = %s, want dept-001", me.CurrentNode.NodeID)
	}
}

func stringPtr(value string) *string {
	return &value
}
