package entity

import (
	"time"
)

// ModalityRoomWeeklyVolumeEntity 机房周检查量预估数据库实体（对应modality_room_weekly_volumes表）
// 记录每个机房按星期几、时间段、检查类型的预估检查量
type ModalityRoomWeeklyVolumeEntity struct {
	ID             string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID          string    `gorm:"index;type:varchar(64);not null"`
	ModalityRoomID string    `gorm:"uniqueIndex:idx_weekly_volume_unique;type:varchar(64);not null"` // 机房ID
	Weekday        int       `gorm:"uniqueIndex:idx_weekly_volume_unique;not null"`                  // 周几：0=周日,1=周一,...,6=周六
	TimePeriodID   string    `gorm:"uniqueIndex:idx_weekly_volume_unique;type:varchar(64);not null"` // 时间段ID
	ScanTypeID     string    `gorm:"uniqueIndex:idx_weekly_volume_unique;type:varchar(64);not null"` // 检查类型ID
	Volume         int       `gorm:"not null;default:0"`                                             // 预估检查量
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (ModalityRoomWeeklyVolumeEntity) TableName() string {
	return "modality_room_weekly_volumes"
}
