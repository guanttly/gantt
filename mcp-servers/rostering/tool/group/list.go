package group

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listGroupsTool 查询分组列表工具
type listGroupsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListGroupsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listGroupsTool{logger: logger, provider: provider}
}

func (t *listGroupsTool) Name() string {
	return "rostering.group.list"
}

func (t *listGroupsTool) Description() string {
	return "List groups with optional filters"
}

func (t *listGroupsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"page": map[string]any{
				"type":        "number",
				"description": "Page number (default: 1)",
			},
			"pageSize": map[string]any{
				"type":        "number",
				"description": "Page size (default: 20)",
			},
			"keyword": map[string]any{
				"type":        "string",
				"description": "Search keyword",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Filter by group type",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listGroupsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	page := common.GetIntWithDefault(input, "page", 1)
	pageSize := common.GetIntWithDefault(input, "pageSize", 20)
	keyword := common.GetString(input, "keyword")
	groupType := common.GetString(input, "type")

	req := &model.ListGroupsRequest{
		OrgID:    orgID,
		Type:     groupType,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}

	groups, err := t.provider.Group().GetList(ctx, req)
	if err != nil {
		t.logger.Error("Failed to list groups", "error", err)
		return common.NewExecuteError("Failed to list groups", err)
	}

	result := map[string]any{
		"items":    groups.Groups,
		"total":    groups.TotalCount,
		"page":     page,
		"pageSize": pageSize,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listGroupsTool)(nil)
