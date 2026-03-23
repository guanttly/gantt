package model

import (
	"time"
)

// Group 工作组/分组领域模型
type Group struct {
	ID          string            `json:"id"`
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	Code        string            `json:"code"`
	Type        GroupType         `json:"type"`
	Description string            `json:"description"`
	ParentID    *string           `json:"parentId,omitempty"`
	LeaderID    *string           `json:"leaderId,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      GroupStatus       `json:"status"`
	MemberCount int               `json:"memberCount"` // 成员数量
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty"`
}

// GroupType 分组类型
type GroupType string

const (
	GroupTypeDepartment GroupType = "department" // 部门
	GroupTypeTeam       GroupType = "team"       // 团队
	GroupTypeShift      GroupType = "shift"      // 班组
	GroupTypeProject    GroupType = "project"    // 项目组
	GroupTypeCustom     GroupType = "custom"     // 自定义
)

// GroupStatus 分组状态
type GroupStatus string

const (
	GroupStatusActive   GroupStatus = "active"   // 活跃
	GroupStatusInactive GroupStatus = "inactive" // 停用
	GroupStatusArchived GroupStatus = "archived" // 归档
)

// IsActive 是否活跃
func (g *Group) IsActive() bool {
	return g.Status == GroupStatusActive
}

// GroupMember 分组成员关系领域模型
type GroupMember struct {
	ID         uint64     `json:"id"`
	GroupID    string     `json:"groupId"`
	EmployeeID string     `json:"employeeId"`
	Role       string     `json:"role"`
	JoinedAt   time.Time  `json:"joinedAt"`
	LeftAt     *time.Time `json:"leftAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// IsActive 成员是否在组内
func (gm *GroupMember) IsActive() bool {
	return gm.LeftAt == nil
}

// GroupFilter 分组查询过滤器
type GroupFilter struct {
	OrgID    string
	Type     GroupType
	Status   GroupStatus
	ParentID *string
	Keyword  string // 按名称、编码模糊搜索
	Page     int
	PageSize int
}

// GroupListResult 分组列表结果
type GroupListResult struct {
	Items    []*Group `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// GroupWithMembers 带成员的分组
type GroupWithMembers struct {
	*Group
	Members     []*Employee `json:"members"`
	MemberCount int         `json:"memberCount"`
}
