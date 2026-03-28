package employee

import (
	"context"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 员工数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建员工仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建员工。
func (r *Repository) Create(ctx context.Context, emp *Employee) error {
	if emp.ID == "" {
		emp.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(emp).Error
}

// GetByID 根据 ID 查询员工。
func (r *Repository) GetByID(ctx context.Context, id string) (*Employee, error) {
	var emp Employee
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		First(&emp).Error
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

// Update 更新员工。
func (r *Repository) Update(ctx context.Context, emp *Employee) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(emp).Error
}

// Delete 删除员工（硬删除）。
func (r *Repository) Delete(ctx context.Context, id string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		Delete(&Employee{}).Error
}

// List 分页查询员工列表。
func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Employee, int64, error) {
	var employees []Employee
	var total int64

	tx := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Model(&Employee{})
	if opts.PrioritizeAdmins {
		adminRoleSubQuery := r.db.WithContext(tenant.SkipTenantGuard(ctx)).
			Table("employee_app_roles").
			Select("DISTINCT employee_id").
			Where("app_role = ?", "app:schedule_admin")
		tx = tx.Joins("LEFT JOIN (?) AS admin_roles ON admin_roles.employee_id = employees.id", adminRoleSubQuery)
	}

	// 搜索过滤
	if opts.Keyword != "" {
		keyword := "%" + opts.Keyword + "%"
		tx = tx.Where("name LIKE ? OR employee_no LIKE ? OR phone LIKE ?", keyword, keyword, keyword)
	}
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.Position != "" {
		tx = tx.Where("position = ?", opts.Position)
	}
	if opts.Category != "" {
		tx = tx.Where("category = ?", opts.Category)
	}

	// 计算总数
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (opts.Page - 1) * opts.Size
	if offset < 0 {
		offset = 0
	}
	ordered := tx
	if opts.PrioritizeAdmins {
		ordered = ordered.
			Order("CASE WHEN admin_roles.employee_id IS NULL THEN 1 ELSE 0 END ASC").
			Order("CASE WHEN employees.employee_no IS NULL OR employees.employee_no = '' THEN 1 ELSE 0 END ASC").
			Order("employees.employee_no ASC").
			Order("employees.name ASC")
	} else {
		ordered = ordered.Order("name ASC")
	}
	err := ordered.
		Offset(offset).
		Limit(opts.Size).
		Find(&employees).Error
	return employees, total, err
}

// GetByOrgNodeAndNo 根据组织节点和工号查询员工（唯一性检查）。
func (r *Repository) GetByOrgNodeAndNo(ctx context.Context, orgNodeID, employeeNo string) (*Employee, error) {
	var emp Employee
	err := r.db.WithContext(ctx).
		Where("org_node_id = ? AND employee_no = ?", orgNodeID, employeeNo).
		First(&emp).Error
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

// BatchCreate 批量创建员工。
func (r *Repository) BatchCreate(ctx context.Context, employees []Employee) error {
	return r.db.WithContext(ctx).CreateInBatches(employees, 100).Error
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Employee{})
}

// ListOptions 列表查询选项。
type ListOptions struct {
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	Keyword  string `json:"keyword"`
	Status   string `json:"status"`
	Position string `json:"position"`
	Category string `json:"category"`
	PrioritizeAdmins bool `json:"prioritize_admins"`
}
