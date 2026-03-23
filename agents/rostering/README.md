# Scheduling Agent（排班智能体）

基于 DDD + FSM 的排班智能体，集成 AI 意图识别、V4 规则引擎、工作流引擎与 MCP 协议。核心理念：**LLM 只做"从合格候选人中选人"这一个决策，其余全部代码化。**

---

## 目录

- [架构概览](#架构概览)
- [V4 排班系统设计](#v4-排班系统设计)
- [V4 规则配置管理](#v4-规则配置管理)
- [FSM 工作流引擎](#fsm-工作流引擎)
- [Plan 多意图执行框架](#plan-多意图执行框架)
- [规则工作流](#规则工作流)
- [班次类型分类](#班次类型分类)
- [WebSocket 消息协议](#websocket-消息协议)
- [开发指南](#开发指南)


---

## 架构概览

```
agents/rostering/
├── config/          # 配置模型与加载器
├── domain/
│   ├── model/       # 排班/人员/会话/意图/规则 等领域模型
│   ├── repository/  # 仓储接口
│   ├── service/     # 领域服务接口（data/session/intent/rule/ai）
│   ├── workflow/    # 工作流事件/状态常量定义
│   └── port/        # 外部依赖抽象接口
├── internal/
│   ├── engine/      # ★ V4 确定性规则引擎
│   ├── service/     # 应用服务实现（意图识别/排班AI/聚合等）
│   ├── workflow/    # FSM 工作流引擎 + 各业务工作流
│   │   ├── engine/       # 核心 Actor/System/Metrics
│   │   ├── common/       # Plan 多意图执行框架
│   │   ├── schedule/     # 排班工作流（V1→已废弃）
│   │   ├── schedule_v2/  # 排班工作流 V2（已废弃）
│   │   ├── schedule_v3/  # 排班工作流 V3（渐进式，当前生产）
│   │   ├── schedule_v4/  # 排班工作流 V4（引擎化，迭代中）
│   │   ├── rule/         # 规则管理工作流
│   │   └── state/        # 跨工作流状态常量
│   ├── infrastructure/   # MCP 网关/仓储实现
│   └── wiring/           # 依赖注入
├── setup.go         # 服务启动入口
└── test/            # 集成测试
```

**依赖服务（MCP 协议）**：`data-server` | `relational-graph-server` | `context-server`

---

## V4 排班系统设计

### V3 vs V4 核心对比

| 维度 | V3（当前生产） | V4（迭代中） |
|------|-------------|------------|
| 规则执行 | LLM 理解自然语言规则 | 代码精确执行结构化规则 |
| 候选人过滤 | LLM-1（人员过滤） | `CandidateFilter`（代码化） |
| 规则匹配 | LLM-2（规则过滤） | `RuleMatcher`（代码化） |
| 冲突检测 | LLM-3（冲突检测） | `ConstraintChecker`（代码化） |
| 排班校验 | LLM-5（校验） | `ScheduleValidator`（代码化） |
| LLM 职责 | 理解规则+过滤人员+排班+校验 | **仅排班选人** |
| 单次LLM调用 | ~720 次（20班×7天） | ~160 次（节省 78%） |
| 班次顺序 | `SchedulingPriority` 整数 | 拓扑排序（ShiftDependency DAG） |
| 规则模型 | 纯文本 `RuleData` | 结构化分类 + 方向性关联（Role） |

### V4 规则分类体系

```
Rule.Category:
  constraint（约束型，必须遵守）
    ├── forbid   — exclusive / forbidden_day
    ├── limit    — maxCount / consecutiveMax / minRestDays
    └── must     — required_together / periodic
  preference（偏好型，尽量满足）
    ├── prefer   — preferred
    ├── suggest
    └── combinable
  dependency（依赖型，定义执行顺序）
    ├── source   — 人员必须来自前一日某班次
    ├── resource — 当日班次人员需保留给次日
    └── order    — 规则执行顺序依赖

RuleAssociation.Role:
  target    — 被约束对象（默认，向后兼容V3）
  source    — 数据来源（依赖型规则）
  reference — 引用对象（排他规则）
```

**示例**：`"下夜班人员必须来自前一日上半夜班"` → category=dependency, subCategory=source,
`associations=[{shiftId: "下夜班", role: "target"}, {shiftId: "上半夜班", role: "source"}]`

### V4 规则引擎（`internal/engine/`）

```
IRuleEngine
├── PrepareSchedulingContext(input)  → SchedulingContext  # 替代 LLM-1/2/3
│   ├── CandidateFilter      — 根据请假/占位/固定排班过滤候选人
│   ├── RuleMatcher          — 按班次ID+关联关系精确匹配规则
│   ├── ConstraintChecker    — 检查每位候选人的约束违反情况
│   └── PreferenceScorer     — 对候选人偏好打分
│
├── ValidateSchedule(result, rules)  → ValidationResult   # 替代 LLM-5
└── ValidateGlobal(draft, allRules)  → GlobalValidationResult

IDependencyResolver
├── ResolveShiftOrder(shifts, deps)  → []string  # 拓扑排序，确定班次执行顺序
└── DetectCircularDependency(edges)  → [][]string
```

**设计约束**：零 LLM 调用；所有方法确定性（相同输入→相同输出）；100规则×50人×7天 < 500ms。

### V4 工作流上下文三层结构

```
L1: CreateV4Context（全局）— AllStaff / AllShifts / AllRules / RuleOrganization / ShiftExecutionOrder
  └─深拷贝→ L2: TaskExecutionContextV4（任务）— TargetShifts / MatchedRules / ShiftDependencies
               └─拆分→ L3: ShiftTaskContextV4（单班次）— FilteredRules / LLMBrief / RuleEngine输出
```

`LLMBrief` 是规则引擎生成的结构化 Prompt 摘要，包含：合格候选人列表（含偏好分）、必须遵守约束、排除人员+原因。LLM 只需从合格候选人中做选择。

### V4 数据库变更

```sql
ALTER TABLE scheduling_rules ADD COLUMN category VARCHAR(32) DEFAULT '';
ALTER TABLE scheduling_rules ADD COLUMN sub_category VARCHAR(32) DEFAULT '';
ALTER TABLE scheduling_rules ADD COLUMN original_rule_id VARCHAR(64) DEFAULT '';
ALTER TABLE rule_associations ADD COLUMN role VARCHAR(32) DEFAULT 'target';

CREATE TABLE rule_dependencies (id, org_id, dependent_rule_id, depends_on_rule_id, dependency_type, ...);
CREATE TABLE rule_conflicts    (id, org_id, rule_id_1, rule_id_2, conflict_type, resolution_priority, ...);
CREATE TABLE shift_dependencies(id, org_id, dependent_shift_id, depends_on_shift_id, dependency_type, rule_id, ...);
```

---

## V4 规则配置管理

### 核心流程

```
用户输入自然语言规则
  → POST /v1/rules/parse → LLM 解析（仅此一次）→ 三层验证
  → 前端展示解析结果 + 回译文本 → 人工审核/编辑
  → POST /v1/scheduling-rules（保存结构化规则）
  → 排班引擎直接使用结构化数据（不再重复调用LLM）
```

### 三层验证

1. **结构化验证** — 必填字段/枚举合法性/关联对象存在性
2. **回译验证** — LLM 将结构化结果反向生成自然语言，与原文比对（相似度 > 0.85）
3. **模拟验证** — 对样本数据执行规则引擎，验证规则行为符合预期

### V4 新增 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/rules/parse` | 自然语言解析为结构化规则 |
| POST | `/v1/rules/parse/batch` | 批量解析 |
| POST | `/v1/rules/validate` | 三层验证 |
| GET/POST/DELETE | `/v1/rules/{id}/dependencies` | 规则依赖 CRUD |
| GET/POST/DELETE | `/v1/rules/{id}/conflicts` | 规则冲突 CRUD |
| GET/POST/DELETE | `/v1/shifts/dependencies` | 班次依赖 CRUD |
| GET | `/v1/rules/organization` | 规则组织视图 |

### 前后端枚举统一（V4）

| 语义 | V3前端 | V4统一值（后端） |
|------|-------|--------------|
| 最大班次数 | `max_shifts` | `maxCount` |
| 禁止模式 | `forbidden_pattern` | `exclusive` |
| 偏好模式 | `preferred_pattern` | `preferred` |
| 日维度 | `daily` | `same_day` |
| 周维度 | `weekly` | `same_week` |

### V3→V4 迁移策略

- **阶段1（自动推断）**：按 `rule_type` 推断 `category/sub_category`（100% 覆盖，SQL 脚本执行）
- **阶段2（LLM辅助）**：解析 `rule_data` 自然语言，填充更精确分类+关联Role（可选）
- **阶段3（人工审核）**：管理员逐条确认
- V3 数据不修改任何现有字段，新增字段全部有默认值，迁移可重复执行、可回滚

**V3 兼容代码隔离规范**：所有 V3 兼容层集中在独立文件（`v3_compat.go` / `v3-compat.ts`），
标记 `// @deprecated V3`，全量迁移完成后一键删除（`scripts/cleanup_v3_compat.sh`）。

---

## FSM 工作流引擎

### 架构

```
pkg/workflow/engine/
├── types.go     # State / Event / Transition / Context 基础类型
├── actor.go     # Actor：单会话事件循环 + 状态迁移 + 决策日志
├── system.go    # ActorSystem：多会话路由 + 懒创建 + 终态回收
├── metrics.go   # Prometheus 指标（transitionCounts / stateActive / stateDwell）
└── persist.go   # 会话 CAS 更新工具

agents/rostering/internal/workflow/
├── fsm_actor.go           # 向后兼容层（旧 API 适配到新引擎）
├── engine/                # 核心引擎实例
├── common/                # Plan 多意图框架
├── schedule_v3/           # 排班创建 V3（当前生产）
├── schedule_v4/           # 排班创建 V4（迭代中）
└── rule/                  # 规则管理工作流
```

### 排班工作流 V3 状态机

```
v3.init
  → v3.info_collecting (收集排班周期/班次/人数)
  → v3.personal_needs  (收集个人需求)
  → v3.plan_generation → v3.plan_review
  → v3.progressive_task (渐进式排班，每班次×每日循环)
      内部：规则预分析(LLM-1/2/3并行) → 排班(LLM-4) → 校验(LLM-5) → 重试
  → v3.global_review   (全局评审)
  → v3.save_draft / v3.complete
```

V4 状态名前缀改为 `v4.*`，工作流名 `schedule.create.v4`，
新增 `v4.rule_validation` 状态（规则解析验证步骤）。

### Actor 核心机制

- **串行处理**：每个 Session 对应一个 Actor，事件串行处理，无并发问题
- **决策日志**：内存保留最近 1000 条，`WorkflowMeta.Extra.decisionLog` 保留最近 50 条摘要
- **终态回收**：Actor 进入 completed/failed 后立即标记 terminal，janitorLoop 每分钟清理
- **CAS 保护**：草案更新通过 `expectedDraftVersion` 防止版本不一致

### 排班工作流支持修改模式

```
intake（统一入口，按 payload.mode 分流）
  ├─ start_new → collecting_new → drafting_new → confirming
  └─ start_modify → collecting_edit → drafting_edit → confirming（加载 baseline + 应用 changeSet）
```

```json
// 修改排班 payload 示例
{
  "mode": "modify",
  "targetScheduleId": "sch-001",
  "changeSet": { "addShift": {"date":"2025-10-03","staff":"x"}, "removeShiftDates":["2025-10-02"] },
  "expectedDraftVersion": 3
}
```

---

## Plan 多意图执行框架

支持一次用户输入触发多个操作的顺序执行。

### 标准执行流程

```
Idle → IntentRecognizing → PlanGenerating → PlanConfirming → Executing → Completed/Failed
```

### 实现步骤

```go
// 1. 实现 IntentExecutor
type MyExecutor struct{}
func (e *MyExecutor) ExecuteIntent(ctx, dto, wfCtx, intent) error {
    switch intent.Type {
    case IntentMyAction: return wfCtx.Send(ctx, Event_My_Action, intent)
    }
    return nil
}

// 2. 配置并注册
config := common.PlanExecutorConfig{
    IntentExecutor: &MyExecutor{},
    EventExecuteNext: Event_My_ExecuteNext,
    // ... 其他事件/状态映射
}
builder := common.NewPlanTransitionBuilder(config)
transitions := builder.BuildPlanTransitions(State_Idle, State_Recognizing, ...)
```

---

## 规则工作流

`internal/workflow/rule/` — 通过 MCP relational-graph-server 管理规则。

| 子工作流 | 状态流转 | MCP 工具 |
|---------|---------|---------|
| Extract（提取） | Idle→Extracting→Checking→Confirming→Executing→Completed | `scheduling_rules_upsert` |
| Query（查询） | Idle→Querying→Completed | `scheduling_query` |
| Delete（删除） | Idle→Checking→Confirming→Executing→Completed | `scheduling_rules_delete` |

支持多意图（通过 Plan 框架）：`rule.create` / `rule.update` / `rule.query` / `rule.delete`

> **TODO**：`getRelGraphGateway` 当前返回 nil，需注入真实 Gateway（推荐方案：在 `IWFContext` 接口新增 `IRelationalGraphGateway()` 方法）。

---

## 班次类型分类

### 用户可创建类型（前端表单选择）

| 代码 | 名称 | 说明 |
|------|------|------|
| `regular` | 常规班次 | 日常工作班次（白班/夜班等） |
| `overtime` | 加班班次 | 节假日或额外工作 |
| `standby` | 备班班次 | 待命/应急班次 |

### 系统/工作流类型（只读）

| 代码 | 名称 | 来源 | 排班优先级 |
|------|------|------|----------|
| `normal` | 普通班次 | `regular` 映射 | — |
| `special` | 特殊班次 | `overtime/standby` 映射 | — |
| `fixed` | 固定班次 | 固定人员配置 | 10（最高） |
| `research` | 科研班次 | 科研学习 | 70 |
| `fill` | 填充班次 | 补充排班不足 | 90（最低） |

---

## WebSocket 消息协议

### 客户端 → 服务端

| 消息类型 | 说明 |
|---------|------|
| `user_message` | 自然语言输入（`{content: "生成10月班表"}`） |
| `workflow_command` | 工作流操作（`{command: "confirm|cancel|draft_adjust"}`） |
| `context_collect` | 触发上下文聚合（`{force: true}`） |
| `finalize` | 最终生成排班 |
| `fetch_decisions` | 分页拉取 FSM 决策日志 |
| `fetch_snapshot` | 获取会话快照 |
| `ping` | 心跳 |

### 服务端 → 客户端

| 消息类型 | 说明 |
|---------|------|
| `assistant_message` | AI 助手消息 |
| `session_updated` | SessionDTO 变更 |
| `workflow_update` | 阶段+可用按钮更新（`{state, phase, actions[]}`） |
| `decision_log_append` | FSM 决策记录追加 |
| `context_update` | 上下文数据更新 |
| `validation_result` | 规则校验结果（V4） |
| `finalize_completed` | 生成完成（含排班结果） |
| `error` | 错误（含 `retryable` 标志） |

### 典型交互序列

```
1. 用户发送 user_message: "生成10月放射科班表"
2. 服务端发送 workflow_update: {phase: "Collecting", actions: [{command:"confirm"}]}
3. 用户发送 workflow_command: {command: "confirm"}
4. 服务端异步执行，推送 assistant_message 进度
5. 用户发送 finalize
6. 服务端发送 finalize_completed
```

---

## 开发指南

### 新增工作流

```go
// 1. 创建 myworkflow/definition.go
func init() {
    engine.Register(WorkflowDefinition{
        Name: "my_workflow",
        InitialState: StateIdle,
        Transitions: []Transition{...},
    })
}

// 2. 编写 myworkflow/actions.go
// 3. handler 中发送事件
system.SendEvent(ctx, sessionID, "my_workflow", EventStart, payload)
```

### IntentService 说明

`IntentService` 仅负责**初始意图识别**（用户首次输入），始终使用 `initial` 策略，代码约 300 行。
工作流内的意图分析由 `ISchedulingAIService.AnalyzeAdjustIntent()` 负责（关注点分离）。

### 配置示例

```yaml
# config/agents/rostering-agent.yml
scheduling:
  version: "v3"        # "v3" | "v4"
  v4_enabled: false    # V4 功能总开关
  v4_orgs: []          # 白名单组织ID（灰度）
```

### 运行

```bash
go run ./setup.go                           # 直接运行
# 或从上层 cmd 入口
go run ../../cmd/agents/rostering/main.go
```
