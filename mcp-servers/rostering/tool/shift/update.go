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

// updateShiftTool 更新班次工具
type updateShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateShiftTool{logger: logger, provider: provider}
}

func (t *updateShiftTool) Name() string {
	return "rostering.shift.update"
}

func (t *updateShiftTool) Description() string {
	return "Update shift information"
}

func (t *updateShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Shift code (optional)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Shift name",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Shift type: regular, overtime, holiday, rest, flexible (optional)",
				"enum":        []string{"regular", "overtime", "holiday", "rest", "flexible"},
			},
			"startTime": map[string]any{
				"type":        "string",
				"description": "Start time (HH:mm)",
			},
			"endTime": map[string]any{
				"type":        "string",
				"description": "End time (HH:mm)",
			},
			"duration": map[string]any{
				"type":        "number",
				"description": "Shift duration in hours (optional)",
			},
			"isOvernight": map[string]any{
				"type":        "boolean",
				"description": "Whether shift crosses midnight (optional)",
			},
			"restTime": map[string]any{
				"type":        "number",
				"description": "Rest time in hours",
			},
			"color": map[string]any{
				"type":        "string",
				"description": "Color code for display",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Description (optional)",
			},
			"priority": map[string]any{
				"type":        "number",
				"description": "Scheduling priority (optional)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether shift is active (optional)",
			},
			"schedulingPriority": map[string]any{
				"type":        "number",
				"description": "Scheduling priority (deprecated, use priority)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Status (deprecated, use isActive): active, inactive",
			},
		},
		"required": []string{"id"},
	}
}

func (t *updateShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Shift ID is required")
	}

	req := &model.UpdateShiftRequest{
		OrgID:              common.GetString(input, "orgId"),
		Code:               common.GetString(input, "code"),
		Name:               common.GetString(input, "name"),
		Type:               common.GetString(input, "type"),
		StartTime:          common.GetString(input, "startTime"),
		EndTime:            common.GetString(input, "endTime"),
		RestDuration:       common.GetFloat(input, "restTime"),
		Color:              common.GetString(input, "color"),
		Description:        common.GetString(input, "description"),
		SchedulingPriority: common.GetInt(input, "schedulingPriority"),
	}

	// Handle duration
	if duration, ok := input["duration"].(float64); ok {
		req.Duration = int(duration)
	}

	// Handle isOvernight
	if isOvernight, ok := input["isOvernight"].(bool); ok {
		req.IsOvernight = isOvernight
	}

	// Handle priority
	if priority, ok := input["priority"].(float64); ok {
		req.Priority = int(priority)
	}

	// Handle isActive
	if isActive, ok := input["isActive"].(bool); ok {
		req.IsActive = isActive
	}

	shift, err := t.provider.Shift().Update(ctx, id, req)
	if err != nil {
		t.logger.Error("Failed to update shift", "id", id, "error", err)
		return common.NewExecuteError("Failed to update shift", err)
	}

	data, _ := json.MarshalIndent(shift, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateShiftTool)(nil)
