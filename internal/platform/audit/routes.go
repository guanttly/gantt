package audit

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册审计日志查询路由到 /api/v1/admin/audit-logs。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/admin/audit-logs", func(r chi.Router) {
		r.Get("/", h.List)
	})
}
