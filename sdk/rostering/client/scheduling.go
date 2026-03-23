package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// Scheduling 排班管理实现

func (c *rosteringClient) BatchAssignSchedule(ctx context.Context, req model.BatchAssignRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolSchedulingBatchAssign.String(), req)
	if err != nil {
		return fmt.Errorf("batch assign schedule: %w", err)
	}
	return nil
}

func (c *rosteringClient) GetScheduleByDateRange(ctx context.Context, req model.GetScheduleByDateRangeRequest) (*model.ScheduleResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolSchedulingGetByDateRange.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get schedule by date range: %w", err)
	}

	var response model.ScheduleResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal schedule response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) GetScheduleSummary(ctx context.Context, req model.GetScheduleSummaryRequest) (*model.ScheduleSummaryResponse, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolSchedulingGetSummary.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get schedule summary: %w", err)
	}

	var response model.ScheduleSummaryResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal schedule summary response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) DeleteSchedule(ctx context.Context, orgID, employeeID, date string) error {
	req := struct {
		OrgID      string `json:"orgId"`
		EmployeeID string `json:"employeeId"`
		Date       string `json:"date"`
	}{
		OrgID:      orgID,
		EmployeeID: employeeID,
		Date:       date,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolSchedulingDelete.String(), req)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}
