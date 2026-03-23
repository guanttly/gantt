package model

import (
	"time"
)

// ScanType 检查类型领域模型
// 放射科检查类型，如平扫、增强等
type ScanType struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code"`        // 类型编码
	Name        string     `json:"name"`        // 类型名称
	Description string     `json:"description"` // 类型说明
	IsActive    bool       `json:"isActive"`    // 是否启用
	SortOrder   int        `json:"sortOrder"`   // 排序序号
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// ScanTypeFilter 检查类型查询过滤器
type ScanTypeFilter struct {
	OrgID    string
	Keyword  string // 按名称、编码模糊搜索
	IsActive *bool  // 是否启用
	Page     int
	PageSize int
}

// ScanTypeListResult 检查类型列表结果
type ScanTypeListResult struct {
	Items    []*ScanType `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
