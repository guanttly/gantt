package system_setting

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// setSystemSettingTool 设置系统设置工具
type setSystemSettingTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewSetSystemSettingTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &setSystemSettingTool{logger: logger, provider: provider}
}

func (t *setSystemSettingTool) Name() string {
	return "rostering.system_setting.set"
}

func (t *setSystemSettingTool) Description() string {
	return "Set or update a system setting value by key for an organization."
}

func (t *setSystemSettingTool) InputSchema() map[string]any {
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
			"value": map[string]any{
				"type":        "string",
				"description": "Setting value",
			},
		},
		"required": []string{"orgId", "key", "value"},
	}
}

func (t *setSystemSettingTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	key := common.GetString(input, "key")
	value := common.GetString(input, "value")

	if orgID == "" {
		return common.NewValidationError("orgId is required")
	}
	if key == "" {
		return common.NewValidationError("key is required")
	}
	if value == "" {
		return common.NewValidationError("value is required")
	}

	err := t.provider.SystemSetting().SetSetting(ctx, orgID, key, value)
	if err != nil {
		t.logger.Error("Failed to set system setting", "orgId", orgID, "key", key, "error", err)
		return common.NewExecuteError("Failed to set system setting", err)
	}

	response := map[string]any{
		"key":   key,
		"value": value,
		"message": "Setting updated successfully",
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*setSystemSettingTool)(nil)

