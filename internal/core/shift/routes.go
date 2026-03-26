package shift

import (
"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册班次管理路由到 /api/v1/shifts。
func RegisterRoutes(r chi.Router, h *Handler) {
r.Route("/shifts", func(r chi.Router) {
r.Get("/", h.List)
r.Post("/", h.Create)
r.Get("/dependencies", h.GetDependencies)
r.Post("/dependencies", h.AddDependency)
r.Put("/{id}", h.Update)
r.Delete("/{id}", h.Delete)
})
}
