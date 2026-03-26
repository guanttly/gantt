package employee

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册员工管理路由到 /api/v1/employees。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/employees", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}
