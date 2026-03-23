package model

type ApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type Page[T any] struct {
	Total int64 `json:"total"`
	Size  int   `json:"size"`
	Page  int   `json:"page"`
	Items []T   `json:"items"`
}

type AIThinkResponse struct {
	Thinking string `json:"think"`
	Response string `json:"response"`
}
