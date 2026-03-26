package auth

import (
	"context"
	"testing"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 创建内存 SQLite 测试数据库，自动迁移所有 M02+M03 表。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	// 迁移 tenant 表
	if err := db.AutoMigrate(&tenant.OrgNode{}); err != nil {
		t.Fatalf("迁移 org_nodes 失败: %v", err)
	}
	// 迁移 auth 表
	if err := db.AutoMigrate(&User{}, &Role{}, &UserNodeRole{}); err != nil {
		t.Fatalf("迁移 auth 表失败: %v", err)
	}
	return db
}

// setupTestService 创建带有完整依赖的测试 Service。
func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)

	authRepo := NewRepository(db)
	tenantRepo := tenant.NewRepository(db)
	jwtMgr := NewJWTManager(JWTConfig{
		Secret:          "test-secret-for-integration",
		AccessTokenTTL:  2 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "gantt-saas-test",
	})
	svc := NewService(authRepo, tenantRepo, jwtMgr, nil) // Redis 传 nil，限流相关跳过

	return svc, db
}

// seedTestOrgNode 创建测试用组织节点。
func seedTestOrgNode(t *testing.T, db *gorm.DB, id, name, path string, depth int) {
	t.Helper()
	node := &tenant.OrgNode{
		ID:       id,
		NodeType: tenant.NodeTypeDepartment,
		Name:     name,
		Code:     id,
		Path:     path,
		Depth:    depth,
		Status:   tenant.StatusActive,
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("创建测试节点 %s 失败: %v", id, err)
	}
}

// ── 验收标准 1: 注册 ──

func TestService_Register_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	// 预置角色和节点
	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("初始化系统角色失败: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	// 注册
	result, err := svc.Register(ctx, RegisterInput{
		Username:  "zhangsan",
		Email:     "zs@example.com",
		Password:  "Abc12345",
		OrgNodeID: "dept-001",
		RoleName:  string(RoleScheduler),
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// 验证返回
	if result.User.Username != "zhangsan" {
		t.Errorf("Username = %q, want %q", result.User.Username, "zhangsan")
	}
	if result.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if result.CurrentNode == nil || result.CurrentNode.NodeID != "dept-001" {
		t.Error("CurrentNode should be dept-001")
	}

	// 验证密码以 bcrypt 存储（非明文）
	var user User
	if err := db.Where("username = ?", "zhangsan").First(&user).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if user.PasswordHash == "Abc12345" {
		t.Error("密码不应以明文存储")
	}
	if len(user.PasswordHash) < 50 {
		t.Error("密码哈希长度不足，可能未使用 bcrypt")
	}
}

func TestService_Register_WeakPassword(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Username: "weak",
		Email:    "weak@example.com",
		Password: "123",
	})
	if err != ErrWeakPassword {
		t.Errorf("Register(weak password) error = %v, want %v", err, ErrWeakPassword)
	}
}

