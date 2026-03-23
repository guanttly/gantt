# V4 评审报告（二）— P0 变更验证 + V4 排班流程评审

> **评审日期**: 2026-02-11  
> **评审范围**: P0 修复代码变更 + V4 排班工作流 / 规则引擎 / 执行器全流程  
> **结论**: P0 变更 **基本通过**（1 个 BUG）；V4 排班流程 **骨架已建但无法运行**

---

## 第一部分：P0 变更验证

### 总体判定：✅ 基本通过

P0 声称修改的 7 个文件全部验证，V4 字段的端到端数据链路已打通。

| # | 文件 | 状态 | 问题 |
|---|------|------|------|
| 1 | 领域模型 `scheduling_rule.go` | ✅ | 6 个 V4 字段全部存在，Filter 4 个筛选字段齐全 |
| 2 | Entity 层 `scheduling_rule_entity.go` | ✅ | GORM 字段 + Tag 正确 |
| 3 | Mapper 层 `scheduling_rule_mapper.go` | ✅ | 正反映射完整，Association Role 有默认值逻辑 |
| 4 | Handler `scheduling_rule_handler.go` | ✅ | Create/Update/List 均穿透 V4 字段，默认值正确 |
| 5 | Repository `scheduling_rule_repository.go` | 🐛 | `AddAssociations` 未映射 Role 字段 |
| 6 | SDK `rule.go` | ✅ | Rule/CreateReq/UpdateReq/ListReq 全部补齐 |
| 7 | 迁移 SQL | ✅ | 6 列 + 5 索引 + 回滚脚本 |
| 8 | MCP `create.go` / `list.go` | ✅ | InputSchema 含 V4 字段 + enum 约束 |

### 🐛 BUG-1：`AddAssociations` 丢失 Role 字段

**位置**: `services/management-service/internal/repository/scheduling_rule_repository.go` — `AddAssociations` 方法

**问题**: 手动构建 Entity 时遗漏了 `Role` 字段映射：

```go
// 当前代码（有BUG）
entities[i] = &entity.SchedulingRuleAssociationEntity{
    ID:              uuid.New().String(),
    OrgID:           orgID,
    RuleID:          ruleID,
    AssociationType: string(assoc.AssociationType),
    AssociationID:   assoc.AssociationID,
    // ❌ 缺失: Role: assoc.Role,
}
```

**影响**: 通过 `AddAssociations` API（独立添加关联，非创建规则时的内联关联）添加的关联，`role` 始终为数据库默认值 `"target"`，即使传入 `"source"` 或 `"reference"` 也会被忽略。

> 注意：创建规则时的内联 Association 通过 Mapper 映射，**不受此 Bug 影响**。

**修复**: 添加 `Role: assoc.Role` 行。

### ⚠️ 建议改进：缺少 SourceType 常量

领域模型有 `CategoryConstraint`/`CategoryPreference` 等分类常量和 `RoleTarget`/`RoleSource` 等角色常量，但 **缺少 SourceType 的对应常量**：

```go
// 建议添加（与已有的 Category/Role 常量风格一致）
const (
    SourceTypeManual    = "manual"
    SourceTypeLLMParsed = "llm_parsed"
    SourceTypeMigrated  = "migrated"
)
```

当前各层使用字符串字面量 `"manual"` 等，分散在 Handler、Parser 等处，存在拼写不一致风险。

---

## 第二部分：V4 排班流程评审

### 总体判定：⚠️ 骨架已建，但完全无法运行

| 模块 | 完成度 | 判定 |
|------|--------|------|
| V4 工作流状态机定义 | ~70% | 状态转换完整，但 action 全为 TODO |
| V4 工作流注册 & 路由 | 0% | 🔴 致命：未导入、未路由 |
| 确定性规则引擎 | ~60% | 核心流程可走通，多个子模块为存根 |
| V4 执行器 | ~30% | 骨架在，核心排班逻辑为 TODO |
| 规则组织服务 (management-service) | ~90% | ✅ 最完整的模块 |
| 规则解析服务 (management-service) | ~85% | ✅ 三层验证器完整实现 |

