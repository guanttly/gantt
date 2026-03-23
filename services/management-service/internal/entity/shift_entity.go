package entity

import (
	"time"
)

// ShiftEntity 班次数据库实体（对应shifts表）
type ShiftEntity struct {
	ID                 string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID              string     `gorm:"index;type:varchar(64);not null"`
	Name               string     `gorm:"type:varchar(128);not null"`                         // 班次名称：如"本部穿刺班"、"CT/MRI审核上午班"
	Code               string     `gorm:"uniqueIndex:idx_org_code;type:varchar(64);not null"` // 班次编码：用于规则匹配
	Type               string     `gorm:"type:varchar(32)"`                                   // 班次类型：morning/afternoon/night/full_day
	Description        string     `gorm:"type:text"`                                          // 班次说明
	StartTime          string     `gorm:"type:varchar(8);not null"`                           // 开始时间：08:00
	EndTime            string     `gorm:"type:varchar(8);not null"`                           // 结束时间：17:00
	Duration           int        `gorm:""`                                                   // 班次时长（分钟）
	IsOvernight        bool       `gorm:""`                                                   // 是否跨夜班次
	Color              string     `gorm:"type:varchar(32)"`                                   // 显示颜色
	Priority           int        `gorm:"default:0"`                                          // 优先级（用于自动排班时的优先顺序）
	SchedulingPriority int        `gorm:"default:0"`                                          // 排班优先级（用于排班排序）
	IsActive           bool       `gorm:"default:true"`                                       // 是否启用
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime"`
	DeletedAt          *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (ShiftEntity) TableName() string {
	return "shifts"
}

// ShiftAssignmentEntity 班次分配数据库实体（对应shift_assignments表）
// 记录实际的排班结果
type ShiftAssignmentEntity struct {
	ID         string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID      string     `gorm:"index;type:varchar(64);not null"`
	EmployeeID string     `gorm:"index;type:varchar(64);not null"` // 员工ID
	ShiftID    string     `gorm:"index;type:varchar(64);not null"` // 班次ID
	Date       time.Time  `gorm:"type:date;index;not null"`        // 排班日期
	Notes      string     `gorm:"type:text"`                       // 备注
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
	DeletedAt  *time.Time `gorm:"index"` // 软删除
}

// TableName 表名
func (ShiftAssignmentEntity) TableName() string {
	return "shift_assignments"
}
