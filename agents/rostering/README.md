# Scheduling Service

基于 DDD（Domain-Driven Design）的智能排班管理服务，集成 AI 意图识别、工作流引擎与 MCP（Model Context Protocol）协议，通过与 data-server、relational-graph-server、context-server 对接，提供完整的排班业务支撑与智能交互能力。

## ✨ 核心特性

- 🤖 **AI 驱动的意图识别**：支持自然语言交互，自动识别排班创建、调整、查询等多种意图
- 🔄 **工作流引擎**：基于 FSM（有限状态机）的工作流系统，支持多意图顺序执行与状态追踪
- 🌐 **MCP 协议集成**：通过 MCP 与多个后端服务无缝对接，支持超时重试、健康检查
- 📊 **知识图谱集成**：与 relational-graph-server 集成，支持排班规则管理与冲突检测
- 💬 **实时 WebSocket 交互**：提供 WS 端口实现推拉结合的实时消息推送
- 📦 **上下文聚合**：自动收集历史排班、人员档案等上下文数据，支持向量化存储
- 🎯 **智能人员预选**：基于规则、技能、历史数据的智能人员筛选与推荐
- 🔧 **DDD 架构**：清晰的领域层、应用层、基础设施层分离，易于维护和扩展

## 🏗️ 架构概览

本服务采用 DDD（领域驱动设计）分层架构，通过 MCP 协议与多个后端服务集成。

```
scheduling-service/
├── cmd/                           # 可执行入口（上层 app）
│   └── services/scheduling-service/
│       └── main.go               # 主程序入口
├── config/                        # 配置管理
│   ├── config.go                  # 配置模型与加载器
│   └── config.example.yml         # 配置示例文件
├── domain/                        # 领域层（核心业务逻辑）
│   ├── model/                     # 领域模型与值对象
│   │   ├── schedule.go           # 排班实体与值对象
│   │   ├── staff.go              # 人员实体
│   │   ├── session.go            # 会话模型
│   │   ├── intent.go             # 意图识别模型
│   │   └── ...
│   ├── repository/                # 领域仓储接口（数据访问抽象）
│   │   ├── schedule_repository.go
│   │   └── staffing_repository.go
│   ├── service/                   # 领域服务接口
│   │   ├── data_service.go        # 数据访问服务（排班、人员等）
│   │   ├── session.go             # 会话管理服务
│   │   ├── intent_service.go      # 意图识别服务
│   │   ├── rule.go                # 规则管理服务
│   │   └── aggregation.go         # 上下文聚合服务
│   ├── workflow/                  # 工作流领域模型
│   │   ├── actor_system.go        # Actor 系统接口定义
│   │   ├── event.go               # 工作流事件定义
│   │   └── event_*.go             # 各业务工作流事件
│   └── port/                      # 端口接口（外部依赖抽象）
├── internal/                      # 内部实现（应用层与基础设施层）
│   ├── infrastructure/            # 基础设施层
│   │   ├── mcp/                   # MCP 集成基础设施
│   │   │   ├── toolbus.go         # MCP 工具总线（重试/超时/健康检查）
│   │   │   ├── dataserver_gateway.go   # Data-Server 网关
│   │   │   ├── relationalgraph_gateway.go # Relational-Graph-Server 网关
│   │   │   ├── dataserver_types.go      # Data-Server DTO 定义
│   │   │   └── relationalgraph_types.go # Relational-Graph-Server DTO 定义
│   │   └── repository/            # 仓储实现（调用 MCP 网关）
│   │       ├── schedule_repository_impl.go
│   │       └── staffing_repository_impl.go
│   ├── context/                   # Context-Server 网关
│   │   └── context_gateway.go     # 上下文服务网关实现
│   ├── service/                   # 应用服务实现
│   │   ├── data_service.go        # 数据服务实现
│   │   ├── session_service.go     # 会话管理实现
│   │   ├── intent_service.go      # 意图识别实现（AI 调用）
│   │   ├── rule_service.go        # 规则管理实现
│   │   ├── aggregation_service.go # 上下文聚合实现
│   │   ├── scheduling_ai_service.go # 排班 AI 服务（人员预选、草案生成）
│   │   └── store.go               # 内存会话存储
│   ├── workflow/                  # 工作流引擎与执行器
│   │   ├── fsm_actor.go           # FSM Actor 核心实现
│   │   ├── engine/                # 工作流引擎
│   │   │   ├── actor.go           # Actor 实现
│   │   │   ├── system.go          # Actor 系统
│   │   │   └── message_builder.go # 消息构建器
│   │   ├── common/                # 通用工作流组件
│   │   │   ├── plan_executor.go   # 通用执行计划框架
│   │   │   ├── plan_actions.go    # 计划执行动作
│   │   │   └── reasons.go         # 失败原因常量
│   │   ├── schedule/              # 排班工作流定义
│   │   │   ├── definition.go      # 排班工作流状态机定义
│   │   │   └── actions_*.go       # 各阶段动作实现
│   │   ├── rule/                  # 规则管理工作流
│   │   │   ├── definition.go
│   │   │   ├── plan_executor_impl.go # 多意图执行器
│   │   │   └── actions_*.go
│   │   ├── dept/                  # 部门管理工作流
│   │   │   ├── definition.go
│   │   │   ├── plan_executor_impl.go
│   │   │   └── actions_*.go
│   │   └── general/               # 通用工作流（帮助、闲聊）
│   ├── port/                      # 适配器层（外部接口）
│   │   ├── http/                  # HTTP 传输层
│   │   │   └── transport.go       # HTTP 路由与处理器注册
│   │   ├── handlers/              # HTTP 处理器
│   │   │   └── scheduling.go      # 排班相关 API 端点
│   │   └── ws/                    # WebSocket 端点
│   │       ├── server.go          # WS 服务器
│   │       ├── hub.go             # WS 消息广播中心
│   │       ├── messages.go        # WS 消息类型定义
│   │       ├── observer.go        # 工作流观察者（推送工作流事件）
│   │       └── publisher.go       # 事件发布器
│   ├── wiring/                    # 依赖注入容器
│   │   └── container.go           # 统一装配所有依赖
│   └── task/                      # 任务调度（如有）
├── setup.go                       # 服务装配与启动入口
├── go.mod / go.sum                # Go 模块依赖
├── doc/                           # 详细文档
│   ├── fsm-guide.md               # ⭐ FSM 工作流引擎完全指南
│   ├── workflow/                  # 工作流文档
│   │   ├── README.md              # FSM 工作流重构说明
│   │   └── scheduling-interaction.md # ⭐ 排班前后端交互示例
│   ├── ai-staff-preselection.md   # AI 人员预选文档
│   ├── schedule_draft_generation.md # 排班草案生成文档
│   ├── user-supplementary-implementation.md # 用户补充信息实现
│   └── error-codes.md             # 错误码说明
└── test/                          # 测试
    └── unit/                      # 单元测试
```

