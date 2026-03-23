package leave

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteLeaveTool 删除请假工具
type deleteLeaveTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteLeaveTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteLeaveTool{logger: logger, provider: provider}
}

func (t *deleteLeaveTool) Name() string {
	return "rostering.leave.delete"
}

func (t *deleteLeaveTool) Description() string {
	return "Delete a leave request"
}

func (t *deleteLeaveTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Leave request ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteLeaveTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Leave request ID is required")
	}

	err := t.provider.Leave().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete leave", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete leave", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Leave request %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteLeaveTool)(nil)
