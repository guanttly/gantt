package model

import "time"

// Department 部门领域模型
type Department struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code,omitempty"`
	Name        string     `json:"name"`
	ParentID    *string    `json:"parentId,omitempty"`
	Level       int        `json:"level,omitempty"`
	Path        string     `json:"path,omitempty"`
	Description string     `json:"description,omitempty"`
	ManagerID   *string    `json:"managerId,omitempty"`
	SortOrder   int        `json:"sortOrder,omitempty"`
	IsActive    bool       `json:"isActive,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`

	// 扩展字段（不存储在数据库）
	ParentName    string        `json:"parentName,omitempty"`
	ManagerName   string        `json:"managerName,omitempty"`
	EmployeeCount int           `json:"employeeCount,omitempty"`
	Children      []*Department `json:"children,omitempty"`
}

// CreateDepartmentRequest 创建部门请求
type CreateDepartmentRequest struct {
	OrgID       string  `json:"orgId"`
	Code        string  `json:"code,omitempty"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId,omitempty"`
	Description string  `json:"description,omitempty"`
	ManagerID   *string `json:"managerId,omitempty"`
	SortOrder   int     `json:"sortOrder,omitempty"`
	IsActive    bool    `json:"isActive,omitempty"`
}

// UpdateDepartmentRequest 更新部门请求
type UpdateDepartmentRequest struct {
	OrgID       string  `json:"orgId"`
	Code        string  `json:"code,omitempty"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId,omitempty"`
	Description string  `json:"description,omitempty"`
	ManagerID   *string `json:"managerId,omitempty"`
	SortOrder   int     `json:"sortOrder,omitempty"`
	IsActive    bool    `json:"isActive,omitempty"`
}

// ListDepartmentsResponse 部门列表响应
type ListDepartmentsResponse struct {
	Departments []*Department `json:"departments"`
	TotalCount  int           `json:"totalCount"`
}

// DepartmentTreeNode 部门树节点
type DepartmentTreeNode struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	ParentID string                `json:"parentId,omitempty"`
	Children []*DepartmentTreeNode `json:"children,omitempty"`
}
