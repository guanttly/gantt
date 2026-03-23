package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/tool"
)

// GetSetting 获取系统设置值
func (c *rosteringClient) GetSetting(ctx context.Context, orgID, key string) (string, error) {
	req := map[string]any{
		"orgId": orgID,
		"key":   key,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolSystemSettingGet.String(), req)
	if err != nil {
		return "", fmt.Errorf("get system setting: %w", err)
	}

	var response struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal system setting response: %w", err)
	}

	return response.Value, nil
}

// SetSetting 设置系统设置值
func (c *rosteringClient) SetSetting(ctx context.Context, orgID, key, value string) error {
	req := map[string]any{
		"orgId": orgID,
		"key":   key,
		"value": value,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolSystemSettingSet.String(), req)
	if err != nil {
		return fmt.Errorf("set system setting: %w", err)
	}

	return nil
}

