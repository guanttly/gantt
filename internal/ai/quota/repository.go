package quota

import (
	"context"
	"time"

	"gantt-saas/internal/tenant"

	"gorm.io/gorm"
)

// Repository 配额数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建配额 Repository。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetOrCreateQuota 获取或创建配额记录。
func (r *Repository) GetOrCreateQuota(ctx context.Context, orgNodeID, provider string, defaultLimit int) (*AIQuota, error) {
	var quota AIQuota
	err := r.db.WithContext(ctx).
		Where("org_node_id = ? AND provider = ?", orgNodeID, provider).
		First(&quota).Error

	if err == gorm.ErrRecordNotFound {
		now := time.Now()
		resetAt := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		quota = AIQuota{
			Provider:     provider,
			MonthlyLimit: defaultLimit,
			UsedTokens:   0,
			ResetAt:      resetAt,
			TenantModel: tenant.TenantModel{
				OrgNodeID: orgNodeID,
			},
		}
		if err := r.db.WithContext(ctx).Create(&quota).Error; err != nil {
			return nil, err
		}
		return &quota, nil
	}

	if err != nil {
		return nil, err
	}

	// 检查是否需要重置
	if time.Now().After(quota.ResetAt) {
		now := time.Now()
		quota.UsedTokens = 0
		quota.ResetAt = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		r.db.WithContext(ctx).Save(&quota)
	}

	return &quota, nil
}

// IncrementUsage 原子递增 used_tokens。
func (r *Repository) IncrementUsage(ctx context.Context, quotaID uint64, tokens int) error {
	return r.db.WithContext(ctx).
		Model(&AIQuota{}).
		Where("id = ?", quotaID).
		Update("used_tokens", gorm.Expr("used_tokens + ?", tokens)).
		Error
}

// CreateUsageLog 记录使用日志。
func (r *Repository) CreateUsageLog(ctx context.Context, log *AIUsageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListUsageLogs 查询使用记录。
func (r *Repository) ListUsageLogs(ctx context.Context, orgNodeID string, page, size int) ([]AIUsageLog, int64, error) {
	var logs []AIUsageLog
	var total int64

	query := r.db.WithContext(ctx).
		Where("org_node_id = ?", orgNodeID).
		Order("created_at DESC")

	query.Model(&AIUsageLog{}).Count(&total)

	offset := (page - 1) * size
	err := query.Offset(offset).Limit(size).Find(&logs).Error
	return logs, total, err
}

// GetQuotaByOrg 查询组织配额。
func (r *Repository) GetQuotaByOrg(ctx context.Context, orgNodeID string) ([]AIQuota, error) {
	var quotas []AIQuota
	err := r.db.WithContext(ctx).
		Where("org_node_id = ?", orgNodeID).
		Find(&quotas).Error
	return quotas, err
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&AIQuota{}, &AIUsageLog{})
}
