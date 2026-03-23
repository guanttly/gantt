# Model Context Protocol (MCP) 实现

> 完整实现 MCP 2025-06-18 标准规范的 Go 语言库

## 目录

- [协议理念](#协议理念)
- [设计目的](#设计目的)
- [协议规范](#协议规范)
- [消息格式](#消息格式)
- [JSON示例](#json示例)
- [使用指南](#使用指南)
- [API参考](#api参考)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

## 协议理念

### 什么是 MCP？

Model Context Protocol (MCP) 是一个**开放标准协议**，旨在为 AI 模型（如 GPT、Claude 等）提供与外部工具和数据源交互的统一接口。

### 核心理念

1. **标准化**: 提供统一的接口规范，避免各家实现不兼容
2. **可扩展性**: 支持插件式扩展，可轻松添加新工具
3. **互操作性**: 任何遵循 MCP 的客户端都能与任何 MCP 服务器通信
4. **简单性**: 基于成熟的 JSON-RPC 2.0 协议，易于实现和调试
5. **安全性**: 内置权限控制和参数验证机制

### 协议架构

```
┌─────────────────┐          MCP Protocol          ┌─────────────────┐
│                 │  ◄─────────────────────────►   │                 │
│   AI 模型/客户端 │     (JSON-RPC 2.0 over HTTP)   │   工具服务器      │
│   (MCP Client)  │                                │  (MCP Server)   │
│                 │  ◄─────────────────────────►   │                 │
└─────────────────┘                                └─────────────────┘
        │                                                   │
        │                                                   │
        ├─ Initialize                                      ├─ Tool Registry
        ├─ List Tools                                      ├─ Tool Handlers
        └─ Call Tool                                       └─ Response Builder
```

## 设计目的

### 1. 解决的问题

#### 问题：AI模型功能受限
- **现状**: AI模型只能处理文本，无法访问实时数据或执行操作
- **解决**: 通过 MCP，AI 可以调用外部工具获取数据、执行操作

#### 问题：工具接口不统一
- **现状**: 每个工具都有自己的 API 格式，集成复杂
- **解决**: MCP 提供统一的调用接口，简化集成

#### 问题：扩展性差
- **现状**: 添加新功能需要修改 AI 模型代码
- **解决**: 通过 MCP 服务器注册新工具，无需修改客户端

### 2. 应用场景

#### 🔍 数据查询
- 查询数据库
- 搜索知识库
- 获取实时信息（天气、新闻、股票等）

#### 🛠️ 工具调用
- 发送邮件
- 创建日历事件
- 调用外部 API

#### 📊 数据分析
- 执行计算
- 生成图表
- 数据转换

#### 🤖 自动化
- 工作流自动化
- 定时任务
- 批量操作

### 3. 设计目标

| 目标 | 说明 |
|------|------|
| **简单性** | 基于 JSON-RPC 2.0，易于理解和实现 |
| **可靠性** | 明确的错误处理和状态管理 |
| **性能** | 支持批量操作和连接复用 |
| **安全性** | 参数验证、权限控制、审计日志 |
| **兼容性** | 向后兼容，支持版本协商 |

## 协议规范

### 版本信息

- **当前版本**: `2025-06-18`
- **基于**: JSON-RPC 2.0
- **传输协议**: HTTP/HTTPS, WebSocket, stdio
- **编码**: UTF-8

### 协议层次

```
┌─────────────────────────────────┐
│   MCP Application Layer         │  ← 工具定义、调用逻辑
├─────────────────────────────────┤
│   MCP Protocol Layer            │  ← 标准方法：initialize, tools/list, tools/call
├─────────────────────────────────┤
│   JSON-RPC 2.0 Layer            │  ← 请求/响应格式
├─────────────────────────────────┤
│   Transport Layer (HTTP/WS)     │  ← 网络传输
└─────────────────────────────────┘
```

### 生命周期

```
Client                                    Server
  │                                         │
  ├─────── initialize ──────────────────►  │
  │                                         │
  │  ◄────── initialize result ───────────┤
  │                                         │
  ├─────── tools/list ──────────────────►  │
  │                                         │
  │  ◄────── tools list ───────────────────┤
  │                                         │
  ├─────── tools/call ──────────────────►  │
  │         (tool_name, arguments)          │
  │                                         │
  │  ◄────── tool result ──────────────────┤
  │         (content)                       │
  │                                         │
  ├─────── tools/call ──────────────────►  │
  │         (another tool)                  │
  │                                         │
  │  ◄────── tool result ──────────────────┤
  │                                         │
  └─────── close ──────────────────────►   │
```

### 标准方法

| 方法 | 用途 | 必需 |
|------|------|------|
| `initialize` | 建立连接，协商版本和能力 | ✅ 是 |
| `tools/list` | 列出所有可用工具 | ✅ 是 |
| `tools/call` | 调用指定工具 | ✅ 是 |
| `resources/list` | 列出资源（可选） | ❌ 否 |
| `prompts/list` | 列出提示模板（可选） | ❌ 否 |

### 分页机制详解

MCP 协议在 `tools/list` 等列表方法中支持**基于游标（cursor-based）的分页机制**，用于处理大量数据的场景。

#### 为什么需要分页？

假设一个 MCP 服务器注册了 1000 个工具：
- ❌ **不分页**: 一次性返回 1000 个工具的完整信息，响应体巨大（可能几MB）
- ✅ **使用分页**: 每次返回 50 个工具，客户端按需加载更多

#### 分页工作原理

```
┌──────────────┐
│   客户端      │
└──────┬───────┘
       │
       │  1. 首次请求（无 cursor）
       ├────────────────────────►  ┌──────────────┐
       │                           │   服务器      │
       │  2. 返回第1页 + nextCursor │              │
       │◄────────────────────────┤  │  [工具 1-50]  │
       │  {                        │              │
       │    tools: [1-50],         └──────────────┘
       │    nextCursor: "token_2"
       │  }
       │
       │  3. 请求第2页（带 cursor）
       ├────────────────────────►  ┌──────────────┐
       │  {cursor: "token_2"}      │   服务器      │
       │                           │              │
       │  4. 返回第2页 + nextCursor │  [工具 51-100]│
       │◄────────────────────────┤  │              │
       │  {                        └──────────────┘
       │    tools: [51-100],
       │    nextCursor: "token_3"
       │  }
       │
       │  5. 继续请求第3页...
       │
```

#### JSON 示例

##### 第一页请求（无 cursor）

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

##### 第一页响应（包含 nextCursor）

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "tool_1",
        "description": "第1个工具",
        "inputSchema": {...}
      },
      {
        "name": "tool_2",
        "description": "第2个工具",
        "inputSchema": {...}
      },
      // ... 共 50 个工具
      {
        "name": "tool_50",
        "description": "第50个工具",
        "inputSchema": {...}
      }
    ],
    "nextCursor": "eyJwYWdlIjoyLCJvZmZzZXQiOjUwfQ=="
  }
}
```

**关键字段说明：**
- `tools`: 当前页的工具列表
- `nextCursor`: 下一页的游标令牌
  - ✅ **有值**: 表示还有更多数据，客户端可以继续请求
  - ❌ **null 或不存在**: 表示已经是最后一页

##### 第二页请求（带 cursor）

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {
    "cursor": "eyJwYWdlIjoyLCJvZmZzZXQiOjUwfQ=="
  }
}
```

##### 第二页响应

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "tool_51",
        "description": "第51个工具",
        "inputSchema": {...}
      },
      // ... 更多工具
    ],
    "nextCursor": "eyJwYWdlIjozLCJvZmZzZXQiOjEwMH0="
  }
}
```

##### 最后一页响应（无 nextCursor）

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "tools": [
      {
        "name": "tool_991",
        "description": "第991个工具",
        "inputSchema": {...}
      },
      // ... 剩余 10 个工具
      {
        "name": "tool_1000",
        "description": "第1000个工具",
        "inputSchema": {...}
      }
    ]
    // 注意：没有 nextCursor 字段，表示这是最后一页
  }
}
```

