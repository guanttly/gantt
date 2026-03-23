package repository

import (
	"context"

	"jusha/agent/server/context/domain/model"
)

// IConversationRepository 会话仓储接口
type IConversationRepository interface {
	// CreateConversation 创建新会话
	CreateConversation(ctx context.Context, conversation *model.Conversation) error

	// GetConversation 获取会话
	GetConversation(ctx context.Context, id string) (*model.Conversation, error)

	// AppendMessage 添加消息到会话
	AppendMessage(ctx context.Context, message *model.ConversationMessage) error

	// GetMessages 获取会话消息列表
	GetMessages(ctx context.Context, conversationID string, limit int) ([]*model.ConversationMessage, error)
	
	// MessageExists 检查消息是否已存在（通过 messageID）
	MessageExists(ctx context.Context, conversationID, messageID string) (bool, error)
	
	// GetMessageByMessageID 通过 messageID 获取消息
	GetMessageByMessageID(ctx context.Context, conversationID, messageID string) (*model.ConversationMessage, error)
	
	// GetMessageIDs 获取会话中所有已保存的消息ID集合
	GetMessageIDs(ctx context.Context, conversationID string) (map[string]bool, error)

	// ListConversations 列出会话（通过 meta 过滤，支持任意类型）
	ListConversations(ctx context.Context, filters map[string]any, limit, offset int) ([]*model.Conversation, int, error)
	
	// UpdateWorkflowContext 更新工作流上下文
	UpdateWorkflowContext(ctx context.Context, conversationID string, context model.JSONMap) error
	
	// UpdateMeta 更新会话元数据（合并更新）
	UpdateMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error
	
	// IncrementMessageCount 增加消息计数
	IncrementMessageCount(ctx context.Context, conversationID string) error
}
