package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的分组类型
type Group = sdk_model.Group
type CreateGroupRequest = sdk_model.CreateGroupRequest
type UpdateGroupRequest = sdk_model.UpdateGroupRequest
type ListGroupsRequest = sdk_model.ListGroupsRequest
type ListGroupsResponse = sdk_model.ListGroupsResponse
type AddGroupMemberRequest = sdk_model.AddGroupMemberRequest
type RemoveGroupMemberRequest = sdk_model.RemoveGroupMemberRequest
