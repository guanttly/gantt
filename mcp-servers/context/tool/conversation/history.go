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

// getHistoryTool 获取会话历史工具
type getHistoryTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetHistoryTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getHistoryTool{logger: logger, provider: provider}
}

func (t *getHistoryTool) Name() string {
	return "conversation.history"
}

func (t *getHistoryTool) Description() string {
	return "Get conversation history messages"
}

func (t *getHistoryTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Conversation ID",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Limit the number of messages to return (0 means no limit)",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getHistoryTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	conversationID := common.GetString(input, "id")
	limit := common.GetIntWithDefault(input, "limit", 0)

	if conversationID == "" {
		return common.NewValidationError("id is required")
	}

	messages, err := t.provider.Conversation().GetConversationHistory(ctx, conversationID, limit)
	if err != nil {
		t.logger.Error("Failed to get conversation history", "error", err, "conversationID", conversationID)
		return common.NewExecuteError("Failed to get conversation history", err)
	}

	// 转换为响应格式
	messageList := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		msgMap := map[string]any{
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": msg.Timestamp.Format(time.RFC3339),
		}
		if msg.MessageID != "" {
			msgMap["messageId"] = msg.MessageID
		}
		// 包含 Metadata（可能包含 Actions）
		if len(msg.Metadata) > 0 {
			msgMap["metadata"] = map[string]any(msg.Metadata)
		}
		messageList = append(messageList, msgMap)
	}

	response := map[string]any{
		"messages": messageList,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getHistoryTool)(nil)
