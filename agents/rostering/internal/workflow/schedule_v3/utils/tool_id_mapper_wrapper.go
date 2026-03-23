package utils

import (
	"context"

	"jusha/mcp/pkg/ai/toolcalling"
)

// ToolIDMapperWrapper 工具ID映射包装器
// 在工具执行前，将参数中的简短ID替换回真实ID
type ToolIDMapperWrapper struct {
	tool              toolcalling.ITool
	shiftReverseMappings map[string]string
	ruleReverseMappings  map[string]string
}

// NewToolIDMapperWrapper 创建工具ID映射包装器
func NewToolIDMapperWrapper(
	tool toolcalling.ITool,
	shiftReverseMappings, ruleReverseMappings map[string]string,
) toolcalling.ITool {
	return &ToolIDMapperWrapper{
		tool:                  tool,
		shiftReverseMappings: shiftReverseMappings,
		ruleReverseMappings:  ruleReverseMappings,
	}
}

// Name 返回工具名称
func (w *ToolIDMapperWrapper) Name() string {
	return w.tool.Name()
}

// Description 返回工具描述
func (w *ToolIDMapperWrapper) Description() string {
	return w.tool.Description()
}

// InputSchema 返回工具输入参数模式
func (w *ToolIDMapperWrapper) InputSchema() map[string]any {
	return w.tool.InputSchema()
}

// Execute 执行工具（在执行前替换参数中的ID）
func (w *ToolIDMapperWrapper) Execute(ctx context.Context, arguments map[string]any) (*toolcalling.ToolResult, error) {
	// 在执行前，替换参数中的简短ID为真实ID
	// 注意：这里需要深拷贝arguments，避免修改原始参数
	replacedArguments := make(map[string]any)
	for k, v := range arguments {
		replacedArguments[k] = v
	}
	
	// 替换参数中的ID
	ReplaceIDsInToolArguments(replacedArguments, w.shiftReverseMappings, w.ruleReverseMappings)
	
	// 使用替换后的参数执行工具
	return w.tool.Execute(ctx, replacedArguments)
}
