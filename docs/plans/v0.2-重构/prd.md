# AI 智能排班 SaaS 平台 — 产品需求文档（重构版）

> **文档版本**: 2.0  
> **文档日期**: 2026-03-23  
> **产品名称**: AI 智能排班 SaaS 平台  
> **产品阶段**: 重构立项  
> **版权方**: 聚煞科技

---

## 一、重构背景与现状问题

### 1.1 为什么必须重构

当前系统经历了 V1 → V2 → V3 → V4 四次迭代，每次迭代在前一版本上叠加而非替换，导致代码库严重退化。系统最初设计为面向单一医疗机构的「智能框架」，用于兼容老旧排班系统，而非面向多租户的云端产品。现在需要将其转型为独立部署的多租户 SaaS 平台，当前架构从根本上无法满足。

### 1.2 现有架构核心问题

#### 问题一：历史版本共存，代码混乱

历史上四个版本的排班工作流（schedule / schedule_v2 / schedule_v3 / schedule_v4）长期并存，造成代码结构退化。当前仓库已经完成 V2/V3 工作流删除，运行时入口收敛为 `schedule` 与 `schedule_v4`；当前剩余收尾项主要是文档校准、配置兼容别名和少量历史元数据语义，而不再是旧工作流代码本体。

#### 问题二：过度依赖 MCP 协议造成架构扭曲

系统将 MCP（Model Context Protocol）从「AI 模型调用外部工具的标准协议」扭曲为「所有服务间通信的唯一手段」。排班智能体通过 MCP ToolBus 调用 Data Server 做一次简单的员工查询，链路为：业务代码 → MCP Client → JSON-RPC 序列化 → HTTP → MCP Server → 反序列化 → 数据库查询 → 序列化 → HTTP 响应 → 反序列化。一个本可以是一行 SQL 的操作被包装了六层抽象。MCP 适合 AI 模型与工具的松耦合通信，不适合作为核心业务服务间的调用协议。

#### 问题三：职责边界混乱

- **Rostering Agent** 同时承担 AI 意图识别、工作流编排、规则引擎、排班算法、全局评审五种职责
- **Scheduling Service** 与 **Rostering Agent** 的职责高度重叠（都做意图识别和工作流编排）
- **Management Service** 本应是核心数据管理服务，当前标注为「预留模块」，实际已实现大量 CRUD 和 V4 迁移 API，但排班智能体不直接调用它而是通过 MCP 绕行 Data Server
- **Data Server** 本质只是 Management Service 的一层 MCP 包装
- **Relational Graph Server** 将 Neo4j 图数据库引入仅用于规则关系管理，实际使用中大量规则逻辑已迁移到 V4 代码化引擎，图数据库沦为冗余

#### 问题四：多租户能力缺失

- 租户隔离仅通过业务代码中手动传递 `orgId` 参数实现，无统一的租户上下文中间件
- 路由守卫（`router-guards.ts`）中的登录鉴权代码全部被注释掉，认证形同虚设
- 前端用户系统依赖外部 `auth-server`，无独立的租户管理、订阅计费、配额控制能力
- 数据库表无统一的租户隔离机制，完全依赖业务代码手动拼接 `WHERE org_id = ?`，极易遗漏
- 无租户数据备份、迁移、销毁的生命周期管理

#### 问题五：兼容层膨胀

- `v3_compat.go` / `v3-compat.ts` 散布在前后端各处，做 V3↔V4 枚举值转换
- 领域模型中存在大量 `Deprecated` 类型转换函数（`ConvertOccupiedSlotsToMap`、`ConvertRequirementsToMap` 等），「计划在 v3.1.0 移除」但从未移除
- `TaskResult` 结构体中 `ScheduleDraft` 标注「已废弃」同时新增 `ShiftSchedules`，调用方需同时处理两种格式
- 前端存在 V3 和 V4 两套枚举值选项列表（`ruleTypeOptions` 和 `v3RuleTypeOptions`），UI 需同时兼容

#### 问题六：AI 调用架构不合理

- V3 对 LLM 的调用高达约 720 次/排班周期（20 班 × 7 天），包含大量可确定性计算的规则匹配和校验工作
- V4 试图将 LLM 限制为「仅选人」，但 V3 的 LLM 调用逻辑仍在生产运行
- 规则解析仍依赖 LLM 每次排班时重新理解自然语言规则，而非一次解析后结构化存储
- AI 模型配置分散在多处（`SchedulingAI.TaskModels`、`aiFactory`、`assessmentModel`），无统一管理

---

## 二、产品定位（重构后）

### 2.1 产品定义

