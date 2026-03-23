package employee

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getEmployeeTool 获取员工详情工具
type getEmployeeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetEmployeeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getEmployeeTool{logger: logger, provider: provider}
}

func (t *getEmployeeTool) Name() string {
	return "rostering.employee.get"
}

func (t *getEmployeeTool) Description() string {
	return "Get detailed information of a specific employee by ID"
}

func (t *getEmployeeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Employee ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getEmployeeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Employee ID is required")
	}

	employee, err := t.provider.Employee().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get employee", "id", id, "error", err)
		return common.NewExecuteError("Failed to get employee", err)
	}

	data, _ := json.MarshalIndent(employee, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getEmployeeTool)(nil)