---

### 2.1 🔴 致命问题：V4 工作流未接入主流程

#### 问题 A：工作流未注册

**位置**: `agents/rostering/setup.go`

```go
// 当前 import
_ "jusha/agent/rostering/internal/workflow/schedule_v2"
_ "jusha/agent/rostering/internal/workflow/schedule_v3"
// ❌ 缺失: _ "jusha/agent/rostering/internal/workflow/schedule_v4"
```

V4 工作流目录 `agents/rostering/internal/workflow/schedule_v4/` 存在，`init()` 函数已写好自动注册逻辑，但因为没有被 import，**工作流定义不会被注册到引擎中**。

#### 问题 B：版本路由无 V4 分支

**位置**: `agents/rostering/internal/workflow/schedule/init.go` — `InitializeWorkflowWithVersion`

```go
switch version {
case "v3": → WorkflowScheduleCreateV3
case "v2": → WorkflowScheduleCreateV2
default:   → WorkflowScheduleCreateV2
// ❌ 缺失: case "v4": → WorkflowScheduleCreateV4
}
```

即使用户选择 V4 版本，也会 fallback 到 V2。

#### 修复

```go
// setup.go 添加 import
_ "jusha/agent/rostering/internal/workflow/schedule_v4"

// init.go 添加 case
case "v4":
    return session.WorkflowInitResult{
        WorkflowName: string(schedule.WorkflowScheduleCreateV4),
        Version:      "v4",
    }, true
```

---

### 2.2 🔴 所有工作流 Action 为空壳

**位置**: `agents/rostering/internal/workflow/schedule_v4/create/action.go`

状态机定义（`definition.go`）的约 20 个状态转换结构完整，但所有 action 函数只有日志 + `return nil`：

| Action 函数 | 设计职责 | 实现状态 |
|------------|---------|---------|
| `actInfoCollecting` | 收集排班基础信息 | ❌ TODO |
| `actInfoCollected` | 信息收集完成处理 | ❌ TODO |
| `actPeriodConfirmed` | 排班周期确认 | ❌ TODO |
| `actShiftConfirmed` | 班次确认 | ❌ TODO |
| `actStaffCountConfirmed` | 人数确认 | ❌ TODO |
| `actPersonalNeedsConfirmed` | 个人需求确认 | ❌ TODO |
| `actRuleOrganization` | **V4 核心：规则组织** | ❌ TODO |
| `actScheduling` | **V4 核心：执行排班** | ❌ TODO |
| `actValidation` | **V4 核心：确定性校验** | ❌ TODO |
| `actReviewConfirmed` | 审核确认 | ❌ TODO |
| `actUserModification` | 用户修改处理 | ❌ TODO |
| `actCompleted` | 完成 | ✅（仅日志） |

> **影响**: 即使注册并路由了 V4 工作流，进入任何状态都只会打日志然后立即转移到下一个状态，不会执行任何业务逻辑。

---

### 2.3 确定性规则引擎评审

**目录**: `agents/rostering/internal/engine/`

#### ✅ 已实现的组件

| 组件 | 文件 | 说明 |
|------|------|------|
| **RuleEngine 入口** | `rule_engine.go` | `PrepareSchedulingContext()` + `ValidateSchedule()` 流程完整 |
| **CandidateFilter** | `candidate_filter.go` | 基于请假/固定排班过滤候选人，逻辑完整 |
| **RuleMatcher** | `rule_matcher.go` | 按 Category/SubCategory 分类规则，`isRuleApplicable()` 正常工作 |
| **ConstraintChecker** | `constraint_checker.go` | `maxCount`/`consecutiveMax`/`minRestDays`/`exclusive` 已实现 |
| **Types** | `types.go` | `SchedulingContext`/`LLMBrief` 等类型定义完整 |

