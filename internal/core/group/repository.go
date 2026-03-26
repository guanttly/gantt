package group

import (
"context"

"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

// Repository 分组数据访问层。
type Repository struct {
db *gorm.DB
}

// NewRepository 创建分组仓储。
func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// Create 创建分组。
func (r *Repository) Create(ctx context.Context, g *EmployeeGroup) error {
if g.ID == "" {
g.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(g).Error
}

// GetByID 根据 ID 查询分组。
func (r *Repository) GetByID(ctx context.Context, id string) (*EmployeeGroup, error) {
var g EmployeeGroup
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
First(&g).Error
if err != nil {
return nil, err
}
return &g, nil
}

// Update 更新分组。
func (r *Repository) Update(ctx context.Context, g *EmployeeGroup) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(g).Error
}

// Delete 删除分组。
func (r *Repository) Delete(ctx context.Context, id string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("id = ?", id).
Delete(&EmployeeGroup{}).Error
}

// List 查询分组列表。
func (r *Repository) List(ctx context.Context) ([]EmployeeGroup, error) {
var groups []EmployeeGroup
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Order("name ASC").
Find(&groups).Error
return groups, err
}

// AddMember 添加成员。
func (r *Repository) AddMember(ctx context.Context, m *GroupMember) error {
if m.ID == "" {
m.ID = uuid.New().String()
}
return r.db.WithContext(ctx).Create(m).Error
}

// RemoveMember 移除成员。
func (r *Repository) RemoveMember(ctx context.Context, groupID, employeeID string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("group_id = ? AND employee_id = ?", groupID, employeeID).
Delete(&GroupMember{}).Error
}

// GetMembers 获取分组成员列表。
func (r *Repository) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
var members []GroupMember
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("group_id = ?", groupID).
Find(&members).Error
return members, err
}

// GetMember 查询某成员是否在分组中。
func (r *Repository) GetMember(ctx context.Context, groupID, employeeID string) (*GroupMember, error) {
var m GroupMember
err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("group_id = ? AND employee_id = ?", groupID, employeeID).
First(&m).Error
if err != nil {
return nil, err
}
return &m, nil
}

// DeleteMembersByGroup 删除分组下所有成员。
func (r *Repository) DeleteMembersByGroup(ctx context.Context, groupID string) error {
return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
Where("group_id = ?", groupID).
Delete(&GroupMember{}).Error
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
return r.db.AutoMigrate(&EmployeeGroup{}, &GroupMember{})
}