func TestService_Register_DuplicateUsername(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("初始化系统角色失败: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, err := svc.Register(ctx, RegisterInput{
		Username: "dup_user", Email: "a@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})
	if err != nil {
		t.Fatalf("第一次注册失败: %v", err)
	}

	_, err = svc.Register(ctx, RegisterInput{
		Username: "dup_user", Email: "b@example.com", Password: "Abc12345",
	})
	if err != ErrUsernameExists {
		t.Errorf("重复用户名 error = %v, want %v", err, ErrUsernameExists)
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("初始化系统角色失败: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, err := svc.Register(ctx, RegisterInput{
		Username: "user_a", Email: "dup@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})
	if err != nil {
		t.Fatalf("第一次注册失败: %v", err)
	}

	_, err = svc.Register(ctx, RegisterInput{
		Username: "user_b", Email: "dup@example.com", Password: "Abc12345",
	})
	if err != ErrEmailExists {
		t.Errorf("重复邮箱 error = %v, want %v", err, ErrEmailExists)
	}
}

func TestService_Register_NoNode(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 不指定节点时只创建用户，不签发 Token
	result, err := svc.Register(ctx, RegisterInput{
		Username: "nonode_user",
		Email:    "nonode@example.com",
		Password: "Abc12345",
	})
	if err != nil {
		t.Fatalf("Register(no node) error = %v", err)
	}
	if result.AccessToken != "" {
		t.Error("无节点注册不应返回 AccessToken")
	}
	if result.User.Username != "nonode_user" {
		t.Errorf("Username = %q, want %q", result.User.Username, "nonode_user")
	}
}

// ── 验收标准 2: 登录 ──

func TestService_Login_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	// 先注册
	_, err := svc.Register(ctx, RegisterInput{
		Username: "logintest", Email: "login@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}

	// 登录
	result, err := svc.Login(ctx, LoginInput{
		Username:  "logintest",
		Password:  "Abc12345",
		OrgNodeID: "dept-001",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// 验收标准：返回 JWT，Claims 包含 uid/nid/npath/role
	if result.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if result.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}
	if result.CurrentNode == nil {
		t.Fatal("CurrentNode should not be nil")
	}
	if result.CurrentNode.NodeID != "dept-001" {
		t.Errorf("CurrentNode.NodeID = %q, want %q", result.CurrentNode.NodeID, "dept-001")
	}
	if result.CurrentNode.RoleName != string(RoleScheduler) {
		t.Errorf("CurrentNode.RoleName = %q, want %q", result.CurrentNode.RoleName, string(RoleScheduler))
	}

	// 验证 JWT Claims 正确
	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, err := jwtMgr.ParseToken(result.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken(access) error = %v", err)
	}
	if claims.UserID == "" {
		t.Error("Claims.UserID should not be empty")
	}
	if claims.OrgNodeID != "dept-001" {
		t.Errorf("Claims.OrgNodeID = %q, want %q", claims.OrgNodeID, "dept-001")
	}
	if claims.OrgNodePath != "/org1/dept-001" {
		t.Errorf("Claims.OrgNodePath = %q, want %q", claims.OrgNodePath, "/org1/dept-001")
	}
	if claims.RoleName != string(RoleScheduler) {
		t.Errorf("Claims.RoleName = %q, want %q", claims.RoleName, string(RoleScheduler))
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "wrongpw", Email: "wrongpw@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	_, err := svc.Login(ctx, LoginInput{
		Username: "wrongpw", Password: "WrongPass1", OrgNodeID: "dept-001",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("Login(wrong password) error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginInput{
		Username: "nonexistent", Password: "Abc12345",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("Login(not found) error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestService_Login_DisabledUser(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "disabled", Email: "disabled@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 禁用用户
	db.Model(&User{}).Where("username = ?", "disabled").Update("status", UserStatusDisabled)

	_, err := svc.Login(ctx, LoginInput{
		Username: "disabled", Password: "Abc12345", OrgNodeID: "dept-001",
	})
	if err != ErrUserDisabled {
		t.Errorf("Login(disabled) error = %v, want %v", err, ErrUserDisabled)
	}
}

func TestService_Login_AutoSelectNode(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "autonode", Email: "autonode@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 不指定 OrgNodeID，应自动选取第一个可用节点
	result, err := svc.Login(ctx, LoginInput{
		Username: "autonode", Password: "Abc12345",
	})
	if err != nil {
		t.Fatalf("Login(auto node) error = %v", err)
	}
	if result.CurrentNode == nil || result.CurrentNode.NodeID != "dept-001" {
		t.Error("应自动选择第一个可用节点 dept-001")
	}
}

// ── 验收标准 3: Token 验证 ──

func TestService_Login_TokenValid(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "tokentest", Email: "token@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	result, _ := svc.Login(ctx, LoginInput{
		Username: "tokentest", Password: "Abc12345", OrgNodeID: "dept-001",
	})

	// 有效 Token 可正常解析
	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, err := jwtMgr.ParseToken(result.AccessToken)
	if err != nil {
		t.Fatalf("有效 Token 解析失败: %v", err)
	}
	if claims.UserID == "" {
		t.Error("Claims.UserID 不应为空")
	}

	// 无效 Token 应返回错误
	_, err = jwtMgr.ParseToken("invalid.token.string")
	if err != ErrInvalidToken {
		t.Errorf("无效 Token error = %v, want %v", err, ErrInvalidToken)
	}

	// 过期 Token 应返回过期错误
	expiredMgr := NewJWTManager(JWTConfig{
		Secret:         "test-secret-for-integration",
		AccessTokenTTL: -1 * time.Second,
	})
	expiredToken, _ := expiredMgr.GenerateAccessToken("user-1", "node-1", "/path", string(RolePlatformAdmin))
	_, err = jwtMgr.ParseToken(expiredToken)
	if err != ErrExpiredToken {
		t.Errorf("过期 Token error = %v, want %v", err, ErrExpiredToken)
	}
}

// ── 验收标准 4: 刷新 Token ──

func TestService_RefreshToken_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "refreshtest", Email: "refresh@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	loginResult, _ := svc.Login(ctx, LoginInput{
		Username: "refreshtest", Password: "Abc12345", OrgNodeID: "dept-001",
	})

	// 用 Refresh Token 换新的 Access Token
	refreshResult, err := svc.RefreshToken(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if refreshResult.AccessToken == "" {
		t.Error("刷新后 AccessToken 不应为空")
	}
	// 验证新 Token 可正确解析且包含正确的信息
	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	refreshClaims, err := jwtMgr.ParseToken(refreshResult.AccessToken)
	if err != nil {
		t.Fatalf("解析刷新后 Token 失败: %v", err)
	}
	if refreshClaims.OrgNodeID != "dept-001" {
		t.Errorf("刷新后 OrgNodeID = %q, want %q", refreshClaims.OrgNodeID, "dept-001")
	}
	if refreshResult.CurrentNode == nil || refreshResult.CurrentNode.NodeID != "dept-001" {
		t.Error("刷新后 CurrentNode 应保持 dept-001")
	}
}

func TestService_RefreshToken_InvalidToken(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, "invalid-refresh-token")
	if err != ErrInvalidToken {
		t.Errorf("RefreshToken(invalid) error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestService_RefreshToken_DisabledUser(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, _ = svc.Register(ctx, RegisterInput{
		Username: "refresh_disabled", Email: "rd@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	loginResult, _ := svc.Login(ctx, LoginInput{
		Username: "refresh_disabled", Password: "Abc12345", OrgNodeID: "dept-001",
	})

	// 禁用用户后刷新应失败
	db.Model(&User{}).Where("username = ?", "refresh_disabled").Update("status", UserStatusDisabled)

	_, err := svc.RefreshToken(ctx, loginResult.RefreshToken)
	if err != ErrUserDisabled {
		t.Errorf("RefreshToken(disabled user) error = %v, want %v", err, ErrUserDisabled)
	}
}

// ── 验收标准 5: 节点切换 ──

func TestService_SwitchNode_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)
	seedTestOrgNode(t, db, "dept-002", "外科", "/org1/dept-002", 1)

	// 注册并关联到 dept-001
	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "switchtest", Email: "switch@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 分配 dept-002 角色
	_, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-002", RoleName: string(RoleDeptAdmin),
	})
	if err != nil {
		t.Fatalf("AssignRole error = %v", err)
	}

	// 构造当前 Claims
	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, _ := jwtMgr.ParseToken(regResult.AccessToken)

	// 切换到 dept-002
	switchResult, err := svc.SwitchNode(ctx, claims, SwitchNodeInput{OrgNodeID: "dept-002"})
	if err != nil {
		t.Fatalf("SwitchNode() error = %v", err)
	}

	if switchResult.CurrentNode == nil || switchResult.CurrentNode.NodeID != "dept-002" {
		t.Error("切换后 CurrentNode 应为 dept-002")
	}
	if switchResult.CurrentNode.RoleName != string(RoleDeptAdmin) {
		t.Errorf("切换后角色应为 %s, got %q", RoleDeptAdmin, switchResult.CurrentNode.RoleName)
	}
	if switchResult.AccessToken == "" {
		t.Error("切换后应返回新 AccessToken")
	}

	// 验证新 Token 的 Claims
	newClaims, err := jwtMgr.ParseToken(switchResult.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken(new token) error = %v", err)
	}
	if newClaims.OrgNodeID != "dept-002" {
		t.Errorf("新 Token OrgNodeID = %q, want %q", newClaims.OrgNodeID, "dept-002")
	}
	if newClaims.RoleName != string(RoleDeptAdmin) {
		t.Errorf("新 Token RoleName = %q, want %q", newClaims.RoleName, string(RoleDeptAdmin))
	}
}

