package conversation

import (
	"context"
	"encoding/json"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listConversationsTool 列出会话工具
type listConversationsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListConversationsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listConversationsTool{logger: logger, provider: provider}
}

func (t *listConversationsTool) Name() string {
	return "conversation.list"
}

func (t *listConversationsTool) Description() string {
	return "List conversations filtered by meta fields"
}

func (t *listConversationsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"metaFilters": map[string]any{
				"type":        "object",
				"description": "Meta field filters (e.g., {\"orgId\": \"xxx\", \"userId\": \"yyy\"})",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of conversations to return",
				"default":     20,
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Offset for pagination",
				"default":     0,
			},
		},
	}
}

func (t *listConversationsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	// 解析 metaFilters
	metaFilters := make(map[string]any)
	if metaFiltersRaw, ok := input["metaFilters"]; ok {
		if metaMap, ok := metaFiltersRaw.(map[string]any); ok {
			metaFilters = metaMap
		}
	}

	limit := common.GetIntWithDefault(input, "limit", 20)
	offset := common.GetIntWithDefault(input, "offset", 0)

	conversations, total, err := t.provider.Conversation().ListConversations(ctx, metaFilters, limit, offset)
	if err != nil {
		t.logger.Error("Failed to list conversations", "error", err)
		return common.NewExecuteError("Failed to list conversations", err)
	}

	// 构建响应
	conversationRecords := make([]map[string]any, 0, len(conversations))
	for _, conv := range conversations {
		record := map[string]any{
			"id":   conv.ID,
			"meta": map[string]any(conv.Meta),
		}
		// 添加 UpdatedAt 和 TotalMessages
		if !conv.UpdatedAt.IsZero() {
			record["updatedAt"] = conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		record["totalMessages"] = conv.TotalMessages
		conversationRecords = append(conversationRecords, record)
	}

	response := map[string]any{
		"conversations": conversationRecords,
		"total":         total,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listConversationsTool)(nil)
