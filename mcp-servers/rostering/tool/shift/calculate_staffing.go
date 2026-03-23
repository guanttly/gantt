package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// calculateStaffingTool 计算排班人数工具
type calculateStaffingTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCalculateStaffingTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &calculateStaffingTool{logger: logger, provider: provider}
}

func (t *calculateStaffingTool) Name() string {
	return "rostering.shift.calculate_staffing"
}

func (t *calculateStaffingTool) Description() string {
	return "Calculate recommended staff count for a shift based on modality room weekly volumes and staffing rules. Returns daily recommended staff count for each day of the week."
}

func (t *calculateStaffingTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID to calculate staffing for",
			},
		},
		"required": []string{"orgId", "shiftId"},
	}
}

func (t *calculateStaffingTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftID := common.GetString(input, "shiftId")

	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	preview, err := t.provider.Shift().CalculateStaffing(ctx, orgID, shiftID)
	if err != nil {
		t.logger.Error("Failed to calculate staffing", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to calculate staffing", err)
	}

	data, _ := json.MarshalIndent(preview, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*calculateStaffingTool)(nil)