func TestService_SwitchNode_NoPermission(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)
	seedTestOrgNode(t, db, "dept-003", "骨科", "/org1/dept-003", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "noperm", Email: "noperm@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, _ := jwtMgr.ParseToken(regResult.AccessToken)

	// 切换到未授权节点，应返回 403
	_, err := svc.SwitchNode(ctx, claims, SwitchNodeInput{OrgNodeID: "dept-003"})
	if err != ErrNoNodePermission {
		t.Errorf("SwitchNode(无权限) error = %v, want %v", err, ErrNoNodePermission)
	}
}

// ── 验收标准 6: 权限检查 ──

func TestService_RBAC_EmployeeCannotCreateSchedule(t *testing.T) {
	// 验收标准：employee 角色无法调用 POST /schedules（schedule:create）
	if HasPermission(string(RoleEmployee), "schedule:create") {
		t.Error("employee 不应有 schedule:create 权限")
	}
	if !HasPermission(string(RoleScheduler), "schedule:create") {
		t.Error("scheduler 应有 schedule:create 权限")
	}
	if !HasPermission(string(RoleDeptAdmin), "schedule:create") {
		t.Error("dept_admin 应有 schedule:create 权限")
	}
	if !HasPermission(string(RolePlatformAdmin), "schedule:create") {
		t.Error("platform_admin 应有 schedule:create 权限")
	}
}

