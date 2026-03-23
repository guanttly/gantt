package entity

import (
	"time"
)

// ModalityRoomVolumeEntity 机房检查量数据库实体（对应modality_room_volumes表）
// 记录每个机房每天各时间段的检查/报告量
type ModalityRoomVolumeEntity struct {
	ID             string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID          string    `gorm:"index;type:varchar(64);not null"`
	ModalityRoomID string    `gorm:"uniqueIndex:idx_volume_room_date_period;type:varchar(64);not null"` // 机房ID
	Date           time.Time `gorm:"uniqueIndex:idx_volume_room_date_period;type:date;not null"`        // 日期
	TimePeriodID   string    `gorm:"uniqueIndex:idx_volume_room_date_period;type:varchar(64);not null"` // 时间段ID
	ReportVolume   int       `gorm:"not null;default:0"`                                                // 报告量/检查量
	Notes          string    `gorm:"type:text"`                                                         // 备注
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (ModalityRoomVolumeEntity) TableName() string {
	return "modality_room_volumes"
}
