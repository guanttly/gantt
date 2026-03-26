package group

import (
"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册分组管理路由到 /api/v1/groups。
func RegisterRoutes(r chi.Router, h *Handler) {
r.Route("/groups", func(r chi.Router) {
r.Get("/", h.List)
r.Post("/", h.Create)
r.Put("/{id}", h.Update)
r.Delete("/{id}", h.Delete)
r.Get("/{id}/members", h.GetMembers)
r.Post("/{id}/members", h.AddMember)
r.Delete("/{id}/members/{eid}", h.RemoveMember)
})
}
