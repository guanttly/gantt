package shift

import (
	"context"
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			duration INTEGER NOT NULL,
			is_cross_day BOOLEAN NOT NULL DEFAULT FALSE,
			color TEXT,
			priority INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME,
			updated_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}

	node := tenant.OrgNode{
		ID:           "org-001",
		NodeType:     tenant.NodeTypeOrganization,
		Name:         "测试机构",
		Code:         "org-001",
		Path:         "/org-001",
		Depth:        0,
		IsLoginPoint: true,
		Status:       tenant.StatusActive,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("创建测试组织节点失败: %v", err)
	}

	return NewService(NewRepository(db)), db, node
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
