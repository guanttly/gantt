package shift

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// addShiftGroupTool 添加班次关联分组工具
type addShiftGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewAddShiftGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &addShiftGroupTool{logger: logger, provider: provider}
}

func (t *addShiftGroupTool) Name() string {
	return "rostering.shift.add_group"
}

func (t *addShiftGroupTool) Description() string {
	return "Add a group association to a shift"
}

func (t *addShiftGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"groupId": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Priority (optional, default 0)",
			},
		},
		"required": []string{"shiftId", "groupId"},
	}
}

func (t *addShiftGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}
	groupID := common.GetString(input, "groupId")
	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}
	priority := common.GetIntWithDefault(input, "priority", 0)

	req := &model.AddShiftGroupRequest{
		GroupID:  groupID,
		Priority: priority,
	}

	if err := t.provider.Shift().AddGroup(ctx, shiftID, req); err != nil {
		return common.NewExecuteError("Failed to add group to shift", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: "Group added to shift successfully",
			},
		},
	}, nil
}
