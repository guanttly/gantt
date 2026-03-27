package group

import (
	"gantt-saas/internal/core/approle"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册分组管理路由到 /api/v1/groups。
func RegisterRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/groups", h, appRoleSvc)
}

// RegisterPlatformRoutes 注册平台侧分组管理路由到 /api/v1/platform/groups。
func RegisterPlatformRoutes(r chi.Router, h *Handler, appRoleSvc *approle.Service) {
	registerAt(r, "/platform/groups", h, appRoleSvc)
}

// RegisterAppRefRoutes 注册排班应用只读分组引用路由到 /api/v1/app/scheduling/ref/groups。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/groups", func(r chi.Router) {
		r.Get("/", h.List)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler, appRoleSvc *approle.Service) {
	r.Route(basePath, func(r chi.Router) {
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:view:node", "group:manage")).Get("/", h.List)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:manage")).Post("/", h.Create)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:manage")).Put("/{id}", h.Update)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:manage")).Delete("/{id}", h.Delete)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:view:node", "group:manage")).Get("/{id}/members", h.GetMembers)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:manage")).Post("/{id}/members", h.AddMember)
		r.With(approle.RequireAnyPermission(appRoleSvc, "group:manage")).Delete("/{id}/members/{eid}", h.RemoveMember)
	})
}
