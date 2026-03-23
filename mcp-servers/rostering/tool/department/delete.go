package department

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteDepartmentTool 删除部门工具
type deleteDepartmentTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteDepartmentTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteDepartmentTool{logger: logger, provider: provider}
}

func (t *deleteDepartmentTool) Name() string {
	return "rostering.department.delete"
}

func (t *deleteDepartmentTool) Description() string {
	return "Delete a department from the system"
}

func (t *deleteDepartmentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Department ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteDepartmentTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Department ID is required")
	}

	err := t.provider.Department().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete department", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete department", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Department %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteDepartmentTool)(nil)
