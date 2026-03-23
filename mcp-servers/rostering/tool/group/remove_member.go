package group

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// removeGroupMemberTool 移除分组成员工具
type removeGroupMemberTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewRemoveGroupMemberTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &removeGroupMemberTool{logger: logger, provider: provider}
}

func (t *removeGroupMemberTool) Name() string {
	return "rostering.group.remove_member"
}

func (t *removeGroupMemberTool) Description() string {
	return "Remove a member from a group"
}

func (t *removeGroupMemberTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"groupId": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee ID to remove",
			},
		},
		"required": []string{"groupId", "employeeId"},
	}
}

func (t *removeGroupMemberTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	groupID := common.GetString(input, "groupId")
	employeeID := common.GetString(input, "employeeId")

	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}
	if employeeID == "" {
		return common.NewValidationError("Employee ID is required")
	}

	req := &model.RemoveGroupMemberRequest{
		GroupID:    groupID,
		EmployeeID: employeeID,
	}

	err := t.provider.Group().RemoveMember(ctx, req)
	if err != nil {
		t.logger.Error("Failed to remove group member", "groupId", groupID, "employeeId", employeeID, "error", err)
		return common.NewExecuteError("Failed to remove group member", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent("Member removed successfully")},
	}, nil
}

var _ mcp.ITool = (*removeGroupMemberTool)(nil)
