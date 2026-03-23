# MCP PKG

一个为 兼容MCP工具的 Go 语言通用组件包，提供了 AI 模型集成、配置管理、数据库操作、服务发现等核心功能。

## 特性

- **AI 模型集成**: 支持多种 AI 提供商（OpenAI、Ollama、百炼等）的统一接口
- **MCP 协议支持**: 完整实现 Model Context Protocol 2025-06-18 标准规范，支持 AI 模型与工具的标准化交互
- **Workflow 引擎**: 通用工作流基础设施，包含 FSM、会话管理、WebSocket 集成 → [详细文档](./workflow/README.md)
- **WebSocket 管理**: 通用 WebSocket 连接管理，支持分组广播和心跳 → [详细文档](./ws/README.md)
- **服务发现**: 基于 Nacos 的服务注册与发现
- **配置管理**: 统一的配置加载和管理机制
- **数据库操作**: MySQL、Redis 等数据库的封装
- **日志系统**: 基于 go-kit 的结构化日志
- **文件存储**: MinIO 对象存储集成
- **错误处理**: 标准化错误分类和错误码系统 → [详细文档](./errors/)
- **工具库**: 各种实用工具函数

## 安装

### 前置要求

由于此包托管在私有 GitLab 实例 (192.168.20.3) 上，使用前需要进行以下配置：

#### 1. 配置环境变量

```bash
# 设置私有模块和强制直接访问
export GOPROXY="direct"
export GOPRIVATE="192.168.20.3"
export GONOPROXY="192.168.20.3"
export GONOSUMDB="192.168.20.3"
```

Windows PowerShell:
```powershell
$env:GOPROXY = "direct"
$env:GOPRIVATE = "192.168.20.3"
$env:GONOPROXY = "192.168.20.3"
$env:GONOSUMDB = "192.168.20.3"
```

#### 2. 配置 Git 协议重写 ⚠️ **必须配置**

**重要**: 由于仓库仅支持 Git 访问，必须配置 URL 重写：

```bash
# 将 HTTPS 请求重写为 SSH Git 协议
git config --global url."git@192.168.20.3:".insteadOf "https:///"
```

#### 3. 确保 SSH 密钥配置

确保你的 SSH 密钥已添加到 GitLab 并且可以访问仓库：

```bash
# 测试 SSH 连接
ssh -T git@192.168.20.3
```

#### 4. 其他 Git 访问方式（可选）

使用以下方式之一配置 GitLab 访问：

**方式 1: SSH 访问（推荐）**

```bash
git config --global url."ssh://git@/".insteadOf "http:///"
```

**方式 2: Token 访问**
```bash
git config --global url."http://gitlab-ci-token:YOUR_TOKEN@/".insteadOf "http:///"
```

**方式 3: 用户名密码**
```bash
git config --global url."http://username:password@/".insteadOf "http:///"
```

#### 5. 安装包

```bash
go get jusha/mcp/pkg@latest
```

或指定版本：
```bash
go get jusha/mcp/pkg@v1.0.0
```

### 故障排除

#### 错误: `unrecognized import path` 或 `EOF`

如果遇到类似错误：
```
go: jusha/mcp/pkg@v1.0.0: unrecognized import path "jusha/mcp/pkg": https fetch: Get "https://jusha/mcp/pkg?go-get=1": EOF
```

**解决方案**：
1. 确保已设置 `GOPROXY=direct`
2. 确保已配置 Git URL 重写（步骤 2）
3. 测试 SSH 连接到 GitLab

#### 验证配置

```bash
# 检查环境变量
go env GOPROXY GOPRIVATE GONOSUMDB GONOPROXY

# 检查 Git 配置
git config --global --get-regexp url

# 测试 SSH 连接
ssh -T git@192.168.20.3
```

## 目录结构

```
├── ai/               # AI模型提供商集成
├── client/           # 服务客户端管理
├── config/           # 配置管理
├── database/         # 数据库操作
├── discovery/        # 服务发现
├── errors/           # 错误处理
├── license/          # 许可证监控
├── logging/          # 日志系统
├── mcp/              # Model Context Protocol
├── middleware/       # 中间件
├── minio/            # MinIO对象存储
├── model/            # 数据模型
├── mysql/            # MySQL数据库
├── onlyoffice/       # OnlyOffice集成
├── redis/            # Redis缓存
├── serviceinit/      # 服务初始化
├── utils/            # 工具函数库
└── version/          # 版本管理
```

