package rule

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// deleteRuleTool 删除规则工具
type deleteRuleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewDeleteRuleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &deleteRuleTool{logger: logger, provider: provider}
}

func (t *deleteRuleTool) Name() string {
	return "rostering.rule.delete"
}

func (t *deleteRuleTool) Description() string {
	return "Delete a scheduling rule"
}

func (t *deleteRuleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Rule ID to delete",
			},
		},
		"required": []string{"id"},
	}
}

func (t *deleteRuleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Rule ID is required")
	}

	err := t.provider.Rule().Delete(ctx, id)
	if err != nil {
		t.logger.Error("Failed to delete rule", "id", id, "error", err)
		return common.NewExecuteError("Failed to delete rule", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Rule %s deleted successfully", id))},
	}, nil
}

var _ mcp.ITool = (*deleteRuleTool)(nil)