## 🔧 MCP 集成架构

本服务通过 MCP（Model Context Protocol）与多个后端服务集成，形成完整的数据与知识服务能力。

### 集成的 MCP 服务

#### 1. Data-Server（数据服务）
提供排班、人员、团队等核心数据的 CRUD 操作。

**已集成工具**：
- **排班管理**
  - `schedule_manager.query_schedules` - 查询排班
  - `schedule_manager.upsert_schedule` - 创建/更新排班
  - `schedule_manager.batch_upsert_schedules` - 批量更新排班
  - `schedule_manager.delete_schedule` - 删除排班
  - `schedule_manager.ensure_db_configured` - 确保数据库配置

- **人员管理**
  - `staff_manager.list_staff` - 列出人员
  - `staff_manager.search_staff` - 搜索人员
  - `staff_manager.check_staff_exists` - 检查人员是否存在
  - `staff_manager.create_staff` - 创建人员
  - `staff_manager.get_eligible_staff` - 获取符合条件的人员

- **团队与班次管理**
  - `staffing_manager.list_teams` - 列出团队
  - `staffing_manager.create_team` - 创建团队
  - `staffing_manager.update_team` - 更新团队
  - `staffing_manager.assign_team_members` - 分配团队成员
  - `staffing_manager.list_shifts` - 列出班次
  - `staffing_manager.create_shift` - 创建班次
  - `staffing_manager.get_teams_by_shift_id` - 获取班次关联团队（支持向上递归）

- **请假管理**
  - `staffing_manager.create_leave` - 创建请假记录
  - `staffing_manager.get_leave_records` - 获取请假记录

- **数据库管理**
  - `staffing_manager.auto_migrate` - 自动迁移数据库结构
  - `staffing_manager.list_staff_classifications` - 列出人员分类
  - `staffing_manager.list_staff_roles` - 列出人员角色

#### 2. Relational-Graph-Server（关系图谱服务）
提供排班规则管理、知识图谱查询与冲突检测。

**已集成工具**：
- `scheduling_rules_upsert` - 创建/更新排班规则（支持自然语言）
- `scheduling_rules_delete` - 删除排班规则
- `scheduling_rules_list` - 列出排班规则
- `scheduling_query` - 查询排班知识图谱
- `neo4j_query` - 执行 Neo4j Cypher 查询

#### 3. Context-Server（上下文服务）
提供会话上下文管理、历史数据向量化存储与检索。

**功能**：
- 批量文档向量化与存储
- 上下文检索（基于向量相似度）
- 历史数据归集与聚合
- 自动嵌入（auto_embed）支持

### MCP 基础设施组件

#### IToolBus（工具总线）
统一的 MCP 工具调用接口，提供：
- ✅ 超时控制（可配置默认超时）
- ✅ 自动重试（指数退避）
- ✅ 健康检查
- ✅ 错误处理与降级

**配置示例**：
```yaml
mcp:
  agentServiceUrl: "http://localhost:8090"
  dataServerTimeout: 30000  # ms
  retryCount: 3
  retryDelayMs: 1000
```

#### IDataServerGateway（数据服务网关）
基于 IToolBus 封装 data-server 工具调用：
- 工具名称映射与常量管理
- 请求/响应 DTO 序列化/反序列化
- 类型安全的方法调用

#### IRelationalGraphGateway（图谱服务网关）
基于 IToolBus 封装 relational-graph-server 工具调用：
- 规则 CRUD 操作封装
- 图谱查询结果解析
- Neo4j 查询执行

#### IContextGateway（上下文服务网关）
基于 MCP 封装 context-server 调用：
- 批量文档向量化
- 上下文检索与聚合
- 自动嵌入配置

## 🧩 核心服务接口

### IDataService（数据服务）
位置：`domain/service/data_service.go`

统一封装对 data-server 的 MCP 调用，提供排班业务所需的数据访问能力。

```go
type IDataService interface {
    // 排班数据服务
    QuerySchedules(ctx context.Context, filter d_model.ScheduleQueryFilter) (*d_model.ScheduleQueryResult, error)
    UpsertSchedule(ctx context.Context, req d_model.ScheduleUpsertRequest) (*d_model.ScheduleEntry, error)
    BatchUpsertSchedules(ctx context.Context, batch d_model.ScheduleBatch) (*d_model.BatchUpsertResult, error)
    DeleteSchedule(ctx context.Context, userID, date string) error

    // 人员数据服务
    GetStaffProfiles(ctx context.Context, orgID, department, modality string) ([]d_model.Staff, error)
    ListStaff(ctx context.Context, filter d_model.StaffListFilter) (*d_model.StaffListResult, error)
    SearchStaff(ctx context.Context, filter d_model.StaffSearchFilter) (*d_model.StaffListResult, error)
    CheckStaffExists(ctx context.Context, orgID, name string) (bool, error)
    GetStaff(ctx context.Context, userID string) (*d_model.Staff, error)
    CreateStaff(ctx context.Context, req d_model.StaffCreateRequest) (string, error)

    // 团队与班次管理
    ListTeams(ctx context.Context, orgID string) ([]d_model.Team, error)
    CreateTeam(ctx context.Context, req d_model.CreateTeamRequest) (string, error)
    UpdateTeam(ctx context.Context, req d_model.UpdateTeamRequest) error
    AssignTeamMembers(ctx context.Context, req d_model.TeamMemberAssignRequest) error
    ListShifts(ctx context.Context, orgID string, teamID string) ([]d_model.Shift, error)
    GetTeamsByShiftID(ctx context.Context, orgID, shiftID string, withFallback bool) ([]d_model.Team, error)

    // 请假数据服务
    CreateLeave(ctx context.Context, req d_model.LeaveCreateRequest) (string, error)
    GetLeaveRecords(ctx context.Context, staffID string, startDate, endDate string) ([]d_model.LeaveRecord, error)

    // 数据库管理服务
    EnsureScheduleDBConfigured() error
    AutoMigrateStaffing(ctx context.Context) error
}
```

### ISessionService（会话管理服务）
位置：`domain/service/session.go`

负责多轮对话会话的生命周期管理。

```go
type ISessionService interface {
    StartSession(ctx context.Context, req *d_model.StartSessionRequest) (*d_model.SessionDTO, error)
    SendSessionMessage(ctx context.Context, sessionID string, req *d_model.SessionMessageRequest) (*d_model.SessionDTO, error)
    GetSession(ctx context.Context, sessionID string) (*d_model.SessionDTO, error)
    ListSessions(ctx context.Context, states ...string) ([]*d_model.SessionDTO, error)
    AddAssistantMessage(ctx context.Context, sessionID string, content string) (*d_model.SessionDTO, error)
    FinalizeSession(ctx context.Context, sessionID string) (*d_model.SessionDTO, error)
    UpdateContextExtra(ctx context.Context, sessionID string, values map[string]any) (*d_model.SessionDTO, error)
    GetSchedulePreview(ctx context.Context, sessionID string) (map[string]any, error)
}
```