#### Cursor 的实现方式

Cursor（游标）是一个**不透明的字符串令牌**，客户端不需要理解其内容，只需要原样传回服务器。

##### 常见实现方式

**1. Base64 编码的 JSON（推荐）**

```go
type PaginationState struct {
    Page   int    `json:"page"`
    Offset int    `json:"offset"`
    // 可以包含其他状态信息
}

// 编码
state := PaginationState{Page: 2, Offset: 50}
jsonData, _ := json.Marshal(state)
cursor := base64.StdEncoding.EncodeToString(jsonData)
// 结果: "eyJwYWdlIjoyLCJvZmZzZXQiOjUwfQ=="

// 解码
jsonData, _ := base64.StdEncoding.DecodeString(cursor)
var state PaginationState
json.Unmarshal(jsonData, &state)
```

**2. 加密的令牌（安全）**

```go
// 使用 AES 加密分页状态，防止客户端篡改
encryptedToken := encrypt(pageState, secretKey)
cursor := base64.URLEncoding.EncodeToString(encryptedToken)
```

**3. 数据库游标（高性能）**

```sql
-- 使用 LIMIT/OFFSET
SELECT * FROM tools 
ORDER BY id 
LIMIT 50 OFFSET 100;

-- 或使用 keyset pagination（更高效）
SELECT * FROM tools 
WHERE id > last_seen_id 
ORDER BY id 
LIMIT 50;
```

