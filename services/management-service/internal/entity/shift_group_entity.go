package entity

import (
	"time"
)

// ShiftGroupEntity 班次与分组的关联表（多对多）
// 业务含义：某个班次的人员要从哪些分组中挑选
// 例如：神经巡诊-早班 可以从CT组、MR组选人
type ShiftGroupEntity struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	ShiftID   string    `gorm:"uniqueIndex:idx_shift_group;type:varchar(64);not null"` // 班次ID
	GroupID   string    `gorm:"uniqueIndex:idx_shift_group;type:varchar(64);not null"` // 分组ID
	Priority  int       `gorm:"default:0"`                                             // 优先级，数字越小优先级越高
	IsActive  bool      `gorm:"default:true"`                                          // 是否启用
	Notes     string    `gorm:"type:text"`                                             // 备注说明
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (ShiftGroupEntity) TableName() string {
	return "shift_groups"
}
