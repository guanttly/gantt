package chat

import (
	"context"
	"fmt"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/ai/intent"

	"go.uber.org/zap"
)

// UserMessage 用户消息。
type UserMessage struct {
	Content        string `json:"message"`
	ConversationID string `json:"conversation_id"`
}

// BotResponse 机器人响应。
type BotResponse struct {
	Reply    string            `json:"reply"`
	Intent   string            `json:"intent,omitempty"`
	Entities map[string]string `json:"entities,omitempty"`
	Actions  []Action          `json:"actions,omitempty"`
	Usage    ai.TokenUsage     `json:"usage,omitempty"`
}

// Action 可执行动作。
type Action struct {
	Type    string         `json:"type"`
	Label   string         `json:"label"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Handler 对话处理器。
type Handler struct {
	intentParser *intent.Parser
	provider     ai.Provider
	logger       *zap.Logger
}

// NewHandler 创建对话处理器。
func NewHandler(intentParser *intent.Parser, provider ai.Provider, logger *zap.Logger) *Handler {
	return &Handler{
		intentParser: intentParser,
		provider:     provider,
		logger:       logger.Named("chat"),
	}
}

// Handle 处理用户消息，根据意图路由到不同处理逻辑。
func (h *Handler) Handle(ctx context.Context, msg UserMessage) (*BotResponse, error) {
	// 1. 意图识别
	intentResult, err := h.intentParser.Parse(ctx, msg.Content)
	if err != nil {
		h.logger.Warn("意图识别失败，降级为通用对话", zap.Error(err))
		return h.handleGenericChat(ctx, msg)
	}

	h.logger.Debug("意图识别结果",
		zap.String("action", intentResult.Action),
		zap.Float64("confidence", intentResult.Confidence),
	)

	// 2. 根据意图路由
	switch intentResult.Action {
	case "create_schedule":
		return &BotResponse{
			Reply:    "好的，我来帮您创建排班计划。请提供排班的起止日期和参与班次。",
			Intent:   intentResult.Action,
			Entities: intentResult.Entities,
			Actions: []Action{
				{Type: "create_schedule", Label: "创建排班", Payload: map[string]any{"entities": intentResult.Entities}},
			},
		}, nil

	case "adjust_schedule":
		return &BotResponse{
			Reply:    "收到，我来帮您调整排班。请告诉我需要调整的内容。",
			Intent:   intentResult.Action,
			Entities: intentResult.Entities,
			Actions: []Action{
				{Type: "adjust_schedule", Label: "调整排班", Payload: map[string]any{"entities": intentResult.Entities}},
			},
		}, nil

	case "query_schedule":
		return &BotResponse{
			Reply:    fmt.Sprintf("正在为您查询排班信息..."),
			Intent:   intentResult.Action,
			Entities: intentResult.Entities,
			Actions: []Action{
				{Type: "query_schedule", Label: "查询排班", Payload: map[string]any{"entities": intentResult.Entities}},
			},
		}, nil

	case "query_rule":
		return &BotResponse{
			Reply:    "正在查询相关规则配置...",
			Intent:   intentResult.Action,
			Entities: intentResult.Entities,
			Actions: []Action{
				{Type: "query_rule", Label: "查询规则", Payload: map[string]any{"entities": intentResult.Entities}},
			},
		}, nil

	default:
		return h.handleGenericChat(ctx, msg)
	}
}

// handleGenericChat 通用 AI 对话。
func (h *Handler) handleGenericChat(ctx context.Context, msg UserMessage) (*BotResponse, error) {
	systemPrompt := "You are an AI scheduling assistant. Help users manage schedules, query information, and create/adjust scheduling plans. Reply in the same language as the user."

	resp, err := h.provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: msg.Content},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("AI 对话失败: %w", err)
	}

	return &BotResponse{
		Reply: resp.Content,
		Usage: resp.Usage,
	}, nil
}
