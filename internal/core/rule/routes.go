package rule

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册规则管理路由到 /api/v1/rules。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/rules", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/effective", h.GetEffective)
		r.Post("/", h.Create)
		r.Post("/validate", h.Validate)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}
