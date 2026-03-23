package client

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// Rule 规则管理实现

func (c *rosteringClient) CreateRule(ctx context.Context, req model.CreateRuleRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create rule: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal rule create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateRule(ctx context.Context, req model.UpdateRuleRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolRuleUpdate.String(), req)
	if err != nil {
		return fmt.Errorf("update rule: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListRules(ctx context.Context, req model.ListRulesRequest) (*model.Page[*model.Rule], error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}

	var response model.Page[*model.Rule]
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal rule list response: %w", err)
	}

	return &response, nil
}

func (c *rosteringClient) GetRule(ctx context.Context, orgID, ruleID string) (*model.Rule, error) {
	req := struct {
		OrgID  string `json:"orgId"`
		RuleID string `json:"ruleId"`
	}{
		OrgID:  orgID,
		RuleID: ruleID,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGet.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}

	var response struct {
		Rule model.Rule `json:"rule"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal rule get response: %w", err)
	}

	return &response.Rule, nil
}

func (c *rosteringClient) DeleteRule(ctx context.Context, orgID, ruleID string) error {
	req := struct {
		OrgID  string `json:"orgId"`
		RuleID string `json:"ruleId"`
	}{
		OrgID:  orgID,
		RuleID: ruleID,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolRuleDelete.String(), req)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	return nil
}

func (c *rosteringClient) AddRuleAssociations(ctx context.Context, req model.AddRuleAssociationsRequest) error {
	_, err := c.toolBus.Execute(ctx, tool.ToolRuleAddAssociations.String(), req)
	if err != nil {
		return fmt.Errorf("add rule associations: %w", err)
	}
	return nil
}

func (c *rosteringClient) GetRulesForEmployee(ctx context.Context, orgID, employeeID, date string) ([]*model.Rule, error) {
	req := struct {
		OrgID      string `json:"orgId"`
		EmployeeID string `json:"employeeId"`
		Date       string `json:"date,omitempty"`
	}{
		OrgID:      orgID,
		EmployeeID: employeeID,
		Date:       date,
	}

	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForEmployee.String(), req)
	if err != nil {
		return nil, fmt.Errorf("get rules for employee: %w", err)
	}

	// API直接返回数组，不是对象包装
	var rules []*model.Rule
	if err := json.Unmarshal(result, &rules); err != nil {
		return nil, fmt.Errorf("unmarshal rules for employee response: %w", err)
	}

	return rules, nil
}

func (c *rosteringClient) GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error) {
	input := map[string]any{
		"orgId":   orgID,
		"groupId": groupID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForGroup.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get rules for group: %w", err)
	}

	var rules []*model.Rule
	if err := json.Unmarshal(result, &rules); err != nil {
		return nil, fmt.Errorf("unmarshal rules response: %w", err)
	}
	return rules, nil
}

func (c *rosteringClient) GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	input := map[string]any{
		"orgId":   orgID,
		"shiftId": shiftID,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForShift.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get rules for shift: %w", err)
	}

	var rules []*model.Rule
	if err := json.Unmarshal(result, &rules); err != nil {
		return nil, fmt.Errorf("unmarshal rules response: %w", err)
	}
	return rules, nil
}

func (c *rosteringClient) GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error) {
	input := map[string]any{
		"orgId":       orgID,
		"employeeIds": employeeIDs,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForEmployees.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get rules for employees: %w", err)
	}

	// toolBus.Execute 已经处理了 MCP 格式，直接解析结果
	var rules map[string][]*model.Rule
	if err := json.Unmarshal(result, &rules); err == nil {
		return rules, nil
	}

	// 如果直接解析失败，尝试解析 MCP 工具返回的结果格式（向后兼容）
	var toolResult struct {
		Content []struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data,omitempty"`
			Text string          `json:"text,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}

	// 提取 data 字段
	for _, content := range toolResult.Content {
		if content.Type == "data" && len(content.Data) > 0 {
			if err := json.Unmarshal(content.Data, &rules); err == nil {
				return rules, nil
			}
		}
	}

	// 如果没有找到 data 类型，尝试从 text 类型解析
	for _, content := range toolResult.Content {
		if content.Type == "text" && content.Text != "" {
			if err := json.Unmarshal([]byte(content.Text), &rules); err == nil {
				return rules, nil
			}
		}
	}

	// 记录详细错误信息
	contentTypes := make([]string, len(toolResult.Content))
	for i, c := range toolResult.Content {
		contentTypes[i] = c.Type
	}

	return nil, fmt.Errorf("no data content found in tool result (contentCount: %d, contentTypes: %v)", len(toolResult.Content), contentTypes)
}

func (c *rosteringClient) GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error) {
	input := map[string]any{
		"orgId":    orgID,
		"shiftIds": shiftIDs,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForShifts.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get rules for shifts: %w", err)
	}

	// toolBus.Execute 已经处理了 MCP 格式，直接解析结果
	var rules map[string][]*model.Rule
	if err := json.Unmarshal(result, &rules); err == nil {
		return rules, nil
	}

	// 如果直接解析失败，尝试解析 MCP 工具返回的结果格式（向后兼容）
	var toolResult struct {
		Content []struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data,omitempty"`
			Text string          `json:"text,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}

	// 提取 data 字段
	for _, content := range toolResult.Content {
		if content.Type == "data" && len(content.Data) > 0 {
			if err := json.Unmarshal(content.Data, &rules); err == nil {
				return rules, nil
			}
		}
	}

	// 如果没有找到 data 类型，尝试从 text 类型解析
	for _, content := range toolResult.Content {
		if content.Type == "text" && content.Text != "" {
			if err := json.Unmarshal([]byte(content.Text), &rules); err == nil {
				return rules, nil
			}
		}
	}

	// 记录详细错误信息
	contentTypes := make([]string, len(toolResult.Content))
	for i, c := range toolResult.Content {
		contentTypes[i] = c.Type
	}

	return nil, fmt.Errorf("no data content found in tool result (contentCount: %d, contentTypes: %v)", len(toolResult.Content), contentTypes)
}

func (c *rosteringClient) GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error) {
	input := map[string]any{
		"orgId":    orgID,
		"groupIds": groupIDs,
	}
	result, err := c.toolBus.Execute(ctx, tool.ToolRuleGetForGroups.String(), input)
	if err != nil {
		return nil, fmt.Errorf("get rules for groups: %w", err)
	}

	// 解析 MCP 工具返回的结果格式
	var toolResult struct {
		Content []struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}

	// 提取 data 字段
	var rules map[string][]*model.Rule
	for _, content := range toolResult.Content {
		if content.Type == "data" && len(content.Data) > 0 {
			if err := json.Unmarshal(content.Data, &rules); err != nil {
				return nil, fmt.Errorf("unmarshal rules map: %w", err)
			}
			return rules, nil
		}
	}

	return nil, fmt.Errorf("no data content found in tool result")
}
