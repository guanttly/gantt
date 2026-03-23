package group

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getGroupTool 获取分组详情工具
type getGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getGroupTool{logger: logger, provider: provider}
}

func (t *getGroupTool) Name() string {
	return "rostering.group.get"
}

func (t *getGroupTool) Description() string {
	return "Get detailed information of a specific group by ID"
}

func (t *getGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Group ID is required")
	}

	group, err := t.provider.Group().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get group", "id", id, "error", err)
		return common.NewExecuteError("Failed to get group", err)
	}

	data, _ := json.MarshalIndent(group, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getGroupTool)(nil)
