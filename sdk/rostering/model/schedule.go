package model

import "time"

// 排班管理相关的请求和响应结构

// ScheduleQueryRequest 排班查询请求
type ScheduleQueryRequest struct {
	UserID    string `json:"userId,omitempty"`
	StartDate string `json:"startDate,omitempty"` // YYYY-MM-DD
	EndDate   string `json:"endDate,omitempty"`   // YYYY-MM-DD
	OrgID     string `json:"orgId,omitempty"`
	Status    string `json:"status,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

// ScheduleQueryResponse 排班查询响应
type ScheduleQueryResponse struct {
	Items []ScheduleEntry `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
}

// ScheduleEntry 排班条目实体
type ScheduleEntry struct {
	ID               string    `json:"id"`
	UserID           string    `json:"userId"`
	WorkDate         string    `json:"workDate"` // YYYY-MM-DD
	ShiftCode        string    `json:"shiftCode"`
	ShiftID          *string   `json:"shiftId,omitempty"`
	TeamID           *string   `json:"teamId,omitempty"`
	StaffID          *string   `json:"staffId,omitempty"`
	ClassificationID *string   `json:"classificationId,omitempty"`
	StartTime        string    `json:"startTime,omitempty"`
	EndTime          string    `json:"endTime,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	OrgID            string    `json:"orgId"`
	Status           string    `json:"status"`
	ApprovalID       *string   `json:"approvalId,omitempty"`
	CreatedAt        time.Time `json:"createdAt,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt,omitempty"`
}

// ScheduleUpsertRequest 排班创建/更新请求
type ScheduleUpsertRequest struct {
	UserID    string `json:"userId"`
	WorkDate  string `json:"workDate"` // YYYY-MM-DD
	ShiftCode string `json:"shiftCode"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
	Notes     string `json:"notes,omitempty"`
	OrgID     string `json:"orgId,omitempty"`
	Status    string `json:"status,omitempty"`
}

// ScheduleUpsertResponse 排班创建/更新响应
type ScheduleUpsertResponse struct {
	Entry ScheduleEntry `json:"entry"`
}

// ScheduleBatchRequest 批量排班处理请求
type ScheduleBatchRequest struct {
	Items []ScheduleUpsertRequest `json:"items"`
}

// ScheduleBatchResponse 批量排班处理响应
type ScheduleBatchResponse struct {
	Result BatchUpsertResult `json:"result"`
}

// BatchUpsertResult 批量处理结果
type BatchUpsertResult struct {
	Total    int               `json:"total"`
	Upserted int               `json:"upserted"`
	Failed   int               `json:"failed"`
	Details  []BatchItemResult `json:"details,omitempty"`
}

// BatchItemResult 批量处理项目结果
type BatchItemResult struct {
	Index   int            `json:"index"`
	Success bool           `json:"success"`
	Entry   *ScheduleEntry `json:"entry,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ScheduleDeleteRequest 排班删除请求
type ScheduleDeleteRequest struct {
	UserID   string `json:"userId"`
	WorkDate string `json:"workDate"` // YYYY-MM-DD
}
