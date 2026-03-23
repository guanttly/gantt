package model

import "time"

// Group 分组领域模型
type Group struct {
	ID          string            `json:"id"`
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	Code        string            `json:"code,omitempty"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	ParentID    *string           `json:"parentId,omitempty"`
	LeaderID    *string           `json:"leaderId,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      string            `json:"status,omitempty"`
	MemberCount int               `json:"memberCount,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty"`
}

// CreateGroupRequest 创建分组请求
type CreateGroupRequest struct {
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	Code        string            `json:"code,omitempty"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	ParentID    *string           `json:"parentId,omitempty"`
	LeaderID    *string           `json:"leaderId,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// UpdateGroupRequest 更新分组请求
type UpdateGroupRequest struct {
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	Code        string            `json:"code,omitempty"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	ParentID    *string           `json:"parentId,omitempty"`
	LeaderID    *string           `json:"leaderId,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// ListGroupsRequest 查询分组列表请求
type ListGroupsRequest struct {
	OrgID    string
	Type     string
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

// ListGroupsResponse 分组列表响应
type ListGroupsResponse struct {
	Groups     []*Group `json:"groups"`
	TotalCount int      `json:"totalCount"`
}

// GroupMembersResponse 分组成员响应
type GroupMembersResponse struct {
	Members []*Employee `json:"members"`
}

// AddGroupMemberRequest 添加成员请求
type AddGroupMemberRequest struct {
	GroupID    string `json:"groupId"`
	EmployeeID string `json:"employeeId"`
}

// RemoveGroupMemberRequest 移除成员请求
type RemoveGroupMemberRequest struct {
	GroupID    string `json:"groupId"`
	EmployeeID string `json:"employeeId"`
}