## 快速开始

### MCP协议快速参考

| 操作 | 方法名 | 必需字段 | 可选字段 |
|------|--------|---------|---------|
| 初始化 | `initialize` | `protocolVersion`, `capabilities`, `clientInfo` | - |
| 列出工具 | `tools/list` | - | `cursor` |
| 调用工具 | `tools/call` | `name` | `arguments` |

所有请求必须包含：
- `jsonrpc: "2.0"`
- `id`: 请求唯一标识符
- `method`: 方法名
- `params`: 参数对象

### 1. AI 模型调用

```go
import "jusha/mcp/pkg/ai"

// 创建AI提供商
provider := ai.NewOpenAIProvider(config)
response, err := provider.Chat(context.Background(), messages)
```

### 2. 配置管理

```go
import "jusha/mcp/pkg/config"

// 加载配置
cfg, err := config.LoadConfig("config.yaml")
```

### 3. 服务发现

```go
import "jusha/mcp/pkg/discovery"

// Nacos服务发现
client := discovery.NewNacosClient(config)
services, err := client.GetServices()
```

### 4. MCP 协议

```go
import "jusha/mcp/pkg/mcp"

// 创建MCP服务器
server := mcp.NewServer("my-tool-server", "1.0.0")

// 注册工具
server.RegisterTool(mcp.Tool{
    Name: "calculator",
    Description: "Simple calculator",
    InputSchema: calculatorSchema,
})

// 启动服务器
server.Start(":8080")
```

## 支持的 AI 提供商

- **OpenAI**: GPT 系列模型
- **Ollama**: 本地模型部署
- **百炼**: 阿里云通义千问
- **本地模型**: 自定义本地模型接口

## 核心组件

### AI 模块 (`ai/`)

提供统一的 AI 模型调用接口，支持对话生成、文本向量化、结果重排等功能。

### MCP 模块 (`mcp/`)

实现 Model Context Protocol 规范，支持 AI 模型与外部工具的标准化交互。

> 📖 **详细文档**: [MCP 模块完整文档](mcp/README.md)  
> 包含协议理念、设计目的、规范说明、完整JSON示例和最佳实践

#### 快速概览

MCP (Model Context Protocol) 是一个开放标准协议，为 AI 模型提供与外部工具交互的统一接口。

**核心特性**:
- ✅ 完整实现 MCP 2025-06-18 标准规范
- ✅ 基于 JSON-RPC 2.0 协议
- ✅ 支持 HTTP/HTTPS 传输
- ✅ 内置连接池管理
- ✅ 完善的错误处理
- ✅ 类型安全的 Go 实现

#### MCP协议交互示例

MCP (Model Context Protocol) 基于 JSON-RPC 2.0，以下是标准的协议交互JSON示例：

##### 1. 初始化 (Initialize)

**客户端请求：**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "experimental": {}
    },
    "clientInfo": {
      "name": "my-mcp-client",
      "version": "1.0.0"
    }
  }
}
```

**服务器响应：**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "tools": {},
      "logging": {}
    },
    "serverInfo": {
      "name": "my-mcp-server",
      "version": "1.0.0"
    }
  }
}
```

##### 2. 列出工具 (List Tools)

**客户端请求：**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

**服务器响应：**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "get_weather",
        "description": "获取指定城市的天气信息",
        "inputSchema": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "城市名称"
            },
            "unit": {
              "type": "string",
              "description": "温度单位",
              "enum": ["celsius", "fahrenheit"]
            }
          },
          "required": ["location"]
        }
      },
      {
        "name": "search_database",
        "description": "搜索知识库",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {
              "type": "string",
              "description": "搜索关键词"
            },
            "limit": {
              "type": "integer",
              "description": "返回结果数量"
            }
          },
          "required": ["query"]
        }
      }
    ]
  }
}
```

##### 3. 调用工具 (Call Tool)

**客户端请求（带参数）：**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_weather",
    "arguments": {
      "location": "北京",
      "unit": "celsius"
    }
  }
}
```

**客户端请求（无参数，arguments字段被省略）：**
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

