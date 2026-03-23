package shift

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getShiftGroupMembersTool 获取班次关联分组成员工具
type getShiftGroupMembersTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetShiftGroupMembersTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getShiftGroupMembersTool{logger: logger, provider: provider}
}

func (t *getShiftGroupMembersTool) Name() string {
	return "rostering.shift.get_group_members"
}

func (t *getShiftGroupMembersTool) Description() string {
	return "Get members of all groups associated with a specific shift"
}

func (t *getShiftGroupMembersTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftId": map[string]any{
				"type":        "string",
				"description": "Shift ID",
			},
		},
		"required": []string{"shiftId"},
	}
}

func (t *getShiftGroupMembersTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftID := common.GetString(input, "shiftId")
	if shiftID == "" {
		return common.NewValidationError("Shift ID is required")
	}

	// 直接调用服务层方法，该方法会调用 Management Service 的优化接口
	members, err := t.provider.Shift().GetGroupMembers(ctx, shiftID)
	if err != nil {
		t.logger.Error("Failed to get shift group members", "shiftId", shiftID, "error", err)
		return common.NewExecuteError("Failed to get shift group members", err)
	}

	data, err := json.Marshal(members)
	if err != nil {
		return common.NewExecuteError("Failed to marshal response", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
