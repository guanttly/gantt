// pkg/errors/example_test.go
package errors_test

import (
	"fmt"
	"jusha/mcp/pkg/errors"
)

// 示例：创建和使用 NotFoundError
func ExampleNotFoundError() {
	err := errors.NewNotFoundError("user not found", nil)

	// 检查错误类型
	if errors.IsNotFoundError(err) {
		fmt.Println("This is a NotFoundError")
	}

	// 获取错误码
	code := errors.GetErrorCode(err)
	fmt.Printf("Error code: %d\n", code.Code())
	fmt.Printf("Error type: %s\n", code.String())

	// Output:
	// This is a NotFoundError
	// Error code: 1000
	// Error type: Not Found
}

// 示例：从任意错误获取错误码
func ExampleGetErrorCode() {
	// 创建不同类型的错误
	err1 := errors.NewValidationError("invalid email", nil)
	err2 := errors.NewAuthenticationError("invalid credentials")
	err3 := errors.NewInfrastructureError("database error", nil)

	// 获取错误码
	code1 := errors.GetErrorCode(err1)
	code2 := errors.GetErrorCode(err2)
	code3 := errors.GetErrorCode(err3)

	fmt.Printf("Validation Error code: %d\n", code1.Code())
	fmt.Printf("Authentication Error code: %d\n", code2.Code())
	fmt.Printf("Infrastructure Error code: %d\n", code3.Code())

	// Output:
	// Validation Error code: 3000
	// Authentication Error code: 2000
	// Infrastructure Error code: 4000
}

// 示例：使用 CodedError 接口
func ExampleCodedError() {
	err := errors.NewConflictError("resource already exists", nil)

	// 类型断言为 CodedError
	if codedErr, ok := err.(errors.CodedError); ok {
		code := codedErr.Code()
		fmt.Printf("Error code: %d - %s\n", code.Code(), code.String())
	}

	// Output:
	// Error code: 1001 - Conflict
}

// 示例：HTTP 错误处理
func ExampleGetHTTPStatusCode() {
	// 不同类型的错误映射到不同的 HTTP 状态码
	err1 := errors.NewNotFoundError("resource not found", nil)
	err2 := errors.NewValidationError("invalid input", nil)
	err3 := errors.NewAuthenticationError("authentication failed")

	status1 := errors.GetHTTPStatusCode(err1)
	status2 := errors.GetHTTPStatusCode(err2)
	status3 := errors.GetHTTPStatusCode(err3)

	fmt.Printf("NotFoundError -> HTTP %d\n", status1)
	fmt.Printf("ValidationError -> HTTP %d\n", status2)
	fmt.Printf("AuthenticationError -> HTTP %d\n", status3)

	// Output:
	// NotFoundError -> HTTP 404
	// ValidationError -> HTTP 400
	// AuthenticationError -> HTTP 401
}

// 示例：错误码比较
func ExampleErrorCode_comparison() {
	err := errors.NewProcessingError("processing failed", nil)
	code := errors.GetErrorCode(err)

	// 使用错误码进行判断
	switch code {
	case errors.NOT_FOUND:
		fmt.Println("Resource not found")
	case errors.VALIDATION_ERROR:
		fmt.Println("Validation failed")
	case errors.PROCESSING_ERROR:
		fmt.Println("Processing failed")
	default:
		fmt.Println("Other error")
	}

	// Output:
	// Processing failed
}