#### ⚠️ 存根/不完整的组件

| 组件 | 文件 | 问题 |
|------|------|------|
| **PreferenceScorer** | `preference_scorer.go` | `computeRulePreferenceScore()` **硬编码返回 0.5**，未实现真正的偏好评分 |
| **ScheduleValidator** | `schedule_validator.go` | `checkConstraintRule()` 和 `checkPreferenceRule()` **返回 nil**，未实现实际校验 |
| **ConstraintChecker** | `constraint_checker.go` | `forbidden_day` 检查为 **TODO** |

#### ❌ 缺失的组件

| 设计组件 | 说明 |
|---------|------|
| **DependencyResolver** | 设计要求 `dependency_resolver.go` 用于在排班前解析规则依赖链，**文件不存在** |
| **ValidateGlobal()** | 设计要求全局校验方法（公平性报告/工时均衡等），**未实现** |
| **IRuleEngine 接口** | 无抽象接口定义，不可 mock/测试 |

#### 🟡 设计偏离

**RuleMatcher 不读 Category 字段**:

```go
// rule_matcher.go 当前实现
// 通过 RuleType 推断分类，而非读取 Category 字段
func (m *RuleMatcher) inferCategory(ruleType string) string {
    switch ruleType {
    case "exclusive", "maxCount", "forbidden_day", "required_together", "periodic":
        return "constraint"
    // ...
    }
}
```

SDK Rule 模型 **已有** `Category` 字段（P0 已补齐），但引擎仍通过 RuleType 硬编码推断。应直接读取 `rule.Category`，仅在为空时 fallback 推断。

**DependencyAnalyzer 依赖中文文本匹配**:

```go
// dependency_analyzer.go
if strings.Contains(rule.Description, "来自") || strings.Contains(rule.Description, "必须从") {
    // 判定为依赖关系
}
```

设计中应使用 `RuleAssociation.Role` 字段（`source`/`reference`）判断依赖类型，而非解析中文描述。SDK 已有 `Role` 字段，但代码注释写着 `// TODO: 如果SDK model有Role字段`。

---

### 2.4 V4 执行器评审

**文件**: `agents/rostering/internal/workflow/schedule_v4/executor/v4_executor.go`

#### 结构

```
V4Executor
├── RuleOrganizer → 调用 management-service 组织规则（✅ 已实现）
├── RuleEngine    → 确定性引擎准备上下文（✅ 已实现）
├── 按班次依赖顺序逐个排班（✅ 流程框架完整）
└── selectStaff() → ❌ 硬编码取前 N 个候选人
```

#### 核心问题：selectStaff 未接入 LLM

```go
// v4_executor.go — selectStaff
// TODO: 实际应该调用LLM，传入ctx.LLMBrief
// 这里只是简单选择前N个
for i := 0; i < ctx.RequiredCount && i < len(candidates); i++ {
    selected = append(selected, candidates[i].StaffID)
}
```

V4 设计的核心思想是：规则引擎做确定性过滤/评分 → 生成 LLMBrief → **一次 LLM 调用**做最终选人决策。当前 LLMBrief 已正确构建，但 LLM 调用未接入。

---

### 2.5 Management-Service 侧（✅ 较完整）

#### ✅ RuleOrganizerService

**文件**: `services/management-service/internal/service/rule_organizer_service.go`

- 从数据库加载规则、依赖、冲突
- 规则按 Category 分组（constraint / preference / dependency）
- 拓扑排序（规则依赖 + 班次依赖）
- 已注册到 DI 容器

#### ✅ RuleParserService + 三层验证器

**文件**: `services/management-service/internal/service/rule_parser_service.go` + `rule_parser_validator.go`

| 功能 | 状态 | 说明 |
|------|------|------|
| LLM 语义化解析 | ✅ | 完整的 prompt + 响应解析 |
| 三层验证器 | ✅ | 结构/语义/业务三层验证 |
| 名称匹配 (Name→ID) | ✅ | validator 中实现关联对象存在性检查 |
| `checkConflictsWithExisting` | ✅ | 检查名称重复 + 互斥冲突 + 资源冲突 |
| 依赖/冲突关系保存 | ✅ | 调用 DependencyRepo / ConflictRepo |

