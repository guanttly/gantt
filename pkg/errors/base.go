// pkg/errors/base.go
package errors

import (
	"errors"
	"fmt"
)

// BaseError 是一个通用的错误基础结构，提供消息和错误包装功能
type BaseError struct {
	code    ErrorCode
	prefix  string
	msg     string
	wrapped error
}

// NewBaseError 创建一个新的基础错误
func NewBaseError(code ErrorCode, prefix, msg string, wrapped error) *BaseError {
	return &BaseError{
		code:    code,
		prefix:  prefix,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口
func (e *BaseError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, e.prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, e.prefix, e.msg)
}

// Unwrap 实现错误解包，兼容 Go 1.13+ 的错误处理机制
func (e *BaseError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *BaseError) Code() ErrorCode {
	return e.code
}

// Message 返回错误消息
func (e *BaseError) Message() string {
	return e.msg
}

// Prefix 返回错误前缀
func (e *BaseError) Prefix() string {
	return e.prefix
}

// IsType 检查错误是否为指定类型
func IsType(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Wrap 包装一个错误并添加上下文消息
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf 包装一个错误并添加格式化的上下文消息
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
