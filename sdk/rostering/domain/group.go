package domain

import (
	"context"
	"jusha/agent/sdk/rostering/model"
)

// IGroupService 分组管理接口
type IGroupService interface {
	// CreateGroup 创建分组
	CreateGroup(ctx context.Context, req *model.CreateGroupRequest) (string, error)

	// UpdateGroup 更新分组信息
	UpdateGroup(ctx context.Context, id string, req *model.UpdateGroupRequest) error

	// ListGroups 获取分组列表
	ListGroups(ctx context.Context, req *model.ListGroupsRequest) (*model.ListGroupsResponse, error)

	// AddGroupMember 添加分组成员
	AddGroupMember(ctx context.Context, req *model.AddGroupMemberRequest) error

	// RemoveGroupMember 移除分组成员
	RemoveGroupMember(ctx context.Context, req *model.RemoveGroupMemberRequest) error

	// GetGroupMembers 获取分组成员
	GetGroupMembers(ctx context.Context, groupID string) (*model.GroupMembersResponse, error)
}
