package admin

import (
	"context"
	"testing"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupOrganizationService(t *testing.T) (*OrganizationService, *gorm.DB) {
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

	if err := db.Create(&auth.Role{
		ID:          auth.RoleIDOrgAdmin,
		Name:        string(auth.RoleOrgAdmin),
		DisplayName: "机构管理员",
		Permissions: auth.JSONArray{"org:*"},
		IsSystem:    true,
	}).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}

	if err := db.Create(&tenant.OrgNode{
		ID:           "platform-root-id",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "平台管理",
		Code:         "platform-root",
		Path:         "/platform-root-id",
		Depth:        0,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}).Error; err != nil {
		t.Fatalf("创建平台根节点失败: %v", err)
	}

	return NewOrganizationService(db), db
}

func TestOrganizationService_Create(t *testing.T) {
	svc, db := setupOrganizationService(t)
	ctx := context.Background()
	contactName := "张三"
	contactPhone := "13800138000"
	adminPhone := "13900139000"

	result, err := svc.Create(ctx, CreateOrganizationInput{
		Name:         "XX 市第一人民医院",
		Code:         "hospital-001",
		ContactName:  &contactName,
		ContactPhone: &contactPhone,
		Admin: CreateOrganizationAdmin{
			Username: "hospital001_admin",
			Email:    "admin@hospital001.com",
			Phone:    &adminPhone,
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if result.Organization.Code != "hospital-001" {
		t.Fatalf("organization.code = %q, want %q", result.Organization.Code, "hospital-001")
	}
	if result.Admin.Username != "hospital001_admin" {
		t.Fatalf("admin.username = %q, want %q", result.Admin.Username, "hospital001_admin")
	}
	if !result.Admin.MustResetPwd {
		t.Fatal("新建机构管理员必须强制改密")
	}
	if result.Admin.DefaultPassword == "" {
		t.Fatal("默认密码不应为空")
	}

	var org tenant.OrgNode
	if err := db.Where("id = ?", result.Organization.ID).First(&org).Error; err != nil {
		t.Fatalf("查询机构节点失败: %v", err)
	}
	if org.ContactName == nil || *org.ContactName != contactName {
		t.Fatal("机构联系人未正确持久化")
	}
	if org.ContactPhone == nil || *org.ContactPhone != contactPhone {
		t.Fatal("机构联系电话未正确持久化")
	}

	var adminUser auth.User
	if err := db.Where("id = ?", result.Admin.ID).First(&adminUser).Error; err != nil {
		t.Fatalf("查询机构管理员失败: %v", err)
	}
	if !adminUser.MustResetPwd {
		t.Fatal("机构管理员 must_reset_pwd 应为 true")
	}
	if adminUser.PasswordHash == result.Admin.DefaultPassword {
		t.Fatal("机构管理员密码应以哈希形式存储")
	}

	var count int64
	if err := db.Model(&auth.UserNodeRole{}).Where("user_id = ? AND org_node_id = ?", adminUser.ID, org.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询管理员角色关联失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("管理员角色关联数量 = %d, want 1", count)
	}
}

func TestOrganizationService_List_ExcludesPlatformRoot(t *testing.T) {
	svc, db := setupOrganizationService(t)
	ctx := context.Background()

	if err := db.Create(&tenant.OrgNode{
		ID:           "org-001",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "测试机构",
		Code:         "org-001",
		Path:         "/org-001",
		Depth:        0,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}).Error; err != nil {
		t.Fatalf("创建测试机构失败: %v", err)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Code != "org-001" {
		t.Fatalf("items[0].code = %q, want %q", items[0].Code, "org-001")
	}
}
