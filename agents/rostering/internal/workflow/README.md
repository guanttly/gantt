# Workflow Module

工作流模块负责处理所有业务流程的状态管理和流转。

## 目录结构

```
workflow/
├── README.md                    # 本文件
├── fsm_actor.go                 # FSM Actor 实现
├── common/                      # 通用组件
│   ├── README.md               # 通用组件说明
│   ├── reasons.go              # 失败原因常量
│   ├── plan_executor.go        # 通用执行计划框架 - 接口
│   ├── plan_actions.go         # 通用执行计划框架 - 实现
│   ├── PLAN_QUICK_REFERENCE.md # 计划框架快速参考 ⭐
│   ├── PLAN_ARCHITECTURE.md    # 计划框架架构说明
│   ├── PLAN_EXAMPLES.md        # 计划框架使用示例
│   └── PLAN_REFACTORING_SUMMARY.md # 重构总结
├── dept/                        # 部门管理工作流
│   ├── definition.go
│   ├── plan_executor_impl.go   # 计划执行器实现 ✅
│   ├── actions_plan.go         # (已废弃)
│   └── ...
├── rule/                        # 规则管理工作流
│   ├── definition.go
│   ├── plan_executor_impl.go   # 计划执行器实现 ✅
│   ├── actions_extract.go
│   ├── actions_query.go
│   ├── actions_delete.go
│   └── README.md
├── schedule/                    # 排班工作流
│   ├── definition.go
│   └── ...
├── general/                     # 通用工作流
└── engine/                      # 工作流引擎
    ├── actor.go
    ├── message_builder.go
    └── ...
```

## 工作流列表

### 1. Dept (部门管理) ✅

**状态**: 已完成，支持多意图执行

**功能**:
- 人员管理（创建、更新、删除）
- 团队管理（创建、更新、分配）
- 技能管理（创建、更新、分配）
- 地点管理（创建、更新）

**特性**:
- ✅ 支持多意图顺序执行
- ✅ 用户确认机制
- ✅ 执行状态追踪
- ✅ 详细的执行结果报告

**文档**: [dept/README.md](./dept/README.md)

### 2. Rule (规则管理) ✅

**状态**: 已完成，支持多意图执行

**功能**:
- 规则提取（从自然语言）
- 规则查询
- 规则更新
- 规则删除

**特性**:
- ✅ 支持多意图顺序执行
- ✅ 与 relational-graph-server 集成
- ✅ 冲突检测和处理
- ✅ 用户确认机制

**文档**: [rule/README.md](./rule/README.md)

### 3. Schedule (排班) 🚧

**状态**: 开发中

**功能**:
- 排班创建
- 排班调整
- 排班查询
- 排班优化
- 排班验证

**待实现**:
- 🚧 多意图顺序执行支持
- 🚧 计划执行器实现

### 4. General (通用) 📋

**状态**: 计划中

**功能**:
- 帮助和指引
- 闲聊处理

## 通用执行计划框架

### 概述

通用执行计划框架是一个可复用的多意图顺序执行解决方案，适用于所有需要批量操作的工作流。

### 核心概念

1. **IntentExecutor**: 意图执行器接口，定义如何执行单个意图
2. **PlanExecutorConfig**: 配置类，定义工作流特定的事件和状态
3. **PlanTransitionBuilder**: 构建器，自动生成标准的状态转换

### 标准流程

```
用户输入 → 意图识别 → 生成计划 → 用户确认 → 顺序执行 → 完成/失败
```

### 快速开始

#### 1. 实现执行器

```go
type MyExecutor struct{}

func (e *MyExecutor) ExecuteIntent(ctx, dto, wfCtx, intent) error {
    switch intent.Type {
    case IntentMyAction:
        return wfCtx.Send(ctx, Event_My_Action, intent)
    }
}

func (e *MyExecutor) GetWorkflowType() Workflow {
    return Workflow_MyWorkflow
}
```

#### 2. 配置并构建

```go
func initPlanTransitions() []Transition {
    config := common.PlanExecutorConfig{
        IntentExecutor:       &MyExecutor{},
        EventExecuteNext:     Event_My_ExecuteNext,
        EventAllCompleted:    Event_My_AllCompleted,
        EventExecutionFailed: Event_My_ExecutionFailed,
        EventPlanReady:       Event_My_PlanReady,
        StateCompleted:       State_My_Completed,
        StateFailed:          State_My_Failed,
        StateConfirming:      State_My_PlanConfirming,
    }
    
    builder := common.NewPlanTransitionBuilder(config)
    return builder.BuildPlanTransitions(...)
}
```

#### 3. 添加到工作流

```go
func collectTransitions() []Transition {
    transitions := []Transition{}
    transitions = append(transitions, initPlanTransitions()...)
    // ... 其他转换
    return transitions
}
```

### 详细文档

- 📖 [快速参考](./common/PLAN_QUICK_REFERENCE.md) ⭐ **推荐新手阅读**
- 📚 [架构说明](./common/PLAN_ARCHITECTURE.md)
- 💡 [使用示例](./common/PLAN_EXAMPLES.md)
- 📝 [重构总结](./common/PLAN_REFACTORING_SUMMARY.md)

## 工作流引擎

### 核心组件