func TestService_RBAC_EmployeeReadSelf(t *testing.T) {
	if !HasPermission(string(RoleEmployee), "schedule:read:self") {
		t.Error("employee 应有 schedule:read:self 权限")
	}
	if !HasPermission(string(RoleEmployee), "leave:create:self") {
		t.Error("employee 应有 leave:create:self 权限")
	}
	if HasPermission(string(RoleEmployee), "employee:read") {
		t.Error("employee 不应有 employee:read 权限")
	}
}

// ── 密码重置 ──

func TestService_ResetPassword_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "resetpw", Email: "resetpw@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 重置密码
	err := svc.ResetPassword(ctx, regResult.User.ID, ResetPasswordInput{
		OldPassword: "Abc12345",
		NewPassword: "NewPass123",
	})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	// 用旧密码登录应失败
	_, err = svc.Login(ctx, LoginInput{
		Username: "resetpw", Password: "Abc12345", OrgNodeID: "dept-001",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("Login(old pw) error = %v, want %v", err, ErrInvalidCredentials)
	}

	// 用新密码登录应成功
	result, err := svc.Login(ctx, LoginInput{
		Username: "resetpw", Password: "NewPass123", OrgNodeID: "dept-001",
	})
	if err != nil {
		t.Fatalf("Login(new pw) error = %v", err)
	}
	if result.AccessToken == "" {
		t.Error("新密码登录后应返回 Token")
	}
}

func TestService_ResetPassword_WrongOldPassword(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "wrongold", Email: "wrongold@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	err := svc.ResetPassword(ctx, regResult.User.ID, ResetPasswordInput{
		OldPassword: "WrongOld1",
		NewPassword: "NewPass123",
	})
	if err != ErrOldPasswordMismatch {
		t.Errorf("ResetPassword(wrong old) error = %v, want %v", err, ErrOldPasswordMismatch)
	}
}

func TestService_ResetPassword_WeakNewPassword(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "weaknew", Email: "weaknew@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	err := svc.ResetPassword(ctx, regResult.User.ID, ResetPasswordInput{
		OldPassword: "Abc12345",
		NewPassword: "123",
	})
	if err != ErrWeakPassword {
		t.Errorf("ResetPassword(weak new) error = %v, want %v", err, ErrWeakPassword)
	}
}

// ── GetMe ──

func TestService_GetMe_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)
	seedTestOrgNode(t, db, "dept-002", "外科", "/org1/dept-002", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "metest", Email: "metest@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 增加第二个节点
	_, _ = svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-002", RoleName: string(RoleEmployee),
	})

	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, _ := jwtMgr.ParseToken(regResult.AccessToken)

	meResult, err := svc.GetMe(ctx, claims)
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}

	if meResult.User.Username != "metest" {
		t.Errorf("Username = %q, want %q", meResult.User.Username, "metest")
	}
	if meResult.CurrentNode == nil || meResult.CurrentNode.NodeID != "dept-001" {
		t.Error("CurrentNode 应为 dept-001")
	}
	if len(meResult.AvailableNodes) != 2 {
		t.Errorf("AvailableNodes 数量 = %d, want 2", len(meResult.AvailableNodes))
	}
}

// ── 角色分配 ──

func TestService_AssignRole_Success(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "assigntest", Email: "assign@example.com", Password: "Abc12345",
	})

	unr, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-001", RoleName: string(RoleDeptAdmin),
	})
	if err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}
	if unr.OrgNodeID != "dept-001" {
		t.Errorf("OrgNodeID = %q, want %q", unr.OrgNodeID, "dept-001")
	}
	if unr.Role == nil || unr.Role.Name != string(RoleDeptAdmin) {
		t.Error("分配的角色应为 dept_admin")
	}
}

