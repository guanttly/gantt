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

// QueryStaffTool 查询人员工具
type QueryStaffTool struct {
	BaseRosteringTool
	rosteringService d_service.IRosteringService
}

// NewQueryStaffTool 创建查询人员工具
func NewQueryStaffTool(
	logger logging.ILogger,
	rosteringService d_service.IRosteringService,
) *QueryStaffTool {
	return &QueryStaffTool{
		BaseRosteringTool: BaseRosteringTool{
			name:        "queryStaff",
			description: "查询人员信息，包括资质、可用性、当前排班状态等。",
			logger:      logger,
		},
		rosteringService: rosteringService,
	}
}

// InputSchema 返回输入参数模式
func (t *QueryStaffTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"staffIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "人员ID列表（可选，不提供则查询所有人员）",
			},
			"shiftID": map[string]any{
				"type":        "string",
				"description": "班次ID（可选，查询该班次的人员）",
			},
			"orgID": map[string]any{
				"type":        "string",
				"description": "组织ID（必需）",
			},
		},
		"required": []string{"orgID"},
	}
}

// Execute 执行工具
func (t *QueryStaffTool) Execute(ctx context.Context, arguments map[string]any) (*toolcalling.ToolResult, error) {
	t.logger.Info("QueryStaffTool: Executing", "arguments", arguments)

	orgID, ok := arguments["orgID"].(string)
	if !ok || orgID == "" {
		return t.NewErrorResult(fmt.Errorf("orgID is required")), nil
	}

	var staffList []*d_model.Employee
	var err error

	if shiftID, ok := arguments["shiftID"].(string); ok && shiftID != "" {
		// 查询班次的人员
		staffList, err = t.rosteringService.GetShiftGroupMembers(ctx, shiftID)
	} else if staffIds, ok := arguments["staffIds"].([]any); ok && len(staffIds) > 0 {
		// 批量查询指定人员
		staffIDStrs := make([]string, 0, len(staffIds))
		for _, id := range staffIds {
			if idStr, ok := id.(string); ok {
				staffIDStrs = append(staffIDStrs, idStr)
			}
		}
		
		// 逐个查询（因为接口不支持批量查询多个ID）
		staffList = make([]*d_model.Employee, 0, len(staffIDStrs))
		for _, staffID := range staffIDStrs {
			staff, err := t.rosteringService.GetStaff(ctx, staffID)
			if err != nil {
				t.logger.Warn("QueryStaffTool: Failed to get staff", "staffID", staffID, "error", err)
				continue
			}
			if staff != nil {
				// Staff 是 Employee 的别名，可以直接使用
				staffList = append(staffList, (*d_model.Employee)(staff))
			}
		}
	} else {
		// 查询所有人员：使用 ListStaff 方法
		filter := d_model.StaffListFilter{
			OrgID: orgID,
			Page:  1,
			PageSize: 1000, // 查询足够多的数据
		}
		listResult, err := t.rosteringService.ListStaff(ctx, filter)
		if err != nil {
			t.logger.Error("QueryStaffTool: Failed to list staff", "error", err)
			return t.NewErrorResult(err), nil
		}
		if listResult != nil && listResult.Items != nil {
			// Items 已经是 []*Employee 类型，直接使用
			staffList = listResult.Items
		}
	}

	if err != nil {
		t.logger.Error("QueryStaffTool: Failed to query staff", "error", err)
		return t.NewErrorResult(err), nil
	}

	result := map[string]any{
		"staff": staffList,
		"count": len(staffList),
		"orgID": orgID,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return t.NewErrorResult(fmt.Errorf("序列化结果失败: %w", err)), nil
	}

	return toolcalling.NewJSONResult(string(resultJSON)), nil
}
