package tools

import (
	"context"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/ai/toolcalling"
	"jusha/mcp/pkg/logging"
)

// QueryRulesTool 查询规则工具
type QueryRulesTool struct {
	BaseRosteringTool
	rosteringService d_service.IRosteringService
}

// NewQueryRulesTool 创建查询规则工具
func NewQueryRulesTool(
	logger logging.ILogger,
	rosteringService d_service.IRosteringService,
) *QueryRulesTool {
	return &QueryRulesTool{
		BaseRosteringTool: BaseRosteringTool{
			name:        "queryRules",
			description: "查询排班规则。可以根据规则ID、班次ID、人员ID等条件查询相关规则。",
			logger:      logger,
		},
		rosteringService: rosteringService,
	}
}

// InputSchema 返回输入参数模式
func (t *QueryRulesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ruleIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "规则ID列表（可选）",
			},
			"shiftIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "班次ID列表（可选）",
			},
			"staffIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "人员ID列表（可选）",
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
func (t *QueryRulesTool) Execute(ctx context.Context, arguments map[string]any) (*toolcalling.ToolResult, error) {
	t.logger.Info("QueryRulesTool: Executing", "arguments", arguments)

	orgID, ok := arguments["orgID"].(string)
	if !ok || orgID == "" {
		return t.NewErrorResult(fmt.Errorf("orgID is required")), nil
	}

	var result map[string]any

	// 根据参数查询规则
	if shiftIds, ok := arguments["shiftIds"].([]any); ok && len(shiftIds) > 0 {
		// 查询班次相关规则
		shiftIDStrs := make([]string, 0, len(shiftIds))
		for _, id := range shiftIds {
			if idStr, ok := id.(string); ok {
				shiftIDStrs = append(shiftIDStrs, idStr)
			}
		}
		rulesMap, err := t.rosteringService.GetRulesForShifts(ctx, orgID, shiftIDStrs)
		if err != nil {
			t.logger.Error("QueryRulesTool: Failed to query rules for shifts", "error", err)
			return t.NewErrorResult(err), nil
		}
		// 转换为列表
		allRules := make([]*d_model.Rule, 0)
		for _, rules := range rulesMap {
			allRules = append(allRules, rules...)
		}
		result = map[string]any{
			"rules":     allRules,
			"count":     len(allRules),
			"orgID":     orgID,
			"byShiftID": rulesMap,
		}
	} else if staffIds, ok := arguments["staffIds"].([]any); ok && len(staffIds) > 0 {
		// 查询人员相关规则
		staffIDStrs := make([]string, 0, len(staffIds))
		for _, id := range staffIds {
			if idStr, ok := id.(string); ok {
				staffIDStrs = append(staffIDStrs, idStr)
			}
		}
		rulesMap, err := t.rosteringService.GetRulesForEmployees(ctx, orgID, staffIDStrs)
		if err != nil {
			t.logger.Error("QueryRulesTool: Failed to query rules for employees", "error", err)
			return t.NewErrorResult(err), nil
		}
		// 转换为列表
		allRules := make([]*d_model.Rule, 0)
		for _, rules := range rulesMap {
			allRules = append(allRules, rules...)
		}
		result = map[string]any{
			"rules":        allRules,
			"count":        len(allRules),
			"orgID":        orgID,
			"byEmployeeID": rulesMap,
		}
	} else {
		// 查询所有规则
		rules, listErr := t.rosteringService.ListRules(ctx, d_model.ListRulesRequest{
			OrgID: orgID,
		})
		if listErr != nil {
			t.logger.Error("QueryRulesTool: Failed to list rules", "error", listErr)
			return t.NewErrorResult(listErr), nil
		}
		result = map[string]any{
			"rules": rules,
			"count": len(rules),
			"orgID": orgID,
		}
	}

	jsonResult, jsonErr := t.NewJSONResult(result)
	if jsonErr != nil {
		return t.NewErrorResult(jsonErr), nil
	}
	return jsonResult, nil
}
