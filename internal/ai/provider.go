// Package ai 提供 AI Provider 适配层，统一多 LLM 后端接口。
package ai

import (
	"context"
)

// Provider 定义统一的 AI Provider 接口。
type Provider interface {
	// Chat 单轮/多轮对话。
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// ChatStream 流式输出。
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
	// Name 返回 Provider 名称。
	Name() string
}

// ChatRequest 聊天请求。
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
}

// Message 聊天消息。
type Message struct {
	Role    string `json:"role"` // system / user / assistant / tool
	Content string `json:"content"`
}

// ChatResponse 聊天响应。
type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        TokenUsage `json:"usage"`
	FinishReason string     `json:"finish_reason"`
}

// StreamChunk 流式输出数据块。
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// TokenUsage Token 消耗统计。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition 工具定义（Function Calling）。
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall 工具调用。
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}
