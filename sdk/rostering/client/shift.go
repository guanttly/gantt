package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// updateShiftPayload 更新班次的完整载荷（包含ID）
type updateShiftPayload struct {
	ID                 string  `json:"id"`
	OrgID              string  `json:"orgId"`
	Name               string  `json:"name"`
	Code               string  `json:"code,omitempty"`
	Type               string  `json:"type,omitempty"`
	Description        string  `json:"description,omitempty"`
	StartTime          string  `json:"startTime"`
	EndTime            string  `json:"endTime"`
	Duration           int     `json:"duration,omitempty"`
	IsOvernight        bool    `json:"isOvernight,omitempty"`
	Color              string  `json:"color,omitempty"`
	Priority           int     `json:"priority,omitempty"`
	SchedulingPriority int     `json:"schedulingPriority"`
	IsActive           bool    `json:"isActive,omitempty"`
	RestDuration       float64 `json:"restDuration,omitempty"`
}

// Shift 班次管理实现

func (c *rosteringClient) CreateShift(ctx context.Context, req *model.CreateShiftRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create shift: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal shift create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateShift(ctx context.Context, id string, req *model.UpdateShiftRequest) error {
	payload := updateShiftPayload{
		ID:                 id,
		OrgID:              req.OrgID,
		Name:               req.Name,
		Code:               req.Code,
		Type:               req.Type,
		Description:        req.Description,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		Duration:           req.Duration,
		IsOvernight:        req.IsOvernight,
		Color:              req.Color,
		Priority:           req.Priority,
		SchedulingPriority: req.SchedulingPriority,
		IsActive:           req.IsActive,
		RestDuration:       req.RestDuration,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolShiftUpdate.String(), payload)
	if err != nil {
		return fmt.Errorf("update shift: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListShifts(ctx context.Context, req *model.ListShiftsRequest) (*model.Page[*model.Shift], error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list shifts: %w", err)
	}

	var response model.Page[*model.Shift]
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal shift list response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) SetShiftGroups(ctx context.Context, req *model.SetShiftGroupsRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolShiftSetGroups.String(), req)
	if err != nil {
		return fmt.Errorf("set shift groups: %w", err)
	}
	return nil
}

func (c *rosteringClient) AddShiftGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error {
	input := map[string]any{
		"shiftId":  shiftID,
		"groupId":  req.GroupID,
		"priority": req.Priority,
	}
	_, err := c.toolBus.Execute(ctx, tool.ToolShiftAddGroup.String(), input)
	if err != nil {
		return fmt.Errorf("add shift group: %w", err)
	}
	return nil
}

func (c *rosteringClient) RemoveShiftGroup(ctx context.Context, shiftID, groupID string) error {
	input := map[string]any{
		"shiftId": shiftID,
		"groupId": groupID,
	}
	_, err := c.toolBus.Execute(ctx, tool.ToolShiftRemoveGroup.String(), input)
	if err != nil {
		return fmt.Errorf("remove shift group: %w", err)
	}
	return nil
}

func (c *rosteringClient) GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error) {
	input := map[string]any{
		"shiftId": shiftID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftGetGroups.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get shift groups: %w", err)
	}

	var groups []*model.ShiftGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("unmarshal shift groups response: %w", err)
	}
	return groups, nil
}

func (c *rosteringClient) GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error) {
	input := map[string]any{
		"shiftId": shiftID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftGetGroupMembers.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get shift group members: %w", err)
	}

	var members []*model.Employee
	if err := json.Unmarshal(result, &members); err != nil {
		return nil, fmt.Errorf("unmarshal shift group members response: %w", err)
	}
	return members, nil
}

func (c *rosteringClient) GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error) {
	input := map[string]any{
		"orgId":   orgID,
		"shiftId": shiftID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftGetWeeklyStaff.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get weekly staff config: %w", err)
	}

	var config model.ShiftWeeklyStaffConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("unmarshal weekly staff config response: %w", err)
	}
	return &config, nil
}

func (c *rosteringClient) SetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string, config []model.WeekdayStaffConfig) error {
	input := map[string]any{
		"orgId":        orgID,
		"shiftId":      shiftID,
		"weeklyConfig": config,
	}
	_, err := c.toolBus.Execute(ctx, tool.ToolShiftSetWeeklyStaff.String(), input)
	if err != nil {
		return fmt.Errorf("set weekly staff config: %w", err)
	}
	return nil
}

func (c *rosteringClient) CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error) {
	input := map[string]any{
		"orgId":   orgID,
		"shiftId": shiftID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolShiftCalculateStaffing.String(), input)
	if err != nil {
		return nil, fmt.Errorf("calculate staffing: %w", err)
	}

	var preview model.StaffingCalculationPreview
	if err := json.Unmarshal(result, &preview); err != nil {
		return nil, fmt.Errorf("unmarshal staffing calculation response: %w", err)
	}
	return &preview, nil
}
