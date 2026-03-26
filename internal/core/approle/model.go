package approle

import (
	"time"

	"gantt-saas/internal/tenant"
)

const (
	RoleScheduleAdmin = "app:schedule_admin"
	RoleScheduler     = "app:scheduler"
	RoleLeaveApprover = "app:leave_approver"
	RoleEmployee      = "app:employee"

	SourceManual = "manual"
	SourceGroup  = "group"
	SourceSystem = "system"
)

var validAppRoles = map[string]bool{
	RoleScheduleAdmin: true,
	RoleScheduler:     true,
	RoleLeaveApprover: true,
}

type EmployeeAppRole struct {
	ID            string     `gorm:"primaryKey;size:64" json:"id"`
	EmployeeID    string     `gorm:"size:64;not null;index:idx_emp_app_role_emp" json:"employee_id"`
	OrgNodeID     string     `gorm:"size:64;not null;index:idx_emp_app_role_node" json:"org_node_id"`
	AppRole       string     `gorm:"size:64;not null;index:idx_emp_app_role_role" json:"app_role"`
	Source        string     `gorm:"size:16;not null;default:manual" json:"source"`
	SourceGroupID *string    `gorm:"size:64;index:idx_emp_app_role_source_group" json:"source_group_id,omitempty"`
	GrantedBy     string     `gorm:"size:64;not null" json:"granted_by"`
	GrantedAt     time.Time  `gorm:"autoCreateTime" json:"granted_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

func (EmployeeAppRole) TableName() string { return "employee_app_roles" }

type GroupDefaultAppRole struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	GroupID   string    `gorm:"size:64;not null;index:idx_group_default_role_group" json:"group_id"`
	OrgNodeID string    `gorm:"size:64;not null;index:idx_group_default_role_node" json:"org_node_id"`
	AppRole   string    `gorm:"size:64;not null" json:"app_role"`
	CreatedBy string    `gorm:"size:64;not null" json:"created_by"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (GroupDefaultAppRole) TableName() string { return "group_default_app_roles" }

type EmployeeAppRoleResponse struct {
	ID              string     `json:"id"`
	EmployeeID      string     `json:"employee_id"`
	OrgNodeID       string     `json:"org_node_id"`
	OrgNodeName     string     `json:"org_node_name"`
	AppRole         string     `json:"app_role"`
	Source          string     `json:"source"`
	SourceGroupID   *string    `json:"source_group_id,omitempty"`
	SourceGroupName *string    `json:"source_group_name,omitempty"`
	GrantedBy       string     `json:"granted_by"`
	GrantedAt       time.Time  `json:"granted_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

type GroupDefaultAppRoleResponse struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	OrgNodeID   string    `json:"org_node_id"`
	OrgNodeName string    `json:"org_node_name"`
	AppRole     string    `json:"app_role"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type AppRoleSummaryItem struct {
	OrgNodeID   string `json:"org_node_id"`
	OrgNodeName string `json:"org_node_name"`
	AppRole     string `json:"app_role"`
	Count       int64  `json:"count"`
}

type ExpiringRoleItem struct {
	EmployeeAppRoleResponse
	EmployeeName string `json:"employee_name"`
}

type MyRolesResponse struct {
	EmployeeID  string                    `json:"employee_id"`
	OrgNodeID   string                    `json:"org_node_id"`
	OrgNodeName string                    `json:"org_node_name"`
	AppRoles    []EmployeeAppRoleResponse `json:"app_roles"`
}

type MyPermissionsResponse struct {
	EmployeeID  string   `json:"employee_id"`
	OrgNodeID   string   `json:"org_node_id"`
	OrgNodeName string   `json:"org_node_name"`
	Permissions []string `json:"permissions"`
}

type employeeRecord struct {
	ID        string
	OrgNodeID string
	Name      string
	Status    string
	Path      string
}

type groupRecord struct {
	ID        string
	OrgNodeID string
	Name      string
	Path      string
}

type groupMemberRecord struct {
	EmployeeID string
	OrgNodeID  string
	Name       string
	Status     string
	Path       string
}

type employeeRoleRow struct {
	EmployeeAppRole
	OrgNodeName     string
	SourceGroupName *string
	EmployeeName    string
}

type summaryRow struct {
	OrgNodeID   string
	OrgNodeName string
	AppRole     string
	Count       int64
}

var _ = tenant.TenantModel{}
