// Package session 提供意图识别相关接口和类型
package session

import "context"

// Intent 意图识别结果
type Intent struct {
	// Type 意图类型（如 "schedule.create", "schedule.query" 等）
	Type string `json:"type"`

	// Confidence 置信度（0.0 - 1.0）
	Confidence float64 `json:"confidence"`

	// Entities 提取的实体信息
	Entities map[string]any `json:"entities"`

	// RawText 原始文本
	RawText string `json:"rawText"`

	// Metadata 额外的元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// IntentRecognizeRequest 意图识别请求
type IntentRecognizeRequest struct {
	// SessionID 会话ID
	SessionID string

	// UserMessage 用户消息
	UserMessage string

	// Context 会话上下文
	Context IntentRecognizeContext
}

// IntentRecognizeContext 意图识别的上下文信息
// 提供会话的基本信息，帮助识别器理解用户意图
type IntentRecognizeContext struct {
	// OrgID 组织ID
	OrgID string `json:"orgId"`

	// UserID 用户ID
	UserID string `json:"userId"`

	// AgentType 代理类型（如 "rostering", "scheduling" 等）
	AgentType string `json:"agentType"`

	// CurrentState 当前会话状态
	CurrentState SessionState `json:"currentState"`

	// Messages 历史消息列表
	Messages []Message `json:"messages"`
}

// IntentRecognizeResponse 意图识别响应
type IntentRecognizeResponse struct {
	// Intent 识别的意图
	Intent *Intent

	// Suggestions 建议的后续操作
	Suggestions []string

	// NeedsMoreInfo 是否需要更多信息
	NeedsMoreInfo bool

	// MissingFields 缺失的必填字段
	MissingFields []string
}

// IIntentRecognizer 意图识别接口
// 用于识别用户输入的意图，支持自然语言理解
//
// 实现建议：
//   - 简单实现：基于规则匹配（关键词、正则）
//   - 中等实现：本地 NLP 模型（如 BERT）
//   - 高级实现：调用 OpenAI/Claude 等大模型 API
type IIntentRecognizer interface {
	// Recognize 识别用户输入的意图
	// 返回意图识别结果，如果无法识别返回 nil
	Recognize(ctx context.Context, req IntentRecognizeRequest) (*IntentRecognizeResponse, error)

	// ValidateIntent 验证意图的必填字段是否完整
	// 返回缺失的字段列表
	ValidateIntent(ctx context.Context, intent *Intent) (missingFields []string, err error)

	// SupportedIntents 返回支持的意图类型列表
	SupportedIntents() []string
}