#### 代码实现示例

##### 服务器端实现

```go
package main

import (
    "encoding/base64"
    "encoding/json"
    "jusha/mcp/pkg/mcp/model"
)

type ToolPagination struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
}

func handleListTools(params *model.ListToolsRequest) (*model.ListToolsResult, error) {
    // 默认分页参数
    pageSize := 50
    page := 1
    
    // 解析 cursor
    if params.Cursor != nil && *params.Cursor != "" {
        var pagination ToolPagination
        decoded, err := base64.StdEncoding.DecodeString(*params.Cursor)
        if err == nil {
            json.Unmarshal(decoded, &pagination)
            page = pagination.Page
            pageSize = pagination.PageSize
        }
    }
    
    // 从数据库或注册表中获取工具列表
    allTools := getRegisteredTools() // 假设返回所有工具
    
    // 计算分页
    start := (page - 1) * pageSize
    end := start + pageSize
    
    if start >= len(allTools) {
        // 超出范围，返回空列表
        return &model.ListToolsResult{
            Tools: []model.Tool{},
        }, nil
    }
    
    if end > len(allTools) {
        end = len(allTools)
    }
    
    // 获取当前页的工具
    pageTools := allTools[start:end]
    
    // 构造结果
    result := &model.ListToolsResult{
        Tools: pageTools,
    }
    
    // 如果还有更多数据，生成 nextCursor
    if end < len(allTools) {
        nextPagination := ToolPagination{
            Page:     page + 1,
            PageSize: pageSize,
        }
        jsonData, _ := json.Marshal(nextPagination)
        cursor := base64.StdEncoding.EncodeToString(jsonData)
        result.NextCursor = &cursor
    }
    
    return result, nil
}
```

##### 客户端实现

```go
package main

import (
    "context"
    "fmt"
    "jusha/mcp/pkg/mcp/client"
    "jusha/mcp/pkg/mcp/model"
)

func getAllTools(mcpClient client.MCPClient) ([]model.Tool, error) {
    ctx := context.Background()
    allTools := []model.Tool{}
    
    var cursor *string = nil
    
    // 循环获取所有页
    for {
        // 构造请求参数
        req := &model.ListToolsRequest{
            Cursor: cursor,
        }
        
        // 发送请求
        result, err := mcpClient.ListToolsWithCursor(ctx, cursor)
        if err != nil {
            return nil, fmt.Errorf("获取工具列表失败: %w", err)
        }
        
        // 追加到结果中
        allTools = append(allTools, result.Tools...)
        
        fmt.Printf("已获取 %d 个工具\n", len(allTools))
        
        // 检查是否还有更多数据
        if result.NextCursor == nil || *result.NextCursor == "" {
            // 没有更多数据，退出循环
            break
        }
        
        // 使用 nextCursor 继续请求
        cursor = result.NextCursor
    }
    
    fmt.Printf("总共获取了 %d 个工具\n", len(allTools))
    return allTools, nil
}
```

#### 分页最佳实践

##### 1. 服务器端

✅ **推荐做法：**
- 设置合理的默认页大小（如 50 或 100）
- 使用不透明的 cursor，不要暴露内部实现
- cursor 包含必要的状态信息（页码、偏移量、排序等）
- 使用 Base64 或加密确保 cursor 的安全性
- 验证 cursor 的有效性，防止恶意篡改

❌ **避免：**
- cursor 使用明文数字（如 `"2"` 表示第2页）
- cursor 暴露数据库结构（如 `"offset=100&limit=50"`）
- 不验证 cursor，直接使用可能导致安全问题

##### 2. 客户端

✅ **推荐做法：**
- 将 cursor 视为不透明字符串，不要解析或修改
- 检查 `nextCursor` 是否存在来判断是否有更多数据
- 实现错误重试机制
- 提供"加载更多"或自动加载功能

❌ **避免：**
- 尝试解析或修改 cursor 的内容
- 假设 cursor 的格式或含义
- 忽略分页，一次性加载所有数据

#### 使用场景对比

| 场景 | 是否需要分页 | 原因 |
|------|------------|------|
| **工具数量 < 20** | ❌ 否 | 数据量小，一次返回即可 |
| **工具数量 20-100** | ⚠️ 可选 | 视情况而定，可以支持但不强制 |
| **工具数量 > 100** | ✅ 是 | 必须使用分页，否则响应过大 |
| **移动端应用** | ✅ 是 | 减少流量消耗 |
| **实时搜索** | ✅ 是 | 快速返回首页结果 |

