package model

import "time"

// Employee 员工领域模型
type Employee struct {
	ID           string     `json:"id"`
	OrgID        string     `json:"orgId"`
	EmployeeID   string     `json:"employeeId"`
	UserID       string     `json:"userId,omitempty"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone,omitempty"`
	Email        string     `json:"email,omitempty"`
	DepartmentID string     `json:"department"` // 对应management-service的department字段
	Position     string     `json:"position,omitempty"`
	Role         string     `json:"role,omitempty"`
	Status       string     `json:"status,omitempty"`
	HireDate     *time.Time `json:"hireDate,omitempty"`
	Groups       []*Group   `json:"groups,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

// CreateEmployeeRequest 创建员工请求
type CreateEmployeeRequest struct {
	OrgID        string     `json:"orgId"`
	EmployeeID   string     `json:"employeeId"`
	UserID       string     `json:"userId,omitempty"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone,omitempty"`
	Email        string     `json:"email,omitempty"`
	DepartmentID string     `json:"department"` // 对应management-service的department字段
	Position     string     `json:"position,omitempty"`
	Role         string     `json:"role,omitempty"`
	Status       string     `json:"status,omitempty"`
	HireDate     *time.Time `json:"hireDate,omitempty"`
}

// UpdateEmployeeRequest 更新员工请求
type UpdateEmployeeRequest struct {
	OrgID        string     `json:"orgId"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone,omitempty"`
	Email        string     `json:"email,omitempty"`
	DepartmentID string     `json:"department"` // 对应management-service的department字段
	Position     string     `json:"position,omitempty"`
	Role         string     `json:"role,omitempty"`
	Status       string     `json:"status,omitempty"`
	HireDate     *time.Time `json:"hireDate,omitempty"`
}

// ListEmployeesRequest 查询员工列表请求
type ListEmployeesRequest struct {
	OrgID        string `json:"orgId"`
	DepartmentID string `json:"departmentId"`
	Keyword      string `json:"keyword"`
	Status       string `json:"status"`
	Page         int    `json:"page"`
	PageSize     int    `json:"pageSize"`
}
