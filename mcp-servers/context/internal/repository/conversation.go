package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"jusha/agent/server/context/domain/model"
	"jusha/agent/server/context/domain/repository"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

type conversationRepository struct {
	db     *gorm.DB
	logger logging.ILogger
}

func NewConversationRepository(db *gorm.DB, logger logging.ILogger) repository.IConversationRepository {
	return &conversationRepository{
		db:     db,
		logger: logger.With("component", "ConversationRepository"),
	}
}

func (r *conversationRepository) CreateConversation(ctx context.Context, conversation *model.Conversation) error {
	if conversation.ID == "" {
		conversation.ID = uuid.NewString()
	}
	if conversation.CreatedAt.IsZero() {
		conversation.CreatedAt = time.Now()
	}
	if conversation.UpdatedAt.IsZero() {
		conversation.UpdatedAt = time.Now()
	}

	if err := r.db.WithContext(ctx).Create(conversation).Error; err != nil {
		r.logger.Error("Failed to create conversation", "error", err, "id", conversation.ID)
		return fmt.Errorf("create conversation: %w", err)
	}

	return nil
}

func (r *conversationRepository) GetConversation(ctx context.Context, id string) (*model.Conversation, error) {
	var conversation model.Conversation
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
		r.logger.Error("Failed to get conversation", "error", err, "id", id)
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	// Meta 已经在 Scan 方法中解析

	return &conversation, nil
}

func (r *conversationRepository) AppendMessage(ctx context.Context, message *model.ConversationMessage) error {
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	const maxRetries = 5
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 更长的指数退避，添加随机抖动以减少冲突：50ms, 150ms, 350ms, 750ms
			baseDelay := time.Duration(50*(1<<uint(attempt-1))) * time.Millisecond
			jitter := time.Duration(rand.Int63n(int64(baseDelay / 2)))
			delay := baseDelay + jitter
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			r.logger.Warn("Retrying AppendMessage after deadlock",
				"attempt", attempt,
				"delay_ms", delay.Milliseconds(),
				"conversationID", message.ConversationID,
				"messageID", message.MessageID)
		}

		// 先尝试查找是否已存在，减少锁冲突
		if message.MessageID != "" {
			var existingMsg model.ConversationMessage
			err := r.db.WithContext(ctx).
				Where("conversation_id = ? AND message_id = ?", message.ConversationID, message.MessageID).
				First(&existingMsg).Error

			if err == nil {
				// 消息已存在，执行更新
				err = r.db.WithContext(ctx).
					Model(&existingMsg).
					Updates(map[string]interface{}{
						"metadata":  message.Metadata,
						"content":   message.Content,
						"role":      message.Role,
						"timestamp": message.Timestamp,
					}).Error

				if err == nil {
					return nil
				}
				// 更新失败，检查是否是死锁
				errStr := err.Error()
				if strings.Contains(errStr, "Error 1213") || strings.Contains(errStr, "Deadlock") {
					lastErr = err
					continue // 重试
				}
				r.logger.Error("Failed to update message", "error", err, "conversationID", message.ConversationID, "messageID", message.MessageID)
				return fmt.Errorf("update message: %w", err)
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				// 查询出错但不是"未找到"
				errStr := err.Error()
				if strings.Contains(errStr, "Error 1213") || strings.Contains(errStr, "Deadlock") {
					lastErr = err
					continue // 重试
				}
				r.logger.Error("Failed to query message", "error", err, "conversationID", message.ConversationID, "messageID", message.MessageID)
				return fmt.Errorf("query message: %w", err)
			}
			// 记录不存在，继续插入
		}

		// 使用 ON DUPLICATE KEY UPDATE 插入新消息
		err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "conversation_id"},
					{Name: "message_id"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"metadata", "content", "role", "timestamp"}),
			}).
			Create(message).Error

		if err == nil {
			return nil
		}

		// 检查是否是死锁错误
		errStr := err.Error()
		if strings.Contains(errStr, "Error 1213") || strings.Contains(errStr, "Deadlock") {
			lastErr = err
			continue // 重试
		}

		// 其他错误，直接返回
		r.logger.Error("Failed to append message", "error", err, "conversationID", message.ConversationID, "messageID", message.MessageID, "role", message.Role)
		return fmt.Errorf("append message: %w", err)
	}

	// 重试失败
	r.logger.Error("Failed to append message after retries", "error", lastErr, "conversationID", message.ConversationID, "messageID", message.MessageID, "role", message.Role)
	return fmt.Errorf("append message after %d retries: %w", maxRetries, lastErr)
}

