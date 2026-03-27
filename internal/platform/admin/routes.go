package admin

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册平台超级管理员路由到 /api/v1/admin/*。
func RegisterRoutes(r chi.Router, dashHandler *DashboardHandler, sysHandler *SystemConfigHandler, orgHandler *OrganizationHandler) {
	r.Get("/admin/dashboard", dashHandler.GetDashboard)
	r.Route("/admin/organizations", func(r chi.Router) {
		r.Get("/", orgHandler.List)
		r.Post("/", orgHandler.Create)
		r.Get("/{id}", orgHandler.GetByID)
		r.Put("/{id}/suspend", orgHandler.Suspend)
		r.Put("/{id}/activate", orgHandler.Activate)
	})
	r.Route("/admin/system", func(r chi.Router) {
		r.Get("/config", sysHandler.GetConfig)
		r.Put("/config", sysHandler.UpdateConfig)
	})
}

// RegisterPlatformUserRoutes 注册平台账号管理路由到 /api/v1/admin/platform-users。
func RegisterPlatformUserRoutes(r chi.Router, userHandler *PlatformUserHandler) {
	r.Route("/admin/platform-users", func(r chi.Router) {
		r.Get("/", userHandler.List)
		r.Post("/", userHandler.Create)
		r.Get("/{id}", userHandler.GetByID)
		r.Put("/{id}/reset-pwd", userHandler.ResetPassword)
		r.Put("/{id}/enable", userHandler.Enable)
		r.Put("/{id}/disable", userHandler.Disable)
		r.Delete("/{id}", userHandler.Delete)
	})
}
