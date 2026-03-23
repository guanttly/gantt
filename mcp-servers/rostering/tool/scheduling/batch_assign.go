package scheduling

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// batchAssignScheduleTool 批量分配排班工具
type batchAssignScheduleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewBatchAssignScheduleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &batchAssignScheduleTool{logger: logger, provider: provider}
}

func (t *batchAssignScheduleTool) Name() string {
	return "rostering.scheduling.batch_assign"
}

func (t *batchAssignScheduleTool) Description() string {
	return "Batch assign schedules to employees"
}

func (t *batchAssignScheduleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"assignments": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"date": map[string]any{
							"type":        "string",
							"description": "Schedule date (YYYY-MM-DD)",
						},
						"employeeId": map[string]any{
							"type":        "string",
							"description": "Employee ID",
						},
						"shiftId": map[string]any{
							"type":        "string",
							"description": "Shift ID",
						},
					},
					"required": []string{"date", "employeeId", "shiftId"},
				},
				"description": "Array of schedule assignments",
			},
		},
		"required": []string{"orgId", "assignments"},
	}
}

func (t *batchAssignScheduleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	if orgID == "" {
		return common.NewValidationError("Organization ID is required")
	}

	assignmentsRaw, ok := input["assignments"].([]interface{})
	if !ok {
		return common.NewValidationError("Invalid assignments format")
	}

	assignments := make([]*model.ScheduleAssignment, 0, len(assignmentsRaw))
	for _, item := range assignmentsRaw {
		if assMap, ok := item.(map[string]any); ok {
			assignments = append(assignments, &model.ScheduleAssignment{
				Date:       common.GetString(assMap, "date"),
				EmployeeID: common.GetString(assMap, "employeeId"),
				ShiftID:    common.GetString(assMap, "shiftId"),
			})
		}
	}

	req := &model.BatchAssignRequest{
		OrgID:       orgID,
		Assignments: assignments,
	}

	err := t.provider.Scheduling().BatchAssign(ctx, req)
	if err != nil {
		t.logger.Error("Failed to batch assign schedule", "error", err)
		return common.NewExecuteError("Failed to batch assign schedule", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Successfully assigned %d schedules", len(assignments)))},
	}, nil
}

var _ mcp.ITool = (*batchAssignScheduleTool)(nil)
