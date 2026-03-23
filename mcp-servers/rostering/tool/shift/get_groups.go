package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getShiftGroupsTool 获取班次关联分组工具
type getShiftGroupsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetShiftGroupsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getShiftGroupsTool{logger: logger, provider: provider}
}

func (t *getShiftGroupsTool) Name() string {
	return "rostering.shift.get_groups"
}

func (t *getShiftGroupsTool) Description() string {
	return "Get groups associated with a specific shift"
}

func (t *getShiftGroupsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
		},
		"required": []string{"shiftId"},
	}
}

func (t *getShiftGroupsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	groups, err := t.provider.Shift().GetGroups(ctx, shiftID)
	if err != nil {
		t.logger.Error("Failed to get shift groups", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to get shift groups", err)
	}

	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return common.NewExecuteError("Failed to marshal response", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}