### IIntentService（意图识别服务）
位置：`domain/service/intent_service.go`

基于 AI 的自然语言意图识别与解析。

```go
type IIntentService interface {
    // 检测并标注意图（主意图识别）
    DetectAndAnnotate(ctx context.Context, sessionID string) (*d_model.SessionDTO, *d_model.IntentDetectionPayload, error)
    
    // 分析二级意图
    AnalyzeSubIntents(ctx context.Context, sessionID string, main d_model.IntentType) ([]*d_model.IntentResult, error)
    
    // 解析补充信息（用户补充缺失字段时）
    ParseSupplementaryInfo(ctx context.Context, sessionID string, missingFields map[string]string) (*d_model.SupplementaryInfoResult, error)
    
    // 执行 AI 人员预选
    PreSelectStaff(ctx context.Context, sessionID string, staffList []d_model.Staff, rules string, history string) (*d_model.StaffPreSelectionResult, error)
    
    // 生成排班草案
    GenerateScheduleDraft(ctx context.Context, sessionID string, params d_model.DraftGenerationParams) (*d_model.DraftGenerationResult, error)
}
```

### IRuleService（规则管理服务）
位置：`domain/service/rule.go`

负责排班规则的提取、确认、校验与存储。

```go
type IRuleService interface {
    // 基础规则操作
    ExtractRules(ctx context.Context, sessionID string, latestUserMessage string) (*d_model.SessionDTO, error)
    ConfirmRule(ctx context.Context, sessionID string, ruleKey string, confirmed bool) (*d_model.SessionDTO, error)
    ConfirmAll(ctx context.Context, sessionID string) (*d_model.SessionDTO, error)
    Validate(ctx context.Context, sessionID string) (*d_model.RuleValidationResult, error)

    // 规则存储操作（通过 relational-graph-server）
    UpsertSchedulingRules(ctx context.Context, text string, domain string, forceUpdate bool) (*d_model.SchedulingRulesUpsertResponse, error)
    DeleteSchedulingRules(ctx context.Context, ruleIDs []string, domain string) (*d_model.SchedulingRulesDeleteResponse, error)
    ListSchedulingRules(ctx context.Context, domain string, ruleType string, page, pageSize int) (*d_model.SchedulingRulesListResponse, error)
    QueryScheduling(ctx context.Context, queryType string, entities []string, constraints map[string]any, options d_model.QueryOptions) (*d_model.SchedulingQueryResponse, error)
    ExecuteNeo4jQuery(ctx context.Context, cypher string, parameters map[string]any, operationType string, domain string, returnGraph bool) (*d_model.Neo4jQueryResponse, error)
}
```

### IAggregationService（上下文聚合服务）
位置：`domain/service/aggregation.go`

负责收集和聚合排班所需的上下文数据。

```go
type IAggregationService interface {
    CollectSchedulingContext(ctx context.Context, sessionID string) (*d_model.SessionDTO, error)
    ValidateContext(ctx context.Context, sessionID string) error
}
```

### IActorSystem（工作流系统）
位置：`domain/workflow/actor_system.go`

基于 FSM 的工作流引擎，管理状态转换与决策追踪。

```go
type IActorSystem interface {
    SetObserver(o Observer)
    SetEventPublisher(p WorkflowEventPublisher)
    SetSessionService(s ISessionService)
    SetIntentService(s IIntentService)
    SetDataService(s IDataService)
    SetRuleService(s IRuleService)
    
    QueryDecisions(sessionID string, opts DecisionQueryOptions) ([]Decision, int)
    PrometheusRegistry() *prometheus.Registry
    MetricsSnapshot() MetricSnapshot
    SendEvent(ctx context.Context, id string, workflow Workflow, event Event, payload any) error
    
    SetConfirmWaitTTL(id string, d time.Duration)
    GetConfirmWaitTTL(id string) time.Duration
    SetStepDelay(id string, d time.Duration)
    GetStepDelay(id string) time.Duration
    Stop()
}
```

## � 工作流系统

本服务实现了基于 FSM（有限状态机）的工作流引擎，支持复杂的业务流程编排。

### 工作流类型

#### 1. Schedule Workflow（排班工作流）✅
**状态**：已完成核心功能

**支持的意图**：
- `schedule.create` - 创建排班
- `schedule.adjust` - 调整排班
- `schedule.query` - 查询排班

**流程阶段**：
1. **意图识别** - 识别用户意图与参数
2. **信息补全** - 缺失参数时提示用户补充
3. **上下文收集** - 收集历史排班、人员档案等
4. **AI 人员预选** - 基于规则与历史智能筛选人员
5. **草案生成** - AI 生成排班初稿
6. **用户确认** - 用户确认或调整草案
7. **最终生成** - 生成最终排班并持久化
8. **后续调整**（可选）- 支持对已生成排班的微调

**关键特性**：
- ✅ 智能参数补全与验证
- ✅ AI 驱动的人员预选
- ✅ 增量式草案生成（逐班次）
- ✅ 多轮调整与确认
- ✅ 完整的决策追踪

#### 2. Rule Workflow（规则管理工作流）✅
**状态**：已完成，支持多意图顺序执行

**支持的意图**：
- `rule.extract` - 从自然语言提取规则
- `rule.query` - 查询已有规则
- `rule.update` - 更新规则
- `rule.delete` - 删除规则

**流程阶段**：
1. **意图识别** - 识别规则操作类型
2. **执行计划生成** - 创建多意图执行计划
3. **用户确认** - 确认执行计划
4. **顺序执行** - 按顺序执行各意图
5. **完成/失败** - 汇总执行结果

**关键特性**：
- ✅ 支持多意图批量执行
- ✅ 与 relational-graph-server 集成
- ✅ 规则冲突检测
- ✅ 详细的执行结果报告

#### 3. Dept Workflow（部门管理工作流）✅
**状态**：已完成，支持多意图顺序执行

**支持的意图**：
- `dept.staff.create` - 创建人员
- `dept.staff.update` - 更新人员
- `dept.team.create` - 创建团队
- `dept.team.update` - 更新团队
- `dept.team.assign` - 分配团队成员

**流程阶段**：
1. **意图识别** - 识别部门管理操作
2. **执行计划生成** - 创建执行计划
3. **用户确认** - 确认计划
4. **顺序执行** - 执行各意图
5. **完成** - 返回执行结果

