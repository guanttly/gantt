package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// batchParseRulesTool 批量解析规则工具（V4）
type batchParseRulesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewBatchParseRulesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &batchParseRulesTool{logger: logger, provider: provider}
}

func (t *batchParseRulesTool) Name() string {
	return "rostering.rule.batch_parse"
}

func (t *batchParseRulesTool) Description() string {
	return "Batch parse multiple natural language rule descriptions into structured rules (V4)"
}

func (t *batchParseRulesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"ruleTexts": map[string]any{
				"type":        "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Array of natural language rule descriptions",
			},
			"shiftNames": map[string]any{
				"type":        "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Optional: List of shift names (for better matching)",
			},
			"groupNames": map[string]any{
				"type":        "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Optional: List of group names (for better matching)",
			},
		},
		"required": []string{"orgId", "ruleTexts"},
	}
}

func (t *batchParseRulesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	ruleTextsRaw, ok := input["ruleTexts"].([]interface{})
	if !ok || len(ruleTextsRaw) == 0 {
		return common.NewValidationError("ruleTexts is required and cannot be empty")
	}

	ruleTexts := make([]string, len(ruleTextsRaw))
	for i, text := range ruleTextsRaw {
		if str, ok := text.(string); ok {
			ruleTexts[i] = str
		}
	}

	shiftNames := []string{}
	if shiftNamesRaw, ok := input["shiftNames"].([]interface{}); ok {
		for _, name := range shiftNamesRaw {
			if str, ok := name.(string); ok {
				shiftNames = append(shiftNames, str)
			}
		}
	}

	groupNames := []string{}
	if groupNamesRaw, ok := input["groupNames"].([]interface{}); ok {
		for _, name := range groupNamesRaw {
			if str, ok := name.(string); ok {
				groupNames = append(groupNames, str)
			}
		}
	}

	// TODO: 调用批量解析服务
	// 这里需要调用 management-service 的批量解析 API

	t.logger.Info("Batch parse rules tool called", "orgId", orgID, "count", len(ruleTexts))

	// 模拟返回
	result := map[string]any{
		"results": []map[string]any{},
		"errors":  []map[string]any{},
		"total":   len(ruleTexts),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*batchParseRulesTool)(nil)
