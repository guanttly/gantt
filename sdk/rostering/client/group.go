package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// updateGroupPayload 更新分组的完整载荷（包含ID）
type updateGroupPayload struct {
	ID          string            `json:"id"`
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	Code        string            `json:"code,omitempty"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	ParentID    *string           `json:"parentId,omitempty"`
	LeaderID    *string           `json:"leaderId,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// Group 分组管理实现

func (c *rosteringClient) CreateGroup(ctx context.Context, req *model.CreateGroupRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolGroupCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create group: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal group create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateGroup(ctx context.Context, id string, req *model.UpdateGroupRequest) error {
	payload := updateGroupPayload{
		ID:          id,
		OrgID:       req.OrgID,
		Name:        req.Name,
		Code:        req.Code,
		Type:        req.Type,
		Description: req.Description,
		ParentID:    req.ParentID,
		LeaderID:    req.LeaderID,
		Attributes:  req.Attributes,
		Status:      req.Status,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolGroupUpdate.String(), payload)
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListGroups(ctx context.Context, req *model.ListGroupsRequest) (*model.ListGroupsResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolGroupList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	var response model.ListGroupsResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal group list response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) AddGroupMember(ctx context.Context, req *model.AddGroupMemberRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolGroupAddMember.String(), req)
	if err != nil {
		return fmt.Errorf("add group member: %w", err)
	}
	return nil
}

func (c *rosteringClient) RemoveGroupMember(ctx context.Context, req *model.RemoveGroupMemberRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolGroupRemoveMember.String(), req)
	if err != nil {
		return fmt.Errorf("remove group member: %w", err)
	}
	return nil
}

func (c *rosteringClient) GetGroupMembers(ctx context.Context, groupID string) (*model.GroupMembersResponse, error) {
	input := map[string]any{
		"groupId": groupID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolGroupGetMembers.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}

	var response model.GroupMembersResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal group members response: %w", err)
	}

	return &response, nil
}