**AI 智能排班 SaaS 平台**是一个面向中大型组织（医疗、制造、零售等行业）的云端多租户排班管理平台。组织注册后即可使用，通过 AI 辅助实现排班的自动化创建、规则管理和智能调整。

### 2.2 核心原则

- **产品化优先**：不是一个可定制的框架，而是一个开箱即用的 SaaS 产品
- **确定性优先**：排班核心逻辑 100% 代码化执行，AI 仅作为辅助而非关键路径
- **简洁架构**：消除所有历史版本代码、兼容层和冗余抽象
- **原生多租户**：租户隔离从数据库到 API 到前端全链路内建

### 2.3 目标用户

| 角色 | 说明 |
|------|------|
| 平台管理员 | 管理租户、订阅、配额、系统运维 |
| 租户管理员 | 管理本组织的人员、科室、班次、规则 |
| 排班负责人 | 创建排班计划、审核排班方案、处理调班 |
| 普通员工 | 查看个人排班、提交偏好和请假 |

---

## 三、系统架构（重构后）

### 3.1 架构总览

重构后采用简洁的单体模块化架构（Modular Monolith），彻底移除 MCP 协议，AI 通过普通 HTTP 适配器调用大模型 API。

```
┌──────────────────────────────────────────────────────────────┐
│                       前端 (SPA)                              │
│           Vue 3 + TypeScript + Element Plus                   │
│                                                               │
│  组织管理 │ 排班中心 │ 人员管理 │ 规则管理 │ 数据看板         │
└──────────────────────────────────────────────────────────────┘
                          │ HTTPS + WebSocket
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                    API Gateway / BFF                           │
│     认证 │ 组织节点路由 │ 限流 │ 审计日志                      │
└──────────────────────────────────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
┌────────────────┐ ┌─────────────┐ ┌──────────────┐
│  Core Service  │ │  AI Service  │ │ Admin Service │
│  (合并原       │ │              │ │               │
│  Management +  │ │  对话管理    │ │ 组织树管理    │
│  Agent非AI部分 │ │  意图识别    │ │ 账户管理      │
│  + Data Server)│ │  AI辅助选人  │ │ AI 模型配置   │
│                │ │  规则解析    │ │ 运维监控      │
│  排班管道      │ └──────┬──────┘ └──────────────┘
│  规则引擎      │        │                │
│  数据管理      │        │ HTTP           │
└───────┬────────┘        ▼                │
        │   ┌────────────────────────┐     │
        │   │  外部大模型 API         │     │
        │   │  (OpenAI/阿里百炼/...) │     │
        │   └────────────────────────┘     │
        │                                  │
        ▼ (GORM 进程内直连)                ▼
┌──────────────────────────────────────────────────────────────┐
│                     数据层                                    │
│                                                               │
│  MySQL 8.0+ (主库, GORM 全局 Scope 租户隔离)                  │
│  Redis (缓存/会话/限流)                                       │
└──────────────────────────────────────────────────────────────┘
        │                 │                │
```

### 3.2 关键架构决策

| 决策 | 当前状态 | 重构后 | 理由 |
|------|---------|--------|------|
| 内部通信 | MCP over HTTP（全量） | 进程内函数调用 | 核心业务不需要 JSON-RPC 序列化开销 |
| MCP 协议 | 所有服务间通信 + AI 调用 | **完全移除** | AI 调用通过普通 HTTP 适配器直连大模型 API，无需 MCP 抽象层 |
| 数据库 | MySQL + Milvus + Neo4j | MySQL 8.0+（主） + Redis（缓存） | 保留团队熟悉的 MySQL；移除 Milvus（语义检索不再需要）和 Neo4j（规则关系用结构化表 + 代码引擎） |
| 服务形态 | 7 个微服务 + 7 个 Git Submodule | 单体模块化（3 个内部模块） | 团队规模不支撑微服务运维，模块化保留未来拆分能力 |
| 排班版本 | V1/V2/V3/V4 共存 | 仅保留 V4 引擎化逻辑，清除所有历史代码 | 彻底消除兼容层和混乱的条件分支 |
| 服务发现 | Nacos | 云原生（K8s Service / 环境变量） | SaaS 部署于云端，无需独立注册中心 |
| 租户隔离 | 手动传 orgId | GORM 全局 Scope 自动注入 + 中间件 | 应用层统一拦截，确保所有查询自动携带 tenant_id |
| 租户结构 | 扁平（单层 org_id） | **树状组织**（机构→院区→科室），各节点独立登录和规则配置 | 贴合医疗等行业的多级组织架构 |
| 认证 | 外部 auth-server（注释掉的路由守卫） | 内建 JWT + RBAC | SaaS 产品需要独立的认证体系 |
| Core Service | Management Service + Rostering Agent + Data Server + Scheduling Service 分散 | **合并为 Core Service**（非 AI 部分）；AI 相关剥离到 AI Service | 消除职责重叠和 MCP 绕行，直连数据库 |
| 排班编排 | FSM 状态机引擎（`pkg/workflow/`），V1~V4 四套工作流共存 | **Pipeline + 可复用 Step** 轻量管道 | 排班是线性流程，不需要复杂状态机；Step 可被多条 Pipeline 复用，零冗余 |

