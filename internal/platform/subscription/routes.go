package subscription

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册订阅管理路由到 /api/v1/admin/subscriptions。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/admin/subscriptions", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
	})
}
