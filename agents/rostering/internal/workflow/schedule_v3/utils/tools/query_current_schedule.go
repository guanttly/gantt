package tools

import (
	"context"
	"encoding/json"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/ai/toolcalling"
	"jusha/mcp/pkg/logging"
)

// QueryCurrentScheduleTool 查询当前排班状态工具
type QueryCurrentScheduleTool struct {
	BaseRosteringTool
	rosteringService d_service.IRosteringService
}

// NewQueryCurrentScheduleTool 创建查询当前排班工具
func NewQueryCurrentScheduleTool(
	logger logging.ILogger,
	rosteringService d_service.IRosteringService,
) *QueryCurrentScheduleTool {
	return &QueryCurrentScheduleTool{
		BaseRosteringTool: BaseRosteringTool{
			name:        "queryCurrentSchedule",
			description: "查询当前排班状态，包括已安排的班次、人员分配、可用时间段等。",
			logger:      logger,
		},
		rosteringService: rosteringService,
	}
}

// InputSchema 返回输入参数模式
func (t *QueryCurrentScheduleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"startDate": map[string]any{
				"type":        "string",
				"description": "开始日期（YYYY-MM-DD格式）",
			},
			"endDate": map[string]any{
				"type":        "string",
				"description": "结束日期（YYYY-MM-DD格式）",
			},
			"shiftIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "班次ID列表（可选）",
			},
			"orgID": map[string]any{
				"type":        "string",
				"description": "组织ID（必需）",
			},
		},
		"required": []string{"orgID", "startDate", "endDate"},
	}
}

// Execute 执行工具
func (t *QueryCurrentScheduleTool) Execute(ctx context.Context, arguments map[string]any) (*toolcalling.ToolResult, error) {
	t.logger.Info("QueryCurrentScheduleTool: Executing", "arguments", arguments)

	orgID, ok := arguments["orgID"].(string)
	if !ok || orgID == "" {
		return t.NewErrorResult(fmt.Errorf("orgID is required")), nil
	}

	startDate, ok := arguments["startDate"].(string)
	if !ok || startDate == "" {
		return t.NewErrorResult(fmt.Errorf("startDate is required")), nil
	}

	endDate, ok := arguments["endDate"].(string)
	if !ok || endDate == "" {
		return t.NewErrorResult(fmt.Errorf("endDate is required")), nil
	}

	// 构建查询过滤器
	filter := d_model.ScheduleQueryFilter{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      1,
		PageSize:  1000, // 查询足够多的数据
	}

	// 查询排班数据
	queryResult, err := t.rosteringService.QuerySchedules(ctx, filter)
	if err != nil {
		t.logger.Error("QueryCurrentScheduleTool: Failed to query schedules", "error", err)
		return t.NewErrorResult(fmt.Errorf("查询排班数据失败: %w", err)), nil
	}

	// 如果指定了班次ID，过滤结果
	var filteredSchedules []*d_model.ScheduleAssignment
	if shiftIds, ok := arguments["shiftIds"].([]any); ok && len(shiftIds) > 0 {
		shiftIDMap := make(map[string]bool)
		for _, id := range shiftIds {
			if idStr, ok := id.(string); ok {
				shiftIDMap[idStr] = true
			}
		}
		for _, schedule := range queryResult.Schedules {
			if shiftIDMap[schedule.ShiftID] {
				filteredSchedules = append(filteredSchedules, schedule)
			}
		}
	} else {
		filteredSchedules = queryResult.Schedules
	}

	// 构建结果
	result := map[string]any{
		"orgID":     orgID,
		"startDate": startDate,
		"endDate":   endDate,
		"count":     len(filteredSchedules),
		"total":     queryResult.Total,
		"schedules": filteredSchedules,
	}

	// 转换为JSON字符串
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return t.NewErrorResult(fmt.Errorf("序列化结果失败: %w", err)), nil
	}

	return toolcalling.NewJSONResult(string(resultJSON)), nil
}
