package auth

import (
	"context"

	"gantt-saas/internal/tenant"

	"gorm.io/gorm"
)

// Repository 认证相关数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建认证仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ── User ──

// CreateUser 创建用户。
func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(user).Error
}

// GetUserByID 根据 ID 查询用户。
func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名查询用户。
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱查询用户。
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息。
func (r *Repository) UpdateUser(ctx context.Context, user *User) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Save(user).Error
}

// ── Role ──

// CreateRole 创建角色。
func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(role).Error
}

// GetRoleByID 根据 ID 查询角色。
func (r *Repository) GetRoleByID(ctx context.Context, id string) (*Role, error) {
	var role Role
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRoleByName 根据角色名查询角色。
func (r *Repository) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetAllRoles 获取所有角色。
func (r *Repository) GetAllRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Order("name ASC").Find(&roles).Error
	return roles, err
}

// UpsertRole 创建或更新角色（用于系统角色初始化）。
func (r *Repository) UpsertRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", role.ID).
		Assign(Role{
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Permissions: role.Permissions,
			IsSystem:    role.IsSystem,
		}).
		FirstOrCreate(role).Error
}

// ── UserNodeRole ──

// CreateUserNodeRole 创建用户-节点-角色关联。
func (r *Repository) CreateUserNodeRole(ctx context.Context, unr *UserNodeRole) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(unr).Error
}

// GetUserNodeRoles 获取用户在所有节点的角色关联列表。
func (r *Repository) GetUserNodeRoles(ctx context.Context, userID string) ([]UserNodeRole, error) {
	var unrs []UserNodeRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Preload("Role").
		Where("user_id = ?", userID).
		Find(&unrs).Error
	return unrs, err
}

// GetUserNodeRole 查询用户在指定节点下的角色关联。
func (r *Repository) GetUserNodeRole(ctx context.Context, userID, orgNodeID string) (*UserNodeRole, error) {
	var unr UserNodeRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Preload("Role").
		Where("user_id = ? AND org_node_id = ?", userID, orgNodeID).
		First(&unr).Error
	if err != nil {
		return nil, err
	}
	return &unr, nil
}

// DeleteUserNodeRole 删除用户-节点-角色关联。
func (r *Repository) DeleteUserNodeRole(ctx context.Context, id string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).
		Delete(&UserNodeRole{}).Error
}

// GetUsersByNodeID 获取某组织节点下的所有用户关联。
func (r *Repository) GetUsersByNodeID(ctx context.Context, orgNodeID string) ([]UserNodeRole, error) {
	var unrs []UserNodeRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Preload("User").
		Preload("Role").
		Where("org_node_id = ?", orgNodeID).
		Find(&unrs).Error
	return unrs, err
}

// ── AutoMigrate ──

// AutoMigrate 自动迁移认证相关表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&User{}, &Role{}, &UserNodeRole{})
}
