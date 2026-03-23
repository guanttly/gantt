# WebSocket 通用连接管理

通用的 WebSocket 连接管理包，可独立使用，也可与其他模块集成。

## 特性

- ✅ **独立可用**：无业务依赖，纯基础设施
- ✅ **连接管理**：自动心跳、断线重连支持
- ✅ **分组广播**：支持客户端分组和定向广播
- ✅ **元数据扩展**：每个连接可携带业务元数据
- ✅ **并发安全**：线程安全的连接管理
- ✅ **灵活集成**：通过 MessageHandler 集成业务逻辑
- ✅ **面向接口**：核心组件均定义接口，支持自定义实现

## 架构

```
pkg/ws/
├── client.go    # IClient 接口 + Client 默认实现
├── hub.go       # IHub 接口 + Hub 默认实现
├── server.go    # IServer 接口 + Server 默认实现
└── factory.go   # 便利构造函数
```

## 核心接口

### IClient - 客户端接口
```go
type IClient interface {
    ID() string
    SetID(id string)
    Send(data []byte) bool
    Close()
    SetMetadata(key string, value any)
    GetMetadata(key string) (any, bool)
    Context() context.Context
}
```

### IHub - 连接管理中心接口
```go
type IHub interface {
    Register(client *Client, groupID string)
    Unregister(client *Client)
    Broadcast(groupID string, message []byte) int
    BroadcastAll(message []byte) int
    GetGroupClients(groupID string) []*Client
    ClientCount() int
    GroupCount() int
}
```

### IServer - WebSocket 服务器接口
```go
type IServer interface {
    HandleWS(w http.ResponseWriter, r *http.Request)
    SetMessageHandler(handler MessageHandler)
    GetHub() *Hub
}
```

## 快速开始

### 方式 1: 使用默认实现（推荐快速开发）

```go
import "jusha/mcp/pkg/ws"

// 一键创建默认 Server（内部自动创建 Hub）
server := ws.NewDefaultServer(logger, 
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        log.Printf("Received: %s from client %s", data, client.ID())
        client.Send([]byte("pong"))
        return nil
    }),
)

// 设置 HTTP 路由
http.HandleFunc("/ws", server.HandleWS)
http.ListenAndServe(":8080", nil)
```

### 方式 2: 使用具体类型（更直接）

```go
import "jusha/mcp/pkg/ws"

// 直接创建 Hub
hub := ws.NewHub()

// 直接创建 Server
server := ws.NewServer(hub, logger, 
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        // 处理消息
        return nil
    }),
)
```

### 方式 3: 自定义实现（高级场景）

```go
// 实现自定义 Hub（如分布式 Hub）
type RedisHub struct {
    // ... Redis 连接
}

func (h *RedisHub) Register(client *ws.Client, groupID string) {
    // 实现 Redis 存储逻辑
}

// 其他接口方法...

// 使用自定义实现
hub := &RedisHub{}
server := ws.NewDefaultServer(logger)
```

### 2. 客户端分组

```go
// 客户端连接时分配到组
server := ws.NewServer(hub, logger,
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        // 解析消息获取 sessionID
        var msg struct {
            SessionID string `json:"sessionId"`
        }
        json.Unmarshal(data, &msg)
        
        // 将客户端加入 session 组
        hub.Register(client, msg.SessionID)
        client.SetID(msg.SessionID)
        
        return nil
    }),
)

// 向特定组广播
hub.Broadcast("session-123", []byte("update"))

// 获取组内客户端数量
count := hub.GroupClientCount("session-123")
```

### 3. 元数据管理

```go
// 设置元数据
client.SetMetadata("userID", "user123")
client.SetMetadata("role", "admin")

// 读取元数据
userID, ok := client.GetMetadata("userID")
if ok {
    log.Printf("User: %v", userID)
}

// 获取所有元数据
metadata := client.GetAllMetadata()
```

### 4. 自定义升级器

