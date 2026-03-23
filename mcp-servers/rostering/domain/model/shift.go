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
	WeeklyStaffSummary string     `json:"weeklyStaffSummary,omitempty"` // 周人数配置摘要
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

// ListShiftsResponse 班次列表响应
type ListShiftsResponse struct {
	Shifts     []*Shift `json:"shifts"`
	TotalCount int      `json:"totalCount"`
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

// WeekdayStaffConfig 单日人数配置
type WeekdayStaffConfig struct {
	Weekday     int    `json:"weekday"`     // 0=周日,1=周一,...,6=周六
	WeekdayName string `json:"weekdayName"` // 周日/周一/.../周六
	StaffCount  int    `json:"staffCount"`  // 人数
	IsCustom    bool   `json:"isCustom"`    // 是否自定义配置
}

// ShiftWeeklyStaffConfig 班次周人数配置
type ShiftWeeklyStaffConfig struct {
	ShiftID      string                `json:"shiftId"`
	ShiftName    string                `json:"shiftName,omitempty"`
	WeeklyConfig []*WeekdayStaffConfig `json:"weeklyConfig"` // 7天配置
}

// SetShiftWeeklyStaffRequest 设置班次周人数请求
type SetShiftWeeklyStaffRequest struct {
	WeeklyConfig []struct {
		Weekday    int `json:"weekday"`
		StaffCount int `json:"staffCount"`
	} `json:"weeklyConfig"`
}

// DailyStaffingResult 每日排班人数计算结果
type DailyStaffingResult struct {
	Weekday     int    `json:"weekday"`     // 0-6
	WeekdayName string `json:"weekdayName"` // 周日/周一/.../周六
	Volume      int    `json:"volume"`      // 当日检查量
	StaffCount  int    `json:"staffCount"`  // 推荐人数
}

// StaffingCalculationPreview 排班人数计算预览
type StaffingCalculationPreview struct {
	ShiftID          string                 `json:"shiftId"`
	ShiftName        string                 `json:"shiftName"`
	TimePeriodID     string                 `json:"timePeriodId"`
	TimePeriodName   string                 `json:"timePeriodName"`
	ModalityRooms    []map[string]any       `json:"modalityRooms"`
	TotalVolume      int                    `json:"totalVolume"`
	DataDays         int                    `json:"dataDays"`
	WeeklyVolume     int                    `json:"weeklyVolume"`
	AvgReportLimit   int                    `json:"avgReportLimit"`
	RoundingMode     string                 `json:"roundingMode"`
	CalculatedCount  int                    `json:"calculatedCount"`
	CurrentCount     int                    `json:"currentCount"`
	CalculationSteps string                 `json:"calculationSteps"`
	DailyResults     []*DailyStaffingResult `json:"dailyResults"` // 每日计算结果
}
