package shift

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// removeShiftGroupTool 移除班次关联分组工具
type removeShiftGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewRemoveShiftGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &removeShiftGroupTool{logger: logger, provider: provider}
}

func (t *removeShiftGroupTool) Name() string {
	return "rostering.shift.remove_group"
}

func (t *removeShiftGroupTool) Description() string {
	return "Remove a group association from a shift"
}

func (t *removeShiftGroupTool) InputSchema() map[string]any {
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
		},
		"required": []string{"shiftId", "groupId"},
	}
}

func (t *removeShiftGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}
	groupID := common.GetString(input, "groupId")
	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}

	if err := t.provider.Shift().RemoveGroup(ctx, shiftID, groupID); err != nil {
		return common.NewExecuteError("Failed to remove group from shift", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: "Group removed from shift successfully",
			},
		},
	}, nil
}
