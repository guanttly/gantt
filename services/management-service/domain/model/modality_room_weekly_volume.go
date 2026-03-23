package model

import (
	"time"
)

// ModalityRoomWeeklyVolume 机房周检查量预估领域模型
type ModalityRoomWeeklyVolume struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"orgId"`
	ModalityRoomID string    `json:"modalityRoomId"` // 机房ID
	Weekday        int       `json:"weekday"`        // 周几：0=周日,1=周一,...,6=周六
	TimePeriodID   string    `json:"timePeriodId"`   // 时间段ID
	ScanTypeID     string    `json:"scanTypeId"`     // 检查类型ID
	Volume         int       `json:"volume"`         // 预估检查量
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// 冗余字段，用于展示
	TimePeriodName string `json:"timePeriodName,omitempty"`
	ScanTypeName   string `json:"scanTypeName,omitempty"`
}

// WeeklyVolumeItem 周检查量配置项（用于批量保存）
type WeeklyVolumeItem struct {
	Weekday      int    `json:"weekday"`      // 周几
	TimePeriodID string `json:"timePeriodId"` // 时间段ID
	ScanTypeID   string `json:"scanTypeId"`   // 检查类型ID
	Volume       int    `json:"volume"`       // 预估检查量
}

// WeeklyVolumeListResult 周检查量列表结果（带展示信息）
type WeeklyVolumeListResult struct {
	Items []*ModalityRoomWeeklyVolume `json:"items"`
}

// WeeklyVolumeSaveRequest 批量保存周检查量请求
type WeeklyVolumeSaveRequest struct {
	ModalityRoomID string              `json:"modalityRoomId"`
	Items          []*WeeklyVolumeItem `json:"items"`
}

// 注意：GetWeekdayName 函数已在 shift_weekly_staff.go 中定义，这里复用该函数
