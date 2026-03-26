package schedule

import (
	"testing"

	"gantt-saas/internal/tenant"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestService_GetSelfAssignments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	statements := []string{
		`CREATE TABLE schedules (id TEXT PRIMARY KEY, name TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL, status TEXT NOT NULL, pipeline_type TEXT, config TEXT, created_by TEXT NOT NULL, created_at DATETIME, updated_at DATETIME, org_node_id TEXT NOT NULL)`,
		`CREATE TABLE shifts (id TEXT PRIMARY KEY, name TEXT NOT NULL, code TEXT NOT NULL, start_time TEXT NOT NULL, end_time TEXT NOT NULL, duration INTEGER NOT NULL, is_cross_day BOOLEAN NOT NULL DEFAULT FALSE, color TEXT, priority INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, created_at DATETIME, updated_at DATETIME, org_node_id TEXT NOT NULL)`,
		`CREATE TABLE schedule_assignments (id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, employee_id TEXT NOT NULL, shift_id TEXT NOT NULL, date TEXT NOT NULL, source TEXT NOT NULL, created_at DATETIME, org_node_id TEXT NOT NULL)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}

	ctx := tenant.WithOrgNode(t.Context(), "dept-001", "/org-root/dept-001")
	if err := db.Exec(`INSERT INTO schedules (id, name, start_date, end_date, status, created_by, org_node_id) VALUES ('sch-pub', '已发布排班', '2026-03-24', '2026-03-30', 'published', 'tester', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建已发布排班失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedules (id, name, start_date, end_date, status, created_by, org_node_id) VALUES ('sch-draft', '草稿排班', '2026-03-24', '2026-03-30', 'draft', 'tester', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建草稿排班失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO shifts (id, name, code, start_time, end_time, duration, color, status, org_node_id) VALUES ('shift-day', '白班', 'DAY', '08:00', '16:00', 480, '#409EFF', 'active', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建班次失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-001', 'sch-pub', 'emp-001', 'shift-day', '2026-03-26', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建已发布排班分配失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-002', 'sch-draft', 'emp-001', 'shift-day', '2026-03-27', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建草稿排班分配失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO schedule_assignments (id, schedule_id, employee_id, shift_id, date, source, org_node_id) VALUES ('asg-003', 'sch-pub', 'emp-002', 'shift-day', '2026-03-26', 'system', 'dept-001')`).Error; err != nil {
		t.Fatalf("创建其他员工排班分配失败: %v", err)
	}

	svc := NewService(NewRepository(db), nil, nil, nil, nil, zap.NewNop())
	items, err := svc.GetSelfAssignments(ctx, "emp-001", "2026-03-24", "2026-03-30")
	if err != nil {
		t.Fatalf("GetSelfAssignments() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ScheduleID != "sch-pub" {
		t.Fatalf("schedule_id = %s, want sch-pub", items[0].ScheduleID)
	}
	if items[0].ShiftName != "白班" {
		t.Fatalf("shift_name = %s, want 白班", items[0].ShiftName)
	}
}
