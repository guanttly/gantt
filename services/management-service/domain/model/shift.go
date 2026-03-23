package model

import (
	"fmt"
	"time"
)

// Shift 班次领域模型
type Shift struct {
	ID                 string     `json:"id"`
	OrgID              string     `json:"orgId"`
	Name               string     `json:"name"`
	Code               string     `json:"code"`
	Type               ShiftType  `json:"type"`
	Description        string     `json:"description"`
	StartTime          string     `json:"startTime"` // 格式: HH:MM
	EndTime            string     `json:"endTime"`   // 格式: HH:MM
	Duration           int        `json:"duration"`  // 时长（分钟）
	IsOvernight        bool       `json:"isOvernight"`
	Color              string     `json:"color"`
	Priority           int        `json:"priority"`
	SchedulingPriority int        `json:"schedulingPriority"` // 排班优先级（用于排班排序）
	IsActive           bool       `json:"isActive"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	DeletedAt          *time.Time `json:"deletedAt,omitempty"`

	// 扩展字段（不持久化，用于列表返回）
	WeeklyStaffSummary string `json:"weeklyStaffSummary,omitempty"` // 周人数摘要，如"工作日2人/周末1人"
}

// ShiftType 班次类型
type ShiftType string

const (
	ShiftTypeMorning   ShiftType = "morning"   // 早班
	ShiftTypeAfternoon ShiftType = "afternoon" // 中班
	ShiftTypeEvening   ShiftType = "evening"   // 晚班
	ShiftTypeNight     ShiftType = "night"     // 夜班
	ShiftTypeFullDay   ShiftType = "full_day"  // 全天
	ShiftTypeCustom    ShiftType = "custom"    // 自定义
)

// GetDurationHours 获取时长（小时）
func (s *Shift) GetDurationHours() float64 {
	return float64(s.Duration) / 60.0
}

// ValidateShiftTime 验证班次时间格式和逻辑
func (s *Shift) ValidateShiftTime() error {
	// 简单验证: 检查时间格式 HH:MM
	if len(s.StartTime) != 5 || s.StartTime[2] != ':' {
		return fmt.Errorf("invalid startTime format, expected HH:MM, got %s", s.StartTime)
	}
	if len(s.EndTime) != 5 || s.EndTime[2] != ':' {
		return fmt.Errorf("invalid endTime format, expected HH:MM, got %s", s.EndTime)
	}
	return nil
}

// ShiftFilter 班次查询过滤器
type ShiftFilter struct {
	OrgID    string
	Type     *ShiftType
	IsActive *bool  // 是否启用
	Keyword  string // 按名称、编码模糊搜索
	Page     int
	PageSize int
}

// ShiftListResult 班次列表结果
type ShiftListResult struct {
	Items    []*Shift `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// ShiftAssignment 班次分配领域模型
type ShiftAssignment struct {
	ID         string     `json:"id"`
	OrgID      string     `json:"orgId"`
	EmployeeID string     `json:"employeeId"`
	ShiftID    string     `json:"shiftId"`
	Date       time.Time  `json:"date"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
}

// ShiftGroup 班次与分组的关联领域模型
type ShiftGroup struct {
	ID        uint64    `json:"id"`
	ShiftID   string    `json:"shiftId"`
	GroupID   string    `json:"groupId"`
	Priority  int       `json:"priority"`  // 优先级，数字越小优先级越高
	IsActive  bool      `json:"isActive"`  // 是否启用
	Notes     string    `json:"notes"`     // 备注说明
	GroupName string    `json:"groupName"` // 分组名称（冗余字段，用于前端展示）
	GroupCode string    `json:"groupCode"` // 分组编码（冗余字段，用于前端展示）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
