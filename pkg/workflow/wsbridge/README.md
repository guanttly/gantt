# Workflow-WebSocket Bridge

workflow 和 WebSocket 的集成桥接层，提供会话和连接的关联管理。

> **📖 完整文档**: 请参阅 [Workflow Infrastructure](../INFRASTRUCTURE.md)  
> 本模块已集成到 `workflow.Infrastructure` 中，推荐使用完整的基础设施而非单独使用此桥接层。

## 快速开始

推荐使用 `workflow.NewDefaultInfrastructure()` 获取完整功能：

```go
import "jusha/mcp/pkg/workflow"

// 创建完整基础设施（包含 wsbridge）
infra := workflow.NewDefaultInfrastructure(logger)

// 访问 bridge
bridge := infra.GetBridge()
```

## 核心功能

- ✅ **会话绑定**：将 WebSocket 客户端绑定到 workflow session
- ✅ **自动广播**：session 更新时自动通知所有连接的客户端
- ✅ **分组管理**：按 sessionID 自动分组管理连接
- ✅ **消息封装**：统一的消息格式

## 核心接口

### IBridge - 桥接接口
```go
type IBridge interface {
    // 绑定客户端到 session
    BindSession(client *ws.Client, sessionID string) error
    
    // 广播到 session 的所有客户端
    BroadcastToSession(sessionID string, messageType string, data any) error
    
    // session 更新回调
    OnSessionUpdate(sess *session.Session)
    
    // 消息添加回调
    OnMessageAdded(sessionID string, msg session.Message)
    
    // 获取 Hub
    GetHub() ws.IHub
    
    // 获取 session 的客户端
    GetSessionClients(sessionID string) []*ws.Client
    GetSessionClientCount(sessionID string) int
}
```

## 快速开始

### 方式 1: 使用默认实现（推荐）

```go
import (
    "jusha/mcp/pkg/workflow/wsbridge"
)

// 一键创建默认桥接（内部自动创建 Hub 和 SessionService）
bridge := wsbridge.NewDefaultBridge(logger)

// 使用桥接
server := ws.NewDefaultServer(logger,
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        var msg struct {
            Type      string `json:"type"`
            SessionID string `json:"sessionId"`
        }
        json.Unmarshal(data, &msg)
        
        // 绑定客户端到 session
        if msg.SessionID != "" {
            bridge.BindSession(client, msg.SessionID)
        }
        return nil
    }),
)
```

### 方式 2: 使用具体类型（更直接）

```go
// 创建依赖
hub := ws.NewHub()
sessionStore := session.NewInMemoryStore()
sessionService := session.NewService(sessionStore, logger)

// 创建桥接
bridge := wsbridge.NewBridge(hub, sessionService, logger)
```

### 方式 3: 自定义实现

```go
// 实现自定义 Bridge
type CustomBridge struct {
    // 自定义字段
}

func (b *CustomBridge) BindSession(client *ws.Client, sessionID string) error {
    // 自定义绑定逻辑（如记录到数据库）
}

// 其他接口方法...
```

## 使用示例

### 1. 绑定客户端到 Session

```go
// 在 WebSocket 消息处理器中
server := ws.NewServer(hub, logger,
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        var msg struct {
            Type      string `json:"type"`
            SessionID string `json:"sessionId"`
        }
        
        if err := json.Unmarshal(data, &msg); err != nil {
            return err
        }
        
        // 绑定客户端到 session
        if msg.SessionID != "" {
            if err := bridge.BindSession(client, msg.SessionID); err != nil {
                return err
            }
        }
        
        // 处理业务消息...
        
        return nil
    }),
)
```

### 3. 广播 Session 更新

```go
// 方式1：手动广播
sess, _ := sessionService.Update(ctx, sessionID, func(s *session.Session) error {
    s.SetState(session.StateProcessing, "正在处理")
    return nil
})

bridge.BroadcastToSession(sess.ID, "session_updated", sess)

// 方式2：使用回调自动广播
bridge.OnSessionUpdate(sess)
```

### 4. 广播消息

```go
// 添加消息并自动广播
sess, _ := sessionService.AddAssistantMessage(ctx, sessionID, "排班已生成")

// 广播消息
bridge.OnMessageAdded(sessionID, sess.Messages[len(sess.Messages)-1])
```

### 5. 查询连接状态

```go
// 获取 session 的所有客户端
clients := bridge.GetSessionClients(sessionID)
log.Printf("Session %s has %d clients", sessionID, len(clients))

// 获取客户端数量
count := bridge.GetSessionClientCount(sessionID)
```

## 消息格式

### 通用消息结构

```json
{
  "type": "session_updated",
  "sessionId": "session-123",
  "data": {
    // 消息数据
  }
}
```

### 常见消息类型

#### session_updated
```json
{
  "type": "session_updated",
  "sessionId": "session-123",
  "data": {
    "id": "session-123",
    "state": "processing",
    "stateDesc": "正在处理",
    "messages": [...],
    ...
  }
}
```

