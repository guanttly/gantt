package rule

import (
	"context"
	"encoding/json"
	"time"

	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// updateRuleTool 更新规则工具
type updateRuleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewUpdateRuleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &updateRuleTool{logger: logger, provider: provider}
}

func (t *updateRuleTool) Name() string {
	return "rostering.rule.update"
}

func (t *updateRuleTool) Description() string {
	return "Update scheduling rule"
}

func (t *updateRuleTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Rule ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Rule name",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Rule code (optional)",
			},
			"ruleType": map[string]any{
				"type":        "string",
				"description": "Rule type: exclusive, combinable, required_together, periodic, maxCount, forbidden_day, preferred",
				"enum":        []string{"exclusive", "combinable", "required_together", "periodic", "maxCount", "forbidden_day", "preferred"},
			},
			"applyScope": map[string]any{
				"type":        "string",
				"description": "Apply scope: global or specific",
				"enum":        []string{"global", "specific"},
			},
			"timeScope": map[string]any{
				"type":        "string",
				"description": "Time scope: same_day, same_week, same_month, custom",
				"enum":        []string{"same_day", "same_week", "same_month", "custom"},
			},
			"timeOffsetDays": map[string]any{
				"type":        "integer",
				"description": "Time offset in days for cross-day dependencies (e.g. -1 for yesterday, 1 for tomorrow, -7 for last week).",
			},
			"ruleData": map[string]any{
				"type":        "object",
				"description": "Rule-specific data (JSON object)",
			},
			"maxCount": map[string]any{
				"type":        "number",
				"description": "Maximum count (optional)",
			},
			"consecutiveMax": map[string]any{
				"type":        "number",
				"description": "Maximum consecutive occurrences (optional)",
			},
			"intervalDays": map[string]any{
				"type":        "number",
				"description": "Interval days between shifts (optional)",
			},
			"minRestDays": map[string]any{
				"type":        "number",
				"description": "Minimum rest days (optional)",
			},
			"validFrom": map[string]any{
				"type":        "string",
				"description": "Valid from date (ISO 8601 format, optional)",
			},
			"validTo": map[string]any{
				"type":        "string",
				"description": "Valid to date (ISO 8601 format, optional)",
			},
			"priority": map[string]any{
				"type":        "number",
				"description": "Priority level (optional)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether rule is active (optional)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Rule description (optional)",
			},
			// V4新增字段
			"category": map[string]any{
				"type":        "string",
				"description": "Rule category: constraint, preference, dependency",
				"enum":        []string{"constraint", "preference", "dependency"},
			},
			"subCategory": map[string]any{
				"type":        "string",
				"description": "Rule sub-category: forbid, must, limit, prefer, suggest, source, resource, order",
				"enum":        []string{"forbid", "must", "limit", "prefer", "suggest", "source", "resource", "order"},
			},
			"sourceType": map[string]any{
				"type":        "string",
				"description": "Source type: manual, llm_parsed, migrated",
				"enum":        []string{"manual", "llm_parsed", "migrated"},
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Rule version: v3 or v4",
				"enum":        []string{"v3", "v4"},
			},
		},
		"required": []string{"id"},
	}
}

func (t *updateRuleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Rule ID is required")
	}

	req := &model.UpdateRuleRequest{
		OrgID:       common.GetString(input, "orgId"),
		Name:        common.GetString(input, "name"),
		RuleType:    common.GetString(input, "ruleType"),
		ApplyScope:  common.GetString(input, "applyScope"),
		TimeScope:   common.GetString(input, "timeScope"),
		Description: common.GetString(input, "description"),
	}

	// Handle ruleData
	if ruleData, ok := input["ruleData"].(map[string]any); ok {
		jsonData, _ := json.Marshal(ruleData)
		req.RuleData = string(jsonData)
	}

	// Handle timeOffsetDays
	if offsetVal, ok := input["timeOffsetDays"].(float64); ok {
		offsetInt := int(offsetVal)
		req.TimeOffsetDays = &offsetInt
	}

	// Handle numeric parameters
	if maxCount, ok := input["maxCount"].(float64); ok {
		maxCountInt := int(maxCount)
		req.MaxCount = &maxCountInt
	}
	if consecutiveMax, ok := input["consecutiveMax"].(float64); ok {
		consecutiveMaxInt := int(consecutiveMax)
		req.ConsecutiveMax = &consecutiveMaxInt
	}
	if intervalDays, ok := input["intervalDays"].(float64); ok {
		intervalDaysInt := int(intervalDays)
		req.IntervalDays = &intervalDaysInt
	}
	if minRestDays, ok := input["minRestDays"].(float64); ok {
		minRestDaysInt := int(minRestDays)
		req.MinRestDays = &minRestDaysInt
	}

	// Handle date ranges
	if validFromStr := common.GetString(input, "validFrom"); validFromStr != "" {
		validFrom, err := time.Parse(time.RFC3339, validFromStr)
		if err == nil {
			req.ValidFrom = &validFrom
		}
	}
	if validToStr := common.GetString(input, "validTo"); validToStr != "" {
		validTo, err := time.Parse(time.RFC3339, validToStr)
		if err == nil {
			req.ValidTo = &validTo
		}
	}

	// Handle priority
	if priority, ok := input["priority"].(float64); ok {
		req.Priority = int(priority)
	}

	// Handle isActive
	if isActive, ok := input["isActive"].(bool); ok {
		req.IsActive = isActive
	}

	rule, err := t.provider.Rule().Update(ctx, id, req)
	if err != nil {
		t.logger.Error("Failed to update rule", "id", id, "error", err)
		return common.NewExecuteError("Failed to update rule", err)
	}

	data, _ := json.MarshalIndent(rule, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*updateRuleTool)(nil)
