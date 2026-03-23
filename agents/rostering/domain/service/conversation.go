package service

import (
	"context"

	"jusha/mcp/pkg/workflow/session"
)

// IConversationService 会话管理服务接口
// 提供会话保存、查询、压缩等功能
type IConversationService interface {
	// SaveConversation 保存会话消息到上下文服务
	// sessionID: 当前 session ID
	// messages: 要保存的消息列表（通常为 session 的所有消息）
	SaveConversation(ctx context.Context, sessionID string, messages []session.Message) error

	// GetConversationHistory 获取会话历史消息
	// sessionID: 当前 session ID
	// limit: 限制返回的消息数量，0 表示不限制
	GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]session.Message, error)

	// ListConversations 列出用户的会话列表
	// orgID: 组织ID
	// userID: 用户ID
	// limit: 限制返回的会话数量，0 表示不限制
	ListConversations(ctx context.Context, orgID, userID string, limit int) ([]*ConversationSummary, error)

	// CompressConversation 压缩会话（总结旧消息）
	// sessionID: 当前 session ID
	// 当消息数量超过阈值时，保留最近的消息，将更早的消息总结为摘要
	CompressConversation(ctx context.Context, sessionID string) error

	// LoadConversation 加载指定会话到当前 session
	// sessionID: 当前 session ID
	// conversationID: 要加载的 conversation ID
	LoadConversation(ctx context.Context, sessionID, conversationID string) error
}

// ConversationSummary 会话摘要信息
type ConversationSummary struct {
	ID            string `json:"id"`             // Conversation ID
	Title         string `json:"title"`           // 会话标题（从第一条消息或 meta 中提取）
	LastMessageAt string `json:"lastMessageAt"`  // 最后消息时间
	MessageCount  int    `json:"messageCount"`   // 消息数量
	OrgID         string `json:"orgId"`          // 组织ID
	UserID        string `json:"userId"`         // 用户ID
}
