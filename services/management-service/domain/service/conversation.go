package service

import (
	"context"
	"time"
)

// IConversationService 对话记录服务接口（management-service 层）
type IConversationService interface {
	// ListScheduleConversations 查询与排班相关的对话记录
	// 支持按 orgId、userId、日期范围、工作流状态等查询
	ListScheduleConversations(ctx context.Context, filter *ScheduleConversationFilter) ([]*ConversationSummary, error)

	// CreateOrUpdateConversation 创建或更新对话记录（由排班智能体调用）
	CreateOrUpdateConversation(ctx context.Context, req *CreateOrUpdateConversationRequest) error
}

// ConversationEntity 对话记录实体（用于 Repository 层）
type ConversationEntity struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"orgId"`
	UserID         string    `json:"userId"`
	Title          string    `json:"title"`
	WorkflowType   string    `json:"workflowType"`
	ConversationID string    `json:"conversationId"`
	CreatedAt      time.Time `json:"createdAt"`
	LastMessageAt  time.Time `json:"lastMessageAt"`
	MessageCount   int       `json:"messageCount"`
	ScheduleStartDate string  `json:"scheduleStartDate,omitempty"`
	ScheduleEndDate   string  `json:"scheduleEndDate,omitempty"`
	ScheduleID        string  `json:"scheduleId,omitempty"`
	ScheduleStatus    string  `json:"scheduleStatus,omitempty"`
}

// CreateOrUpdateConversationRequest 创建或更新对话记录请求
type CreateOrUpdateConversationRequest struct {
	ConversationID string                 `json:"conversationId"` // context-server 的 conversation ID
	OrgID          string                 `json:"orgId"`
	UserID         string                 `json:"userId"`
	WorkflowType   string                 `json:"workflowType"`
	LastMessageAt  time.Time              `json:"lastMessageAt"`
	MessageCount   int                    `json:"messageCount"`
	Meta           map[string]any         `json:"meta,omitempty"` // 从 context-server 的 meta 中提取的信息
}

// ScheduleConversationFilter 排班对话记录查询过滤器
type ScheduleConversationFilter struct {
	OrgID        string   `json:"orgId,omitempty"`
	UserID       string   `json:"userId,omitempty"`
	StartDate    string   `json:"startDate,omitempty"` // 排班开始日期
	EndDate      string   `json:"endDate,omitempty"`   // 排班结束日期
	WorkflowType string   `json:"workflowType,omitempty"` // "schedule.create", "schedule.adjust"
	Status       string   `json:"status,omitempty"` // "in_progress", "completed", "cancelled"
	Limit        int      `json:"limit,omitempty"`
}

// ConversationSummary 对话记录摘要
type ConversationSummary struct {
	ID             string `json:"id"`              // 管理服务的内部ID
	ConversationID string `json:"conversationId"`  // context-server 的 conversation ID
	Title          string `json:"title,omitempty"`
	LastMessageAt  string `json:"lastMessageAt,omitempty"`
	MessageCount   int    `json:"messageCount,omitempty"`
	OrgID          string `json:"orgId,omitempty"`
	UserID         string `json:"userId,omitempty"`
	
	// 排班相关信息
	ScheduleStartDate string   `json:"scheduleStartDate,omitempty"`
	ScheduleEndDate   string   `json:"scheduleEndDate,omitempty"`
	ScheduleShiftIds  []string `json:"scheduleShiftIds,omitempty"`
	ScheduleID        string   `json:"scheduleId,omitempty"`
	ScheduleStatus    string   `json:"scheduleStatus,omitempty"`
	WorkflowType      string   `json:"workflowType,omitempty"`
}

// ScheduleWorkflowContext 排班工作流上下文
type ScheduleWorkflowContext struct {
	ConversationID string                 `json:"conversationId"`
	WorkflowMeta   map[string]any         `json:"workflowMeta"`
	ScheduleContext map[string]any        `json:"scheduleContext"` // ScheduleCreateContext 或 ScheduleAdjustContext
	Messages       []ConversationMessage  `json:"messages,omitempty"`
}

// ConversationMessage 对话消息
type ConversationMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
}
