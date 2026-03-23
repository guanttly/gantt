package model

import "time"

// Shift 班次领域模型
type Shift struct {
	ID                 string     `json:"id"`
	OrgID              string     `json:"orgId"`
	Name               string     `json:"name"`
	Code               string     `json:"code,omitempty"`
	Type               string     `json:"type,omitempty"`
	Description        string     `json:"description,omitempty"`
	StartTime          string     `json:"startTime"`
	EndTime            string     `json:"endTime"`
	Duration           int        `json:"duration,omitempty"` // 时长（分钟）
	IsOvernight        bool       `json:"isOvernight,omitempty"`
	Color              string     `json:"color,omitempty"`
	Priority           int        `json:"priority,omitempty"`
	SchedulingPriority int        `json:"schedulingPriority"`
	IsActive           bool       `json:"isActive,omitempty"`
	Status             string     `json:"status,omitempty"` // 保留向后兼容
	RestDuration       float64    `json:"restDuration,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	DeletedAt          *time.Time `json:"deletedAt,omitempty"`
}

// CreateShiftRequest 创建班次请求
type CreateShiftRequest struct {
	OrgID              string  `json:"orgId"`
	Name               string  `json:"name"`
	Code               string  `json:"code,omitempty"`
	Type               string  `json:"type,omitempty"`
	Description        string  `json:"description,omitempty"`
	StartTime          string  `json:"startTime"`
	EndTime            string  `json:"endTime"`
	Duration           int     `json:"duration,omitempty"`
	IsOvernight        bool    `json:"isOvernight,omitempty"`
	Color              string  `json:"color,omitempty"`
	Priority           int     `json:"priority,omitempty"`
	SchedulingPriority int     `json:"schedulingPriority"`
	IsActive           bool    `json:"isActive,omitempty"`
	RestDuration       float64 `json:"restDuration,omitempty"`
}

// UpdateShiftRequest 更新班次请求
type UpdateShiftRequest struct {
	OrgID              string  `json:"orgId"`
	Name               string  `json:"name"`
	Code               string  `json:"code,omitempty"`
	Type               string  `json:"type,omitempty"`
	Description        string  `json:"description,omitempty"`
	StartTime          string  `json:"startTime"`
	EndTime            string  `json:"endTime"`
	Duration           int     `json:"duration,omitempty"`
	IsOvernight        bool    `json:"isOvernight,omitempty"`
	Color              string  `json:"color,omitempty"`
	Priority           int     `json:"priority,omitempty"`
	SchedulingPriority int     `json:"schedulingPriority"`
	IsActive           bool    `json:"isActive,omitempty"`
	RestDuration       float64 `json:"restDuration,omitempty"`
}

// ListShiftsRequest 查询班次列表请求
type ListShiftsRequest struct {
	OrgID    string
	Page     int
	PageSize int
	Keyword  string
	Status   string
}

// SetShiftGroupsRequest 设置班次分组请求
type SetShiftGroupsRequest struct {
	GroupIDs []string `json:"groupIds"`
}

// AddShiftGroupRequest 添加班次分组请求
type AddShiftGroupRequest struct {
	GroupID  string `json:"groupId"`
	Priority int    `json:"priority"`
}

// ShiftGroup 班次关联分组
type ShiftGroup struct {
	ID        uint64 `json:"id"`
	ShiftID   string `json:"shiftId"`
	GroupID   string `json:"groupId"`
	Priority  int    `json:"priority"`
	IsActive  bool   `json:"isActive"`
	GroupName string `json:"groupName"`
	GroupCode string `json:"groupCode"`
}
