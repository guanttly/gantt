package conversation

import (
	"context"
	"encoding/json"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateMetaTool 更新会话元数据工具
type updateMetaTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateMetaTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateMetaTool{logger: logger, provider: provider}
}

func (t *updateMetaTool) Name() string {
	return "conversation.update_meta"
}

func (t *updateMetaTool) Description() string {
	return "Update conversation metadata (merge update)"
}

func (t *updateMetaTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"conversationId": map[string]any{
				"type":        "string",
				"description": "Conversation ID",
			},
			"metaUpdates": map[string]any{
				"type":        "object",
				"description": "Meta fields to update (supports any type)",
			},
		},
		"required": []string{"conversationId", "metaUpdates"},
	}
}

func (t *updateMetaTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	conversationID := common.GetString(input, "conversationId")
	if conversationID == "" {
		return common.NewValidationError("conversationId is required")
	}

	metaUpdatesRaw, ok := input["metaUpdates"]
	if !ok {
		return common.NewValidationError("metaUpdates is required")
	}

	metaUpdates, ok := metaUpdatesRaw.(map[string]any)
	if !ok {
		return common.NewValidationError("metaUpdates must be an object")
	}

	if err := t.provider.Conversation().UpdateConversationMeta(ctx, conversationID, metaUpdates); err != nil {
		t.logger.Error("Failed to update conversation meta", "error", err, "conversationID", conversationID)
		return common.NewExecuteError("Failed to update conversation meta", err)
	}

	response := map[string]any{
		"success": true,
		"message": "Conversation meta updated successfully",
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateMetaTool)(nil)
