package tenant

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Context 工具函数测试 ──

func TestWithOrgNode(t *testing.T) {
	ctx := context.Background()
	ctx = WithOrgNode(ctx, "node-1", "/org1/node-1")

	if got := GetOrgNodeID(ctx); got != "node-1" {
		t.Errorf("GetOrgNodeID() = %q, want %q", got, "node-1")
	}
	if got := GetOrgNodePath(ctx); got != "/org1/node-1" {
		t.Errorf("GetOrgNodePath() = %q, want %q", got, "/org1/node-1")
	}
}

func TestWithScopeTree(t *testing.T) {
	ctx := context.Background()
	if IsScopeTree(ctx) {
		t.Error("IsScopeTree() should be false by default")
	}

	ctx = WithScopeTree(ctx, true)
	if !IsScopeTree(ctx) {
		t.Error("IsScopeTree() should be true after WithScopeTree(true)")
	}
}

func TestGetOrgNodeID_Empty(t *testing.T) {
	ctx := context.Background()
	if got := GetOrgNodeID(ctx); got != "" {
		t.Errorf("GetOrgNodeID() on empty context = %q, want empty", got)
	}
}

// ── SkipTenantGuard 测试 ──

func TestSkipTenantGuard(t *testing.T) {
	ctx := context.Background()
	if isGuardSkipped(ctx) {
		t.Error("isGuardSkipped should be false by default")
	}

	ctx = SkipTenantGuard(ctx)
	if !isGuardSkipped(ctx) {
		t.Error("isGuardSkipped should be true after SkipTenantGuard")
	}
}

// ── OrgNode 模型测试 ──

func TestOrgNode_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active", StatusActive, true},
		{"suspended", StatusSuspended, false},
		{"unknown", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &OrgNode{Status: tt.status}
			if got := node.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgNode_TableName(t *testing.T) {
	node := OrgNode{}
	if got := node.TableName(); got != "org_nodes" {
		t.Errorf("TableName() = %q, want %q", got, "org_nodes")
	}
}

// ── 输入参数校验测试 ──

func TestValidNodeTypes(t *testing.T) {
	validTypes := []string{NodeTypeOrganization, NodeTypeCampus, NodeTypeDepartment, NodeTypeCustom}
	for _, nt := range validTypes {
		if !validNodeTypes[nt] {
			t.Errorf("node type %q should be valid", nt)
		}
	}

	if validNodeTypes["invalid"] {
		t.Error("'invalid' should not be a valid node type")
	}
}

// ── buildTree 测试 ──

func TestBuildTree(t *testing.T) {
	root := &OrgNode{
		ID:       "root",
		NodeType: NodeTypeOrganization,
		Name:     "总部",
		Path:     "/root",
		Depth:    0,
	}

	campusID := "campus-1"
	deptID := "dept-1"
	nodes := []OrgNode{
		*root,
		{
			ID:       campusID,
			ParentID: &root.ID,
			NodeType: NodeTypeCampus,
			Name:     "院区A",
			Path:     "/root/campus-1",
			Depth:    1,
		},
		{
			ID:       deptID,
			ParentID: &campusID,
			NodeType: NodeTypeDepartment,
			Name:     "科室X",
			Path:     "/root/campus-1/dept-1",
			Depth:    2,
		},
	}

	tree := buildTree(root, nodes)

	// 根节点
	if tree.ID != "root" {
		t.Errorf("root ID = %q, want %q", tree.ID, "root")
	}

	// 一级子节点
	if len(tree.Children) != 1 {
		t.Fatalf("root children count = %d, want 1", len(tree.Children))
	}
	campus := tree.Children[0]
	if campus.ID != "campus-1" {
		t.Errorf("campus ID = %q, want %q", campus.ID, "campus-1")
	}

	// 二级子节点
	if len(campus.Children) != 1 {
		t.Fatalf("campus children count = %d, want 1", len(campus.Children))
	}
	dept := campus.Children[0]
	if dept.ID != "dept-1" {
		t.Errorf("dept ID = %q, want %q", dept.ID, "dept-1")
	}
	if len(dept.Children) != 0 {
		t.Errorf("dept children count = %d, want 0", len(dept.Children))
	}
}

func TestBuildTree_Empty(t *testing.T) {
	root := &OrgNode{
		ID:   "root",
		Name: "根节点",
		Path: "/root",
	}

	tree := buildTree(root, []OrgNode{*root})

	if tree.ID != "root" {
		t.Errorf("root ID = %q, want %q", tree.ID, "root")
	}
	if len(tree.Children) != 0 {
		t.Errorf("children count = %d, want 0", len(tree.Children))
	}
}

// ── Service Create 路径计算测试（单元级别） ──

func TestCreateNodeInput_Validation(t *testing.T) {
	tests := []struct {
		name     string
		nodeType string
		wantErr  error
	}{
		{"valid organization", NodeTypeOrganization, nil},
		{"valid campus", NodeTypeCampus, nil},
		{"valid department", NodeTypeDepartment, nil},
		{"valid custom", NodeTypeCustom, nil},
		{"invalid type", "invalid", ErrInvalidNodeType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !validNodeTypes[tt.nodeType] && tt.wantErr == nil {
				t.Errorf("node type %q should be valid", tt.nodeType)
			}
			if validNodeTypes[tt.nodeType] && tt.wantErr != nil {
				t.Errorf("node type %q should be invalid", tt.nodeType)
			}
		})
	}
}