**服务器响应（成功）：**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "北京当前温度：15°C，晴朗"
      }
    ]
  }
}
```

**服务器响应（错误）：**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "details": "location参数不能为空"
    }
  }
}
```

##### 4. 带分页的工具列表

**客户端请求（使用cursor分页）：**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/list",
  "params": {
    "cursor": "page_2_token"
  }
}
```

**服务器响应（包含nextCursor）：**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "tools": [
      {
        "name": "tool_11",
        "description": "第11个工具",
        "inputSchema": {
          "type": "object",
          "properties": {}
        }
      }
    ],
    "nextCursor": "page_3_token"
  }
}
```

##### 5. 代码使用示例

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
    
    // 创建MCP客户端
    mcpClient := client.NewHTTPMCPClientWithTimeout(
        "http://localhost:8080/mcp",
        logger,
        30*time.Second,
    )
    
    ctx := context.Background()
    
    // 1. 初始化连接
    err := mcpClient.Initialize(ctx, model.ClientInfo{
        Name:    "my-client",
        Version: "1.0.0",
    })
    if err != nil {
        log.Fatal("初始化失败:", err)
    }
    
    // 2. 列出可用工具
    tools, err := mcpClient.ListTools(ctx)
    if err != nil {
        log.Fatal("获取工具列表失败:", err)
    }
    
    log.Printf("可用工具: %d个\n", len(tools))
    for _, tool := range tools {
        log.Printf("- %s: %s\n", tool.Name, tool.Description)
    }
    
    // 3. 调用工具
    result, err := mcpClient.CallTool(ctx, "get_weather", map[string]any{
        "location": "北京",
        "unit":     "celsius",
    })
    if err != nil {
        log.Fatal("调用工具失败:", err)
    }
    
    // 4. 处理结果
    for _, content := range result.Content {
        if content.Type == "text" {
            log.Printf("结果: %s\n", content.Text)
        }
    }
}
```

#### MCP协议关键点

1. **协议版本**: 当前最新版本为 `2025-06-18`
2. **JSON-RPC 2.0**: 所有消息必须包含 `jsonrpc: "2.0"`
3. **请求ID**: 每个请求需要唯一的ID，用于匹配响应
4. **方法命名**: 
   - 初始化: `initialize`
   - 列出工具: `tools/list`
   - 调用工具: `tools/call`
5. **可选字段**: 使用 `omitempty` 标签，空值时省略而非发送空对象
6. **错误处理**: 使用标准JSON-RPC错误格式（code, message, data）

---

📚 **需要更多信息？**

- **[MCP 模块完整文档](mcp/README.md)** - 协议理念、设计目的、完整规范和最佳实践
- **[MCP 协议格式对比](../docs/MCP_PROTOCOL_COMPARISON.md)** - 标准与非标准实现的区别
- **[MCP 合规性报告](../docs/MCP_PROTOCOL_COMPLIANCE.md)** - 协议修复和验证报告
- **[MCP 官方规范](https://spec.modelcontextprotocol.io/specification/2025-06-18/)** - 官方协议文档

---

#### 与非标准实现的区别

⚠️ **注意**: 某些实现可能使用非标准格式，例如：
- 方法名: `call_tool` 而非 `tools/call`
- 字段名: `tool_name` 而非 `name`
- 额外字段: `context` 等（非标准）

**我们的实现严格遵循MCP官方规范**，确保与标准MCP服务器的兼容性。

### 配置模块 (`config/`)

基于 Viper 的配置管理，支持 YAML、JSON 等格式，提供配置热重载。

### 服务发现 (`discovery/`)

基于 Nacos 的服务注册与发现，支持健康检查和负载均衡。

### 数据库模块 (`database/`, `mysql/`, `redis/`)

提供 MySQL、Redis 等数据库的连接池管理和操作封装。

### 日志模块 (`logging/`)

基于 go-kit/log 的结构化日志系统，支持多种输出格式和日志轮转。

## 依赖要求

- Go 1.23.2+
- MySQL 5.7+
- Redis 6.0+
- Nacos 2.0+ (可选，用于服务发现)

## 配置示例

```yaml
# common.yaml
host: "localhost"
ports:
  http: 8080
  grpc: 9090

database:
  mysql:
    host: "localhost"
    port: 3306
    username: "user"
    password: "password"
    database: "knowledge_base"

