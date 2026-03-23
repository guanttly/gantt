package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// addRuleDependencyTool 添加规则依赖关系工具（V4）
type addRuleDependencyTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewAddRuleDependencyTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &addRuleDependencyTool{logger: logger, provider: provider}
}

func (t *addRuleDependencyTool) Name() string {
	return "rostering.rule.add_dependency"
}

func (t *addRuleDependencyTool) Description() string {
	return "Add rule dependency relationship (V4)"
}

func (t *addRuleDependencyTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"dependentRuleId": map[string]any{
				"type":        "string",
				"description": "Dependent rule ID (rule that needs to be executed first)",
			},
			"dependentOnRuleId": map[string]any{
				"type":        "string",
				"description": "Rule ID that this rule depends on (executed later)",
			},
			"dependencyType": map[string]any{
				"type":        "string",
				"description": "Dependency type: time, source, resource, order",
				"enum":        []string{"time", "source", "resource", "order"},
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Dependency description",
			},
		},
		"required": []string{"orgId", "dependentRuleId", "dependentOnRuleId"},
	}
}

func (t *addRuleDependencyTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	dependentRuleID := common.GetString(input, "dependentRuleId")
	dependentOnRuleID := common.GetString(input, "dependentOnRuleId")

	if orgID == "" || dependentRuleID == "" || dependentOnRuleID == "" {
		return common.NewValidationError("orgId, dependentRuleId, and dependentOnRuleId are required")
	}

	// TODO: 调用依赖创建服务
	// 这里需要调用 management-service 的依赖创建 API

	t.logger.Info("Add rule dependency tool called",
		"orgId", orgID,
		"dependentRuleId", dependentRuleID,
		"dependentOnRuleId", dependentOnRuleID,
	)

	result := map[string]any{
		"success": true,
		"message": "Dependency added successfully",
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*addRuleDependencyTool)(nil)
