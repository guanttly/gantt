# Errors Package

本包提供了一套统一的、分类清晰的错误处理机制，每个错误都包含唯一的错误码用于识别和判断。

## 文件结构

```
pkg/errors/
├── errors.go          # 包级别文档和说明
├── base.go            # 通用错误基础结构和工具函数
├── code.go            # 错误码定义
├── resource.go        # 资源相关错误
├── auth.go            # 认证和授权错误
├── business.go        # 业务逻辑错误
├── infrastructure.go  # 基础设施错误
├── domain.go          # 领域特定错误
├── http.go            # HTTP 错误和工具函数
└── rpc.go             # RPC/MCP 错误 (JSON-RPC 2.0)
```

## 错误码分类

错误码按照类别进行分段管理：

- **0-99**: 通用错误码
- **1000-1099**: 资源相关错误
- **2000-2099**: 认证授权错误
- **3000-3099**: 业务逻辑错误
- **4000-4099**: 基础设施错误
- **5000-5999**: 领域特定错误
- **6000-6099**: HTTP 错误
- **负数区间 (-32768 ~ -1)**: RPC/MCP 错误 (符合 JSON-RPC 2.0 规范)

## 错误分类

### 1. 资源相关错误 (`resource.go`)

**NotFoundError** (错误码: 1000) - 资源未找到错误
```go
err := errors.NewNotFoundError("user not found", nil)
if errors.IsNotFoundError(err) {
    code := err.(*errors.NotFoundError).Code() // 返回 NOT_FOUND (1000)
}
```

**ConflictError** (错误码: 1001) - 资源冲突错误
```go
err := errors.NewConflictError("user already exists", nil)
if errors.IsConflictError(err) {
    code := err.(*errors.ConflictError).Code() // 返回 CONFLICT (1001)
}
```

### 2. 认证授权错误 (`auth.go`)

**AuthenticationError** (错误码: 2000) - 认证失败错误
```go
err := errors.NewAuthenticationError("invalid credentials")
code := err.(*errors.AuthenticationError).Code() // 返回 AUTHENTICATION_FAILED (2000)
```

**AuthorizationError** (错误码: 2002) - 授权失败错误
```go
err := errors.NewAuthorizationError("insufficient permissions", nil)
code := err.(*errors.AuthorizationError).Code() // 返回 FORBIDDEN (2002)
```

### 3. 业务逻辑错误 (`business.go`)

**ValidationError** (错误码: 3000) - 输入验证错误
```go
err := errors.NewValidationError("invalid email format", nil)
code := err.(*errors.ValidationError).Code() // 返回 VALIDATION_ERROR (3000)
```

**LogicError** (错误码: 3001) - 业务逻辑错误
```go
err := errors.NewLogicError("invalid state transition", nil)
code := err.(*errors.LogicError).Code() // 返回 LOGIC_ERROR (3001)
```

**ProcessingError** (错误码: 3002/3003) - 数据处理错误
```go
err := errors.NewProcessingError("failed to convert data", originalErr)
code := err.(*errors.ProcessingError).Code() // 返回 PROCESSING_ERROR (3002)

// 或者使用特殊情况 - 无效参数
err := errors.NewInvalidArgumentError("argument cannot be nil")
code := err.(*errors.ProcessingError).Code() // 返回 INVALID_ARGUMENT (3003)
```

### 4. 基础设施错误 (`infrastructure.go`)

**InfrastructureError** (错误码: 4000) - 基础设施错误
```go
err := errors.NewInfrastructureError("database connection failed", dbErr)
// 或者包装现有错误
err := errors.WrapInfrastructureError(dbErr, "failed to query database")
code := err.(*errors.InfrastructureError).Code() // 返回 INFRASTRUCTURE_ERROR (4000)
```

**ConfigurationError** (错误码: 4003) - 配置错误
```go
err := errors.NewConfigurationError("missing required config", nil)
code := err.(*errors.ConfigurationError).Code() // 返回 CONFIGURATION_ERROR (4003)
```

