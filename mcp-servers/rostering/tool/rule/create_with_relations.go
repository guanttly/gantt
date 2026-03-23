package rule

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// createRuleWithRelationsTool 创建带班次关系的规则工具 (V4.1)
type createRuleWithRelationsTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateRuleWithRelationsTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createRuleWithRelationsTool{logger: logger, provider: provider}
}

func (t *createRuleWithRelationsTool) Name() string {
	return "rostering.rule.create_with_relations"
}

func (t *createRuleWithRelationsTool) Description() string {
	return `Create a scheduling rule with shift relations (V4.1).

This tool creates rules with structured shift relationships:
- For EXCLUSIVE rules: specify subject shifts (source) and object shifts (target to exclude)
- For COMBINABLE rules: specify subject and object shifts that can be combined
- For REQUIRED_TOGETHER rules: specify shifts that must be scheduled together
- For MAX_COUNT/PERIODIC rules: specify target shifts to limit

Shift relation roles:
- subject: The source shift(s) that trigger the rule
- object: The target shift(s) that are affected by the rule
- target: For single-target rules (maxCount, periodic)

Apply scopes:
- all: Apply to all employees (default)
- employee: Apply to specific employees
- group: Apply to specific groups
- exclude_group: Exclude specific groups

Time offsets (TimeOffsetDays):
- For cross-day dependencies (e.g., "must be the person who worked shift B yesterday"), use timeOffsetDays instead of just timeScope.
- -1 means yesterday, -2 means the day before yesterday, 1 means tomorrow, 7 means same day next week.
- If no cross-day dependency exists, leave it empty or 0.`
}

func (t *createRuleWithRelationsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Rule name",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Rule description",
			},
			"ruleType": map[string]any{
				"type":        "string",
				"description": "Rule type: exclusive, combinable, required_together, max_count, consecutive_max, min_rest, interval, periodic, custom",
				"enum":        []string{"exclusive", "combinable", "required_together", "max_count", "consecutive_max", "min_rest", "interval", "periodic", "custom"},
			},
			"timeScope": map[string]any{
				"type":        "string",
				"description": "Time scope: same_day, same_week, same_month, custom",
				"enum":        []string{"same_day", "same_week", "same_month", "custom"},
			},
			"timeOffsetDays": map[string]any{
				"type":        "integer",
				"description": "Time offset in days for cross-day dependencies (e.g. -1 for yesterday, 1 for tomorrow, -7 for last week). Essential for rules like 'must be the person who worked shift B yesterday'.",
			},
			"associations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"associationType": map[string]any{
							"type":        "string",
							"description": "Association type: shift, employee, group",
							"enum":        []string{"shift", "employee", "group"},
						},
						"associationId": map[string]any{
							"type":        "string",
							"description": "Associated ID",
						},
						"role": map[string]any{
							"type":        "string",
							"description": "Role in the rule: subject, object, target",
							"enum":        []string{"subject", "object", "target"},
						},
					},
					"required": []string{"associationType", "associationId"},
				},
				"description": "Associations defining the rule structure",
			},
			"applyScopes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"scopeType": map[string]any{
							"type":        "string",
							"description": "Scope type: all, employee, group, exclude_employee, exclude_group",
							"enum":        []string{"all", "employee", "group", "exclude_employee", "exclude_group"},
						},
						"scopeId": map[string]any{
							"type":        "string",
							"description": "ID of employee or group (not needed for 'all')",
						},
						"scopeName": map[string]any{
							"type":        "string",
							"description": "Name for display",
						},
					},
					"required": []string{"scopeType"},
				},
				"description": "Apply scopes defining who this rule applies to",
			},
			"maxCount": map[string]any{
				"type":        "integer",
				"description": "Maximum count (for max_count rule type)",
			},
			"consecutiveMax": map[string]any{
				"type":        "integer",
				"description": "Maximum consecutive days (for consecutive_max rule type)",
			},
			"intervalDays": map[string]any{
				"type":        "integer",
				"description": "Interval days (for interval rule type)",
			},
			"minRestDays": map[string]any{
				"type":        "integer",
				"description": "Minimum rest days (for min_rest rule type)",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Rule priority (higher = more important)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether the rule is active",
			},
		},
		"required": []string{"orgId", "name", "ruleType"},
	}
}

func (t *createRuleWithRelationsTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	orgID := common.GetString(input, "orgId")
	name := common.GetString(input, "name")
	ruleType := common.GetString(input, "ruleType")

	if orgID == "" || name == "" || ruleType == "" {
		return common.NewValidationError("orgId, name and ruleType are required")
	}

	// 构建请求
	req := &model.CreateRuleWithRelationsRequest{
		CreateRuleRequest: model.CreateRuleRequest{
			OrgID:       orgID,
			Name:        name,
			Description: common.GetString(input, "description"),
			RuleType:    ruleType,
			TimeScope:   common.GetString(input, "timeScope"),
			Priority:    common.GetInt(input, "priority"),
			IsActive:    true,
			Version:     "v4",
			SourceType:  "manual",
		},
	}

	// 设置时间范围默认值
	if req.TimeScope == "" {
		req.TimeScope = "same_day"
	}

	// 处理时间偏移量
	if input["timeOffsetDays"] != nil {
		offset := common.GetInt(input, "timeOffsetDays")
		req.TimeOffsetDays = &offset
	}

	// 处理数值参数
	if v := common.GetInt(input, "maxCount"); v > 0 {
		req.MaxCount = &v
	}
	if v := common.GetInt(input, "consecutiveMax"); v > 0 {
		req.ConsecutiveMax = &v
	}
	if v := common.GetInt(input, "intervalDays"); v > 0 {
		req.IntervalDays = &v
	}
	if v := common.GetInt(input, "minRestDays"); v > 0 {
		req.MinRestDays = &v
	}

	// 解析关联关系
	var parsedAssociations []model.RuleAssociation
	if relations, ok := input["associations"].([]interface{}); ok {
		for _, item := range relations {
			if relMap, ok := item.(map[string]any); ok {
				parsedAssociations = append(parsedAssociations, model.RuleAssociation{
					AssociationType: common.GetString(relMap, "associationType"),
					AssociationID:   common.GetString(relMap, "associationId"),
					Role:            common.GetString(relMap, "role"),
				})
			}
		}
	}
	req.Associations = parsedAssociations

	// 解析适用范围
	if scopes, ok := input["applyScopes"].([]interface{}); ok {
		for _, item := range scopes {
			if scopeMap, ok := item.(map[string]any); ok {
				req.ApplyScopes = append(req.ApplyScopes, model.ApplyScope{
					ScopeType: common.GetString(scopeMap, "scopeType"),
					ScopeID:   common.GetString(scopeMap, "scopeId"),
					ScopeName: common.GetString(scopeMap, "scopeName"),
				})
			}
		}
	}

	// 如果没有指定适用范围，默认全局
	if len(req.ApplyScopes) == 0 {
		req.ApplyScopes = []model.ApplyScope{{ScopeType: "all"}}
	}

	// 设置ApplyScope字段(用于后端兼容)
	if len(req.ApplyScopes) == 1 && req.ApplyScopes[0].ScopeType == "all" {
		req.ApplyScope = "global"
	} else {
		req.ApplyScope = "specific"
	}

	rule, err := t.provider.Rule().CreateWithRelations(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create rule with relations", "error", err)
		return common.NewExecuteError("Failed to create rule", err)
	}

	result, _ := json.MarshalIndent(rule, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(fmt.Sprintf("Rule created successfully:\n%s", string(result)))},
	}, nil
}
