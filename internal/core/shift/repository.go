package shift

import (
	"context"
	"errors"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func scopeByQualifiedOrgNode(ctx context.Context, db *gorm.DB, qualifiedColumn string) *gorm.DB {
	nodeID := tenant.GetOrgNodeID(ctx)
	if nodeID == "" {
		return db
	}
	return db.Where(qualifiedColumn+" = ?", nodeID)
}

// Repository 班次数据访问层。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建班次仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建班次。
func (r *Repository) Create(ctx context.Context, s *Shift) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(s).Error
}

// GetByID 根据 ID 查询班次。
func (r *Repository) GetByID(ctx context.Context, id string) (*Shift, error) {
	var s Shift
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Update 更新班次。
func (r *Repository) Update(ctx context.Context, s *Shift) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(s).Error
}

// Delete 删除班次（硬删除）。
func (r *Repository) Delete(ctx context.Context, id string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id = ?", id).
		Delete(&Shift{}).Error
}

// List 查询班次列表。
func (r *Repository) List(ctx context.Context) ([]Shift, error) {
	var shifts []Shift
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Order("priority ASC, code ASC").
		Find(&shifts).Error
	return shifts, err
}

// ListActive 查询启用中的班次列表。
func (r *Repository) ListActive(ctx context.Context) ([]Shift, error) {
	var shifts []Shift
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("status = ?", StatusActive).
		Order("priority ASC, code ASC").
		Find(&shifts).Error
	return shifts, err
}

