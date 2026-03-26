package admin

import (
	"context"
	"errors"
	"testing"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPlatformUserService(t *testing.T) (*PlatformUserService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&tenant.OrgNode{}, &auth.User{}, &auth.Role{}, &auth.UserNodeRole{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	roles := []auth.Role{
		{
			ID:          auth.RoleIDPlatformAdmin,
			Name:        string(auth.RolePlatformAdmin),
			DisplayName: "平台管理员",
			Permissions: auth.JSONArray{"*"},
			IsSystem:    true,
		},
		{
			ID:          auth.RoleIDOrgAdmin,
			Name:        string(auth.RoleOrgAdmin),
			DisplayName: "机构管理员",
			Permissions: auth.JSONArray{"org:*"},
			IsSystem:    true,
		},
		{
			ID:          auth.RoleIDDeptAdmin,
			Name:        string(auth.RoleDeptAdmin),
			DisplayName: "科室管理员",
			Permissions: auth.JSONArray{"dept:*"},
			IsSystem:    true,
		},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
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
	org := tenant.OrgNode{
		ID:           "org-001",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "测试机构",
		Code:         "org-001",
		Path:         "/platform-root-id/org-001",
		Depth:        1,
		ParentID:     &root.ID,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	dept := tenant.OrgNode{
		ID:           "dept-001",
		NodeType:     tenant.NodeTypeDepartment,
		Name:         "急诊科",
		Code:         "dept-001",
		Path:         "/platform-root-id/org-001/dept-001",
		Depth:        2,
		ParentID:     &org.ID,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	if err := db.Create(&[]tenant.OrgNode{root, org, dept}).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}

	return NewPlatformUserService(db), db
}

func TestPlatformUserService_Create(t *testing.T) {
	svc, db := setupPlatformUserService(t)
	ctx := context.Background()
	phone := "13800138000"

	result, err := svc.Create(ctx, CreatePlatformUserInput{
		Username:  "org_admin_001",
		Email:     "org-admin-001@example.com",
		Phone:     &phone,
		OrgNodeID: "org-001",
		RoleName:  string(auth.RoleOrgAdmin),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if result.User.Username != "org_admin_001" {
		t.Fatalf("user.username = %q, want %q", result.User.Username, "org_admin_001")
	}
	if result.DefaultPassword == "" {
		t.Fatal("默认密码不应为空")
	}
	if len(result.User.Roles) != 1 || result.User.Roles[0].OrgNodeID != "org-001" {
		t.Fatal("平台账号角色信息未正确返回")
	}

	var user auth.User
	if err := db.Where("id = ?", result.User.ID).First(&user).Error; err != nil {
		t.Fatalf("查询平台账号失败: %v", err)
	}
	if !user.MustResetPwd {
		t.Fatal("新建平台账号必须强制改密")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(result.DefaultPassword)); err != nil {
		t.Fatal("数据库中的密码哈希与默认密码不匹配")
	}

	var count int64
	if err := db.Model(&auth.UserNodeRole{}).Where("user_id = ? AND org_node_id = ?", result.User.ID, "org-001").Count(&count).Error; err != nil {
		t.Fatalf("查询平台账号角色绑定失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("平台账号角色绑定数量 = %d, want 1", count)
	}
}

func TestPlatformUserService_ResetPassword(t *testing.T) {
	svc, db := setupPlatformUserService(t)
	ctx := context.Background()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("生成初始密码失败: %v", err)
	}

	user := auth.User{
		ID:           "user-001",
		Username:     "dept_admin_001",
		Email:        "dept-admin-001@example.com",
		PasswordHash: string(oldHash),
		Status:       auth.UserStatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	if err := db.Create(&auth.UserNodeRole{
		ID:        "unr-001",
		UserID:    user.ID,
		OrgNodeID: "dept-001",
		RoleID:    auth.RoleIDDeptAdmin,
	}).Error; err != nil {
		t.Fatalf("创建测试角色绑定失败: %v", err)
	}

	result, err := svc.ResetPassword(ctx, user.ID)
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if result.DefaultPassword == "" {
		t.Fatal("重置后的默认密码不应为空")
	}
	if !result.MustResetPwd {
		t.Fatal("重置密码后必须强制改密")
	}

	var updated auth.User
	if err := db.Where("id = ?", user.ID).First(&updated).Error; err != nil {
		t.Fatalf("查询更新后的平台账号失败: %v", err)
	}
	if !updated.MustResetPwd {
		t.Fatal("数据库中的 must_reset_pwd 应为 true")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte(result.DefaultPassword)); err != nil {
		t.Fatal("重置后的密码哈希与默认密码不匹配")
	}
}

func TestPlatformUserService_ListAndDisable(t *testing.T) {
	svc, db := setupPlatformUserService(t)
	ctx := context.Background()
	phone := "13900139000"

	user := auth.User{
		ID:           "user-002",
		Username:     "dept_admin_list",
		Email:        "dept-admin-list@example.com",
		Phone:        &phone,
		PasswordHash: "hashed-password",
		Status:       auth.UserStatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	if err := db.Create(&auth.UserNodeRole{
		ID:        "unr-002",
		UserID:    user.ID,
		OrgNodeID: "dept-001",
		RoleID:    auth.RoleIDDeptAdmin,
	}).Error; err != nil {
		t.Fatalf("创建测试角色绑定失败: %v", err)
	}

	items, err := svc.List(ctx, "dept-001")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if len(items[0].Roles) != 1 || items[0].Roles[0].RoleName != string(auth.RoleDeptAdmin) {
		t.Fatal("平台账号列表未返回正确角色信息")
	}

	if err := svc.Disable(ctx, user.ID, "another-user"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	var updated auth.User
	if err := db.Where("id = ?", user.ID).First(&updated).Error; err != nil {
		t.Fatalf("查询禁用后的平台账号失败: %v", err)
	}
	if updated.Status != auth.UserStatusDisabled {
		t.Fatalf("status = %q, want %q", updated.Status, auth.UserStatusDisabled)
	}

	if err := svc.Disable(ctx, user.ID, user.ID); err == nil {
		t.Fatal("禁用自己应返回错误")
	}
}

func TestPlatformUserService_List_RespectsManageScope(t *testing.T) {
	svc, db := setupPlatformUserService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/platform-root-id/org-001")

	users := []auth.User{
		{ID: "user-platform", Username: "platform_admin", Email: "platform@example.com", PasswordHash: "x", Status: auth.UserStatusActive},
		{ID: "user-org", Username: "org_admin", Email: "org@example.com", PasswordHash: "x", Status: auth.UserStatusActive},
		{ID: "user-dept", Username: "dept_admin", Email: "dept@example.com", PasswordHash: "x", Status: auth.UserStatusActive},
		{ID: "user-employee", Username: "employee", Email: "employee@example.com", PasswordHash: "x", Status: auth.UserStatusActive},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	bindings := []auth.UserNodeRole{
		{ID: "unr-platform", UserID: "user-platform", OrgNodeID: "platform-root-id", RoleID: auth.RoleIDPlatformAdmin},
		{ID: "unr-org", UserID: "user-org", OrgNodeID: "org-001", RoleID: auth.RoleIDOrgAdmin},
		{ID: "unr-dept", UserID: "user-dept", OrgNodeID: "dept-001", RoleID: auth.RoleIDDeptAdmin},
		{ID: "unr-employee", UserID: "user-employee", OrgNodeID: "dept-001", RoleID: auth.RoleIDEmployee},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatalf("创建测试角色绑定失败: %v", err)
	}

	items, err := svc.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.ID == "user-platform" {
			t.Fatal("org_admin 不应看到平台根上的 platform_admin 账号")
		}
		if item.ID == "user-employee" {
			t.Fatal("平台账号列表不应混入 employee 业务账号")
		}
		for _, role := range item.Roles {
			if role.OrgNodeID == "platform-root-id" {
				t.Fatal("返回结果不应包含超出当前组织树范围的角色")
			}
		}
	}
}

func TestPlatformUserService_Disable_DeniesOutOfScopeUser(t *testing.T) {
	svc, db := setupPlatformUserService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/platform-root-id/org-001")

	user := auth.User{
		ID:           "user-platform",
		Username:     "platform_admin",
		Email:        "platform@example.com",
		PasswordHash: "hashed-password",
		Status:       auth.UserStatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试平台账号失败: %v", err)
	}
	if err := db.Create(&auth.UserNodeRole{
		ID:        "unr-platform",
		UserID:    user.ID,
		OrgNodeID: "platform-root-id",
		RoleID:    auth.RoleIDPlatformAdmin,
	}).Error; err != nil {
		t.Fatalf("创建测试角色绑定失败: %v", err)
	}

	err := svc.Disable(ctx, user.ID, "org-actor")
	if err == nil {
		t.Fatal("禁用超出当前组织树范围的平台账号应返回错误")
	}
	if !errors.Is(err, ErrManageScopeDenied) {
		t.Fatalf("error = %v, want ErrManageScopeDenied", err)
	}

	var unchanged auth.User
	if err := db.Where("id = ?", user.ID).First(&unchanged).Error; err != nil {
		t.Fatalf("查询测试平台账号失败: %v", err)
	}
	if unchanged.Status != auth.UserStatusActive {
		t.Fatalf("status = %q, want %q", unchanged.Status, auth.UserStatusActive)
	}
}