### 3.3 技术栈（重构后）

**后端**: Go 1.23+（标准库 net/http + chi 路由 / GORM / WebSocket）

**前端**: Vue 3 + TypeScript（Element Plus / Pinia / ECharts / Vite）

**数据库**: MySQL 8.0+（GORM + 租户 Scope）/ Redis 7+

**AI**: OpenAI API / 阿里百炼 / 其他 OpenAI 兼容接口（统一适配器）

**部署**: Docker + Kubernetes / 云托管 MySQL + Redis

---

## 四、多租户设计

### 4.1 组织树模型

租户不再是扁平结构，而是支持多级树状组织。每个节点（机构/院区/科室）可独立登录、独立管理排班，上级节点可查看下级数据。

```
Platform (平台)
  └── Organization (机构/医院)          ← 顶级租户，独立登录入口
        ├── Campus A (院区A)            ← 二级节点，可独立登录
        │     ├── Department X (科室X)  ← 叶子节点，排班执行单元
        │     └── Department Y (科室Y)
        └── Campus B (院区B)
              └── Department Z (科室Z)
```

**组织节点属性**:

| 字段 | 说明 |
|------|------|
| id | 节点唯一标识 |
| parent_id | 父节点 ID（顶级为 NULL） |
| node_type | 节点类型：organization / campus / department / custom |
| name | 节点名称 |
| code | 节点编码（同级唯一） |
| path | 物化路径（如 `/org1/campusA/deptX`，用于快速查询祖先/后代） |
| depth | 层级深度（0=机构，1=院区，2=科室...） |
| is_login_point | 是否可作为独立登录入口 |

**数据可见性规则**:

- 每个节点只能看到**自身及其所有后代节点**的数据
- 科室级别是排班的最小执行单元，排班数据归属到科室
- 上级节点（院区/机构）可查看下级所有科室的排班汇总
- 查询时通过 `path LIKE '/org1/campusA/%'` 快速过滤后代数据

### 4.2 数据隔离策略

**隔离粒度**: 科室级。每个科室只能看到自己的排班数据，上级节点可查看所有下级。

**数据库层**: 所有业务表包含 `org_node_id`（归属的组织节点）列

```sql
-- 所有业务表包含 org_node_id，建立索引
-- 排班数据归属到科室节点
ALTER TABLE schedule_assignments ADD INDEX idx_org_node (org_node_id);
```

**应用层**: GORM 全局 Scope + 请求级组织上下文中间件

```
请求进入 → JWT 解析 → 提取用户当前登录的 org_node_id → 存入 Context
         → 查询本节点数据：WHERE org_node_id = ?
         → 查询含下级数据：WHERE org_node_id IN (本节点及所有后代节点 ID)
         → 响应返回
```

```go
// 仅查本节点数据
func NodeScope(nodeID string) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("org_node_id = ?", nodeID)
    }
}

// 查本节点 + 所有后代节点数据（上级查看下级）
func NodeTreeScope(nodePath string) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("org_node_id IN (SELECT id FROM org_nodes WHERE path LIKE ?)",
            nodePath + "%")
    }
}
```

**防护机制**: GORM Callback 拦截所有 SQL，检测是否携带 `org_node_id` 条件；未携带时拒绝执行并告警。

### 4.3 规则继承与覆盖

组织树中的规则遵循**继承+覆盖**模型：子节点默认继承上级所有规则，可自行覆盖或新增。

**继承规则**:

- 机构级规则自动对所有院区、科室生效
- 院区级规则自动对该院区下所有科室生效
- 科室级规则仅对本科室生效
- 子节点可创建与上级同类型的规则进行覆盖（以子节点为准）
- 子节点可新增上级没有的规则

**规则来源标记**:

| 字段 | 说明 |
|------|------|
| org_node_id | 规则归属的组织节点 |
| is_inherited | 是否从上级继承（只读展示，不可编辑；编辑时自动创建本级副本） |
| override_rule_id | 覆盖的上级规则 ID（NULL 表示新增规则） |

