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

// GroupRepository 分组仓储实现
type GroupRepository struct {
	db *gorm.DB
}

// NewGroupRepository 创建分组仓储实例
func NewGroupRepository(db *gorm.DB) repository.IGroupRepository {
	return &GroupRepository{db: db}
}

// Create 创建分组
func (r *GroupRepository) Create(ctx context.Context, group *model.Group) error {
	groupEntity, err := mapper.GroupModelToEntity(group)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(groupEntity).Error
}

// Update 更新分组信息
func (r *GroupRepository) Update(ctx context.Context, group *model.Group) error {
	groupEntity, err := mapper.GroupModelToEntity(group)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", group.OrgID, group.ID).
		Updates(groupEntity).Error
}

// Delete 删除分组（软删除）
func (r *GroupRepository) Delete(ctx context.Context, orgID, groupID string) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, groupID).
		Delete(&entity.GroupEntity{}).Error
}

// GetByID 根据ID获取分组
func (r *GroupRepository) GetByID(ctx context.Context, orgID, groupID string) (*model.Group, error) {
	var groupEntity entity.GroupEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, groupID).
		First(&groupEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.GroupEntityToModel(&groupEntity)
}

// GetByCode 根据编码获取分组
func (r *GroupRepository) GetByCode(ctx context.Context, orgID, code string) (*model.Group, error) {
	var groupEntity entity.GroupEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&groupEntity).Error
	if err != nil {
		return nil, err
	}
	return mapper.GroupEntityToModel(&groupEntity)
}

// List 查询分组列表
func (r *GroupRepository) List(ctx context.Context, filter *model.GroupFilter) (*model.GroupListResult, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}

	query := r.db.WithContext(ctx).Model(&entity.GroupEntity{}).
		Where("org_id = ?", filter.OrgID)

	// 应用过滤条件
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
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
	var groupEntities []*entity.GroupEntity
	offset := (filter.Page - 1) * filter.PageSize
	err := query.Offset(offset).Limit(filter.PageSize).
		Order("code ASC").
		Find(&groupEntities).Error
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	groups, err := mapper.GroupEntitiesToModels(groupEntities)
	if err != nil {
		return nil, err
	}

	// 填充成员数量
	for _, group := range groups {
		var memberCount int64
		if err := r.db.WithContext(ctx).Model(&entity.GroupMemberEntity{}).
			Where("group_id = ?", group.ID).
			Count(&memberCount).Error; err == nil {
			group.MemberCount = int(memberCount)
		}
	}

	return &model.GroupListResult{
		Items:    groups,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// Exists 检查分组是否存在
func (r *GroupRepository) Exists(ctx context.Context, orgID, groupID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GroupEntity{}).
		Where("org_id = ? AND id = ?", orgID, groupID).
		Count(&count).Error
	return count > 0, err
}

// GetChildren 获取子分组
func (r *GroupRepository) GetChildren(ctx context.Context, orgID, parentID string) ([]*model.Group, error) {
	var groupEntities []*entity.GroupEntity
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND parent_id = ?", orgID, parentID).
		Order("created_at ASC").
		Find(&groupEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.GroupEntitiesToModels(groupEntities)
}

// AddMember 添加成员到分组
func (r *GroupRepository) AddMember(ctx context.Context, member *model.GroupMember) error {
	memberEntity := mapper.GroupMemberModelToEntity(member)
	return r.db.WithContext(ctx).Create(memberEntity).Error
}

// RemoveMember 从分组移除成员
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, employeeID string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ? AND employee_id = ?", groupID, employeeID).
		Delete(&entity.GroupMemberEntity{}).Error
}

