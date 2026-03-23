package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"jusha/mcp/pkg/model"

	pkg_error "jusha/mcp/pkg/errors"
)

// PageRequest 分页请求
type PageRequest struct {
	Page int `json:"page" form:"page"` // 页码，从1开始
	Size int `json:"size" form:"size"` // 每页数量
}

// IDRequest ID请求
type IDRequest struct {
	ID string `json:"id" validate:"required"`
}

// IDsRequest 批量ID请求
type IDsRequest struct {
	IDs []string `json:"ids" validate:"required,min=1"`
}

// 响应帮助函数

// RespondJSON 返回JSON响应
func RespondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// RespondSuccess 返回成功响应
func RespondSuccess(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusOK, model.ApiResponse{
		Code:    int(pkg_error.SUCCESS),
		Message: "success",
		Data:    data,
	})
}

// RespondCreated 返回创建成功响应
func RespondCreated(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusCreated, model.ApiResponse{
		Code:    int(pkg_error.SUCCESS),
		Message: "created",
		Data:    data,
	})
}

// RespondError 返回错误响应（自定义错误码）
func RespondError(w http.ResponseWriter, httpStatus int, code pkg_error.ErrorCode, message string) {
	RespondJSON(w, httpStatus, model.ApiResponse{
		Code:    int(code),
		Message: message,
		Data:    nil,
	})
}

// RespondBadRequest 返回400错误
func RespondBadRequest(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusBadRequest, pkg_error.VALIDATION_ERROR, message)
}

// RespondNotFound 返回404错误
func RespondNotFound(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusNotFound, pkg_error.NOT_FOUND, message)
}

// RespondInternalError 返回500错误
func RespondInternalError(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusInternalServerError, pkg_error.INTERNAL, message)
}

// RespondUnauthorized 返回401错误
func RespondUnauthorized(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusUnauthorized, pkg_error.UNAUTHORIZED, message)
}

// RespondForbidden 返回403错误
func RespondForbidden(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusForbidden, pkg_error.FORBIDDEN, message)
}

// RespondConflict 返回409错误
func RespondConflict(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusConflict, pkg_error.CONFLICT, message)
}

// RespondPage 返回分页响应
func RespondPage(w http.ResponseWriter, total int64, page, size int, items interface{}) {
	RespondSuccess(w, map[string]interface{}{
		"total": total,
		"page":  page,
		"size":  size,
		"items": items,
	})
}

// RespondOK 返回200成功响应（简化版RespondSuccess的别名）
func RespondOK(w http.ResponseWriter, data interface{}) {
	RespondSuccess(w, data)
}

// RespondNoContent 返回204无内容响应
func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// parsePageParams 解析分页参数
func parsePageParams(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := r.URL.Query().Get("pageSize"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	return page, pageSize
}

// parseDate 解析日期字符串（YYYY-MM-DD格式）
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
