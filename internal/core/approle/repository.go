package approle

import (
	"context"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&EmployeeAppRole{}, &GroupDefaultAppRole{})
}

func (r *Repository) GetEmployeeByID(ctx context.Context, id string) (*employeeRecord, error) {
	var row employeeRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employees").
		Select("employees.id, employees.org_node_id, employees.name, employees.status, org_nodes.path").
		Joins("JOIN org_nodes ON org_nodes.id = employees.org_node_id").
		Where("employees.id = ?", id).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *Repository) ListEmployeesByIDs(ctx context.Context, ids []string) ([]employeeRecord, error) {
	if len(ids) == 0 {
		return []employeeRecord{}, nil
	}
	var rows []employeeRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employees").
		Select("employees.id, employees.org_node_id, employees.name, employees.status, org_nodes.path").
		Joins("JOIN org_nodes ON org_nodes.id = employees.org_node_id").
		Where("employees.id IN ?", ids).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) GetGroupByID(ctx context.Context, id string) (*groupRecord, error) {
	var row groupRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employee_groups").
		Select("employee_groups.id, employee_groups.org_node_id, employee_groups.name, org_nodes.path").
		Joins("JOIN org_nodes ON org_nodes.id = employee_groups.org_node_id").
		Where("employee_groups.id = ?", id).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *Repository) GetBoundEmployeeByUserID(ctx context.Context, userID string) (*employeeRecord, error) {
	var row employeeRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("platform_users").
		Select("employees.id, employees.org_node_id, employees.name, employees.status, org_nodes.path").
		Joins("JOIN employees ON employees.id = platform_users.bound_employee_id").
		Joins("JOIN org_nodes ON org_nodes.id = employees.org_node_id").
		Where("platform_users.id = ?", userID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *Repository) ListGroupMembers(ctx context.Context, groupID string) ([]groupMemberRecord, error) {
	var rows []groupMemberRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("group_members").
		Select("group_members.employee_id, employees.org_node_id, employees.name, employees.status, org_nodes.path").
		Joins("JOIN employees ON employees.id = group_members.employee_id").
		Joins("JOIN org_nodes ON org_nodes.id = employees.org_node_id").
		Where("group_members.group_id = ?", groupID).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListEmployeeRoles(ctx context.Context, employeeID string) ([]employeeRoleRow, error) {
	var rows []employeeRoleRow
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employee_app_roles").
		Select("employee_app_roles.*, org_nodes.name AS org_node_name, employee_groups.name AS source_group_name, employees.name AS employee_name").
		Joins("JOIN org_nodes ON org_nodes.id = employee_app_roles.org_node_id").
		Joins("JOIN employees ON employees.id = employee_app_roles.employee_id").
		Joins("LEFT JOIN employee_groups ON employee_groups.id = employee_app_roles.source_group_id").
		Where("employee_app_roles.employee_id = ?", employeeID).
		Order("employee_app_roles.granted_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListEmployeeRolesByEmployeeIDs(ctx context.Context, employeeIDs []string) ([]employeeRoleRow, error) {
	if len(employeeIDs) == 0 {
		return []employeeRoleRow{}, nil
	}
	var rows []employeeRoleRow
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employee_app_roles").
		Select("employee_app_roles.*, org_nodes.name AS org_node_name, employee_groups.name AS source_group_name, employees.name AS employee_name").
		Joins("JOIN org_nodes ON org_nodes.id = employee_app_roles.org_node_id").
		Joins("JOIN employees ON employees.id = employee_app_roles.employee_id").
		Joins("LEFT JOIN employee_groups ON employee_groups.id = employee_app_roles.source_group_id").
		Where("employee_app_roles.employee_id IN ?", employeeIDs).
		Order("employee_app_roles.granted_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) GetEmployeeRoleByID(ctx context.Context, id string) (*EmployeeAppRole, error) {
	var role EmployeeAppRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) FindEmployeeRole(ctx context.Context, employeeID, orgNodeID, appRole string) (*EmployeeAppRole, error) {
	var role EmployeeAppRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("employee_id = ? AND org_node_id = ? AND app_role = ?", employeeID, orgNodeID, appRole).
		First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) CreateEmployeeRole(ctx context.Context, role *EmployeeAppRole) error {
	if role.ID == "" {
		role.ID = uuid.New().String()
	}
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(role).Error
}

func (r *Repository) DeleteEmployeeRole(ctx context.Context, id string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", id).Delete(&EmployeeAppRole{}).Error
}

func (r *Repository) DeleteEmployeeRolesByGroup(ctx context.Context, groupID string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("source = ? AND source_group_id = ?", SourceGroup, groupID).
		Delete(&EmployeeAppRole{}).Error
}

func (r *Repository) DeleteEmployeeRoleByGroupMember(ctx context.Context, groupID, employeeID string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("employee_id = ? AND source = ? AND source_group_id = ?", employeeID, SourceGroup, groupID).
		Delete(&EmployeeAppRole{}).Error
}

func (r *Repository) DeleteAllEmployeeRoles(ctx context.Context, employeeID string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("employee_id = ?", employeeID).Delete(&EmployeeAppRole{}).Error
}

func (r *Repository) DeleteExpiredRoles(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&EmployeeAppRole{})
	return result.RowsAffected, result.Error
}

func (r *Repository) ListGroupDefaultRoles(ctx context.Context, groupID string) ([]GroupDefaultAppRoleResponse, error) {
	var rows []GroupDefaultAppRoleResponse
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("group_default_app_roles").
		Select("group_default_app_roles.id, group_default_app_roles.group_id, group_default_app_roles.org_node_id, org_nodes.name AS org_node_name, group_default_app_roles.app_role, group_default_app_roles.created_by, group_default_app_roles.created_at").
		Joins("JOIN org_nodes ON org_nodes.id = group_default_app_roles.org_node_id").
		Where("group_default_app_roles.group_id = ?", groupID).
		Order("group_default_app_roles.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListGroupDefaultRoleModels(ctx context.Context, groupID string) ([]GroupDefaultAppRole, error) {
	var rows []GroupDefaultAppRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("group_id = ?", groupID).Find(&rows).Error
	return rows, err
}

func (r *Repository) GetGroupDefaultRoleByID(ctx context.Context, id string) (*GroupDefaultAppRole, error) {
	var role GroupDefaultAppRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) FindGroupDefaultRole(ctx context.Context, groupID, orgNodeID, appRole string) (*GroupDefaultAppRole, error) {
	var role GroupDefaultAppRole
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("group_id = ? AND org_node_id = ? AND app_role = ?", groupID, orgNodeID, appRole).
		First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) CreateGroupDefaultRole(ctx context.Context, role *GroupDefaultAppRole) error {
	if role.ID == "" {
		role.ID = uuid.New().String()
	}
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(role).Error
}

func (r *Repository) DeleteGroupDefaultRole(ctx context.Context, id string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", id).Delete(&GroupDefaultAppRole{}).Error
}

func (r *Repository) DeleteGroupDefaultRolesByGroup(ctx context.Context, groupID string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("group_id = ?", groupID).Delete(&GroupDefaultAppRole{}).Error
}

func (r *Repository) Summaries(ctx context.Context) ([]summaryRow, error) {
	var rows []summaryRow
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employee_app_roles").
		Select("employee_app_roles.org_node_id, org_nodes.name AS org_node_name, employee_app_roles.app_role, COUNT(*) AS count").
		Joins("JOIN org_nodes ON org_nodes.id = employee_app_roles.org_node_id").
		Group("employee_app_roles.org_node_id, org_nodes.name, employee_app_roles.app_role").
		Order("employee_app_roles.org_node_id ASC, employee_app_roles.app_role ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) Expiring(ctx context.Context, before time.Time) ([]employeeRoleRow, error) {
	var rows []employeeRoleRow
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employee_app_roles").
		Select("employee_app_roles.*, org_nodes.name AS org_node_name, employee_groups.name AS source_group_name, employees.name AS employee_name").
		Joins("JOIN org_nodes ON org_nodes.id = employee_app_roles.org_node_id").
		Joins("JOIN employees ON employees.id = employee_app_roles.employee_id").
		Joins("LEFT JOIN employee_groups ON employee_groups.id = employee_app_roles.source_group_id").
		Where("employee_app_roles.expires_at IS NOT NULL AND employee_app_roles.expires_at <= ?", before).
		Order("employee_app_roles.expires_at ASC").
		Scan(&rows).Error
	return rows, err
}