> **注意**: 上一份评审报告中标注的 "`checkConflictsWithExisting` 为 TODO" **已不准确** — 该方法已完整实现（含名称重复检查、互斥冲突检查、资源冲突检查）。

---

### 2.6 V4 状态机设计 vs 实现对比

| 设计状态 | 实现状态 | 差异 |
|---------|---------|------|
| `v4.init` | `_schedule_v4_create_init_` | ✅ 对应 |
| `v4.info_collecting` | `_schedule_v4_create_info_collecting_` | ✅ 对应 |
| `v4.confirm_period` | `_schedule_v4_create_confirm_period_` | ✅ 对应 |
| `v4.confirm_shifts` | `_schedule_v4_create_confirm_shifts_` | ✅ 对应 |
| `v4.confirm_staff_count` | `_schedule_v4_create_confirm_staff_count_` | ✅ 对应 |
| `v4.personal_needs` | `_schedule_v4_create_personal_needs_` | ✅ 对应 |
| `v4.rule_validation` | `_schedule_v4_create_rule_organization_` | ⚠️ 设计叫"验证"，实现叫"组织" |
| **`v4.plan_generation`** | ❌ 缺失 | 🔴 设计有独立的计划生成状态 |
| **`v4.plan_review`** | ❌ 缺失 | 🔴 |
| **`v4.progressive_task`** | 简化为 `_scheduling_` | ⚠️ 设计是渐进式多任务，实现为单一排班 |
| **`v4.task_review`** | ❌ 缺失 | 🔴 |
| **`v4.waiting_adjustment`** | ❌ 缺失 | 🔴 |
| **`v4.task_failed`** | ❌ 缺失 | 🔴 |
| `v4.global_validation` | `_schedule_v4_create_validation_` | ⚠️ 合并了全局校验+人工审核 |
| **`v4.global_review_manual`** | ❌ 缺失 | 🔴 |
| `v4.confirm_saving` | `_schedule_v4_create_review_` | ⚠️ 合并 |
| `v4.completed` | `_schedule_v4_create_completed_` | ✅ 对应 |

> **总结**: 实现有 12 个状态，设计有 ~20 个状态。实现将设计的"渐进式多任务 → 逐任务审核 → 失败重试 → 人工调整"简化为了"一次排班 → 一次审核"的线性流程。这导致 **V4 设计的核心差异化特性（渐进式排班 + 单班次级别的审核和重试）丢失**。

---

## 第三部分：综合问题清单

### 🔴 致命（V4 完全不可运行）

| # | 问题 | 位置 | 修复难度 |
|---|------|------|---------|
| 1 | V4 工作流未 import，init() 不执行 | `setup.go` | 1 行 |
| 2 | 版本路由无 `case "v4"` | `workflow/schedule/init.go` | 5 行 |
| 3 | 12 个 workflow action 全为 TODO | `schedule_v4/create/action.go` | 5-8 天 |
| 4 | `selectStaff` 未接入 LLM | `executor/v4_executor.go` | 1-2 天 |
| 5 | `AddAssociations` 丢失 Role 字段 | `repository/scheduling_rule_repository.go` | 1 行 |

### 🟡 高（功能不完整）

| # | 问题 | 位置 | 修复难度 |
|---|------|------|---------|
| 6 | DependencyResolver 缺失 | `engine/` 需新建 | 2 天 |
| 7 | ValidateGlobal() 缺失 | `engine/rule_engine.go` | 1 天 |
| 8 | ScheduleValidator 全为空壳 | `engine/schedule_validator.go` | 1 天 |
| 9 | PreferenceScorer 硬编码 0.5 | `engine/preference_scorer.go` | 1 天 |
| 10 | `forbidden_day` 约束检查 TODO | `engine/constraint_checker.go` | 0.5 天 |
| 11 | RuleMatcher 不读 Category 字段 | `engine/rule_matcher.go` | 0.5 天 |
| 12 | DependencyAnalyzer 用中文文本匹配 | `executor/dependency_analyzer.go` | 0.5 天 |
| 13 | 缺少 SourceType 常量定义 | `domain/model/scheduling_rule.go` | 5 行 |