**关键特性**：
- ✅ 支持多意图批量执行
- ✅ 执行状态追踪
- ✅ 详细的错误处理

#### 4. General Workflow（通用工作流）✅
**状态**：已完成

**功能**：
- 帮助和指引
- 闲聊处理
- 未知意图处理

### 工作流引擎特性

#### 通用执行计划框架
位置：`internal/workflow/common/`

提供可复用的多意图顺序执行框架：

```go
type IntentExecutor interface {
    ExecuteIntent(ctx, dto, wfCtx, intent) error
    GetWorkflowType() Workflow
}

type PlanExecutorConfig struct {
    IntentExecutor       IntentExecutor
    EventExecuteNext     Event
    EventAllCompleted    Event
    EventExecutionFailed Event
    EventPlanReady       Event
    StateCompleted       State
}
```

**标准流程**：
```
用户输入 → 意图识别 → 生成计划 → 用户确认 → 顺序执行 → 完成/失败
```

#### Actor System（参与者系统）
位置：`internal/workflow/engine/`

基于 Actor 模型的并发工作流引擎：
- 🔄 异步事件处理
- 📊 Prometheus 指标集成
- 🔍 完整的决策日志
- ⚡ 状态迁移追踪
- 🎯 可观测性支持

#### 状态管理
- 状态持久化（内存存储）
- 乐观锁（版本号）
- 状态迁移历史
- 决策日志查询

### 工作流事件类型

**通用事件**：
- `Event_Action_UserConfirm` - 用户确认
- `Event_Action_UserCancel` - 用户取消
- `Event_Action_UserAdjust` - 用户调整
- `Event_Action_UserSupplementary` - 用户补充信息

**各工作流特定事件**：详见 `domain/workflow/event_*.go`

## � 配置说明

配置文件位置：
- 服务配置：`config/scheduling-service.yml`
- 通用配置：`config/common.yml`
- 配置示例：`config/config.example.yml`

### 核心配置项

#### MCP 服务配置

```yaml
# Data-Server 配置
dataServer:
  serverName: "data-server"  # 服务名称（用于服务发现）
  timeout: 30000             # 查询超时（ms）

# Relational-Graph-Server 配置
graphServer:
  serverName: "relational-graph-server"
  timeout: 60000             # 图查询超时（ms）
  defaultMaxDepth: 3         # 默认搜索深度
  defaultLimit: 50           # 查询结果上限
  conflictPairTopN: 10       # 冲突分析人员上限
  enableRulesUpsert: false   # 是否启用规则写入

# Context-Server 配置
contextServer:
  serverName: "context-server"
  historyLimitDefault: 100   # 默认历史记录上限
  topKDefault: 10            # 检索 TopK
  enableAutoEmbed: true      # 启用自动向量化
  maxCountPerPhase: 50       # 每阶段最大处理数
  defaultConcurrency: 5      # 并发度
  defaultRetry: 3            # 重试次数

# MCP 通用配置（在 common.yml 中）
mcp:
  discovery_group_name: "mcp-server"
  client_timeout: 300        # 客户端超时（秒）
```

#### AI 配置

```yaml
intent:
  # 任务专用模型配置（为不同任务配置不同模型）
  taskModels:
    # 主意图识别
    mainIntent:
      provider: "ollama"
      name: "qwen3:14b"
      think: false
    
    # 二级意图识别
    subIntent:
      provider: "ollama"
      name: "qwen3:14b"
      think: false
    
    # 补充信息解析
    supplementaryInfo:
      provider: "ollama"
      name: "qwen3:7b"
      think: false
    
    # 人员预选（可用更大模型提高准确性）
    staffPreSelection:
      provider: "ollama"
      name: "qwen3:14b"
      think: true
    
    # 排班草案生成（复杂任务，建议使用大模型）
    scheduleDraft:
      provider: "ollama"
      name: "qwen3:32b"
      think: true
  
  # 主意图识别提示词
  mainIntentPrompt: |
    你是医学影像科排班与运维场景的意图识别助手...
    （详见 config.example.yml）
  
  # 补充信息解析提示词
  supplementaryInfoPrompt: |
    你是一个智能信息提取助手...
  
  # 人员预选提示词
  staffPreSelectionPrompt: |
    你是排班助手。根据人员列表、规则和历史数据，预选人员...
  
  # 排班草案生成提示词
  scheduleDraftGeneratePrompt: |
    你是医学影像科的排班助手。系统采用逐个班次增量生成模式...
  
  maxHistory: 8              # AI 调用时的历史消息上限
  failureHint: "非常抱歉，智小鲨没能理解您的需求呢..."
  
  # 二级意图提示词（按一级意图分类）
  subPrompts:
    schedule: |
      你是排班助手的子意图识别器...
    rule: |
      你是规则管理的子意图识别器...
    dept: |
      你是部门管理的子意图识别器...
```

#### 服务端口配置

```yaml
ports:
  http_port: 9601  # HTTP 服务端口

# 超时配置
timeout:
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120
  read_header_timeout: 10
```

#### 日志配置

```yaml
log:
  level: "info"           # debug/info/warn/error
  format: "json"          # json/text
  output: "stdout"        # stdout/stderr/文件路径
  enableCaller: true      # 启用调用者信息
  enableStacktrace: false # 启用堆栈跟踪
```

#### 服务发现配置

```yaml
discovery:
  enabled: true
  nacos:
    server_addresses:
      - "127.0.0.1:8848"
    namespace_id: "public"
    group: "DEFAULT_GROUP"
    cluster_name: "DEFAULT"
```

### 配置热更新

本服务支持配置热更新，以下配置变更需要重启：
- 端口配置（`ports.http_port`）
- 主机配置（`host`）
- 超时配置（`timeout.*`）
- 服务发现配置（`discovery.*`）

其他配置可以动态更新，无需重启服务。

## 🚀 使用示例

### HTTP API 示例

#### 1) 创建排班会话

```bash
curl -X POST http://localhost:9601/scheduling/session/start \
  -H "Content-Type: application/json" \
  -d '{
    "orgId": "org-001",
    "bizDateRange": "2025-10-01~2025-10-31",
    "department": "放射科",
    "modality": "CT",
    "initialMessage": "帮我生成10月份的排班"
  }'
```

#### 2) 获取会话信息

```bash
curl http://localhost:9601/scheduling/session/{sessionId}
```

#### 3) 获取排班预览数据

```bash
curl http://localhost:9601/scheduling/session/{sessionId}/schedule/preview
```

#### 4) 获取工作流决策日志

```bash
curl "http://localhost:9601/scheduling/session/{sessionId}/workflow/decisions?limit=10&reverse=true"
```

#### 5) 获取 FSM 指标

