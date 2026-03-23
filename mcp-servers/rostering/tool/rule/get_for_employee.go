package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForEmployeeTool 获取员工相关规则工具
type getRulesForEmployeeTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForEmployeeTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForEmployeeTool{logger: logger, provider: provider}
}

func (t *getRulesForEmployeeTool) Name() string {
	return "rostering.rule.get_for_employee"
}

func (t *getRulesForEmployeeTool) Description() string {
	return "Get all scheduling rules for an employee"
}

func (t *getRulesForEmployeeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee ID",
			},
		},
		"required": []string{"orgId", "employeeId"},
	}
}

func (t *getRulesForEmployeeTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}

	employeeID := common.GetString(input, "employeeId")
	if employeeID == "" {
		return common.NewValidationError("Employee ID is required")
	}

	rules, err := t.provider.Rule().GetForEmployee(ctx, orgID, employeeID)
	if err != nil {
		t.logger.Error("Failed to get rules for employee", "employeeId", employeeID, "error", err)
		return common.NewExecuteError("Failed to get rules for employee", err)
	}

	data, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRulesForEmployeeTool)(nil)
