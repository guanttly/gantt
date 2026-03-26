package audit

import (
	"context"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 审计日志数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建审计日志仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建审计日志记录。
func (r *Repository) Create(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(log).Error
}

// List 分页查询审计日志。
func (r *Repository) List(ctx context.Context, opts ListOptions) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	tx := r.db.WithContext(tenant.SkipTenantGuard(ctx)).Model(&AuditLog{})

	if opts.OrgNodeID != "" {
		tx = tx.Where("org_node_id = ?", opts.OrgNodeID)
	}
	if opts.UserID != "" {
		tx = tx.Where("user_id = ?", opts.UserID)
	}
	if opts.Action != "" {
		tx = tx.Where("action = ?", opts.Action)
	}
	if opts.ResourceType != "" {
		tx = tx.Where("resource_type = ?", opts.ResourceType)
	}
	if !opts.StartTime.IsZero() {
		tx = tx.Where("created_at >= ?", opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		tx = tx.Where("created_at <= ?", opts.EndTime)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (opts.Page - 1) * opts.Size
	if offset < 0 {
		offset = 0
	}
	err := tx.Order("created_at DESC").
		Offset(offset).
		Limit(opts.Size).
		Find(&logs).Error
	return logs, total, err
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&AuditLog{})
}

// ListOptions 审计日志列表查询选项。
type ListOptions struct {
	Page         int       `json:"page"`
	Size         int       `json:"size"`
	OrgNodeID    string    `json:"org_node_id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
}
