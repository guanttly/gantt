package leave

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listLeavesTool 查询请假列表工具
type listLeavesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListLeavesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listLeavesTool{logger: logger, provider: provider}
}

func (t *listLeavesTool) Name() string {
	return "rostering.leave.list"
}

func (t *listLeavesTool) Description() string {
	return "List leave requests with filters"
}

func (t *listLeavesTool) InputSchema() map[string]any {
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
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Filter by employee ID",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Filter by status: pending, approved, rejected",
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Filter from date (YYYY-MM-DD)",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "Filter to date (YYYY-MM-DD)",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listLeavesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	page := common.GetIntWithDefault(input, "page", 1)
	pageSize := common.GetIntWithDefault(input, "pageSize", 20)
	employeeID := common.GetString(input, "employeeId")
	status := common.GetString(input, "status")
	startDate := common.GetString(input, "startDate")
	endDate := common.GetString(input, "endDate")

	req := &model.ListLeavesRequest{
		OrgID:      orgID,
		EmployeeID: employeeID,
		Status:     status,
		StartDate:  startDate,
		EndDate:    endDate,
		Page:       page,
		PageSize:   pageSize,
	}

	leaves, err := t.provider.Leave().GetList(ctx, req)
	if err != nil {
		t.logger.Error("Failed to list leaves", "error", err)
		return common.NewExecuteError("Failed to list leaves", err)
	}

	result := map[string]any{
		"items":    leaves.Leaves,
		"total":    leaves.TotalCount,
		"page":     page,
		"pageSize": pageSize,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listLeavesTool)(nil)