#### 与传统分页的区别

| 特性 | 游标分页（Cursor-based） | 偏移分页（Offset-based） |
|------|------------------------|----------------------|
| **实现方式** | 使用不透明令牌 | 使用 `page` 或 `offset` 参数 |
| **性能** | ✅ 高（适合大数据集） | ❌ 低（OFFSET 慢） |
| **一致性** | ✅ 高（不受插入/删除影响） | ❌ 低（数据可能重复或遗漏） |
| **跳页** | ❌ 不支持直接跳转 | ✅ 支持跳转到任意页 |
| **MCP 标准** | ✅ 官方推荐 | ❌ 不在标准中 |

#### 常见问题

**Q: cursor 过期了怎么办？**

A: 服务器应该：
```go
if !isValidCursor(cursor) {
    return &RPCError{
        Code:    -32602,
        Message: "Invalid cursor",
        Data:    map[string]any{"reason": "cursor expired or invalid"},
    }
}
```

客户端收到错误后，应该重新从第一页开始请求。

**Q: 可以同时支持 cursor 和 offset 吗？**

A: 不建议。MCP 标准只定义了 cursor，添加其他分页方式会导致不兼容。

**Q: 如何知道总共有多少页？**

A: MCP 协议**不要求**返回总数。客户端应该通过 `nextCursor` 是否存在来判断是否有更多数据。如果需要总数，可以在响应中添加额外字段（非标准）：

```json
{
  "tools": [...],
  "nextCursor": "...",
  "_meta": {
    "total": 1000,
    "pageSize": 50
  }
}
```

## 消息格式

### 基础消息结构

所有 MCP 消息都遵循 JSON-RPC 2.0 格式：

```go
type JSONRPCMessage struct {
    JSONRpc string    `json:"jsonrpc"`           // 固定为 "2.0"
    ID      any       `json:"id,omitempty"`      // 请求ID（数字或字符串）
    Method  string    `json:"method,omitempty"`  // 方法名
    Params  any       `json:"params,omitempty"`  // 参数对象
    Result  any       `json:"result,omitempty"`  // 结果对象
    Error   *RPCError `json:"error,omitempty"`   // 错误对象
}
```

### 请求消息

```go
// 请求必须包含：jsonrpc, id, method, params
{
    "jsonrpc": "2.0",
    "id": <number|string>,
    "method": "<method_name>",
    "params": <object>
}
```

### 响应消息

```go
// 成功响应
{
    "jsonrpc": "2.0",
    "id": <对应的请求ID>,
    "result": <object>
}

// 错误响应
{
    "jsonrpc": "2.0",
    "id": <对应的请求ID>,
    "error": {
        "code": <number>,
        "message": "<string>",
        "data": <any>
    }
}
```

### 标准错误码

| 错误码 | 含义 | 说明 |
|--------|------|------|
| -32700 | Parse error | JSON解析错误 |
| -32600 | Invalid Request | 无效的请求格式 |
| -32601 | Method not found | 方法不存在 |
| -32602 | Invalid params | 参数无效 |
| -32603 | Internal error | 内部错误 |
| -32000 ~ -32099 | Server error | 服务器自定义错误 |

## JSON示例

### 1. Initialize - 初始化连接

#### 请求

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "experimental": {},
      "sampling": {}
    },
    "clientInfo": {
      "name": "my-ai-assistant",
      "version": "1.0.0"
    }
  }
}
```

#### 响应

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "tools": {},
      "logging": {},
      "prompts": {}
    },
    "serverInfo": {
      "name": "knowledge-base-server",
      "version": "2.1.0"
    }
  }
}
```

### 2. Tools/List - 列出工具

#### 请求（无分页）

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