ai:
  providers:
    openai:
      api_key: "your-openai-key"
      base_url: "https://api.openai.com/v1"
    ollama:
      base_url: "http://localhost:11434"

discovery:
  nacos:
    server_addr: "localhost:8848"
    namespace_id: "public"
```

## 常见问题 (FAQ)

### MCP协议相关

**Q: 我看到其他地方的MCP实现使用 `call_tool` 方法名，为什么这里是 `tools/call`？**

A: 标准MCP协议使用 `tools/call`，而 `call_tool` 是非标准实现。我们严格遵循MCP官方规范（2025-06-18版本）。详见：[MCP协议格式对比文档](../docs/MCP_PROTOCOL_COMPARISON.md)

**Q: CallTool的arguments参数什么时候可以省略？**

A: 当工具不需要参数时，`arguments`字段应该被完全省略（而非发送空对象`{}`）。我们的实现会自动处理这个逻辑。

**Q: 如何在MCP请求中传递上下文信息（如会话ID）？**

A: 推荐在HTTP头部传递上下文信息，例如：
```go
httpReq.Header.Set("X-Conversation-ID", conversationID)
httpReq.Header.Set("X-Session-ID", sessionID)
```
不建议修改标准MCP协议结构来添加`context`字段。

**Q: MCP协议支持哪些内容类型？**

A: 目前支持两种内容类型：
- `text`: 文本内容
- `data`: 数据内容（通常是base64编码）

**Q: 如何处理大量工具的分页？**

A: MCP 协议使用**基于游标（cursor）的分页机制**。使用 `tools/list` 时：

```go
// 首次请求（获取第一页）
result, err := mcpClient.ListTools(ctx)

// 如果有更多数据，result.NextCursor 会有值
if result.NextCursor != nil {
    // 请求下一页
    nextResult, err := mcpClient.ListToolsWithCursor(ctx, result.NextCursor)
}
```

JSON 示例：
```json
{
  "method": "tools/list",
  "params": {
    "cursor": "eyJwYWdlIjoyLCJvZmZzZXQiOjUwfQ=="
  }
}
```

**关键点：**
- `cursor` 是不透明字符串，客户端不应解析或修改
- `nextCursor` 为 null 或不存在表示已是最后一页
- 详细说明请参阅：[MCP分页机制详解](mcp/README.md#分页机制详解)

### 配置相关

**Q: 如何切换不同的AI提供商？**

A: 在配置文件中设置 `ai.default_provider` 或在代码中通过工厂方法创建：
```go
provider, err := ai.NewProvider(ai.ProviderTypeOpenAI, config)
```

**Q: 支持哪些配置文件格式？**

A: 支持 YAML、JSON、TOML 等多种格式，推荐使用 YAML。

### 性能相关

**Q: MCP客户端支持连接池吗？**

A: 是的，使用 `MCPClientPool` 可以管理多个客户端连接：
```go
pool := client.NewMCPClientPool(cfg, endpoint, clientInfo, maxClients, logger)
```

**Q: 如何设置请求超时？**

A: 在创建客户端时指定超时时间：
```go
client := client.NewHTTPMCPClientWithTimeout(endpoint, logger, 30*time.Second)
```

## 相关文档

### 核心模块文档
- **[Workflow 引擎](./workflow/README.md)** - 通用工作流基础设施（FSM + Session + WebSocket）
  - [完整基础设施方案](./workflow/INFRASTRUCTURE.md) - 开箱即用的集成方案
  - [WebSocket 桥接](./workflow/wsbridge/README.md) - Session 和 WebSocket 的集成
- **[WebSocket 管理](./ws/README.md)** - 通用 WebSocket 连接管理
- **[错误处理](./errors/)** - 标准化错误分类和错误码系统

### MCP 协议文档
- [MCP协议合规性修复报告](../docs/MCP_PROTOCOL_COMPLIANCE.md)
- [MCP协议格式对比文档](../docs/MCP_PROTOCOL_COMPARISON.md)
- [MCP官方规范](https://spec.modelcontextprotocol.io/specification/2025-06-18/)

## 贡献

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

本项目采用内部许可证，仅供内部使用。

## 项目状态

本项目正在积极开发中，从 AI 知识库服务中拆分出的通用组件包，旨在提供可复用的基础设施组件。
