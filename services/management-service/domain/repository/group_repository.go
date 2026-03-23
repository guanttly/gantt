package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// IGroupRepository 分组仓储接口
type IGroupRepository interface {
	// Create 创建分组
	Create(ctx context.Context, group *model.Group) error

	// Update 更新分组信息
	Update(ctx context.Context, group *model.Group) error

	// Delete 删除分组（软删除）
	Delete(ctx context.Context, orgID, groupID string) error

	// GetByID 根据ID获取分组
	GetByID(ctx context.Context, orgID, groupID string) (*model.Group, error)

	// GetByCode 根据编码获取分组
	GetByCode(ctx context.Context, orgID, code string) (*model.Group, error)

	// List 查询分组列表
	List(ctx context.Context, filter *model.GroupFilter) (*model.GroupListResult, error)

	// Exists 检查分组是否存在
	Exists(ctx context.Context, orgID, groupID string) (bool, error)

	// GetChildren 获取子分组
	GetChildren(ctx context.Context, orgID, parentID string) ([]*model.Group, error)

	// AddMember 添加成员到分组
	AddMember(ctx context.Context, member *model.GroupMember) error

	// RemoveMember 从分组移除成员
	RemoveMember(ctx context.Context, groupID, employeeID string) error

	// GetMembers 获取分组成员列表
	GetMembers(ctx context.Context, groupID string) ([]*model.Employee, error)

	// GetMemberGroups 获取员工所属的分组列表
	GetMemberGroups(ctx context.Context, orgID, employeeID string) ([]*model.Group, error)

	// BatchGetMemberGroups 批量获取员工所属的分组列表（返回 map[employeeID][]*Group）
	BatchGetMemberGroups(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Group, error)

	// IsMember 检查员工是否在分组中
	IsMember(ctx context.Context, groupID, employeeID string) (bool, error)

	// GetGroupWithMembers 获取带成员的分组信息
	GetGroupWithMembers(ctx context.Context, orgID, groupID string) (*model.GroupWithMembers, error)

	// UpdateMemberRole 更新成员在组内的角色
	UpdateMemberRole(ctx context.Context, groupID, employeeID, role string) error
}
