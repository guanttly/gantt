package shift

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteShiftTool 删除班次工具
type deleteShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteShiftTool{logger: logger, provider: provider}
}

func (t *deleteShiftTool) Name() string {
	return "rostering.shift.delete"
}

func (t *deleteShiftTool) Description() string {
	return "Delete a shift from the system"
}

func (t *deleteShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Shift ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Shift ID is required")
	}

	err := t.provider.Shift().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete shift", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete shift", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Shift %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteShiftTool)(nil)
