package conversation

import (
	"context"
	"encoding/json"
	"time"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// appendMessageTool 添加消息工具
type appendMessageTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewAppendMessageTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &appendMessageTool{logger: logger, provider: provider}
}

func (t *appendMessageTool) Name() string {
	return "conversation.append"
}

func (t *appendMessageTool) Description() string {
	return "Append a message to a conversation"
}

func (t *appendMessageTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Conversation ID",
			},
			"messageId": map[string]any{
				"type":        "string",
				"description": "Message ID (business layer unique identifier, optional)",
			},
			"role": map[string]any{
				"type":        "string",
				"description": "Message role: user, assistant, or system",
				"enum":        []string{"user", "assistant", "system"},
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Message content",
			},
			"metadata": map[string]any{
				"type":        "object",
				"description": "Message metadata (optional)",
			},
		},
		"required": []string{"id", "role", "content"},
	}
}

func (t *appendMessageTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	conversationID := common.GetString(input, "id")
	messageID := common.GetString(input, "messageId")
	role := common.GetString(input, "role")
	content := common.GetString(input, "content")

	if conversationID == "" || role == "" || content == "" {
		return common.NewValidationError("id, role, and content are required")
	}

	// 解析 metadata（可选）
	var metadata map[string]any
	if metadataRaw, ok := input["metadata"]; ok {
		if metaMap, ok := metadataRaw.(map[string]any); ok {
			metadata = metaMap
		}
	}

	message, err := t.provider.Conversation().AppendMessage(ctx, conversationID, messageID, role, content, metadata)
	if err != nil {
		t.logger.Error("Failed to append message", "error", err, "conversationID", conversationID, "messageID", messageID)
		return common.NewExecuteError("Failed to append message", err)
	}

	response := map[string]any{
		"role":      message.Role,
		"content":   message.Content,
		"timestamp": message.Timestamp.Format(time.RFC3339),
	}
	if message.MessageID != "" {
		response["messageId"] = message.MessageID
	}
	if message.Metadata != nil {
		response["metadata"] = map[string]any(message.Metadata)
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*appendMessageTool)(nil)
