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

// getScheduleSummaryTool 获取排班统计工具
type getScheduleSummaryTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetScheduleSummaryTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getScheduleSummaryTool{logger: logger, provider: provider}
}

func (t *getScheduleSummaryTool) Name() string {
	return "rostering.scheduling.get_summary"
}

func (t *getScheduleSummaryTool) Description() string {
	return "Get scheduling statistics and summary"
}

func (t *getScheduleSummaryTool) InputSchema() map[string]any {
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
		},
		"required": []string{"orgId", "startDate", "endDate"},
	}
}

func (t *getScheduleSummaryTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	startDate := common.GetString(input, "startDate")
	endDate := common.GetString(input, "endDate")

	req := &model.GetScheduleSummaryRequest{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	summary, err := t.provider.Scheduling().GetSummary(ctx, req)
	if err != nil {
		t.logger.Error("Failed to get schedule summary", "error", err)
		return common.NewExecuteError("Failed to get schedule summary", err)
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getScheduleSummaryTool)(nil)
