// Package response 提供统一的 HTTP 响应格式。
package response

import (
	"encoding/json"
	"net/http"
)

// ErrorBody 统一错误响应结构。
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情。
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PageResponse 统一分页响应结构。
type PageResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// SuccessResponse 统一成功响应结构。
type SuccessResponse struct {
	Data interface{} `json:"data"`
}

// JSON 发送 JSON 响应。
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// OK 发送 200 成功响应。
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, SuccessResponse{Data: data})
}

// Created 发送 201 创建成功响应。
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, SuccessResponse{Data: data})
}

// NoContent 发送 204 无内容响应。
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Page 发送分页响应。
func Page(w http.ResponseWriter, data interface{}, total int64, page, size int) {
	JSON(w, http.StatusOK, PageResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// Error 发送错误响应。
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorBody{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// BadRequest 发送 400 错误。
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized 发送 401 错误。
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden 发送 403 错误。
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound 发送 404 错误。
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict 发送 409 错误。
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, "CONFLICT", message)
}

// InternalError 发送 500 错误。
func InternalError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
