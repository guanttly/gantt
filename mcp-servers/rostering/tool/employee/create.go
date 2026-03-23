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

// createEmployeeTool 创建员工工具
type createEmployeeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateEmployeeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createEmployeeTool{logger: logger, provider: provider}
}

func (t *createEmployeeTool) Name() string {
	return "rostering.employee.create"
}

func (t *createEmployeeTool) Description() string {
	return "Create a new employee in the rostering system"
}

func (t *createEmployeeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee work number/ID",
			},
			"userId": map[string]any{
				"type":        "string",
				"description": "Associated user ID (optional)",
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
			"department": map[string]any{
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
		"required": []string{"orgId", "employeeId", "name"},
	}
}

func (t *createEmployeeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.CreateEmployeeRequest{
		OrgID:        common.GetString(input, "orgId"),
		EmployeeID:   common.GetString(input, "employeeId"),
		UserID:       common.GetString(input, "userId"),
		Name:         common.GetString(input, "name"),
		Phone:        common.GetString(input, "phone"),
		Email:        common.GetString(input, "email"),
		DepartmentID: common.GetString(input, "department"),
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

	employee, err := t.provider.Employee().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create employee", "error", err)
		return common.NewExecuteError("Failed to create employee", err)
	}

	data, _ := json.MarshalIndent(employee, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createEmployeeTool)(nil)
