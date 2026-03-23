package model

import "time"

// Leave 请假领域模型（对应management-service的LeaveRecord）
type Leave struct {
	ID           string     `json:"id"`
	OrgID        string     `json:"orgId,omitempty"`
	EmployeeID   string     `json:"employeeId"`
	EmployeeName string     `json:"employeeName,omitempty"`
	Type         string     `json:"type"`
	LeaveType    string     `json:"leaveType,omitempty"` // 向后兼容
	StartDate    string     `json:"startDate"`           // 保持字符串格式以兼容前端
	EndDate      string     `json:"endDate"`
	Days         float64    `json:"days,omitempty"`
	StartTime    *string    `json:"startTime,omitempty"`
	EndTime      *string    `json:"endTime,omitempty"`
	Status       string     `json:"status,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

// CreateLeaveRequest 创建请假请求
type CreateLeaveRequest struct {
	OrgID      string  `json:"orgId,omitempty"`
	EmployeeID string  `json:"employeeId"`
	Type       string  `json:"type"`
	StartDate  string  `json:"startDate"`
	EndDate    string  `json:"endDate"`
	Days       float64 `json:"days,omitempty"`
	StartTime  *string `json:"startTime,omitempty"`
	EndTime    *string `json:"endTime,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

// UpdateLeaveRequest 更新请假请求
type UpdateLeaveRequest struct {
	OrgID     string  `json:"orgId,omitempty"`
	Type      string  `json:"type"`
	StartDate string  `json:"startDate"`
	EndDate   string  `json:"endDate"`
	Days      float64 `json:"days,omitempty"`
	StartTime *string `json:"startTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Reason    string  `json:"reason,omitempty"`
	Status    string  `json:"status,omitempty"`
}

// ListLeavesRequest 查询请假列表请求
type ListLeavesRequest struct {
	OrgID      string
	EmployeeID string
	Type       string
	StartDate  string
	EndDate    string
	Status     string
	Keyword    string
	Page       int
	PageSize   int
}

// ListLeavesResponse 请假列表响应
type ListLeavesResponse struct {
	Leaves     []*Leave `json:"leaves"`
	TotalCount int      `json:"totalCount"`
}

// LeaveBalance 假期余额
type LeaveBalance struct {
	ID         uint64  `json:"id,omitempty"`
	OrgID      string  `json:"orgId,omitempty"`
	EmployeeID string  `json:"employeeId,omitempty"`
	LeaveType  string  `json:"leaveType,omitempty"` // 向后兼容
	Type       string  `json:"type,omitempty"`
	Year       int     `json:"year,omitempty"`
	Total      float64 `json:"total"`
	Used       float64 `json:"used"`
	Remaining  float64 `json:"remaining"`
}
