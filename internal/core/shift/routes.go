package shift

import (
	"gantt-saas/internal/core/approle"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册班次管理路由到 /api/v1/shifts。
func RegisterRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/shifts", h, appRoleSvc)
}

// RegisterPlatformRoutes 注册平台侧班次管理路由到 /api/v1/platform/shifts。
func RegisterPlatformRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/platform/shifts", h, appRoleSvc)
}

// RegisterAppRefRoutes 注册排班应用只读班次引用路由到 /api/v1/app/scheduling/ref/shifts。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/shifts", func(r chi.Router) {
		r.Get("/", h.ListAvailable)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler, appRoleSvc *approle.Service) {
	r.Route(basePath, func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:view:node", "shift:manage")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:manage")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:view:node", "shift:manage")).Get("/dependencies", h.GetDependencies)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:manage")).Post("/dependencies", h.AddDependency)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:manage")).Put("/{id}", h.Update)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:manage")).Put("/{id}/toggle", h.ToggleStatus)
		r.With(approle.RequireAnyPermission(appRoleSvc, "shift:manage")).Delete("/{id}", h.Delete)
	})
}
