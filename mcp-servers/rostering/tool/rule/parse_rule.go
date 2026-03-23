package rule

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// parseRuleTool 解析语义化规则工具（V4）
type parseRuleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewParseRuleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &parseRuleTool{logger: logger, provider: provider}
}

func (t *parseRuleTool) Name() string {
	return "rostering.rule.parse"
}

func (t *parseRuleTool) Description() string {
	return "Parse natural language rule description into structured rules (V4)"
}

func (t *parseRuleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Rule name",
			},
			"ruleDescription": map[string]any{
				"type":        "string",
				"description": "Natural language rule description",
			},
			"applyScope": map[string]any{
				"type":        "string",
				"description": "Apply scope (optional, system can auto-detect)",
			},
			"priority": map[string]any{
				"type":        "number",
				"description": "Priority (default: 5)",
			},
			"validFrom": map[string]any{
				"type":        "string",
				"description": "Valid from date (ISO 8601 format)",
			},
			"validTo": map[string]any{
				"type":        "string",
				"description": "Valid to date (ISO 8601 format)",
			},
		},
		"required": []string{"orgId", "name", "ruleDescription"},
	}
}

func (t *parseRuleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	// TODO: 调用规则解析服务
	// 这里需要调用 management-service 的规则解析 API
	// 由于 MCP 工具通过 provider 访问服务，需要确保 provider 支持规则解析

	t.logger.Info("Parse rule tool called", "orgId", common.GetString(input, "orgId"))

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent("Rule parsing functionality will be implemented through management-service API")},
	}, nil
}

var _ mcp.ITool = (*parseRuleTool)(nil)
