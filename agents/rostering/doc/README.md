# Scheduling Service

基于 DDD（Domain-Driven Design）的排班管理服务，已通过 MCP（Model Context Protocol）与 data-server 对接真实数据源，并提供 HTTP/WS 端口与工作流支撑。

## 🏗️ 架构概览（与当前代码一致）

```
scheduling-service/
├── config/                        # 配置
│   ├── config.go                  # 服务侧配置模型与装载
│   └── config.example.yml         # 示例配置
├── domain/                        # 领域层
│   ├── model/                     # 领域模型与值对象
│   │   ├── schedule.go
│   │   ├── staff.go
│   │   ├── staffing.go
│   │   └── ...
│   ├── repository/                # 领域仓储接口
│   │   ├── schedule_repository.go
│   │   └── staffing_repository.go
│   └── service/                   # 领域服务接口
│       ├── data_service.go        # IDataService 等
│       ├── aggregation.go
│       ├── execution.go
│       ├── rule.go
│       └── session.go
├── internal/                      # 内部实现
│   ├── context/                   # 上下文网关（context-server）
│   ├── infrastructure/
│   │   ├── mcp/                   # MCP 工具总线与 data-server 网关
│   │   │   ├── toolbus.go         # IToolBus（重试/超时/健康检查）
│   │   │   ├── dataserver_gateway.go   # IDataServerGateway（封装工具名）
│   │   │   └── dataserver_types.go     # DTO/请求响应类型
│   │   └── repository/            # 仓储实现（调用网关）
│   │       ├── schedule_repository_impl.go
│   │       └── staffing_repository_impl.go
│   ├── port/                      # 适配器层
│   │   ├── http/
│   │   └── ws/                    # WebSocket（messages.go 等）
│   ├── service/                   # 应用服务实现
│   │   ├── data_service.go        # IDataService 实现
│   │   ├── aggregation_service.go
│   │   ├── execution_service.go
│   │   ├── intent_service.go
│   │   ├── rule_service.go
│   │   ├── scheduling_ai_service.go
│   │   └── session_service.go
│   ├── wiring/
│   │   └── container.go           # 依赖装配容器（ServiceProvider）
│   └── workflow/                  # 工作流与参与者系统
├── setup.go                       # SetupDependenciesAndRun（对上层 app 提供启动装配）
├── go.mod / go.sum
└── test/
```

## 🔧 MCP 集成（当前实现）

### 组件角色

- IToolBus（internal/infrastructure/mcp/toolbus.go）
  - 对 MCPServerManager 进行超时/重试/健康检查包装，统一 Execute(toolName, payload)
- IDataServerGateway（internal/infrastructure/mcp/dataserver_gateway.go）
  - 基于 IToolBus 封装 data-server 工具调用与 DTO 序列化/反序列化
- 仓储实现：
  - scheduleRepositoryImpl / staffingRepositoryImpl 调用 IDataServerGateway 完成数据访问

### 已对接的 data-server 工具（名称与代码一致）

- 排班管理
  - schedule_manager.query_schedules
  - schedule_manager.upsert_schedule
  - schedule_manager.batch_upsert_schedules
  - schedule_manager.delete_schedule
- 人员管理
  - staff_manager.list_staff
  - staff_manager.create_staff
  - staff_manager.get_eligible_staff
- 配置管理
  - staffing_manager.list_teams / create_team
  - staffing_manager.list_locations / create_location
  - staffing_manager.list_skills / create_skill
- 班次需求
  - staffing_manager.list_shift_demands / create_shift_demand / check_coverage
- 请假管理
  - staffing_manager.create_leave / get_leave_records
- 数据库管理
  - schedule_manager.ensure_db_configured
  - staffing_manager.auto_migrate

## 🧩 领域服务接口（IDataService，当前签名）

位置：`domain/service/data_service.go`

```go
type IDataService interface {
        // 排班
        QuerySchedules(ctx context.Context, filter d_model.ScheduleQueryFilter) (*d_model.ScheduleQueryResult, error)
        UpsertSchedule(ctx context.Context, req d_model.ScheduleUpsertRequest) (*d_model.ScheduleEntry, error)
        BatchUpsertSchedules(ctx context.Context, batch d_model.ScheduleBatch) (*d_model.BatchUpsertResult, error)
        DeleteSchedule(ctx context.Context, userID, date string) error

        // 人员
        GetStaffProfiles(ctx context.Context, orgID, department, modality string) ([]d_model.Staff, error)
        GetEligibleStaff(ctx context.Context, filter d_model.EligibleStaffFilter) ([]d_model.Staff, error)
        ListStaff(ctx context.Context, filter d_model.StaffListFilter) (*d_model.StaffListResult, error)
        CreateStaff(ctx context.Context, req d_model.StaffCreateRequest) (int64, error)

        // 配置
        ListTeams(ctx context.Context, orgID string) ([]d_model.Team, error)
        CreateTeam(ctx context.Context, req d_model.CreateTeamRequest) (int64, error)
        ListLocations(ctx context.Context, orgID string) ([]d_model.Location, error)
        CreateLocation(ctx context.Context, req d_model.LocationCreateRequest) (int64, error)
        ListSkills(ctx context.Context, orgID string) ([]d_model.Skill, error)
        CreateSkill(ctx context.Context, req d_model.SkillCreateRequest) (int64, error)

        // 班次需求
        ListShiftDemands(ctx context.Context, orgID, date string, locationID, shiftID *int64) ([]d_model.ShiftDemand, error)
        CreateShiftDemand(ctx context.Context, req d_model.ShiftDemandCreateRequest) (int64, error)
        CheckCoverage(ctx context.Context, orgID, date string, locationID, shiftID int64) (*d_model.CoverageStatus, error)

        // 请假
        CreateLeave(ctx context.Context, req d_model.LeaveCreateRequest) (int64, error)
        GetLeaveRecords(ctx context.Context, staffID string, startDate, endDate string) ([]d_model.LeaveRecord, error)

        // 数据库管理
        EnsureScheduleDBConfigured() error
        AutoMigrateStaffing(ctx context.Context) error
}
```