**InitializationError** (错误码: 4004) - 初始化错误
```go
err := errors.NewInitializationError("failed to initialize service", initErr)
code := err.(*errors.InitializationError).Code() // 返回 INITIALIZATION_ERROR (4004)
```

### 5. 领域特定错误 (`domain.go`)

**JobError** (错误码: 5001) - 作业相关错误
```go
err := errors.NewNotFoundJob("job not found", nil, "job-123")
if errors.IsJobError(err) {
    jobErr := err.(*errors.JobError)
    code := jobErr.Code() // 返回 JOB_NOT_FOUND (5001)
    jobID := jobErr.JobID() // 返回 "job-123"
}
```

### 6. HTTP 错误 (`http.go`)

**HTTPError** - HTTP 响应错误
```go
// 通用 HTTP 错误
err := errors.NewHTTPError(404, "resource not found")

// 预定义的 HTTP 错误
err := errors.NewBadRequestError("invalid request")
err := errors.NewUnauthorizedError("authentication required")
err := errors.NewForbiddenError("access denied")
err := errors.NewInternalServerError("internal server error")

// 检查 HTTP 错误
if errors.IsHTTPError(err) {
    httpErr := err.(*errors.HTTPError)
    statusCode := httpErr.StatusCode()
}

// 从任意错误获取 HTTP 状态码
statusCode := errors.GetHTTPStatusCode(err)
```

### 7. RPC/MCP 错误 (`rpc.go`)

RPC 错误符合 JSON-RPC 2.0 规范，用于 RPC 调用和 MCP (Model Context Protocol) 协议。

**标准 JSON-RPC 2.0 错误**

```go
// 解析错误 (-32700)
err := errors.NewParseError("invalid JSON")

// 无效请求 (-32600)
err := errors.NewInvalidRequestError("missing required field")

// 方法未找到 (-32601)
err := errors.NewMethodNotFoundError("getUserInfo")

// 无效参数 (-32602)
err := errors.NewInvalidParamsError("userId must be a number")

// RPC 内部错误 (-32603)
err := errors.NewRPCInternalError("database connection failed", dbErr)
```

**MCP 特定错误**

```go
// 无效范围 (-32001)
err := errors.NewMCPInvalidRangeError("line range exceeds file length")

// 无效工具 (-32002)
err := errors.NewMCPInvalidToolError("calculator")

// 无效资源 (-32003)
err := errors.NewMCPInvalidResourceError("file://path/to/file")

// 工具未找到 (-32004)
err := errors.NewMCPToolNotFoundError("search_documents")

// 工具执行错误 (-32005)
err := errors.NewMCPToolExecError("process_data", execErr)
```

**RPCError 高级用法**

```go
// 创建带额外数据的 RPC 错误
err := errors.NewRPCErrorWithData(
    errors.RPC_INVALID_PARAMS,
    "validation failed",
    map[string]string{
        "field": "email",
        "reason": "invalid format",
    },
)

// 转换为 JSON-RPC 2.0 标准错误格式
if rpcErr, ok := err.(*errors.RPCError); ok {
    jsonError := rpcErr.ToJSONRPCError()
    // 返回: {"code": -32602, "message": "validation failed", "data": {...}}
}

// 检查是否为 RPC 错误
if errors.IsRPCError(err) {
    rpcErr := err.(*errors.RPCError)
    code := rpcErr.Code()
    message := rpcErr.Message()
    data := rpcErr.Data()
}

// 获取 RPC 错误码
code := errors.GetRPCErrorCode(err) // 返回 RPC 错误码或 UNKNOWN
```

## 错误码工具函数

### GetErrorCode - 获取错误码

从任意错误中提取错误码：

```go
err := errors.NewNotFoundError("user not found", nil)

// 获取错误码
code := errors.GetErrorCode(err) // 返回 NOT_FOUND (1000)

// 获取错误码的整数值
codeInt := code.Code() // 返回 1000

// 获取错误码的字符串描述
codeStr := code.String() // 返回 "Not Found"
```

### CodedError 接口

所有包含错误码的错误都实现了 `CodedError` 接口：

