package group

import (
	"context"
	"errors"
	"fmt"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrGroupNotFound  = errors.New("分组不存在")
	ErrMemberExists   = errors.New("成员已在分组中")
	ErrMemberNotFound = errors.New("成员不在分组中")
	ErrNotDeptNode    = errors.New("只有科室级（department）节点可以管理排班分组")
)

// CreateInput 创建分组的输入参数。
type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// UpdateInput 更新分组的输入参数。
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Service 分组业务逻辑层。
type Service struct {
	repo            *Repository
	appRoleSyncer   AppRoleSyncer
	orgNodeResolver OrgNodeTypeChecker
}

type OrgNodeTypeChecker interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
}

type AppRoleSyncer interface {
	SyncRolesForGroupMember(ctx context.Context, groupID, employeeID, grantedBy string) error
	RevokeRolesForGroupMember(ctx context.Context, groupID, employeeID string) error
	CleanupGroup(ctx context.Context, groupID string) error
}

// NewService 创建分组服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetAppRoleSyncer(syncer AppRoleSyncer) {
	s.appRoleSyncer = syncer
}

func (s *Service) SetOrgNodeResolver(resolver OrgNodeTypeChecker) {
	s.orgNodeResolver = resolver
}

func (s *Service) ensureDepartmentNode(ctx context.Context) error {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return fmt.Errorf("缺少组织节点信息")
	}
	if s.orgNodeResolver == nil {
		return nil
	}
	node, err := s.orgNodeResolver.GetByID(ctx, orgNodeID)
	if err != nil {
		return fmt.Errorf("查询组织节点失败: %w", err)
	}
	if !tenant.IsLeafNodeType(node.NodeType) {
		return ErrNotDeptNode
	}
	return nil
}

// Create 创建分组。
func (s *Service) Create(ctx context.Context, input CreateInput) (*EmployeeGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	g := &EmployeeGroup{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.Create(ctx, g); err != nil {
		return nil, fmt.Errorf("创建分组失败: %w", err)
	}

	return g, nil
}

// GetByID 获取分组详情。
func (s *Service) GetByID(ctx context.Context, id string) (*EmployeeGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return g, nil
}

// Update 更新分组信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*EmployeeGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		g.Name = *input.Name
	}
	if input.Description != nil {
		g.Description = input.Description
	}

	if err := s.repo.Update(ctx, g); err != nil {
		return nil, fmt.Errorf("更新分组失败: %w", err)
	}

	return g, nil
}

// Delete 删除分组（同时删除成员关联）。
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}

	if s.appRoleSyncer != nil {
		if err := s.appRoleSyncer.CleanupGroup(ctx, id); err != nil {
			return err
		}
	}

	if err := s.repo.DeleteMembersByGroup(ctx, id); err != nil {
		return fmt.Errorf("删除分组成员失败: %w", err)
	}

	return s.repo.Delete(ctx, id)
}

// List 查询分组列表。
func (s *Service) List(ctx context.Context) ([]EmployeeGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	return s.repo.List(ctx)
}

// GetMembers 获取分组成员列表。
func (s *Service) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	// 验证分组存在
	_, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	return s.repo.GetMembers(ctx, groupID)
}

// AddMember 添加成员到分组。
func (s *Service) AddMember(ctx context.Context, groupID, employeeID, grantedBy string) (*GroupMember, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 验证分组存在
	_, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	// 检查是否已存在
	existing, err := s.repo.GetMember(ctx, groupID, employeeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrMemberExists
	}

	m := &GroupMember{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		EmployeeID: employeeID,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.AddMember(ctx, m); err != nil {
		return nil, fmt.Errorf("添加成员失败: %w", err)
	}

	if s.appRoleSyncer != nil {
		if err := s.appRoleSyncer.SyncRolesForGroupMember(ctx, groupID, employeeID, grantedBy); err != nil {
			_ = s.repo.RemoveMember(ctx, groupID, employeeID)
			return nil, err
		}
	}

	return m, nil
}

// RemoveMember 从分组中移除成员。
func (s *Service) RemoveMember(ctx context.Context, groupID, employeeID string) error {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}

	_, err := s.repo.GetMember(ctx, groupID, employeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return err
	}

	if err := s.repo.RemoveMember(ctx, groupID, employeeID); err != nil {
		return err
	}
	if s.appRoleSyncer != nil {
		if err := s.appRoleSyncer.RevokeRolesForGroupMember(ctx, groupID, employeeID); err != nil {
			return err
		}
	}
	return nil
}

// RemoveEmployeeFromAllGroups 将员工从所有分组中移除（员工调动时调用），返回移除的分组数。
func (s *Service) RemoveEmployeeFromAllGroups(ctx context.Context, employeeID string) (int64, error) {
	// 先查出该员工属于哪些分组，以便撤销角色
	members, err := s.repo.GetMembersByEmployeeID(ctx, employeeID)
	if err != nil {
		return 0, err
	}

	if s.appRoleSyncer != nil {
		for _, m := range members {
			_ = s.appRoleSyncer.RevokeRolesForGroupMember(ctx, m.GroupID, employeeID)
		}
	}

	return s.repo.RemoveEmployeeFromAllGroups(ctx, employeeID)
}

// GetMemberEmployeeIDs 获取指定分组的所有成员员工ID列表。
func (s *Service) GetMemberEmployeeIDs(ctx context.Context, groupID string) ([]string, error) {
	return s.repo.GetMemberEmployeeIDs(ctx, groupID)
}