## 📝 配置说明

示例参考：`config/config.example.yml` 与通用 `config/common.yml`。

### MCP 客户端

```yaml
mcp:
  agentServiceUrl: "http://localhost:8090"
  dataServerTimeout: 30000 # ms，IToolBus 默认超时覆盖
  retryCount: 3 # IToolBus 重试次数
  retryDelayMs: 1000 # IToolBus 重试间隔
```

通用配置中亦包含：

```yaml
mcp:
  discovery_group_name: "mcp-server"
  client_timeout: 300 # seconds，MCP 客户端超时（由上层通用包使用）
```

### 上下文服务器（context-server）

```yaml
contextServer:
  serverName: "context-server"
  historyLimitDefault: 100
  topKDefault: 10
  enableAutoEmbed: true
  maxCountPerPhase: 50
  defaultConcurrency: 5
  defaultRetry: 3
```

## 🚀 使用示例（基于当前接口）

> 相关类型位于 `domain/model` 包，以字符串日期（YYYY-MM-DD）为主。

### 1) 查询排班

```go
filter := d_model.ScheduleQueryFilter{
    OrgID:     "org-001",
    UserID:    "u-001",          // 可选
    StartDate: "2024-01-01",
    EndDate:   "2024-01-31",
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

### 2) 批量写入/更新排班

```go
batch := d_model.ScheduleBatch{
    Items: []d_model.ScheduleUpsertRequest{
        { OrgID: "org-001", UserID: "u-001", WorkDate: "2024-01-15", ShiftCode: "shift-morning", Status: "Scheduled" },
        { OrgID: "org-001", UserID: "u-002", WorkDate: "2024-01-15", ShiftCode: "shift-morning", Status: "Scheduled" },
    },
}

result, err := dataService.BatchUpsertSchedules(ctx, batch)
if err != nil {
    slog.Error("批量更新排班失败", "err", err)
    return
}
slog.Info("批量结果", "upserted", result.Upserted, "failed", result.Failed)
```

### 3) 获取员工档案

```go
profiles, err := dataService.GetStaffProfiles(ctx, "org-001", "放射科", "CT")
if err != nil {
    slog.Error("获取员工信息失败", "err", err)
    return
}
for _, p := range profiles {
    fmt.Printf("员工: %s, 技能: %v\n", p.Name, p.Skills)
}
```

## 🔍 错误处理

服务包含完整的错误处理机制：

- MCP 客户端自动重试
- 连接健康检查
- 降级处理支持
- 详细的错误日志

## 📋 部署与运行

1. 依赖服务

- data-server（MCP）
- context-server（MCP，可选：用于上下文聚合）
- Agent / MCP Server Manager（由服务发现发现并路由 MCP 调用）
- MySQL、Redis 等后端资源（由 data-server 管理）

2. 配置

- 复制并调整 `config/config.example.yml` 与通用 `config/common.yml`

3. 启动方式

- 本模块默认不提供独立 main；通过上层 app 进程装配启动。
- 供外部调用的装配入口：`setup.go` 的 `SetupDependenciesAndRun`（统一 HTTP/WS 路由、健康检查、依赖注入与网关创建）。
- 若需独立运行，可在上层创建一个 main 调用上述装配函数。

## 🧪 测试

```bash
# 运行单元测试
go test ./...

# 运行特定测试
go test ./internal/service -v

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 监控指标

服务提供以下监控指标：

- MCP 调用成功/失败率
- 响应时间分布
- 重试次数统计
- 连接池状态
- 业务操作计数

## 🔧 开发指南

### 添加新的数据访问方法

1. 在领域仓储接口（`domain/repository/*.go`）中定义方法
2. 在仓储实现（`internal/infrastructure/repository/*.go`）中通过 `IDataServerGateway` 增加调用逻辑
3. 在 `IDataService`（`domain/service/data_service.go`）与实现（`internal/service/data_service.go`）中暴露业务方法
4. 增补 DTO（如需，`internal/infrastructure/mcp/dataserver_types.go`）与工具名
5. 添加单元测试（`test/unit`）

### 扩展 MCP 工具支持

1. 在 `internal/infrastructure/mcp/dataserver_gateway.go` 定义工具名常量与调用方法
2. 在 `dataserver_types.go` 中补充请求/响应 DTO 与转换
3. 在仓储层调用新网关方法，并在服务层透出
4. 根据需要扩展 `IToolBus` 的重试/超时参数

## 📝 变更日志

### v1.0.0 (2024-01-15)

- ✅ 完成 DDD 架构设计
- ✅ 实现 MCP 客户端基础设施
- ✅ 集成 data-server 排班工具
- ✅ 添加完整的错误处理和重试机制
- ✅ 完善配置管理和文档

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 原有功能说明 (Legacy)

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
