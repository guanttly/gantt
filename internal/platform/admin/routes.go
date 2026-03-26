package admin

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册平台管理路由到 /api/v1/admin/*。
func RegisterRoutes(r chi.Router, dashHandler *DashboardHandler, sysHandler *SystemConfigHandler) {
	r.Get("/admin/dashboard", dashHandler.GetDashboard)
	r.Route("/admin/system", func(r chi.Router) {
		r.Get("/config", sysHandler.GetConfig)
		r.Put("/config", sysHandler.UpdateConfig)
	})
}
