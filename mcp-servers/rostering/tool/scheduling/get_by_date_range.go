package scheduling

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getScheduleByDateRangeTool 按日期范围查询排班工具
type getScheduleByDateRangeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetScheduleByDateRangeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getScheduleByDateRangeTool{logger: logger, provider: provider}
}

func (t *getScheduleByDateRangeTool) Name() string {
	return "rostering.scheduling.get_by_date_range"
}

func (t *getScheduleByDateRangeTool) Description() string {
	return "Get schedules by date range with optional filters"
}

func (t *getScheduleByDateRangeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Start date (YYYY-MM-DD)",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "End date (YYYY-MM-DD)",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Filter by employee ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Filter by shift ID",
			},
		},
		"required": []string{"orgId", "startDate", "endDate"},
	}
}

func (t *getScheduleByDateRangeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	startDate := common.GetString(input, "startDate")
	endDate := common.GetString(input, "endDate")

	req := &model.GetScheduleByDateRangeRequest{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if orgID == "" || startDate == "" || endDate == "" {
		return common.NewValidationError("orgId, startDate, and endDate are required")
	}

	records, err := t.provider.Scheduling().GetByDateRange(ctx, req)
	if err != nil {
		t.logger.Error("Failed to get schedule", "error", err)
		return common.NewExecuteError("Failed to get schedule", err)
	}

	data, _ := json.MarshalIndent(records, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getScheduleByDateRangeTool)(nil)
