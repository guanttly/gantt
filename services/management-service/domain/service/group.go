package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// IGroupService 分组管理领域服务接口
type IGroupService interface {
	// CreateGroup 创建分组
	CreateGroup(ctx context.Context, group *model.Group) error

	// UpdateGroup 更新分组信息
	UpdateGroup(ctx context.Context, group *model.Group) error

	// DeleteGroup 删除分组
	DeleteGroup(ctx context.Context, orgID, groupID string) error

	// GetGroup 获取分组详情
	GetGroup(ctx context.Context, orgID, groupID string) (*model.Group, error)

	// ListGroups 查询分组列表
	ListGroups(ctx context.Context, filter *model.GroupFilter) (*model.GroupListResult, error)

	// GetGroupWithMembers 获取带成员的分组信息
	GetGroupWithMembers(ctx context.Context, orgID, groupID string) (*model.GroupWithMembers, error)

	// AddMember 添加成员到分组
	AddMember(ctx context.Context, groupID, employeeID, role string) error

	// BatchAddMembers 批量添加成员到分组
	BatchAddMembers(ctx context.Context, groupID string, employeeIDs []string, role string) error

	// RemoveMember 从分组移除成员
	RemoveMember(ctx context.Context, groupID, employeeID string) error

	// GetMembers 获取分组成员列表
	GetMembers(ctx context.Context, groupID string) ([]*model.Employee, error)

	// GetEmployeeGroups 获取员工所属的分组列表
	GetEmployeeGroups(ctx context.Context, orgID, employeeID string) ([]*model.Group, error)

	// UpdateMemberRole 更新成员在组内的角色
	UpdateMemberRole(ctx context.Context, groupID, employeeID, role string) error
}
