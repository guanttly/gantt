package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IGroupService 分组服务接口
type IGroupService interface {
	Create(ctx context.Context, req *model.CreateGroupRequest) (*model.Group, error)
	GetList(ctx context.Context, req *model.ListGroupsRequest) (*model.ListGroupsResponse, error)
	Get(ctx context.Context, id string) (*model.Group, error)
	Update(ctx context.Context, id string, req *model.UpdateGroupRequest) (*model.Group, error)
	Delete(ctx context.Context, id string) error
	GetMembers(ctx context.Context, groupID string) (*model.GroupMembersResponse, error)
	AddMember(ctx context.Context, req *model.AddGroupMemberRequest) error
	RemoveMember(ctx context.Context, req *model.RemoveGroupMemberRequest) error
}
