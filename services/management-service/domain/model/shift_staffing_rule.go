package model

import (
	"time"
)

// RoundingMode 取整方式
type RoundingMode string

const (
	RoundingModeCeil  RoundingMode = "ceil"  // 向上取整
	RoundingModeFloor RoundingMode = "floor" // 向下取整
)

// ShiftStaffingRule 班次排班人数计算规则领域模型
type ShiftStaffingRule struct {
	ID              string          `json:"id"`
	ShiftID         string          `json:"shiftId"`         // 班次ID
	ShiftName       string          `json:"shiftName"`       // 班次名称（冗余，方便展示）
	ModalityRoomIDs []string        `json:"modalityRoomIds"` // 关联的机房ID列表
	ModalityRooms   []*ModalityRoom `json:"modalityRooms"`   // 关联的机房详情（冗余，方便展示）
	TimePeriodID    string          `json:"timePeriodId"`    // 时间段ID
	TimePeriodName  string          `json:"timePeriodName"`  // 时间段名称（冗余，方便展示）
	AvgReportLimit  int             `json:"avgReportLimit"`  // 人均报告处理上限
	RoundingMode    RoundingMode    `json:"roundingMode"`    // 取整方式
	IsActive        bool            `json:"isActive"`        // 是否启用
	Description     string          `json:"description"`     // 规则说明
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// CreateShiftStaffingRuleRequest 创建/更新规则请求
type CreateShiftStaffingRuleRequest struct {
	ShiftID         string       `json:"shiftId"`
	ModalityRoomIDs []string     `json:"modalityRoomIds"`
	TimePeriodID    string       `json:"timePeriodId"`
	AvgReportLimit  int          `json:"avgReportLimit"`
	RoundingMode    RoundingMode `json:"roundingMode"`
	Description     string       `json:"description"`
}

// DailyStaffingResult 单日排班人数计算结果
type DailyStaffingResult struct {
	Weekday         int    `json:"weekday"`         // 周几：0=周日,1=周一,...,6=周六
	WeekdayName     string `json:"weekdayName"`     // 周几名称
	DailyVolume     int    `json:"dailyVolume"`     // 当日检查量
	CalculatedCount int    `json:"calculatedCount"` // 计算推荐人数
	CurrentCount    int    `json:"currentCount"`    // 当前配置人数（来自shift_weekly_staff）
}

// StaffingCalculationPreview 排班人数计算预览
type StaffingCalculationPreview struct {
	ShiftID          string                       `json:"shiftId"`
	ShiftName        string                       `json:"shiftName"`
	TimePeriodID     string                       `json:"timePeriodId"`
	TimePeriodName   string                       `json:"timePeriodName"`
	ModalityRooms    []*ModalityRoomVolumeSummary `json:"modalityRooms"`    // 各机房报告量明细
	TotalVolume      int                          `json:"totalVolume"`      // 总报告量
	DataDays         int                          `json:"dataDays"`         // 实际有数据的天数
	WeeklyVolume     int                          `json:"weeklyVolume"`     // 折算周报告量（不足7天按平均*7）
	AvgReportLimit   int                          `json:"avgReportLimit"`   // 使用的人均上限
	RoundingMode     RoundingMode                 `json:"roundingMode"`     // 取整方式
	CalculatedCount  int                          `json:"calculatedCount"`  // 计算推荐总人数（周总量计算）
	DailyResults     []*DailyStaffingResult       `json:"dailyResults"`     // 每日计算结果
	CalculationSteps string                       `json:"calculationSteps"` // 计算过程说明
}

// ApplyStaffCountRequest 确认写入排班人数请求
type ApplyStaffCountRequest struct {
	ShiftID    string `json:"shiftId"`
	StaffCount int    `json:"staffCount"`         // 用户确认或调整后的值
	ApplyMode  string `json:"applyMode"`          // 写入模式：default=写入通用默认值，weekly=写入周配置
	Weekdays   []int  `json:"weekdays,omitempty"` // ApplyMode=weekly时，指定要写入的周几（0-6）
}

// ApplyStaffCountResult 写入结果
type ApplyStaffCountResult struct {
	ShiftID       string `json:"shiftId"`
	AppliedCount  int    `json:"appliedCount"`
	ApplyMode     string `json:"applyMode"`
	AffectedDays  []int  `json:"affectedDays,omitempty"` // 受影响的周几
	PreviousCount int    `json:"previousCount"`          // 之前的值
	Message       string `json:"message"`
}