```go
type CodedError interface {
    error
    Code() ErrorCode
}

// 使用示例
func HandleError(err error) {
    if codedErr, ok := err.(errors.CodedError); ok {
        code := codedErr.Code()
        fmt.Printf("Error code: %d - %s\n", code.Code(), code.String())
    }
}
```

## 通用工具 (`base.go`)

**BaseError** - 基础错误结构（可用于扩展自定义错误）
```go
baseErr := errors.NewBaseError(errors.UNKNOWN, "custom error", "something went wrong", originalErr)
code := baseErr.Code() // 返回错误码
```

**Wrap/Wrapf** - 错误包装函数
```go
// 包装错误并添加上下文
err := errors.Wrap(originalErr, "failed to process request")

// 格式化包装
err := errors.Wrapf(originalErr, "failed to process user %s", userID)
```

## 最佳实践

1. **使用错误码进行判断**：
```go
err := someOperation()
code := errors.GetErrorCode(err)
switch code {
case errors.NOT_FOUND:
    // 处理未找到错误
case errors.VALIDATION_ERROR:
    // 处理验证错误
default:
    // 处理其他错误
}
```

2. **错误消息格式**：
所有错误都包含错误码前缀，格式为：`[错误码] 错误类型 - 消息`
```go
err := errors.NewNotFoundError("user not found", nil)
fmt.Println(err.Error()) // 输出: [1000] user not found
```

3. **选择合适的错误类型**：根据错误的性质选择最合适的错误类型
4. **保留错误链**：使用 `wrapped` 参数保留原始错误，便于追踪
5. **使用类型检查**：使用 `Is...Error()` 函数进行类型检查，而不是字符串比较
6. **提供清晰的消息**：错误消息应该清晰描述发生了什么
7. **避免信息泄露**：认证错误等敏感错误不要包含过多细节

## 错误处理示例

### 基本错误处理

```go
func GetUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        if errors.IsNotFoundError(err) {
            return nil, errors.NewNotFoundError("user not found", err)
        }
        return nil, errors.WrapInfrastructureError(err, "failed to query user")
    }
    
    if !user.Active {
        return nil, errors.NewLogicError("user is inactive", nil)
    }
    
    return user, nil
}
```

### HTTP 处理器中的错误处理

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Error   string `json:"error"`
}

func HandleError(w http.ResponseWriter, err error) {
    // 获取错误码
    errorCode := errors.GetErrorCode(err)
    
    // 获取 HTTP 状态码
    httpStatusCode := errors.GetHTTPStatusCode(err)
    
    response := ErrorResponse{
        Code:    errorCode.Code(),
        Message: errorCode.String(),
        Error:   err.Error(),
    }
    
    w.WriteHeader(httpStatusCode)
    json.NewEncoder(w).Encode(response)
}
```

### 统一的错误响应

```go
func UnifiedErrorHandler(err error) map[string]interface{} {
    code := errors.GetErrorCode(err)
    
    return map[string]interface{}{
        "success": false,
        "code":    code.Code(),
        "type":    code.String(),
        "message": err.Error(),
    }
}