```bash
# 获取 JSON 格式指标
curl http://localhost:9601/scheduling/fsm/metrics

# 获取 Prometheus 格式指标
curl http://localhost:9601/scheduling/metrics
```

### WebSocket 交互示例

#### 建立连接

```typescript
const ws = new WebSocket('ws://localhost:9601/scheduling/ws')

ws.onopen = () => {
  console.log('连接成功')
}

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)
  console.log('收到消息:', msg)
}
```

#### 发送用户消息

```typescript
ws.send(JSON.stringify({
  type: 'user_message',
  sessionId: 'sess-001',
  data: {
    content: '帮我生成10月份的排班'
  }
}))
```

#### 触发工作流命令

```typescript
// 确认执行计划
ws.send(JSON.stringify({
  type: 'workflow_command',
  sessionId: 'sess-001',
  data: {
    command: 'confirm'
  }
}))

// 取消操作
ws.send(JSON.stringify({
  type: 'workflow_command',
  sessionId: 'sess-001',
  data: {
    command: 'cancel'
  }
}))
```

#### 收集上下文

```typescript
ws.send(JSON.stringify({
  type: 'context_collect',
  sessionId: 'sess-001',
  data: {
    force: true  // 强制重新收集
  }
}))
```

#### 拉取快照

```typescript
ws.send(JSON.stringify({
  type: 'fetch_snapshot',
  sessionId: 'sess-001',
  data: {
    parts: ['session', 'decisions']
  }
}))
```

### 服务内部调用示例

#### 1) 查询排班

```go
filter := d_model.ScheduleQueryFilter{
    OrgID:     "org-001",
    UserID:    "u-001",
    StartDate: "2025-10-01",
    EndDate:   "2025-10-31",
    Page:      1,
    PageSize:  50,
}

res, err := dataService.QuerySchedules(ctx, filter)
if err != nil {
    slog.Error("查询排班失败", "err", err)
    return
}
slog.Info("查询成功", "total", res.Total, "count", len(res.Items))
```

#### 2) 批量写入/更新排班

```go
batch := d_model.ScheduleBatch{
    Items: []d_model.ScheduleUpsertRequest{
        { 
            OrgID: "org-001", 
            UserID: "u-001", 
            WorkDate: "2025-10-15", 
            ShiftCode: "shift-morning", 
            Status: "Scheduled" 
        },
        { 
            OrgID: "org-001", 
            UserID: "u-002", 
            WorkDate: "2025-10-15", 
            ShiftCode: "shift-morning", 
            Status: "Scheduled" 
        },
    },
}

result, err := dataService.BatchUpsertSchedules(ctx, batch)
if err != nil {
    slog.Error("批量更新排班失败", "err", err)
    return
}
slog.Info("批量结果", "upserted", result.Upserted, "failed", result.Failed)
```

#### 3) 创建人员

```go
req := d_model.StaffCreateRequest{
    OrgID:      "org-001",
    Name:       "张三",
    Department: "放射科",
    Skills:     []string{"CT", "MRI"},
    Role:       "技师",
}

staffID, err := dataService.CreateStaff(ctx, req)
if err != nil {
    slog.Error("创建人员失败", "err", err)
    return
}
slog.Info("创建成功", "staffID", staffID)
```

#### 4) 管理排班规则

```go
// 创建/更新规则
response, err := ruleService.UpsertSchedulingRules(ctx, 
    "张三和李四不能在同一班", 
    "org-001", 
    false,
)
if err != nil {
    slog.Error("规则创建失败", "err", err)
    return
}

// 查询规则
rules, err := ruleService.ListSchedulingRules(ctx, 
    "org-001", 
    "conflict", 
    1, 
    20,
)
```

## 🔍 错误处理与监控

### 错误处理机制

本服务实现了完整的错误处理体系：

- **MCP 客户端自动重试**：支持指数退避重试策略
- **连接健康检查**：定期检查 MCP 服务连接状态
- **超时控制**：可配置的请求超时时间
- **降级处理**：MCP 服务不可用时的降级策略
- **详细的错误日志**：结构化日志记录所有错误上下文

### 错误码体系

错误码位置：`doc/error-codes.md`

常见错误码：
- `ERR_INTENT_DETECTION_FAILED` - 意图识别失败
- `ERR_MISSING_REQUIRED_PARAMS` - 缺少必需参数
- `ERR_MCP_CALL_FAILED` - MCP 调用失败
- `ERR_WORKFLOW_TRANSITION_INVALID` - 无效的工作流状态转换
- `ERR_SESSION_NOT_FOUND` - 会话不存在
- `ERR_RULE_VALIDATION_FAILED` - 规则验证失败

### 监控指标

#### Prometheus 指标

服务提供丰富的 Prometheus 指标：

**MCP 相关指标**：
- `mcp_call_total` - MCP 调用总数（按工具名分类）
- `mcp_call_duration_seconds` - MCP 调用耗时分布
- `mcp_call_errors_total` - MCP 调用失败总数
- `mcp_retry_total` - MCP 重试总数

**工作流指标**：
- `workflow_state_transitions_total` - 状态转换总数（按工作流和状态分类）
- `workflow_active_sessions` - 活跃会话数
- `workflow_decision_count` - 决策记录总数
- `workflow_execution_duration_seconds` - 工作流执行耗时

**AI 调用指标**：
- `ai_intent_detection_total` - 意图识别调用总数
- `ai_intent_detection_duration_seconds` - 意图识别耗时
- `ai_staff_preselection_total` - 人员预选调用总数
- `ai_draft_generation_total` - 草案生成调用总数

#### 指标端点

```bash
# JSON 格式指标快照
GET /scheduling/fsm/metrics

# Prometheus 格式指标
GET /scheduling/metrics
```

### 日志管理

#### 日志级别

- `DEBUG` - 详细调试信息
- `INFO` - 一般信息（默认）
- `WARN` - 警告信息
- `ERROR` - 错误信息

#### 结构化日志

所有日志采用结构化格式（JSON），便于日志分析：

```json
{
  "time": "2025-10-24T10:00:00Z",
  "level": "INFO",
  "msg": "MCP call completed",
  "component": "DataServerGateway",
  "tool": "schedule_manager.query_schedules",
  "duration_ms": 150,
  "success": true
}
```

### 健康检查

```bash
GET /health
```

返回示例：
```json
{
  "status": "healthy",
  "timestamp": "2025-10-24T10:00:00Z",
  "services": {
    "data-server": "healthy",
    "relational-graph-server": "healthy",
    "context-server": "healthy"
  }
}
```

## 📋 部署与运行

### 依赖服务

本服务需要以下外部服务支持：

1. **MCP 服务**
   - `data-server` - 数据管理服务（必需）
   - `relational-graph-server` - 知识图谱服务（可选）
   - `context-server` - 上下文管理服务（可选）