**生效规则计算**（排班时）:

```
对于科室 D 的排班：
  1. 收集 D 的所有祖先节点的规则（沿 path 向上遍历）
  2. 按从上到下的顺序合并：机构规则 → 院区规则 → 科室规则
  3. 同类型规则：下级覆盖上级
  4. 最终得到该科室的「生效规则集」
```

### 4.4 组织节点生命周期

| 阶段 | 能力 |
|------|------|
| 创建 | 平台管理员创建顶级机构 → 机构管理员创建院区/科室 |
| 配置 | 各节点独立配置班次、规则、人员 |
| 登录 | 标记为 `is_login_point` 的节点可作为独立登录入口 |
| 停用 | 停用节点后其下所有数据冻结，不可创建新排班 |

---

## 五、核心功能模块

### 5.1 排班引擎（确定性，零 LLM 依赖）

排班引擎是系统核心，100% 代码化，不依赖任何 LLM 调用。AI 辅助是可选增强，不是关键路径。

#### 5.1.1 排班创建流程

```
选择排班周期
  → 选择参与班次
  → 配置每日人数需求
  → [可选] 收集个人偏好/临时需求
  → 规则引擎预计算
      ├── 拓扑排序确定班次执行顺序
      ├── 候选人过滤（请假/占位/资质）
      ├── 规则匹配与约束检查
      └── 偏好评分
  → 三阶段排班执行
      ├── 阶段零：固定排班占位
      ├── 阶段一：规则性占位（排他/必须同时/人员规则）
      └── 阶段二：兜底填充（剩余名额分配）
  → 确定性校验（全规则扫描）
  → 用户审核（甘特图预览 + 详情）
  → 确认保存
```

#### 5.1.2 排班管道架构（Pipeline + 可复用 Step）

排班流程采用 **Pipeline + 可复用 Step** 架构，替代原有的 FSM 状态机引擎。每个 Step 是独立的原子处理单元，Pipeline 是 Step 的有序编排组合。

**核心设计**:

- **Step 接口**：每个 Step 实现 `Execute(ctx, state) error`，是独立的、可单元测试的原子操作
- **Pipeline**：Step 的有序组合，本身不含业务逻辑，仅负责顺序执行和错误中断
- **ScheduleState**：在 Pipeline 中流转的共享状态对象，包含排班周期、候选人、规则集、排班结果等

**可复用 Step 池**:

| Step | 职责 | 被哪些 Pipeline 复用 |
|------|------|---------------------|
| LoadRules | 加载生效规则集（含组织树继承合并） | 创建、调整、AI辅助 |
| FilterCandidates | 候选人过滤（请假/占位/资质） | 创建、AI辅助 |
| PhaseZero | 固定排班占位 | 创建、AI辅助 |
| PhaseOne | 规则性占位（排他/必须同时/人员规则） | 创建、AI辅助 |
| PhaseTwo | 兜底填充（评分排序 + 随机化） | 创建 |
| AISelect | AI 辅助选人（替代 PhaseTwo 的评分兜底） | AI辅助 |
| FullValidation | 全规则扫描校验 | 创建、调整、AI辅助 |
| SaveDraft | 保存排班草稿 | 创建、调整、AI辅助 |
| NotifyWS | WebSocket 推送排班进度/结果 | 创建、AI辅助 |

**预置 Pipeline**:

```
确定性排班 Pipeline（零 AI）:
  LoadRules → FilterCandidates → PhaseZero → PhaseOne
  → PhaseTwo → FullValidation → SaveDraft → NotifyWS

AI 辅助排班 Pipeline（PhaseTwo 替换为 AISelect）:
  LoadRules → FilterCandidates → PhaseZero → PhaseOne
  → AISelect → PhaseTwo(剩余) → FullValidation → SaveDraft → NotifyWS

排班调整 Pipeline（局部校验）:
  LoadRules → ApplyEdit → FullValidation → SaveDraft → NotifyWS
```

**与 FSM 状态机的对比**:

| 维度 | 原 FSM 状态机 | Pipeline + Step |
|------|-------------|----------------|
| Step 复用 | 靠 Transition.Act 引用，状态转换表耦合 | Step 是独立 struct，天然可复用 |
| 新增流程 | 需定义完整状态枚举 + 转换表 + 事件 | 组合已有 Step 即可 |
| AI 分支 | 整个状态图需考虑 AI 可选路径 | 替换/插入一个 AI Step |
| 可测试性 | 需 mock 整个 FSM 上下文 | 每个 Step 独立单元测试 |
| 调试 | 状态转换日志难追踪 | 线性执行，日志清晰 |