// ── TenantModel 嵌入测试 ──

func TestTenantModel(t *testing.T) {
	type Employee struct {
		ID   string `gorm:"primaryKey;size:64"`
		Name string `gorm:"size:64;not null"`
		TenantModel
	}

	emp := Employee{
		ID:   "emp-1",
		Name: "张三",
		TenantModel: TenantModel{
			OrgNodeID: "dept-1",
		},
	}

	if emp.OrgNodeID != "dept-1" {
		t.Errorf("OrgNodeID = %q, want %q", emp.OrgNodeID, "dept-1")
	}
}

func TestIsProtectedNode(t *testing.T) {
	protected := &OrgNode{Code: protectedPlatformRootCode}
	normalParent := "root"
	normal := &OrgNode{Code: "dept-001", ParentID: &normalParent}

	if !isProtectedNode(protected) {
		t.Fatal("platform-root 顶级节点应被视为受保护节点")
	}

	if isProtectedNode(normal) {
		t.Fatal("普通节点不应被视为受保护节点")
	}
}

func TestService_Delete_ProtectedNode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&OrgNode{}); err != nil {
		t.Fatalf("迁移 org_nodes 失败: %v", err)
	}

	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	node := &OrgNode{
		ID:       "platform-root",
		NodeType: NodeTypeOrganization,
		Name:     "平台管理",
		Code:     protectedPlatformRootCode,
		Path:     "/platform-root",
		Depth:    0,
		Status:   StatusActive,
	}
	if err := repo.Create(ctx, node); err != nil {
		t.Fatalf("创建测试节点失败: %v", err)
	}

	err = svc.Delete(ctx, node.ID)
	if err != ErrProtectedNode {
		t.Fatalf("Delete() error = %v, want %v", err, ErrProtectedNode)
	}

	stored, err := repo.GetByID(ctx, node.ID)
	if err != nil {
		t.Fatalf("受保护节点不应被删除: %v", err)
	}
	if stored.ID != node.ID {
		t.Fatalf("保留节点 ID = %q, want %q", stored.ID, node.ID)
	}
}

func TestService_Create_RootNodeMustBeOrganization(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&OrgNode{}); err != nil {
		t.Fatalf("迁移 org_nodes 失败: %v", err)
	}

	svc := NewService(NewRepository(db))
	ctx := context.Background()

	_, err = svc.Create(ctx, CreateNodeInput{
		Name:     "根部门",
		Code:     "root-dept",
		NodeType: NodeTypeDepartment,
	})
	if err != ErrInvalidRootType {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidRootType)
	}

	_, err = svc.Create(ctx, CreateNodeInput{
		Name:     "根机构",
		Code:     "root-org",
		NodeType: NodeTypeOrganization,
	})
	if err != nil {
		t.Fatalf("创建根机构失败: %v", err)
	}

	var count int64
	if err := db.Model(&OrgNode{}).Count(&count).Error; err != nil {
		t.Fatalf("统计节点数量失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("节点数量 = %d, want 1", count)
	}
}

func TestService_GetRootTrees_IncludesChildren(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&OrgNode{}); err != nil {
		t.Fatalf("迁移 org_nodes 失败: %v", err)
	}

	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()

	root := &OrgNode{
		ID:       "root-1",
		NodeType: NodeTypeOrganization,
		Name:     "鼓楼医院",
		Code:     "gulou-root",
		Path:     "/root-1",
		Depth:    0,
		Status:   StatusActive,
	}
	childParentID := root.ID
	child := &OrgNode{
		ID:       "child-1",
		ParentID: &childParentID,
		NodeType: NodeTypeDepartment,
		Name:     "心内科",
		Code:     "dept-cardiology",
		Path:     "/root-1/child-1",
		Depth:    1,
		Status:   StatusActive,
	}

	if err := repo.Create(ctx, root); err != nil {
		t.Fatalf("创建根节点失败: %v", err)
	}
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("创建子节点失败: %v", err)
	}

	trees, err := svc.GetRootTrees(ctx)
	if err != nil {
		t.Fatalf("GetRootTrees() error = %v", err)
	}
	if len(trees) != 1 {
		t.Fatalf("root tree count = %d, want 1", len(trees))
	}
	if len(trees[0].Children) != 1 {
		t.Fatalf("root children count = %d, want 1", len(trees[0].Children))
	}
	if trees[0].Children[0].Code != "dept-cardiology" {
		t.Fatalf("child code = %q, want %q", trees[0].Children[0].Code, "dept-cardiology")
	}
}
