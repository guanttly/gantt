package model

// ConversationRecord 会话记录
type ConversationRecord struct {
	ID            string                 `json:"id"`
	Meta          map[string]any          `json:"meta,omitempty"`
	UpdatedAt     string                 `json:"updatedAt,omitempty"`     // 最后更新时间（ISO 8601 格式）
	TotalMessages int                    `json:"totalMessages,omitempty"`  // 消息总数
}

// ConversationMessage 会话消息（SDK 层，支持扩展字段）
type ConversationMessage struct {
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	Timestamp      string                 `json:"timestamp,omitempty"`
	MessageID      string                 `json:"messageId,omitempty"` // 业务层消息唯一标识
	
	// 扩展字段（可选）
	IsCompressed   bool                   `json:"isCompressed,omitempty"`
	CompressedFrom []uint                 `json:"compressedFrom,omitempty"`
	HasEmbedding   bool                   `json:"hasEmbedding,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
}

// Conversation New
type ConversationNewRequest struct {
	Meta map[string]any `json:"meta,omitempty"`
}

type ConversationNewResponse struct {
	ConversationRecord
}

// Conversation Append（支持扩展字段）
type ConversationAppendRequest struct {
	ID            string                 `json:"id"`            // Conversation ID
	MessageID     string                 `json:"messageId,omitempty"` // 业务层消息唯一标识（来自 session.Message.ID）
	Role          string                 `json:"role"`
	Content       string                 `json:"content"`
	
	// 可选扩展字段
	Metadata      map[string]any         `json:"metadata,omitempty"`
	Importance    float64                `json:"importance,omitempty"`
	ContentTokens int                    `json:"contentTokens,omitempty"`
}

type ConversationAppendResponse struct {
	ConversationMessage
}

// Conversation History
type ConversationHistoryRequest struct {
	ID    string `json:"id"`
	Limit int    `json:"limit,omitempty"`
}

type ConversationHistoryResponse struct {
	Messages []ConversationMessage `json:"messages"`
}

// Conversation List（按 Meta 字段查询）
type ConversationListRequest struct {
	MetaFilters map[string]any `json:"metaFilters,omitempty"` // 例如: {"orgId": "xxx", "userId": "yyy"}
	Limit       int            `json:"limit,omitempty"`
	Offset      int            `json:"offset,omitempty"`
}

type ConversationListResponse struct {
	Conversations []ConversationRecord `json:"conversations"`
	Total         int                  `json:"total"`
}

// Workflow Context Update
type WorkflowContextUpdateRequest struct {
	ConversationID string                 `json:"conversationId"`
	Context        map[string]any         `json:"context"`
}

// Workflow Context Get
type WorkflowContextGetRequest struct {
	ConversationID string `json:"conversationId"`
}

type WorkflowContextGetResponse struct {
	Context map[string]any `json:"context"`
}

// Update Conversation Meta
type UpdateConversationMetaRequest struct {
	ConversationID string                 `json:"conversationId"`
	MetaUpdates    map[string]any         `json:"metaUpdates"`
}
