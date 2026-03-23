package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// updateDepartmentPayload 更新部门的完整载荷（包含ID）
type updateDepartmentPayload struct {
	ID          string  `json:"id"`
	OrgID       string  `json:"orgId"`
	Code        string  `json:"code,omitempty"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId,omitempty"`
	Description string  `json:"description,omitempty"`
	ManagerID   *string `json:"managerId,omitempty"`
	SortOrder   int     `json:"sortOrder,omitempty"`
	IsActive    bool    `json:"isActive,omitempty"`
}

// Department 部门管理实现

func (c *rosteringClient) CreateDepartment(ctx context.Context, req *model.CreateDepartmentRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolDepartmentCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create department: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal department create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateDepartment(ctx context.Context, id string, req *model.UpdateDepartmentRequest) error {
	payload := updateDepartmentPayload{
		ID:          id,
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolDepartmentUpdate.String(), payload)
	if err != nil {
		return fmt.Errorf("update department: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListDepartments(ctx context.Context, orgID string, page, pageSize int) (*model.ListDepartmentsResponse, error) {
	req := map[string]interface{}{
		"orgId":    orgID,
		"page":     page,
		"pageSize": pageSize,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolDepartmentList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}

	var response model.ListDepartmentsResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal department list response: %w", err)
	}

	return &response, nil
}
