package leave

import (
	"gantt-saas/internal/core/approle"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册请假管理路由到 /api/v1/leaves。
func RegisterRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	r.Route("/leaves", func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self", "leave:view:node", "leave:approve")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Put("/{id}", h.Update)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Delete("/{id}", h.Delete)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:approve")).Put("/{id}/approve", h.Approve)
	})
}

// RegisterAppRoutes 注册排班应用请假路由到 /api/v1/app/leaves。
func RegisterAppRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	r.Route("/app/leaves", func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self", "leave:view:node", "leave:approve")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Put("/{id}", h.Update)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:create:self")).Delete("/{id}", h.Delete)
		r.With(approle.RequireAnyPermission(appRoleSvc, "leave:approve")).Put("/{id}/approve", h.Approve)
	})
}
