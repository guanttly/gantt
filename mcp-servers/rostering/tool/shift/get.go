package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getShiftTool 获取班次详情工具
type getShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getShiftTool{logger: logger, provider: provider}
}

func (t *getShiftTool) Name() string {
	return "rostering.shift.get"
}

func (t *getShiftTool) Description() string {
	return "Get detailed information of a specific shift by ID"
}

func (t *getShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Shift ID is required")
	}

	shift, err := t.provider.Shift().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get shift", "id", id, "error", err)
		return common.NewExecuteError("Failed to get shift", err)
	}

	data, _ := json.MarshalIndent(shift, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getShiftTool)(nil)
