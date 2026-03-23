package scheduling

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteScheduleTool 删除排班工具
type deleteScheduleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteScheduleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteScheduleTool{logger: logger, provider: provider}
}

func (t *deleteScheduleTool) Name() string {
	return "rostering.scheduling.delete"
}

func (t *deleteScheduleTool) Description() string {
	return "Delete a schedule record"
}

func (t *deleteScheduleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Schedule ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteScheduleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Schedule ID is required")
	}

	err := t.provider.Scheduling().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete schedule", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete schedule", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Schedule %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteScheduleTool)(nil)