2. **基础设施**
   - **Agent/MCP Server Manager** - MCP 服务发现与路由
   - **Nacos** - 服务注册与配置中心
   - **MySQL** - 数据存储（由 data-server 管理）
   - **Redis** - 缓存（由 data-server 管理）
   - **Neo4j** - 知识图谱（由 relational-graph-server 管理）

3. **AI 服务**
   - **Ollama** / **OpenAI** / **百炼** - AI 推理服务（根据配置选择）

### 配置准备

1. **复制配置文件**

```bash
# 复制示例配置
cp config/config.example.yml config/scheduling-service.yml

# 编辑配置文件
vim config/scheduling-service.yml
```

2. **配置 MCP 服务地址**

在 `config/scheduling-service.yml` 中配置：
```yaml
dataServer:
  serverName: "data-server"
  
graphServer:
  serverName: "relational-graph-server"
  
contextServer:
  serverName: "context-server"
```

3. **配置 AI 服务**

```yaml
ai:
  provider: "ollama"  # 或 openai、bailian
  baseUrl: "http://localhost:11434"
```

### 启动方式

#### 方式一：通过上层 App 启动（推荐）

本服务默认通过上层 app 统一装配启动：

```bash
cd cmd/services/scheduling-service
go run main.go
```

#### 方式二：独立运行

如需独立运行，已提供完整的启动入口：

```bash
# 编译
go build -o scheduling-service cmd/services/scheduling-service/main.go

# 运行
./scheduling-service
```

#### 方式三：Docker 部署

```bash
# 构建镜像
docker build -t scheduling-service:latest .

# 运行容器
docker run -d \
  --name scheduling-service \
  -p 9601:9601 \
  -v $(pwd)/config:/app/config \
  scheduling-service:latest
```

### 环境变量

支持以下环境变量覆盖配置：

```bash
# 服务端口
export HTTP_PORT=9601

# Nacos 配置
export NACOS_SERVER_ADDR="127.0.0.1:8848"
export NACOS_NAMESPACE="public"
export NACOS_GROUP="DEFAULT_GROUP"

# 日志级别
export LOG_LEVEL="info"
```

### 健康检查

启动后，通过健康检查端点验证服务状态：

```bash
curl http://localhost:9601/health
```

### 服务注册

服务启动后会自动注册到 Nacos，可在 Nacos 控制台查看：
- 服务名：`scheduling-service`
- 端口：`9601`
- 健康检查：`/health`

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service -v

# 运行单元测试
go test ./test/unit -v

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率统计
go tool cover -func=coverage.out
```

### 测试结构

```
test/
├── unit/                           # 单元测试
│   ├── relationalgraph_gateway_test.go
│   └── ...
├── integration/                    # 集成测试（如有）
└── fixtures/                       # 测试数据
```

### 测试用例示例

```go
func TestDataServerGateway_QuerySchedules(t *testing.T) {
    // 准备测试数据
    filter := ScheduleQueryRequest{
        OrgID: "org-001",
        StartDate: "2025-10-01",
        EndDate: "2025-10-31",
    }
    
    // 执行测试
    result, err := gateway.QuerySchedules(ctx, filter)
    
    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Greater(t, result.Total, 0)
}
```

### Mock 与 Stub

测试中使用 Mock 对象模拟外部依赖：

```go
// Mock MCP ToolBus
type mockToolBus struct {
    executeFunc func(ctx context.Context, tool string, args any) (json.RawMessage, error)
}

