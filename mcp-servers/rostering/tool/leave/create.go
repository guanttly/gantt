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

// createLeaveTool 创建请假工具
type createLeaveTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateLeaveTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createLeaveTool{logger: logger, provider: provider}
}

func (t *createLeaveTool) Name() string {
	return "rostering.leave.create"
}

func (t *createLeaveTool) Description() string {
	return "Create a leave request for an employee"
}

func (t *createLeaveTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID (optional)",
			},
			"employeeId": map[string]any{
				"type":        "string",
				"description": "Employee ID",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Leave type: annual, sick, personal, maternity, paternity, marriage, bereavement, compensatory, other",
				"enum":        []string{"annual", "sick", "personal", "maternity", "paternity", "marriage", "bereavement", "compensatory", "other"},
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Start date (YYYY-MM-DD)",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "End date (YYYY-MM-DD)",
			},
			"days": map[string]any{
				"type":        "number",
				"description": "Number of days (optional, will be calculated if not provided)",
			},
			"startTime": map[string]any{
				"type":        "string",
				"description": "Start time for partial day leave (HH:mm format, optional)",
			},
			"endTime": map[string]any{
				"type":        "string",
				"description": "End time for partial day leave (HH:mm format, optional)",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Leave reason (optional)",
			},
		},
		"required": []string{"employeeId", "type", "startDate", "endDate"},
	}
}

func (t *createLeaveTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.CreateLeaveRequest{
		OrgID:      common.GetString(input, "orgId"),
		EmployeeID: common.GetString(input, "employeeId"),
		Type:       common.GetString(input, "type"),
		StartDate:  common.GetString(input, "startDate"),
		EndDate:    common.GetString(input, "endDate"),
		Days:       common.GetFloat(input, "days"),
		Reason:     common.GetString(input, "reason"),
	}

	// Handle time strings
	if startTime := common.GetString(input, "startTime"); startTime != "" {
		req.StartTime = &startTime
	}
	if endTime := common.GetString(input, "endTime"); endTime != "" {
		req.EndTime = &endTime
	}

	leave, err := t.provider.Leave().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create leave", "error", err)
		return common.NewExecuteError("Failed to create leave", err)
	}

	data, _ := json.MarshalIndent(leave, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createLeaveTool)(nil)