```go
customUpgrader := websocket.Upgrader{
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
    CheckOrigin: func(r *http.Request) bool {
        // 限制允许的来源
        origin := r.Header.Get("Origin")
        return origin == "https://example.com"
    },
}

server := ws.NewServer(hub, logger, 
    ws.WithUpgrader(customUpgrader),
)
```

## API 参考

### Client

```go
type Client struct{}

// 消息发送
func (c *Client) Send(data []byte) bool

// 连接管理
func (c *Client) Close()
func (c *Client) ID() string
func (c *Client) SetID(id string)

// 元数据
func (c *Client) SetMetadata(key string, value any)
func (c *Client) GetMetadata(key string) (any, bool)
func (c *Client) GetAllMetadata() map[string]any

// 消息循环（由 Server 自动调用）
func (c *Client) WritePump()
func (c *Client) ReadPump(handler MessageHandler)
```

### Hub

```go
type Hub struct{}

// 客户端管理
func (h *Hub) Register(client *Client, groupID string)
func (h *Hub) Unregister(client *Client)

// 广播
func (h *Hub) Broadcast(groupID string, data []byte) int
func (h *Hub) BroadcastAll(data []byte) int

// 查询
func (h *Hub) GetGroupClients(groupID string) []*Client
func (h *Hub) GetAllClients() []*Client
func (h *Hub) ClientCount() int
func (h *Hub) GroupCount() int
func (h *Hub) GroupClientCount(groupID string) int
```

### Server

```go
type Server struct{}

// 创建
func NewServer(hub *Hub, logger logging.ILogger, opts ...ServerOption) *Server

// HTTP 处理
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request)

// 配置
func (s *Server) SetMessageHandler(handler MessageHandler)
func (s *Server) GetHub() *Hub
```

## 集成示例

### 与 Workflow 集成

```go
// 在 workflow 包中创建桥接层
package workflow

import (
    "jusha/mcp/pkg/ws"
    "jusha/mcp/pkg/workflow/session"
)

type WSBridge struct {
    hub            *ws.Hub
    sessionService session.Service
}

func (b *WSBridge) HandleMessage(client *ws.Client, data []byte) error {
    // 解析消息
    var msg Message
    if err := json.Unmarshal(data, &msg); err != nil {
        return err
    }
    
    // 绑定 session
    if msg.SessionID != "" {
        b.hub.Register(client, msg.SessionID)
        client.SetID(msg.SessionID)
    }
    
    // 更新 session
    sess, err := b.sessionService.Get(ctx, msg.SessionID)
    if err != nil {
        return err
    }
    
    // 业务逻辑...
    
    return nil
}
```

### 与 HTTP 路由集成

```go
import (
    "github.com/gorilla/mux"
    "jusha/mcp/pkg/ws"
)

func SetupRoutes(r *mux.Router, hub *ws.Hub, logger logging.ILogger) {
    server := ws.NewServer(hub, logger)
    
    r.HandleFunc("/ws", server.HandleWS)
    r.HandleFunc("/api/broadcast/{groupID}", func(w http.ResponseWriter, r *http.Request) {
        groupID := mux.Vars(r)["groupID"]
        data, _ := ioutil.ReadAll(r.Body)
        
        count := hub.Broadcast(groupID, data)
        json.NewEncoder(w).Encode(map[string]int{"sent": count})
    })
}
```

## 设计原则

1. **独立性**：不依赖任何业务模块
2. **可扩展**：通过 MessageHandler 和 Metadata 扩展
3. **高性能**：非阻塞发送，自动心跳
4. **简洁性**：核心 API 简单易用

## 性能考虑

- **发送缓冲**：每个客户端 256 字节缓冲
- **心跳间隔**：54 秒（防止代理超时）
- **读写超时**：10 秒写超时，60 秒读超时
- **非阻塞发送**：队列满时自动丢弃

## 最佳实践

1. **分组管理**：按 sessionID 或 roomID 分组
2. **元数据使用**：存储用户身份、权限等
3. **错误处理**：在 MessageHandler 中处理业务错误
4. **资源清理**：连接断开时清理业务资源
