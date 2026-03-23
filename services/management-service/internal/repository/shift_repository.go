package repository

import (
	"context"
	"fmt"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/gantt/service/management/internal/mapper"

	"gorm.io/gorm"
)

// ShiftRepository 班次仓储实现
type ShiftRepository struct {
	db *gorm.DB
}

// NewShiftRepository 创建班次仓储实例
func NewShiftRepository(db *gorm.DB) repository.IShiftRepository {
	return &ShiftRepository{db: db}
}

// Create 创建班次
func (r *ShiftRepository) Create(ctx context.Context, shift *model.Shift) error {
	shiftEntity := mapper.ShiftModelToEntity(shift)
	return r.db.WithContext(ctx).Create(shiftEntity).Error
}

// Update 更新班次信息
func (r *ShiftRepository) Update(ctx context.Context, shift *model.Shift) error {
	shiftEntity := mapper.ShiftModelToEntity(shift)
	return r.db.WithContext(ctx).
		Model(&entity.ShiftEntity{}).
		Where("org_id = ? AND id = ?", shift.OrgID, shift.ID).
		Omit("created_at").
		Select("*"). // 明确选择所有字段，确保 bool 类型的 false 值也能被更新
		Updates(shiftEntity).Error
}

// Delete 删除班次（软删除）
func (r *ShiftRepository) Delete(ctx context.Context, orgID, shiftID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, shiftID).
		Delete(&entity.ShiftEntity{}).Error
}

// GetByID 根据ID获取班次
// GetByID 根据ID获取班次
func (r *ShiftRepository) GetByID(ctx context.Context, orgID, shiftID string) (*model.Shift, error) {
	var shiftEntity entity.ShiftEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, shiftID).
		First(&shiftEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftEntityToModel(&shiftEntity), nil
}

// GetByCode 根据编码获取班次
func (r *ShiftRepository) GetByCode(ctx context.Context, orgID, code string) (*model.Shift, error) {
	var shiftEntity entity.ShiftEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&shiftEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftEntityToModel(&shiftEntity), nil
}

