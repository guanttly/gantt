package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getRuleDependenciesTool 获取规则依赖关系工具（V4）
type getRuleDependenciesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetRuleDependenciesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getRuleDependenciesTool{logger: logger, provider: provider}
}

func (t *getRuleDependenciesTool) Name() string {
	return "rostering.rule.get_dependencies"
}

func (t *getRuleDependenciesTool) Description() string {
	return "Get rule dependencies (V4)"
}

func (t *getRuleDependenciesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"ruleId": map[string]any{
				"type":        "string",
				"description": "Optional: Rule ID to filter dependencies",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *getRuleDependenciesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	ruleID := common.GetString(input, "ruleId")

	// TODO: 调用依赖查询服务
	// 这里需要调用 management-service 的依赖查询 API

	t.logger.Info("Get rule dependencies tool called", "orgId", orgID, "ruleId", ruleID)

	// 模拟返回
	result := map[string]any{
		"dependencies": []map[string]any{},
		"count":        0,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getRuleDependenciesTool)(nil)
