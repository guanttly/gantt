# FSM 工作流重构与对话承接说明

本说明详细介绍重构后的工作流架构（Engine + 多业务工作流），以及系统如何承接并理解一次“排班长”的长对话，全流程的数据流、事件流与状态流。

## 1. 目录结构与职责

```
internal/workflows/
├── fsm_actor.go              # 向后兼容层（旧API适配到新引擎）
├── engine/                   # 核心FSM引擎（通用、无业务）
│   ├── types.go             # 基础类型/接口（State/Event/Transition/Context/...）
│   ├── actor.go             # Actor事件循环、状态迁移、决策记录
│   ├── system.go            # ActorSystem、工作流注册表、查找/路由
│   ├── metrics.go           # 指标（Prometheus）与快照
│   └── persist.go           # 会话CAS更新工具
├── scheduling/               # 调度工作流（业务实现）
│   ├── definition.go        # 状态/事件/迁移注册
│   ├── actions_draft.go     # 草案生成（Step1~5）
│   ├── actions_finalize.go  # 确认生成（Finalize）
│   └── actions_adjust.go    # 草案/结果调整
├── ruleupdate/              # 规则更新工作流（示例）
│   ├── definition.go
│   └── actions.go
├── staffupdate/             # 人员更新工作流（示例）
│   ├── definition.go
│   └── actions.go
└── common/
        ├── reasons.go           # 失败原因常量（指标统一）
        └── actions_helper.go    # 通用动作按钮构建（WorkflowMeta.Actions）
```

核心设计目标：
- 引擎无业务、工作流可插拔；
- 会话（Session）为对话与状态的承载体；
- 工作流通过状态/事件/动作来驱动对话与生成；
- 按需扩展新工作流（例如规则更新/人员更新），无需改引擎。

## 2. 核心组件与契约

- Engine.State / Engine.Event：字符串别名，统一状态/事件标识。
- Engine.Transition：从某状态，在某事件下迁移到下一状态，可选 Guard（校验）与 Act（副作用）。
- Engine.Actor（单会话）：
    - 维护当前状态、处理事件（串行），记录决策（Decision），更新 SessionDTO.WorkflowMeta。
    - 通过 Context 提供依赖访问：Store、SessionService（消息）、ExecService（Finalize）、IAggregationPort（外部数据汇聚）、Metrics。
- Engine.System（多会话）：
    - 按 sessionID + workflowName 懒创建/路由 Actor；
    - 暴露 SendEvent / MetricsSnapshot / QueryDecisions 等；
    - 内置“终态回收 + 定时清理”，避免内存累积。
- WorkflowDefinition（业务工作流）：
    - Name、InitialState、Transitions、可选 OnActorInit/OnTerminal 钩子。
    - 在 `init()` 中注册（engine.Register）。

## 3. 排班工作流（scheduling）的状态/事件

状态：
- collecting → drafting → confirming → generating → completed/failed
- completed → post_adjust（可选）

事件：
- start、steps_done、user_confirm、user_cancel、timeout、draft_adjust、finalize_ok、finalize_failed、post_adjust

行为要点：
- intake（入口分流）：根据 payload.mode（create/modify）或上下文自动判断，路由到新建或修改路径；
- start_new / start_modify：分别触发新建/修改路径的异步 Step1~5，逐步写入 Context.Extra（步骤完成、候选、草案/改动建议等），最终在 confirming 汇合；
- draft_adjust：在 confirming 阶段对草案进行增/删/改，并递增 DraftVersion；
- post_adjust：在 completed 后可进行微调（备注/建议）。

### 重构：scheduling 同时支持“新建”与“修改”

为使工作流名“scheduling”覆盖“新建排班”和“修改现有排班”的两种意图，建议引入“入口分流 + 双路径汇合”的设计：

- 初始状态：intake（统一入口）
- 新建路径：collecting_new → drafting_new → confirming
- 修改路径：collecting_edit → drafting_edit → confirming
- 终态：confirming → generating → completed/failed（→ post_adjust 可选）

事件与守卫：
- intake：读取 payload 决定分流；若未指定 mode，按上下文（是否存在 targetScheduleId/日期范围）推断，否则回退到 create
- start_new：进入新建路径，异步 Step1~5 产出草案
- start_modify：进入修改路径，异步 Step1~5 基于既有排班 + changeSet 产出“改动草案”
- guardModifyTargetExists：校验 payload.targetScheduleId 或 dateRange 的有效性
- user_confirm / user_cancel / timeout / draft_adjust / finalize_ok / finalize_failed 与原语义一致

