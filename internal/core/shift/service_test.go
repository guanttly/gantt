package shift

import (
	"context"
	"errors"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

func setupShiftService(t *testing.T) (*Service, *gorm.DB, tenant.OrgNode) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	statements := []string{
		`CREATE TABLE org_nodes (
			id TEXT PRIMARY KEY,
			parent_id TEXT,
			node_type TEXT NOT NULL,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			contact_name TEXT,
			contact_phone TEXT,
			path TEXT NOT NULL,
			depth INTEGER NOT NULL DEFAULT 0,
			is_login_point BOOLEAN NOT NULL DEFAULT FALSE,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE shifts (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			name TEXT NOT NULL,
			code TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'regular',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			duration INTEGER NOT NULL,
			is_cross_day BOOLEAN NOT NULL DEFAULT FALSE,
			color TEXT,
			priority INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			description TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE shift_groups (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			shift_id TEXT NOT NULL,
			group_id TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			notes TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE employee_groups (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE fixed_assignments (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			shift_id TEXT NOT NULL,
			employee_id TEXT NOT NULL,
			pattern_type TEXT NOT NULL,
			weekdays TEXT,
			week_pattern TEXT,
			monthdays TEXT,
			specific_dates TEXT,
			start_date TEXT,
			end_date TEXT,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE shift_weekly_staff (
			id TEXT PRIMARY KEY,
			org_node_id TEXT NOT NULL,
			shift_id TEXT NOT NULL,
			weekday INTEGER NOT NULL,
			staff_count INTEGER NOT NULL DEFAULT 0,
			is_custom BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}

	org := tenant.OrgNode{
		ID:           "org-001",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "测试机构",
		Code:         "org-001",
		Path:         "/org-001",
		Depth:        0,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	orgParentID := org.ID
	dept := tenant.OrgNode{
		ID:           "dept-001",
		ParentID:     &orgParentID,
		NodeType:     tenant.NodeTypeDepartment,
		Name:         "急诊科",
		Code:         "dept-001",
		Path:         "/org-001/dept-001",
		Depth:        1,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	if err := db.Create(&[]tenant.OrgNode{org, dept}).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}

	svc := NewService(NewRepository(db))
	svc.SetOrgNodeResolver(&mockOrgNodeResolver{nodes: map[string]tenant.OrgNode{
		org.ID:  org,
		dept.ID: dept,
	}})

	return svc, db, dept
}

func TestService_ToggleStatus(t *testing.T) {
	svc, db, node := setupShiftService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	shift := Shift{
		ID:         "shift-001",
		Name:       "夜班",
		Code:       "night",
		StartTime:  "20:00",
		EndTime:    "08:00",
		Duration:   12,
		IsCrossDay: true,
		Color:      "#000000",
		Priority:   1,
		Status:     StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: node.ID,
		},
	}
	if err := db.Create(&shift).Error; err != nil {
		t.Fatalf("创建测试班次失败: %v", err)
	}

	updated, err := svc.ToggleStatus(ctx, shift.ID)
	if err != nil {
		t.Fatalf("ToggleStatus() error = %v", err)
	}
	if updated.Status != StatusDisabled {
		t.Fatalf("status = %q, want %q", updated.Status, StatusDisabled)
	}

	updated, err = svc.ToggleStatus(ctx, shift.ID)
	if err != nil {
		t.Fatalf("second ToggleStatus() error = %v", err)
	}
	if updated.Status != StatusActive {
		t.Fatalf("status = %q, want %q", updated.Status, StatusActive)
	}
}

func TestService_ListAvailable(t *testing.T) {
	svc, db, node := setupShiftService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)
	shifts := []Shift{
		{
			ID:        "shift-active",
			Name:      "白班",
			Code:      "day",
			StartTime: "08:00",
			EndTime:   "16:00",
			Duration:  8,
			Color:     "#ffffff",
			Priority:  1,
			Status:    StatusActive,
			TenantModel: tenant.TenantModel{
				OrgNodeID: node.ID,
			},
		},
		{
			ID:         "shift-disabled",
			Name:       "停用夜班",
			Code:       "night-off",
			StartTime:  "20:00",
			EndTime:    "08:00",
			Duration:   12,
			IsCrossDay: true,
			Color:      "#000000",
			Priority:   2,
			Status:     StatusDisabled,
			TenantModel: tenant.TenantModel{
				OrgNodeID: node.ID,
			},
		},
	}
	if err := db.Create(&shifts).Error; err != nil {
		t.Fatalf("创建测试班次失败: %v", err)
	}

	available, err := svc.ListAvailable(ctx)
	if err != nil {
		t.Fatalf("ListAvailable() error = %v", err)
	}
	if len(available) != 1 {
		t.Fatalf("len(available) = %d, want 1", len(available))
	}
	if available[0].ID != "shift-active" {
		t.Fatalf("available[0].id = %q, want %q", available[0].ID, "shift-active")
	}
}

func TestService_GetShiftGroups_DoesNotRequireGroupCodeColumn(t *testing.T) {
	svc, db, node := setupShiftService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)

	shift := Shift{
		ID:        "shift-001",
		Name:      "白班",
		Code:      "day",
		Type:      ShiftTypeRegular,
		StartTime: "08:00",
		EndTime:   "16:00",
		Duration:  8,
		Color:     "#ffffff",
		Priority:  1,
		Status:    StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: node.ID,
		},
	}
	if err := db.Create(&shift).Error; err != nil {
		t.Fatalf("创建测试班次失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-001', ?, 'CT/MRI轮转1')`, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO shift_groups (id, org_node_id, shift_id, group_id, priority, is_active) VALUES ('sg-001', ?, 'shift-001', 'grp-001', 0, TRUE)`, node.ID).Error; err != nil {
		t.Fatalf("创建测试班次分组关联失败: %v", err)
	}

	items, err := svc.GetShiftGroups(ctx, shift.ID)
	if err != nil {
		t.Fatalf("GetShiftGroups() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].GroupName == nil || *items[0].GroupName != "CT/MRI轮转1" {
		t.Fatalf("group_name = %v, want CT/MRI轮转1", items[0].GroupName)
	}
}

func TestService_List_IncludesGroupNamesInSummary(t *testing.T) {
	svc, db, node := setupShiftService(t)
	ctx := tenant.WithOrgNode(context.Background(), node.ID, node.Path)

	shift := Shift{
		ID:        "shift-001",
		Name:      "下夜班",
		Code:      "night",
		Type:      ShiftTypeRegular,
		StartTime: "00:00",
		EndTime:   "08:00",
		Duration:  8,
		Color:     "#ffffff",
		Priority:  1,
		Status:    StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: node.ID,
		},
	}
	if err := db.Create(&shift).Error; err != nil {
		t.Fatalf("创建测试班次失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-001', ?, '江北夜班')`, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO employee_groups (id, org_node_id, name) VALUES ('grp-002', ?, '本部夜班')`, node.ID).Error; err != nil {
		t.Fatalf("创建测试分组失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO shift_groups (id, org_node_id, shift_id, group_id, priority, is_active) VALUES ('sg-001', ?, 'shift-001', 'grp-001', 0, TRUE)`, node.ID).Error; err != nil {
		t.Fatalf("创建测试班次分组关联失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO shift_groups (id, org_node_id, shift_id, group_id, priority, is_active) VALUES ('sg-002', ?, 'shift-001', 'grp-002', 1, TRUE)`, node.ID).Error; err != nil {
		t.Fatalf("创建测试班次分组关联失败: %v", err)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if len(items[0].GroupNames) != 2 {
		t.Fatalf("len(group_names) = %d, want 2", len(items[0].GroupNames))
	}
	if items[0].GroupNames[0] != "江北夜班" || items[0].GroupNames[1] != "本部夜班" {
		t.Fatalf("group_names = %v, want [江北夜班 本部夜班]", items[0].GroupNames)
	}
	if items[0].GroupSummary != "江北夜班、本部夜班" {
		t.Fatalf("group_summary = %q, want %q", items[0].GroupSummary, "江北夜班、本部夜班")
	}
}

func TestService_RejectNonDepartmentNode(t *testing.T) {
	svc, _, _ := setupShiftService(t)
	ctx := tenant.WithOrgNode(context.Background(), "org-001", "/org-001")

	if _, err := svc.Create(ctx, CreateInput{Name: "白班", Code: "day", StartTime: "08:00", EndTime: "16:00", Duration: 8}); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("Create() error = %v, want %v", err, ErrNotDeptNode)
	}

	if _, err := svc.List(ctx); !errors.Is(err, ErrNotDeptNode) {
		t.Fatalf("List() error = %v, want %v", err, ErrNotDeptNode)
	}
}
