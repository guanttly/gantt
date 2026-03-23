package toolcalling

import (
	"context"
)

// ITool 工具接口
type ITool interface {
	// Name 工具名称
	Name() string
	
	// Description 工具描述
	Description() string
	
	// InputSchema 工具输入参数模式（JSON Schema）
	InputSchema() map[string]any
	
	// Execute 执行工具
	Execute(ctx context.Context, arguments map[string]any) (*ToolResult, error)
}

// ToolResult 工具执行结果
type ToolResult struct {
	// Content 结果内容（文本或JSON）
	Content string `json:"content"`
	
	// IsError 是否为错误结果
	IsError bool `json:"isError,omitempty"`
	
	// Metadata 元数据（可选）
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewTextResult 创建文本结果
func NewTextResult(content string) *ToolResult {
	return &ToolResult{
		Content: content,
		IsError: false,
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err error) *ToolResult {
	return &ToolResult{
		Content: err.Error(),
		IsError: true,
	}
}

// NewJSONResult 创建JSON结果
func NewJSONResult(content string) *ToolResult {
	return &ToolResult{
		Content: content,
		IsError: false,
		Metadata: map[string]any{
			"format": "json",
		},
	}
}
