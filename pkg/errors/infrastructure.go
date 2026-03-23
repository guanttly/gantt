// pkg/errors/infrastructure.go
package errors

import (
	"errors"
	"fmt"
)

//==============================================================================
// InfrastructureError - 基础设施错误
//==============================================================================

// InfrastructureError 表示与外部系统（数据库、网络、文件系统等）交互时发生的错误。
// 这类错误通常不是由用户输入或业务逻辑直接引起的。
type InfrastructureError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewInfrastructureError 是 InfrastructureError 的构造函数。
func NewInfrastructureError(msg string, wrapped error) error {
	return &InfrastructureError{
		code:    INFRASTRUCTURE_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// WrapInfrastructureError 是一个辅助函数，用于将现有错误包装为 InfrastructureError。
func WrapInfrastructureError(err error, msg string) error {
	return &InfrastructureError{
		code:    INFRASTRUCTURE_ERROR,
		msg:     msg,
		wrapped: err,
	}
}

// Error 实现 error 接口。
func (e *InfrastructureError) Error() string {
	// 添加前缀以明确错误类型
	prefix := "infrastructure error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *InfrastructureError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *InfrastructureError) Code() ErrorCode {
	return e.code
}

// IsInfrastructureError 检查 err 是否为 InfrastructureError 类型或包装了 InfrastructureError。
func IsInfrastructureError(err error) bool {
	var target *InfrastructureError
	return errors.As(err, &target)
}

//==============================================================================
// ConfigurationError - 配置错误
//==============================================================================

// ConfigurationError 表示配置相关的错误，例如缺少必要的配置项、配置格式错误等。
type ConfigurationError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewConfigurationError 是 ConfigurationError 的构造函数。
func NewConfigurationError(msg string, wrapped error) error {
	return &ConfigurationError{
		code:    CONFIGURATION_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *ConfigurationError) Error() string {
	prefix := "configuration error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *ConfigurationError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *ConfigurationError) Code() ErrorCode {
	return e.code
}

// IsConfigurationError 检查 err 是否为 ConfigurationError 类型或包装了 ConfigurationError。
func IsConfigurationError(err error) bool {
	var target *ConfigurationError
	return errors.As(err, &target)
}

//==============================================================================
// InitializationError - 初始化错误
//==============================================================================

// InitializationError 表示在系统或组件初始化过程中发生的错误。
type InitializationError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewInitializationError 是 InitializationError 的构造函数。
func NewInitializationError(msg string, wrapped error) error {
	return &InitializationError{
		code:    INITIALIZATION_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *InitializationError) Error() string {
	prefix := "initialization error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *InitializationError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *InitializationError) Code() ErrorCode {
	return e.code
}

// IsInitializationError 检查 err 是否为 InitializationError 类型或包装了 InitializationError。
func IsInitializationError(err error) bool {
	var target *InitializationError
	return errors.As(err, &target)
}
