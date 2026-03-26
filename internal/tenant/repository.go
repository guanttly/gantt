package tenant

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 组织树数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建组织树仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建组织节点。
func (r *Repository) Create(ctx context.Context, node *OrgNode) error {
	if node.ID == "" {
		node.ID = uuid.New().String()
	}
	return r.db.WithContext(SkipTenantGuard(ctx)).Create(node).Error
}

// GetByID 根据 ID 查询节点。
func (r *Repository) GetByID(ctx context.Context, id string) (*OrgNode, error) {
	var node OrgNode
	err := r.db.WithContext(SkipTenantGuard(ctx)).Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetByParentAndCode 根据父节点 ID 和 code 查询节点（用于唯一性检查）。
func (r *Repository) GetByParentAndCode(ctx context.Context, parentID *string, code string) (*OrgNode, error) {
	var node OrgNode
	tx := r.db.WithContext(SkipTenantGuard(ctx))
	if parentID == nil {
		tx = tx.Where("parent_id IS NULL AND code = ?", code)
	} else {
		tx = tx.Where("parent_id = ? AND code = ?", *parentID, code)
	}
	result := tx.Limit(1).Find(&node)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &node, nil
}

// Update 更新组织节点。
func (r *Repository) Update(ctx context.Context, node *OrgNode) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).Save(node).Error
}

// GetChildren 获取直接子节点列表。
func (r *Repository) GetChildren(ctx context.Context, parentID string) ([]OrgNode, error) {
	var nodes []OrgNode
	err := r.db.WithContext(SkipTenantGuard(ctx)).
		Where("parent_id = ?", parentID).
		Order("code ASC").
		Find(&nodes).Error
	return nodes, err
}

// GetDescendants 获取某节点的所有后代节点（通过物化路径）。
func (r *Repository) GetDescendants(ctx context.Context, nodePath string) ([]OrgNode, error) {
	var nodes []OrgNode
	err := r.db.WithContext(SkipTenantGuard(ctx)).
		Where("path LIKE ?", nodePath+"%").
		Order("depth ASC, code ASC").
		Find(&nodes).Error
	return nodes, err
}

// GetRootNodes 获取所有顶级节点（机构）。
func (r *Repository) GetRootNodes(ctx context.Context) ([]OrgNode, error) {
	var nodes []OrgNode
	err := r.db.WithContext(SkipTenantGuard(ctx)).
		Where("parent_id IS NULL").
		Order("code ASC").
		Find(&nodes).Error
	return nodes, err
}

// UpdateStatusByPath 批量更新某路径下所有节点的状态。
func (r *Repository) UpdateStatusByPath(ctx context.Context, pathPrefix string, status string) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("path LIKE ?", pathPrefix+"%").
		Update("status", status).Error
}

// UpdatePathPrefix 批量更新路径前缀（节点移动时使用）。
func (r *Repository) UpdatePathPrefix(ctx context.Context, oldPrefix, newPrefix string) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("path LIKE ?", oldPrefix+"%").
		Update("path", gorm.Expr("REPLACE(path, ?, ?)", oldPrefix, newPrefix)).Error
}

// GetDescendantIDs 获取某路径下所有活跃节点的 ID 列表。
func (r *Repository) GetDescendantIDs(ctx context.Context, nodePath string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("path LIKE ? AND status = ?", nodePath+"%", StatusActive).
		Pluck("id", &ids).Error
	return ids, err
}

// CountByParent 统计某父节点下的直接子节点数量。
func (r *Repository) CountByParent(ctx context.Context, parentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("parent_id = ?", parentID).
		Count(&count).Error
	return count, err
}

// Delete 删除组织节点（硬删除）。
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).
		Where("id = ?", id).
		Delete(&OrgNode{}).Error
}

// UpdateDepthByPath 批量更新某路径下所有节点的 depth 差值。
func (r *Repository) UpdateDepthByPath(ctx context.Context, pathPrefix string, depthDelta int) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("path LIKE ?", pathPrefix+"%").
		Update("depth", gorm.Expr("depth + ?", depthDelta)).Error
}

// UpdateParent 更新节点的父节点 ID。
func (r *Repository) UpdateParent(ctx context.Context, nodeID string, parentID *string) error {
	return r.db.WithContext(SkipTenantGuard(ctx)).
		Model(&OrgNode{}).
		Where("id = ?", nodeID).
		Update("parent_id", parentID).Error
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&OrgNode{})
}
