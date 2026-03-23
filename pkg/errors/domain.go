// pkg/errors/domain.go
package errors

import (
	"errors"
	"fmt"
)

//==============================================================================
// JobError - 作业相关错误
//==============================================================================

// JobError 表示未找到指定的作业或作业相关的错误。
type JobError struct {
	code    ErrorCode
	msg     string
	jobID   string
	wrapped error
}

// NewNotFoundJob 创建一个新的作业未找到错误
func NewNotFoundJob(msg string, wrapped error, jobID string) error {
	return &JobError{
		code:    JOB_NOT_FOUND,
		msg:     msg,
		jobID:   jobID,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口
func (e *JobError) Error() string {
	if e.jobID != "" {
		return fmt.Sprintf("[%d] job error [%s] - %s: %v", e.code, e.jobID, e.msg, e.wrapped)
	}
	return fmt.Sprintf("[%d] job error - %s: %v", e.code, e.msg, e.wrapped)
}

// Unwrap 实现错误解包
func (e *JobError) Unwrap() error {
	return e.wrapped
}

// Code 返回错误码
func (e *JobError) Code() ErrorCode {
	return e.code
}

// JobID 返回作业ID
func (e *JobError) JobID() string {
	return e.jobID
}

// IsJobError 检查 err 是否为 JobError 类型或包装了 JobError。
func IsJobError(err error) bool {
	var target *JobError
	return errors.As(err, &target)
}
