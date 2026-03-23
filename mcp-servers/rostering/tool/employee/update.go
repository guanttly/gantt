package employee

import (
	"context"
	"encoding/json"
	"time"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateEmployeeTool 更新员工工具
type updateEmployeeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateEmployeeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateEmployeeTool{logger: logger, provider: provider}
}

func (t *updateEmployeeTool) Name() string {
	return "rostering.employee.update"
}

func (t *updateEmployeeTool) Description() string {
	return "Update employee information"
}

func (t *updateEmployeeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Employee ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Employee name",
			},
			"phone": map[string]any{
				"type":        "string",
				"description": "Phone number (optional)",
			},
			"email": map[string]any{
				"type":        "string",
				"description": "Email address (optional)",
			},
			"departmentId": map[string]any{
				"type":        "string",
				"description": "Department ID (optional)",
			},
			"position": map[string]any{
				"type":        "string",
				"description": "Position/job title (optional)",
			},
			"role": map[string]any{
				"type":        "string",
				"description": "Role for permissions (optional)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Employee status: active, inactive, leave, suspend, study_leave",
				"enum":        []string{"active", "inactive", "leave", "suspend", "study_leave"},
			},
			"hireDate": map[string]any{
				"type":        "string",
				"description": "Hire date in ISO 8601 format (optional)",
			},
		},
		"required": []string{"id"},
	}
}

func (t *updateEmployeeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Employee ID is required")
	}

	req := &model.UpdateEmployeeRequest{
		OrgID:        common.GetString(input, "orgId"),
		Name:         common.GetString(input, "name"),
		Phone:        common.GetString(input, "phone"),
		Email:        common.GetString(input, "email"),
		DepartmentID: common.GetString(input, "departmentId"),
		Position:     common.GetString(input, "position"),
		Role:         common.GetString(input, "role"),
		Status:       common.GetString(input, "status"),
	}

	// Parse hireDate if provided
	if hireDateStr := common.GetString(input, "hireDate"); hireDateStr != "" {
		if hireDate, err := time.Parse(time.RFC3339, hireDateStr); err == nil {
			req.HireDate = &hireDate
		}
	}

	employee, err := t.provider.Employee().Update(ctx, id, req)
	if err != nil {
		t.logger.Error("Failed to update employee", "id", id, "error", err)
		return common.NewExecuteError("Failed to update employee", err)
	}

	data, _ := json.MarshalIndent(employee, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateEmployeeTool)(nil)
