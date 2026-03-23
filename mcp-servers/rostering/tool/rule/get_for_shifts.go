package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForShiftsTool 批量获取多个班次相关的规则工具
type getRulesForShiftsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForShiftsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForShiftsTool{logger: logger, provider: provider}
}

func (t *getRulesForShiftsTool) Name() string {
	return "rostering.rule.get_for_shifts"
}

func (t *getRulesForShiftsTool) Description() string {
	return "Batch get scheduling rules for multiple shifts. Returns a map of shiftId -> rules array."
}

func (t *getRulesForShiftsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of shift IDs",
			},
		},
		"required": []string{"orgId", "shiftIds"},
	}
}

func (t *getRulesForShiftsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftIDs := common.GetStringArray(input, "shiftIds")

	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}
	if len(shiftIDs) == 0 {
		return common.NewValidationError("shiftIds is required and must not be empty")
	}

	rules, err := t.provider.Rule().GetForShifts(ctx, orgID, shiftIDs)
	if err != nil {
		t.logger.Error("Failed to get rules for shifts", "orgId", orgID, "shiftCount", len(shiftIDs), "error", err)
		return common.NewExecuteError("Failed to get rules for shifts", err)
	}

	data, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRulesForShiftsTool)(nil)

