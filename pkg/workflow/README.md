# Workflow Engine - 通用工作流引擎

通用的工作流引擎基础设施，适用于所有基于对话的 Agent。

> **🚀 快速开始**: 推荐直接使用 [完整基础设施方案](./INFRASTRUCTURE.md)，一行代码获取 WebSocket + Session + FSM + ServiceRegistry  
> 本文档介绍底层组件，如需直接使用请跳转到完整方案文档。

## 📚 文档导航

- **[本文档 - README.md]** - 底层组件介绍（Session、Engine、WebSocket Bridge）
- **[INFRASTRUCTURE.md](./INFRASTRUCTURE.md)** - 🌟 完整基础设施方案（推荐）
- **[wsbridge/README.md](./wsbridge/README.md)** - WebSocket 桥接（已集成到 Infrastructure）

## 架构

```
pkg/workflow/
├── infrastructure.go # 完整基础设施（推荐使用）
├── session/          # 会话管理层
│   ├── session.go    # Session 模型（类型安全）
│   ├── store.go      # 存储接口 + 内存实现
│   ├── service.go    # 会话服务
│   └── factory.go    # 便利构造函数
├── engine/           # FSM 引擎层
│   ├── actor.go      # Actor (FSM 执行器)
│   ├── system.go     # System (Actor 管理)
│   ├── context.go    # 执行上下文
│   ├── types.go      # 类型定义和验证
│   ├── helper.go     # 工作流辅助函数
│   └── metrics.go    # 指标收集
└── wsbridge/         # WebSocket 集成桥接
    ├── bridge.go     # 桥接实现
    └── factory.go    # 便利构造函数
```

## 特性

### ✅ 完全通用
- 零业务依赖
- 可独立编译和测试
- 适用于任何 Agent

### ✅ 面向接口
- 核心组件均定义接口
- 提供默认实现
- 支持自定义扩展

### ✅ 会话管理
- 统一的 Session 模型
- 消息历史管理
- 状态管理
- CAS 更新支持

### ✅ 可扩展
- 业务数据通过 `Session.Data` 扩展
- 业务服务通过 `ServiceRegistry` 注入
- 灵活的元数据支持

## 核心接口

### IStore - 会话存储接口
```go
type IStore interface {
    Get(id string) (*Session, error)
    Set(session *Session) error
    Update(id string, expectedVersion int64, mutate func(*Session) error) (bool, int64, error)
    Delete(id string) error
    List(filter SessionFilter) ([]*Session, error)
    Count(filter SessionFilter) (int, error)
}
```

### ISessionService - 会话服务接口
```go
type ISessionService interface {
    // 生命周期
    Create(ctx context.Context, req CreateSessionRequest) (*Session, error)
    Get(ctx context.Context, id string) (*Session, error)
    Update(ctx context.Context, id string, mutate func(*Session) error) (*Session, error)
    Delete(ctx context.Context, id string) error
    
    // 消息管理
    AddMessage(ctx context.Context, sessionID string, msg Message) (*Session, error)
    AddUserMessage(ctx context.Context, sessionID string, content string) (*Session, error)
    AddAssistantMessage(ctx context.Context, sessionID string, content string) (*Session, error)
    
    // 状态管理
    SetState(ctx context.Context, sessionID string, state SessionState, desc string) (*Session, error)
    SetError(ctx context.Context, sessionID string, errMsg string) (*Session, error)
    
    // 业务数据
    SetData(ctx context.Context, sessionID string, key string, value any) (*Session, error)
    GetData(ctx context.Context, sessionID string, key string) (any, bool, error)
}
```

### Context - 工作流执行上下文
```go
type Context interface {
    ID() string
    Logger() logging.ILogger
    Now() time.Time
    
    // 会话访问
    Session() *session.Session
    SessionService() session.ISessionService
    
    // 业务服务访问（可插拔）
    Services() ServiceRegistry
    
    // 指标
    Metrics() Metrics
}
```

## 快速开始

### 方式 1: 使用默认实现（推荐快速开发）

```go
import (
    "jusha/mcp/pkg/workflow/session"
)

// 创建默认存储
store := session.NewDefaultStore()

// 创建默认服务
sessionService := session.NewDefaultService(store, logger)

// 创建会话
sess, err := sessionService.Create(ctx, session.CreateSessionRequest{
    OrgID:     "org123",
    UserID:    "user456",
    AgentType: "rostering",
    InitialData: map[string]any{
        "intent": "create_schedule",
    },
})

// 添加消息
sess, _ = sessionService.AddUserMessage(ctx, sess.ID, "帮我创建排班")
sess, _ = sessionService.AddAssistantMessage(ctx, sess.ID, "好的，请提供日期范围")

// 更新状态
sess, _ = sessionService.SetState(ctx, sess.ID, session.StateProcessing, "正在处理")

// 设置业务数据
sess, _ = sessionService.SetData(ctx, sess.ID, "scheduleResult", result)
```

### 2. 注册业务服务

