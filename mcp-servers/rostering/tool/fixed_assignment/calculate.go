package fixed_assignment

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// calculateFixedScheduleTool 计算固定排班工具
type calculateFixedScheduleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCalculateFixedScheduleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &calculateFixedScheduleTool{logger: logger, provider: provider}
}

func (t *calculateFixedScheduleTool) Name() string {
	return "rostering.fixed_assignment.calculate"
}

func (t *calculateFixedScheduleTool) Description() string {
	return "Calculate fixed schedule for a shift within a date range. Returns a map of date -> staff IDs."
}

func (t *calculateFixedScheduleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Start date in YYYY-MM-DD format",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "End date in YYYY-MM-DD format",
			},
		},
		"required": []string{"shiftId", "startDate", "endDate"},
	}
}

func (t *calculateFixedScheduleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	startDate := common.GetString(input, "startDate")
	endDate := common.GetString(input, "endDate")

	if shiftID == "" {
		return common.NewValidationError("shiftId is required")
	}
	if startDate == "" {
		return common.NewValidationError("startDate is required")
	}
	if endDate == "" {
		return common.NewValidationError("endDate is required")
	}

	schedule, err := t.provider.FixedAssignment().CalculateFixedSchedule(ctx, shiftID, startDate, endDate)
	if err != nil {
		t.logger.Error("Failed to calculate fixed schedule", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to calculate fixed schedule", err)
	}

	data, _ := json.MarshalIndent(schedule, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*calculateFixedScheduleTool)(nil)
