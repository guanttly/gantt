package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRulesForGroupsTool 批量获取多个分组相关的规则工具
type getRulesForGroupsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRulesForGroupsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRulesForGroupsTool{logger: logger, provider: provider}
}

func (t *getRulesForGroupsTool) Name() string {
	return "rostering.rule.get_for_groups"
}

func (t *getRulesForGroupsTool) Description() string {
	return "Batch get scheduling rules for multiple groups. Returns a map of groupId -> rules array."
}

func (t *getRulesForGroupsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"groupIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of group IDs",
			},
		},
		"required": []string{"orgId", "groupIds"},
	}
}

func (t *getRulesForGroupsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	groupIDs := common.GetStringArray(input, "groupIds")

	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}
	if len(groupIDs) == 0 {
		return common.NewValidationError("groupIds is required and must not be empty")
	}

	rules, err := t.provider.Rule().GetForGroups(ctx, orgID, groupIDs)
	if err != nil {
		t.logger.Error("Failed to get rules for groups", "orgId", orgID, "groupCount", len(groupIDs), "error", err)
		return common.NewExecuteError("Failed to get rules for groups", err)
	}

	data, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRulesForGroupsTool)(nil)

