package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRuleConflictsTool 获取规则冲突关系工具（V4）
type getRuleConflictsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRuleConflictsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRuleConflictsTool{logger: logger, provider: provider}
}

func (t *getRuleConflictsTool) Name() string {
	return "rostering.rule.get_conflicts"
}

func (t *getRuleConflictsTool) Description() string {
	return "Get rule conflicts (V4)"
}

func (t *getRuleConflictsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"ruleId": map[string]any{
				"type":        "string",
				"description": "Optional: Rule ID to filter conflicts",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *getRuleConflictsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	ruleID := common.GetString(input, "ruleId")

	// TODO: 调用冲突查询服务
	// 这里需要调用 management-service 的冲突查询 API

	t.logger.Info("Get rule conflicts tool called", "orgId", orgID, "ruleId", ruleID)

	// 模拟返回
	result := map[string]any{
		"conflicts": []map[string]any{},
		"count":     0,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRuleConflictsTool)(nil)
