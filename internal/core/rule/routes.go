package rule

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册规则管理路由到 /api/v1/rules。
func RegisterRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/rules", h, h.RestoreInheritance)
}

// RegisterPlatformRoutes 注册平台侧规则管理路由到 /api/v1/platform/rules。
func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/platform/rules", h, h.RestoreInheritance)
}

// RegisterAppRefRoutes 注册排班应用只读规则引用路由到 /api/v1/app/scheduling/ref/rules。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/rules", func(r chi.Router) {
		r.Get("/", h.GetEffective)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler, restoreHandler http.HandlerFunc) {
	r.Route(basePath, func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/effective", h.GetEffective)
		r.Post("/", h.Create)
		r.Post("/validate", h.Validate)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Put("/{id}/disable", h.DisableInherited)
		r.Put("/{id}/restore", restoreHandler)
		r.Put("/{id}/enable", restoreHandler)
		r.Delete("/{id}", h.Delete)
	})
}
