package rule

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesByObjectShiftTool 根据客体班次获取规则工具 (V4.1)
type getRulesByObjectShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesByObjectShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesByObjectShiftTool{logger: logger, provider: provider}
}

func (t *getRulesByObjectShiftTool) Name() string {
	return "rostering.rule.get_by_object_shift"
}

func (t *getRulesByObjectShiftTool) Description() string {
	return `Get all rules where the specified shift is the OBJECT (target).

In V4.1 rule model:
- Object shift is the one that is AFFECTED by the rule
- For exclusive rules: "A班次排它B班次" - B is the object (cannot be scheduled if A is scheduled)
- For combinable rules: "A班次可与B班次组合" - B is the object

Use this to find what rules affect a specific shift.`
}

func (t *getRulesByObjectShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID to find rules where it is the object",
			},
		},
		"required": []string{"orgId", "shiftId"},
	}
}

func (t *getRulesByObjectShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftID := common.GetString(input, "shiftId")

	if orgID == "" || shiftID == "" {
		return common.NewValidationError("orgId and shiftId are required")
	}

	rules, err := t.provider.Rule().GetRulesByObjectShift(ctx, orgID, shiftID)
	if err != nil {
		t.logger.Error("Failed to get rules by object shift", "error", err)
		return common.NewExecuteError("Failed to get rules", err)
	}

	if len(rules) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.NewTextContent("No rules found where this shift is the object")},
		}, nil
	}

	result, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Found %d rules where shift is the object:\n%s", len(rules), string(result)))},
	}, nil
}
