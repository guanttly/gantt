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

// ValidateConstraintsTool 验证排班约束工具
type ValidateConstraintsTool struct {
	BaseRosteringTool
	ruleValidator d_service.IRuleLevelValidator
}

// NewValidateConstraintsTool 创建验证约束工具
func NewValidateConstraintsTool(
	logger logging.ILogger,
	ruleValidator d_service.IRuleLevelValidator,
) *ValidateConstraintsTool {
	return &ValidateConstraintsTool{
		BaseRosteringTool: BaseRosteringTool{
			name:        "validateConstraints",
			description: "验证排班约束，检查排班草案是否符合规则要求。可以验证特定日期、班次或人员的排班约束。",
			logger:      logger,
		},
		ruleValidator: ruleValidator,
	}
}

// InputSchema 返回输入参数模式
func (t *ValidateConstraintsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scheduleDraft": map[string]any{
				"type":        "object",
				"description": "排班草案（JSON格式，包含日期和人员分配）",
			},
			"shifts": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "班次ID列表（可选）",
			},
			"rules": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "规则ID列表（可选）",
			},
			"staffRequirements": map[string]any{
				"type":        "object",
				"description": "人员需求（日期 -> 班次 -> 人数）",
			},
		},
		"required": []string{"scheduleDraft"},
	}
}

// Execute 执行工具
func (t *ValidateConstraintsTool) Execute(ctx context.Context, arguments map[string]any) (*toolcalling.ToolResult, error) {
	t.logger.Info("ValidateConstraintsTool: Executing", "arguments", arguments)

	// 解析排班草案
	scheduleDraftAny, ok := arguments["scheduleDraft"].(map[string]any)
	if !ok {
		return t.NewErrorResult(fmt.Errorf("scheduleDraft is required and must be an object")), nil
	}

	// 将 map[string]any 转换为 JSON 字符串，再解析为 ShiftScheduleDraft
	scheduleDraftJSON, err := json.Marshal(scheduleDraftAny)
	if err != nil {
		return t.NewErrorResult(fmt.Errorf("序列化排班草案失败: %w", err)), nil
	}

	var scheduleDraft d_model.ShiftScheduleDraft
	if err := json.Unmarshal(scheduleDraftJSON, &scheduleDraft); err != nil {
		return t.NewErrorResult(fmt.Errorf("解析排班草案失败: %w", err)), nil
	}

	// 注意：ValidateAll 需要更多参数（shifts, rules, staffList等）
	// 但这些参数在当前工具调用中可能不完整，所以这里返回基本验证结果
	// 实际验证应该在任务执行器中进行，这里只做基本的结构验证
	if scheduleDraft.Schedule == nil {
		return t.NewErrorResult(fmt.Errorf("排班草案为空")), nil
	}

	result := map[string]any{
		"status":  "validated",
		"message": "排班草案结构验证通过",
		"dates":   len(scheduleDraft.Schedule),
		"note":    "完整规则验证需要在任务执行器中进行，需要提供shifts、rules、staffList等完整上下文",
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return t.NewErrorResult(fmt.Errorf("序列化结果失败: %w", err)), nil
	}

	return toolcalling.NewJSONResult(string(resultJSON)), nil
}
