package auth

import (
	"testing"
)

func TestHasPermission_PlatformAdmin(t *testing.T) {
	// platform_admin 有全部权限
	if !HasPermission(string(RolePlatformAdmin), "anything") {
		t.Error("platform_admin should have any permission")
	}
	if !HasPermission(string(RolePlatformAdmin), "employee:create") {
		t.Error("platform_admin should have employee:create")
	}
}

func TestHasPermission_OrgAdmin(t *testing.T) {
	// org_admin 有 employee:* 等
	if !HasPermission(string(RoleOrgAdmin), "employee:read") {
		t.Error("org_admin should have employee:read")
	}
	if !HasPermission(string(RoleOrgAdmin), "employee:create") {
		t.Error("org_admin should have employee:create")
	}
	if !HasPermission(string(RoleOrgAdmin), "schedule:create") {
		t.Error("org_admin should have schedule:create")
	}
}

func TestHasPermission_Employee(t *testing.T) {
	// employee 只有有限权限
	if !HasPermission(string(RoleEmployee), "schedule:read:self") {
		t.Error("employee should have schedule:read:self")
	}
	if HasPermission(string(RoleEmployee), "schedule:create") {
		t.Error("employee should NOT have schedule:create")
	}
	if HasPermission(string(RoleEmployee), "employee:read") {
		t.Error("employee should NOT have employee:read")
	}
}

func TestHasPermission_Scheduler(t *testing.T) {
	if !HasPermission(string(RoleScheduler), "schedule:create") {
		t.Error("scheduler should have schedule:create")
	}
	if !HasPermission(string(RoleScheduler), "employee:read") {
		t.Error("scheduler should have employee:read")
	}
	if HasPermission(string(RoleScheduler), "employee:create") {
		t.Error("scheduler should NOT have employee:create")
	}
}

func TestHasPermission_UnknownRole(t *testing.T) {
	if HasPermission("unknown_role", "anything") {
		t.Error("unknown role should not have any permission")
	}
}

func TestMatchPermission(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"*", "anything", true},
		{"employee:*", "employee:read", true},
		{"employee:*", "employee:create", true},
		{"employee:read", "employee:read", true},
		{"employee:read", "employee:create", false},
		{"schedule:read:self", "schedule:read:self", true},
		{"schedule:read:self", "schedule:read", false},
		{"preference:*:self", "preference:create:self", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"→"+tt.target, func(t *testing.T) {
			if got := matchPermission(tt.pattern, tt.target); got != tt.want {
				t.Errorf("matchPermission(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		})
	}
}

func TestSystemRoles(t *testing.T) {
	if len(SystemRoles) != 5 {
		t.Errorf("SystemRoles count = %d, want 5", len(SystemRoles))
	}
	for _, role := range SystemRoles {
		if role.ID == "" || role.Name == "" || role.DisplayName == "" {
			t.Errorf("system role %+v has empty fields", role)
		}
		if !role.IsSystem {
			t.Errorf("system role %s should have IsSystem=true", role.Name)
		}
	}
}
