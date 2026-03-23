package system_setting

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getSystemSettingTool 获取系统设置工具
type getSystemSettingTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetSystemSettingTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getSystemSettingTool{logger: logger, provider: provider}
}

func (t *getSystemSettingTool) Name() string {
	return "rostering.system_setting.get"
}

func (t *getSystemSettingTool) Description() string {
	return "Get a system setting value by key for an organization. Returns the setting value, or default value if not set."
}

func (t *getSystemSettingTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Setting key",
			},
		},
		"required": []string{"orgId", "key"},
	}
}

func (t *getSystemSettingTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	key := common.GetString(input, "key")

	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}
	if key == "" {
		return common.NewValidationError("key is required")
	}

	value, err := t.provider.SystemSetting().GetSetting(ctx, orgID, key)
	if err != nil {
		t.logger.Error("Failed to get system setting", "orgId", orgID, "key", key, "error", err)
		return common.NewExecuteError("Failed to get system setting", err)
	}

	response := map[string]any{
		"key":   key,
		"value": value,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getSystemSettingTool)(nil)

