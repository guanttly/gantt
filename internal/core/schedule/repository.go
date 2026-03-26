package schedule

import (
"context"
"encoding/json"

"gantt-saas/internal/core/schedule/step"
"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

// Repository 排班数据访问层。
type Repository struct {
db *gorm.DB
}

// NewRepository 创建排班仓储。
func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// ── Schedule CRUD ──────────────────────────────

// CreateSchedule 创建排班计划。
func (r *Repository) CreateSchedule(ctx context.Context, s *Schedule) error {
if s.ID == "" {
s.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(s).Error
}

// GetScheduleByID 根据 ID 查询排班计划。
func (r *Repository) GetScheduleByID(ctx context.Context, id string) (*Schedule, error) {
var s Schedule
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
First(&s).Error
if err != nil {
return nil, err
}
return &s, nil
}

// UpdateSchedule 更新排班计划。
func (r *Repository) UpdateSchedule(ctx context.Context, s *Schedule) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(s).Error
}

// UpdateScheduleStatus 更新排班计划状态（实现 step.DraftSaver 接口）。
func (r *Repository) UpdateScheduleStatus(ctx context.Context, id, status string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Model(&Schedule{}).
Where("id = ?", id).
Update("status", status).Error
}

// ListSchedules 查询排班计划列表。
func (r *Repository) ListSchedules(ctx context.Context, opts ScheduleListOptions) ([]Schedule, int64, error) {
var schedules []Schedule
var total int64

tx := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Model(&Schedule{})

if opts.Status != "" {
tx = tx.Where("status = ?", opts.Status)
}
if opts.StartDate != "" {
tx = tx.Where("end_date >= ?", opts.StartDate)
}
if opts.EndDate != "" {
tx = tx.Where("start_date <= ?", opts.EndDate)
}

if err := tx.Count(&total).Error; err != nil {
return nil, 0, err
}

offset := (opts.Page - 1) * opts.Size
if offset < 0 {
offset = 0
}
err := tx.Order("created_at DESC").
Offset(offset).
Limit(opts.Size).
Find(&schedules).Error
return schedules, total, err
}

// DeleteSchedule 删除排班计划（级联删除）。
func (r *Repository) DeleteSchedule(ctx context.Context, id string) error {
return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
scopedTx := tenant.ApplyScope(ctx, tx)
if err := scopedTx.Where("schedule_id = ?", id).Delete(&Change{}).Error; err != nil {
return err
}
if err := scopedTx.Where("schedule_id = ?", id).Delete(&Assignment{}).Error; err != nil {
return err
}
return scopedTx.Where("id = ?", id).Delete(&Schedule{}).Error
})
}

// ── Assignment CRUD ──────────────────────────────

// ListAssignmentsBySchedule 查询某排班计划的所有分配。
func (r *Repository) ListAssignmentsBySchedule(ctx context.Context, scheduleID string) ([]Assignment, error) {
var assignments []Assignment
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("schedule_id = ?", scheduleID).
Order("date ASC, shift_id ASC, employee_id ASC").
Find(&assignments).Error
return assignments, err
}

// DeleteAssignment 删除单条排班分配（实现 step.AssignmentRepo 接口）。
func (r *Repository) DeleteAssignment(ctx context.Context, id string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
Delete(&Assignment{}).Error
}

// DeleteAssignmentsByScheduleID 删除某排班计划的所有分配（实现 step.DraftSaver 接口）。
func (r *Repository) DeleteAssignmentsByScheduleID(ctx context.Context, scheduleID string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("schedule_id = ?", scheduleID).
Delete(&Assignment{}).Error
}

// BatchSaveAssignments 批量保存排班分配（实现 step.DraftSaver 接口）。
func (r *Repository) BatchSaveAssignments(ctx context.Context, assignments []step.Assignment) error {
if len(assignments) == 0 {
return nil
}
dbAssignments := make([]Assignment, 0, len(assignments))
for _, a := range assignments {
if a.ID == "" {
a.ID = uuid.New().String()
}
dbAssignments = append(dbAssignments, Assignment{
ID:         a.ID,
ScheduleID: a.ScheduleID,
EmployeeID: a.EmployeeID,
ShiftID:    a.ShiftID,
Date:       a.Date,
Source:     a.Source,
TenantModel: tenant.TenantModel{
OrgNodeID: a.OrgNodeID,
},
})
}
return r.db.WithContext(ctx).CreateInBatches(dbAssignments, 100).Error
}

// CreateChange 创建变更记录（实现 step.AssignmentRepo 接口）。
func (r *Repository) CreateChange(ctx context.Context, c *step.ChangeRecord) error {
change := Change{
ID:           c.ID,
ScheduleID:   c.ScheduleID,
AssignmentID: c.AssignmentID,
ChangeType:   c.ChangeType,
BeforeData:   json.RawMessage(c.BeforeData),
AfterData:    json.RawMessage(c.AfterData),
Reason:       c.Reason,
ChangedBy:    c.ChangedBy,
TenantModel: tenant.TenantModel{
OrgNodeID: c.OrgNodeID,
},
}
if change.ID == "" {
change.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(&change).Error
}

// ListChangesBySchedule 查询变更记录。
func (r *Repository) ListChangesBySchedule(ctx context.Context, scheduleID string, opts ChangeListOptions) ([]Change, int64, error) {
var changes []Change
var total int64

tx := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Model(&Change{}).
Where("schedule_id = ?", scheduleID)

if err := tx.Count(&total).Error; err != nil {
return nil, 0, err
}

offset := (opts.Page - 1) * opts.Size
if offset < 0 {
offset = 0
}
err := tx.Order("created_at DESC").
Offset(offset).
Limit(opts.Size).
Find(&changes).Error
return changes, total, err
}

// ── 查询选项 ──────────────────────────────

// ScheduleListOptions 排班计划列表查询选项。
type ScheduleListOptions struct {
Page      int    `json:"page"`
Size      int    `json:"size"`
Status    string `json:"status"`
StartDate string `json:"start_date"`
EndDate   string `json:"end_date"`
}

// ChangeListOptions 变更记录列表查询选项。
type ChangeListOptions struct {
Page int `json:"page"`
Size int `json:"size"`
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
return r.db.AutoMigrate(&Schedule{}, &Assignment{}, &Change{})
}

// ── 确保 Repository 实现 step 接口 ──────────────────────────────

var _ step.DraftSaver = (*Repository)(nil)
var _ step.AssignmentRepo = (*Repository)(nil)
