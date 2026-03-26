package approle

import "github.com/go-chi/chi/v5"

func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	r.Get("/platform/app-roles/summary", h.Summary)
	r.Get("/platform/app-roles/expiring", h.Expiring)

	r.Post("/platform/employees/batch-app-roles", h.BatchAssignEmployeeRoles)
	r.Get("/platform/employees/{id}/app-roles", h.ListEmployeeRoles)
	r.Post("/platform/employees/{id}/app-roles", h.AssignEmployeeRole)
	r.Delete("/platform/employees/{id}/app-roles/{roleId}", h.RemoveEmployeeRole)

	r.Get("/platform/groups/{id}/default-app-roles", h.ListGroupDefaultRoles)
	r.Post("/platform/groups/{id}/default-app-roles", h.AssignGroupDefaultRole)
	r.Delete("/platform/groups/{id}/default-app-roles/{roleId}", h.RemoveGroupDefaultRole)
}

func RegisterUserRoutes(r chi.Router, h *Handler) {
	r.Get("/auth/app-roles", h.MyRoles)
	r.Get("/auth/app-permissions", h.MyPermissions)
}

func RegisterAppRoutes(r chi.Router, h *Handler) {
	r.Get("/app/scheduling/auth/my-roles", h.MyRoles)
	r.Get("/app/scheduling/auth/permissions", h.MyPermissions)
}