#### 5.1.3 规则引擎

**规则分类**:

| 类别 | 子类 | 说明 | 示例 |
|------|------|------|------|
| constraint（约束） | forbid | 禁止规则 | 排他班次、禁止日期 |
| constraint | limit | 限制规则 | 最大次数、最小休息天数 |
| constraint | must | 必须规则 | 必须同时、周期性 |
| preference（偏好） | prefer | 偏好规则 | 优先安排 |
| preference | combinable | 可合并 | 上下午班可合并 |
| dependency（依赖） | source | 人员来源 | 下夜班来自上半夜班 |
| dependency | order | 执行顺序 | 班次间依赖 |

**规则配置方式**:

- 结构化表单（主要方式）：管理员通过表单直接配置
- AI 辅助解析（可选增强）：粘贴自然语言描述 → AI 一次性解析为结构化 → 人工确认 → 保存

**性能要求**: 100 规则 × 50 人 × 7 天 < 500ms

#### 5.1.4 排班调整

- 手动调整：甘特图拖拽 / 表格编辑
- 冲突实时检测：修改后即时校验规则合规性（复用 `FullValidation` Step）
- 变更记录：每次调整记录操作人、时间、原因
- 调整流程使用独立的「排班调整 Pipeline」，复用 LoadRules、FullValidation、SaveDraft 等 Step

### 5.2 AI 辅助（可选增强，非关键路径）

AI 功能作为增强能力叠加在确定性排班引擎之上，关闭 AI 后系统仍可正常使用。

#### 5.2.1 AI 辅助排班选人

当排班引擎的兜底填充阶段遇到多名候选人评分接近时，可选择调用 AI 做最终选人决策。

输入：合格候选人列表（含偏好评分）+ 上下文（历史排班、工作量均衡度）

输出：推荐排班方案

此功能可通过配置开关关闭，关闭后使用评分排序 + 随机化填充。

#### 5.2.2 AI 对话交互

- 自然语言查询排班（"下周谁上夜班"）
- 自然语言调整排班（"把张三周五的班换成李四"）
- 自然语言规则解析（将描述转为结构化规则配置）

#### 5.2.3 AI 调用管控

- 统一 AI HTTP 适配器（直连大模型 API，支持 OpenAI / 阿里百炼 / 其他 OpenAI 兼容接口）
- 组织级 AI 调用配额控制（管理后台配置）
- 调用日志与成本核算
- AI 模型参数集中在管理后台配置（模型地址、API Key、默认模型等）

### 5.3 数据管理

#### 5.3.1 员工管理

- 员工档案 CRUD（姓名、工号、手机、邮箱、职位、角色、状态、入职日期）
- 角色体系：可配置（如医疗场景：住院医 / 主治医 / 副高 / 正高）
- 分类标签：可配置（如一线 / 二线 / 三线）
- 批量导入导出（Excel）

#### 5.3.2 科室与分组管理

- 科室树形结构
- 分组管理（班组、项目组等）
- 分组成员管理
- 分组与班次关联

#### 5.3.3 班次管理

- 班次模板 CRUD（名称、编码、时段、时长、是否跨天、颜色标识）
- 班次优先级与依赖关系配置
- 班次状态管理（启用/停用）

#### 5.3.4 请假管理

- 请假记录 CRUD
- 自动影响排班候选人过滤
- 假期余额管理

### 5.4 排班展示与交互

#### 5.4.1 甘特图视图

- 纵轴：员工列表，横轴：日期时间线
- 支持周视图 / 月视图切换
- 班次色块展示，同日多班次堆叠
- 交互：双击添加、拖拽调整、键盘删除
- 实时冲突高亮

#### 5.4.2 表格视图

- 员工 × 日期矩阵
- 单元格内显示班次名称
- 支持批量编辑

#### 5.4.3 统计看板

- 工作量分布（按员工 / 按班次）
- 规则合规率
- 加班/欠班统计
- 排班覆盖率

### 5.5 管理后台（Admin Service）

#### 5.5.1 组织树管理

- 创建/编辑/停用组织节点（机构→院区→科室）
- 配置节点是否可作为独立登录入口
- 查看组织树结构全景
- 节点间数据统计汇总

#### 5.5.2 账户管理

- 内建 JWT 认证（注册 / 登录 / 密码重置）
- RBAC 权限模型
- 预置角色（平台管理员 / 机构管理员 / 科室管理员 / 排班负责人 / 普通员工）
- 用户与组织节点绑定（一个用户可关联多个节点，切换登录上下文）