#### message_added
```json
{
  "type": "message_added",
  "sessionId": "session-123",
  "data": {
    "id": "msg-456",
    "role": "assistant",
    "content": "排班已生成",
    "timestamp": "2025-01-12T10:00:00Z"
  }
}
```

## 完整示例

```go
package main

import (
    "encoding/json"
    "net/http"
    
    "jusha/mcp/pkg/ws"
    "jusha/mcp/pkg/workflow/session"
    "jusha/mcp/pkg/workflow/wsbridge"
)

func main() {
    // 1. 创建基础组件
    hub := ws.NewHub()
    sessionStore := session.NewInMemoryStore()
    sessionService := session.NewService(sessionStore, logger)
    bridge := wsbridge.NewBridge(hub, sessionService, logger)
    
    // 2. 创建 WebSocket 服务器
    wsServer := ws.NewServer(hub, logger,
        ws.WithMessageHandler(handleWSMessage(bridge, sessionService)),
    )
    
    // 3. 设置路由
    http.HandleFunc("/ws", wsServer.HandleWS)
    http.HandleFunc("/api/session", handleCreateSession(sessionService, bridge))
    
    http.ListenAndServe(":8080", nil)
}

func handleWSMessage(bridge *wsbridge.Bridge, sessionService session.Service) ws.MessageHandler {
    return func(client *ws.Client, data []byte) error {
        var msg struct {
            Type      string         `json:"type"`
            SessionID string         `json:"sessionId"`
            Data      map[string]any `json:"data"`
        }
        
        if err := json.Unmarshal(data, &msg); err != nil {
            return err
        }
        
        switch msg.Type {
        case "bind":
            // 绑定到 session
            return bridge.BindSession(client, msg.SessionID)
            
        case "user_message":
            // 添加用户消息
            content := msg.Data["content"].(string)
            sess, err := sessionService.AddUserMessage(ctx, msg.SessionID, content)
            if err != nil {
                return err
            }
            
            // 广播更新
            bridge.OnSessionUpdate(sess)
            
        case "ping":
            // 心跳响应
            client.Send([]byte(`{"type":"pong"}`))
        }
        
        return nil
    }
}

func handleCreateSession(sessionService session.Service, bridge *wsbridge.Bridge) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 创建 session
        sess, err := sessionService.Create(ctx, session.CreateSessionRequest{
            OrgID:     "org123",
            UserID:    "user456",
            AgentType: "rostering",
        })
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        
        json.NewEncoder(w).Encode(sess)
    }
}
```

## 集成到现有 Agent

### 1. 更新 HTTP 传输层

```go
// agents/rostering/internal/port/http/transport.go
import (
    "jusha/mcp/pkg/ws"
    "jusha/mcp/pkg/workflow/wsbridge"
)

func NewHTTPHandler(svc service.IServiceProvider, logger logging.ILogger) http.Handler {
    r := mux.NewRouter()
    
    // 创建 WebSocket 基础设施
    hub := ws.NewHub()
    bridge := wsbridge.NewBridge(hub, svc.GetSessionService(), logger)
    
    // WebSocket 端点
    wsServer := ws.NewServer(hub, logger,
        ws.WithMessageHandler(newWSHandler(bridge, svc)),
    )
    r.Path("/scheduling/ws").HandlerFunc(wsServer.HandleWS)
    
    return r
}
```

### 2. 实现消息处理器

```go
func newWSHandler(bridge *wsbridge.Bridge, svc service.IServiceProvider) ws.MessageHandler {
    return func(client *ws.Client, data []byte) error {
        // 解析消息
        var msg ClientMessage
        if err := json.Unmarshal(data, &msg); err != nil {
            return err
        }
        
        // 绑定 session
        if msg.SessionID != "" {
            bridge.BindSession(client, msg.SessionID)
        }
        
        // 处理业务逻辑...
        
        return nil
    }
}
```

## API 参考

### Bridge

```go
type Bridge struct{}

// 创建
func NewBridge(hub *ws.Hub, sessionService session.Service, logger logging.ILogger) *Bridge

// 会话绑定
func (b *Bridge) BindSession(client *ws.Client, sessionID string) error

// 广播
func (b *Bridge) BroadcastToSession(sessionID string, messageType string, data any) error

// 回调
func (b *Bridge) OnSessionUpdate(sess *session.Session)
func (b *Bridge) OnMessageAdded(sessionID string, msg session.Message)

// 查询
func (b *Bridge) GetSessionClients(sessionID string) []*ws.Client
func (b *Bridge) GetSessionClientCount(sessionID string) int
func (b *Bridge) GetHub() *ws.Hub
```

## 设计原则

1. **松耦合**：ws 和 workflow 独立，通过 bridge 集成
2. **自动化**：session 更新自动广播
3. **简洁**：统一的消息格式和 API
4. **灵活**：支持自定义消息类型
