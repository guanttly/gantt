package auth

type RoleName string

const (
	RolePlatformAdmin RoleName = "platform_admin"
	RoleOrgAdmin      RoleName = "org_admin"
	RoleDeptAdmin     RoleName = "dept_admin"
	RoleScheduler     RoleName = "scheduler"
	RoleEmployee      RoleName = "employee"
)

const (
	RoleIDPlatformAdmin = "role-platform-admin"
	RoleIDOrgAdmin      = "role-org-admin"
	RoleIDDeptAdmin     = "role-dept-admin"
	RoleIDScheduler     = "role-scheduler"
	RoleIDEmployee      = "role-employee"
)
