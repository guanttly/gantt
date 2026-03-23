package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// Leave 请假管理实现

func (c *rosteringClient) CreateLeave(ctx context.Context, req model.CreateLeaveRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolLeaveCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create leave: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal leave create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateLeave(ctx context.Context, orgID, leaveID string, req model.UpdateLeaveRequest) error {
	type updateReq struct {
		OrgID   string `json:"orgId"`
		LeaveID string `json:"leaveId"`
		model.UpdateLeaveRequest
	}

	fullReq := updateReq{
		OrgID:              orgID,
		LeaveID:            leaveID,
		UpdateLeaveRequest: req,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolLeaveUpdate.String(), fullReq)
	if err != nil {
		return fmt.Errorf("update leave: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListLeaves(ctx context.Context, req model.ListLeavesRequest) (*model.ListLeavesResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolLeaveList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list leaves: %w", err)
	}

	var response model.ListLeavesResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal leave list response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) GetLeave(ctx context.Context, orgID, leaveID string) (*model.Leave, error) {
	req := struct {
		OrgID   string `json:"orgId"`
		LeaveID string `json:"leaveId"`
	}{
		OrgID:   orgID,
		LeaveID: leaveID,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolLeaveGet.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get leave: %w", err)
	}

	var response struct {
		Leave model.Leave `json:"leave"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal leave get response: %w", err)
	}

	return &response.Leave, nil
}

func (c *rosteringClient) DeleteLeave(ctx context.Context, orgID, leaveID string) error {
	req := struct {
		OrgID   string `json:"orgId"`
		LeaveID string `json:"leaveId"`
	}{
		OrgID:   orgID,
		LeaveID: leaveID,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolLeaveDelete.String(), req)
	if err != nil {
		return fmt.Errorf("delete leave: %w", err)
	}
	return nil
}

func (c *rosteringClient) GetLeaveBalance(ctx context.Context, orgID, employeeID, leaveType string, year int) (*model.LeaveBalance, error) {
	req := struct {
		OrgID      string `json:"orgId"`
		EmployeeID string `json:"employeeId"`
		LeaveType  string `json:"leaveType,omitempty"`
		Year       int    `json:"year,omitempty"`
	}{
		OrgID:      orgID,
		EmployeeID: employeeID,
		LeaveType:  leaveType,
		Year:       year,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolLeaveGetBalance.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get leave balance: %w", err)
	}

	var response model.LeaveBalance
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal leave balance response: %w", err)
	}

	return &response, nil
}
