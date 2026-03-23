package group

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteGroupTool 删除分组工具
type deleteGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteGroupTool{logger: logger, provider: provider}
}

func (t *deleteGroupTool) Name() string {
	return "rostering.group.delete"
}

func (t *deleteGroupTool) Description() string {
	return "Delete a group from the system"
}

func (t *deleteGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Group ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Group ID is required")
	}

	err := t.provider.Group().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete group", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete group", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Group %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteGroupTool)(nil)
