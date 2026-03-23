package model

import (
	"time"
)

// TimePeriod 时间段领域模型
type TimePeriod struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code"`        // 时间段编码
	Name        string     `json:"name"`        // 时间段名称
	StartTime   string     `json:"startTime"`   // 开始时间：HH:MM
	EndTime     string     `json:"endTime"`     // 结束时间：HH:MM
	IsCrossDay  bool       `json:"isCrossDay"`  // 是否跨日
	Description string     `json:"description"` // 时间段说明
	IsActive    bool       `json:"isActive"`    // 是否启用
	SortOrder   int        `json:"sortOrder"`   // 排序序号
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// TimePeriodFilter 时间段查询过滤器
type TimePeriodFilter struct {
	OrgID    string
	Keyword  string // 按名称、编码模糊搜索
	IsActive *bool  // 是否启用
	Page     int
	PageSize int
}

// TimePeriodListResult 时间段列表结果
type TimePeriodListResult struct {
	Items    []*TimePeriod `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}
