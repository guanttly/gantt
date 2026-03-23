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

// createShiftTool 创建班次工具
type createShiftTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateShiftTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createShiftTool{logger: logger, provider: provider}
}

func (t *createShiftTool) Name() string {
	return "rostering.shift.create"
}

func (t *createShiftTool) Description() string {
	return "Create a new shift (work schedule template)"
}

func (t *createShiftTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
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
				"description": "Shift type: morning, afternoon, evening, night, full_day, custom",
				"enum":        []string{"morning", "afternoon", "evening", "night", "full_day", "custom"},
			},
			"startTime": map[string]any{
				"type":        "string",
				"description": "Start time (HH:mm format)",
			},
			"endTime": map[string]any{
				"type":        "string",
				"description": "End time (HH:mm format)",
			},
			"duration": map[string]any{
				"type":        "number",
				"description": "Duration in minutes (optional)",
			},
			"isOvernight": map[string]any{
				"type":        "boolean",
				"description": "Whether shift spans overnight (optional)",
			},
			"restDuration": map[string]any{
				"type":        "number",
				"description": "Rest duration in hours (optional)",
			},
			"color": map[string]any{
				"type":        "string",
				"description": "Color code for display (optional)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Description (optional)",
			},
			"priority": map[string]any{
				"type":        "number",
				"description": "Priority (optional)",
			},
			"schedulingPriority": map[string]any{
				"type":        "number",
				"description": "Scheduling priority",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether shift is active (optional)",
			},
		},
		"required": []string{"orgId", "name", "startTime", "endTime"},
	}
}

func (t *createShiftTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.CreateShiftRequest{
		OrgID:              common.GetString(input, "orgId"),
		Code:               common.GetString(input, "code"),
		Name:               common.GetString(input, "name"),
		Type:               common.GetString(input, "type"),
		StartTime:          common.GetString(input, "startTime"),
		EndTime:            common.GetString(input, "endTime"),
		Duration:           common.GetInt(input, "duration"),
		Color:              common.GetString(input, "color"),
		Description:        common.GetString(input, "description"),
		Priority:           common.GetInt(input, "priority"),
		SchedulingPriority: common.GetInt(input, "schedulingPriority"),
		RestDuration:       common.GetFloat(input, "restDuration"),
	}

	// Handle isOvernight boolean
	if isOvernightVal, ok := input["isOvernight"].(bool); ok {
		req.IsOvernight = isOvernightVal
	}

	// Handle isActive boolean
	if isActiveVal, ok := input["isActive"].(bool); ok {
		req.IsActive = isActiveVal
	}

	shift, err := t.provider.Shift().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create shift", "error", err)
		return common.NewExecuteError("Failed to create shift", err)
	}

	data, _ := json.MarshalIndent(shift, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createShiftTool)(nil)