// 使用示例
err := errors.NewValidationError("invalid email", nil)
response := UnifiedErrorHandler(err)
// 返回:
// {
//     "success": false,
//     "code": 3000,
//     "type": "Validation Error",
//     "message": "[3000] validation error - invalid email"
// }
```

## 错误码速查表

| 错误码 | 常量名 | 描述 | 错误类型 |
|--------|--------|------|----------|
| 0 | SUCCESS | 成功 | - |
| 1 | UNKNOWN | 未知错误 | - |
| 2 | INTERNAL | 内部错误 | - |
| 1000 | NOT_FOUND | 资源未找到 | NotFoundError |
| 1001 | CONFLICT | 资源冲突 | ConflictError |
| 2000 | AUTHENTICATION_FAILED | 认证失败 | AuthenticationError |
| 2001 | UNAUTHORIZED | 未授权 | - |
| 2002 | FORBIDDEN | 禁止访问 | AuthorizationError |
| 3000 | VALIDATION_ERROR | 验证错误 | ValidationError |
| 3001 | LOGIC_ERROR | 逻辑错误 | LogicError |
| 3002 | PROCESSING_ERROR | 处理错误 | ProcessingError |
| 3003 | INVALID_ARGUMENT | 无效参数 | ProcessingError |
| 4000 | INFRASTRUCTURE_ERROR | 基础设施错误 | InfrastructureError |
| 4001 | DATABASE_ERROR | 数据库错误 | - |
| 4002 | NETWORK_ERROR | 网络错误 | - |
| 4003 | CONFIGURATION_ERROR | 配置错误 | ConfigurationError |
| 4004 | INITIALIZATION_ERROR | 初始化错误 | InitializationError |
| 5000 | JOB_ERROR | 作业错误 | JobError |
| 5001 | JOB_NOT_FOUND | 作业未找到 | JobError |
| 6000 | HTTP_ERROR | HTTP错误 | HTTPError |
| 6001 | HTTP_BAD_REQUEST | HTTP坏请求 | HTTPError |
| 6002 | HTTP_NOT_FOUND | HTTP未找到 | HTTPError |
| 6003 | HTTP_INTERNAL_ERROR | HTTP内部错误 | HTTPError |

### RPC/MCP 错误码 (JSON-RPC 2.0)

| 错误码 | 常量名 | 描述 | 错误类型 |
|--------|--------|------|----------|
| -32700 | RPC_PARSE_ERROR | 解析错误 - 无效的JSON | RPCError |
| -32600 | RPC_INVALID_REQUEST | 无效的请求 | RPCError |
| -32601 | RPC_METHOD_NOT_FOUND | 方法未找到 | RPCError |
| -32602 | RPC_INVALID_PARAMS | 无效的参数 | RPCError |
| -32603 | RPC_INTERNAL_ERROR | RPC内部错误 | RPCError |
| -32001 | MCP_INVALID_RANGE | 无效范围 | RPCError |
| -32002 | MCP_INVALID_TOOL | 无效工具 | RPCError |
| -32003 | MCP_INVALID_RESOURCE | 无效资源 | RPCError |
| -32004 | MCP_TOOL_NOT_FOUND | 工具未找到 | RPCError |
| -32005 | MCP_TOOL_EXEC_ERROR | 工具执行失败 | RPCError |

## 迁移指南

如果你的代码使用了旧的 `errors.go` 文件，现在所有的错误类型已经被拆分到不同的文件中，但导入方式保持不变：

```go
import "your-project/pkg/errors"

// 所有的函数和类型都可以正常使用
err := errors.NewNotFoundError("not found", nil)

