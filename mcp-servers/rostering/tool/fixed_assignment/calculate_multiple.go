package fixed_assignment

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// calculateMultipleFixedSchedulesTool 批量计算多个班次的固定排班工具
type calculateMultipleFixedSchedulesTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCalculateMultipleFixedSchedulesTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &calculateMultipleFixedSchedulesTool{logger: logger, provider: provider}
}

func (t *calculateMultipleFixedSchedulesTool) Name() string {
	return "rostering.fixed_assignment.calculate_multiple"
}

func (t *calculateMultipleFixedSchedulesTool) Description() string {
	return "Calculate fixed schedules for multiple shifts within a date range. Returns a map of shiftId -> (date -> staff IDs)."
}

func (t *calculateMultipleFixedSchedulesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shiftIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of shift IDs",
			},
			"startDate": map[string]any{
				"type":        "string",
				"description": "Start date in YYYY-MM-DD format",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "End date in YYYY-MM-DD format",
			},
		},
		"required": []string{"shiftIds", "startDate", "endDate"},
	}
}

func (t *calculateMultipleFixedSchedulesTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	shiftIDs := common.GetStringArray(input, "shiftIds")
	startDate := common.GetString(input, "startDate")
	endDate := common.GetString(input, "endDate")

	if len(shiftIDs) == 0 {
		return common.NewValidationError("shiftIds is required and must not be empty")
	}
	if startDate == "" {
		return common.NewValidationError("startDate is required")
	}
	if endDate == "" {
		return common.NewValidationError("endDate is required")
	}

	// 循环调用单个班次的计算API（因为后端没有批量计算API）
	result := make(map[string]map[string][]string)
	for _, shiftID := range shiftIDs {
		schedule, err := t.provider.FixedAssignment().CalculateFixedSchedule(ctx, shiftID, startDate, endDate)
		if err != nil {
			t.logger.Warn("Failed to calculate fixed schedule for shift", "shiftId", shiftID, "error", err)
			// 继续处理其他班次，不中断整个批量操作
			result[shiftID] = make(map[string][]string)
			continue
		}
		result[shiftID] = schedule
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*calculateMultipleFixedSchedulesTool)(nil)