func (m *mockToolBus) Execute(ctx context.Context, tool string, args any) (json.RawMessage, error) {
    if m.executeFunc != nil {
        return m.executeFunc(ctx, tool, args)
    }
    return nil, nil
}
```

## 🔧 开发指南

### 添加新的数据访问方法

1. **在领域仓储接口中定义方法**
   
   编辑 `domain/repository/schedule_repository.go` 或 `staffing_repository.go`：
   ```go
   type IScheduleRepository interface {
       // ...现有方法
       GetScheduleByID(ctx context.Context, scheduleID string) (*d_model.ScheduleEntry, error)
   }
   ```

2. **在 MCP 网关中添加工具调用**
   
   在 `internal/infrastructure/mcp/dataserver_gateway.go` 中：
   ```go
   const (
       // ...现有工具
       ToolScheduleGetByID = "schedule_manager.get_by_id"
   )
   
   func (gw *dataServerGateway) GetScheduleByID(ctx context.Context, req ScheduleGetByIDRequest) (*ScheduleGetByIDResponse, error) {
       // 实现工具调用
   }
   ```

3. **实现仓储层**
   
   在 `internal/infrastructure/repository/schedule_repository_impl.go` 中：
   ```go
   func (r *scheduleRepositoryImpl) GetScheduleByID(ctx context.Context, scheduleID string) (*d_model.ScheduleEntry, error) {
       req := mcp.ScheduleGetByIDRequest{ID: scheduleID}
       resp, err := r.gateway.GetScheduleByID(ctx, req)
       // ...转换和返回
   }
   ```

4. **在服务层暴露**
   
   在 `domain/service/data_service.go` 和 `internal/service/data_service.go` 中添加对应方法。

5. **添加测试**
   
   在 `test/unit/` 中添加单元测试。

### 扩展工作流

1. **定义新的工作流事件和状态**
   
   在 `domain/workflow/` 中创建新的事件定义文件：
   ```go
   // event_my_workflow.go
   const (
       Event_MyWorkflow_Start Event = "_event_my_workflow_start_"
       Event_MyWorkflow_Complete Event = "_event_my_workflow_complete_"
   )
   
   const (
       State_MyWorkflow_Init State = "_state_my_workflow_init_"
       State_MyWorkflow_Processing State = "_state_my_workflow_processing_"
   )
   ```

2. **实现工作流定义**
   
   在 `internal/workflow/myworkflow/` 中创建：
   ```go
   // definition.go
   func initMyWorkflowTransitions() []Transition {
       return []Transition{
           {
               From: State_MyWorkflow_Init,
               Event: Event_MyWorkflow_Start,
               To: State_MyWorkflow_Processing,
               Act: onMyWorkflowStart,
           },
           // ...更多转换
       }
   }
   
   func init() {
       engine.RegisterWorkflow(
           Workflow_MyWorkflow,
           State_MyWorkflow_Init,
           initMyWorkflowTransitions(),
       )
   }
   ```

3. **实现动作函数**
   
   ```go
   // actions_start.go
   func onMyWorkflowStart(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, payload any) error {
       // 实现业务逻辑
       return nil
   }
   ```

4. **注册工作流**
   
   在 `internal/wiring/container.go` 中导入：
   ```go
   import _ "jusha/agent/rostering/internal/workflow/myworkflow"
   ```

### 添加新的 MCP 服务集成

1. **定义网关接口**
   
   在 `internal/infrastructure/mcp/` 中创建新网关：
   ```go
   // myservice_gateway.go
   type IMyServiceGateway interface {
       DoSomething(ctx context.Context, req MyRequest) (*MyResponse, error)
   }
   ```

2. **实现网关**
   
   ```go
   type myServiceGateway struct {
       toolBus IToolBus
       logger logging.ILogger
   }
   
   func NewMyServiceGateway(toolBus IToolBus, logger logging.ILogger) IMyServiceGateway {
       return &myServiceGateway{
           toolBus: toolBus,
           logger:  logger,
       }
   }
   ```

3. **在容器中注册**
   
   在 `internal/wiring/container.go` 中添加网关初始化。

### 代码规范

1. **命名约定**
   - 接口以 `I` 开头（如 `IDataService`）
   - 实现类以 `Impl` 或具体名称结尾
   - 常量使用大写蛇形命名（如 `Event_Action_UserConfirm`）

2. **错误处理**
   - 使用 `fmt.Errorf` 包装错误，保留调用栈
   - 记录详细的上下文信息到日志
   - 返回业务友好的错误信息

3. **日志记录**
   - 使用结构化日志（slog）
   - 包含必要的上下文字段
   - 区分日志级别

4. **注释规范**
   - 公开接口和方法必须有注释
   - 复杂逻辑添加说明注释
   - 使用 TODO/FIXME 标记待处理项

## 📝 变更日志

### v2.0.0 (2025-10-24) - 当前版本

**重大更新**：
- ✅ 完整的工作流系统（Schedule/Rule/Dept/General）
- ✅ AI 驱动的意图识别与参数补全
- ✅ 智能人员预选与排班草案生成
- ✅ 多意图顺序执行框架
- ✅ WebSocket 实时交互支持
- ✅ 知识图谱集成（relational-graph-server）
- ✅ 上下文聚合与向量化（context-server）
- ✅ Prometheus 监控指标
- ✅ 完整的决策追踪与日志

**功能增强**：
- 🎯 增量式草案生成（逐班次生成）
- 🎯 多轮对话与上下文管理
- 🎯 规则冲突检测
- 🎯 执行计划可视化
- 🎯 配置热更新支持

**技术改进**：
- 🔧 DDD 架构完善
- 🔧 依赖注入容器优化
- 🔧 错误处理体系增强
- 🔧 日志结构化改进

### v1.0.0 (2024-01-15) - 初始版本

- ✅ 完成 DDD 架构设计
- ✅ 实现 MCP 客户端基础设施
- ✅ 集成 data-server 排班工具
- ✅ 添加完整的错误处理和重试机制
- ✅ 完善配置管理和文档

## 📚 相关文档

### 核心文档

- **[FSM 工作流引擎完全指南](./doc/fsm-guide.md)** ⭐ - FSM 核心概念、架构设计与使用指南
- **[排班工作流前后端交互示例](./doc/workflow/scheduling-interaction.md)** ⭐ - 以排班为例详解前后端交互流程

### 工作流文档

- [工作流系统详解](./internal/workflow/README.md)
- [FSM 工作流重构说明](./doc/workflow/README.md)
- [Rule Workflow 说明](./doc/workflow/rule/README.md)
- [通用执行计划框架快速参考](./internal/workflow/common/PLAN_QUICK_REFERENCE.md)

### AI 与业务逻辑

- [AI 人员预选文档](./doc/ai-staff-preselection.md)
- [排班草案生成文档](./doc/schedule_draft_generation.md)
- [用户补充信息实现](./doc/user-supplementary-implementation.md)

### 其他

- [错误码说明](./doc/error-codes.md)

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 贡献流程

1. **Fork 项目**
   ```bash
   # Fork 到你的账号下，然后克隆
   git clone https://github.com/your-username/scheduling-service.git
   cd scheduling-service
   ```

2. **创建功能分支**
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **提交更改**
   ```bash
   git add .
   git commit -m 'feat: add some amazing feature'
   ```
   
   提交信息格式：
   - `feat:` 新功能
   - `fix:` 修复 Bug
   - `docs:` 文档更新
   - `style:` 代码格式调整
   - `refactor:` 代码重构
   - `test:` 测试相关
   - `chore:` 构建/工具相关

4. **推送到分支**
   ```bash
   git push origin feature/amazing-feature
   ```

5. **开启 Pull Request**
   - 描述你的更改
   - 关联相关 Issue
   - 等待 Code Review

### 开发规范

- 遵循 Go 代码规范和本项目的编码约定
- 为新功能添加单元测试
- 更新相关文档
- 确保所有测试通过
- 运行 `go fmt` 格式化代码

### 报告 Bug

在 [Issues](https://github.com/your-org/scheduling-service/issues) 中提交 Bug 报告，请包含：
- 问题描述
- 复现步骤
- 期望行为
- 实际行为
- 环境信息（Go 版本、操作系统等）

### 功能请求

欢迎提出新功能建议！请在 Issues 中描述：
- 功能的使用场景
- 期望的实现方式
- 可能的影响

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 🙏 致谢

感谢所有为本项目做出贡献的开发者！

特别感谢：
- MCP 协议团队
- Go 社区
- 所有提供反馈和建议的用户

---

## 📡 WebSocket 实时交互协议

本服务提供 WebSocket 端口实现推拉结合的实时消息推送，支持完整的工作流状态更新与决策追踪。

### 连接端点

```
ws://localhost:9601/scheduling/ws
```

### 消息类型

所有消息类型定义见 `internal/port/ws/messages.go`。

#### 客户端 → 服务端消息

| 消息类型 | 说明 | 数据格式 |
|---------|------|---------|
| `user_message` | 发送用户自然语言消息 | `{content: string, metadata?: object}` |
| `workflow_command` | 发送工作流命令 | `{command: string, payload?: object}` |
| `context_collect` | 触发上下文聚合 | `{force?: boolean}` |
| `start_generate` | 启动草案生成 | `{}` |
| `finalize` | 执行最终生成 | `{}` |
| `fetch_decisions` | 拉取决策日志 | `{limit?: number, offset?: number}` |
| `fetch_snapshot` | 拉取会话快照 | `{parts?: string[]}` |
| `ping` | 心跳 | `{nonce?: string}` |

#### 服务端 → 客户端消息

| 消息类型 | 说明 |
|---------|------|
| `assistant_message` | 助手回复消息 |
| `session_updated` | 会话状态更新 |
| `workflow_update` | 工作流状态更新 |
| `decision_log_append` | 决策日志追加 |
| `context_update` | 上下文数据更新 |
| `validation_result` | 验证结果 |
| `finalize_completed` | 生成完成 |
| `error` | 错误消息 |
| `pong` | 心跳响应 |
| `session_snapshot` | 会话快照 |
| `info_validation_result` | 参数验证结果 |
| `execution_plan_ready` | 执行计划就绪 |
| `intent_execution_completed` | 单个意图执行完成 |

### 典型交互流程

#### 排班创建流程

```
1. 客户端连接 WebSocket
2. 发送 user_message: "帮我生成10月份的排班"
3. 服务端返回 assistant_message: "正在识别您的意图..."
4. 服务端返回 info_validation_result: 提示缺失参数（如团队、班次）
5. 客户端发送补充信息
6. 服务端返回 workflow_update: 进入上下文收集阶段
7. 客户端发送 context_collect
8. 服务端返回 context_update: 上下文数据收集完成
9. 服务端返回 workflow_update: 进入 AI 人员预选阶段
10. 服务端返回 workflow_update: 进入草案生成确认阶段
11. 客户端发送 workflow_command: {command: "confirm"}
12. 服务端逐班次生成草案，持续推送 session_updated
13. 服务端返回 workflow_update: 草案生成完成，等待用户确认
14. 客户端发送 finalize
15. 服务端返回 finalize_completed: 排班生成成功
```

### 心跳机制

```javascript
// 每 30 秒发送一次心跳
setInterval(() => {
  ws.send(JSON.stringify({
    type: 'ping',
    sessionId: sessionId,
    data: { nonce: Date.now().toString() }
  }))
}, 30000)
```

### 断线重连

```javascript
function connect() {
  const ws = new WebSocket('ws://localhost:9601/scheduling/ws')
  
  ws.onclose = () => {
    console.log('连接断开，3秒后重连...')
    setTimeout(connect, 3000)
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error)
  }
}
```

完整的 WebSocket 消息类型和示例请参见 `internal/port/ws/messages.go`。

### 意图识别与会话管理

面向排班场景的多轮会话 + 规则收集 + AI 意图识别与排班生成服务。

#### 功能概览

- 会话管理：多轮消息、上下文与决策日志
- 通用意图识别：支持排班创建、调整、规则更新、人员状态更新、查询、帮助、闲聊
- 规则抽取（启发式，后续可接入图谱 AI 工具）
- 数据聚合（现已集成 MCP data-server）
- 排班生成（占位调用，待对接 `scheduling_generate` 工具）

#### 意图识别配置

```yaml
intent:
  systemPrompt: |
    你是一个企业排班与运营领域的智能意图分析助手...
  maxHistory: 8
  failureHint: "我暂时没有理解你的意图，请说明是生成排班、调整班表、修改规则还是更新人员状态。"
