package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/logging"

	domain_service "jusha/gantt/service/management/domain/service"
)

// GroupServiceImpl 分组管理服务实现
type GroupServiceImpl struct {
	groupRepo    repository.IGroupRepository
	employeeRepo repository.IEmployeeRepository
	logger       logging.ILogger
}

// NewGroupService 创建分组管理服务
func NewGroupService(
	groupRepo repository.IGroupRepository,
	employeeRepo repository.IEmployeeRepository,
	logger logging.ILogger,
) domain_service.IGroupService {
	return &GroupServiceImpl{
		groupRepo:    groupRepo,
		employeeRepo: employeeRepo,
		logger:       logger.With("service", "GroupService"),
	}
}

// CreateGroup 创建分组
func (s *GroupServiceImpl) CreateGroup(ctx context.Context, group *model.Group) error {
	// 验证必填字段
	if group.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if group.Name == "" {
		return fmt.Errorf("name is required")
	}

	// 生成ID
	if group.ID == "" {
		group.ID = uuid.New().String()
	}

	// 设置默认类型
	if group.Type == "" {
		group.Type = model.GroupTypeCustom
	}

	// 如果有父组,验证父组存在
	if group.ParentID != nil && *group.ParentID != "" {
		exists, err := s.groupRepo.Exists(ctx, group.OrgID, *group.ParentID)
		if err != nil {
			return fmt.Errorf("check parent group existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("parent group not found: %s", *group.ParentID)
		}
	}

	// 创建分组
	if err := s.groupRepo.Create(ctx, group); err != nil {
		s.logger.Error("Failed to create group", "error", err)
		return fmt.Errorf("create group: %w", err)
	}

	s.logger.Info("Group created successfully", "groupId", group.ID, "name", group.Name)
	return nil
}

// UpdateGroup 更新分组信息
func (s *GroupServiceImpl) UpdateGroup(ctx context.Context, group *model.Group) error {
	if group.ID == "" {
		return fmt.Errorf("group id is required")
	}
	if group.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查分组是否存在
	exists, err := s.groupRepo.Exists(ctx, group.OrgID, group.ID)
	if err != nil {
		return fmt.Errorf("check group existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("group not found")
	}

	// 如果更新父组,验证父组存在且不形成循环
	if group.ParentID != nil && *group.ParentID != "" {
		if *group.ParentID == group.ID {
			return fmt.Errorf("group cannot be its own parent")
		}
		exists, err := s.groupRepo.Exists(ctx, group.OrgID, *group.ParentID)
		if err != nil {
			return fmt.Errorf("check parent group existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("parent group not found: %s", *group.ParentID)
		}
	}

	if err := s.groupRepo.Update(ctx, group); err != nil {
		s.logger.Error("Failed to update group", "error", err)
		return fmt.Errorf("update group: %w", err)
	}

	s.logger.Info("Group updated successfully", "groupId", group.ID)
	return nil
}

// DeleteGroup 删除分组
func (s *GroupServiceImpl) DeleteGroup(ctx context.Context, orgID, groupID string) error {
	if orgID == "" || groupID == "" {
		return fmt.Errorf("orgId and groupId are required")
	}

	// 检查分组是否存在
	exists, err := s.groupRepo.Exists(ctx, orgID, groupID)
	if err != nil {
		return fmt.Errorf("check group existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("group not found")
	}

	// 检查是否有子组
	children, err := s.groupRepo.GetChildren(ctx, orgID, groupID)
	if err != nil {
		return fmt.Errorf("check children: %w", err)
	}
	if len(children) > 0 {
		return fmt.Errorf("cannot delete group with children, please delete children first")
	}

	if err := s.groupRepo.Delete(ctx, orgID, groupID); err != nil {
		s.logger.Error("Failed to delete group", "error", err)
		return fmt.Errorf("delete group: %w", err)
	}

	s.logger.Info("Group deleted successfully", "groupId", groupID)
	return nil
}

// GetGroup 获取分组详情
func (s *GroupServiceImpl) GetGroup(ctx context.Context, orgID, groupID string) (*model.Group, error) {
	if orgID == "" || groupID == "" {
		return nil, fmt.Errorf("orgId and groupId are required")
	}

	group, err := s.groupRepo.GetByID(ctx, orgID, groupID)
	if err != nil {
		s.logger.Error("Failed to get group", "error", err)
		return nil, fmt.Errorf("get group: %w", err)
	}

	return group, nil
}

// ListGroups 查询分组列表
func (s *GroupServiceImpl) ListGroups(ctx context.Context, filter *model.GroupFilter) (*model.GroupListResult, error) {
	if filter == nil {
		filter = &model.GroupFilter{}
	}
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	// 设置默认分页
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	result, err := s.groupRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list groups", "error", err)
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return result, nil
}

// GetGroupWithMembers 获取带成员的分组信息
func (s *GroupServiceImpl) GetGroupWithMembers(ctx context.Context, orgID, groupID string) (*model.GroupWithMembers, error) {
	if orgID == "" || groupID == "" {
		return nil, fmt.Errorf("orgId and groupId are required")
	}

	group, err := s.groupRepo.GetByID(ctx, orgID, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	members, err := s.groupRepo.GetMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	return &model.GroupWithMembers{
		Group:   group,
		Members: members,
	}, nil
}

// AddMember 添加成员到分组
func (s *GroupServiceImpl) AddMember(ctx context.Context, groupID, employeeID, role string) error {
	if groupID == "" || employeeID == "" {
		return fmt.Errorf("groupId and employeeId are required")
	}

	member := &model.GroupMember{
		GroupID:    groupID,
		EmployeeID: employeeID,
		Role:       role,
	}

	// 验证组和员工存在性可以在repository层处理
	if err := s.groupRepo.AddMember(ctx, member); err != nil {
		s.logger.Error("Failed to add member", "error", err, "groupId", groupID, "employeeId", employeeID)
		return fmt.Errorf("add member: %w", err)
	}

	s.logger.Info("Member added to group", "groupId", groupID, "employeeId", employeeID, "role", role)
	return nil
}

// BatchAddMembers 批量添加成员到分组
func (s *GroupServiceImpl) BatchAddMembers(ctx context.Context, groupID string, employeeIDs []string, role string) error {
	if groupID == "" {
		return fmt.Errorf("groupId is required")
	}
	if len(employeeIDs) == 0 {
		return fmt.Errorf("employeeIds is required")
	}

	// 批量添加成员
	for _, employeeID := range employeeIDs {
		if employeeID == "" {
			continue
		}

		member := &model.GroupMember{
			GroupID:    groupID,
			EmployeeID: employeeID,
			Role:       role,
		}

		// 忽略重复添加的错误，继续添加其他成员
		if err := s.groupRepo.AddMember(ctx, member); err != nil {
			s.logger.Warn("Failed to add member, skipping", "error", err, "groupId", groupID, "employeeId", employeeID)
			// 不返回错误，继续添加其他成员
		}
	}

	s.logger.Info("Batch add members to group", "groupId", groupID, "count", len(employeeIDs))
	return nil
}

// RemoveMember 从分组移除成员
func (s *GroupServiceImpl) RemoveMember(ctx context.Context, groupID, employeeID string) error {
	if groupID == "" || employeeID == "" {
		return fmt.Errorf("groupId and employeeId are required")
	}

	if err := s.groupRepo.RemoveMember(ctx, groupID, employeeID); err != nil {
		s.logger.Error("Failed to remove member", "error", err)
		return fmt.Errorf("remove member: %w", err)
	}

	s.logger.Info("Member removed from group", "groupId", groupID, "employeeId", employeeID)
	return nil
}

// GetMembers 获取分组成员列表
func (s *GroupServiceImpl) GetMembers(ctx context.Context, groupID string) ([]*model.Employee, error) {
	if groupID == "" {
		return nil, fmt.Errorf("groupId is required")
	}

	members, err := s.groupRepo.GetMembers(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get members", "error", err)
		return nil, fmt.Errorf("get members: %w", err)
	}

	return members, nil
}

// GetEmployeeGroups 获取员工所属的分组列表
func (s *GroupServiceImpl) GetEmployeeGroups(ctx context.Context, orgID, employeeID string) ([]*model.Group, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}

	groups, err := s.groupRepo.GetMemberGroups(ctx, orgID, employeeID)
	if err != nil {
		s.logger.Error("Failed to get employee groups", "error", err)
		return nil, fmt.Errorf("get employee groups: %w", err)
	}

	return groups, nil
}

// UpdateMemberRole 更新成员在组内的角色
func (s *GroupServiceImpl) UpdateMemberRole(ctx context.Context, groupID, employeeID, role string) error {
	if groupID == "" || employeeID == "" {
		return fmt.Errorf("groupId and employeeId are required")
	}

	if err := s.groupRepo.UpdateMemberRole(ctx, groupID, employeeID, role); err != nil {
		s.logger.Error("Failed to update member role", "error", err)
		return fmt.Errorf("update member role: %w", err)
	}

	s.logger.Info("Member role updated", "groupId", groupID, "employeeId", employeeID, "role", role)
	return nil
}
