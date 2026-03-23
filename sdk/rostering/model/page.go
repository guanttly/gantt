package model

// Page 分页结果泛型
type Page[T any] struct {
	Items    []T  `json:"items"`    // 数据列表
	Total    int  `json:"total"`    // 总记录数
	Page     int  `json:"page"`     // 当前页码
	PageSize int  `json:"pageSize"` // 每页大小
	HasMore  bool `json:"hasMore"`  // 是否有更多数据
}

// NewPage 创建分页结果
func NewPage[T any](items []T, total, page, pageSize int) *Page[T] {
	hasMore := false
	if pageSize > 0 {
		hasMore = (page * pageSize) < total
	}

	return &Page[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	}
}

// EmptyPage 创建空分页结果
func EmptyPage[T any]() *Page[T] {
	return &Page[T]{
		Items:    []T{},
		Total:    0,
		Page:     1,
		PageSize: 0,
		HasMore:  false,
	}
}