payload 契约（建议）：
```json
{
    "mode": "create | modify",
    "targetScheduleId": "可选，modify必需（或提供dateRange）",
    "dateRange": {"start": "2025-10-01", "end": "2025-10-07"},
    "changeSet": {
        "addShift": {"date": "2025-10-03", "staff": "x"},
        "removeShiftDates": ["2025-10-02"],
        "replace": {"date": "2025-10-01", "staff": "y"}
    },
    "expectedDraftVersion": 3
}
```

实现要点：
- actions_draft：新增 actIntake（或在现有 ActStart 中根据 mode 分支），拆分 actStartNew / actStartModify；
- actStartModify：
    - 加载目标排班（by targetScheduleId 或 dateRange）作为 baseline
    - 应用 changeSet 形成“改动草案”，标注 diff（新增/移除/替换）
    - 将步骤进度、草案与 diff 写入 Context.Extra 和 WorkflowMeta.Extra
    - 汇合到 confirming，并暴露适当的 Actions（Confirm/Cancel/Adjust）
- 守卫 guardModifyTargetExists：在 start_modify 前校验目标是否存在/可编辑

兼容策略：
- 保留旧事件 start（无 payload 或无 mode）作为“新建”的别名，内部转发为 start_new；
- 旧调用方无需改动即可保持新建语义；
- 新增“修改排班”入口按钮（UI）→ 发送 intake 或 start_modify，并携带 payload（mode=modify + targetScheduleId/dateRange + changeSet）。

迁移步骤（最小化改动）：
1) 在 scheduling/definition.go 增加状态 intake / collecting_edit / drafting_edit 与事件 start_new/start_modify；
2) 在 actions_draft.go 中抽取现有 start 为 actStartNew；新增 actStartModify 与（可选）actIntake；
3) 在 handler 将“创建排班”映射为 start_new（或沿用 start 以兼容）；“修改排班”映射为 start_modify，并传入 payload；
4) README 与 OpenAPI/前端事件映射文档同步更新。

## 4. “排班长对话”的承接与理解（端到端）

一次长对话由“消息 + 状态 + 动作”三条线并行承载：

1) 消息线（Message Log）
- 用户输入（自然语言）通过 ISessionService 进入 SessionDTO.Messages。
- 工作流动作（例如 Step 提示、确认提示）通过 ISessionService.AddAssistantMessage 写入“助手消息”，让用户感知流程进展（例如“第1步：获取最近排班数据…”、“草案已生成，等待确认”）。
- 若 ISessionService 为空（测试环境），动作里会用 logger 输出代替，保证流程不中断。

2) 状态线（Workflow State/Phase）
- Actor 驱动状态迁移，写入 SessionDTO.WorkflowMeta：
    - Workflow：工作流名（scheduling）
    - State：外部可见的大状态（源于 SessionDTO.State）
    - Phase：更细粒度的内部阶段（draft_confirming / generating / completed …）
    - Actions：当前可用的按钮（Confirm/Cancel/Adjust 等），由 `common/actions_helper.go` 生成
    - Extra：透传辅助信息（如 draftVersion、decisionLog 收敛版）

3) 动作线（Action Buttons → Events）
- 前端根据 WorkflowMeta.Actions 渲染按钮；
- 用户点按钮 → 网关/handler 将其映射为工作流事件（如 Confirm→user_confirm）；
- 可带 payload 参数（例如 expectedDraftVersion、adjust 参数），Actor 收到后执行 Guard/Act。

借此，系统能“理解并承接”一次多轮对话：
- 每条用户输入/系统提示都留痕（Messages）；
- 每一步业务推进都可追踪（决策日志 DecisionLog + WorkflowMeta.Extra.decisionLog 最近50条摘要）；
- 任意时刻的可执行动作与所处阶段明确（Actions + Phase）；
- 草案多轮调整与版本一致性通过 Guard（expectedDraftVersion）保障；
- 超时/取消/失败都有统一的终态归类与指标收集（Finalization Fail Reasons）。

### 多轮草案调整（payload 约定）
在 confirming 阶段，支持以下调整参数（示例）：

```json
{
    "addShift": {"date": "2025-10-03", "staff": "x"},
    "removeShiftDates": ["2025-10-02"],
    "replace": {"date": "2025-10-01", "staff": "y"},
    "expectedDraftVersion": 3
}
```

注意 expectedDraftVersion 用于与后端草案版本对齐，避免误确认（Guard 不通过则不迁移）。

## 5. 生命周期与内存管理（避免累积）