#### 请求（带分页）

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {
    "cursor": "eyJwYWdlIjogMn0="
  }
}
```

#### 响应

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "search_knowledge_base",
        "description": "在知识库中搜索相关信息",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {
              "type": "string",
              "description": "搜索关键词"
            },
            "limit": {
              "type": "integer",
              "description": "返回结果数量",
              "default": 10,
              "minimum": 1,
              "maximum": 100
            },
            "filters": {
              "type": "object",
              "description": "过滤条件",
              "properties": {
                "category": {
                  "type": "string",
                  "enum": ["技术", "产品", "市场"]
                },
                "date_range": {
                  "type": "object",
                  "properties": {
                    "start": {"type": "string", "format": "date"},
                    "end": {"type": "string", "format": "date"}
                  }
                }
              }
            }
          },
          "required": ["query"]
        }
      },
      {
        "name": "get_weather",
        "description": "获取指定城市的实时天气信息",
        "inputSchema": {
          "type": "object",
          "properties": {
            "city": {
              "type": "string",
              "description": "城市名称"
            },
            "unit": {
              "type": "string",
              "description": "温度单位",
              "enum": ["celsius", "fahrenheit"],
              "default": "celsius"
            }
          },
          "required": ["city"]
        }
      },
      {
        "name": "calculate",
        "description": "执行数学计算",
        "inputSchema": {
          "type": "object",
          "properties": {
            "expression": {
              "type": "string",
              "description": "数学表达式，例如：2 + 2 * 3"
            }
          },
          "required": ["expression"]
        }
      }
    ],
    "nextCursor": "eyJwYWdlIjogM30="
  }
}
```

### 3. Tools/Call - 调用工具

#### 请求（带参数）

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "search_knowledge_base",
    "arguments": {
      "query": "如何实现微服务架构",
      "limit": 5,
      "filters": {
        "category": "技术"
      }
    }
  }
}
```

#### 请求（无参数，arguments字段省略）

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "get_current_time"
  }
}
```

#### 响应（成功）

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "找到 5 条相关结果：\n\n1. 微服务架构设计原则\n   - 单一职责\n   - 服务自治\n   - 去中心化治理\n\n2. 服务拆分策略\n   - 按业务能力拆分\n   - 按子域拆分\n   - 按团队拆分\n\n3. 通信机制\n   - REST API\n   - gRPC\n   - 消息队列\n\n4. 数据管理\n   - 每个服务独立数据库\n   - 事件驱动数据同步\n   - CQRS模式\n\n5. 部署运维\n   - 容器化（Docker）\n   - 编排（Kubernetes）\n   - 服务网格（Istio）"
      }
    ]
  }
}
```

#### 响应（包含多种内容类型）

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "这是搜索结果的文本描述"
      },
      {
        "type": "data",
        "data": "eyJyZXN1bHRzIjogWy4uLl19"
      }
    ]
  }
}
```

#### 响应（错误）

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "field": "query",
      "reason": "query参数不能为空"
    }
  }
}
```

### 4. 复杂场景示例

#### 场景：多轮对话中的工具调用

```json
// 第一次调用：搜索
{
  "jsonrpc": "2.0",
  "id": "msg_001",
  "method": "tools/call",
  "params": {
    "name": "search_knowledge_base",
    "arguments": {
      "query": "Docker容器化最佳实践"
    }
  }
}

// 响应
{
  "jsonrpc": "2.0",
  "id": "msg_001",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "找到以下Docker最佳实践：\n1. 使用多阶段构建\n2. 最小化镜像层数\n3. 使用.dockerignore\n..."
      }
    ]
  }
}

// 第二次调用：基于第一次结果继续查询
{
  "jsonrpc": "2.0",
  "id": "msg_002",
  "method": "tools/call",
  "params": {
    "name": "get_document_detail",
    "arguments": {
      "doc_id": "docker_best_practices_001"
    }
  }
}
```

## 使用指南

### 客户端实现

#### 1. 创建客户端

```go
package main

import (
    "context"
    "log"
    "time"
    
    "jusha/mcp/pkg/mcp/client"
    "jusha/mcp/pkg/mcp/model"
    "jusha/mcp/pkg/logging"
)

func main() {
    logger := slog.Default()
    
    // 创建HTTP客户端
    mcpClient := client.NewHTTPMCPClientWithTimeout(
        "http://localhost:8080/mcp",
        logger,
        30*time.Second,
    )
    
    ctx := context.Background()
    
    // 初始化
    err := mcpClient.Initialize(ctx, model.ClientInfo{
        Name:    "my-ai-client",
        Version: "1.0.0",
    })
    if err != nil {
        log.Fatal("初始化失败:", err)
    }
    
    log.Println("✓ 客户端初始化成功")
}
```

#### 2. 列出可用工具

```go
// 获取所有工具
tools, err := mcpClient.ListTools(ctx)
if err != nil {
    log.Fatal("获取工具列表失败:", err)
}

// 打印工具信息
for _, tool := range tools {
    log.Printf("工具: %s\n", tool.Name)
    log.Printf("  描述: %s\n", tool.Description)
    log.Printf("  参数: %+v\n", tool.InputSchema)
}
```

#### 3. 调用工具

```go
// 调用工具（带参数）
result, err := mcpClient.CallTool(ctx, "search_knowledge_base", map[string]any{
    "query": "Go语言并发编程",
    "limit": 10,
})
if err != nil {
    log.Fatal("调用工具失败:", err)
}

