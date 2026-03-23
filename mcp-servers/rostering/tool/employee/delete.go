package employee

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteEmployeeTool 删除员工工具
type deleteEmployeeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteEmployeeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteEmployeeTool{logger: logger, provider: provider}
}

func (t *deleteEmployeeTool) Name() string {
	return "rostering.employee.delete"
}

func (t *deleteEmployeeTool) Description() string {
	return "Delete an employee from the system"
}

func (t *deleteEmployeeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Employee ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteEmployeeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Employee ID is required")
	}

	err := t.provider.Employee().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete employee", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete employee", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Employee %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteEmployeeTool)(nil)