- Actor 决策日志内存上限：在内存仅保留最近 1000 条（可按需下调，例如 200/500）；写入 WorkflowMeta.Extra.decisionLog 仅保留最近 50 条摘要用于前端快速查看。
- 终态回收：
    - Actor 进入 completed/failed 会标记 terminal；
    - System.SendEvent 后若检测到 terminal 会立即回收（Stop + 从 map 删除）；
    - 同时还有后台清理协程（janitorLoop，每1分钟扫描一次），确保终态 Actor 不会长期占用内存；
    - Stop 时会对 metrics.state_active 做递减，避免指标长时间堆积。

## 6. 扩展新工作流（规则更新/人员更新）

新增一个工作流只需：
1) 在子目录创建 `definition.go`，定义状态/事件/迁移，`init()` 中 `engine.Register(def)`；
2) 编写 `actions.go` 实现各迁移副作用（通过 Context 访问 Store/ISessionService/IExecService 等）；
3) handler 中发送事件时指定 workflowName（如 "rule_update"）。

无需修改 Engine 或其他工作流的代码。耦合度低、扩展快。

## 7. 关键接口与使用方式

### 7.1 创建 ActorSystem（兼容旧API）

旧代码：
- `workflows.NewActorSystem(logger, store, sessionSvc, agg, exec)` 返回向后兼容包装类型；
- `as.Send(ctx, sessionID, workflows.FSMEventStart(), nil)` 仍然可用（默认 workflowName="scheduling"）。

新代码（直连引擎）：
- `engine.NewSystem(logger, store, sessionSvc, execSvc, aggPort)`；
- `system.SendEvent(ctx, sessionID, "scheduling", scheduling.EventStart, payload)`。

### 7.2 决策日志查询

- `system.QueryDecisions(sessionID, opts)` 返回过滤后的决策与总数；
- 支持按 Event/From/To/Since/Offset/Limit/Reverse 过滤与分页；
- 前端快速显示使用 WorkflowMeta.Extra.decisionLog（最近 50 条），需要完整或更多条目时再调 QueryDecisions。

### 7.3 指标

- `system.MetricsSnapshot()` 返回当前快照（transitionCounts/stateActive/stateDwell/workflowBuckets）；
- `system.PrometheusRegistry()` 可挂入 HTTP /metrics；
- Finalize 失败会按原因分类计数（common/reasons.go）。

## 8. 与对话入口（HTTP/Handler）的连接

在例如 `internal/port/handlers/scheduling.go` 内：
- 用户自然语言消息 → `ISessionService.SendSessionMessage`（或等价链路），用于消息面板存储；
- 用户点击“生成排班/确认/调整/取消”等按钮 → 将按钮 Command 映射为工作流事件，构造 payload（如 expectedDraftVersion / 调整参数），调用 `ActorSystem.Send`（旧）或 `System.SendEvent`（新）分发给对应 session 的 Actor；
- Actor 在动作过程中通过 `ISessionService.AddAssistantMessage` 推送系统/助手提示，从而与长对话流无缝衔接。

## 9. 典型时序（调度）

1) 用户：发起排班会话；
2) System：创建 Actor（scheduling，state=collecting）；
3) 事件 start：
     - 异步执行 Step1~5，写入 Context.Extra（步骤与草案）；
     - 进入 confirming，同时设置 WorkflowMeta.Actions（Confirm/Cancel/Adjust）；
4) 用户：
     - 选择“调整草案”（可多轮，递增草案版本），或
     - 选择“确认生成”（带 expectedDraftVersion）；
5) 事件 user_confirm：
     - IExecService.FinalizeSession → 进入 completed 或 failed；
     - 完成后可 post_adjust；
6) 回收：State 到达终态后 Actor 被回收，避免内存累积。

## 10. 常见问题与边界

- Guard 不通过（如预期版本与后端不一致）：不迁移状态，保持在 confirming。
- 超时（timeout）：进入 failed 并清空 Actions；
- IExecService 不可用：finalize_failed，失败原因归为 `internal_missing_dependency`；
- 无 ISessionService（测试环境）：动作仍可执行但消息仅记录日志。

---

通过以上机制，系统可清晰地承接并理解排班长的整段对话：
- 所有对话上下文（消息/状态/动作/版本/草案内容）统一承载在 SessionDTO 与 WorkflowMeta 中；
- 业务推进通过工作流事件驱动，UI 按钮即事件入口；
- 完整的可追踪性（DecisionLog + Metrics）；
- 可插拔的工作流定义，便于未来扩展更多“对话+流程”场景。