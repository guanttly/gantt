package conversation

import (
	"context"
	"encoding/json"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// newConversationTool 创建新会话工具
type newConversationTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewNewConversationTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &newConversationTool{logger: logger, provider: provider}
}

func (t *newConversationTool) Name() string {
	return "conversation.new"
}

func (t *newConversationTool) Description() string {
	return "Create a new conversation"
}

func (t *newConversationTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"meta": map[string]any{
				"type":        "object",
				"description": "Conversation metadata (optional, supports any type)",
			},
		},
	}
}

func (t *newConversationTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	// 解析 meta（支持任意类型）
	meta := make(map[string]any)
	if metaRaw, ok := input["meta"]; ok {
		if metaMap, ok := metaRaw.(map[string]any); ok {
			meta = metaMap
		}
	}

	conversation, err := t.provider.Conversation().CreateConversation(ctx, meta)
	if err != nil {
		t.logger.Error("Failed to create conversation", "error", err)
		return common.NewExecuteError("Failed to create conversation", err)
	}

	response := map[string]any{
		"id":   conversation.ID,
		"meta": map[string]any(conversation.Meta),
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*newConversationTool)(nil)
