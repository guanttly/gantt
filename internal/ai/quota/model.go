package quota

import (
	"time"

	"gantt-saas/internal/tenant"
)

// AIQuota AI 配额模型。
type AIQuota struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider     string    `gorm:"size:32;not null;default:openai" json:"provider"`
	MonthlyLimit int       `gorm:"not null;default:100000" json:"monthly_limit"`
	UsedTokens   int       `gorm:"not null;default:0" json:"used_tokens"`
	ResetAt      time.Time `gorm:"not null" json:"reset_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	tenant.TenantModel
}

func (AIQuota) TableName() string { return "ai_quotas" }

// AIUsageLog AI 使用记录。
type AIUsageLog struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           string    `gorm:"size:64;not null" json:"user_id"`
	Provider         string    `gorm:"size:32;not null" json:"provider"`
	Model            string    `gorm:"size:64;not null" json:"model"`
	PromptTokens     int       `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens int       `gorm:"not null;default:0" json:"completion_tokens"`
	Purpose          string    `gorm:"size:64;not null" json:"purpose"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	tenant.TenantModel
}

func (AIUsageLog) TableName() string { return "ai_usage_logs" }

// QuotaStatus 配额状态（API 返回）。
type QuotaStatus struct {
	Provider     string  `json:"provider"`
	MonthlyLimit int     `json:"monthly_limit"`
	UsedTokens   int     `json:"used_tokens"`
	Remaining    int     `json:"remaining"`
	UsagePercent float64 `json:"usage_percent"`
	ResetAt      string  `json:"reset_at"`
}

// UsageRecord 使用记录（API 返回）。
type UsageRecord struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Purpose          string `json:"purpose"`
	CreatedAt        string `json:"created_at"`
}
