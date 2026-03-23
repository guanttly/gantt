package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jusha/agent/server/context/domain/model"
	"jusha/agent/server/context/domain/repository"
	"jusha/agent/server/context/domain/service"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

type conversationService struct {
	logger logging.ILogger
	repo   repository.IConversationRepository
}

func NewConversationService(
	logger logging.ILogger,
	repo repository.IConversationRepository,
) service.IConversationService {
	return &conversationService{
		logger: logger.With("component", "ConversationService"),
		repo:   repo,
	}
}

func (s *conversationService) CreateConversation(ctx context.Context, meta map[string]any) (*model.Conversation, error) {
	conversation := &model.Conversation{
		ID:        uuid.NewString(),
		Meta:      model.JSONMap(meta),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	return conversation, nil
}

func (s *conversationService) AppendMessage(ctx context.Context, conversationID, messageID, role, content string, metadata map[string]any) (*model.ConversationMessage, error) {
	message := &model.ConversationMessage{
		ConversationID: conversationID,
		MessageID:      messageID,
		Role:           role,
		Content:        content,
		Timestamp:      time.Now(),
	}

	// 即使 metadata 为 nil，也传入（会被设置为 NULL）
	if metadata != nil {
		message.Metadata = model.JSONMap(metadata)
	}

	// 检查消息是否存在（用于计数）
	// 注意：在并发情况下，这个检查可能不准确，但用于计数目的已经足够
	exists := false
	if messageID != "" {
		var err error
		exists, err = s.repo.MessageExists(ctx, conversationID, messageID)
		if err != nil {
			// 如果检查失败，继续执行（可能是并发情况）
			exists = false
		}
	}

	// 使用 ON DUPLICATE KEY UPDATE，让 Repository 层处理插入或更新
	if err := s.repo.AppendMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("append message: %w", err)
	}

	// 只有在插入新消息时才增加计数
	if !exists {
		if err := s.repo.IncrementMessageCount(ctx, conversationID); err != nil {
			s.logger.Warn("Failed to increment message count", "error", err, "conversationID", conversationID)
		}
	}

	return message, nil
}

func (s *conversationService) GetMessageIDs(ctx context.Context, conversationID string) (map[string]bool, error) {
	messageIDs, err := s.repo.GetMessageIDs(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get message IDs: %w", err)
	}
	return messageIDs, nil
}

func (s *conversationService) GetConversationHistory(ctx context.Context, conversationID string, limit int) ([]*model.ConversationMessage, error) {
	messages, err := s.repo.GetMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("get conversation history: %w", err)
	}

	return messages, nil
}

func (s *conversationService) ListConversations(ctx context.Context, metaFilters map[string]any, limit, offset int) ([]*model.Conversation, int, error) {
	conversations, total, err := s.repo.ListConversations(ctx, metaFilters, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	return conversations, total, nil
}

func (s *conversationService) UpdateWorkflowContext(ctx context.Context, conversationID string, context map[string]any) error {
	if err := s.repo.UpdateWorkflowContext(ctx, conversationID, model.JSONMap(context)); err != nil {
		return fmt.Errorf("update workflow context: %w", err)
	}
	return nil
}

func (s *conversationService) GetWorkflowContext(ctx context.Context, conversationID string) (map[string]any, error) {
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		// 如果 conversation 不存在，返回空的 context 而不是错误
		// 这样可以支持加载还没有保存过 workflow context 的对话
		if strings.Contains(err.Error(), "conversation not found") {
			// Conversation 不存在，返回空的 context（可能是新对话还没有保存过 workflow context）
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if conversation.WorkflowContext == nil {
		return make(map[string]any), nil
	}

	return map[string]any(conversation.WorkflowContext), nil
}

func (s *conversationService) UpdateConversationMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error {
	if err := s.repo.UpdateMeta(ctx, conversationID, metaUpdates); err != nil {
		return fmt.Errorf("update conversation meta: %w", err)
	}
	return nil
}