### 🟠 中（设计偏离）

| # | 问题 | 说明 |
|---|------|------|
| 14 | 状态机简化（12 vs 20 状态） | 丢失渐进式排班/单班次审核/失败重试能力 |
| 15 | Agent 侧无 V4 domain model | `rule_v4.go`/`rule_dependency.go` 等计划文件未创建 |
| 16 | RuleEngine 无接口定义 | 不可 mock / 单测受限 |

---

## 第四部分：改进路线图

### Phase 1：V4 可运行（3 天）

```
Day 1:
  ├── 修复 BUG: AddAssociations 添加 Role 映射 [1行]
  ├── setup.go 添加 V4 workflow import [1行]
  ├── init.go 添加 case "v4" 路由 [5行]
  ├── 添加 SourceType 常量 [5行]
  └── RuleMatcher 改为读取 Category 字段 [20行]

Day 2-3:
  ├── 实现 V4 核心 action: actRuleOrganization
  │   → 调用 management-service RuleOrganizerService
  ├── 实现 V4 核心 action: actScheduling
  │   → 调用 V4Executor.ExecuteScheduling()
  ├── 实现 V4 核心 action: actValidation
  │   → 调用 RuleEngine.ValidateSchedule()
  └── selectStaff 接入 LLM 调用
```

### Phase 2：引擎完善（4 天）

```
Day 4-5:
  ├── 实现 DependencyResolver
  ├── 实现 ScheduleValidator (checkConstraintRule/checkPreferenceRule)
  ├── 实现 PreferenceScorer 真正的评分算法
  ├── 实现 forbidden_day 约束检查
  └── DependencyAnalyzer 改用 Role 字段

Day 6-7:
  ├── 实现 ValidateGlobal() + 公平性报告
  ├── 定义 IRuleEngine 接口
  ├── 创建 Agent 侧 V4 domain model
  └── 补充信息收集阶段 action (actInfoCollecting 等)
      → 可复用 V3 的信息收集逻辑
```

### Phase 3：状态机完善（3 天）

```
Day 8-10:
  ├── 补充渐进式排班状态 (plan_generation/plan_review)
  ├── 补充单班次审核 (task_review)
  ├── 补充失败重试 (task_failed/waiting_adjustment)
  ├── 补充人工审核 (global_review_manual)
  └── 端到端测试
```

### 工时总结

| Phase | 范围 | 工时 |
|-------|------|------|
| Phase 1 | V4 可运行 | 3 天 |
| Phase 2 | 引擎完善 | 4 天 |
| Phase 3 | 状态机完善 | 3 天 |
| **合计** | | **~10 人天** |

---

## 附录：上一份评审报告勘误

| 原评审结论 | 更正 |
|-----------|------|
| `checkConflictsWithExisting` 为 TODO | ❌ **不准确** — 已完整实现（含名称重复/互斥冲突/资源冲突检查） |
| 三层验证器未实现 | ❌ **不准确** — `rule_parser_validator.go` 中已实现完整的三层验证（结构/语义/业务） |
| 名称模糊匹配器缺失 | ⚠️ **部分不准确** — validator 中已实现关联对象存在性检查，但不是独立的 NameMatcher 模块 |
| MCP 工具无 V4 字段 | ❌ **不准确** — P0 已修复，`create.go`/`list.go` 的 InputSchema 已包含 V4 字段 |

> 以上差异说明 P0 修复后部分问题已解决，或前次评审时遗漏了已有实现。本次评审已核实代码实际状态。
