package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// listRulesTool 查询规则列表工具
type listRulesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewListRulesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &listRulesTool{logger: logger, provider: provider}
}

func (t *listRulesTool) Name() string {
	return "rostering.rule.list"
}

func (t *listRulesTool) Description() string {
	return "List scheduling rules"
}

func (t *listRulesTool) InputSchema() map[string]any {
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
			"type": map[string]any{
				"type":        "string",
				"description": "Filter by rule type",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Filter by status",
			},
			"applyScope": map[string]any{
				"type":        "string",
				"description": "Filter by apply scope (e.g. shift ID, group ID)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Filter by active status",
			},
			"keyword": map[string]any{
				"type":        "string",
				"description": "Search keyword",
			},
			// V4新增筛选字段
			"category": map[string]any{
				"type":        "string",
				"description": "Filter by category: constraint, preference, dependency",
				"enum":        []string{"constraint", "preference", "dependency"},
			},
			"subCategory": map[string]any{
				"type":        "string",
				"description": "Filter by sub-category",
			},
			"sourceType": map[string]any{
				"type":        "string",
				"description": "Filter by source type: manual, llm_parsed, migrated",
				"enum":        []string{"manual", "llm_parsed", "migrated"},
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Filter by version: v3, v4",
				"enum":        []string{"v3", "v4"},
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *listRulesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	page := common.GetIntWithDefault(input, "page", 1)
	pageSize := common.GetIntWithDefault(input, "pageSize", 20)
	ruleType := common.GetString(input, "type")
	status := common.GetString(input, "status")
	applyScope := common.GetString(input, "applyScope")
	keyword := common.GetString(input, "keyword")

	var isActive *bool
	if val, ok := input["isActive"]; ok {
		if v, ok := val.(bool); ok {
			isActive = &v
		}
	}

	req := &model.ListRulesRequest{
		OrgID:      orgID,
		RuleType:   ruleType,
		Status:     status,
		ApplyScope: applyScope,
		IsActive:   isActive,
		Keyword:    keyword,
		// V4新增筛选字段
		Category:    common.GetString(input, "category"),
		SubCategory: common.GetString(input, "subCategory"),
		SourceType:  common.GetString(input, "sourceType"),
		Version:     common.GetString(input, "version"),
		Page:        page,
		PageSize:    pageSize,
	}

	rules, err := t.provider.Rule().GetList(ctx, req)
	if err != nil {
		t.logger.Error("Failed to list rules", "error", err)
		return common.NewExecuteError("Failed to list rules", err)
	}

	result := map[string]any{
		"items":    rules.Rules,
		"total":    rules.TotalCount,
		"page":     page,
		"pageSize": pageSize,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*listRulesTool)(nil)
