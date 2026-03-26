package approle

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/tenant"
)

func TestRequireAnyPermission_AllowsAndInjectsContext(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	ctx := tenant.WithOrgNode(t.Context(), dept.ID, dept.Path)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO platform_users (id, bound_employee_id) VALUES ('user-001', 'emp-001')`).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	if _, err := svc.AssignEmployeeRole(ctx, "emp-001", AssignEmployeeRoleInput{AppRole: RoleScheduler, OrgNodeID: dept.ID}, "platform-user-1"); err != nil {
		t.Fatalf("AssignEmployeeRole() error = %v", err)
	}

	jwtMgr := auth.NewJWTManager(auth.JWTConfig{Secret: "test-secret", AccessTokenTTL: time.Hour, RefreshTokenTTL: time.Hour})
	token, err := jwtMgr.GenerateAccessToken("user-001", dept.ID, dept.Path, string(auth.RoleDeptAdmin))
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	var employeeID string
	var permissions []string
	handler := auth.AuthMiddleware(jwtMgr)(RequireAnyPermission(svc, "schedule:execute")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		employeeID = CurrentEmployeeID(r.Context())
		permissions = CurrentPermissions(r.Context())
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if employeeID != "emp-001" {
		t.Fatalf("employeeID = %q, want %q", employeeID, "emp-001")
	}
	if len(permissions) == 0 {
		t.Fatal("permissions should not be empty")
	}
	if !hasAnyPermission(permissions, []string{"schedule:execute"}) {
		t.Fatal("expected injected permissions to include schedule:execute")
	}
}

func TestRequireAnyPermission_DeniesWithoutPermission(t *testing.T) {
	svc, db, _, dept := setupAppRoleService(t)
	if err := db.Exec(`INSERT INTO employees (id, org_node_id, name, status) VALUES ('emp-001', ?, '张三', 'active')`, dept.ID).Error; err != nil {
		t.Fatalf("创建测试员工失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO platform_users (id, bound_employee_id) VALUES ('user-001', 'emp-001')`).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}

	jwtMgr := auth.NewJWTManager(auth.JWTConfig{Secret: "test-secret", AccessTokenTTL: time.Hour, RefreshTokenTTL: time.Hour})
	token, err := jwtMgr.GenerateAccessToken("user-001", dept.ID, dept.Path, string(auth.RoleEmployee))
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	handler := auth.AuthMiddleware(jwtMgr)(RequireAnyPermission(svc, "schedule:create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
