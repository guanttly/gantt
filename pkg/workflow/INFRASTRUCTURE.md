# Workflow Infrastructure - 完整工作流基础设施

提供开箱即用的完整工作流基础设施，包含 **WebSocket** + **Session** + **Workflow Engine**。

## 🎯 核心能力

```
Infrastructure = 
    WebSocket (连接管理) 
  + Session (会话管理)
  + ServiceRegistry (业务服务注册)
  + Metrics (指标收集)
  + Bridge (自动集成)
```

**这不仅仅是 WebSocket + Session，而是完整的 Workflow 基础设施！**

## ✨ 核心优势

- ✅ **极简使用**：一行代码创建完整基础设施
- ✅ **自动集成**：WebSocket ↔ Session ↔ Workflow 自动连接
- ✅ **服务注册**：ServiceRegistry 实现业务服务解耦
- ✅ **指标收集**：内置 Prometheus 指标
- ✅ **灵活扩展**：接口化设计，可替换任何组件

## 🚀 快速开始

### 最简方式（带 Workflow 能力）

```go
package main

import (
    "net/http"
    "jusha/mcp/pkg/workflow"
)

func main() {
    // 1. 创建基础设施
    infra := workflow.NewDefaultInfrastructure(logger)
    
    // 2. 注册业务服务（Workflow 中使用）
    infra.GetServiceRegistry().Register("intentService", myIntentService)
    infra.GetServiceRegistry().Register("rosteringService", myRosteringService)
    
    // 3. 注册路由
    http.HandleFunc("/ws", infra.HandleWebSocket)
    http.HandleFunc("/api/event", handleSendEvent(infra))
    
    // 4. 启动
    http.ListenAndServe(":8080", nil)
}

// 发送工作流事件的 HTTP 端点
func handleSendEvent(infra workflow.IWorkflowInfrastructure) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            SessionID string `json:"sessionId"`
            Event     string `json:"event"`
            Payload   any    `json:"payload"`
        }
        json.NewDecoder(r.Body).Decode(&req)
        
        // 发送事件到 Workflow
        err := infra.SendEvent(
            r.Context(),
            req.SessionID,
            engine.Event(req.Event),
            req.Payload,
        )
        
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        
        w.WriteHeader(200)
    }
}
```

### Infrastructure 提供的 Workflow 能力

```go
type IWorkflowInfrastructure interface {
    // WebSocket 处理
    HandleWebSocket(w http.ResponseWriter, r *http.Request)
    
    // Session 管理
    GetSessionService() session.ISessionService
    
    // WebSocket-Session 桥接
    GetBridge() wsbridge.IBridge
    
    // 业务服务注册（核心！）
    GetServiceRegistry() engine.ServiceRegistry
    
    // 发送工作流事件（核心！）
    SendEvent(ctx context.Context, sessionID string, event engine.Event, payload any) error
}
```

### 关键：ServiceRegistry 的作用

```go
// 注册业务服务
registry := infra.GetServiceRegistry()
registry.Register("intentService", myIntentService)
registry.Register("dataService", myDataService)

// 在 Workflow Transition 的 Act 函数中使用
func createScheduleAction(ctx context.Context, sess *session.Session, wctx engine.Context, payload any) error {
    // 从 Context 获取注册的服务
    intentSvc := wctx.Services().MustGet("intentService").(IIntentService)
    dataSvc := wctx.Services().MustGet("dataService").(IDataService)
    
    // 使用服务执行业务逻辑
    intent := intentSvc.Parse(sess.ID)
    schedule := dataSvc.CreateSchedule(intent)
    
    // 更新 session
    sess.SetData("schedule", schedule)
    
    return nil
}
```

这就是 **Infrastructure 与 Workflow 的关系**：
- Infrastructure 提供基础能力（WebSocket, Session, ServiceRegistry）
- 业务 Agent 注册服务到 ServiceRegistry
- Workflow Engine (Actor/System) 通过 Context.Services() 访问这些服务
- 实现业务逻辑与基础设施的解耦

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    "jusha/mcp/pkg/workflow"
    "jusha/mcp/pkg/workflow/session"
    "jusha/mcp/pkg/ws"
)

func main() {
    // 创建基础设施
    infra := workflow.NewDefaultInfrastructure(logger, 
        ws.WithMessageHandler(handleBusinessMessage(infra)),
    )
    
    // HTTP 路由
    http.HandleFunc("/ws", infra.HandleWebSocket)
    http.HandleFunc("/api/session", handleCreateSession(infra))
    
    http.ListenAndServe(":8080", nil)
}

