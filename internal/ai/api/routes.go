package api

import (
"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册 AI 模块路由。
func RegisterRoutes(r chi.Router, h *Handler) {
r.Route("/ai", func(r chi.Router) {
r.Post("/chat", h.Chat)
r.Post("/parse-rule", h.ParseRule)
r.Get("/quota", h.GetQuota)
r.Get("/usage", h.GetUsage)
})
}
