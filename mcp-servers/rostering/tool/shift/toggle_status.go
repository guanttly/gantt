package shift

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// toggleShiftStatusTool 切换班次状态工具
type toggleShiftStatusTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewToggleShiftStatusTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &toggleShiftStatusTool{logger: logger, provider: provider}
}

func (t *toggleShiftStatusTool) Name() string {
	return "rostering.shift.toggle_status"
}

func (t *toggleShiftStatusTool) Description() string {
	return "Toggle shift status between active and inactive"
}

func (t *toggleShiftStatusTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "New status: active, inactive",
				"enum":        []string{"active", "inactive"},
			},
		},
		"required": []string{"id", "status"},
	}
}

func (t *toggleShiftStatusTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	status := common.GetString(input, "status")

	if id == "" {
		return common.NewValidationError("Shift ID is required")
	}
	if status == "" {
		return common.NewValidationError("Status is required")
	}

	err := t.provider.Shift().ToggleStatus(ctx, id, status)
	if err != nil {
		t.logger.Error("Failed to toggle shift status", "id", id, "status", status, "error", err)
		return common.NewExecuteError("Failed to toggle shift status", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Shift status changed to %s", status))},
	}, nil
}

var _ mcp.ITool = (*toggleShiftStatusTool)(nil)
