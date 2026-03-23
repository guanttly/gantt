package department

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getDepartmentTool 获取部门详情工具
type getDepartmentTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetDepartmentTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getDepartmentTool{logger: logger, provider: provider}
}

func (t *getDepartmentTool) Name() string {
	return "rostering.department.get"
}

func (t *getDepartmentTool) Description() string {
	return "Get detailed information of a specific department by ID"
}

func (t *getDepartmentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Department ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getDepartmentTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Department ID is required")
	}

	dept, err := t.provider.Department().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get department", "id", id, "error", err)
		return common.NewExecuteError("Failed to get department", err)
	}

	data, _ := json.MarshalIndent(dept, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getDepartmentTool)(nil)
