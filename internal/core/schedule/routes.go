package schedule

import (
	"gantt-saas/internal/core/approle"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册排班管理路由到 /api/v1/schedules。
func RegisterRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:self")).Get("/scheduling/assignments/self", h.GetSelfAssignments)
	r.Route("/schedules", func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:create")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}", h.GetByID)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:create")).Delete("/{id}", h.Delete)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:execute")).Post("/{id}/generate", h.Generate)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/assignments", h.GetAssignments)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:adjust")).Put("/{id}/assignments", h.AdjustAssignments)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:adjust")).Post("/{id}/validate", h.Validate)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:publish")).Post("/{id}/publish", h.Publish)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/changes", h.GetChanges)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/summary", h.GetSummary)
	})
}

// RegisterAppRoutes 注册排班应用路由到 /api/v1/app/scheduling/plans。
func RegisterAppRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:self")).Get("/app/scheduling/assignments/self", h.GetSelfAssignments)
	r.Route("/app/scheduling/plans", func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:create")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}", h.GetByID)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:create")).Delete("/{id}", h.Delete)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:execute")).Put("/{id}/execute", h.Generate)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/assignments", h.GetAssignments)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:adjust")).Put("/{id}/adjust", h.AdjustAssignments)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:adjust")).Post("/{id}/validate", h.Validate)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:publish")).Put("/{id}/publish", h.Publish)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/changes", h.GetChanges)
		r.With(approle.RequireAnyPermission(appRoleSvc, "schedule:view:node", "schedule:view:all")).Get("/{id}/summary", h.GetSummary)
	})
}
