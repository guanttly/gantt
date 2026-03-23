package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getWeeklyStaffTool 获取班次周人数配置工具
type getWeeklyStaffTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetWeeklyStaffTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getWeeklyStaffTool{logger: logger, provider: provider}
}

func (t *getWeeklyStaffTool) Name() string {
	return "rostering.shift.get_weekly_staff"
}

func (t *getWeeklyStaffTool) Description() string {
	return "Get the weekly staff count configuration for a shift (Monday to Sunday)"
}

func (t *getWeeklyStaffTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
		},
		"required": []string{"orgId", "shiftId"},
	}
}

func (t *getWeeklyStaffTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftID := common.GetString(input, "shiftId")

	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	config, err := t.provider.Shift().GetWeeklyStaff(ctx, orgID, shiftID)
	if err != nil {
		t.logger.Error("Failed to get weekly staff config", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to get weekly staff configuration", err)
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getWeeklyStaffTool)(nil)