#### 5.5.3 AI 模型配置

- 配置 AI 提供商（OpenAI / 阿里百炼 / 自定义 OpenAI 兼容接口）
- 配置模型参数（API 地址、API Key、默认模型名称、温度等）
- 按组织节点配置不同的模型（如机构统一配置，科室可覆盖）
- AI 功能开关（全局 / 按组织节点）
- AI 调用日志查看

#### 5.5.4 运维监控

- 服务健康检查
- 性能指标（API 延迟、排班引擎耗时）
- 错误追踪
- 审计日志（谁在何时做了什么操作）

---

## 六、数据库设计

### 6.1 设计原则

- 所有业务表包含 `tenant_id` 列并建立联合索引
- 使用 VARCHAR(64) 存储 UUID 主键（兼容 MySQL 无原生 UUID 类型）
- 使用 MySQL JSON 类型存储灵活结构数据
- 时间字段统一使用 `DATETIME` 并在应用层处理时区

### 6.2 核心表结构

```
── 平台级 ──
org_nodes               # 组织树节点（机构/院区/科室，自引用树）
users                   # 用户账户（跨组织唯一）
user_node_roles         # 用户-组织节点-角色关联
ai_model_configs        # AI 模型配置（按组织节点，支持继承）

── 组织节点级（均含 org_node_id） ──
employees               # 员工
employee_groups         # 分组
group_members           # 分组成员
shifts                  # 班次模板
shift_dependencies      # 班次依赖关系

rules                   # 排班规则（含继承/覆盖标记）
rule_associations       # 规则关联（班次/分组/员工）
rule_dependencies       # 规则依赖
rule_conflicts          # 规则冲突

schedules               # 排班计划（周期）
schedule_assignments    # 排班分配（员工×日期×班次）
schedule_changes        # 排班变更记录

leaves                  # 请假记录
personal_preferences    # 个人偏好

ai_call_logs            # AI 调用日志
audit_logs              # 审计日志
```

### 6.3 与当前数据库的对比

| 项目 | 当前 | 重构后 |
|------|------|--------|
| 数据库 | MySQL（保留） | MySQL 8.0+（继续使用，零迁移成本） |
| 租户隔离 | 应用层 WHERE org_id（手动） | GORM 全局 Scope 自动注入 + Callback 防漏检测 |
| 向量存储 | 独立 Milvus 集群 | 移除（语义检索功能不再需要） |
| 图数据库 | 独立 Neo4j 集群 | 移除；规则关系用结构化表 + 代码引擎 |
| V3 兼容字段 | 大量 nullable 兼容列 | 清除；仅保留 V4 结构化字段 |

---

## 七、前端架构（重构后）

### 7.1 清理项

- 删除所有 V3 兼容枚举和选项列表（`v3RuleTypeOptions` 等）
- 删除所有 `v3-compat.ts` 文件
- 移除被注释掉的路由守卫代码，实现真正的认证鉴权
- 移除对外部 `auth-server` 的依赖，使用内建认证 API
- 统一 API 路径前缀，消除 `/auth-server`、`/api/management/`、`/api/mcp/` 的混杂

### 7.2 页面结构

```
/login                      # 登录（支持选择组织节点登录）
/dashboard                  # 首页看板
/scheduling                 # 排班中心
  /scheduling/create        # 创建排班（向导式）
  /scheduling/calendar      # 排班日历（甘特图 + 表格）
  /scheduling/history       # 排班历史
/employees                  # 员工管理
/shifts                     # 班次管理
/rules                      # 规则管理（显示继承来源，支持覆盖）
/leaves                     # 请假管理
/ai-assistant               # AI 助手（对话式交互）
/settings                   # 设置
  /settings/organization    # 组织节点信息
  /settings/users           # 用户与权限
/admin                      # 管理后台（仅管理员）
  /admin/org-tree           # 组织树管理
  /admin/accounts           # 账户管理
  /admin/ai-config          # AI 模型配置
  /admin/monitoring         # 运维监控
```

---

## 八、API 设计

### 8.1 设计原则

- RESTful 风格，统一前缀 `/api/v1/`
- 所有请求通过 JWT Bearer Token 认证
- 当前组织节点 ID 从 Token 中自动提取，无需手动传参
- 上级查看下级数据时通过 `?scope=tree` 参数切换
- 分页统一使用 `?page=1&size=20`
- 错误响应统一格式 `{ "error": { "code": "xxx", "message": "xxx" } }`

### 8.2 核心 API 分组

