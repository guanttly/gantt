package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForShiftTool 获取班次规则工具
type getRulesForShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForShiftTool{logger: logger, provider: provider}
}

func (t *getRulesForShiftTool) Name() string {
	return "rostering.rule.get_for_shift"
}

func (t *getRulesForShiftTool) Description() string {
	return "Get scheduling rules for a specific shift"
}

func (t *getRulesForShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
		},
		"required": []string{"orgId", "shiftId"},
	}
}

func (t *getRulesForShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}

	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	rules, err := t.provider.Rule().GetForShift(ctx, orgID, shiftID)
	if err != nil {
		return common.NewExecuteError("Failed to get rules for shift", err)
	}

	data, err := json.Marshal(rules)
	if err != nil {
		return common.NewExecuteError("Failed to marshal rules data", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