// 业务消息处理器
func handleBusinessMessage(infra *workflow.Infrastructure) ws.MessageHandler {
    return func(client *ws.Client, data []byte) error {
        var msg struct {
            Type      string `json:"type"`
            SessionID string `json:"sessionId"`
            Content   string `json:"content"`
        }
        
        if err := json.Unmarshal(data, &msg); err != nil {
            return err
        }
        
        sessionSvc := infra.GetSessionService()
        bridge := infra.GetBridge()
        
        switch msg.Type {
        case "bind":
            // 自动绑定 session（默认已处理）
            return bridge.BindSession(client, msg.SessionID)
            
        case "user_message":
            // 添加用户消息
            sess, err := sessionSvc.AddUserMessage(
                context.Background(), 
                msg.SessionID, 
                msg.Content,
            )
            if err != nil {
                return err
            }
            
            // 自动广播更新
            bridge.OnSessionUpdate(sess)
            
            // TODO: 触发 workflow 处理
            
        case "ping":
            client.Send([]byte(`{"type":"pong"}`))
        }
        
        return nil
    }
}

// 创建 session 的 HTTP 端点
func handleCreateSession(infra *workflow.Infrastructure) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sessionSvc := infra.GetSessionService()
        
        sess, err := sessionSvc.Create(context.Background(), session.CreateSessionRequest{
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

## 📦 包含的组件

### Infrastructure 结构

```go
type Infrastructure struct {
    Hub            ws.IHub                    // WebSocket 连接管理
    SessionService session.ISessionService    // 会话管理
    Bridge         wsbridge.IBridge           // WebSocket-Session 桥接
    WSServer       ws.IWSServer              // WebSocket 服务器
}
```

### 主要方法

```go
// 创建基础设施
infra := workflow.NewDefaultInfrastructure(logger, opts...)

// WebSocket HTTP 处理器
infra.HandleWebSocket(w, r)

// 获取服务（业务层使用）
sessionService := infra.GetSessionService()
bridge := infra.GetBridge()
```

## 🔧 自定义扩展

### 方式 1: 保持默认，扩展业务逻辑

```go
// 使用默认基础设施 + 自定义消息处理
infra := workflow.NewDefaultInfrastructure(logger,
    ws.WithMessageHandler(myCustomHandler),
)
```

### 方式 2: 替换特定组件

```go
// 使用自定义 Store（如 Redis）
redisStore := NewRedisStore(redisClient)
sessionService := session.NewService(redisStore, logger)

// 手动组装基础设施
hub := ws.NewDefaultHub()
bridge := wsbridge.NewBridge(hub, sessionService, logger)
wsServer := ws.NewDefaultServer(logger)

infra := &workflow.Infrastructure{
    Hub:            hub,
    SessionService: sessionService,
    Bridge:         bridge,
    WSServer:       wsServer,
}
```

### 方式 3: 完全自定义

```go
// 实现自己的 IHub, ISessionService 等
// 然后手动组装 Infrastructure
```

## 🎨 架构优势

### 组件自动连接

```
Client 连接 → WSServer.HandleWS()
              ↓
          自动创建 Client
              ↓
          注册到 Hub
              ↓
    默认 Handler 处理 "bind"
              ↓
      Bridge.BindSession()
              ↓
   Client ←→ Session 建立关联
```

### 消息自动转发

```
业务代码更新 Session
        ↓
  Bridge.OnSessionUpdate()
        ↓
   Hub.Broadcast(sessionID)
        ↓
所有绑定的 Client 收到更新
```

## 📝 默认消息格式

### 客户端 → 服务器

```json
{
  "type": "bind",
  "sessionId": "session-123"
}
```

### 服务器 → 客户端

```json
{
  "type": "session_updated",
  "sessionId": "session-123",
  "data": {
    "id": "session-123",
    "state": "processing",
    "messages": [...]
  }
}
```

## 🆚 对比

### 使用 Infrastructure（推荐）

```go
// 3 行代码
infra := workflow.NewDefaultInfrastructure(logger)
http.HandleFunc("/ws", infra.HandleWebSocket)
http.ListenAndServe(":8080", nil)
```

### 手动组装（灵活但繁琐）

```go
// 15+ 行代码
hub := ws.NewHub()
store := session.NewInMemoryStore()
sessionService := session.NewService(store, logger)
bridge := wsbridge.NewBridge(hub, sessionService, logger)
server := ws.NewServer(hub, logger,
    ws.WithMessageHandler(func(client *ws.Client, data []byte) error {
        // 手动解析绑定逻辑
        var msg struct {
            Type      string `json:"type"`
            SessionID string `json:"sessionId"`
        }
        json.Unmarshal(data, &msg)
        if msg.Type == "bind" {
            bridge.BindSession(client, msg.SessionID)
        }
        return nil
    }),
)
http.HandleFunc("/ws", server.HandleWS)
http.ListenAndServe(":8080", nil)
```

## 🎓 使用建议

- **快速原型**：直接使用 `NewDefaultInfrastructure`
- **生产环境**：替换 Store 为 Redis/数据库实现
- **特殊需求**：继承默认实现，重写特定方法
- **测试场景**：使用内存实现，快速启动

## 下一步

- 查看 [Session 文档](session/README.md) 了解会话管理
- 查看 [WebSocket 文档](../ws/README.md) 了解连接管理
- 查看 [Engine 文档](engine/types.go) 了解工作流引擎
