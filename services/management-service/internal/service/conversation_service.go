package service

import (
	"context"
	"fmt"
	"time"

	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

type conversationService struct {
	logger           logging.ILogger
	conversationRepo repository.IConversationRepository
}

func NewConversationService(
	conversationRepo repository.IConversationRepository,
	logger logging.ILogger,
) service.IConversationService {
	return &conversationService{
		logger:           logger.With("component", "ConversationService"),
		conversationRepo: conversationRepo,
	}
}

// ListScheduleConversations 查询与排班相关的对话记录
func (s *conversationService) ListScheduleConversations(ctx context.Context, filter *service.ScheduleConversationFilter) ([]*service.ConversationSummary, error) {
	// 从自己的数据库查询
	entities, err := s.conversationRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list conversations failed: %w", err)
	}

	// 转换为 ConversationSummary
	summaries := make([]*service.ConversationSummary, 0, len(entities))
	for _, entity := range entities {
		summary := &service.ConversationSummary{
			ID:             entity.ID,
			ConversationID: entity.ConversationID, // context-server 的 conversation ID
			Title:          entity.Title,
			LastMessageAt:  entity.LastMessageAt.Format(time.RFC3339),
			MessageCount:   entity.MessageCount,
			OrgID:          entity.OrgID,
			UserID:         entity.UserID,
			WorkflowType:   entity.WorkflowType,
		}

		if entity.ScheduleStartDate != "" {
			summary.ScheduleStartDate = entity.ScheduleStartDate
		}
		if entity.ScheduleEndDate != "" {
			summary.ScheduleEndDate = entity.ScheduleEndDate
		}
		if entity.ScheduleID != "" {
			summary.ScheduleID = entity.ScheduleID
		}
		if entity.ScheduleStatus != "" {
			summary.ScheduleStatus = entity.ScheduleStatus
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// CreateOrUpdateConversation 创建或更新对话记录（由排班智能体调用）
func (s *conversationService) CreateOrUpdateConversation(ctx context.Context, req *service.CreateOrUpdateConversationRequest) error {
	// 尝试通过 conversationID 查找现有记录
	existing, err := s.conversationRepo.GetByConversationID(ctx, req.ConversationID)
	if err != nil {
		// 如果不存在，创建新记录
		now := time.Now()
		if req.LastMessageAt.IsZero() {
			req.LastMessageAt = now
		}

		// 生成标题
		title := s.generateTitle(req.LastMessageAt, req.WorkflowType)

		// 从 meta 中提取排班相关信息
		scheduleStartDate := ""
		scheduleEndDate := ""
		scheduleID := ""
		scheduleStatus := ""
		if req.Meta != nil {
			// 添加日志，追踪meta信息接收
			s.logger.Info("Extracting schedule info from meta",
				"conversationID", req.ConversationID,
				"metaKeys", func() []string {
					keys := make([]string, 0, len(req.Meta))
					for k := range req.Meta {
						keys = append(keys, k)
					}
					return keys
				}(),
				"rawMeta", req.Meta)

			if val, ok := req.Meta["scheduleStartDate"].(string); ok {
				scheduleStartDate = val
			}
			if val, ok := req.Meta["scheduleEndDate"].(string); ok {
				scheduleEndDate = val
			}
			if val, ok := req.Meta["scheduleId"].(string); ok {
				scheduleID = val
			}
			if val, ok := req.Meta["scheduleStatus"].(string); ok {
				scheduleStatus = val
			}

			// 记录提取结果
			s.logger.Info("Extracted schedule info",
				"conversationID", req.ConversationID,
				"scheduleStartDate", scheduleStartDate,
				"scheduleEndDate", scheduleEndDate,
				"scheduleID", scheduleID,
				"scheduleStatus", scheduleStatus)
		} else {
			s.logger.Warn("Meta is nil when creating conversation",
				"conversationID", req.ConversationID)
		}

		entity := &service.ConversationEntity{
			ID:                uuid.NewString(),
			OrgID:             req.OrgID,
			UserID:            req.UserID,
			Title:             title,
			WorkflowType:      req.WorkflowType,
			ConversationID:    req.ConversationID,
			CreatedAt:         now,
			LastMessageAt:     req.LastMessageAt,
			MessageCount:      req.MessageCount,
			ScheduleStartDate: scheduleStartDate,
			ScheduleEndDate:   scheduleEndDate,
			ScheduleID:        scheduleID,
			ScheduleStatus:    scheduleStatus,
		}

		return s.conversationRepo.Create(ctx, entity)
	}

	// 如果存在，更新记录
	updates := make(map[string]any)
	updates["last_message_at"] = req.LastMessageAt
	updates["message_count"] = req.MessageCount

	// 重新生成标题（使用最新的时间）
	updates["title"] = s.generateTitle(req.LastMessageAt, req.WorkflowType)

	// 更新排班相关信息（即使值为空也要更新，确保状态同步）
	if req.Meta != nil {
		// 添加日志，追踪更新时的meta信息
		s.logger.Info("Updating conversation with meta",
			"conversationID", req.ConversationID,
			"existingID", existing.ID,
			"metaKeys", func() []string {
				keys := make([]string, 0, len(req.Meta))
				for k := range req.Meta {
					keys = append(keys, k)
				}
				return keys
			}(),
			"rawMeta", req.Meta)

		// 排班周期信息：如果meta中有值就更新，即使为空字符串也要更新（允许清空）
		if val, ok := req.Meta["scheduleStartDate"].(string); ok {
			updates["schedule_start_date"] = val
			s.logger.Debug("Updating scheduleStartDate", "value", val)
		}
		if val, ok := req.Meta["scheduleEndDate"].(string); ok {
			updates["schedule_end_date"] = val
			s.logger.Debug("Updating scheduleEndDate", "value", val)
		}
		// 排班ID和状态：如果meta中有值就更新
		if val, ok := req.Meta["scheduleId"].(string); ok {
			updates["schedule_id"] = val
			s.logger.Debug("Updating scheduleId", "value", val)
		}
		if val, ok := req.Meta["scheduleStatus"].(string); ok {
			updates["schedule_status"] = val
			s.logger.Debug("Updating scheduleStatus", "value", val)
		}

		// 记录更新结果
		s.logger.Info("Conversation update prepared",
			"conversationID", req.ConversationID,
			"updateKeys", func() []string {
				keys := make([]string, 0, len(updates))
				for k := range updates {
					keys = append(keys, k)
				}
				return keys
			}())
	} else {
		s.logger.Warn("Meta is nil when updating conversation",
			"conversationID", req.ConversationID,
			"existingID", existing.ID)
	}

	return s.conversationRepo.Update(ctx, existing.ID, updates)
}

// generateTitle 生成标题：格式为"{日期}的{工作流类型}"
func (s *conversationService) generateTitle(t time.Time, workflowType string) string {
	// 格式化日期为"12月30日"
	dateStr := t.Format("1月2日")

	// 工作流类型映射
	workflowTypeMap := map[string]string{
		"schedule.create": "创建排班",
		"schedule.adjust": "调整排班",
	}

	workflowName := workflowType
	if name, ok := workflowTypeMap[workflowType]; ok {
		workflowName = name
	}

	return fmt.Sprintf("%s的%s", dateStr, workflowName)
}
