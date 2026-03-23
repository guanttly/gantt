package model

// APIResponse 通用API响应结构
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// PageData 分页数据（泛型）
type PageData[T any] struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Items []T   `json:"items"`
}