```

---

## 📡 WebSocket 消息流转（以排班为例）

本服务提供 WS 通道推/拉结合的实时交互能力，消息类型定义见 `internal/port/ws/messages.go` 的 `MessageType` 枚举。以下以“排班会话从创建到生成”的典型流程为例说明端到端消息交互。

### 1) 建链与心跳

- 客户端发起连接：`GET /scheduling/ws`
- 心跳：
  - C→S：`{"type":"ping","data":{"nonce":"123"}}`
  - S→C：`{"type":"pong","data":{"nonce":"123"}}`

### 2) 开始会话与首条消息

- C→S 用户消息：
  - type: `user_message`
  - data: `UserMessageData{ content: "帮我生成10月放射科的班表" }`
- 服务端处理：
  - 保存用户消息，进行意图识别（失败则返回助手提示）
  - 可能返回：
    - S→C `assistant_message`（提示或引导）
    - S→C `session_updated`（DTO 变化）
    - S→C `workflow_update`（初始阶段/可用动作）

示例（简化）：

```json
{
  "type": "assistant_message",
  "sessionId": "sess-001",
  "data": { "role": "assistant", "content": "为你准备数据中…" },
  "ts": "2025-09-28T10:00:00Z"
}
```

### 3) 触发上下文聚合

- C→S：`context_collect`（可带 `force`）
- 服务端：
  - 通过 data-server 拉取当前/上月排班、员工档案等
  - 写入 context-server（auto_embed 或批量向量化）
  - 推送：
    - S→C `context_update`
    - S→C `decision_log_append`（FSM 若有迁移）

### 4) 启动草案/进入工作流

- C→S 工作流命令：`workflow_command`，如 `{"command":"draft_adjust"}` 或 `{"command":"confirm"}`
- 服务端：
  - 将命令映射为 FSM 事件
  - 推送：
    - S→C `workflow_update`（阶段变更、动作集更新）
    - S→C `decision_log_append`

### 5) 拉取决策与快照（可选）

- C→S `fetch_decisions`：分页/筛选拉取 FSM 决策
- C→S `fetch_snapshot`：获取 `session`、`decisions` 等部件的快照
- S→C 分别返回 `decision_log_append` 或 `session_snapshot`

### 6) 最终生成（Finalize）

- C→S：`finalize`
- 服务端：
  - 切换会话至 `generating`；调用后端生成工具
  - 成功：S→C `finalize_completed`（含最新 DTO 与结果）
  - 失败：S→C `error`（可含 `retryable`）

### 常用消息一览（节选）

- 客户端 → 服务端：

  - `user_message`：发送自然语言推进流程
  - `workflow_command`：confirm/cancel/draft_adjust/post_adjust
  - `context_collect`：聚合上下文并持久化至 context-server
  - `start_generate`：启动草案（如使用）
  - `finalize`：最终生成
  - `fetch_decisions` / `fetch_snapshot`：拉取决策/快照
  - `ping`：心跳

- 服务端 → 客户端：
  - `assistant_message`：助手提示
  - `session_updated`：DTO 变更
  - `workflow_update`：阶段与可用动作
  - `decision_log_append`：FSM 决策记录
  - `context_update`：上下文数据更新
  - `validation_result`：规则校验结果
  - `finalize_completed`：生成完成
  - `error` / `pong`

### 端到端最小示例

1. 发送首条用户消息

```json
{
  "type": "user_message",
  "sessionId": "sess-001",
  "data": { "content": "生成10月放射科班表" }
}
```

2. 触发上下文聚合

```json
{
  "type": "context_collect",
  "sessionId": "sess-001",
  "data": { "force": true }
}
```

3. 前端收到工作流阶段更新

```json
{
  "type": "workflow_update",
  "sessionId": "sess-001",
  "data": {
    "state": "Active",
    "phase": "Collecting",
    "actions": [{ "command": "confirm", "label": "确认" }]
  }
}
```

4. 发送最终生成

```json
{ "type": "finalize", "sessionId": "sess-001", "data": {} }
```

5. 收到生成完成

```json
{
  "type": "finalize_completed",
  "sessionId": "sess-001",
  "data": { "result": { "shifts": [] } }
}
```
