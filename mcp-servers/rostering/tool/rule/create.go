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

// createRuleTool 创建排班规则工具
type createRuleTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewCreateRuleTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &createRuleTool{logger: logger, provider: provider}
}

func (t *createRuleTool) Name() string {
	return "rostering.rule.create"
}

func (t *createRuleTool) Description() string {
	return "Create a scheduling rule"
}

func (t *createRuleTool) InputSchema() map[string]any {
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
				"description": "Time offset in days for cross-day dependencies (e.g. -1 for yesterday, 1 for tomorrow, -7 for last week). Essential for rules like 'must be the person who worked shift B yesterday'.",
			},
			"ruleData": map[string]any{
				"type":        "string",
				"description": "Rule description text (e.g., 'every other week night shift', 'no more than 2 consecutive days')",
			},
			"maxCount": map[string]any{
				"type":        "number",
				"description": "Maximum count (for maxCount type rules, e.g., 'maximum 3 times per week')",
			},
			"consecutiveMax": map[string]any{
				"type":        "number",
				"description": "Maximum consecutive days (for maxCount type rules, e.g., 'no more than 2 consecutive days')",
			},
			"intervalDays": map[string]any{
				"type":        "number",
				"description": "Interval days (for periodic type rules, e.g., '7 days apart', 'every other week = 14 days')",
			},
			"minRestDays": map[string]any{
				"type":        "number",
				"description": "Minimum rest days (for rest rules, e.g., 'at least 1 day rest after night shift')",
			},
			"priority": map[string]any{
				"type":        "number",
				"description": "Priority (higher = more important)",
			},
			"isActive": map[string]any{
				"type":        "boolean",
				"description": "Whether rule is active",
			},
			"validFrom": map[string]any{
				"type":        "string",
				"description": "Valid from date (ISO 8601 format)",
			},
			"validTo": map[string]any{
				"type":        "string",
				"description": "Valid to date (ISO 8601 format)",
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
			"originalRuleId": map[string]any{
				"type":        "string",
				"description": "Original rule ID (if parsed from semantic rule)",
			},
			"sourceType": map[string]any{
				"type":        "string",
				"description": "Rule source type: manual, llm_parsed, migrated",
				"enum":        []string{"manual", "llm_parsed", "migrated"},
			},
			"parseConfidence": map[string]any{
				"type":        "number",
				"description": "LLM parse confidence (0.0-1.0)",
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Rule version: v3, v4",
				"enum":        []string{"v3", "v4"},
			},
		},
		"required": []string{"orgId", "name", "description", "ruleType"},
	}
}

func (t *createRuleTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	req := &model.CreateRuleRequest{
		OrgID:       common.GetString(input, "orgId"),
		Name:        common.GetString(input, "name"),
		Description: common.GetString(input, "description"),
		RuleType:    common.GetString(input, "ruleType"),
		ApplyScope:  common.GetString(input, "applyScope"),
		TimeScope:   common.GetString(input, "timeScope"),
		RuleData:    common.GetString(input, "ruleData"),
		Priority:    common.GetInt(input, "priority"),
		// V4新增字段
		Category:       common.GetString(input, "category"),
		SubCategory:    common.GetString(input, "subCategory"),
		OriginalRuleID: common.GetString(input, "originalRuleId"),
		SourceType:     common.GetString(input, "sourceType"),
		Version:        common.GetString(input, "version"),
	}

	// Handle timeOffsetDays
	if offsetVal, ok := input["timeOffsetDays"].(float64); ok {
		offsetInt := int(offsetVal)
		req.TimeOffsetDays = &offsetInt
	}

	// Handle parseConfidence (float)
	if parseConfidenceVal, ok := input["parseConfidence"].(float64); ok {
		req.ParseConfidence = &parseConfidenceVal
	}

	// Handle numeric rule parameters
	if maxCount := common.GetInt(input, "maxCount"); maxCount > 0 {
		req.MaxCount = &maxCount
	}
	if consecutiveMax := common.GetInt(input, "consecutiveMax"); consecutiveMax > 0 {
		req.ConsecutiveMax = &consecutiveMax
	}
	if intervalDays := common.GetInt(input, "intervalDays"); intervalDays > 0 {
		req.IntervalDays = &intervalDays
	}
	if minRestDays := common.GetInt(input, "minRestDays"); minRestDays > 0 {
		req.MinRestDays = &minRestDays
	}

	// Handle isActive boolean
	if isActiveVal, ok := input["isActive"].(bool); ok {
		req.IsActive = isActiveVal
	}

	// Parse date fields if provided
	if validFromStr := common.GetString(input, "validFrom"); validFromStr != "" {
		if validFrom, err := time.Parse(time.RFC3339, validFromStr); err == nil {
			req.ValidFrom = &validFrom
		}
	}
	if validToStr := common.GetString(input, "validTo"); validToStr != "" {
		if validTo, err := time.Parse(time.RFC3339, validToStr); err == nil {
			req.ValidTo = &validTo
		}
	}

	rule, err := t.provider.Rule().Create(ctx, req)
	if err != nil {
		t.logger.Error("Failed to create rule", "error", err)
		return common.NewExecuteError("Failed to create rule", err)
	}

	data, _ := json.MarshalIndent(rule, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createRuleTool)(nil)
