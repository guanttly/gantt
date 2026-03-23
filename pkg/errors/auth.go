// pkg/errors/auth.go
package errors

import (
	"errors"
	"fmt"
)

//==============================================================================
// AuthenticationError - 认证错误
//==============================================================================

// AuthenticationError 表示认证失败（例如，无效凭证、Token 过期等）。
type AuthenticationError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewAuthenticationError 是 AuthenticationError 的构造函数。
func NewAuthenticationError(msg string) error {
	// 通常认证错误不包装底层错误以避免信息泄露
	return &AuthenticationError{
		code:    AUTHENTICATION_FAILED,
		msg:     msg,
		wrapped: nil,
	}
}

// Error 实现 error 接口。
func (e *AuthenticationError) Error() string {
	prefix := "authentication error"
	// 不显示 wrapped error
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *AuthenticationError) Unwrap() error {
	return e.wrapped // 即使通常为 nil，也实现接口
}

// Code 返回错误码
func (e *AuthenticationError) Code() ErrorCode {
	return e.code
}

// IsAuthenticationError 检查 err 是否为 AuthenticationError 类型或包装了 AuthenticationError。
func IsAuthenticationError(err error) bool {
	var target *AuthenticationError
	return errors.As(err, &target)
}

//==============================================================================
// AuthorizationError - 授权错误
//==============================================================================

// AuthorizationError 表示操作因权限不足而被拒绝。
type AuthorizationError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewAuthorizationError 是 AuthorizationError 的构造函数。
func NewAuthorizationError(msg string, wrapped error) error {
	return &AuthorizationError{
		code:    FORBIDDEN,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *AuthorizationError) Error() string {
	prefix := "authorization error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *AuthorizationError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *AuthorizationError) Code() ErrorCode {
	return e.code
}

// IsAuthorizationError 检查 err 是否为 AuthorizationError 类型或包装了 AuthorizationError。
func IsAuthorizationError(err error) bool {
	var target *AuthorizationError
	return errors.As(err, &target)
}
