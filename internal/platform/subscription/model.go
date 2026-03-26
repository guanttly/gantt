package subscription

import (
	"time"
)

// 订阅套餐类型。
const (
	PlanFree     = "free"
	PlanStandard = "standard"
	PlanPremium  = "premium"
)

// 订阅状态。
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusExpired   = "expired"
)

// PlanDefaults 套餐默认配额定义。
var PlanDefaults = map[string]PlanQuota{
	PlanFree:     {MaxEmployees: 20, MaxAITokens: 10000},
	PlanStandard: {MaxEmployees: 200, MaxAITokens: 100000},
	PlanPremium:  {MaxEmployees: 0, MaxAITokens: 500000}, // 0 = 不限
}

// PlanQuota 套餐配额。
type PlanQuota struct {
	MaxEmployees int
	MaxAITokens  int
}

// Subscription 订阅模型。
type Subscription struct {
	ID           string     `gorm:"primaryKey;size:64" json:"id"`
	OrgNodeID    string     `gorm:"size:64;not null;uniqueIndex:uk_sub_org" json:"org_node_id"`
	Plan         string     `gorm:"size:32;not null;default:free" json:"plan"`
	Status       string     `gorm:"size:16;not null;default:active" json:"status"`
	MaxEmployees int        `gorm:"not null;default:20" json:"max_employees"`
	MaxAITokens  int        `gorm:"not null;default:10000" json:"max_ai_tokens"`
	StartDate    time.Time  `gorm:"type:date;not null" json:"start_date"`
	EndDate      *time.Time `gorm:"type:date" json:"end_date"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名。
func (Subscription) TableName() string {
	return "subscriptions"
}

// IsActive 订阅是否有效。
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive
}

// IsUnlimitedEmployees 员工配额是否不限制。
func (s *Subscription) IsUnlimitedEmployees() bool {
	return s.MaxEmployees == 0
}
