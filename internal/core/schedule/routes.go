package schedule

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册排班管理路由到 /api/v1/schedules。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/schedules", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/generate", h.Generate)
		r.Get("/{id}/assignments", h.GetAssignments)
		r.Put("/{id}/assignments", h.AdjustAssignments)
		r.Post("/{id}/validate", h.Validate)
		r.Post("/{id}/publish", h.Publish)
		r.Get("/{id}/changes", h.GetChanges)
		r.Get("/{id}/summary", h.GetSummary)
	})
}
