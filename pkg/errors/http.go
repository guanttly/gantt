// pkg/errors/http.go
package errors

import (
	"fmt"
	"net/http"
)

//==============================================================================
// HTTPError - HTTP错误响应结构体
//==============================================================================

// HTTPError 表示 HTTP 请求/响应中的错误，包含状态码和消息
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewHTTPError 创建一个新的 HTTP 错误
func NewHTTPError(statusCode int, message string) error {
	return &HTTPError{
		Code:    statusCode,
		Message: message,
	}
}

// NewForbiddenError 创建一个 403 Forbidden 错误
func NewForbiddenError(message string) error {
	return &HTTPError{
		Code:    http.StatusForbidden,
		Message: message,
	}
}

// NewBadRequestError 创建一个 400 Bad Request 错误
func NewBadRequestError(message string) error {
	return &HTTPError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

// NewUnauthorizedError 创建一个 401 Unauthorized 错误
func NewUnauthorizedError(message string) error {
	return &HTTPError{
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

// NewInternalServerError 创建一个 500 Internal Server Error 错误
func NewInternalServerError(message string) error {
	return &HTTPError{
		Code:    http.StatusInternalServerError,
		Message: message,
	}
}

// Error 实现 error 接口
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", e.Code, e.Message)
}

// StatusCode 返回 HTTP 状态码
func (e *HTTPError) StatusCode() int {
	return e.Code
}

// IsHTTPError 检查 err 是否为 HTTPError 类型
func IsHTTPError(err error) bool {
	_, ok := err.(*HTTPError)
	return ok
}

// GetHTTPStatusCode 从错误中提取 HTTP 状态码，如果无法提取则返回 500
func GetHTTPStatusCode(err error) int {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.Code
	}

	// 根据错误类型映射到相应的 HTTP 状态码
	switch {
	case IsNotFoundError(err):
		return http.StatusNotFound
	case IsValidationError(err):
		return http.StatusBadRequest
	case IsAuthenticationError(err):
		return http.StatusUnauthorized
	case IsAuthorizationError(err):
		return http.StatusForbidden
	case IsConflictError(err):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

//==============================================================================
// 错误码工具函数
//==============================================================================

// CodedError 定义了具有错误码的错误接口
type CodedError interface {
	error
	Code() ErrorCode
}

// GetErrorCode 从错误中提取错误码，如果无法提取则返回 UNKNOWN
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return SUCCESS
	}

	// 尝试类型断言为 CodedError
	if codedErr, ok := err.(CodedError); ok {
		return codedErr.Code()
	}

	// 根据错误类型返回对应的错误码
	switch {
	case IsNotFoundError(err):
		return NOT_FOUND
	case IsConflictError(err):
		return CONFLICT
	case IsAuthenticationError(err):
		return AUTHENTICATION_FAILED
	case IsAuthorizationError(err):
		return FORBIDDEN
	case IsValidationError(err):
		return VALIDATION_ERROR
	case IsLogicError(err):
		return LOGIC_ERROR
	case IsProcessingError(err):
		return PROCESSING_ERROR
	case IsInfrastructureError(err):
		return INFRASTRUCTURE_ERROR
	case IsConfigurationError(err):
		return CONFIGURATION_ERROR
	case IsInitializationError(err):
		return INITIALIZATION_ERROR
	case IsJobError(err):
		return JOB_ERROR
	case IsHTTPError(err):
		return HTTP_ERROR
	default:
		return UNKNOWN
	}
}
