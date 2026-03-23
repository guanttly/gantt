package shift

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// setShiftGroupsTool 设置班次关联分组工具
type setShiftGroupsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewSetShiftGroupsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &setShiftGroupsTool{logger: logger, provider: provider}
}

func (t *setShiftGroupsTool) Name() string {
	return "rostering.shift.set_groups"
}

func (t *setShiftGroupsTool) Description() string {
	return "Set groups associated with a shift"
}

func (t *setShiftGroupsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
			"groupIds": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Array of group IDs",
			},
		},
		"required": []string{"shiftId", "groupIds"},
	}
}

func (t *setShiftGroupsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	groupIDs := common.GetStringArray(input, "groupIds")

	req := &model.SetShiftGroupsRequest{
		GroupIDs: groupIDs,
	}

	err := t.provider.Shift().SetGroups(ctx, shiftID, req)
	if err != nil {
		t.logger.Error("Failed to set shift groups", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to set shift groups", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent("Shift groups updated successfully")},
	}, nil
}

var _ mcp.ITool = (*setShiftGroupsTool)(nil)
