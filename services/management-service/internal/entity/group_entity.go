package entity

import (
	"time"
)

// GroupEntity 分组数据库实体（对应groups表）
type GroupEntity struct {
	ID          string     `gorm:"primaryKey;type:varchar(64)"`
	OrgID       string     `gorm:"index;type:varchar(64);not null"`
	Name        string     `gorm:"type:varchar(128);not null"`
	Code        string     `gorm:"uniqueIndex:idx_org_code;type:varchar(64);not null"`
	Type        string     `gorm:"type:varchar(32)"`
	Description string     `gorm:"type:text"`
	ParentID    *string    `gorm:"index;type:varchar(64)"`
	LeaderID    *string    `gorm:"type:varchar(64)"`
	Attributes  string     `gorm:"type:text"` // JSON字符串
	Status      string     `gorm:"type:varchar(32);default:'active'"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"index"`
}

// TableName 表名
func (GroupEntity) TableName() string {
	return "groups"
}

// GroupMemberEntity 分组成员关系数据库实体（对应group_members表）
type GroupMemberEntity struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	GroupID    string    `gorm:"uniqueIndex:idx_group_employee;type:varchar(64);not null"`
	EmployeeID string    `gorm:"uniqueIndex:idx_group_employee;type:varchar(64);not null"`
	Role       string    `gorm:"type:varchar(64)"`
	JoinedAt   time.Time `gorm:"autoCreateTime"`
	LeftAt     *time.Time
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (GroupMemberEntity) TableName() string {
	return "group_members"
}
