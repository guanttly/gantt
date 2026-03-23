package service

import (
	"context"
	"time"
)

// IRuleMigrationService 规则迁移服务接口
type IRuleMigrationService interface {
	// PreviewMigration 预览迁移（分析 V3 规则，生成迁移计划）
	PreviewMigration(ctx context.Context, orgID string) (*MigrationPreview, error)

	// ExecuteMigration 执行迁移（将 V3 规则迁移为 V4 规则）
	ExecuteMigration(ctx context.Context, orgID string, plan *MigrationPlan) (*MigrationResult, error)

	// RollbackMigration 回滚迁移（恢复 V3 规则）
	RollbackMigration(ctx context.Context, orgID string, migrationID string) error

	// GetMigrationStatus 获取迁移状态
	GetMigrationStatus(ctx context.Context, orgID string, migrationID string) (*MigrationStatus, error)
}

// MigrationPreview 迁移预览
type MigrationPreview struct {
	TotalV3Rules      int                    `json:"totalV3Rules"`      // V3 规则总数
	MigratableRules   []*MigratableRule      `json:"migratableRules"`   // 可迁移的规则列表
	UnmigratableRules []*UnmigratableRule    `json:"unmigratableRules"` // 无法迁移的规则列表
	EstimatedTime     time.Duration          `json:"estimatedTime"`     // 预计迁移时间
	Warnings          []string               `json:"warnings"`          // 警告信息
}

// MigratableRule 可迁移的规则
type MigratableRule struct {
	RuleID      string            `json:"ruleId"`
	RuleName    string            `json:"ruleName"`
	RuleType    string            `json:"ruleType"`
	ApplyScope  string            `json:"applyScope"`
	TimeScope   string            `json:"timeScope"`
	// 迁移后的 V4 字段
	Category    string            `json:"category"`
	SubCategory string            `json:"subCategory"`
	SourceType  string            `json:"sourceType"` // 迁移后为 "migrated"
	Version     string            `json:"version"`     // 迁移后为 "v4"
	// 迁移建议
	Suggestions []string          `json:"suggestions"` // 迁移建议
}

// UnmigratableRule 无法迁移的规则
type UnmigratableRule struct {
	RuleID    string   `json:"ruleId"`
	RuleName  string   `json:"ruleName"`
	Reason    string   `json:"reason"`    // 无法迁移的原因
	Suggestions []string `json:"suggestions"` // 处理建议
}

// MigrationPlan 迁移计划
type MigrationPlan struct {
	OrgID           string   `json:"orgId"`
	RuleIDs         []string `json:"ruleIds"`         // 要迁移的规则ID列表
	AutoClassify    bool     `json:"autoClassify"`    // 是否自动分类
	FillDefaults    bool     `json:"fillDefaults"`    // 是否填充默认值
	DryRun          bool     `json:"dryRun"`          // 是否仅预览（不实际执行）
}

// MigrationResult 迁移结果
type MigrationResult struct {
	MigrationID      string                `json:"migrationId"`
	TotalRules       int                   `json:"totalRules"`
	SuccessCount     int                   `json:"successCount"`
	FailedCount      int                   `json:"failedCount"`
	FailedRules      []*FailedMigrationRule `json:"failedRules"`
	CreatedAt        time.Time             `json:"createdAt"`
}

// FailedMigrationRule 迁移失败的规则
type FailedMigrationRule struct {
	RuleID  string `json:"ruleId"`
	RuleName string `json:"ruleName"`
	Error   string `json:"error"`
}

// MigrationStatus 迁移状态
type MigrationStatus struct {
	MigrationID  string    `json:"migrationId"`
	Status       string    `json:"status"`       // pending/running/completed/failed/rolled_back
	Progress     float64   `json:"progress"`     // 进度 (0.0-1.0)
	TotalRules   int       `json:"totalRules"`
	ProcessedRules int     `json:"processedRules"`
	SuccessCount int       `json:"successCount"`
	FailedCount  int       `json:"failedCount"`
	StartedAt    time.Time `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}