```
── 认证 ──
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/password/reset
POST   /api/v1/auth/switch-node          # 切换当前登录的组织节点

── 员工 ──
GET    /api/v1/employees
POST   /api/v1/employees
GET    /api/v1/employees/:id
PUT    /api/v1/employees/:id
DELETE /api/v1/employees/:id
POST   /api/v1/employees/import

── 班次 ──
GET    /api/v1/shifts
POST   /api/v1/shifts
PUT    /api/v1/shifts/:id
DELETE /api/v1/shifts/:id
POST   /api/v1/shifts/dependencies

── 规则 ──
GET    /api/v1/rules                      # 返回生效规则集（含继承标记）
GET    /api/v1/rules/effective            # 计算当前节点的最终生效规则
POST   /api/v1/rules
PUT    /api/v1/rules/:id
DELETE /api/v1/rules/:id
POST   /api/v1/rules/parse                # AI 辅助解析（可选）
POST   /api/v1/rules/validate

── 排班 ──
POST   /api/v1/schedules                  # 创建排班计划
GET    /api/v1/schedules/:id
POST   /api/v1/schedules/:id/generate     # 执行排班生成
GET    /api/v1/schedules/:id/assignments
PUT    /api/v1/schedules/:id/assignments
POST   /api/v1/schedules/:id/validate
POST   /api/v1/schedules/:id/publish

── AI ──
POST   /api/v1/ai/chat                    # AI 对话
POST   /api/v1/ai/suggest                 # AI 排班建议
GET    /api/v1/ai/usage                   # AI 用量统计

── 管理后台 ──
GET    /api/v1/admin/org-nodes            # 组织树
POST   /api/v1/admin/org-nodes
PUT    /api/v1/admin/org-nodes/:id
GET    /api/v1/admin/accounts             # 账户管理
POST   /api/v1/admin/accounts
GET    /api/v1/admin/ai-config            # AI 模型配置
PUT    /api/v1/admin/ai-config
GET    /api/v1/admin/metrics              # 运维指标
```

---

## 九、重构执行计划

### 9.1 阶段划分

#### 阶段一：地基搭建（4 周）

- 搭建新项目骨架（Go 单体模块化 + Vue 3 前端）
- 实现组织树模型（org_nodes 表 + 物化路径 + 层级管理）
- 实现多租户隔离（GORM 全局 Scope + 组织节点中间件 + Callback 防漏）
- 实现认证系统（JWT + RBAC + 组织节点切换）
- 实现基础数据管理 API（员工/班次/分组 CRUD）
- 前端登录（支持选择组织节点）+ 管理后台骨架

#### 阶段二：排班核心（4 周）

- 移植 V4 规则引擎（清除所有 V3 兼容代码）
- 实现规则继承与覆盖机制（沿组织树向上合并规则）
- 实现 Pipeline + Step 排班管道框架（可复用 Step 池 + 确定性排班 Pipeline + 排班调整 Pipeline）
- 实现排班甘特图展示（科室级 + 上级汇总视图）
- 实现规则管理（结构化表单配置，显示继承来源）
- 实现请假管理 + 排班候选人过滤

#### 阶段三：AI 增强（3 周）

- 实现 AI 调用适配器（统一 HTTP 适配器 + 配额控制）
- 实现 AI 对话交互（意图识别 + 对话管理）
- 实现 AI 辅助规则解析
- 实现 AISelect Step 并组装「AI 辅助排班 Pipeline」（复用已有确定性 Step，仅新增 AI 选人环节）

#### 阶段四：平台能力（3 周）

- 实现订阅与计费系统
- 实现平台管理后台
- 实现审计日志
- 实现数据导入导出
- 实现统计看板

#### 阶段五：上线准备（2 周）

- 性能测试与优化
- 安全审计
- 文档编写
- 灰度发布方案
- 数据迁移工具（从旧系统迁移）

### 9.2 数据迁移策略

- 继续使用 MySQL，无需跨数据库迁移，大幅降低风险
- 编写 SQL 迁移脚本：清洗所有 V3 兼容数据，统一为 V4 格式
- 新增 `tenant_id` 列并回填数据（现有 `org_id` 映射为 `tenant_id`）
- 删除废弃的 V3 兼容列和冗余索引
- 旧表结构冻结 → 迁移脚本执行 → 数据验证 → 新代码上线
- 保留旧表备份 30 天

### 9.3 清除清单

重构需要**彻底删除**的内容：

