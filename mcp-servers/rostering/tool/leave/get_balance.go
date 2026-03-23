package leave

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getLeaveBalanceTool 获取请假余额工具
type getLeaveBalanceTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetLeaveBalanceTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getLeaveBalanceTool{logger: logger, provider: provider}
}

func (t *getLeaveBalanceTool) Name() string {
	return "rostering.leave.get_balance"
}

func (t *getLeaveBalanceTool) Description() string {
	return "Get leave balance for an employee"
}

func (t *getLeaveBalanceTool) InputSchema() map[string]any {
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
			"year": map[string]any{
				"type":        "number",
				"description": "Year (default: current year)",
			},
		},
		"required": []string{"orgId", "employeeId"},
	}
}

func (t *getLeaveBalanceTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	employeeID := common.GetString(input, "employeeId")

	if orgID == "" || employeeID == "" {
		return common.NewValidationError("Organization ID and Employee ID are required")
	}

	balances, err := t.provider.Leave().GetBalance(ctx, employeeID)
	if err != nil {
		t.logger.Error("Failed to get leave balance", "orgId", orgID, "employeeId", employeeID, "error", err)
		return common.NewExecuteError("Failed to get leave balance", err)
	}

	data, _ := json.MarshalIndent(balances, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getLeaveBalanceTool)(nil)