func (r *conversationRepository) GetMessages(ctx context.Context, conversationID string, limit int) ([]*model.ConversationMessage, error) {
	var messages []*model.ConversationMessage
	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("timestamp ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&messages).Error; err != nil {
		r.logger.Error("Failed to get messages", "error", err, "conversationID", conversationID)
		return nil, fmt.Errorf("get messages: %w", err)
	}

	return messages, nil
}

func (r *conversationRepository) MessageExists(ctx context.Context, conversationID, messageID string) (bool, error) {
	if messageID == "" {
		return false, nil
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.ConversationMessage{}).
		Where("conversation_id = ? AND message_id = ?", conversationID, messageID).
		Count(&count).Error; err != nil {
		r.logger.Error("Failed to check message exists", "error", err, "conversationID", conversationID, "messageID", messageID)
		return false, fmt.Errorf("check message exists: %w", err)
	}

	return count > 0, nil
}

func (r *conversationRepository) GetMessageByMessageID(ctx context.Context, conversationID, messageID string) (*model.ConversationMessage, error) {
	var message model.ConversationMessage
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND message_id = ?", conversationID, messageID).
		First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found: %w", err)
		}
		r.logger.Error("Failed to get message by messageID", "error", err, "conversationID", conversationID, "messageID", messageID)
		return nil, fmt.Errorf("get message by messageID: %w", err)
	}
	return &message, nil
}

func (r *conversationRepository) GetMessageIDs(ctx context.Context, conversationID string) (map[string]bool, error) {
	var messages []*model.ConversationMessage
	if err := r.db.WithContext(ctx).
		Select("message_id").
		Where("conversation_id = ? AND message_id IS NOT NULL AND message_id != ''", conversationID).
		Find(&messages).Error; err != nil {
		r.logger.Error("Failed to get message IDs", "error", err, "conversationID", conversationID)
		return nil, fmt.Errorf("get message IDs: %w", err)
	}

	messageIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.MessageID != "" {
			messageIDs[msg.MessageID] = true
		}
	}
	return messageIDs, nil
}

func (r *conversationRepository) ListConversations(ctx context.Context, filters map[string]any, limit, offset int) ([]*model.Conversation, int, error) {
	var conversations []*model.Conversation
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.Conversation{})

	// 通过 meta JSON 字段过滤（支持任意类型）
	for key, value := range filters {
		// 使用 JSON_EXTRACT 和 JSON_UNQUOTE 处理不同类型的值
		query = query.Where("JSON_EXTRACT(meta, ?) = ?", fmt.Sprintf("$.%s", key), value)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("Failed to count conversations", "error", err)
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}

	query = query.Order("updated_at DESC")

	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&conversations).Error; err != nil {
		r.logger.Error("Failed to list conversations", "error", err)
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}

	return conversations, int(total), nil
}

func (r *conversationRepository) UpdateWorkflowContext(ctx context.Context, conversationID string, context model.JSONMap) error {
	updateData := map[string]interface{}{
		"workflow_context": context,
		"updated_at":       time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(updateData).Error; err != nil {
		r.logger.Error("Failed to update workflow context", "error", err, "conversationID", conversationID)
		return fmt.Errorf("update workflow context: %w", err)
	}

	return nil
}

func (r *conversationRepository) UpdateMeta(ctx context.Context, conversationID string, metaUpdates map[string]any) error {
	// 先获取现有的 conversation
	conversation, err := r.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	// 合并 meta 更新
	currentMeta := map[string]any(conversation.Meta)
	if currentMeta == nil {
		currentMeta = make(map[string]any)
	}

	for key, value := range metaUpdates {
		currentMeta[key] = value
	}

	// 更新数据库
	updateData := map[string]interface{}{
		"meta":       model.JSONMap(currentMeta),
		"updated_at": time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(updateData).Error; err != nil {
		r.logger.Error("Failed to update meta", "error", err, "conversationID", conversationID)
		return fmt.Errorf("update meta: %w", err)
	}

	return nil
}

func (r *conversationRepository) IncrementMessageCount(ctx context.Context, conversationID string) error {
	if err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("total_messages", gorm.Expr("total_messages + ?", 1)).Error; err != nil {
		r.logger.Warn("Failed to increment message count", "error", err, "conversationID", conversationID)
		return fmt.Errorf("increment message count: %w", err)
	}
	return nil
}
