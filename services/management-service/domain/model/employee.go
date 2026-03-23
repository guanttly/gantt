package model

import (
	"time"
)

// Employee 员工领域模型（业务层使用）
type Employee struct {
	ID           string         `json:"id"`
	OrgID        string         `json:"orgId"`
	EmployeeID   string         `json:"employeeId"` // 工号
	UserID       string         `json:"userId"`     // 关联用户ID
	Name         string         `json:"name"`
	Phone        string         `json:"phone"`
	Email        string         `json:"email"`
	DepartmentID string         `json:"department"` // 关联部门ID（前端显示为department）
	Position     string         `json:"position"`   // 职位
	Role         string         `json:"role"`       // 角色（权限相关）
	Status       EmployeeStatus `json:"status"`
	HireDate     *time.Time     `json:"hireDate,omitempty"` // 入职日期
	Groups       []*Group       `json:"groups,omitempty"`   // 所属分组列表
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    *time.Time     `json:"deletedAt,omitempty"` // 软删除时间戳
}

// EmployeeStatus 员工状态
type EmployeeStatus string

const (
	EmployeeStatusActive     EmployeeStatus = "active"      // 在职
	EmployeeStatusInactive   EmployeeStatus = "inactive"    // 离职
	EmployeeStatusLeave      EmployeeStatus = "leave"       // 休假中
	EmployeeStatusSuspend    EmployeeStatus = "suspend"     // 暂停排班
	EmployeeStatusStudyLeave EmployeeStatus = "study_leave" // 进修
)

// IsActive 是否在职
func (e *Employee) IsActive() bool {
	return e.Status == EmployeeStatusActive
}

// CanBeScheduled 是否可以被排班
func (e *Employee) CanBeScheduled() bool {
	return e.Status == EmployeeStatusActive
}

// EmployeeFilter 员工查询过滤器
type EmployeeFilter struct {
	OrgID         string
	Department    string
	Position      string
	Role          string
	Status        EmployeeStatus
	Skills        []string
	Keyword       string // 按姓名、工号、手机号模糊搜索
	IncludeGroups bool   // 是否加载分组信息（默认 false，仅在员工管理页面需要时设为 true）
	Page          int
	PageSize      int
}

// EmployeeListResult 员工列表结果
type EmployeeListResult struct {
	Items    []*Employee `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
