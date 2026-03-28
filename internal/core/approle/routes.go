package approle

import "github.com/go-chi/chi/v5"

func RegisterManagementRoutes(r chi.Router, h *Handler, svc *Service) {
	r.With(RequireAnyPermission(svc, "app-role:manage")).Post("/employees/batch-app-roles", h.BatchAssignEmployeeRoles)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Get("/employees/{id}/app-roles", h.ListEmployeeRoles)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Post("/employees/{id}/app-roles", h.AssignEmployeeRole)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Delete("/employees/{id}/app-roles/{roleId}", h.RemoveEmployeeRole)
}

func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	r.Get("/platform/app-roles/summary", h.Summary)
	r.Get("/platform/app-roles/expiring", h.Expiring)

	r.Post("/platform/employees/batch-app-roles", h.BatchAssignEmployeeRoles)
	r.Get("/platform/employees/app-roles", h.ListEmployeeRolesBatch)
	r.Get("/platform/employees/{id}/app-roles", h.ListEmployeeRoles)
	r.Post("/platform/employees/{id}/app-roles", h.AssignEmployeeRole)
	r.Delete("/platform/employees/{id}/app-roles/{roleId}", h.RemoveEmployeeRole)
}

func RegisterUserRoutes(r chi.Router, h *Handler) {
	r.Get("/auth/app-roles", h.MyRoles)
	r.Get("/auth/app-permissions", h.MyPermissions)
}

func RegisterAppRoutes(r chi.Router, h *Handler) {
	r.Get("/app/scheduling/auth/my-roles", h.MyRoles)
	r.Get("/app/scheduling/auth/permissions", h.MyPermissions)
}

func RegisterAppManagementRoutes(r chi.Router, h *Handler, svc *Service) {
	r.With(RequireAnyPermission(svc, "app-role:manage")).Post("/app/employees/batch-app-roles", h.BatchAssignEmployeeRoles)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Get("/app/employees/{id}/app-roles", h.ListEmployeeRoles)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Post("/app/employees/{id}/app-roles", h.AssignEmployeeRole)
	r.With(RequireAnyPermission(svc, "app-role:manage")).Delete("/app/employees/{id}/app-roles/{roleId}", h.RemoveEmployeeRole)
}
