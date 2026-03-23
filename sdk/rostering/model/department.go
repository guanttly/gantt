package model

import "time"

// Department 部门实体 - 与 MCP Server 保持一致
type Department struct {
	ID            string        `json:"id"`
	OrgID         string        `json:"orgId"`
	Code          string        `json:"code,omitempty"`
	Name          string        `json:"name"`
	ParentID      *string       `json:"parentId,omitempty"`
	Level         int           `json:"level,omitempty"`
	Path          string        `json:"path,omitempty"`
	Description   string        `json:"description,omitempty"`
	ManagerID     *string       `json:"managerId,omitempty"`
	SortOrder     int           `json:"sortOrder,omitempty"`
	IsActive      bool          `json:"isActive,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	DeletedAt     *time.Time    `json:"deletedAt,omitempty"`
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
	*Department
	Children []*DepartmentTreeNode `json:"children,omitempty"`
}

// GetDepartmentTreeRequest 获取部门树请求
type GetDepartmentTreeRequest struct {
	OrgID    string `json:"orgId"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}
