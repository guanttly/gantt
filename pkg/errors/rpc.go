// pkg/errors/rpc.go
package errors

import (
	"fmt"
)

//==============================================================================
// RPCError - RPC/JSON-RPC 2.0 错误
//==============================================================================

// RPCError 表示符合 JSON-RPC 2.0 规范的错误
// 用于 RPC 调用和 MCP (Model Context Protocol) 协议
type RPCError struct {
	code    ErrorCode
	message string
	data    interface{} // 可选的额外错误数据
	wrapped error
}

// NewRPCError 创建一个新的 RPC 错误
func NewRPCError(code ErrorCode, message string) error {
	return &RPCError{
		code:    code,
		message: message,
		data:    nil,
		wrapped: nil,
	}
}

// NewRPCErrorWithData 创建一个带额外数据的 RPC 错误
func NewRPCErrorWithData(code ErrorCode, message string, data interface{}) error {
	return &RPCError{
		code:    code,
		message: message,
		data:    data,
		wrapped: nil,
	}
}

// WrapRPCError 包装一个现有错误为 RPC 错误
func WrapRPCError(code ErrorCode, message string, wrapped error) error {
	return &RPCError{
		code:    code,
		message: message,
		data:    nil,
		wrapped: wrapped,
	}
}

// Error 实现 error 接口
func (e *RPCError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("[%d] %s: %v", e.code, e.message, e.wrapped)
	}
	return fmt.Sprintf("[%d] %s", e.code, e.message)
}

// Code 返回错误码
func (e *RPCError) Code() ErrorCode {
	return e.code
}

// Message 返回错误消息
func (e *RPCError) Message() string {
	return e.message
}

// Data 返回额外的错误数据
func (e *RPCError) Data() interface{} {
	return e.data
}

// Unwrap 实现错误解包
func (e *RPCError) Unwrap() error {
	return e.wrapped
}

// ToJSONRPCError 转换为 JSON-RPC 2.0 标准错误格式
func (e *RPCError) ToJSONRPCError() map[string]interface{} {
	result := map[string]interface{}{
		"code":    int(e.code),
		"message": e.message,
	}
	if e.data != nil {
		result["data"] = e.data
	}
	return result
}

// IsRPCError 检查 err 是否为 RPCError 类型
func IsRPCError(err error) bool {
	_, ok := err.(*RPCError)
	return ok
}

//==============================================================================
// 常用 RPC 错误构造函数
//==============================================================================

// NewParseError 创建解析错误 (-32700)
func NewParseError(message string) error {
	if message == "" {
		message = "Parse error"
	}
	return NewRPCError(RPC_PARSE_ERROR, message)
}

// NewInvalidRequestError 创建无效请求错误 (-32600)
func NewInvalidRequestError(message string) error {
	if message == "" {
		message = "Invalid request"
	}
	return NewRPCError(RPC_INVALID_REQUEST, message)
}

// NewMethodNotFoundError 创建方法未找到错误 (-32601)
func NewMethodNotFoundError(method string) error {
	message := "Method not found"
	if method != "" {
		message = fmt.Sprintf("Method not found: %s", method)
	}
	return NewRPCError(RPC_METHOD_NOT_FOUND, message)
}

// NewInvalidParamsError 创建无效参数错误 (-32602)
func NewInvalidParamsError(message string) error {
	if message == "" {
		message = "Invalid params"
	}
	return NewRPCError(RPC_INVALID_PARAMS, message)
}

// NewRPCInternalError 创建 RPC 内部错误 (-32603)
func NewRPCInternalError(message string, wrapped error) error {
	if message == "" {
		message = "Internal error"
	}
	return WrapRPCError(RPC_INTERNAL_ERROR, message, wrapped)
}

//==============================================================================
// MCP 特定错误构造函数
//==============================================================================

// NewMCPInvalidRangeError 创建 MCP 无效范围错误 (-32001)
func NewMCPInvalidRangeError(message string) error {
	if message == "" {
		message = "Invalid range"
	}
	return NewRPCError(MCP_INVALID_RANGE, message)
}

// NewMCPInvalidToolError 创建 MCP 无效工具错误 (-32002)
func NewMCPInvalidToolError(toolName string) error {
	message := "Invalid tool"
	if toolName != "" {
		message = fmt.Sprintf("Invalid tool: %s", toolName)
	}
	return NewRPCError(MCP_INVALID_TOOL, message)
}

// NewMCPInvalidResourceError 创建 MCP 无效资源错误 (-32003)
func NewMCPInvalidResourceError(resource string) error {
	message := "Invalid resource"
	if resource != "" {
		message = fmt.Sprintf("Invalid resource: %s", resource)
	}
	return NewRPCError(MCP_INVALID_RESOURCE, message)
}

// NewMCPToolNotFoundError 创建 MCP 工具未找到错误 (-32004)
func NewMCPToolNotFoundError(toolName string) error {
	message := "Tool not found"
	if toolName != "" {
		message = fmt.Sprintf("Tool not found: %s", toolName)
	}
	return NewRPCError(MCP_TOOL_NOT_FOUND, message)
}

// NewMCPToolExecError 创建 MCP 工具执行错误 (-32005)
func NewMCPToolExecError(toolName string, wrapped error) error {
	message := "Tool execution failed"
	if toolName != "" {
		message = fmt.Sprintf("Tool execution failed: %s", toolName)
	}
	return WrapRPCError(MCP_TOOL_EXEC_ERROR, message, wrapped)
}

//==============================================================================
// 工具函数
//==============================================================================

// GetRPCErrorCode 从错误中提取 RPC 错误码
func GetRPCErrorCode(err error) ErrorCode {
	if err == nil {
		return SUCCESS
	}

	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr.Code()
	}

	// 尝试从 CodedError 接口获取
	if codedErr, ok := err.(CodedError); ok {
		code := codedErr.Code()
		// 检查是否为 RPC 错误码范围（负数）
		if int(code) < 0 {
			return code
		}
	}

	return UNKNOWN
}

// IsRPCErrorCode 检查错误码是否为 RPC 错误码
func IsRPCErrorCode(code ErrorCode) bool {
	return int(code) < 0 && int(code) >= -32768
}