// 新增：现在可以获取错误码
code := errors.GetErrorCode(err)
fmt.Printf("Error code: %d\n", code.Code())
```

**重要变更**：
1. 所有错误现在都包含错误码字段
2. 错误消息格式变更为：`[错误码] 错误类型 - 消息`
3. 新增 `GetErrorCode()` 工具函数用于获取错误码
4. 新增 `CodedError` 接口

无需修改现有的错误创建代码，但错误消息输出格式会包含错误码前缀。

## 错误分类

### 1. 资源相关错误 (`resource.go`)

**NotFoundError** - 资源未找到错误
```go
err := errors.NewNotFoundError("user not found", nil)
if errors.IsNotFoundError(err) {
    // 处理未找到错误
}
```

**ConflictError** - 资源冲突错误
```go
err := errors.NewConflictError("user already exists", nil)
if errors.IsConflictError(err) {
    // 处理冲突错误
}
```

### 2. 认证授权错误 (`auth.go`)

**AuthenticationError** - 认证失败错误
```go
err := errors.NewAuthenticationError("invalid credentials")
if errors.IsAuthenticationError(err) {
    // 处理认证错误
}
```

**AuthorizationError** - 授权失败错误
```go
err := errors.NewAuthorizationError("insufficient permissions", nil)
if errors.IsAuthorizationError(err) {
    // 处理授权错误
}
```

### 3. 业务逻辑错误 (`business.go`)

**ValidationError** - 输入验证错误
```go
err := errors.NewValidationError("invalid email format", nil)
if errors.IsValidationError(err) {
    // 处理验证错误
}
```

**LogicError** - 业务逻辑错误
```go
err := errors.NewLogicError("invalid state transition", nil)
if errors.IsLogicError(err) {
    // 处理逻辑错误
}
```

**ProcessingError** - 数据处理错误
```go
err := errors.NewProcessingError("failed to convert data", originalErr)
// 或者使用特殊情况
err := errors.NewInvalidArgumentError("argument cannot be nil")
if errors.IsProcessingError(err) {
    // 处理处理错误
}
```

### 4. 基础设施错误 (`infrastructure.go`)

**InfrastructureError** - 基础设施错误
```go
err := errors.NewInfrastructureError("database connection failed", dbErr)
// 或者包装现有错误
err := errors.WrapInfrastructureError(dbErr, "failed to query database")
if errors.IsInfrastructureError(err) {
    // 处理基础设施错误
}
```

**ConfigurationError** - 配置错误
```go
err := errors.NewConfigurationError("missing required config", nil)
if errors.IsConfigurationError(err) {
    // 处理配置错误
}
```

**InitializationError** - 初始化错误
```go
err := errors.NewInitializationError("failed to initialize service", initErr)
if errors.IsInitializationError(err) {
    // 处理初始化错误
}
```

### 5. 领域特定错误 (`domain.go`)

**JobError** - 作业相关错误
```go
err := errors.NewNotFoundJob("job not found", nil, "job-123")
if errors.IsJobError(err) {
    // 处理作业错误
}
```

### 6. HTTP 错误 (`http.go`)

**HTTPError** - HTTP 响应错误
```go
// 通用 HTTP 错误
err := errors.NewHTTPError(404, "resource not found")

// 预定义的 HTTP 错误
err := errors.NewBadRequestError("invalid request")
err := errors.NewUnauthorizedError("authentication required")
err := errors.NewForbiddenError("access denied")
err := errors.NewInternalServerError("internal server error")

// 检查 HTTP 错误
if errors.IsHTTPError(err) {
    httpErr := err.(*errors.HTTPError)
    statusCode := httpErr.StatusCode()
}

// 从任意错误获取 HTTP 状态码
statusCode := errors.GetHTTPStatusCode(err)
```

## 通用工具 (`base.go`)

**BaseError** - 基础错误结构（可用于扩展自定义错误）
```go
baseErr := errors.NewBaseError("custom error", "something went wrong", originalErr)
```

**Wrap/Wrapf** - 错误包装函数
```go
// 包装错误并添加上下文
err := errors.Wrap(originalErr, "failed to process request")

// 格式化包装
err := errors.Wrapf(originalErr, "failed to process user %s", userID)
```

## 最佳实践

1. **选择合适的错误类型**：根据错误的性质选择最合适的错误类型
2. **保留错误链**：使用 `wrapped` 参数保留原始错误，便于追踪
3. **使用类型检查**：使用 `Is...Error()` 函数进行类型检查，而不是字符串比较
4. **提供清晰的消息**：错误消息应该清晰描述发生了什么
5. **避免信息泄露**：认证错误等敏感错误不要包含过多细节

## 错误处理示例

```go
func GetUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        if errors.IsNotFoundError(err) {
            return nil, errors.NewNotFoundError("user not found", err)
        }
        return nil, errors.WrapInfrastructureError(err, "failed to query user")
    }
    
    if !user.Active {
        return nil, errors.NewLogicError("user is inactive", nil)
    }
    
    return user, nil
}

// HTTP 处理器中的错误处理
func HandleError(w http.ResponseWriter, err error) {
    statusCode := errors.GetHTTPStatusCode(err)
    
    response := ErrorResponse{
        Error: err.Error(),
        Code:  statusCode,
    }
    
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(response)
}
```

## 迁移指南

如果你的代码使用了旧的 `errors.go` 文件，现在所有的错误类型已经被拆分到不同的文件中，但导入方式保持不变：

```go
import "your-project/pkg/errors"

// 所有的函数和类型都可以正常使用
err := errors.NewNotFoundError("not found", nil)
```

无需修改现有代码，因为所有的公共 API 保持不变。
