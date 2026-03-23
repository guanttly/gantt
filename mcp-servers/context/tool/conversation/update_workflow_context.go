package conversation

import (
	"context"
	"encoding/json"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateWorkflowContextTool 更新工作流上下文工具
type updateWorkflowContextTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateWorkflowContextTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateWorkflowContextTool{logger: logger, provider: provider}
}

func (t *updateWorkflowContextTool) Name() string {
	return "conversation.update_workflow_context"
}

func (t *updateWorkflowContextTool) Description() string {
	return "Update workflow context for a conversation"
}

func (t *updateWorkflowContextTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"conversationId": map[string]any{
				"type":        "string",
				"description": "Conversation ID",
			},
			"context": map[string]any{
				"type":        "object",
				"description": "Workflow context (supports any structure)",
			},
		},
		"required": []string{"conversationId", "context"},
	}
}

func (t *updateWorkflowContextTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	conversationID := common.GetString(input, "conversationId")
	if conversationID == "" {
		return common.NewValidationError("conversationId is required")
	}

	contextRaw, ok := input["context"]
	if !ok {
		return common.NewValidationError("context is required")
	}

	contextMap, ok := contextRaw.(map[string]any)
	if !ok {
		return common.NewValidationError("context must be an object")
	}

	if err := t.provider.Conversation().UpdateWorkflowContext(ctx, conversationID, contextMap); err != nil {
		t.logger.Error("Failed to update workflow context", "error", err, "conversationID", conversationID)
		return common.NewExecuteError("Failed to update workflow context", err)
	}

	response := map[string]any{
		"success": true,
		"message": "Workflow context updated successfully",
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateWorkflowContextTool)(nil)
