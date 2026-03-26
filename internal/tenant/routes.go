package tenant

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册组织树管理路由到 /api/v1/admin/org-nodes。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/admin/org-nodes", func(r chi.Router) {
		r.Get("/", h.GetTree)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Put("/{id}/suspend", h.Suspend)
		r.Put("/{id}/activate", h.Activate)
		r.Put("/{id}/move", h.Move)
		r.Get("/{id}/children", h.GetChildren)
	})
}
