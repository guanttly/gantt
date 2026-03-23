// pkg/errors/resource.go
package errors

import (
	"errors"
	"fmt"
)

//==============================================================================
// NotFoundError - 资源未找到错误
//==============================================================================

// NotFoundError 结构体用于表示未找到错误的具体类型。
// 它实现了 Go 的内建 error 接口。
type NotFoundError struct {
	code ErrorCode
	// msg 是面向用户的、关于未找到内容的描述性信息。
	msg string
	// wrapped 是可选的、被包装的原始错误，有助于追踪错误来源。
	wrapped error
}

// NewNotFoundError 是 NotFoundError 的构造函数。
// 它接收一个用户友好的消息和一个可选的原始错误。
// 返回的是 error 接口类型，但底层是 *NotFoundError 类型。
func NewNotFoundError(msg string, wrapped error) error {
	return &NotFoundError{
		code:    NOT_FOUND,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 方法实现了 error 接口，返回错误的字符串表示形式。
// 如果有被包装的错误，会一并输出。
func (e *NotFoundError) Error() string {
	if e.wrapped != nil {
		// 格式化输出：[错误码] 用户消息: 原始错误信息
		return fmt.Sprintf("[%d] %s: %v", e.code, e.msg, e.wrapped)
	}
	// 只输出用户消息
	return fmt.Sprintf("[%d] %s", e.code, e.msg)
}

// Unwrap 方法使得此错误类型兼容 Go 1.13+ 的错误处理机制 (errors.Is, errors.As)。
// 它返回被包装的原始错误。
func (e *NotFoundError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *NotFoundError) Code() ErrorCode {
	return e.code
}

// IsNotFoundError 检查 err 是否为 NotFoundError 类型或包装了 NotFoundError。
func IsNotFoundError(err error) bool {
	var target *NotFoundError
	return errors.As(err, &target)
}

//==============================================================================
// ConflictError - 冲突错误
//==============================================================================

// ConflictError 表示操作因资源状态冲突而无法完成（例如，尝试创建已存在的资源）。
type ConflictError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewConflictError 是 ConflictError 的构造函数。
func NewConflictError(msg string, wrapped error) error {
	return &ConflictError{
		code:    CONFLICT,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *ConflictError) Error() string {
	prefix := "conflict error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *ConflictError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *ConflictError) Code() ErrorCode {
	return e.code
}

// IsConflictError 检查 err 是否为 ConflictError 类型或包装了 ConflictError。
func IsConflictError(err error) bool {
	var target *ConflictError
	return errors.As(err, &target)
}
