package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRuleTool 获取规则详情工具
type getRuleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRuleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRuleTool{logger: logger, provider: provider}
}

func (t *getRuleTool) Name() string {
	return "rostering.rule.get"
}

func (t *getRuleTool) Description() string {
	return "Get scheduling rule details"
}

func (t *getRuleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Rule ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getRuleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Rule ID is required")
	}

	rule, err := t.provider.Rule().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get rule", "id", id, "error", err)
		return common.NewExecuteError("Failed to get rule", err)
	}

	data, _ := json.MarshalIndent(rule, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRuleTool)(nil)
