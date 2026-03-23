package service

import (
	"context"

	"jusha/agent/server/context/domain/model"
)

// IConversationService 会话服务接口
type IConversationService interface {
	// CreateConversation 创建新会话
	CreateConversation(ctx context.Context, meta map[string]any) (*model.Conversation, error)

	// AppendMessage 添加消息到会话（支持扩展字段）
	// messageID 为业务层消息唯一标识，如果提供且已存在则跳过保存
	AppendMessage(ctx context.Context, conversationID, messageID, role, content string, metadata map[string]any) (*model.ConversationMessage, error)
	
	// GetMessageIDs 获取会话中已保存的消息ID集合（用于去重）
	GetMessageIDs(ctx context.Context, conversationID string) (map[string]bool, error)

	// GetConversationHistory 获取会话历史消息
	GetConversationHistory(ctx context.Context, conversationID string, limit int) ([]*model.ConversationMessage, error)
	
	// ListConversations 按 Meta 字段查询会话列表
	ListConversations(ctx context.Context, metaFilters map[string]any, limit, offset int) ([]*model.Conversation, int, error)
	
	// UpdateWorkflowContext 更新工作流上下文
	UpdateWorkflowContext(ctx context.Context, conversationID string, context map[string]any) error
	
	// GetWorkflowContext 获取工作流上下文
	GetWorkflowContext(ctx context.Context, conversationID string) (map[string]any, error)
	
	// UpdateConversationMeta 更新会话元数据
	UpdateConversationMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error
}
