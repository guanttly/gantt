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

// createGroupTool 创建员工分组工具
type createGroupTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateGroupTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createGroupTool{logger: logger, provider: provider}
}

func (t *createGroupTool) Name() string {
	return "rostering.group.create"
}

func (t *createGroupTool) Description() string {
	return "Create a new employee group"
}

func (t *createGroupTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
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
		"required": []string{"orgId", "name", "type"},
	}
}

func (t *createGroupTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.CreateGroupRequest{
		OrgID:       common.GetString(input, "orgId"),
		Code:        common.GetString(input, "code"),
		Name:        common.GetString(input, "name"),
		Type:        common.GetString(input, "type"),
		Description: common.GetString(input, "description"),
		Status:      common.GetStringWithDefault(input, "status", "active"),
	}

	// Handle parentId
	if parentID := common.GetString(input, "parentId"); parentID != "" {
		req.ParentID = &parentID
	}

	// Handle leaderId
	if leaderID := common.GetString(input, "leaderId"); leaderID != "" {
		req.LeaderID = &leaderID
	}

	group, err := t.provider.Group().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create group", "error", err)
		return common.NewExecuteError("Failed to create group", err)
	}

	data, _ := json.MarshalIndent(group, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createGroupTool)(nil)
