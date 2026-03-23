package domain

import (
	"context"

	"jusha/agent/sdk/context/model"
)

// IContextClient 上下文服务客户端接口
// 提供通用的上下文服务调用能力，可被多个 agent 复用
type IContextClient interface {
	// IConversationService 会话管理服务
	IConversationService
}

// IConversationService 会话管理服务接口
type IConversationService interface {
	// 基础方法
	// CreateConversation 创建新会话
	CreateConversation(ctx context.Context, req model.ConversationNewRequest) (*model.ConversationNewResponse, error)

	// AppendMessage 向会话添加消息
	AppendMessage(ctx context.Context, req model.ConversationAppendRequest) (*model.ConversationAppendResponse, error)

	// GetConversationHistory 获取会话历史消息
	GetConversationHistory(ctx context.Context, req model.ConversationHistoryRequest) (*model.ConversationHistoryResponse, error)
	
	// 查询方法
	// ListConversations 按 Meta 字段查询会话列表
	ListConversations(ctx context.Context, req model.ConversationListRequest) (*model.ConversationListResponse, error)
	
	// Workflow Context 管理
	// UpdateWorkflowContext 更新工作流上下文
	UpdateWorkflowContext(ctx context.Context, req model.WorkflowContextUpdateRequest) error
	
	// GetWorkflowContext 获取工作流上下文
	GetWorkflowContext(ctx context.Context, req model.WorkflowContextGetRequest) (*model.WorkflowContextGetResponse, error)
	
	// 更新 Meta（用于在排班过程中更新排班标识）
	// UpdateConversationMeta 更新会话元数据
	UpdateConversationMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error
}
