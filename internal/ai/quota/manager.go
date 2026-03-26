package quota

import (
	"context"
	"fmt"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/tenant"

	"go.uber.org/zap"
)

// ErrQuotaExceeded 配额超出错误。
var ErrQuotaExceeded = fmt.Errorf("AI 配额已用完")

// Manager 配额管理器。
type Manager struct {
	repo         *Repository
	defaultLimit int
	logger       *zap.Logger
}

// NewManager 创建配额管理器。
func NewManager(repo *Repository, defaultLimit int, logger *zap.Logger) *Manager {
	if defaultLimit <= 0 {
		defaultLimit = 100000
	}
	return &Manager{
		repo:         repo,
		defaultLimit: defaultLimit,
		logger:       logger.Named("quota"),
	}
}

// CheckAndDeduct 检查配额并扣减 token 用量。
func (m *Manager) CheckAndDeduct(ctx context.Context, orgNodeID string, provider string, usage ai.TokenUsage) error {
	quota, err := m.repo.GetOrCreateQuota(ctx, orgNodeID, provider, m.defaultLimit)
	if err != nil {
		return fmt.Errorf("查询配额失败: %w", err)
	}

	if quota.UsedTokens+usage.TotalTokens > quota.MonthlyLimit {
		m.logger.Warn("AI 配额超限",
			zap.String("org_node_id", orgNodeID),
			zap.String("provider", provider),
			zap.Int("used", quota.UsedTokens),
			zap.Int("limit", quota.MonthlyLimit),
			zap.Int("requested", usage.TotalTokens),
		)
		return ErrQuotaExceeded
	}

	if err := m.repo.IncrementUsage(ctx, quota.ID, usage.TotalTokens); err != nil {
		return fmt.Errorf("扣减配额失败: %w", err)
	}

	return nil
}

// RecordUsage 记录 AI 使用日志。
func (m *Manager) RecordUsage(ctx context.Context, provider, model, purpose string, usage ai.TokenUsage) error {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	userID := "" // TODO: 从 ctx 获取 user_id

	log := &AIUsageLog{
		UserID:           userID,
		Provider:         provider,
		Model:            model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		Purpose:          purpose,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	return m.repo.CreateUsageLog(ctx, log)
}

// GetQuotaStatus 获取配额状态。
func (m *Manager) GetQuotaStatus(ctx context.Context, orgNodeID string) ([]QuotaStatus, error) {
	quotas, err := m.repo.GetQuotaByOrg(ctx, orgNodeID)
	if err != nil {
		return nil, err
	}

	result := make([]QuotaStatus, 0, len(quotas))
	for _, q := range quotas {
		remaining := q.MonthlyLimit - q.UsedTokens
		if remaining < 0 {
			remaining = 0
		}
		var usagePercent float64
		if q.MonthlyLimit > 0 {
			usagePercent = float64(q.UsedTokens) / float64(q.MonthlyLimit) * 100
		}
		result = append(result, QuotaStatus{
			Provider:     q.Provider,
			MonthlyLimit: q.MonthlyLimit,
			UsedTokens:   q.UsedTokens,
			Remaining:    remaining,
			UsagePercent: usagePercent,
			ResetAt:      q.ResetAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

// GetUsageLogs 获取使用记录。
func (m *Manager) GetUsageLogs(ctx context.Context, orgNodeID string, page, size int) ([]AIUsageLog, int64, error) {
	return m.repo.ListUsageLogs(ctx, orgNodeID, page, size)
}
