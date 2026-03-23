package department

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listDepartmentsTool 查询部门列表工具
type listDepartmentsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListDepartmentsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listDepartmentsTool{logger: logger, provider: provider}
}

func (t *listDepartmentsTool) Name() string {
	return "rostering.department.list"
}

func (t *listDepartmentsTool) Description() string {
	return "List departments with pagination and optional filters"
}

func (t *listDepartmentsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listDepartmentsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")

	deptList, err := t.provider.Department().GetList(ctx, orgID)
	if err != nil {
		t.logger.Error("Failed to list departments", "error", err)
		return common.NewExecuteError("Failed to list departments", err)
	}

	result := map[string]any{
		"items": deptList,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listDepartmentsTool)(nil)
