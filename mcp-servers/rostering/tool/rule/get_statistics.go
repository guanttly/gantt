package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRuleStatisticsTool 获取规则统计信息工具（V4）
type getRuleStatisticsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRuleStatisticsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRuleStatisticsTool{logger: logger, provider: provider}
}

func (t *getRuleStatisticsTool) Name() string {
	return "rostering.rule.get_statistics"
}

func (t *getRuleStatisticsTool) Description() string {
	return "Get rule statistics (V4)"
}

func (t *getRuleStatisticsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *getRuleStatisticsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	// TODO: 调用规则统计服务
	// 这里需要调用 management-service 的规则统计 API

	t.logger.Info("Get rule statistics tool called", "orgId", orgID)

	// 模拟返回
	stats := map[string]any{
		"total":      0,
		"constraint": 0,
		"preference": 0,
		"dependency": 0,
		"v3":         0,
		"v4":         0,
		"active":     0,
		"inactive":   0,
	}

	data, _ := json.MarshalIndent(stats, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRuleStatisticsTool)(nil)
