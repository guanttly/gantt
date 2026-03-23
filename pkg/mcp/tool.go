package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/mcp/client"
	"jusha/mcp/pkg/mcp/manager"
	"jusha/mcp/pkg/mcp/model"
)

// 类型别名，方便外部包使用
type (
	CallToolResult   = model.CallToolResult
	Content          = model.Content
	MCPClient        = client.MCPClient
	MCPServerManager = manager.MCPServerManager
)

var (
	NewDefaultMCPServerManager = manager.NewDefaultMCPServerManager
)

// 便捷函数别名
var (
	NewTextContent = model.NewTextContent
	NewDataContent = model.NewDataContent
)

// ToolError 工具执行错误，包含业务错误码
type ToolError struct {
	code    errors.ErrorCode
	message string
	data    interface{}
	wrapped error
}

// NewToolError 创建工具错误
func NewToolError(code errors.ErrorCode, message string) error {
	return &ToolError{
		code:    code,
		message: message,
		data:    nil,
		wrapped: nil,
	}
}

// NewToolErrorWithData 创建带数据的工具错误
func NewToolErrorWithData(code errors.ErrorCode, message string, data interface{}) error {
	return &ToolError{
		code:    code,
		message: message,
		data:    data,
		wrapped: nil,
	}
}

// WrapToolError 包装现有错误为工具错误
func WrapToolError(code errors.ErrorCode, message string, wrapped error) error {
	return &ToolError{
		code:    code,
		message: message,
		data:    nil,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口
func (e *ToolError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s: %v", e.code, e.message, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s", e.code, e.message)
}

// Code 返回错误码
func (e *ToolError) Code() errors.ErrorCode {
	return e.code
}

// Message 返回错误消息
func (e *ToolError) Message() string {
	return e.message
}

// Data 返回额外的错误数据
func (e *ToolError) Data() interface{} {
	return e.data
}

// Unwrap 实现错误解包
func (e *ToolError) Unwrap() error {
	return e.wrapped
}

// IsToolError 检查是否为工具错误
func IsToolError(err error) (*ToolError, bool) {
	if err == nil {
		return nil, false
	}
	te, ok := err.(*ToolError)
	return te, ok
}

// ITool 定义工具处理接口
type ITool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	Execute(ctx context.Context, arguments map[string]any) (*model.CallToolResult, error)
}

// IToolRegistry 工具注册表接口
type IToolRegistry interface {
	RegisterTool(tool ITool) error
	UnregisterTool(name string) error
	GetTool(name string) (ITool, error)
	ListTools() []model.Tool
}

// BaseToolHandler 基础工具处理器
type BaseTool struct {
	name        string
	description string
	inputSchema map[string]any
	handler     func(ctx context.Context, arguments map[string]any) (*model.CallToolResult, error)
}

func NewBaseTool(name, description string, inputSchema map[string]any, handler func(ctx context.Context, arguments map[string]any) (*model.CallToolResult, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
		handler:     handler,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) InputSchema() map[string]any {
	return t.inputSchema
}

func (t *BaseTool) Execute(ctx context.Context, arguments map[string]any) (*model.CallToolResult, error) {
	if t.handler == nil {
		return nil, fmt.Errorf("tool handler not implemented for %s", t.name)
	}
	return t.handler(ctx, arguments)
}

// 常用的输入模式定义
type TextInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required,omitempty"`
}

func NewTextInputSchema() *TextInputSchema {
	return &TextInputSchema{
		Type: "object",
		Properties: map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Input text to process",
			},
		},
		Required: []string{"text"},
	}
}

// 创建带有数据的响应
func NewDataResult(data any) (*model.CallToolResult, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, WrapToolError(errors.PROCESSING_ERROR, "failed to marshal data", err)
	}

	return &model.CallToolResult{
		Content: []model.Content{model.NewDataContent(string(dataBytes))},
	}, nil
}