// GetMembers 获取分组成员列表
func (r *GroupRepository) GetMembers(ctx context.Context, groupID string) ([]*model.Employee, error) {
	var employeeEntities []*entity.EmployeeEntity
	err := r.db.WithContext(ctx).
		Table("employees").
		Joins("INNER JOIN group_members ON employees.id = group_members.employee_id").
		Where("group_members.group_id = ?", groupID).
		Order("employees.employee_id ASC").
		Find(&employeeEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.EmployeeEntitiesToModels(employeeEntities)
}

// GetMemberGroups 获取员工所属的分组列表
func (r *GroupRepository) GetMemberGroups(ctx context.Context, orgID, employeeID string) ([]*model.Group, error) {
	var groupEntities []*entity.GroupEntity
	err := r.db.WithContext(ctx).
		Table("`groups`").
		Joins("INNER JOIN `group_members` ON `groups`.`id` = `group_members`.`group_id`").
		Where("`groups`.`org_id` = ? AND `group_members`.`employee_id` = ?", orgID, employeeID).
		Order("`groups`.`code` ASC").
		Find(&groupEntities).Error
	if err != nil {
		return nil, err
	}
	return mapper.GroupEntitiesToModels(groupEntities)
}

// BatchGetMemberGroups 批量获取员工所属的分组列表
func (r *GroupRepository) BatchGetMemberGroups(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Group, error) {
	if len(employeeIDs) == 0 {
		return make(map[string][]*model.Group), nil
	}

	// 查询所有相关的分组成员关系
	type GroupMemberResult struct {
		EmployeeID string
		GroupID    string
		GroupCode  string
		GroupName  string
		GroupType  string
		OrgID      string
	}

	var results []GroupMemberResult

	// 使用反引号包裹表名避免与 SQL 关键字冲突
	err := r.db.WithContext(ctx).Debug().
		Table("group_members").
		Select("`group_members`.`employee_id`, `groups`.`id` as group_id, `groups`.`code` as group_code, `groups`.`name` as group_name, `groups`.`type` as group_type, `groups`.`org_id`").
		Joins("INNER JOIN `groups` ON `groups`.`id` = `group_members`.`group_id`").
		Where("`groups`.`org_id` = ? AND `group_members`.`employee_id` IN ?", orgID, employeeIDs).
		Order("`groups`.`code` ASC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("batch get member groups failed: %w", err)
	}

	// 构建 map
	groupMap := make(map[string][]*model.Group)
	for _, result := range results {
		group := &model.Group{
			ID:    result.GroupID,
			OrgID: result.OrgID,
			Code:  result.GroupCode,
			Name:  result.GroupName,
			Type:  model.GroupType(result.GroupType),
		}
		groupMap[result.EmployeeID] = append(groupMap[result.EmployeeID], group)
	}

	// 为没有分组的员工初始化空数组
	for _, empID := range employeeIDs {
		if _, exists := groupMap[empID]; !exists {
			groupMap[empID] = []*model.Group{}
		}
	}

	return groupMap, nil
}

// IsMember 检查员工是否在分组中
func (r *GroupRepository) IsMember(ctx context.Context, groupID, employeeID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GroupMemberEntity{}).
		Where("group_id = ? AND employee_id = ?", groupID, employeeID).
		Count(&count).Error
	return count > 0, err
}

// GetGroupWithMembers 获取带成员的分组信息
func (r *GroupRepository) GetGroupWithMembers(ctx context.Context, orgID, groupID string) (*model.GroupWithMembers, error) {
	// 获取分组信息
	group, err := r.GetByID(ctx, orgID, groupID)
	if err != nil {
		return nil, err
	}

	// 获取成员列表
	members, err := r.GetMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return &model.GroupWithMembers{
		Group:   group,
		Members: members,
	}, nil
}

// UpdateMemberRole 更新成员在组内的角色
func (r *GroupRepository) UpdateMemberRole(ctx context.Context, groupID, employeeID, role string) error {
	return r.db.WithContext(ctx).Model(&entity.GroupMemberEntity{}).
		Where("group_id = ? AND employee_id = ?", groupID, employeeID).
		Update("role", role).Error
}
