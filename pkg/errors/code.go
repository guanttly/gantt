package errors

// ErrorCode 错误码类型
type ErrorCode int

// 错误码定义
// 使用分段定义，便于按类别管理
const (
	// 通用错误码 (0-99)
	SUCCESS  ErrorCode = 0
	UNKNOWN  ErrorCode = 1
	INTERNAL ErrorCode = 2

	// 资源相关错误码 (1000-1099)
	NOT_FOUND ErrorCode = 1000
	CONFLICT  ErrorCode = 1001

	// 认证授权错误码 (2000-2099)
	AUTHENTICATION_FAILED ErrorCode = 2000
	UNAUTHORIZED          ErrorCode = 2001
	FORBIDDEN             ErrorCode = 2002

	// 业务逻辑错误码 (3000-3099)
	VALIDATION_ERROR ErrorCode = 3000
	LOGIC_ERROR      ErrorCode = 3001
	PROCESSING_ERROR ErrorCode = 3002
	INVALID_ARGUMENT ErrorCode = 3003

	// 基础设施错误码 (4000-4099)
	INFRASTRUCTURE_ERROR ErrorCode = 4000
	DATABASE_ERROR       ErrorCode = 4001
	NETWORK_ERROR        ErrorCode = 4002
	CONFIGURATION_ERROR  ErrorCode = 4003
	INITIALIZATION_ERROR ErrorCode = 4004

	// 领域特定错误码 (5000-5999)
	JOB_ERROR     ErrorCode = 5000
	JOB_NOT_FOUND ErrorCode = 5001

	// HTTP 错误码 (6000-6099)
	HTTP_ERROR          ErrorCode = 6000
	HTTP_BAD_REQUEST    ErrorCode = 6001
	HTTP_NOT_FOUND      ErrorCode = 6002
	HTTP_INTERNAL_ERROR ErrorCode = 6003

	// RPC/MCP 错误码 (基于 JSON-RPC 2.0 规范) (负数区间)
	// 标准 JSON-RPC 2.0 错误码
	RPC_PARSE_ERROR      ErrorCode = -32700 // 解析错误 - 无效的JSON
	RPC_INVALID_REQUEST  ErrorCode = -32600 // 无效的请求 - JSON不是有效的请求对象
	RPC_METHOD_NOT_FOUND ErrorCode = -32601 // 方法未找到
	RPC_INVALID_PARAMS   ErrorCode = -32602 // 无效的参数
	RPC_INTERNAL_ERROR   ErrorCode = -32603 // 内部错误

	// MCP 特定错误码 (扩展)
	MCP_INVALID_RANGE    ErrorCode = -32001 // 无效范围
	MCP_INVALID_TOOL     ErrorCode = -32002 // 无效工具
	MCP_INVALID_RESOURCE ErrorCode = -32003 // 无效资源
	MCP_TOOL_NOT_FOUND   ErrorCode = -32004 // 工具未找到
	MCP_TOOL_EXEC_ERROR  ErrorCode = -32005 // 工具执行失败
)

// String 返回错误码的字符串描述
func (e ErrorCode) String() string {
	switch e {
	// 通用错误
	case SUCCESS:
		return "Success"
	case UNKNOWN:
		return "Unknown Error"
	case INTERNAL:
		return "Internal Error"

	// 资源相关错误
	case NOT_FOUND:
		return "Not Found"
	case CONFLICT:
		return "Conflict"

	// 认证授权错误
	case AUTHENTICATION_FAILED:
		return "Authentication Failed"
	case UNAUTHORIZED:
		return "Unauthorized"
	case FORBIDDEN:
		return "Forbidden"

	// 业务逻辑错误
	case VALIDATION_ERROR:
		return "Validation Error"
	case LOGIC_ERROR:
		return "Logic Error"
	case PROCESSING_ERROR:
		return "Processing Error"
	case INVALID_ARGUMENT:
		return "Invalid Argument"

	// 基础设施错误
	case INFRASTRUCTURE_ERROR:
		return "Infrastructure Error"
	case DATABASE_ERROR:
		return "Database Error"
	case NETWORK_ERROR:
		return "Network Error"
	case CONFIGURATION_ERROR:
		return "Configuration Error"
	case INITIALIZATION_ERROR:
		return "Initialization Error"

	// 领域特定错误
	case JOB_ERROR:
		return "Job Error"
	case JOB_NOT_FOUND:
		return "Job Not Found"

	// HTTP 错误
	case HTTP_ERROR:
		return "HTTP Error"
	case HTTP_BAD_REQUEST:
		return "Bad Request"
	case HTTP_NOT_FOUND:
		return "HTTP Not Found"
	case HTTP_INTERNAL_ERROR:
		return "HTTP Internal Error"

	// RPC/MCP 错误
	case RPC_PARSE_ERROR:
		return "Parse Error"
	case RPC_INVALID_REQUEST:
		return "Invalid Request"
	case RPC_METHOD_NOT_FOUND:
		return "Method Not Found"
	case RPC_INVALID_PARAMS:
		return "Invalid Params"
	case RPC_INTERNAL_ERROR:
		return "RPC Internal Error"
	case MCP_INVALID_RANGE:
		return "Invalid Range"
	case MCP_INVALID_TOOL:
		return "Invalid Tool"
	case MCP_INVALID_RESOURCE:
		return "Invalid Resource"
	case MCP_TOOL_NOT_FOUND:
		return "Tool Not Found"
	case MCP_TOOL_EXEC_ERROR:
		return "Tool Execution Error"

	default:
		return "Unknown Error"
	}
}

// Code 返回错误码的整数值
func (e ErrorCode) Code() int {
	return int(e)
}

// Int 返回错误码的整数值（别名方法）
func (e ErrorCode) Int() int {
	return int(e)
}