func TestService_AssignRole_DuplicateRole(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "duprol", Email: "duprol@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	_, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})
	if err != ErrNodeRoleExists {
		t.Errorf("重复分配角色 error = %v, want %v", err, ErrNodeRoleExists)
	}
}

func TestService_AssignRole_InvalidRole(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "badrol", Email: "badrol@example.com", Password: "Abc12345",
	})

	_, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-001", RoleName: "nonexistent_role",
	})
	if err != ErrRoleNotFound {
		t.Errorf("分配不存在的角色 error = %v, want %v", err, ErrRoleNotFound)
	}
}

func TestService_AssignRole_InvalidNode(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "badnode", Email: "badnode@example.com", Password: "Abc12345",
	})

	_, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "nonexistent-node", RoleName: string(RoleScheduler),
	})
	if err != ErrNoNodePermission {
		t.Errorf("分配到不存在的节点 error = %v, want %v", err, ErrNoNodePermission)
	}
}

func TestService_AssignRole_InvalidUser(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	_, err := svc.AssignRole(ctx, AssignRoleInput{
		UserID: "nonexistent-user", OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})
	if err != ErrUserNotFound {
		t.Errorf("分配给不存在的用户 error = %v, want %v", err, ErrUserNotFound)
	}
}

// ── SeedSystemRoles ──

func TestService_SeedSystemRoles(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	// 第一次初始化
	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles() 首次调用 error = %v", err)
	}

	var count int64
	db.Model(&Role{}).Count(&count)
	if count != 5 {
		t.Errorf("预置角色数量 = %d, want 5", count)
	}

	// 幂等性：再次调用不应出错
	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles() 重复调用 error = %v", err)
	}

	db.Model(&Role{}).Count(&count)
	if count != 5 {
		t.Errorf("重复初始化后角色数量 = %d, want 5", count)
	}

	// 验证所有角色属性
	var roles []Role
	db.Find(&roles)
	for _, role := range roles {
		if !role.IsSystem {
			t.Errorf("角色 %s 应为系统角色", role.Name)
		}
		if role.DisplayName == "" {
			t.Errorf("角色 %s 缺少显示名", role.Name)
		}
	}
}

// ── RemoveRole ──

func TestService_RemoveRole(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "dept-001", "心内科", "/org1/dept-001", 1)

	regResult, _ := svc.Register(ctx, RegisterInput{
		Username: "removerol", Email: "removerol@example.com", Password: "Abc12345",
		OrgNodeID: "dept-001", RoleName: string(RoleScheduler),
	})

	// 查询 UNR ID
	var unr UserNodeRole
	db.Where("user_id = ?", regResult.User.ID).First(&unr)

	if err := svc.RemoveRole(ctx, unr.ID); err != nil {
		t.Fatalf("RemoveRole() error = %v", err)
	}

	// 移除后无法登录到该节点
	_, err := svc.Login(ctx, LoginInput{
		Username: "removerol", Password: "Abc12345", OrgNodeID: "dept-001",
	})
	if err != ErrNoNodePermission {
		t.Errorf("移除角色后登录 error = %v, want %v", err, ErrNoNodePermission)
	}
}

// ── 完整端到端流程 ──