// List 查询班次列表
func (r *ShiftRepository) List(ctx context.Context, filter *model.ShiftFilter) (*model.ShiftListResult, error) {
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.ShiftEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 应用过滤条件
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", keyword, keyword)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var shiftEntities []*entity.ShiftEntity
	offset := (filter.Page - 1) * filter.PageSize
	err := query.Offset(offset).Limit(filter.PageSize).
		Order("scheduling_priority ASC, created_at DESC").
		Find(&shiftEntities).Error
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	shifts := mapper.ShiftEntitiesToModels(shiftEntities)

	return &model.ShiftListResult{
		Items:    shifts,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查班次是否存在
func (r *ShiftRepository) Exists(ctx context.Context, orgID, shiftID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ShiftEntity{}).
		Where("org_id = ? AND id = ?", orgID, shiftID).
		Count(&count).Error
	return count > 0, err
}

// BatchGet 批量获取班次
func (r *ShiftRepository) BatchGet(ctx context.Context, orgID string, shiftIDs []string) ([]*model.Shift, error) {
	if len(shiftIDs) == 0 {
		return []*model.Shift{}, nil
	}

	var shiftEntities []*entity.ShiftEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id IN ?", orgID, shiftIDs).
		Find(&shiftEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftEntitiesToModels(shiftEntities), nil
}

// GetActiveShifts 获取所有启用的班次
func (r *ShiftRepository) GetActiveShifts(ctx context.Context, orgID string) ([]*model.Shift, error) {
	var shiftEntities []*entity.ShiftEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND is_active = ?", orgID, true).
		Order("scheduling_priority ASC, created_at DESC").
		Find(&shiftEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftEntitiesToModels(shiftEntities), nil
}

// ShiftGroupRepository 班次-分组关联仓储实现
type ShiftGroupRepository struct {
	db *gorm.DB
}

// NewShiftGroupRepository 创建班次-分组关联仓储实例
func NewShiftGroupRepository(db *gorm.DB) repository.IShiftGroupRepository {
	return &ShiftGroupRepository{db: db}
}

// AddGroupToShift 为班次添加关联分组
func (r *ShiftGroupRepository) AddGroupToShift(ctx context.Context, shiftGroup *model.ShiftGroup) error {
	entity := mapper.ShiftGroupModelToEntity(shiftGroup)
	return r.db.WithContext(ctx).Create(entity).Error
}

// RemoveGroupFromShift 从班次移除关联分组
func (r *ShiftGroupRepository) RemoveGroupFromShift(ctx context.Context, shiftID, groupID string) error {
	return r.db.WithContext(ctx).
		Where("shift_id = ? AND group_id = ?", shiftID, groupID).
		Delete(&entity.ShiftGroupEntity{}).Error
}

// GetShiftGroups 获取班次关联的所有分组
func (r *ShiftGroupRepository) GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error) {
	var entities []*entity.ShiftGroupEntity
	err := r.db.WithContext(ctx).
		Where("shift_id = ? AND is_active = ?", shiftID, true).
		Order("priority ASC, created_at ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	// 转换为模型
	shiftGroups := mapper.ShiftGroupEntitiesToModels(entities)

	// 批量查询分组信息
	if len(shiftGroups) > 0 {
		// 收集所有分组ID
		groupIDs := make([]string, 0, len(shiftGroups))
		for _, sg := range shiftGroups {
			groupIDs = append(groupIDs, sg.GroupID)
		}

		// 查询分组信息
		var groupEntities []*entity.GroupEntity
		err := r.db.WithContext(ctx).
			Where("id IN ?", groupIDs).
			Find(&groupEntities).Error
		if err == nil {
			// 创建分组ID到分组的映射
			groupMap := make(map[string]*entity.GroupEntity)
			for _, ge := range groupEntities {
				groupMap[ge.ID] = ge
			}

			// 填充分组名称和编码
			for _, sg := range shiftGroups {
				if ge, exists := groupMap[sg.GroupID]; exists {
					sg.GroupName = ge.Name
					sg.GroupCode = ge.Code
				}
			}
		}
	}

	return shiftGroups, nil
}

// GetGroupShifts 获取分组关联的所有班次
func (r *ShiftGroupRepository) GetGroupShifts(ctx context.Context, groupID string) ([]*model.ShiftGroup, error) {
	var entities []*entity.ShiftGroupEntity
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND is_active = ?", groupID, true).
		Order("priority ASC, created_at ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return mapper.ShiftGroupEntitiesToModels(entities), nil
}

// BatchSetShiftGroups 批量设置班次关联的分组（先删除旧的，再添加新的）
func (r *ShiftGroupRepository) BatchSetShiftGroups(ctx context.Context, shiftID string, groupIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除现有关联
		if err := tx.Where("shift_id = ?", shiftID).Delete(&entity.ShiftGroupEntity{}).Error; err != nil {
			return err
		}

		// 如果没有新的分组，直接返回
		if len(groupIDs) == 0 {
			return nil
		}

		// 创建新关联
		entities := make([]*entity.ShiftGroupEntity, 0, len(groupIDs))
		for i, groupID := range groupIDs {
			entities = append(entities, &entity.ShiftGroupEntity{
				ShiftID:  shiftID,
				GroupID:  groupID,
				Priority: i, // 按顺序设置优先级
				IsActive: true,
			})
		}

		return tx.Create(&entities).Error
	})
}

// UpdateShiftGroup 更新班次-分组关联信息
func (r *ShiftGroupRepository) UpdateShiftGroup(ctx context.Context, shiftGroup *model.ShiftGroup) error {
	entity := mapper.ShiftGroupModelToEntity(shiftGroup)
	return r.db.WithContext(ctx).
		Where("shift_id = ? AND group_id = ?", shiftGroup.ShiftID, shiftGroup.GroupID).
		Updates(entity).Error
}

// ExistsShiftGroup 检查班次-分组关联是否存在
func (r *ShiftGroupRepository) ExistsShiftGroup(ctx context.Context, shiftID, groupID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ShiftGroupEntity{}).
		Where("shift_id = ? AND group_id = ?", shiftID, groupID).
		Count(&count).Error
	return count > 0, err
}

// GetShiftGroupMembers 获取班次关联的所有分组的成员（去重）
func (r *ShiftGroupRepository) GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error) {
	var employees []*entity.EmployeeEntity

	// 联表查询：employees -> group_members -> shift_groups
	// 注意：需要去重 (DISTINCT)
	// 按 employee_id 排序（员工编号，代表资历）
	err := r.db.WithContext(ctx).
		Table("employees").
		Distinct("employees.*").
		Joins("JOIN group_members ON group_members.employee_id = employees.id").
		Joins("JOIN shift_groups ON shift_groups.group_id = group_members.group_id").
		Where("shift_groups.shift_id = ? AND shift_groups.is_active = ?", shiftID, true).
		Where("employees.deleted_at IS NULL").
		Order("employees.employee_id ASC").
		Find(&employees).Error

	if err != nil {
		return nil, err
	}

	return mapper.EmployeeEntitiesToModels(employees)
}
