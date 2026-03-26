package shift

import (
"context"

"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

// Repository 班次数据访问层。
type Repository struct {
db *gorm.DB
}

// NewRepository 创建班次仓储。
func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// Create 创建班次。
func (r *Repository) Create(ctx context.Context, s *Shift) error {
if s.ID == "" {
s.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(s).Error
}

// GetByID 根据 ID 查询班次。
func (r *Repository) GetByID(ctx context.Context, id string) (*Shift, error) {
var s Shift
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
First(&s).Error
if err != nil {
return nil, err
}
return &s, nil
}

// Update 更新班次。
func (r *Repository) Update(ctx context.Context, s *Shift) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(s).Error
}

// Delete 删除班次（硬删除）。
func (r *Repository) Delete(ctx context.Context, id string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
Delete(&Shift{}).Error
}

// List 查询班次列表。
func (r *Repository) List(ctx context.Context) ([]Shift, error) {
var shifts []Shift
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Order("priority ASC, code ASC").
Find(&shifts).Error
return shifts, err
}

// GetByOrgNodeAndCode 根据组织节点和编码查询班次（唯一性检查）。
func (r *Repository) GetByOrgNodeAndCode(ctx context.Context, orgNodeID, code string) (*Shift, error) {
var s Shift
err := r.db.WithContext(ctx).
Where("org_node_id = ? AND code = ?", orgNodeID, code).
First(&s).Error
if err != nil {
return nil, err
}
return &s, nil
}

// CreateDependency 创建班次依赖。
func (r *Repository) CreateDependency(ctx context.Context, dep *ShiftDependency) error {
if dep.ID == "" {
dep.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(dep).Error
}

// DeleteDependency 删除班次依赖。
func (r *Repository) DeleteDependency(ctx context.Context, shiftID, dependsOnID, depType string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("shift_id = ? AND depends_on_id = ? AND dependency_type = ?", shiftID, dependsOnID, depType).
Delete(&ShiftDependency{}).Error
}

// GetDependencies 查询班次依赖列表。
func (r *Repository) GetDependencies(ctx context.Context) ([]ShiftDependency, error) {
var deps []ShiftDependency
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Find(&deps).Error
return deps, err
}

// GetDependenciesByShift 查询某班次的依赖列表。
func (r *Repository) GetDependenciesByShift(ctx context.Context, shiftID string) ([]ShiftDependency, error) {
var deps []ShiftDependency
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("shift_id = ?", shiftID).
Find(&deps).Error
return deps, err
}

// DeleteDependenciesByShift 删除某班次的所有依赖。
func (r *Repository) DeleteDependenciesByShift(ctx context.Context, shiftID string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("shift_id = ? OR depends_on_id = ?", shiftID, shiftID).
Delete(&ShiftDependency{}).Error
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
return r.db.AutoMigrate(&Shift{}, &ShiftDependency{})
}
