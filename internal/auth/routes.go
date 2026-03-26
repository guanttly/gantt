package auth

import (
	"github.com/go-chi/chi/v5"
)

// RegisterPublicRoutes 注册公开路由（无需认证）到 /api/v1/auth。
func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)
	})
}

// RegisterProtectedRoutes 注册受保护路由（需认证）到 /api/v1/auth/*。
// 注意：使用直接注册而非 r.Route("/auth") 以避免与 RegisterPublicRoutes 冲突。
func RegisterProtectedRoutes(r chi.Router, h *Handler) {
	r.Post("/auth/switch-node", h.SwitchNode)
	r.Post("/auth/password/reset", h.ResetPassword)
	r.Post("/auth/password/force-reset", h.ForceResetPassword)
	r.Get("/auth/me", h.GetMe)
}

// RegisterAdminRoutes 注册管理路由（需认证+权限）到 /api/v1/admin/auth。
func RegisterAdminRoutes(r chi.Router, h *Handler) {
	r.Route("/admin/auth", func(r chi.Router) {
		r.Post("/assign-role", h.AssignRole)
	})
}
