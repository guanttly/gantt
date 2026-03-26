package leave

import (
"context"

"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

// Repository 请假数据访问层。
type Repository struct {
db *gorm.DB
}

// NewRepository 创建请假仓储。
func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// Create 创建请假记录。
func (r *Repository) Create(ctx context.Context, l *Leave) error {
if l.ID == "" {
l.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(l).Error
}

// GetByID 根据 ID 查询请假记录。
func (r *Repository) GetByID(ctx context.Context, id string) (*Leave, error) {
var l Leave
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
First(&l).Error
if err != nil {
return nil, err
}
return &l, nil
}

// Update 更新请假记录。
func (r *Repository) Update(ctx context.Context, l *Leave) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(l).Error
}

// Delete 删除请假记录。
func (r *Repository) Delete(ctx context.Context, id string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
Delete(&Leave{}).Error
}

// List 分页查询请假列表。
func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Leave, int64, error) {
var leaves []Leave
var total int64

tx := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Model(&Leave{})

if opts.EmployeeID != "" {
tx = tx.Where("employee_id = ?", opts.EmployeeID)
}
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
err := tx.Order("start_date DESC").
Offset(offset).
Limit(opts.Size).
Find(&leaves).Error
return leaves, total, err
}

// GetByEmployeeAndDateRange 查询某员工在日期范围内的已批准请假。
func (r *Repository) GetByEmployeeAndDateRange(ctx context.Context, employeeID, startDate, endDate string) ([]Leave, error) {
var leaves []Leave
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("employee_id = ? AND status = ? AND start_date <= ? AND end_date >= ?",
employeeID, StatusApproved, endDate, startDate).
Find(&leaves).Error
return leaves, err
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
return r.db.AutoMigrate(&Leave{})
}

// ListOptions 列表查询选项。
type ListOptions struct {
Page       int    `json:"page"`
Size       int    `json:"size"`
EmployeeID string `json:"employee_id"`
Status     string `json:"status"`
StartDate  string `json:"start_date"`
EndDate    string `json:"end_date"`
}
