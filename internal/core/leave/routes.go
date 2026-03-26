package leave

import (
"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册请假管理路由到 /api/v1/leaves。
func RegisterRoutes(r chi.Router, h *Handler) {
r.Route("/leaves", func(r chi.Router) {
r.Get("/", h.List)
r.Post("/", h.Create)
r.Put("/{id}", h.Update)
r.Delete("/{id}", h.Delete)
r.Put("/{id}/approve", h.Approve)
})
}
