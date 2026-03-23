package group

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getGroupMembersTool 获取分组成员工具
type getGroupMembersTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetGroupMembersTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getGroupMembersTool{logger: logger, provider: provider}
}

func (t *getGroupMembersTool) Name() string {
	return "rostering.group.get_members"
}

func (t *getGroupMembersTool) Description() string {
	return "Get all members of a group"
}

func (t *getGroupMembersTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"groupId": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
		},
		"required": []string{"groupId"},
	}
}

func (t *getGroupMembersTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	groupID := common.GetString(input, "groupId")
	if groupID == "" {
		return common.NewValidationError("Group ID is required")
	}

	members, err := t.provider.Group().GetMembers(ctx, groupID)
	if err != nil {
		t.logger.Error("Failed to get group members", "groupId", groupID, "error", err)
		return common.NewExecuteError("Failed to get group members", err)
	}

	data, _ := json.MarshalIndent(members, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getGroupMembersTool)(nil)
