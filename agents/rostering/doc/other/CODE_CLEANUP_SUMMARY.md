# Rostering Agent 代码清理总结

## 清理时间
2025-11-12

## 清理目标
删除所有基于旧架构的实现代码，为基于 `pkg/workflow` 的全新实现做准备。保留业务逻辑核心（IntentService、RosteringService）和文档（SOP）。

## 已删除的目录和文件

### 1. domain/workflow/ (完整删除)
旧的工作流定义和 Actor System：
- `actor_system.go` - 旧的 Actor System 接口和实现
- `event.go` - 旧的事件定义
- `event_dept.go` - 部门管理工作流事件
- `event_general.go` - 通用工作流事件
- `event_rule.go` - 规则管理工作流事件
- `event_schedule.go` - 排班工作流事件

### 2. internal/workflow/ (部分删除)
保留目录结构和 README.md，删除实现代码：

**删除的子目录：**
- `common/` - 通用工作流逻辑（command.go, plan_actions.go, plan_executor.go, reasons.go）
- `engine/` - 旧的工作流引擎（actor.go, system.go, persist.go, metrics.go, message_builder.go 等）

**删除的文件：**
- `fsm_actor.go` - 有限状态机 Actor 实现
- `schedule/actions_create.go` - 排班创建流程的所有 Action 实现
- `schedule/definition.go` - 排班工作流定义
- `schedule/plan_executor_impl.go` - 计划执行器实现

**保留的文件：**
- `README.md` - 工作流文档
- `schedule/SOP_CREATE.md` - **排班创建标准操作流程（重要）**

### 3. internal/task/ (完整删除)
旧的任务实现：
- `task.go` - 任务基础结构
- `interfaces.go` - 任务接口
- `schedule_task.go` - 排班任务
- `dept_task.go` - 部门管理任务
- `general_task.go` - 通用任务
- `rule_task.go` - 规则任务

### 4. internal/port/ (部分删除)
旧的 HTTP 和 WebSocket 实现：

**删除的子目录：**
- `handlers/` - HTTP 路由处理器（scheduling.go 等）
- `ws/` - WebSocket 实现（server.go, hub.go, client.go, observer.go, publisher.go, broadcaster.go, messages.go, message_sender.go）

**删除的文件：**
- `http/transport.go` - 旧的 HTTP 传输层（已重新创建临时版本）

**新创建的文件：**
- `http/transport.go` - 临时实现，仅包含健康检查端点，等待使用 pkg Infrastructure 重写

### 5. internal/service/ (部分删除)
**删除的文件：**
- `session.go` - 旧的 SessionService 实现（已由 pkg SessionService 替代）
- `store.go` - 内存存储实现（已由 pkg SessionService 替代）

**保留的文件：**
- `intent.go` - 意图识别服务（已重构为使用 pkg SessionService）
- `rostering.go` - 排班业务服务

## 保留的核心代码

### 1. domain/ 目录
**完整保留，已完成 pkg 迁移：**
- `model/` - 所有领域模型（session.go 使用类型别名映射 pkg 类型）
  - `session.go` - Session、Message、WorkflowMeta 等类型别名
  - `intent.go` - 意图相关模型
  - `schedule.go`, `shift.go`, `employee.go`, `department.go` 等业务模型
  
- `service/` - 服务接口定义
  - `session.go` - `type ISessionService = session.ISessionService`
  - `intent.go` - 意图服务接口
  - `rostering.go` - 排班服务接口
  - `provider.go` - 服务提供者接口

### 2. internal/ 目录
**保留的关键组件：**
- `wiring/container.go` - 依赖注入容器（已更新为使用 pkg Infrastructure）
- `service/intent.go` - 意图识别服务实现（已重构）
- `service/rostering.go` - 排班业务服务实现
- `workflow/README.md` - 工作流文档
- `workflow/schedule/SOP_CREATE.md` - **排班创建 SOP（重要参考）**
- `port/http/transport.go` - 临时 HTTP 传输层

### 3. 其他保留文件
- `setup.go` - 服务启动入口
- `go.mod`, `go.sum` - Go 模块定义
- `README.md` - 项目说明
- `config/` - 配置文件
- `doc/` - 所有文档

