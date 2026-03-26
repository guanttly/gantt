package group

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册分组管理路由到 /api/v1/groups。
func RegisterRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/groups", h)
}

// RegisterPlatformRoutes 注册平台侧分组管理路由到 /api/v1/platform/groups。
func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/platform/groups", h)
}

// RegisterAppRefRoutes 注册排班应用只读分组引用路由到 /api/v1/app/scheduling/ref/groups。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/groups", func(r chi.Router) {
		r.Get("/", h.List)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler) {
	r.Route(basePath, func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/members", h.GetMembers)
		r.Post("/{id}/members", h.AddMember)
		r.Delete("/{id}/members/{eid}", h.RemoveMember)
	})
}
