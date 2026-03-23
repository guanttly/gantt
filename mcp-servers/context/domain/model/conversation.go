package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONMap 用于 GORM 的 JSON 字段类型（支持任意类型）
type JSONMap map[string]any

// Value 实现 driver.Valuer 接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Conversation 会话模型（扩展字段，支持 workflow context 和未来功能）
type Conversation struct {
	ID        string   `gorm:"primaryKey;type:varchar(64)" json:"id"`
	
	// 元数据（业务层可存储 orgId、userId、scheduleId、dateRange 等）
	Meta      JSONMap  `gorm:"type:json" json:"meta,omitempty"` // map[string]any
	
	// Workflow Context（完整 Session 对象）
	WorkflowContext JSONMap `gorm:"type:json" json:"workflowContext,omitempty"`
	
	// 扩展字段（为未来功能预留）
	CompressionStatus string `gorm:"type:varchar(20);default:'none'" json:"compressionStatus,omitempty"`
	LastCompressedAt  *time.Time `gorm:"index" json:"lastCompressedAt,omitempty"`
	TotalMessages    int       `gorm:"default:0" json:"totalMessages,omitempty"`
	CompressedCount  int       `gorm:"default:0" json:"compressedCount,omitempty"`
	UncompressedCount int      `gorm:"default:0" json:"uncompressedCount,omitempty"`
	
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;index" json:"updatedAt"`
}

// ConversationMessage 会话消息模型（扩展字段，支持未来功能）
type ConversationMessage struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID string    `gorm:"type:varchar(64);index" json:"conversationId"`
	
	// 业务层消息唯一标识（来自 session.Message.ID）
	MessageID      string    `gorm:"type:varchar(64);uniqueIndex:idx_conversation_message_id" json:"messageId,omitempty"`
	
	// 基础字段
	Role           string    `gorm:"type:varchar(20);index" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	Timestamp      time.Time `gorm:"autoCreateTime;index" json:"timestamp"`
	
	// 扩展字段（为未来功能预留）
	IsCompressed   bool      `gorm:"default:false;index" json:"isCompressed,omitempty"`
	CompressedFrom JSONMap   `gorm:"type:json" json:"compressedFrom,omitempty"` // []uint 存储为 JSON
	CompressionVersion int   `gorm:"default:0" json:"compressionVersion,omitempty"`
	EmbeddingID    string    `gorm:"type:varchar(64);index" json:"embeddingId,omitempty"`
	HasEmbedding   bool      `gorm:"default:false;index" json:"hasEmbedding,omitempty"`
	ContentTokens  int       `gorm:"default:0" json:"contentTokens,omitempty"`
	Importance     float64   `gorm:"default:0.0" json:"importance,omitempty"`
	Metadata       JSONMap   `gorm:"type:json" json:"metadata,omitempty"`
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}

// TableName 指定表名
func (ConversationMessage) TableName() string {
	return "conversation_messages"
}
