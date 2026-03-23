package rule

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// previewMigrationTool 预览规则迁移工具（V4）
type previewMigrationTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewPreviewMigrationTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &previewMigrationTool{logger: logger, provider: provider}
}

func (t *previewMigrationTool) Name() string {
	return "rostering.rule.preview_migration"
}

func (t *previewMigrationTool) Description() string {
	return "Preview V3 to V4 rule migration (V4)"
}

func (t *previewMigrationTool) InputSchema() map[string]any {
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

func (t *previewMigrationTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}

	// TODO: 调用迁移预览服务
	// 这里需要调用 management-service 的迁移预览 API

	t.logger.Info("Preview migration tool called", "orgId", orgID)

	// 模拟返回
	result := map[string]any{
		"migratableRules":    []map[string]any{},
		"unmigratableRules":  []map[string]any{},
		"totalV3Rules":       0,
		"migratableCount":    0,
		"unmigratableCount":  0,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*previewMigrationTool)(nil)
