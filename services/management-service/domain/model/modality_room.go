package model

import (
	"time"
)

// ModalityRoom 机房领域模型
// 机房指放射科CT/MRI/DR等大型设备的检查室
type ModalityRoom struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code"`        // 机房编码
	Name        string     `json:"name"`        // 机房名称
	Description string     `json:"description"` // 机房说明
	Location    string     `json:"location"`    // 位置信息
	IsActive    bool       `json:"isActive"`    // 是否启用
	SortOrder   int        `json:"sortOrder"`   // 排序序号
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// ModalityRoomFilter 机房查询过滤器
type ModalityRoomFilter struct {
	OrgID    string
	Keyword  string // 按名称、编码模糊搜索
	IsActive *bool  // 是否启用
	Page     int
	PageSize int
}

// ModalityRoomListResult 机房列表结果
type ModalityRoomListResult struct {
	Items    []*ModalityRoom `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}