func TestService_E2E_RegisterLoginSwitchRefresh(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}
	seedTestOrgNode(t, db, "org-001", "测试医院", "/org-001", 0)
	seedTestOrgNode(t, db, "dept-A", "心内科", "/org-001/dept-A", 1)
	seedTestOrgNode(t, db, "dept-B", "骨科", "/org-001/dept-B", 1)

	// Step 1: 注册用户并关联到 dept-A
	regResult, err := svc.Register(ctx, RegisterInput{
		Username: "e2e_user", Email: "e2e@example.com", Password: "E2ePass123",
		OrgNodeID: "dept-A", RoleName: string(RoleScheduler),
	})
	if err != nil {
		t.Fatalf("Step1 Register error = %v", err)
	}
	if regResult.CurrentNode.NodeID != "dept-A" {
		t.Fatal("注册后当前节点应为 dept-A")
	}

	// Step 2: 用密码登录
	loginResult, err := svc.Login(ctx, LoginInput{
		Username: "e2e_user", Password: "E2ePass123", OrgNodeID: "dept-A",
	})
	if err != nil {
		t.Fatalf("Step2 Login error = %v", err)
	}
	if loginResult.AccessToken == "" {
		t.Fatal("登录后应返回 AccessToken")
	}

	// Step 3: 分配第二个节点角色
	_, err = svc.AssignRole(ctx, AssignRoleInput{
		UserID: regResult.User.ID, OrgNodeID: "dept-B", RoleName: string(RoleDeptAdmin),
	})
	if err != nil {
		t.Fatalf("Step3 AssignRole error = %v", err)
	}

	// Step 4: 切换到 dept-B
	jwtMgr := NewJWTManager(JWTConfig{Secret: "test-secret-for-integration"})
	claims, _ := jwtMgr.ParseToken(loginResult.AccessToken)
	switchResult, err := svc.SwitchNode(ctx, claims, SwitchNodeInput{OrgNodeID: "dept-B"})
	if err != nil {
		t.Fatalf("Step4 SwitchNode error = %v", err)
	}
	if switchResult.CurrentNode.NodeID != "dept-B" {
		t.Fatal("切换后当前节点应为 dept-B")
	}
	if switchResult.CurrentNode.RoleName != string(RoleDeptAdmin) {
		t.Fatalf("切换后角色应为 %s, got %q", RoleDeptAdmin, switchResult.CurrentNode.RoleName)
	}

	// Step 5: 刷新 Token
	refreshResult, err := svc.RefreshToken(ctx, switchResult.RefreshToken)
	if err != nil {
		t.Fatalf("Step5 RefreshToken error = %v", err)
	}
	if refreshResult.AccessToken == "" {
		t.Fatal("刷新后应返回新 AccessToken")
	}

	// Step 6: 获取当前用户信息
	newClaims, _ := jwtMgr.ParseToken(refreshResult.AccessToken)
	meResult, err := svc.GetMe(ctx, newClaims)
	if err != nil {
		t.Fatalf("Step6 GetMe error = %v", err)
	}
	if meResult.User.Username != "e2e_user" {
		t.Error("GetMe 用户名不匹配")
	}
	if len(meResult.AvailableNodes) != 2 {
		t.Errorf("可用节点数量 = %d, want 2", len(meResult.AvailableNodes))
	}

	// Step 7: 重置密码
	err = svc.ResetPassword(ctx, newClaims.UserID, ResetPasswordInput{
		OldPassword: "E2ePass123",
		NewPassword: "NewE2e456",
	})
	if err != nil {
		t.Fatalf("Step7 ResetPassword error = %v", err)
	}

	// Step 8: 用新密码登录成功
	_, err = svc.Login(ctx, LoginInput{
		Username: "e2e_user", Password: "NewE2e456", OrgNodeID: "dept-A",
	})
	if err != nil {
		t.Fatalf("Step8 Login(new password) error = %v", err)
	}

	// Step 9: 旧密码登录失败
	_, err = svc.Login(ctx, LoginInput{
		Username: "e2e_user", Password: "E2ePass123", OrgNodeID: "dept-A",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("Step9 Login(old password) error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestService_Login_WithoutOrgNodeID_PrefersPlatformAdmin(t *testing.T) {
	svc, db := setupTestService(t)
	ctx := context.Background()

	if err := svc.SeedSystemRoles(ctx); err != nil {
		t.Fatalf("SeedSystemRoles error: %v", err)
	}

	seedTestOrgNode(t, db, "org-001", "测试医院", "/org-001", 0)
	seedTestOrgNode(t, db, "platform-root", "平台管理", "/platform-root", 0)

	regResult, err := svc.Register(ctx, RegisterInput{
		Username:  "platform_first",
		Email:     "platform_first@example.com",
		Password:  "Platform123",
		OrgNodeID: "org-001",
		RoleName:  string(RoleEmployee),
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}

	_, err = svc.AssignRole(ctx, AssignRoleInput{
		UserID:    regResult.User.ID,
		OrgNodeID: "platform-root",
		RoleName:  string(RolePlatformAdmin),
	})
	if err != nil {
		t.Fatalf("AssignRole error = %v", err)
	}

	loginResult, err := svc.Login(ctx, LoginInput{
		Username: "platform_first",
		Password: "Platform123",
	})
	if err != nil {
		t.Fatalf("Login error = %v", err)
	}

	if loginResult.CurrentNode == nil {
		t.Fatal("CurrentNode should not be nil")
	}

	if loginResult.CurrentNode.RoleName != string(RolePlatformAdmin) {
		t.Fatalf("CurrentNode.RoleName = %q, want %q", loginResult.CurrentNode.RoleName, string(RolePlatformAdmin))
	}

	if loginResult.CurrentNode.NodeID != "platform-root" {
		t.Fatalf("CurrentNode.NodeID = %q, want %q", loginResult.CurrentNode.NodeID, "platform-root")
	}
}
