package shift

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册班次管理路由到 /api/v1/shifts。
func RegisterRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/shifts", h)
}

// RegisterPlatformRoutes 注册平台侧班次管理路由到 /api/v1/platform/shifts。
func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/platform/shifts", h)
}

// RegisterAppRefRoutes 注册排班应用只读班次引用路由到 /api/v1/app/scheduling/ref/shifts。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/shifts", func(r chi.Router) {
		r.Get("/", h.ListAvailable)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler) {
	r.Route(basePath, func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/dependencies", h.GetDependencies)
		r.Post("/dependencies", h.AddDependency)
		r.Put("/{id}", h.Update)
		r.Put("/{id}/toggle", h.ToggleStatus)
		r.Delete("/{id}", h.Delete)
	})
}
