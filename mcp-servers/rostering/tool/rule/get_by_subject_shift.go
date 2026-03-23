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

// getRulesBySubjectShiftTool 根据主体班次获取规则工具 (V4.1)
type getRulesBySubjectShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesBySubjectShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesBySubjectShiftTool{logger: logger, provider: provider}
}

func (t *getRulesBySubjectShiftTool) Name() string {
	return "rostering.rule.get_by_subject_shift"
}

func (t *getRulesBySubjectShiftTool) Description() string {
	return `Get all rules where the specified shift is the SUBJECT (source).

In V4.1 rule model:
- Subject shift is the one that TRIGGERS the rule
- For exclusive rules: "A班次排它B班次" - A is the subject
- For combinable rules: "A班次可与B班次组合" - A is the subject

Use this to find what rules are triggered when scheduling a specific shift.`
}

func (t *getRulesBySubjectShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID to find rules where it is the subject",
			},
		},
		"required": []string{"orgId", "shiftId"},
	}
}

func (t *getRulesBySubjectShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftID := common.GetString(input, "shiftId")

	if orgID == "" || shiftID == "" {
		return common.NewValidationError("orgId and shiftId are required")
	}

	rules, err := t.provider.Rule().GetRulesBySubjectShift(ctx, orgID, shiftID)
	if err != nil {
		t.logger.Error("Failed to get rules by subject shift", "error", err)
		return common.NewExecuteError("Failed to get rules", err)
	}

	if len(rules) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.NewTextContent("No rules found where this shift is the subject")},
		}, nil
	}

	result, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Found %d rules where shift is the subject:\n%s", len(rules), string(result)))},
	}, nil
}
