package chat

import (
	"context"
	"fmt"
	"time"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/ai/intent"

	"go.uber.org/zap"
)

const maxConversationHistoryMessages = 20

// UserMessage 用户消息。
type UserMessage struct {
	Content        string `json:"message"`
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"-"`
	OrgNodeID      string `json:"-"`
}

// BotResponse 机器人响应。
type BotResponse struct {
	ConversationID string            `json:"conversation_id,omitempty"`
	Reply          string            `json:"reply"`
	Intent         string            `json:"intent,omitempty"`
	Entities       map[string]string `json:"entities,omitempty"`
	Actions        []Action          `json:"actions,omitempty"`
	Usage          ai.TokenUsage     `json:"usage,omitempty"`
}

// Action 可执行动作。
type Action struct {
	Type    string         `json:"type"`
	Label   string         `json:"label"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Handler 对话处理器。
type Handler struct {
	intentParser      *intent.Parser
	provider          ai.Provider
	selector          ai.ProviderSelector
	modelResolver     ai.NodeModelResolver
	conversationStore *ConversationStore
	logger            *zap.Logger
}

// NewHandler 创建对话处理器。
func NewHandler(intentParser *intent.Parser, provider ai.Provider, logger *zap.Logger) *Handler {
	return &Handler{
		intentParser: intentParser,
		provider:     provider,
		logger:       logger.Named("chat"),
	}
}

// SetRuntimeConfig sets optional runtime provider/model resolution for chat nodes.
func (h *Handler) SetRuntimeConfig(selector ai.ProviderSelector, resolver ai.NodeModelResolver) {
	h.selector = selector
	h.modelResolver = resolver
}

// SetConversationStore enables persistent conversation history for chat requests.
func (h *Handler) SetConversationStore(store *ConversationStore) {
	h.conversationStore = store
}

// Handle 处理用户消息，根据意图路由到不同处理逻辑。
func (h *Handler) Handle(ctx context.Context, msg UserMessage) (*BotResponse, error) {
	conversationID, err := h.prepareConversation(ctx, msg)
	if err != nil {
		return nil, err
	}
	msg.ConversationID = conversationID

	botResp, err := h.routeMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	botResp.ConversationID = conversationID

	if err := h.persistAssistantReply(ctx, msg, botResp); err != nil {
		return nil, err
	}
	return botResp, nil
}

func (h *Handler) prepareConversation(ctx context.Context, msg UserMessage) (string, error) {
	if h.conversationStore == nil || msg.UserID == "" || msg.OrgNodeID == "" {
		return msg.ConversationID, nil
	}

	conversation, err := h.conversationStore.EnsureConversation(ctx, msg.ConversationID, msg.UserID, msg.OrgNodeID)
	if err != nil {
		return "", fmt.Errorf("准备对话失败: %w", err)
	}
	if _, err := h.conversationStore.AppendMessage(ctx, conversation.ID, msg.UserID, msg.OrgNodeID, "user", msg.Content); err != nil {
		return "", fmt.Errorf("保存用户消息失败: %w", err)
	}
	return conversation.ID, nil
}

func (h *Handler) persistAssistantReply(ctx context.Context, msg UserMessage, botResp *BotResponse) error {
	if h.conversationStore == nil || msg.ConversationID == "" || msg.UserID == "" || msg.OrgNodeID == "" || botResp == nil || botResp.Reply == "" {
		return nil
	}
	if _, err := h.conversationStore.AppendMessage(ctx, msg.ConversationID, msg.UserID, msg.OrgNodeID, "assistant", botResp.Reply); err != nil {
		return fmt.Errorf("保存助手消息失败: %w", err)
	}
	return nil
}

func (h *Handler) routeMessage(ctx context.Context, msg UserMessage) (*BotResponse, error) {
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
			Reply:    "正在为您查询排班信息...",
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
	reqMessages := []ai.Message{{Role: "system", Content: systemPrompt}}
	historyMessages, err := h.buildConversationHistory(ctx, msg)
	if err != nil {
		h.logger.Warn("加载对话历史失败，降级为单轮对话", zap.String("conversation_id", msg.ConversationID), zap.Error(err))
	}
	if len(historyMessages) == 0 {
		reqMessages = append(reqMessages, ai.Message{Role: "user", Content: msg.Content})
	} else {
		reqMessages = append(reqMessages, historyMessages...)
	}

	req := ai.ChatRequest{
		Messages:    reqMessages,
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	provider := h.provider
	ctx, cancel, provider, req := ai.ApplyNodeModelConfig(ctx, provider, h.selector, h.modelResolver, ai.AppScheduling, ai.WorkflowAIChat, ai.NodeChatReply, req, 120*time.Second, h.logger)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI 对话失败: %w", err)
	}

	return &BotResponse{
		Reply: resp.Content,
		Usage: resp.Usage,
	}, nil
}

func (h *Handler) buildConversationHistory(ctx context.Context, msg UserMessage) ([]ai.Message, error) {
	if h.conversationStore == nil || msg.ConversationID == "" || msg.UserID == "" || msg.OrgNodeID == "" {
		return nil, nil
	}

	messages, err := h.conversationStore.GetConversationMessages(ctx, msg.ConversationID, msg.UserID, msg.OrgNodeID)
	if err != nil {
		return nil, err
	}
	if len(messages) > maxConversationHistoryMessages {
		messages = messages[len(messages)-maxConversationHistoryMessages:]
	}

	history := make([]ai.Message, 0, len(messages))
	for _, item := range messages {
		if item.Content == "" {
			continue
		}
		history = append(history, ai.Message{Role: item.Role, Content: item.Content})
	}
	return history, nil
}
