# FSM 工作流引擎完全指南

## 📖 目录

1. [什么是 FSM？](#什么是-fsm)
2. [核心概念](#核心概念)
3. [架构设计](#架构设计)
4. [工作原理](#工作原理)
5. [如何使用 FSM](#如何使用-fsm)
6. [最佳实践](#最佳实践)
7. [常见问题](#常见问题)

---

## 什么是 FSM？

FSM（Finite State Machine，有限状态机）是一种数学模型，用于描述系统在不同状态之间的转换。在我们的调度服务中，FSM 用于管理复杂的业务工作流，如排班创建、部门管理、规则更新等。

### 为什么使用 FSM？

传统的工作流实现往往通过大量的 `if-else` 或 `switch` 语句来处理不同的业务状态，这会导致：

❌ **代码难以维护**：状态逻辑分散在各处  
❌ **扩展困难**：添加新状态需要修改多处代码  
❌ **难以追踪**：无法清晰了解业务流程的执行路径  
❌ **测试复杂**：状态转换难以全面测试

FSM 通过声明式的方式定义状态转换规则，带来以下优势：

✅ **清晰的状态管理**：所有状态和转换一目了然  
✅ **易于扩展**：添加新状态和转换无需修改现有代码  
✅ **可追踪**：完整的决策日志记录每次状态变化  
✅ **易于测试**：可以独立测试每个状态转换  
✅ **并发安全**：Actor 模型保证单会话的串行处理

---

## 核心概念

### 1. 状态（State）

**状态**表示工作流在某个时刻所处的阶段。每个状态代表了业务流程中的一个明确的节点。

```go
type State string

const (
    State_Schedule_Idle              State = "_state_schedule_idle_"
    State_Schedule_IntentRecognizing State = "_state_schedule_intent_recognizing_"
    State_Schedule_PlanGenerating    State = "_state_schedule_plan_generating_"
    State_Schedule_PlanConfirming    State = "_state_schedule_plan_confirming_"
    State_Schedule_Executing         State = "_state_schedule_executing_"
    State_Schedule_Completed         State = "_state_schedule_completed_"
    State_Schedule_Failed            State = "_state_schedule_failed_"
)
```

**状态的特点**：
- 互斥性：一个工作流在同一时刻只能处于一个状态
- 确定性：每个状态有明确的业务含义
- 有限性：状态数量是有限的

### 2. 事件（Event）

**事件**是触发状态转换的动作或信号。事件可以来自用户操作、系统回调或外部服务响应。

```go
type Event string

const (
    Event_Schedule_IntentRecognized Event = "_event_schedule_intent_recognized_"
    Event_Schedule_PlanReady        Event = "_event_schedule_plan_ready_"
    Event_Schedule_UserConfirmed    Event = "_event_schedule_user_confirmed_"
    Event_Schedule_UserCancelled    Event = "_event_schedule_user_cancelled_"
    Event_Schedule_AllCompleted     Event = "_event_schedule_all_completed_"
)
```

**事件的来源**：
- 用户操作：确认、取消、调整等按钮点击
- AI 服务回调：意图识别完成、方案生成完成
- 系统事件：超时、步骤完成、错误发生
- 外部服务：数据查询完成、规则验证完成

### 3. 转换（Transition）

**转换**定义了在特定状态下，接收到特定事件后，系统应该如何响应。

```go
type Transition struct {
    From  State                    // 起始状态
    Event Event                    // 触发事件
    To    State                    // 目标状态
    Guard func(*SessionDTO, *Actor, any) bool  // 可选：条件判断
    Act   func(context.Context, *SessionDTO, *Actor, any) error  // 可选：副作用动作
}
```

**转换示例**：
```go
Transition{
    From:  State_Schedule_PlanGenerating,
    Event: Event_Schedule_PlanReady,
    To:    State_Schedule_PlanConfirming,
    Act:   actSchedulePlanReady,  // 处理方案就绪的动作
}
```

**Guard（守卫）**：
- 在状态转换前进行条件检查
- 如果返回 `false`，转换将被拒绝
- 用于验证业务规则，如版本号匹配、权限检查等

```go
Guard: func(dto *SessionDTO, actor *Actor, payload any) bool {
    // 检查草案版本是否匹配
    if p, ok := payload.(map[string]any); ok {
        expected := p["expectedDraftVersion"].(int64)
        return dto.DraftVersion == expected
    }
    return false
}
```

**Act（动作）**：
- 在状态转换时执行的业务逻辑
- 可以调用外部服务、更新数据、发送消息等
- 通过 Context 访问依赖服务

```go
Act: func(ctx context.Context, dto *SessionDTO, actor *Actor, payload any) error {
    // 发送消息给用户
    actor.ISessionService().AddAssistantMessage(ctx, dto.SessionID, "方案已生成，请确认")
    
    // 更新工作流元数据
    dto.WorkflowMeta.Phase = "plan_confirming"
    dto.WorkflowMeta.Actions = []WorkflowAction{
        {Action: "confirm", Label: "确认"},
        {Action: "cancel", Label: "取消"},
        {Action: "adjust", Label: "调整"},
    }
    
    return nil
}
```

### 4. Actor（执行者）

**Actor** 是 FSM 的核心执行单元，每个会话（Session）对应一个 Actor 实例。

```go
type Actor struct {
    id         string              // 会话 ID
    state      State               // 当前状态
    trans      map[State]map[Event]Transition  // 转换规则表
    decisions  []Decision          // 决策日志
    inbox      chan message        // 事件消息队列
    // ... 其他字段
}
```

**Actor 的职责**：
- **状态管理**：维护当前状态
- **事件处理**：接收事件并执行相应的转换
- **决策记录**：记录每次状态变化的决策日志
- **并发控制**：通过消息队列保证事件的串行处理
- **依赖访问**：提供对各种服务的访问接口

**Actor 的生命周期**：
1. **创建**：首次发送事件时懒加载创建
2. **运行**：处理事件、执行转换、记录决策
3. **终止**：到达终态（completed/failed）
4. **回收**：终态后自动从内存中移除

### 5. System（系统）

**System** 管理所有 Actor 实例，负责路由和生命周期管理。

```go
type System struct {
    actors      map[string]*Actor   // sessionID -> Actor
    definitions map[Workflow]*WorkflowDefinition  // 工作流定义注册表
    // ... 其他字段
}
```

**System 的职责**：
- **Actor 路由**：根据 sessionID 和 workflowName 找到对应的 Actor
- **懒加载创建**：需要时才创建 Actor 实例
- **注册管理**：管理所有工作流定义的注册
- **内存管理**：定期清理已终止的 Actor
- **指标收集**：统计状态转换、执行时长等指标

### 6. 决策日志（Decision Log）

**决策日志**记录每次状态转换的详细信息，用于审计和问题排查。

```go
type Decision struct {
    At        time.Time  // 决策时间
    From      State      // 起始状态
    Event     Event      // 触发事件
    To        State      // 目标状态
    Info      string     // 附加信息
    DraftVer  int64      // 草案版本
    ResultVer int64      // 结果版本
}
```

**决策日志的用途**：
- **审计追踪**：了解工作流的完整执行路径
- **问题诊断**：快速定位状态转换异常
- **性能分析**：分析各状态的执行时长
- **用户反馈**：向用户展示流程进展

**内存优化**：
- Actor 内存中仅保留最近 1000 条决策
- WorkflowMeta.Extra.decisionLog 仅保留最近 50 条摘要供前端展示
- 完整日志可通过 `QueryDecisions` API 分页查询

---

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         前端应用                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ 聊天界面 │  │ 按钮交互 │  │ 状态展示 │  │ 日志查看 │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
└───────┼─────────────┼─────────────┼─────────────┼───────────┘
        │             │             │             │
        │ WebSocket   │ HTTP        │ Subscribe   │ HTTP API
        │             │             │             │
┌───────▼─────────────▼─────────────▼─────────────▼───────────┐
│                      API Gateway / Handler                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ 消息处理器   │  │ 命令处理器   │  │ 事件发布器   │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼──────────────────┼──────────────────┼──────────────┘
          │                  │                  │
          │ SendMessage      │ SendEvent        │ Publish
          │                  │                  │
┌─────────▼──────────────────▼──────────────────▼──────────────┐
│                      FSM Engine System                         │
│  ┌────────────────────────────────────────────────────────┐   │
│  │              Actor 路由与生命周期管理                  │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐             │   │
│  │  │ Actor 1  │  │ Actor 2  │  │ Actor N  │   ...       │   │
│  │  │(Session1)│  │(Session2)│  │(SessionN)│             │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘             │   │
│  └───────┼─────────────┼─────────────┼────────────────────┘   │
│          │             │             │                         │
│          │ Event Loop  │             │                         │
│  ┌───────▼─────────────▼─────────────▼────────────────────┐   │
│  │          状态转换引擎 (Transition Engine)              │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐             │   │
│  │  │  Guard   │  │   Act    │  │ Decision │             │   │
│  │  │  检查    │→ │  执行    │→ │  记录    │             │   │
│  │  └──────────┘  └──────────┘  └──────────┘             │   │
│  └────────────────────────────────────────────────────────┘   │
└────────────────────────┬───────────────────────────────────────┘
                         │
                         │ 依赖调用
                         │
┌────────────────────────▼───────────────────────────────────────┐
│                      业务服务层                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ SessionSvc   │  │   DataSvc    │  │   RuleSvc    │         │
│  │ (消息管理)   │  │  (数据查询)  │  │  (规则验证)  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  IntentSvc   │  │ AggregatePort│  │   ExecSvc    │         │
│  │ (意图识别)   │  │ (数据汇聚)   │  │  (执行服务)  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### 目录结构

```
services/scheduling-service/
├── domain/
│   └── workflow/
│       ├── actor_system.go      # Actor/System 接口定义
│       ├── event_schedule.go    # 排班工作流状态/事件定义
│       ├── event_dept.go        # 部门工作流状态/事件定义
│       └── event_rule.go        # 规则工作流状态/事件定义
│
├── internal/
│   └── workflow/
│       ├── fsm_actor.go         # 向后兼容层（适配旧 API）
│       ├── engine/              # FSM 核心引擎（通用、无业务逻辑）
│       │   ├── types.go         # 基础类型定义
│       │   ├── actor.go         # Actor 实现（事件循环、状态转换）
│       │   ├── system.go        # System 实现（路由、注册、管理）
│       │   ├── metrics.go       # Prometheus 指标收集
│       │   └── persist.go       # CAS 更新工具
│       │
│       ├── schedule/            # 排班工作流业务实现
│       │   ├── definition.go    # 工作流定义与注册
│       │   ├── actions_create.go # 创建排班的动作实现
│       │   ├── actions_adjust.go # 调整排班的动作实现
│       │   └── actions_query.go  # 查询排班的动作实现
│       │
│       ├── dept/                # 部门工作流业务实现
│       │   ├── definition.go
│       │   └── actions_*.go
│       │
│       └── common/              # 通用工具
│           ├── actions_helper.go # 按钮动作生成器
│           └── reasons.go        # 失败原因常量
│
└── doc/
    ├── fsm-guide.md             # FSM 完全指南（本文档）
    └── workflow/
        ├── README.md             # 工作流重构说明
        └── scheduling-interaction.md  # 排班前后端交互示例
```

### 模块职责

#### 1. Engine 层（通用引擎）

**职责**：提供通用的 FSM 执行能力，不包含任何业务逻辑

- **types.go**：定义核心类型和接口（State、Event、Transition 等）
- **actor.go**：实现 Actor 的事件循环和状态转换逻辑
- **system.go**：实现 System 的路由、注册和生命周期管理
- **metrics.go**：收集和暴露 Prometheus 指标
- **persist.go**：提供 CAS（Compare-And-Swap）更新工具

**设计原则**：
- ✅ 纯粹的状态机逻辑
- ✅ 可复用、可测试
- ✅ 无业务依赖
- ✅ 支持插件式扩展

#### 2. Workflow 层（业务工作流）

**职责**：实现具体的业务工作流逻辑

每个工作流包含：
- **definition.go**：定义状态、事件和转换规则，并在 `init()` 中注册
- **actions_*.go**：实现各个转换的副作用动作（Act）

**工作流示例**：
- `schedule/`：排班工作流（创建、调整、查询）
- `dept/`：部门工作流（创建、更新、删除、团队管理）
- `rule/`：规则工作流（更新、验证）

**扩展新工作流**：
1. 创建新目录，如 `internal/workflow/myworkflow/`
2. 编写 `definition.go` 定义状态、事件、转换
3. 编写 `actions_*.go` 实现业务逻辑
4. 在 `init()` 中调用 `engine.Register(definition)`

#### 3. Common 层（通用工具）

**职责**：提供跨工作流的通用功能

- **actions_helper.go**：生成标准化的按钮动作（Confirm、Cancel、Adjust 等）
- **reasons.go**：定义失败原因常量，用于统一的指标收集

---

## 工作原理

### 1. 事件驱动流程

```
┌─────────┐
│  用户   │ 1. 发送消息或点击按钮
└────┬────┘
     │
     ▼
┌─────────────────┐
│   Handler       │ 2. 解析为工作流事件
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│   System        │ 3. 路由到对应的 Actor
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│   Actor.inbox   │ 4. 事件进入消息队列
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ Actor.loop()    │ 5. 事件循环处理
│  - 读取事件     │
│  - 查找转换     │
│  - 执行 Guard   │ 6. 条件检查
│  - 执行 Act     │ 7. 执行业务逻辑
│  - 更新状态     │ 8. 状态迁移
│  - 记录决策     │ 9. 记录日志
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  更新 Session   │ 10. 持久化状态
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ 发送通知/消息   │ 11. 反馈给用户
└─────────────────┘
```

### 2. Actor 生命周期

```
                    ┌──────────────┐
                    │   未创建     │
                    └──────┬───────┘
                           │
                           │ 首次 SendEvent
                           ▼
                    ┌──────────────┐
                    │   创建       │
                    │ - NewActor   │
                    │ - 初始状态   │
                    │ - 启动循环   │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
            ┌──────▶│   运行中     │◀──────┐
            │       │ - 处理事件   │       │
            │       │ - 状态转换   │       │
            │       │ - 记录决策   │       │
            │       └──────┬───────┘       │
            │              │                │
            │              │ 非终态         │
            └──────────────┘                │
                           │                │
                           │ 到达终态       │
                           ▼                │
                    ┌──────────────┐        │
                    │   终止       │        │
                    │ - 标记终态   │        │
                    │ - 停止循环   │        │
                    └──────┬───────┘        │
                           │                │
                           │ 定期清理       │
                           ▼                │
                    ┌──────────────┐        │
                    │   回收       │        │
                    │ - 从内存删除 │        │
                    │ - 释放资源   │        │
                    └──────────────┘        │
                                            │
                    ┌───────────────────────┘
                    │ 新事件到达（已终止）
                    │ - 不会重新创建
                    │ - 事件被忽略
                    └───────────────────────┐
```

### 3. 状态转换过程

```go
// 伪代码示例
func (a *Actor) handle(msg message) {
    // 1. 查找转换规则
    transitions := a.trans[a.state]          // 当前状态的所有可能转换
    transition, ok := transitions[msg.event] // 查找匹配的事件
    if !ok {
        return // 无匹配转换，忽略事件
    }

    // 2. 加载会话数据
    dto := a.store.Get(a.id)
    if dto == nil {
        return
    }

    // 3. 执行守卫检查（可选）
    if transition.Guard != nil {
        if !transition.Guard(dto, a, msg.payload) {
            return // 守卫不通过，拒绝转换
        }
    }

    // 4. 执行副作用动作（可选）
    if transition.Act != nil {
        err := transition.Act(msg.ctx, dto, a, msg.payload)
        if err != nil {
            // 记录错误，但继续转换
        }
    }

    // 5. 更新状态
    prevState := a.state
    a.state = transition.To

    // 6. 记录决策
    decision := Decision{
        At:        time.Now(),
        From:      prevState,
        Event:     msg.event,
        To:        transition.To,
        DraftVer:  dto.DraftVersion,
        ResultVer: dto.ResultVersion,
    }
    a.decisions = append(a.decisions, decision)

    // 7. 更新会话元数据
    dto.WorkflowMeta.State = string(a.state)
    a.store.Update(dto)

    // 8. 发布状态变化事件
    a.system.PublishStateChange(dto.SessionID, prevState, a.state, msg.event)

    // 9. 检查是否到达终态
    if isTerminal(a.state) {
        a.terminal = true
        // 触发终态回收
    }

    // 10. 更新指标
    a.metrics.IncTransition(prevState, msg.event, a.state)
}
```

### 4. 并发安全机制

FSM 引擎通过 **Actor 模型** 保证并发安全：

**单个 Actor 的串行化**：
```go
type Actor struct {
    inbox chan message  // 消息队列
    stop  chan struct{} // 停止信号
}

func (a *Actor) loop() {
    for {
        select {
        case msg := <-a.inbox:
            a.handle(msg)  // 串行处理每个事件
        case <-a.stop:
            return
        }
    }
}

func (a *Actor) Send(ctx context.Context, event Event, payload any) error {
    a.inbox <- message{event: event, payload: payload, ctx: ctx}
    return nil
}
```

**优势**：
- ✅ 同一会话的事件按顺序处理，无竞态条件
- ✅ 不同会话的 Actor 可并行执行
- ✅ 无需显式锁，代码简洁清晰

**System 级别的并发控制**：
```go
type System struct {
    actors map[string]*Actor
    mu     sync.RWMutex  // 保护 actors map
}

func (s *System) getOrCreateActor(id string, workflow Workflow) *Actor {
    s.mu.RLock()
    actor, exists := s.actors[id]
    s.mu.RUnlock()
    if exists {
        return actor
    }

    s.mu.Lock()
    defer s.mu.Unlock()
    // Double-check
    if actor, exists := s.actors[id]; exists {
        return actor
    }
    // 创建新 Actor
    actor = NewActor(...)
    s.actors[id] = actor
    return actor
}
```

---

## 如何使用 FSM

### 1. 定义工作流

#### 步骤 1：定义状态和事件

```go
// domain/workflow/event_myworkflow.go
package workflow

const (
    Workflow_MyWorkflow Workflow = "_wf_myworkflow_"
)

// 状态定义
const (
    State_MyWorkflow_Idle       State = "_state_myworkflow_idle_"
    State_MyWorkflow_Processing State = "_state_myworkflow_processing_"
    State_MyWorkflow_Completed  State = "_state_myworkflow_completed_"
    State_MyWorkflow_Failed     State = "_state_myworkflow_failed_"
)

// 事件定义
const (
    Event_MyWorkflow_Start    Event = "_event_myworkflow_start_"
    Event_MyWorkflow_Success  Event = "_event_myworkflow_success_"
    Event_MyWorkflow_Failure  Event = "_event_myworkflow_failure_"
)
```

#### 步骤 2：实现业务动作

```go
// internal/workflow/myworkflow/actions.go
package myworkflow

import (
    "context"
    d_model "jusha/agent/rostering/domain/model"
    . "jusha/agent/rostering/domain/workflow"
)

// actStart 处理工作流启动
func actStart(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 1. 发送消息给用户
    actor.ISessionService().AddAssistantMessage(ctx, dto.SessionID, "开始处理您的请求...")

    // 2. 更新工作流元数据
    dto.WorkflowMeta.Phase = "processing"
    dto.WorkflowMeta.Actions = []d_model.WorkflowAction{
        {Action: "cancel", Label: "取消"},
    }

    // 3. 异步执行业务逻辑
    go func() {
        // 执行耗时操作
        result, err := doSomeWork(ctx, payload)
        
        // 根据结果发送后续事件
        if err != nil {
            actor.Send(ctx, Event_MyWorkflow_Failure, err.Error())
        } else {
            actor.Send(ctx, Event_MyWorkflow_Success, result)
        }
    }()

    return nil
}

// actSuccess 处理成功
func actSuccess(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    actor.ISessionService().AddAssistantMessage(ctx, dto.SessionID, "处理完成！")
    dto.WorkflowMeta.Phase = "completed"
    dto.WorkflowMeta.Actions = nil
    return nil
}

// actFailure 处理失败
func actFailure(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    errMsg := payload.(string)
    actor.ISessionService().AddAssistantMessage(ctx, dto.SessionID, "处理失败：" + errMsg)
    dto.WorkflowMeta.Phase = "failed"
    dto.WorkflowMeta.Actions = nil
    return nil
}
```

#### 步骤 3：定义转换规则

```go
// internal/workflow/myworkflow/definition.go
package myworkflow

import (
    . "jusha/agent/rostering/domain/workflow"
    "jusha/agent/rostering/internal/workflow/engine"
)

func init() {
    // 注册工作流定义
    engine.Register(&WorkflowDefinition{
        Name:         Workflow_MyWorkflow,
        InitialState: State_MyWorkflow_Idle,
        Transitions:  initTransitions(),
    })
}

func initTransitions() []Transition {
    return []Transition{
        // Idle -> Processing (启动)
        {
            From:  State_MyWorkflow_Idle,
            Event: Event_MyWorkflow_Start,
            To:    State_MyWorkflow_Processing,
            Act:   actStart,
        },
        // Processing -> Completed (成功)
        {
            From:  State_MyWorkflow_Processing,
            Event: Event_MyWorkflow_Success,
            To:    State_MyWorkflow_Completed,
            Act:   actSuccess,
        },
        // Processing -> Failed (失败)
        {
            From:  State_MyWorkflow_Processing,
            Event: Event_MyWorkflow_Failure,
            To:    State_MyWorkflow_Failed,
            Act:   actFailure,
        },
    }
}
```

### 2. 触发工作流

#### 从 Handler 触发

```go
// internal/port/http/handler.go
func (h *Handler) HandleStartWorkflow(ctx context.Context, req *StartWorkflowRequest) error {
    // 发送事件到 FSM
    err := h.fsmSystem.SendEvent(
        ctx,
        req.SessionID,           // 会话 ID
        Workflow_MyWorkflow,     // 工作流名称
        Event_MyWorkflow_Start,  // 事件
        req.Payload,             // 可选的载荷
    )
    if err != nil {
        return err
    }
    
    return nil
}
```

#### 从 WebSocket 触发

```go
// internal/port/ws/handler.go
func (h *WSHandler) HandleWorkflowCommand(msg *WorkflowCommandMessage) error {
    // 将用户命令映射为工作流事件
    var event Event
    switch msg.Command {
    case "start":
        event = Event_MyWorkflow_Start
    case "confirm":
        event = Event_MyWorkflow_Confirm
    case "cancel":
        event = Event_MyWorkflow_Cancel
    default:
        return fmt.Errorf("unknown command: %s", msg.Command)
    }
    
    // 发送事件
    return h.fsmSystem.SendEvent(
        context.Background(),
        msg.SessionID,
        Workflow_MyWorkflow,
        event,
        msg.Payload,
    )
}
```

### 3. 使用 Guard（守卫）

Guard 用于在状态转换前进行条件检查：

```go
// 版本校验守卫
func guardDraftVersion(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
    p, ok := payload.(map[string]any)
    if !ok {
        return false
    }
    
    expected, ok := p["expectedDraftVersion"].(int64)
    if !ok {
        return true // 如果没有指定版本，默认通过
    }
    
    // 检查版本是否匹配
    return dto.DraftVersion == expected
}

// 在转换中使用
Transition{
    From:  State_Schedule_PlanConfirming,
    Event: Event_Schedule_UserConfirmed,
    To:    State_Schedule_Executing,
    Guard: guardDraftVersion,  // 添加守卫
    Act:   actExecuteSchedule,
}
```

### 4. 查询决策日志

```go
// 查询最近的决策
decisions, total, err := system.QueryDecisions(sessionID, &DecisionQueryOptions{
    Limit:   50,
    Reverse: true,  // 最新的在前
})

// 按事件过滤
decisions, _, err := system.QueryDecisions(sessionID, &DecisionQueryOptions{
    Event: Event_Schedule_UserConfirmed,
    Limit: 10,
})

// 按时间范围过滤
since := time.Now().Add(-1 * time.Hour)
decisions, _, err := system.QueryDecisions(sessionID, &DecisionQueryOptions{
    Since: &since,
})
```

### 5. 收集指标

```go
// 获取指标快照
snapshot := system.MetricsSnapshot()

// 输出指标
fmt.Printf("状态转换统计:\n")
for key, count := range snapshot.TransitionCounts {
    fmt.Printf("  %s -> %s (%s): %d\n", key.From, key.To, key.Event, count)
}

fmt.Printf("\n活跃状态:\n")
for state, count := range snapshot.StateActive {
    fmt.Printf("  %s: %d\n", state, count)
}

fmt.Printf("\n状态停留时间:\n")
for state, duration := range snapshot.StateDwell {
    fmt.Printf("  %s: %v\n", state, duration)
}
```

---

## 最佳实践

### 1. 状态设计原则

#### ✅ 状态应该是互斥的

**好的设计**：
```go
const (
    State_Idle
    State_Processing
    State_Completed
)
```

**不好的设计**：
```go
const (
    State_Processing
    State_ProcessingWithError  // 应该用字段表示，而不是状态
)
```

#### ✅ 状态应该有明确的业务含义

**好的设计**：
```go
const (
    State_Schedule_Create_ConfirmingDraft  // 清晰：正在确认草案
    State_Schedule_Create_Executing        // 清晰：正在执行创建
)
```

**不好的设计**：
```go
const (
    State_Step1  // 不清晰：Step1 是什么？
    State_Step2
)
```

#### ✅ 使用分层状态命名

```go
// 工作流_子流程_状态
State_Schedule_Create_ConfirmingDraft
State_Schedule_Adjust_QueryingCurrent
State_Dept_Staff_Create_Validating
```

### 2. 事件设计原则

#### ✅ 事件应该表示"发生了什么"

**好的设计**：
```go
Event_Schedule_PlanReady      // 方案已就绪
Event_Schedule_UserConfirmed  // 用户已确认
```

**不好的设计**：
```go
Event_Schedule_ConfirmPlan  // 不清晰：是请求确认还是已确认？
```

#### ✅ 区分用户事件和系统事件

```go
// 用户事件（User-triggered）
Event_Schedule_UserConfirmed
Event_Schedule_UserCancelled
Event_Schedule_UserRequestAdjust

// 系统事件（System-triggered）
Event_Schedule_PlanReady
Event_Schedule_StepCompleted
Event_Schedule_Timeout
Event_Schedule_AIFailed
```

### 3. 动作（Act）最佳实践

#### ✅ 动作应该是幂等的

由于网络重试等原因，动作可能被多次执行，应确保幂等性：

```go
func actSendNotification(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 检查是否已发送
    if dto.WorkflowMeta.Extra["notificationSent"] == true {
        return nil  // 已发送，跳过
    }
    
    // 发送通知
    err := sendNotification(ctx, dto.SessionID)
    if err != nil {
        return err
    }
    
    // 标记已发送
    dto.WorkflowMeta.Extra["notificationSent"] = true
    return nil
}
```

#### ✅ 异步操作应该使用回调事件

对于耗时的异步操作，不要阻塞 Actor：

```go
func actStartProcessing(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 发送提示消息
    actor.ISessionService().AddAssistantMessage(ctx, dto.SessionID, "处理中...")
    
    // 启动异步任务
    go func() {
        result, err := doLongRunningTask(ctx)
        
        // 通过事件回调
        if err != nil {
            actor.Send(ctx, Event_ProcessFailed, err)
        } else {
            actor.Send(ctx, Event_ProcessCompleted, result)
        }
    }()
    
    return nil
}
```

#### ✅ 错误处理要全面

```go
func actQueryData(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 查询数据
    data, err := actor.IDataService().Query(ctx, payload)
    if err != nil {
        // 记录错误
        actor.Logger().Error("query failed", "error", err)
        
        // 发送失败事件
        actor.Send(ctx, Event_QueryFailed, err.Error())
        return err
    }
    
    // 存储结果
    dto.WorkflowMeta.Extra["queryResult"] = data
    
    // 发送成功事件
    actor.Send(ctx, Event_QueryCompleted, data)
    return nil
}
```

### 4. 守卫（Guard）最佳实践

#### ✅ 守卫应该只做检查，不做修改

```go
// ✅ 好的守卫：只检查
func guardVersionMatch(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
    expected := payload.(int64)
    return dto.DraftVersion == expected
}

// ❌ 不好的守卫：修改了状态
func guardAndUpdateVersion(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
    expected := payload.(int64)
    if dto.DraftVersion == expected {
        dto.DraftVersion++  // 不应该在守卫中修改
        return true
    }
    return false
}
```

#### ✅ 守卫应该快速返回

守卫会阻塞 Actor 的事件循环，应避免耗时操作：

```go
// ✅ 好的守卫：快速检查
func guardHasPermission(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
    return dto.UserRole == "admin"
}

// ❌ 不好的守卫：调用外部服务
func guardCheckPermissionFromAPI(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
    // 不要在守卫中调用外部 API
    hasPermission, _ := externalAPI.CheckPermission(dto.UserID)
    return hasPermission
}
```

### 5. 内存管理最佳实践

#### ✅ 及时到达终态

确保工作流最终会到达终态（completed/failed），否则 Actor 不会被回收：

```go
// 为所有异常情况提供到达终态的路径
Transition{
    From:  State_Processing,
    Event: Event_Timeout,
    To:    State_Failed,  // 超时 -> 失败（终态）
    Act:   actHandleTimeout,
}
```

#### ✅ 限制决策日志大小

决策日志会占用内存，应定期清理：

```go
// Engine 已自动限制内存决策为最近 1000 条
// WorkflowMeta.Extra.decisionLog 限制为最近 50 条

// 如需完整日志，使用 QueryDecisions API 分页查询
```

#### ✅ 清理 Extra 中的临时数据

```go
func actCompleted(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 清理不再需要的临时数据
    delete(dto.WorkflowMeta.Extra, "tempData")
    delete(dto.WorkflowMeta.Extra, "intermediateResult")
    
    // 保留重要的结果
    dto.WorkflowMeta.Extra["finalResult"] = payload
    
    return nil
}
```

---

## 常见问题

### Q1: 如何在状态转换时发送消息给用户？

在 `Act` 函数中使用 `ISessionService`：

```go
func actNotifyUser(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    return actor.ISessionService().AddAssistantMessage(
        ctx,
        dto.SessionID,
        "您的请求正在处理中...",
    )
}
```

### Q2: 如何处理超时？

使用定时器发送超时事件：

```go
func actStartWithTimeout(ctx context.Context, dto *d_model.SessionDTO, actor *Actor, payload any) error {
    // 启动超时定时器
    go func() {
        time.Sleep(5 * time.Minute)
        actor.Send(ctx, Event_Timeout, nil)
    }()
    
    // 启动实际处理
    go func() {
        result, err := doWork(ctx)
        if err != nil {
            actor.Send(ctx, Event_Failed, err)
        } else {
            actor.Send(ctx, Event_Completed, result)
        }
    }()
    
    return nil
}
```

### Q3: 如何实现条件分支？

使用多个转换 + 守卫：

```go
// 成功分支
Transition{
    From:  State_Processing,
    Event: Event_Completed,
    To:    State_Success,
    Guard: guardIsSuccess,  // 检查是否成功
    Act:   actHandleSuccess,
}

// 失败分支
Transition{
    From:  State_Processing,
    Event: Event_Completed,
    To:    State_Failed,
    Guard: guardIsFailure,  // 检查是否失败
    Act:   actHandleFailure,
}
```

### Q4: 如何在前端展示工作流状态？

通过 `WorkflowMeta` 字段：

```go
// 后端更新
dto.WorkflowMeta.Phase = "plan_confirming"
dto.WorkflowMeta.Actions = []WorkflowAction{
    {Action: "confirm", Label: "确认", Type: "primary"},
    {Action: "cancel", Label: "取消", Type: "default"},
    {Action: "adjust", Label: "调整", Type: "link"},
}
dto.WorkflowMeta.Extra["currentStep"] = 3
dto.WorkflowMeta.Extra["totalSteps"] = 5

// 前端读取
const phase = session.workflowMeta.phase;  // "plan_confirming"
const actions = session.workflowMeta.actions;  // 渲染按钮
const progress = session.workflowMeta.extra.currentStep / session.workflowMeta.extra.totalSteps;
```

### Q5: 如何调试工作流？

#### 方法 1：查看决策日志

```go
decisions, _, _ := system.QueryDecisions(sessionID, &DecisionQueryOptions{
    Limit:   100,
    Reverse: true,
})

for _, d := range decisions {
    fmt.Printf("%s: %s -[%s]-> %s\n", d.At, d.From, d.Event, d.To)
}
```

#### 方法 2：启用详细日志

```go
actor.Logger().Info("state transition",
    "from", prevState,
    "event", event,
    "to", newState,
    "payload", payload,
)
```

#### 方法 3：使用指标

```go
snapshot := system.MetricsSnapshot()
// 查看状态转换统计、活跃状态、停留时间等
```

### Q6: 如何处理并发冲突？

FSM 通过 Actor 模型自动处理：

- ✅ 同一会话的事件串行处理，无竞态
- ✅ 不同会话并行执行，互不干扰
- ✅ 使用 Guard 进行版本校验

```go
// 版本校验防止并发冲突
Transition{
    From:  State_Confirming,
    Event: Event_Confirm,
    To:    State_Executing,
    Guard: func(dto *d_model.SessionDTO, actor *Actor, payload any) bool {
        expected := payload.(map[string]any)["expectedVersion"].(int64)
        return dto.Version == expected
    },
    Act: actExecute,
}
```

### Q7: 如何扩展新的工作流？

只需 3 步：

1. **定义状态和事件**（`domain/workflow/event_*.go`）
2. **实现业务逻辑**（`internal/workflow/myworkflow/`）
3. **注册工作流**（`definition.go` 的 `init()`）

完全不需要修改 Engine 代码！

### Q8: 如何测试工作流？

```go
func TestMyWorkflow(t *testing.T) {
    // 1. 创建测试依赖
    store := NewMockStore()
    sessionSvc := NewMockSessionService()
    
    // 2. 创建 System
    system := engine.NewSystem(logger, store, sessionSvc, ...)
    
    // 3. 发送事件
    err := system.SendEvent(ctx, "session1", Workflow_MyWorkflow, Event_Start, nil)
    assert.NoError(t, err)
    
    // 4. 验证状态
    actor := system.GetActor("session1")
    assert.Equal(t, State_Processing, actor.State())
    
    // 5. 验证决策日志
    decisions, _, _ := system.QueryDecisions("session1", nil)
    assert.Len(t, decisions, 1)
    assert.Equal(t, Event_Start, decisions[0].Event)
}
```

---

## 总结

FSM 工作流引擎通过声明式的状态转换规则，为复杂的业务流程提供了清晰、可维护、可扩展的解决方案。

**核心优势**：
- 🎯 **清晰的状态管理**：一目了然的业务流程
- 🔧 **易于扩展**：添加新工作流无需修改引擎
- 📊 **完整的追踪**：决策日志记录每次状态变化
- 🔒 **并发安全**：Actor 模型保证串行处理
- 📈 **可观测性**：内置 Prometheus 指标

**下一步**：
- 📘 阅读 [排班工作流前后端交互示例](./workflow/scheduling-interaction.md)
- 📘 阅读 [FSM 工作流重构说明](./workflow/README.md)
- 🔍 查看具体工作流实现：`internal/workflow/schedule/`、`internal/workflow/dept/`

---

**文档版本**：v1.0  
**最后更新**：2025-10-24  
**维护者**：Scheduling Service Team