#### FSM Actor

基于有限状态机（FSM）的 Actor 实现，负责：
- 状态管理和转换
- 事件处理
- 动作执行
- 错误处理

#### Message Builder

消息构建器，用于生成结构化的用户消息：
- 支持多语言
- 支持字段组织
- 支持富文本格式

### 状态转换定义

```go
Transition{
    From: State_A,      // 起始状态
    Event: Event_X,     // 触发事件
    To: State_B,        // 目标状态
    Act: actionFunc,    // 执行动作
}
```

### Action 函数签名

```go
func action(
    ctx context.Context,
    dto *d_model.SessionDTO,
    wfCtx IWFContext,
    payload any,
) error
```

## 开发指南

### 添加新工作流

1. **创建目录和文件**
   ```
   workflow/myworkflow/
   ├── definition.go
   ├── plan_executor_impl.go  # 如需多意图支持
   └── actions_xxx.go
   ```

2. **定义事件和状态** (`domain/workflow/event_myworkflow.go`)
   ```go
   const (
       Event_My_Start Event = "_event_my_start_"
       State_My_Idle State = "_state_my_idle_"
   )
   ```

3. **实现工作流定义** (`definition.go`)
   ```go
   func init() {
       engine.Register(&WorkflowDefinition{
           Name: Workflow_MyWorkflow,
           InitialState: State_My_Idle,
           Transitions: collectTransitions(),
       })
   }
   ```

4. **实现计划执行器**（可选，如需多意图支持）
   - 参考 `dept/plan_executor_impl.go`
   - 参考 `rule/plan_executor_impl.go`

5. **实现业务逻辑** (`actions_xxx.go`)
   - 定义 action 函数
   - 实现业务逻辑
   - 调用外部服务

6. **添加状态描述** (`domain/workflow/event_myworkflow.go`)
   ```go
   var StateMyDescriptions = map[string]map[string]string{
       string(State_My_Idle): {
           "zh-CN": "空闲",
           "en-US": "Idle",
       },
   }
   ```

7. **更新文档**
   - 创建 README.md
   - 说明功能和用法
   - 提供示例

### 最佳实践

1. **状态命名**
   - 使用有意义的状态名
   - 遵循命名规范：`State_Workflow_Action_State`
   - 示例：`State_Dept_Staff_Create_Confirming`

2. **事件命名**
   - 使用动词描述事件
   - 遵循命名规范：`Event_Workflow_Action`
   - 示例：`Event_Dept_Staff_Create`

3. **错误处理**
   - 记录详细的错误日志
   - 返回有意义的错误消息
   - 使用 error wrapping

4. **用户交互**
   - 关键操作前请求确认
   - 提供清晰的操作提示
   - 显示执行进度

5. **测试**
   - 编写单元测试
   - 编写集成测试
   - 模拟各种场景

## 工作流状态检查

### 查看当前状态

```go
dto := wfCtx.Store().Get(sessionID)
currentState := dto.State
```

### 检查执行计划

```go
if dto.ExecutionPlan != nil {
    planID := dto.ExecutionPlan.PlanID
    status := dto.ExecutionPlan.Status
    current := dto.ExecutionPlan.Current
    total := len(dto.ExecutionPlan.Intents)
}
```

### 查看执行结果

```go
for _, result := range dto.ExecutionPlan.Results {
    fmt.Printf("Intent %d: %s - %s\n",
        result.IntentIndex,
        result.IntentType,
        result.Status)
}
```

## 调试技巧

### 1. 启用详细日志

```go
logger := wfCtx.Logger()
logger.Debug("entering state", "state", currentState)
logger.Info("processing intent", "type", intent.Type)
```

### 2. 检查状态转换历史

查看 session 的 `WorkflowMeta` 字段获取工作流执行历史。

### 3. 使用断点

在关键的 action 函数中设置断点，观察：
- dto 的内容
- payload 的类型和值
- wfCtx 提供的服务

### 4. 模拟场景

编写测试用例模拟各种场景：
- 正常流程
- 用户取消
- 执行失败
- 并发更新

## 性能考虑

1. **避免阻塞**: 长时间操作应该异步执行
2. **CAS 更新**: 使用 `UpdateSessionCAS` 避免并发冲突
3. **批量操作**: 合并多个小操作为一个大操作
4. **缓存**: 缓存频繁访问的数据

## 常见问题

### Q: 如何触发工作流？

A: 通过发送事件：
```go
err := wfCtx.Send(ctx, Event_Workflow_Start, payload)
```

### Q: 如何在工作流间传递数据？

A: 通过 session 的字段或 payload：
```go
// 在 action 中更新 session
engine.UpdateSessionCAS(store, sessionID, retries, func(dto) {
    dto.Extra["key"] = value
})
```

### Q: 如何处理子工作流？

A: 触发子工作流事件，等待完成状态：
```go
err := wfCtx.Send(ctx, Event_SubWorkflow, data)
// 状态转换会自动处理完成事件
```

### Q: 如何实现超时处理？

A: 使用 context 的超时机制：
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

## 贡献指南

欢迎贡献代码和文档！

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

内部项目，保留所有权利。

---

**最后更新**: 2025-10-13  
**维护者**: 架构团队  
**版本**: v1.1
