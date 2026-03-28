package rule

import (
	"gantt-saas/internal/core/approle"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册规则管理路由到 /api/v1/rules。
func RegisterRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/rules", h, appRoleSvc)
}

// RegisterPlatformRoutes 注册平台侧规则管理路由到 /api/v1/platform/rules。
func RegisterPlatformRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/platform/rules", h, appRoleSvc)
}

// RegisterAppRoutes 注册排班应用规则管理路由到 /api/v1/app/rules。
func RegisterAppRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/app/rules", h, appRoleSvc)
}

// RegisterAppRefRoutes 注册排班应用只读规则引用路由到 /api/v1/app/scheduling/ref/rules。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/rules", func(r chi.Router) {
		r.Get("/", h.GetEffective)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler, appRoleSvc *approle.Service) {
	r.Route(basePath, func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:view:node", "rule:manage")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:view:node", "rule:manage")).Get("/effective", h.GetEffective)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:manage")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:manage")).Post("/validate", h.Validate)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:view:node", "rule:manage")).Get("/{id}", h.GetByID)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:manage")).Put("/{id}", h.Update)
		r.With(approle.RequireAnyPermission(appRoleSvc, "rule:manage")).Delete("/{id}", h.Delete)
	})
}
