package entity

import (
	"time"
)

// ConversationEntity 对话记录数据库实体（对应 conversations 表）
// 管理服务只存储基本信息和会话ID，完整数据存储在 context-server
type ConversationEntity struct {
	ID        string     `gorm:"primaryKey;type:varchar(64)" json:"id"`
	OrgID     string     `gorm:"index;type:varchar(64);not null" json:"orgId"`
	UserID    string     `gorm:"index;type:varchar(64);not null" json:"userId"`
	Title     string     `gorm:"type:varchar(255)" json:"title"` // 自动生成的标题，如"12月30日的创建排班"
	WorkflowType string  `gorm:"type:varchar(50);index" json:"workflowType"` // 工作流类型，如"schedule.create"
	ConversationID string `gorm:"type:varchar(64);uniqueIndex;not null" json:"conversationId"` // context-server 的 conversation ID
	CreatedAt time.Time  `gorm:"autoCreateTime;index" json:"createdAt"` // 创建时间
	LastMessageAt time.Time `gorm:"index" json:"lastMessageAt"` // 最后聊天时间
	MessageCount int       `gorm:"default:0" json:"messageCount"` // 消息数量
	
	// 排班相关信息（可选）
	ScheduleStartDate *string `gorm:"type:varchar(20)" json:"scheduleStartDate,omitempty"` // 排班开始日期
	ScheduleEndDate   *string `gorm:"type:varchar(20)" json:"scheduleEndDate,omitempty"`   // 排班结束日期
	ScheduleID        *string `gorm:"type:varchar(64)" json:"scheduleId,omitempty"`        // 排班ID
	ScheduleStatus    *string `gorm:"type:varchar(20)" json:"scheduleStatus,omitempty"`     // 排班状态
}

// TableName 表名
func (ConversationEntity) TableName() string {
	return "conversations"
}