// GetByOrgNodeAndCode 根据组织节点和编码查询班次（唯一性检查）。
func (r *Repository) GetByOrgNodeAndCode(ctx context.Context, orgNodeID, code string) (*Shift, error) {
	var s Shift
	err := r.db.WithContext(ctx).
		Where("org_node_id = ? AND code = ?", orgNodeID, code).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListByIDs(ctx context.Context, ids []string) ([]Shift, error) {
	var shifts []Shift
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("id IN ?", ids).
		Find(&shifts).Error
	return shifts, err
}

// CreateDependency 创建班次依赖。
func (r *Repository) CreateDependency(ctx context.Context, dep *ShiftDependency) error {
	if dep.ID == "" {
		dep.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(dep).Error
}

// DeleteDependency 删除班次依赖。
func (r *Repository) DeleteDependency(ctx context.Context, shiftID, dependsOnID, depType string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("shift_id = ? AND depends_on_id = ? AND dependency_type = ?", shiftID, dependsOnID, depType).
		Delete(&ShiftDependency{}).Error
}

// GetDependencies 查询班次依赖列表。
func (r *Repository) GetDependencies(ctx context.Context) ([]ShiftDependency, error) {
	var deps []ShiftDependency
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Find(&deps).Error
	return deps, err
}

// GetDependenciesByShift 查询某班次的依赖列表。
func (r *Repository) GetDependenciesByShift(ctx context.Context, shiftID string) ([]ShiftDependency, error) {
	var deps []ShiftDependency
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("shift_id = ?", shiftID).
		Find(&deps).Error
	return deps, err
}

// DeleteDependenciesByShift 删除某班次的所有依赖。
func (r *Repository) DeleteDependenciesByShift(ctx context.Context, shiftID string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("shift_id = ? OR depends_on_id = ?", shiftID, shiftID).
		Delete(&ShiftDependency{}).Error
}

func (r *Repository) GetShiftGroups(ctx context.Context, shiftID string) ([]ShiftGroup, error) {
	var groups []ShiftGroup
	err := scopeByQualifiedOrgNode(ctx, r.db.WithContext(ctx), "shift_groups.org_node_id").
		Select("shift_groups.*, employee_groups.name AS group_name").
		Joins("LEFT JOIN employee_groups ON employee_groups.id = shift_groups.group_id AND employee_groups.org_node_id = shift_groups.org_node_id").
		Where("shift_groups.shift_id = ?", shiftID).
		Order("shift_groups.priority ASC, shift_groups.created_at ASC").
		Find(&groups).Error
	return groups, err
}

func (r *Repository) ReplaceShiftGroups(ctx context.Context, shiftID string, groupIDs []string) error {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tenant.ApplyScope(ctx, tx).Where("shift_id = ?", shiftID).Delete(&ShiftGroup{}).Error; err != nil {
			return err
		}
		for index, groupID := range groupIDs {
			item := ShiftGroup{
				ID:          uuid.New().String(),
				ShiftID:     shiftID,
				GroupID:     groupID,
				Priority:    index,
				IsActive:    true,
				TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID},
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) UpsertShiftGroup(ctx context.Context, item *ShiftGroup) error {
	var existing ShiftGroup
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Where("shift_id = ? AND group_id = ?", item.ShiftID, item.GroupID).First(&existing).Error
	if err == nil {
		existing.Priority = item.Priority
		existing.IsActive = item.IsActive
		existing.Notes = item.Notes
		return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *Repository) RemoveShiftGroup(ctx context.Context, shiftID, groupID string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Where("shift_id = ? AND group_id = ?", shiftID, groupID).Delete(&ShiftGroup{}).Error
}

func (r *Repository) GetShiftGroupCounts(ctx context.Context, shiftIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(shiftIDs))
	if len(shiftIDs) == 0 {
		return result, nil
	}
	type row struct {
		ShiftID string
		Count   int64
	}
	var rows []row
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Model(&ShiftGroup{}).
		Select("shift_id, COUNT(*) AS count").
		Where("shift_id IN ?", shiftIDs).
		Group("shift_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.ShiftID] = item.Count
	}
	return result, nil
}

func (r *Repository) GetShiftGroupNames(ctx context.Context, shiftIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(shiftIDs))
	if len(shiftIDs) == 0 {
		return result, nil
	}
	type row struct {
		ShiftID   string
		GroupName string
	}
	var rows []row
	err := scopeByQualifiedOrgNode(ctx, r.db.WithContext(ctx), "shift_groups.org_node_id").
		Table("shift_groups").
		Select("shift_groups.shift_id, employee_groups.name AS group_name").
		Joins("LEFT JOIN employee_groups ON employee_groups.id = shift_groups.group_id AND employee_groups.org_node_id = shift_groups.org_node_id").
		Where("shift_groups.shift_id IN ?", shiftIDs).
		Order("shift_groups.shift_id ASC, shift_groups.priority ASC, shift_groups.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.GroupName == "" {
			continue
		}
		result[item.ShiftID] = append(result[item.ShiftID], item.GroupName)
	}
	return result, nil
}

func (r *Repository) GetFixedAssignments(ctx context.Context, shiftID string) ([]FixedAssignment, error) {
	var items []FixedAssignment
	err := scopeByQualifiedOrgNode(ctx, r.db.WithContext(ctx), "fixed_assignments.org_node_id").
		Select("fixed_assignments.*, employees.name AS staff_name").
		Joins("LEFT JOIN employees ON employees.id = fixed_assignments.employee_id AND employees.org_node_id = fixed_assignments.org_node_id").
		Where("fixed_assignments.shift_id = ?", shiftID).
		Order("fixed_assignments.created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *Repository) ReplaceFixedAssignments(ctx context.Context, shiftID string, assignments []FixedAssignment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tenant.ApplyScope(ctx, tx).Where("shift_id = ?", shiftID).Delete(&FixedAssignment{}).Error; err != nil {
			return err
		}
		for _, assignment := range assignments {
			if err := tx.Create(&assignment).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) DeleteFixedAssignment(ctx context.Context, shiftID, assignmentID string) error {
	return tenant.ApplyScope(ctx, r.db.WithContext(ctx)).Where("shift_id = ? AND id = ?", shiftID, assignmentID).Delete(&FixedAssignment{}).Error
}

func (r *Repository) GetFixedAssignmentCounts(ctx context.Context, shiftIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(shiftIDs))
	if len(shiftIDs) == 0 {
		return result, nil
	}
	type row struct {
		ShiftID string
		Count   int64
	}
	var rows []row
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Model(&FixedAssignment{}).
		Select("shift_id, COUNT(*) AS count").
		Where("shift_id IN ?", shiftIDs).
		Where("is_active = ?", true).
		Group("shift_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.ShiftID] = item.Count
	}
	return result, nil
}

func (r *Repository) GetWeeklyStaffConfig(ctx context.Context, shiftID string) ([]ShiftWeeklyStaff, error) {
	var items []ShiftWeeklyStaff
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("shift_id = ?", shiftID).
		Order("weekday ASC").
		Find(&items).Error
	return items, err
}

func (r *Repository) BatchGetWeeklyStaffConfig(ctx context.Context, shiftIDs []string) (map[string][]ShiftWeeklyStaff, error) {
	result := make(map[string][]ShiftWeeklyStaff, len(shiftIDs))
	if len(shiftIDs) == 0 {
		return result, nil
	}
	var items []ShiftWeeklyStaff
	err := tenant.ApplyScope(ctx, r.db.WithContext(ctx)).
		Where("shift_id IN ?", shiftIDs).
		Order("shift_id ASC, weekday ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ShiftID] = append(result[item.ShiftID], item)
	}
	return result, nil
}

func (r *Repository) ReplaceWeeklyStaffConfig(ctx context.Context, shiftID string, items []ShiftWeeklyStaff) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tenant.ApplyScope(ctx, tx).Where("shift_id = ?", shiftID).Delete(&ShiftWeeklyStaff{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AutoMigrate 自动迁移表结构。
func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Shift{}, &ShiftDependency{}, &ShiftGroup{}, &FixedAssignment{}, &ShiftWeeklyStaff{})
}
