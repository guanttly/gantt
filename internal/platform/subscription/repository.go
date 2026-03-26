package subscription

import (
	"context"

	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 订阅数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建订阅仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建订阅记录。
func (r *Repository) Create(ctx context.Context, sub *Subscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Create(sub).Error
}

// GetByID 根据 ID 查询订阅。
func (r *Repository) GetByID(ctx context.Context, id string) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetByOrgNode 根据组织节点 ID 查询订阅。
func (r *Repository) GetByOrgNode(ctx context.Context, orgNodeID string) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("org_node_id = ?", orgNodeID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// Update 更新订阅记录。
func (r *Repository) Update(ctx context.Context, sub *Subscription) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).Save(sub).Error
}

// Delete 删除订阅记录。
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("id = ?", id).
		Delete(&Subscription{}).Error
}

// List 分页查询订阅列表。
func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Subscription, int64, error) {
	var subs []Subscription
	var total int64

	tx := r.db.WithContext(tenant.SkipTenantGuard(ctx)).Model(&Subscription{})

	if opts.Plan != "" {
		tx = tx.Where("plan = ?", opts.Plan)
	}
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
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
		Find(&subs).Error
	return subs, total, err
}

// CountEmployees 统计某组织节点下的员工数量。
func (r *Repository) CountEmployees(ctx context.Context, orgNodeID string) (int64, error) {
	var count int64
	err := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Model(&employee.Employee{}).
		Where("org_node_id = ?", orgNodeID).
		Count(&count).Error
	return count, err
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Subscription{})
}

// ListOptions 列表查询选项。
type ListOptions struct {
	Page   int    `json:"page"`
	Size   int    `json:"size"`
	Plan   string `json:"plan"`
	Status string `json:"status"`
}
