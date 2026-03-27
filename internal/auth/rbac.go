package auth

import "strings"

// 预置角色权限定义。
var rolePermissions = map[string][]string{
	string(RolePlatformAdmin): {"*"},
	string(RoleOrgAdmin):      {"org:*", "employee:*", "ai:*", "platform:manage_scope", "platform:user:manage"},
	string(RoleDeptAdmin):     {"employee:read"},
	string(RoleScheduler):     {"employee:read", "shift:read", "rule:read", "schedule:*", "leave:read"},
	string(RoleEmployee):      {"schedule:read:self", "leave:create:self", "preference:*:self"},
}

// SystemRoles 系统预置角色定义。
var SystemRoles = []Role{
	{ID: RoleIDPlatformAdmin, Name: string(RolePlatformAdmin), DisplayName: "平台管理员", Permissions: rolePermissions[string(RolePlatformAdmin)], IsSystem: true},
	{ID: RoleIDOrgAdmin, Name: string(RoleOrgAdmin), DisplayName: "机构管理员", Permissions: rolePermissions[string(RoleOrgAdmin)], IsSystem: true},
	{ID: RoleIDDeptAdmin, Name: string(RoleDeptAdmin), DisplayName: "科室管理员", Permissions: rolePermissions[string(RoleDeptAdmin)], IsSystem: true},
	{ID: RoleIDScheduler, Name: string(RoleScheduler), DisplayName: "排班负责人", Permissions: rolePermissions[string(RoleScheduler)], IsSystem: true},
	{ID: RoleIDEmployee, Name: string(RoleEmployee), DisplayName: "普通员工", Permissions: rolePermissions[string(RoleEmployee)], IsSystem: true},
}

// HasPermission 检查角色是否拥有指定权限。
func HasPermission(roleName, requiredPermission string) bool {
	perms, ok := rolePermissions[roleName]
	if !ok {
		return false
	}

	for _, perm := range perms {
		if perm == "*" {
			return true
		}
		if matchPermission(perm, requiredPermission) {
			return true
		}
	}
	return false
}

// matchPermission 匹配权限模式。
// 支持通配符：employee:* 匹配 employee:read、employee:write 等。
func matchPermission(pattern, target string) bool {
	if pattern == target {
		return true
	}

	patParts := strings.Split(pattern, ":")
	tgtParts := strings.Split(target, ":")

	for i, pp := range patParts {
		if pp == "*" {
			return true
		}
		if i >= len(tgtParts) || pp != tgtParts[i] {
			return false
		}
	}

	return len(patParts) == len(tgtParts)
}
