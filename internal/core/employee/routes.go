package employee

import (
	"net/http"

	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册员工管理路由到 /api/v1/employees。
func RegisterRoutes(r chi.Router, h *Handler) {
	registerAt(r, "/employees", h)
}

// RegisterPlatformRoutes 注册平台侧员工管理路由到 /api/v1/platform/employees。
func RegisterPlatformRoutes(r chi.Router, h *Handler) {
	r.Route("/platform/employees", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := tenant.WithScopeTree(req.Context(), true)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/batch-transfer", h.BatchTransfer)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}/reset-pwd", h.ResetPassword)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/transfer", h.Transfer)
	})
}

// RegisterAppRefRoutes 注册排班应用只读员工引用路由到 /api/v1/app/scheduling/ref/employees。
func RegisterAppRefRoutes(r chi.Router, h *Handler) {
	r.Route("/app/scheduling/ref/employees", func(r chi.Router) {
		r.Get("/", h.List)
	})
}

func registerAt(r chi.Router, basePath string, h *Handler) {
	r.Route(basePath, func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}
