package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/context/model"
	"jusha/agent/sdk/context/tool"
)

// CreateConversation 创建新会话
func (c *contextClient) CreateConversation(ctx context.Context, req model.ConversationNewRequest) (*model.ConversationNewResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolConversationNew.String(), req)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	var response model.ConversationNewResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal conversation create response: %w", err)
	}

	return &response, nil
}

// AppendMessage 向会话添加消息
func (c *contextClient) AppendMessage(ctx context.Context, req model.ConversationAppendRequest) (*model.ConversationAppendResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolConversationAppend.String(), req)
	if err != nil {
		return nil, fmt.Errorf("append message: %w", err)
	}

	var response model.ConversationAppendResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal conversation append response: %w", err)
	}

	return &response, nil
}

// GetConversationHistory 获取会话历史消息
func (c *contextClient) GetConversationHistory(ctx context.Context, req model.ConversationHistoryRequest) (*model.ConversationHistoryResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolConversationHistory.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get conversation history: %w", err)
	}

	var response model.ConversationHistoryResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal conversation history response: %w", err)
	}

	return &response, nil
}

// ListConversations 按 Meta 字段查询会话列表
func (c *contextClient) ListConversations(ctx context.Context, req model.ConversationListRequest) (*model.ConversationListResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolConversationList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	var response model.ConversationListResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal conversation list response: %w", err)
	}

	return &response, nil
}

// UpdateWorkflowContext 更新工作流上下文
func (c *contextClient) UpdateWorkflowContext(ctx context.Context, req model.WorkflowContextUpdateRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolConversationUpdateWorkflowContext.String(), req)
	if err != nil {
		return fmt.Errorf("update workflow context: %w", err)
	}
	return nil
}

// GetWorkflowContext 获取工作流上下文
func (c *contextClient) GetWorkflowContext(ctx context.Context, req model.WorkflowContextGetRequest) (*model.WorkflowContextGetResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolConversationGetWorkflowContext.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get workflow context: %w", err)
	}

	var response model.WorkflowContextGetResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal workflow context response: %w", err)
	}

	return &response, nil
}

// UpdateConversationMeta 更新会话元数据
func (c *contextClient) UpdateConversationMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error {
	req := model.UpdateConversationMetaRequest{
		ConversationID: conversationID,
		MetaUpdates:    metaUpdates,
	}
	_, err := c.toolBus.Execute(ctx, tool.ToolConversationUpdateMeta.String(), req)
	if err != nil {
		return fmt.Errorf("update conversation meta: %w", err)
	}
	return nil
}
