package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForGroupTool 获取分组规则工具
type getRulesForGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForGroupTool{logger: logger, provider: provider}
}

func (t *getRulesForGroupTool) Name() string {
	return "rostering.rule.get_for_group"
}

func (t *getRulesForGroupTool) Description() string {
	return "Get scheduling rules for a specific group"
}

func (t *getRulesForGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"groupId": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
		},
		"required": []string{"orgId", "groupId"},
	}
}

func (t *getRulesForGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}

	groupID := common.GetString(input, "groupId")
	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}

	rules, err := t.provider.Rule().GetForGroup(ctx, orgID, groupID)
	if err != nil {
		return common.NewExecuteError("Failed to get rules for group", err)
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
