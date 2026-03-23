package group

import (
	"context"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// addGroupMemberTool 添加分组成员工具
type addGroupMemberTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewAddGroupMemberTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &addGroupMemberTool{logger: logger, provider: provider}
}

func (t *addGroupMemberTool) Name() string {
	return "rostering.group.add_member"
}

func (t *addGroupMemberTool) Description() string {
	return "Add a member to a group"
}

func (t *addGroupMemberTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"groupId": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee ID to add",
			},
			"role": map[string]any{
				"type":        "string",
				"description": "Member role: member, leader",
				"enum":        []string{"member", "leader"},
			},
		},
		"required": []string{"groupId", "employeeId"},
	}
}

func (t *addGroupMemberTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	groupID := common.GetString(input, "groupId")
	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}

	req := &model.AddGroupMemberRequest{
		GroupID:    groupID,
		EmployeeID: common.GetString(input, "employeeId"),
	}

	err := t.provider.Group().AddMember(ctx, req)
	if err != nil {
		t.logger.Error("Failed to add group member", "groupId", groupID, "error", err)
		return common.NewExecuteError("Failed to add group member", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent("Member added successfully")},
	}, nil
}

var _ mcp.ITool = (*addGroupMemberTool)(nil)
