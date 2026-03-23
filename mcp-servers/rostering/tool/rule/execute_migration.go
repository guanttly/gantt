package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// executeMigrationTool 执行规则迁移工具（V4）
type executeMigrationTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewExecuteMigrationTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &executeMigrationTool{logger: logger, provider: provider}
}

func (t *executeMigrationTool) Name() string {
	return "rostering.rule.execute_migration"
}

func (t *executeMigrationTool) Description() string {
	return "Execute V3 to V4 rule migration (V4)"
}

func (t *executeMigrationTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"ruleIds": map[string]any{
				"type":        "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Optional: Specific rule IDs to migrate (if empty, migrate all V3 rules)",
			},
			"dryRun": map[string]any{
				"type":        "boolean",
				"description": "Optional: Dry run mode (preview only, no actual migration)",
			},
		},
		"required": []string{"orgId"},
	}
}

func (t *executeMigrationTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	ruleIDs := []string{}
	if ruleIDsRaw, ok := input["ruleIds"].([]interface{}); ok {
		for _, id := range ruleIDsRaw {
			if str, ok := id.(string); ok {
				ruleIDs = append(ruleIDs, str)
			}
		}
	}

	dryRun := false
	if dryRunVal, ok := input["dryRun"].(bool); ok {
		dryRun = dryRunVal
	}

	// TODO: 调用迁移执行服务
	// 这里需要调用 management-service 的迁移执行 API

	t.logger.Info("Execute migration tool called",
		"orgId", orgID,
		"ruleCount", len(ruleIDs),
		"dryRun", dryRun,
	)

	// 模拟返回
	result := map[string]any{
		"migrationId":    "migration-001",
		"status":         "completed",
		"migratedCount":  0,
		"failedCount":    0,
		"dryRun":         dryRun,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*executeMigrationTool)(nil)