```go
import "jusha/mcp/pkg/workflow/engine"

// 创建服务注册表
registry := engine.NewServiceRegistry()

// 注册业务服务
registry.Register("intent", intentService)
registry.Register("rostering", rosteringService)

// 在业务代码中访问
adapter := &YourAdapter{Context: wfCtx}
intentSvc := adapter.IntentService() // 类型安全
```

### 3. 定义工作流

```go
definition := &engine.WorkflowDefinition{
    Name:         "schedule_create",
    InitialState: "idle",
    Transitions: []engine.Transition{
        {
            From:  "idle",
            Event: "user_input",
            To:    "processing",
            Act: func(ctx context.Context, sess *session.Session, wfCtx engine.Context, payload any) error {
                // 业务逻辑
                adapter := &YourAdapter{Context: wfCtx}
                svc := adapter.YourService()
                
                // 访问会话数据
                orgID := sess.OrgID
                
                // 更新会话
                wfCtx.SessionService().SetData(ctx, sess.ID, "result", data)
                
                return nil
            },
        },
    },
}
```

## Session 模型

### 核心字段

```go
type Session struct {
    // 基础标识
    ID, OrgID, UserID, AgentType string
    
    // 状态
    State     SessionState  // idle, processing, waiting, completed, failed
    StateDesc string        // 状态描述
    Status    string        // active, completed, failed
    
    // 消息历史
    Messages  []Message
    
    // 工作流元数据
    WorkflowMeta *WorkflowMeta
    
    // 业务数据（可扩展）
    Data     map[string]any
    Metadata map[string]any
    
    // 版本控制
    Version int64
    
    // 时间戳
    CreatedAt, UpdatedAt, ExpireAt time.Time
}
```

### 扩展业务数据

不同 Agent 可以在 `Data` 字段存储业务特定数据：

```go
// Rostering Agent
sess.Data["intent"] = intentResult
sess.Data["scheduleResult"] = scheduleData

// Department Agent
sess.Data["deptId"] = "dept123"
sess.Data["action"] = "create"

// Rule Agent
sess.Data["ruleType"] = "scheduling"
sess.Data["ruleConfig"] = config
```

## 业务接入

### 创建 Adapter

```go
// agents/your-agent/internal/workflow/adapter.go
package workflow

import (
    "jusha/agent/your-agent/domain/service"
    "jusha/mcp/pkg/workflow/engine"
)

const (
    ServiceKeyYourService = "your_service"
)

type YourAdapter struct {
    engine.Context
}

func (c *YourAdapter) YourService() service.IYourService {
    return c.Services().MustGet(ServiceKeyYourService).(service.IYourService)
}

// 业务状态访问
func (c *YourAdapter) GetBusinessContext() (*YourContext, error) {
    sess := c.Session()
    return &YourContext{
        Field1: sess.Data["field1"].(string),
        Field2: sess.Data["field2"].(int),
    }, nil
}
```

### 初始化

```go
// agents/your-agent/setup.go
func SetupWorkflow(logger logging.ILogger, yourSvc service.IYourService) {
    // 1. 创建通用会话服务
    sessionStore := session.NewInMemoryStore()
    sessionService := session.NewService(sessionStore, logger)
    
    // 2. 注册业务服务
    registry := engine.NewServiceRegistry()
    registry.Register(ServiceKeyYourService, yourSvc)
    
    // 3. 创建 System（后续实现）
    // system := engine.NewSystem(...)
}
```

## 开发状态

✅ **已完成**
- ✅ 通用 Session 模型（类型安全）
- ✅ SessionService 接口和实现
- ✅ InMemoryStore 实现
- ✅ ServiceRegistry（业务服务解耦）
- ✅ Actor 实现（FSM 执行器）
- ✅ System 实现（Actor 生命周期管理）
- ✅ WorkflowContext（统一执行上下文）
- ✅ 事件验证（ValidateEvent、GetAvailableEvents）
- ✅ WorkflowHelper（业务层辅助函数）
- ✅ Infrastructure（完整基础设施）
- ✅ 错误标准化（pkg/errors 集成）
- ✅ Metrics 实现
- ✅ WebSocket 桥接

⏳ **待实现**
- ⏳ 业务层适配器示例（rostering agent 迁移）
- ⏳ 持久化存储实现（MySQL/Redis）
- ⏳ 单元测试补充

## 相关文档

- **[完整基础设施方案](./INFRASTRUCTURE.md)** - 开箱即用的集成方案
- **[WebSocket 管理](../ws/README.md)** - 底层 WebSocket 连接管理
- **[错误处理](../errors/)** - 标准化错误分类

## 设计原则

1. **通用优先**：会话管理是所有 Agent 的共性
2. **解耦业务**：业务服务通过注册表注入
3. **类型安全**：业务层通过 Adapter 提供编译期检查
4. **易于扩展**：新 Agent 只需实现 Adapter

## 贡献指南

添加新功能时，请遵循：
1. 保持通用性，避免业务特定逻辑
2. 完善测试覆盖
3. 更新文档和示例
