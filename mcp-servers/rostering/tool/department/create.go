package department

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// createDepartmentTool 创建部门工具
type createDepartmentTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateDepartmentTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createDepartmentTool{logger: logger, provider: provider}
}

func (t *createDepartmentTool) Name() string {
	return "rostering.department.create"
}

func (t *createDepartmentTool) Description() string {
	return "Create a new department"
}

func (t *createDepartmentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Department code (optional)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Department name",
			},
			"parentId": map[string]any{
				"type":        "string",
				"description": "Parent department ID (optional)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Department description (optional)",
			},
			"managerId": map[string]any{
				"type":        "string",
				"description": "Department manager employee ID (optional)",
			},
			"sortOrder": map[string]any{
				"type":        "number",
				"description": "Sort order (optional)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether department is active (optional)",
			},
		},
		"required": []string{"orgId", "name"},
	}
}

func (t *createDepartmentTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	var parentID *string
	if pid := common.GetString(input, "parentId"); pid != "" {
		parentID = &pid
	}

	var managerID *string
	if mid := common.GetString(input, "managerId"); mid != "" {
		managerID = &mid
	}

	req := &model.CreateDepartmentRequest{
		OrgID:       common.GetString(input, "orgId"),
		Code:        common.GetString(input, "code"),
		Name:        common.GetString(input, "name"),
		ParentID:    parentID,
		Description: common.GetString(input, "description"),
		ManagerID:   managerID,
		SortOrder:   common.GetInt(input, "sortOrder"),
	}

	// Handle isActive boolean
	if isActiveVal, ok := input["isActive"].(bool); ok {
		req.IsActive = isActiveVal
	}

	dept, err := t.provider.Department().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create department", "error", err)
		return common.NewExecuteError("Failed to create department", err)
	}

	data, _ := json.MarshalIndent(dept, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createDepartmentTool)(nil)
