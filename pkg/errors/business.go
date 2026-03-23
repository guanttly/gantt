// pkg/errors/business.go
package errors

import (
	"errors"
	"fmt"
)

//==============================================================================
// ValidationError - 验证错误
//==============================================================================

// ValidationError 表示输入数据未能通过验证规则。
type ValidationError struct {
	code ErrorCode
	msg  string
	// 可以扩展此结构体以包含更详细的字段级错误信息
	// FieldErrors map[string]string
	wrapped error
}

// NewValidationError 是 ValidationError 的构造函数。
func NewValidationError(msg string, wrapped error) error {
	return &ValidationError{
		code:    VALIDATION_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *ValidationError) Error() string {
	prefix := "validation error"
	// 可以扩展此方法以包含字段错误信息
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *ValidationError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *ValidationError) Code() ErrorCode {
	return e.code
}

// IsValidationError 检查 err 是否为 ValidationError 类型或包装了 ValidationError。
func IsValidationError(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}

//==============================================================================
// LogicError - 业务逻辑错误
//==============================================================================

// LogicError 表示在执行业务逻辑过程中发生的、不属于其他特定错误类型的错误。
// 例如，状态转换无效、不满足业务规则前提条件等。
type LogicError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewLogicError 是 LogicError 的构造函数。
func NewLogicError(msg string, wrapped error) error {
	return &LogicError{
		code:    LOGIC_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口。
func (e *LogicError) Error() string {
	prefix := "logic error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *LogicError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *LogicError) Code() ErrorCode {
	return e.code
}

// IsLogicError 检查 err 是否为 LogicError 类型或包装了 LogicError。
func IsLogicError(err error) bool {
	var target *LogicError
	return errors.As(err, &target)
}

//==============================================================================
// ProcessingError - 处理错误
//==============================================================================

// ProcessingError 表示在执行内部业务逻辑或数据处理时遇到的非基础设施问题。
// 例如：数据转换失败、无法执行某个计算步骤等。
type ProcessingError struct {
	code    ErrorCode
	msg     string
	wrapped error
}

// NewProcessingError 是 ProcessingError 的构造函数。
func NewProcessingError(msg string, wrapped error) error {
	return &ProcessingError{
		code:    PROCESSING_ERROR,
		msg:     msg,
		wrapped: wrapped,
	}
}

// NewInvalidArgumentError 创建一个无效参数错误（ProcessingError 的特殊情况）
func NewInvalidArgumentError(msg string) error {
	return &ProcessingError{
		code:    INVALID_ARGUMENT,
		msg:     msg,
		wrapped: nil,
	}
}

// Error 实现 error 接口。
func (e *ProcessingError) Error() string {
	prefix := "processing error"
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s - %s: %v", e.code, prefix, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s - %s", e.code, prefix, e.msg)
}

// Unwrap 实现错误解包。
func (e *ProcessingError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *ProcessingError) Code() ErrorCode {
	return e.code
}

// IsProcessingError 检查 err 是否为 ProcessingError 类型或包装了 ProcessingError。
func IsProcessingError(err error) bool {
	var target *ProcessingError
	return errors.As(err, &target)
}
