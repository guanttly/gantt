package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"jusha/gantt/service/management/domain/service"
	"jusha/gantt/service/management/internal/entity"

	domain_repo "jusha/gantt/service/management/domain/repository"
)

// ConversationRepository 对话记录仓储实现
type ConversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository 创建对话记录仓储
func NewConversationRepository(db *gorm.DB) domain_repo.IConversationRepository {
	return &ConversationRepository{db: db}
}

// Create 创建对话记录
func (r *ConversationRepository) Create(ctx context.Context, conversation *service.ConversationEntity) error {
	entity := &entity.ConversationEntity{
		ID:             conversation.ID,
		OrgID:          conversation.OrgID,
		UserID:         conversation.UserID,
		Title:          conversation.Title,
		WorkflowType:   conversation.WorkflowType,
		ConversationID: conversation.ConversationID,
		CreatedAt:      conversation.CreatedAt,
		LastMessageAt:  conversation.LastMessageAt,
		MessageCount:   conversation.MessageCount,
	}

	if conversation.ScheduleStartDate != "" {
		entity.ScheduleStartDate = &conversation.ScheduleStartDate
	}
	if conversation.ScheduleEndDate != "" {
		entity.ScheduleEndDate = &conversation.ScheduleEndDate
	}
	if conversation.ScheduleID != "" {
		entity.ScheduleID = &conversation.ScheduleID
	}
	if conversation.ScheduleStatus != "" {
		entity.ScheduleStatus = &conversation.ScheduleStatus
	}

	return r.db.WithContext(ctx).Create(entity).Error
}

// Update 更新对话记录
func (r *ConversationRepository) Update(ctx context.Context, id string, updates map[string]any) error {
	updateData := make(map[string]any)
	for k, v := range updates {
		updateData[k] = v
	}
	updateData["last_message_at"] = time.Now()

	return r.db.WithContext(ctx).
		Model(&entity.ConversationEntity{}).
		Where("id = ?", id).
		Updates(updateData).Error
}

// List 查询对话列表
func (r *ConversationRepository) List(ctx context.Context, filter *service.ScheduleConversationFilter) ([]*service.ConversationEntity, error) {
	var entities []*entity.ConversationEntity
	query := r.db.WithContext(ctx).Model(&entity.ConversationEntity{})

	// 应用过滤条件
	if filter.OrgID != "" {
		query = query.Where("org_id = ?", filter.OrgID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.WorkflowType != "" {
		query = query.Where("workflow_type = ?", filter.WorkflowType)
	}
	if filter.StartDate != "" {
		query = query.Where("schedule_start_date >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("schedule_end_date <= ?", filter.EndDate)
	}
	if filter.Status != "" {
		query = query.Where("schedule_status = ?", filter.Status)
	}

	// 排序：按最后消息时间倒序
	query = query.Order("last_message_at DESC")

	// 分页
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	query = query.Limit(limit)

	if err := query.Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	// 转换为 domain model
	result := make([]*service.ConversationEntity, 0, len(entities))
	for _, e := range entities {
		conv := &service.ConversationEntity{
			ID:             e.ID,
			OrgID:          e.OrgID,
			UserID:         e.UserID,
			Title:          e.Title,
			WorkflowType:   e.WorkflowType,
			ConversationID: e.ConversationID,
			CreatedAt:      e.CreatedAt,
			LastMessageAt:  e.LastMessageAt,
			MessageCount:   e.MessageCount,
		}

		if e.ScheduleStartDate != nil {
			conv.ScheduleStartDate = *e.ScheduleStartDate
		}
		if e.ScheduleEndDate != nil {
			conv.ScheduleEndDate = *e.ScheduleEndDate
		}
		if e.ScheduleID != nil {
			conv.ScheduleID = *e.ScheduleID
		}
		if e.ScheduleStatus != nil {
			conv.ScheduleStatus = *e.ScheduleStatus
		}

		result = append(result, conv)
	}

	return result, nil
}

// Get 获取单个对话
func (r *ConversationRepository) Get(ctx context.Context, id string) (*service.ConversationEntity, error) {
	var e entity.ConversationEntity
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	conv := &service.ConversationEntity{
		ID:             e.ID,
		OrgID:          e.OrgID,
		UserID:         e.UserID,
		Title:          e.Title,
		WorkflowType:   e.WorkflowType,
		ConversationID: e.ConversationID,
		CreatedAt:      e.CreatedAt,
		LastMessageAt:  e.LastMessageAt,
		MessageCount:   e.MessageCount,
	}

	if e.ScheduleStartDate != nil {
		conv.ScheduleStartDate = *e.ScheduleStartDate
	}
	if e.ScheduleEndDate != nil {
		conv.ScheduleEndDate = *e.ScheduleEndDate
	}
	if e.ScheduleID != nil {
		conv.ScheduleID = *e.ScheduleID
	}
	if e.ScheduleStatus != nil {
		conv.ScheduleStatus = *e.ScheduleStatus
	}

	return conv, nil
}

// GetByConversationID 通过 conversationID 查询
func (r *ConversationRepository) GetByConversationID(ctx context.Context, conversationID string) (*service.ConversationEntity, error) {
	var e entity.ConversationEntity
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
		return nil, fmt.Errorf("get conversation by conversationID: %w", err)
	}

	conv := &service.ConversationEntity{
		ID:             e.ID,
		OrgID:          e.OrgID,
		UserID:         e.UserID,
		Title:          e.Title,
		WorkflowType:   e.WorkflowType,
		ConversationID: e.ConversationID,
		CreatedAt:      e.CreatedAt,
		LastMessageAt:  e.LastMessageAt,
		MessageCount:   e.MessageCount,
	}

	if e.ScheduleStartDate != nil {
		conv.ScheduleStartDate = *e.ScheduleStartDate
	}
	if e.ScheduleEndDate != nil {
		conv.ScheduleEndDate = *e.ScheduleEndDate
	}
	if e.ScheduleID != nil {
		conv.ScheduleID = *e.ScheduleID
	}
	if e.ScheduleStatus != nil {
		conv.ScheduleStatus = *e.ScheduleStatus
	}

	return conv, nil
}