| 清除项 | 说明 |
|--------|------|
| `schedule/`（V1 工作流） | 已废弃 |
| `schedule_v2/`（V2 工作流） | 已删除 |
| `schedule_v3/`（V3 工作流） | 已删除 |
| 所有 `v3_compat.go` / `v3-compat.ts` | 兼容层 |
| 所有 `@deprecated` 标记的函数和类型 | 过渡期代码 |
| `mcp-servers/data-server` | 被 Core Service 直接数据库访问替代 |
| `mcp-servers/context-server` | 语义检索移除；对话管理内建到 Core Service |
| `mcp-servers/relational-graph-server` | 图数据库移除；规则引擎代码化 |
| Neo4j 依赖 | 移除 |
| Milvus 依赖 | 移除（语义检索功能不再需要） |
| Nacos 依赖 | 用 K8s 原生服务发现替代 |
| MinIO 依赖 | 移除（文件导出使用本地临时存储或云对象存储） |
| 所有 Git Submodule | 合并为单一代码仓库 |
| `services/scheduling-service` | 功能合并到 Core Service |
| `pkg/workflow/` FSM 状态机引擎 | 用 Pipeline + Step 轻量管道替代 |
| 现存 FSM 工作流定义（`schedule/` `schedule_v4/`） | 提取 V4 确定性逻辑到 Step，最终删除剩余 FSM 工作流定义 |
| 外部 `auth-server` 依赖 | 内建认证系统替代 |

---

## 十、非功能需求

### 10.1 性能

| 指标 | 目标 |
|------|------|
| 排班引擎（100 规则 × 50 人 × 7 天） | < 500ms |
| API 平均响应时间 | < 200ms |
| 甘特图渲染（200 员工 × 30 天） | < 1s |
| WebSocket 消息延迟 | < 100ms |
| 并发租户支持 | > 500 |

### 10.2 安全

- JWT Token 认证 + 定期轮换
- GORM 全局 Scope 租户隔离 + Callback 防漏检测
- API 限流（租户级 + 全局级）
- 敏感操作二次认证
- 数据传输全链路 HTTPS
- SQL 注入 / XSS / CSRF 防护
- 审计日志不可篡改

### 10.3 可用性

- SLA 目标：99.9%
- 数据库自动备份（每日）
- 无状态服务设计，支持水平扩展
- 优雅降级：AI 服务不可用时排班引擎正常工作

### 10.4 可观测性

- 结构化日志（JSON 格式）
- Prometheus 指标（排班引擎耗时、AI 调用量、API 延迟）
- 分布式追踪（OpenTelemetry）
- 错误告警（延迟 / 错误率 / 配额耗尽）

---

## 十一、项目结构（重构后）

```
gantt-saas/
├── cmd/
│   ├── server/              # 唯一服务入口（单体模块化，Core/AI/Admin 均为 internal 包）
│   └── migrate/             # 数据库迁移工具（golang-migrate CLI 封装）
├── internal/
│   ├── tenant/              # 多租户基础设施
│   │   ├── middleware.go    # 租户上下文中间件
│   │   ├── scope.go         # GORM 全局 Scope + Callback 防漏
│   │   └── model.go
│   ├── auth/                # 认证与权限
│   │   ├── jwt.go
│   │   ├── rbac.go
│   │   └── handler.go
│   ├── core/                # 核心业务
│   │   ├── employee/        # 员工管理
│   │   ├── department/      # 科室管理
│   │   ├── shift/           # 班次管理
│   │   ├── rule/            # 规则管理
│   │   ├── leave/           # 请假管理
│   │   ├── schedule/        # 排班管理
│   │   │   ├── step/        # 可复用 Step 池（LoadRules/FilterCandidates/PhaseZero/...）
│   │   │   ├── pipeline/    # Pipeline 编排（创建/调整/AI辅助）
│   │   │   ├── engine/      # 规则引擎（确定性）
│   │   │   └── handler.go
│   │   └── group/           # 分组管理
│   ├── ai/                  # AI 服务
│   │   ├── adapter/         # AI 提供商适配器（统一 HTTP 调用大模型）
│   │   ├── chat/            # 对话管理
│   │   ├── intent/          # 意图识别
│   │   └── quota/           # 调用配额
│   ├── admin/               # 平台管理
│   │   ├── tenant/          # 租户管理
│   │   ├── subscription/    # 订阅计费
│   │   └── monitoring/      # 运维监控
│   └── infra/               # 基础设施
│       ├── database/        # 数据库连接
│       ├── cache/           # Redis 缓存
│       ├── websocket/       # WebSocket
│       └── observability/   # 日志/指标/追踪
├── web/                     # Vue 3 前端（单一目录，非 submodule）
├── migrations/              # 数据库迁移脚本
├── deploy/                  # 部署配置
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── k8s/
└── docs/                    # 文档
```