## 架构变更总结

### 旧架构（已删除）
```
旧架构层级：
domain/workflow/actor_system.go (Actor System 定义)
    ↓
internal/workflow/engine/ (自定义工作流引擎)
    ↓
internal/workflow/schedule/definition.go (工作流定义)
    ↓
internal/workflow/schedule/actions_create.go (具体 Action 实现)
    ↓
internal/service/session.go (自定义 SessionService + inMemoryStore)
    ↓
internal/port/ws/ (自定义 WebSocket 实现)
```

### 新架构（待实现）
```
新架构层级：
pkg/workflow/engine (标准工作流引擎)
    ↓
pkg/workflow/session (标准 SessionService)
    ↓
agents/rostering/internal/workflow/schedule/ (基于 pkg 的工作流定义)
    ↓
agents/rostering/internal/service/intent.go (使用 pkg SessionService)
    ↓
agents/rostering/internal/port/http/ (使用 infrastructure.HandleWebSocket)
```

### 关键变化
1. **Session 管理**：从自定义 inMemoryStore → pkg SessionService
2. **工作流引擎**：从自定义 Actor System → pkg/workflow/engine
3. **类型系统**：domain/model 使用类型别名直接映射 pkg 类型
4. **服务集成**：通过 Infrastructure 和 ServiceRegistry 统一管理
5. **WebSocket**：从自定义实现 → infrastructure.HandleWebSocket

## 编译验证
✅ 清理后代码可以成功编译：
```bash
go build -o .\bin\rostering.exe .\setup.go
```

## 下一步工作

### 1. 重新实现工作流（优先级：高）
参考 `internal/workflow/schedule/SOP_CREATE.md`，使用 pkg/workflow/engine 重新实现：
- 定义 WorkflowDefinition
- 实现 Transition 和 Action
- 使用 engine.Context 访问 SessionService 和业务服务
- 注册工作流：`engine.Register(&ScheduleCreateWorkflow{})`

### 2. 实现 HTTP/WebSocket 层（优先级：高）
- 使用 `infrastructure.HandleWebSocket` 提供 WebSocket 端点
- 实现会话创建、消息发送等 REST API
- 使用 `infrastructure.SendEvent` 触发工作流事件
- 实现状态查询、决策日志等查询端点

### 3. 集成测试（优先级：中）
- 测试会话创建和消息发送
- 测试意图识别
- 测试工作流状态转换
- 测试 WebSocket 实时推送
- 验证 pkg 设计的有效性

## 重要文档

### 必读文档
1. **`internal/workflow/schedule/SOP_CREATE.md`** - 排班创建标准操作流程
   - 详细的状态机定义
   - 每个状态的职责和转换条件
   - Action 实现要求
   
2. **`doc/workflow/README.md`** - 工作流架构文档
   
3. **`doc/workflow/plan/`** - 多意图执行计划文档
   - `PLAN_ARCHITECTURE.md` - 计划架构
   - `PLAN_QUICK_REFERENCE.md` - 快速参考
   - `PLAN_EXAMPLES.md` - 示例

### 业务逻辑文档
- `doc/features/` - 功能开发文档
- `doc/fixes/` - 问题修复记录
- `doc/migrations/` - 数据迁移记录

## 注意事项

1. **保留的 SOP 文档**：`SOP_CREATE.md` 是重新实现的关键参考，包含完整的状态机设计
2. **IntentService 已重构**：已经适配 pkg SessionService，可以直接使用
3. **类型系统已统一**：domain/model 使用类型别名，无需适配层
4. **临时 HTTP 实现**：当前只有健康检查端点，需要完整重写
5. **编译通过**：清理后的代码可以编译，但缺少运行时功能

## 验证 pkg 设计的机会
这次重构是验证 pkg/workflow 设计有效性的绝佳机会：
- 真实业务场景：复杂的排班工作流
- 完整生命周期：从会话创建到结果保存
- 多系统集成：AI、数据服务、图谱服务
- 实时交互：WebSocket 推送、状态同步

通过完整实现，可以验证：
- API 设计是否易用
- 扩展点是否充分
- 性能是否满足要求
- 架构是否清晰
