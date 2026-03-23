package entity

import (
	"time"
)

// ModalityRoomEntity 机房数据库实体（对应modality_rooms表）
// 机房指放射科CT/MRI/DR等大型设备的检查室
type ModalityRoomEntity struct {
	ID          string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string     `gorm:"index;type:varchar(64);not null"`
	Code        string     `gorm:"uniqueIndex:idx_modality_room_org_code;type:varchar(64);not null"` // 机房编码
	Name        string     `gorm:"type:varchar(128);not null"`                                       // 机房名称
	Description string     `gorm:"type:text"`                                                        // 机房说明
	Location    string     `gorm:"type:varchar(256)"`                                                // 位置信息
	IsActive    bool       `gorm:"default:true"`                                                     // 是否启用
	SortOrder   int        `gorm:"default:0"`                                                        // 排序序号
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (ModalityRoomEntity) TableName() string {
	return "modality_rooms"
}
