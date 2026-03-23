package conversation

import (
	"context"
	"encoding/json"

	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getWorkflowContextTool 获取工作流上下文工具
type getWorkflowContextTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetWorkflowContextTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getWorkflowContextTool{logger: logger, provider: provider}
}

func (t *getWorkflowContextTool) Name() string {
	return "conversation.get_workflow_context"
}

func (t *getWorkflowContextTool) Description() string {
	return "Get workflow context for a conversation"
}

func (t *getWorkflowContextTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"conversationId": map[string]any{
				"type":        "string",
				"description": "Conversation ID",
			},
		},
		"required": []string{"conversationId"},
	}
}

func (t *getWorkflowContextTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	conversationID := common.GetString(input, "conversationId")
	if conversationID == "" {
		return common.NewValidationError("conversationId is required")
	}

	context, err := t.provider.Conversation().GetWorkflowContext(ctx, conversationID)
	if err != nil {
		t.logger.Error("Failed to get workflow context", "error", err, "conversationID", conversationID)
		return common.NewExecuteError("Failed to get workflow context", err)
	}

	response := map[string]any{
		"context": context,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getWorkflowContextTool)(nil)
