package group

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateGroupTool 更新分组工具
type updateGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateGroupTool{logger: logger, provider: provider}
}

func (t *updateGroupTool) Name() string {
	return "rostering.group.update"
}

func (t *updateGroupTool) Description() string {
	return "Update group information"
}

func (t *updateGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Group ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Group code (optional)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Group name",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Group type: department, team, shift, project, custom",
				"enum":        []string{"department", "team", "shift", "project", "custom"},
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Group description (optional)",
			},
			"parentId": map[string]any{
				"type":        "string",
				"description": "Parent group ID (optional)",
			},
			"leaderId": map[string]any{
				"type":        "string",
				"description": "Group leader employee ID (optional)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Status: active, inactive, archived",
				"enum":        []string{"active", "inactive", "archived"},
			},
		},
		"required": []string{"id"},
	}
}

func (t *updateGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Group ID is required")
	}

	req := &model.UpdateGroupRequest{
		OrgID:       common.GetString(input, "orgId"),
		Code:        common.GetString(input, "code"),
		Name:        common.GetString(input, "name"),
		Type:        common.GetString(input, "type"),
		Description: common.GetString(input, "description"),
		Status:      common.GetString(input, "status"),
	}

	// Handle parentId
	if parentID := common.GetString(input, "parentId"); parentID != "" {
		req.ParentID = &parentID
	}

	// Handle leaderId
	if leaderID := common.GetString(input, "leaderId"); leaderID != "" {
		req.LeaderID = &leaderID
	}

	group, err := t.provider.Group().Update(ctx, id, req)
	if err != nil {
		t.logger.Error("Failed to update group", "id", id, "error", err)
		return common.NewExecuteError("Failed to update group", err)
	}

	data, _ := json.MarshalIndent(group, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateGroupTool)(nil)
