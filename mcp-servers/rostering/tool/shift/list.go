package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listShiftsTool 查询班次列表工具
type listShiftsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListShiftsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listShiftsTool{logger: logger, provider: provider}
}

func (t *listShiftsTool) Name() string {
	return "rostering.shift.list"
}

func (t *listShiftsTool) Description() string {
	return "List shifts with optional filters"
}

func (t *listShiftsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"page": map[string]any{
				"type":        "number",
				"description": "Page number (default: 1)",
			},
			"pageSize": map[string]any{
				"type":        "number",
				"description": "Page size (default: 20)",
			},
			"keyword": map[string]any{
				"type":        "string",
				"description": "Search keyword",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Filter by status",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listShiftsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.ListShiftsRequest{
		OrgID:    common.GetString(input, "orgId"),
		Page:     common.GetIntWithDefault(input, "page", 1),
		PageSize: common.GetIntWithDefault(input, "pageSize", 20),
		Keyword:  common.GetString(input, "keyword"),
		Status:   common.GetString(input, "status"),
	}

	response, err := t.provider.Shift().GetList(ctx, req)
	if err != nil {
		t.logger.Error("Failed to list shifts", "error", err)
		return common.NewExecuteError("Failed to list shifts", err)
	}

	result := map[string]any{
		"items":    response.Shifts,
		"total":    response.TotalCount,
		"page":     req.Page,
		"pageSize": req.PageSize,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listShiftsTool)(nil)
