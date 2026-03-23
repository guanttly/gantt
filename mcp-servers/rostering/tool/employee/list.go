package employee

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listEmployeesTool 查询员工列表工具
type listEmployeesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListEmployeesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listEmployeesTool{logger: logger, provider: provider}
}

func (t *listEmployeesTool) Name() string {
	return "rostering.employee.list"
}

func (t *listEmployeesTool) Description() string {
	return "List employees with optional filters (keyword, department, status). Supports pagination."
}

func (t *listEmployeesTool) InputSchema() map[string]any {
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
				"description": "Search keyword (name, employeeId, phone, email)",
			},
			"departmentId": map[string]any{
				"type":        "string",
				"description": "Filter by department ID",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Filter by status: active, inactive, on_leave",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listEmployeesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.ListEmployeesRequest{
		OrgID:        common.GetString(input, "orgId"),
		Page:         common.GetIntWithDefault(input, "page", 1),
		PageSize:     common.GetIntWithDefault(input, "pageSize", 20),
		Keyword:      common.GetString(input, "keyword"),
		DepartmentID: common.GetString(input, "departmentId"),
		Status:       common.GetString(input, "status"),
	}

	employees, err := t.provider.Employee().GetList(ctx, req)
	if err != nil {
		t.logger.Error("Failed to list employees", "error", err)
		return common.NewExecuteError("Failed to list employees", err)
	}

	result := map[string]any{
		"items":    employees.Employees,
		"total":    employees.TotalCount,
		"page":     req.Page,
		"pageSize": req.PageSize,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listEmployeesTool)(nil)
