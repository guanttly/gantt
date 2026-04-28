package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册 AI 模块路由。
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/ai", func(r chi.Router) {
		r.Post("/chat", h.Chat)
		r.Route("/conversations", func(r chi.Router) {
			r.Get("/", h.ListConversations)
			r.Post("/", h.CreateConversation)
			r.Get("/{conversationID}", h.GetConversation)
			r.Patch("/{conversationID}", h.UpdateConversation)
			r.Delete("/{conversationID}", h.DeleteConversation)
		})
		r.Post("/parse-rule", h.ParseRule)
		r.Post("/parse-rules", h.ParseRulesBatch)
		r.Post("/parse-rules-stream", h.ParseRulesBatchStream)
		r.Get("/quota", h.GetQuota)
		r.Get("/usage", h.GetUsage)
	})
}
