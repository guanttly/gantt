package leave

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateLeaveTool 更新请假工具
type updateLeaveTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateLeaveTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateLeaveTool{logger: logger, provider: provider}
}

func (t *updateLeaveTool) Name() string {
	return "rostering.leave.update"
}

func (t *updateLeaveTool) Description() string {
	return "Update leave request (approve/reject or modify)"
}

func (t *updateLeaveTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Leave request ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee ID",
			},
			"employeeName": map[string]any{
				"type":        "string",
				"description": "Employee name",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Leave type: Annual, Sick, Personal, Maternity, Paternity, Bereavement, Unpaid, Other",
				"enum":        []string{"Annual", "Sick", "Personal", "Maternity", "Paternity", "Bereavement", "Unpaid", "Other"},
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Start date (YYYY-MM-DD)",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "End date (YYYY-MM-DD)",
			},
			"startTime": map[string]any{
				"type":        "string",
				"description": "Start time (ISO 8601 format, optional)",
			},
			"endTime": map[string]any{
				"type":        "string",
				"description": "End time (ISO 8601 format, optional)",
			},
			"days": map[string]any{
				"type":        "number",
				"description": "Number of days",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Leave reason",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Status: Pending, Approved, Rejected, Cancelled",
				"enum":        []string{"Pending", "Approved", "Rejected", "Cancelled"},
			},
			"remark": map[string]any{
				"type":        "string",
				"description": "Approval/rejection remark",
			},
		},
		"required": []string{"id"},
	}
}

func (t *updateLeaveTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Leave request ID is required")
	}

	req := &model.UpdateLeaveRequest{
		OrgID:     common.GetString(input, "orgId"),
		Type:      common.GetString(input, "type"),
		StartDate: common.GetString(input, "startDate"),
		EndDate:   common.GetString(input, "endDate"),
		Days:      common.GetFloat(input, "days"),
		Reason:    common.GetString(input, "reason"),
		Status:    common.GetString(input, "status"),
	}

	// Handle start/end time
	if startTimeStr := common.GetString(input, "startTime"); startTimeStr != "" {
		req.StartTime = &startTimeStr
	}
	if endTimeStr := common.GetString(input, "endTime"); endTimeStr != "" {
		req.EndTime = &endTimeStr
	}

	leave, err := t.provider.Leave().Update(ctx, id, req)
	if err != nil {
		t.logger.Error("Failed to update leave", "id", id, "error", err)
		return common.NewExecuteError("Failed to update leave", err)
	}

	data, _ := json.MarshalIndent(leave, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateLeaveTool)(nil)