// 处理结果
for _, content := range result.Content {
    switch content.Type {
    case "text":
        log.Printf("结果: %s\n", content.Text)
    case "data":
        log.Printf("数据: %s\n", content.Data)
    }
}
```

#### 4. 使用连接池

```go
// 创建连接池
pool := client.NewMCPClientPool(
    cfg,
    "http://localhost:8080/mcp",
    model.ClientInfo{Name: "pool-client", Version: "1.0.0"},
    10, // 最大连接数
    logger,
)

// 初始化连接池
err := pool.Initialize(ctx)
if err != nil {
    log.Fatal("连接池初始化失败:", err)
}

// 获取客户端
client, err := pool.GetClient()
if err != nil {
    log.Fatal("获取客户端失败:", err)
}
defer pool.ReturnClient(client)

// 使用客户端
tools, err := client.ListTools(ctx)
```

### 服务器实现

#### 1. 创建服务器

```go
package main

import (
    "context"
    "log"
    
    "jusha/mcp/pkg/mcp"
    "jusha/mcp/pkg/mcp/model"
)

func main() {
    // 创建服务器
    server := mcp.NewDefaultMCPServer(
        model.ServerInfo{
            Name:    "my-tool-server",
            Version: "1.0.0",
        },
        logger,
    )
    
    // 注册工具
    registerTools(server)
    
    // 启动HTTP服务
    log.Println("启动MCP服务器在 :8080")
    if err := server.ServeHTTP(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

#### 2. 注册工具

```go
func registerTools(server *mcp.DefaultMCPServer) {
    // 注册搜索工具
    server.RegisterTool(model.Tool{
        Name:        "search_knowledge_base",
        Description: "在知识库中搜索信息",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "query": map[string]any{
                    "type":        "string",
                    "description": "搜索关键词",
                },
                "limit": map[string]any{
                    "type":        "integer",
                    "description": "返回结果数量",
                    "default":     10,
                },
            },
            "required": []string{"query"},
        },
    }, handleSearchKnowledgeBase)
    
    // 注册计算器工具
    server.RegisterTool(model.Tool{
        Name:        "calculate",
        Description: "执行数学计算",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "expression": map[string]any{
                    "type":        "string",
                    "description": "数学表达式",
                },
            },
            "required": []string{"expression"},
        },
    }, handleCalculate)
}
```

#### 3. 实现工具处理器

```go
func handleSearchKnowledgeBase(ctx context.Context, args map[string]any) (*model.CallToolResult, error) {
    // 提取参数
    query, ok := args["query"].(string)
    if !ok || query == "" {
        return nil, fmt.Errorf("query参数无效")
    }
    
    limit := 10
    if l, ok := args["limit"].(float64); ok {
        limit = int(l)
    }
    
    // 执行搜索逻辑
    results := searchInDatabase(query, limit)
    
    // 构建响应
    return &model.CallToolResult{
        Content: []model.Content{
            model.NewTextContent(formatSearchResults(results)),
        },
    }, nil
}

func handleCalculate(ctx context.Context, args map[string]any) (*model.CallToolResult, error) {
    expression, ok := args["expression"].(string)
    if !ok {
        return nil, fmt.Errorf("expression参数无效")
    }
    
    // 执行计算
    result, err := evaluate(expression)
    if err != nil {
        return nil, fmt.Errorf("计算错误: %w", err)
    }
    
    return &model.CallToolResult{
        Content: []model.Content{
            model.NewTextContent(fmt.Sprintf("计算结果: %s = %v", expression, result)),
        },
    }, nil
}
```

## API参考

### 客户端接口

```go
type MCPClient interface {
    // 初始化连接
    Initialize(ctx context.Context, clientInfo model.ClientInfo) error
    
    // 列出所有工具
    ListTools(ctx context.Context) ([]model.Tool, error)
    
    // 调用工具
    CallTool(ctx context.Context, name string, arguments map[string]any) (*model.CallToolResult, error)
    
    // 关闭连接
    Close() error
}
```

### 服务器接口

```go
type MCPServer interface {
    // 注册工具
    RegisterTool(tool model.Tool, handler ToolHandler)
    
    // 启动HTTP服务
    ServeHTTP(addr string) error
    
    // 处理请求
    HandleRequest(ctx context.Context, data []byte) ([]byte, error)
}
```

### 数据模型

```go
// 工具定义
type Tool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}

// 工具调用请求
type CallToolRequest struct {
    Name      string         `json:"name"`
    Arguments map[string]any `json:"arguments,omitempty"`
}

// 工具调用结果
type CallToolResult struct {
    Content []Content `json:"content"`
}

// 内容类型
type Content struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    Data string `json:"data,omitempty"`
}
```

## 最佳实践

### 1. 工具设计原则

#### ✅ 单一职责
每个工具应该只做一件事，并且做好：

```go
// ✅ 好的设计
tools := []model.Tool{
    {Name: "search_users", Description: "搜索用户"},
    {Name: "create_user", Description: "创建用户"},
    {Name: "update_user", Description: "更新用户"},
}

// ❌ 不好的设计
tools := []model.Tool{
    {Name: "user_operations", Description: "所有用户操作"},
}
```

#### ✅ 清晰的命名
使用动词+名词的命名方式：

```go
// ✅ 清晰的命名
"get_weather"
"search_knowledge_base"
"calculate_distance"
"send_email"

// ❌ 不清晰的命名
"weather"
"search"
"calc"
"mail"
```

#### ✅ 详细的描述和参数说明

```go
tool := model.Tool{
    Name: "search_knowledge_base",
    Description: "在知识库中搜索相关文档和信息。支持全文搜索和过滤条件。",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "query": map[string]any{
                "type":        "string",
                "description": "搜索关键词，支持多个关键词用空格分隔",
                "minLength":   1,
                "maxLength":   200,
            },
            "limit": map[string]any{
                "type":        "integer",
                "description": "返回结果的最大数量",
                "default":     10,
                "minimum":     1,
                "maximum":     100,
            },
        },
        "required": []string{"query"},
    },
}
```

### 2. 错误处理

#### 使用标准错误码

```go
// 参数错误
if query == "" {
    return &model.MCPMessage{
        JSONRPCMessage: model.JSONRPCMessage{
            JSONRpc: "2.0",
            ID:      requestID,
            Error: &model.RPCError{
                Code:    -32602,
                Message: "Invalid params",
                Data: map[string]any{
                    "field":  "query",
                    "reason": "query不能为空",
                },
            },
        },
    }, nil
}

// 工具不存在
if tool == nil {
    return &model.MCPMessage{
        JSONRPCMessage: model.JSONRPCMessage{
            JSONRpc: "2.0",
            ID:      requestID,
            Error: &model.RPCError{
                Code:    -32601,
                Message: "Method not found",
                Data: map[string]any{
                    "tool": toolName,
                },
            },
        },
    }, nil
}
```

#### 提供详细的错误信息

```go
// ✅ 好的错误信息
return nil, fmt.Errorf("数据库查询失败: 连接超时，请稍后重试")

// ❌ 不好的错误信息
return nil, fmt.Errorf("error")
```

### 3. 性能优化

#### 使用连接池

```go
// 创建连接池而不是每次创建新连接
pool := client.NewMCPClientPool(cfg, endpoint, clientInfo, 10, logger)
defer pool.Close()
```

#### 设置合理的超时

```go
// 为不同操作设置不同的超时时间
shortTimeout := 5 * time.Second   // 快速操作
mediumTimeout := 30 * time.Second // 普通操作
longTimeout := 5 * time.Minute    // 长时间操作

client := client.NewHTTPMCPClientWithTimeout(endpoint, logger, mediumTimeout)
```

#### 批量操作

```go
// 如果需要调用多个工具，考虑批量处理
var wg sync.WaitGroup
results := make(chan *model.CallToolResult, len(tools))

for _, toolName := range tools {
    wg.Add(1)
    go func(name string) {
        defer wg.Done()
        result, _ := client.CallTool(ctx, name, args)
        results <- result
    }(toolName)
}

wg.Wait()
close(results)
```

### 4. 安全性

#### 参数验证

```go
func validateSearchParams(args map[string]any) error {
    query, ok := args["query"].(string)
    if !ok {
        return fmt.Errorf("query必须是字符串")
    }
    
    if len(query) == 0 {
        return fmt.Errorf("query不能为空")
    }
    
    if len(query) > 1000 {
        return fmt.Errorf("query长度不能超过1000个字符")
    }
    
    // 检查SQL注入
    if containsSQLInjection(query) {
        return fmt.Errorf("query包含非法字符")
    }
    
    return nil
}
```

#### 权限控制

```go
func (s *MCPServer) HandleRequest(ctx context.Context, data []byte) ([]byte, error) {
    // 从上下文中获取用户信息
    user := getUserFromContext(ctx)
    
    // 检查权限
    if !user.HasPermission("tools:call") {
        return errorResponse(-32000, "Permission denied")
    }
    
    // 继续处理请求
    // ...
}
```

#### 审计日志

```go
func (s *MCPServer) CallTool(ctx context.Context, name string, args map[string]any) (*model.CallToolResult, error) {
    // 记录调用日志
    s.logger.Info("工具调用",
        "tool", name,
        "user", getUserFromContext(ctx),
        "args", args,
        "timestamp", time.Now(),
    )
    
    result, err := s.executeTool(ctx, name, args)
    
    // 记录结果
    s.logger.Info("工具调用完成",
        "tool", name,
        "success", err == nil,
        "duration", time.Since(start),
    )
    
    return result, err
}
```

### 5. 测试

#### 单元测试示例

```go
func TestCallTool(t *testing.T) {
    // 创建测试服务器
    server := mcp.NewDefaultMCPServer(
        model.ServerInfo{Name: "test-server", Version: "1.0.0"},
        slog.Default(),
    )
    
    // 注册测试工具
    server.RegisterTool(model.Tool{
        Name:        "test_tool",
        Description: "测试工具",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "input": map[string]any{"type": "string"},
            },
        },
    }, func(ctx context.Context, args map[string]any) (*model.CallToolResult, error) {
        return &model.CallToolResult{
            Content: []model.Content{
                model.NewTextContent("test result"),
            },
        }, nil
    })
    
    // 测试调用
    ctx := context.Background()
    request := createCallToolRequest("test_tool", map[string]any{
        "input": "test",
    })
    
    response, err := server.HandleRequest(ctx, request)
    assert.NoError(t, err)
    assert.Contains(t, string(response), "test result")
}
```

## 常见问题

### Q1: 为什么我看到的其他实现使用 `call_tool` 而不是 `tools/call`？

**A:** 标准 MCP 协议使用 `tools/call`。如果您看到 `call_tool`，那可能是：
- 非标准的自定义实现
- 过时的文档或示例
- 其他类似但不同的协议

我们的实现严格遵循 MCP 官方规范 2025-06-18 版本。

### Q2: arguments 字段什么时候可以省略？

**A:** 当工具不需要任何参数时，`arguments` 字段应该完全省略（而不是发送空对象 `{}`）。我们的客户端实现会自动处理这个逻辑。

### Q3: 如何在请求中传递会话上下文？

**A:** 推荐的方式是在 HTTP 头部传递：

```go
httpReq.Header.Set("X-Conversation-ID", conversationID)
httpReq.Header.Set("X-Session-ID", sessionID)
httpReq.Header.Set("X-User-ID", userID)
```

不要修改标准 MCP 协议结构来添加 `context` 字段。

### Q4: 支持 WebSocket 传输吗？

**A:** 目前主要支持 HTTP/HTTPS 传输。WebSocket 支持可以通过自定义传输层实现。

### Q5: 如何处理大文件或流式响应？

**A:** 对于大文件，建议：
1. 使用 `data` 类型的 Content，包含 base64 编码的数据
2. 或者返回文件的 URL，让客户端单独下载
3. 对于流式响应，可以考虑使用 WebSocket 或 Server-Sent Events

### Q6: 工具执行超时如何处理？

**A:** 在工具处理器中使用 context 的超时控制：

```go
func handleLongRunningTool(ctx context.Context, args map[string]any) (*model.CallToolResult, error) {
    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()
    
    // 执行长时间操作
    result := make(chan *model.CallToolResult)
    go func() {
        // 执行实际操作
        result <- doWork(args)
    }()
    
    select {
    case r := <-result:
        return r, nil
    case <-ctx.Done():
        return nil, fmt.Errorf("操作超时")
    }
}
```

### Q7: 如何实现工具的版本管理？

**A:** 建议在工具名称中包含版本信息：

```go
tools := []model.Tool{
    {Name: "search_v1", Description: "搜索 v1（将废弃）"},
    {Name: "search_v2", Description: "搜索 v2（推荐）"},
}
```

或者在 Initialize 时协商版本。

## 相关资源

- **官方规范**: [MCP Specification](https://spec.modelcontextprotocol.io/specification/2025-06-18/)
- **协议对比**: [MCP协议格式对比文档](../../docs/MCP_PROTOCOL_COMPARISON.md)
- **合规性报告**: [MCP协议合规性修复报告](../../docs/MCP_PROTOCOL_COMPLIANCE.md)
- **JSON-RPC 2.0**: [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- **GitHub**: [MCP GitHub Repository](https://github.com/modelcontextprotocol)

## 贡献

欢迎提交问题和改进建议！

## 许可证

本项目采用内部许可证，仅供内部使用。

---

**版本**: 1.0.0  
**最后更新**: 2025年11月4日  
**协议版本**: MCP 2025-06-18
