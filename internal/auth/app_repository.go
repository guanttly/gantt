package auth

import (
	"context"

	"gantt-saas/internal/tenant"

	"gorm.io/gorm"
)

func (r *Repository) GetEmployeeByIDForAppAuth(ctx context.Context, id string) (*appEmployeeRecord, error) {
	var row appEmployeeRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employees").
		Select("employees.id, employees.org_node_id, employees.name, employees.employee_no, employees.phone, employees.email, employees.status, employees.scheduling_role, employees.app_password_hash, employees.app_must_reset_pwd, org_nodes.name AS org_node_name, org_nodes.path AS org_node_path, employees.created_at, employees.updated_at").
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

func (r *Repository) FindEmployeeByLoginID(ctx context.Context, loginID string) (*appEmployeeRecord, error) {
	var rows []appEmployeeRecord
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employees").
		Select("employees.id, employees.org_node_id, employees.name, employees.employee_no, employees.phone, employees.email, employees.status, employees.scheduling_role, employees.app_password_hash, employees.app_must_reset_pwd, org_nodes.name AS org_node_name, org_nodes.path AS org_node_path, employees.created_at, employees.updated_at").
		Joins("JOIN org_nodes ON org_nodes.id = employees.org_node_id").
		Where("employees.employee_no = ? OR employees.phone = ? OR employees.email = ? OR employees.name = ?", loginID, loginID, loginID, loginID).
		Limit(2).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(rows) > 1 {
		return nil, ErrAppLoginIDAmbiguous
	}
	return &rows[0], nil
}

func (r *Repository) UpdateEmployeeAppCredentials(ctx context.Context, id, passwordHash string, mustResetPwd bool) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Table("employees").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"app_password_hash":  passwordHash,
			"app_must_reset_pwd": mustResetPwd,
		}).Error
}
