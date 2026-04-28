package chat

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const conversationTitleLayout = "01-02 15:04"

type Conversation struct {
	ID                 string     `gorm:"primaryKey;size:64" json:"id"`
	OrgNodeID          string     `gorm:"size:64;not null;index:idx_ai_conversation_user_time,priority:1" json:"-"`
	UserID             string     `gorm:"size:64;not null;index:idx_ai_conversation_user_time,priority:2" json:"-"`
	Title              string     `gorm:"size:128;not null" json:"title"`
	MessageCount       int        `gorm:"not null;default:0" json:"message_count"`
	LastMessageAt      *time.Time `gorm:"index:idx_ai_conversation_last_message_at" json:"last_message_at,omitempty"`
	LastMessagePreview string     `gorm:"size:255;not null;default:''" json:"last_message_preview,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (Conversation) TableName() string {
	return "ai_conversations"
}

type ConversationMessage struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ConversationID string    `gorm:"size:64;not null;index:idx_ai_conversation_message_time,priority:1" json:"conversation_id"`
	OrgNodeID      string    `gorm:"size:64;not null;index:idx_ai_conversation_message_owner,priority:1" json:"-"`
	UserID         string    `gorm:"size:64;not null;index:idx_ai_conversation_message_owner,priority:2" json:"-"`
	Role           string    `gorm:"size:16;not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (ConversationMessage) TableName() string {
	return "ai_conversation_messages"
}

type ConversationDetail struct {
	Conversation
	Messages []ConversationMessage `json:"messages"`
}

type ConversationStore struct {
	db *gorm.DB
}

func NewConversationStore(db *gorm.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Conversation{}, &ConversationMessage{})
}

func (s *ConversationStore) CreateConversation(ctx context.Context, userID, orgNodeID string) (*Conversation, error) {
	now := time.Now()
	conversation := &Conversation{
		ID:        uuid.NewString(),
		OrgNodeID: orgNodeID,
		UserID:    userID,
		Title:     now.Format(conversationTitleLayout),
	}
	if err := s.db.WithContext(ctx).Create(conversation).Error; err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *ConversationStore) EnsureConversation(ctx context.Context, conversationID, userID, orgNodeID string) (*Conversation, error) {
	if strings.TrimSpace(conversationID) == "" {
		return s.CreateConversation(ctx, userID, orgNodeID)
	}
	return s.GetConversation(ctx, conversationID, userID, orgNodeID)
}

func (s *ConversationStore) GetConversation(ctx context.Context, conversationID, userID, orgNodeID string) (*Conversation, error) {
	var conversation Conversation
	err := s.db.WithContext(ctx).
		Where("id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (s *ConversationStore) GetConversationMessages(ctx context.Context, conversationID, userID, orgNodeID string) ([]ConversationMessage, error) {
	var messages []ConversationMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *ConversationStore) GetConversationDetail(ctx context.Context, conversationID, userID, orgNodeID string) (*ConversationDetail, error) {
	conversation, err := s.GetConversation(ctx, conversationID, userID, orgNodeID)
	if err != nil {
		return nil, err
	}
	messages, err := s.GetConversationMessages(ctx, conversationID, userID, orgNodeID)
	if err != nil {
		return nil, err
	}
	return &ConversationDetail{
		Conversation: *conversation,
		Messages:     messages,
	}, nil
}

func (s *ConversationStore) ListConversations(ctx context.Context, userID, orgNodeID string) ([]Conversation, error) {
	var conversations []Conversation
	err := s.db.WithContext(ctx).
		Where("org_node_id = ? AND user_id = ?", orgNodeID, userID).
		Order("updated_at DESC").
		Order("created_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}
	return conversations, nil
}

func (s *ConversationStore) UpdateConversationTitle(ctx context.Context, conversationID, userID, orgNodeID, title string) (*Conversation, error) {
	title = strings.TrimSpace(title)
	if err := s.db.WithContext(ctx).
		Model(&Conversation{}).
		Where("id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
		Updates(map[string]any{
			"title":      title,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	return s.GetConversation(ctx, conversationID, userID, orgNodeID)
}

func (s *ConversationStore) DeleteConversation(ctx context.Context, conversationID, userID, orgNodeID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("conversation_id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
			Delete(&ConversationMessage{}).Error; err != nil {
			return err
		}

		result := tx.
			Where("id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
			Delete(&Conversation{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s *ConversationStore) AppendMessage(ctx context.Context, conversationID, userID, orgNodeID, role, content string) (*ConversationMessage, error) {
	message := &ConversationMessage{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		OrgNodeID:      orgNodeID,
		UserID:         userID,
		Role:           role,
		Content:        content,
	}
	now := time.Now()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation Conversation
		if err := tx.
			Where("id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
			First(&conversation).Error; err != nil {
			return err
		}

		if err := tx.Create(message).Error; err != nil {
			return err
		}

		return tx.
			Model(&Conversation{}).
			Where("id = ? AND org_node_id = ? AND user_id = ?", conversationID, orgNodeID, userID).
			Updates(map[string]any{
				"message_count":        gorm.Expr("message_count + ?", 1),
				"last_message_at":      now,
				"last_message_preview": truncatePreview(content),
				"updated_at":           now,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}

func truncatePreview(content string) string {
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) <= 80 {
		return content
	}
	return string(runes[:80]) + "..."
}
