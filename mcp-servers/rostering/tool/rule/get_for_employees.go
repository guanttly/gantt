package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForEmployeesTool 批量获取多个员工相关的规则工具
type getRulesForEmployeesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForEmployeesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForEmployeesTool{logger: logger, provider: provider}
}

func (t *getRulesForEmployeesTool) Name() string {
	return "rostering.rule.get_for_employees"
}

func (t *getRulesForEmployeesTool) Description() string {
	return "Batch get scheduling rules for multiple employees. Returns a map of employeeId -> rules array."
}

func (t *getRulesForEmployeesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"employeeIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of employee IDs",
			},
		},
		"required": []string{"orgId", "employeeIds"},
	}
}

func (t *getRulesForEmployeesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	employeeIDs := common.GetStringArray(input, "employeeIds")

	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}
	if len(employeeIDs) == 0 {
		return common.NewValidationError("employeeIds is required and must not be empty")
	}

	rules, err := t.provider.Rule().GetForEmployees(ctx, orgID, employeeIDs)
	if err != nil {
		t.logger.Error("Failed to get rules for employees", "orgId", orgID, "employeeCount", len(employeeIDs), "error", err)
		return common.NewExecuteError("Failed to get rules for employees", err)
	}

	data, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRulesForEmployeesTool)(nil)

