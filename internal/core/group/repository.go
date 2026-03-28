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

// ListOptions 分组列表查询选项。
type ListOptions struct {
	Keyword string `json:"keyword"`
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
func (r *Repository) List(ctx context.Context, opts ListOptions) ([]EmployeeGroup, error) {
	var groups []EmployeeGroup
	tx := tenant.ApplyScopeOnColumn(ctx, r.db.WithContext(ctx), "employee_groups.org_node_id").
		Select("employee_groups.*, COUNT(group_members.id) AS member_count").
		Joins("LEFT JOIN group_members ON group_members.group_id = employee_groups.id AND group_members.org_node_id = employee_groups.org_node_id").
		Group("employee_groups.id")

	if opts.Keyword != "" {
		keyword := "%" + opts.Keyword + "%"
		tx = tx.Where("employee_groups.name LIKE ? OR employee_groups.description LIKE ?", keyword, keyword)
	}

	err := tx.
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
	err := tenant.ApplyScopeOnColumn(ctx, r.db.WithContext(ctx), "group_members.org_node_id").
		Select(`group_members.*, group_members.created_at AS joined_at, employees.name AS employee_name, employees.employee_no AS employee_no, employees.position AS position, employees.status AS status`).
		Joins("LEFT JOIN employees ON employees.id = group_members.employee_id AND employees.org_node_id = group_members.org_node_id").
		Where("group_id = ?", groupID).
		Order("group_members.created_at DESC").
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

// RemoveEmployeeFromAllGroups 将员工从所有分组中移除，返回受影响行数。
func (r *Repository) RemoveEmployeeFromAllGroups(ctx context.Context, employeeID string) (int64, error) {
	result := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("employee_id = ?", employeeID).
		Delete(&GroupMember{})
	return result.RowsAffected, result.Error
}

// GetMembersByEmployeeID 查询某员工所属的所有分组成员记录。
func (r *Repository) GetMembersByEmployeeID(ctx context.Context, employeeID string) ([]GroupMember, error) {
	var members []GroupMember
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("employee_id = ?", employeeID).
		Find(&members).Error
	return members, err
}

// GetMemberEmployeeIDs 获取指定分组的所有成员员工ID列表。
func (r *Repository) GetMemberEmployeeIDs(ctx context.Context, groupID string) ([]string, error) {
	var ids []string
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Model(&GroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("employee_id", &ids).Error
	return ids, err
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&EmployeeGroup{}, &GroupMember{})
}
