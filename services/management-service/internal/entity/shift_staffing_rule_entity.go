package entity

import (
	"time"
)

// ShiftStaffingRuleEntity 班次排班人数计算规则数据库实体（对应shift_staffing_rules表）
// 每个班次可配置一个计算规则，用于根据机房报告量自动计算默认人数
type ShiftStaffingRuleEntity struct {
	ID              string    `gorm:"primaryKey;type:varchar(64)"`
	ShiftID         string    `gorm:"uniqueIndex;type:varchar(64);not null"`       // 班次ID（一对一关系）
	ModalityRoomIDs string    `gorm:"column:modality_room_ids;type:text;not null"` // 关联的机房ID列表（JSON数组）
	TimePeriodID    string    `gorm:"type:varchar(64);not null"`                   // 时间段ID
	AvgReportLimit  int       `gorm:"default:0"`                                   // 人均报告处理上限（0表示使用全局默认值）
	RoundingMode    string    `gorm:"type:varchar(16);not null;default:'ceil'"`    // 取整方式：ceil=向上取整，floor=向下取整
	IsActive        bool      `gorm:"default:true"`                                // 是否启用
	Description     string    `gorm:"type:text"`                                   // 规则说明
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (ShiftStaffingRuleEntity) TableName() string {
	return "shift_staffing_rules"
}
