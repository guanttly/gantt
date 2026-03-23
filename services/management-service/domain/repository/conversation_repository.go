package repository

import (
	"context"
	"jusha/gantt/service/management/domain/service"
)

// IConversationRepository 对话记录仓储接口
type IConversationRepository interface {
	// Create 创建对话记录
	Create(ctx context.Context, conversation *service.ConversationEntity) error

	// Update 更新对话记录
	Update(ctx context.Context, id string, updates map[string]any) error

	// List 查询对话列表
	List(ctx context.Context, filter *service.ScheduleConversationFilter) ([]*service.ConversationEntity, error)

	// Get 获取单个对话
	Get(ctx context.Context, id string) (*service.ConversationEntity, error)

	// GetByConversationID 通过 conversationID 查询
	GetByConversationID(ctx context.Context, conversationID string) (*service.ConversationEntity, error)
}
