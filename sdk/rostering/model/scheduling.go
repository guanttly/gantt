package model

import "time"

// ScheduleAssignment 排班分配（对应management-service的ShiftAssignment）
type ScheduleAssignment struct {
	ID         string     `json:"id,omitempty"`
	OrgID      string     `json:"orgId,omitempty"`
	Date       string     `json:"date"` // 保持字符串格式以兼容前端
	EmployeeID string     `json:"employeeId"`
	ShiftID    string     `json:"shiftId"`
	Notes      string     `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt,omitempty"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
}

// BatchAssignRequest 批量分配请求
type BatchAssignRequest struct {
	OrgID       string                `json:"orgId,omitempty"`
	Assignments []*ScheduleAssignment `json:"assignments"`
}

// GetScheduleByDateRangeRequest 按日期范围查询排班请求
type GetScheduleByDateRangeRequest struct {
	OrgID     string
	StartDate string
	EndDate   string
}

// ScheduleResponse 排班响应
type ScheduleResponse struct {
	Schedules interface{} `json:"schedules"`
}

// GetScheduleSummaryRequest 获取排班汇总请求
type GetScheduleSummaryRequest struct {
	StartDate string
	EndDate   string
	OrgID     string
}

// ScheduleSummaryResponse 排班汇总响应
type ScheduleSummaryResponse struct {
	Summary interface{} `json:"summary"`
}
