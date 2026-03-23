package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// setWeeklyStaffTool 设置班次周人数配置工具
type setWeeklyStaffTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewSetWeeklyStaffTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &setWeeklyStaffTool{logger: logger, provider: provider}
}

func (t *setWeeklyStaffTool) Name() string {
	return "rostering.shift.set_weekly_staff"
}

func (t *setWeeklyStaffTool) Description() string {
	return "Set the weekly staff count configuration for a shift (Monday to Sunday). Each day can have a different staff count."
}

func (t *setWeeklyStaffTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"weeklyConfig": map[string]any{
				"type":        "array",
				"description": "Weekly staff configuration array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"weekday": map[string]any{
							"type":        "number",
							"description": "Day of week: 0=Sunday, 1=Monday, ..., 6=Saturday",
						},
						"staffCount": map[string]any{
							"type":        "number",
							"description": "Number of staff required for this day",
						},
					},
					"required": []string{"weekday", "staffCount"},
				},
			},
		},
		"required": []string{"orgId", "shiftId", "weeklyConfig"},
	}
}

func (t *setWeeklyStaffTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	shiftID := common.GetString(input, "shiftId")

	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	// 解析 weeklyConfig
	weeklyConfigAny, ok := input["weeklyConfig"]
	if !ok {
		return common.NewValidationError("Weekly configuration is required")
	}

	weeklyConfigJSON, err := json.Marshal(weeklyConfigAny)
	if err != nil {
		return common.NewValidationError("Invalid weekly configuration format")
	}

	var weeklyConfig []struct {
		Weekday    int `json:"weekday"`
		StaffCount int `json:"staffCount"`
	}
	if err := json.Unmarshal(weeklyConfigJSON, &weeklyConfig); err != nil {
		return common.NewValidationError("Invalid weekly configuration format")
	}

	req := &model.SetShiftWeeklyStaffRequest{
		WeeklyConfig: weeklyConfig,
	}

	if err := t.provider.Shift().SetWeeklyStaff(ctx, orgID, shiftID, req); err != nil {
		t.logger.Error("Failed to set weekly staff config", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to set weekly staff configuration", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent("Weekly staff configuration updated successfully")},
	}, nil
}

var _ mcp.ITool = (*setWeeklyStaffTool)(nil)
