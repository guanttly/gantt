# 00. 设计原则与开发规范

## 1. 核心设计原则

### 原则一：确定性优先（Determinism First）

> **能用代码精确计算的，绝不交给 LLM。**

| 操作 | V3 做法 | V4 做法 | 理由 |
|------|---------|---------|------|
| 请假人员过滤 | LLM-1 | 代码：日期区间匹配 | 纯数据运算，结果确定 |
| 规则关联匹配 | LLM-2 | 代码：Association + Category 查询 | 结构化数据查询 |
| 频次/连班检查 | LLM-3 | 代码：计数 + 比较 | 纯数学运算 |
| 排班校验 | LLM-5 | 代码：逐条检查 | 规则已结构化 |
| 排班选人 | LLM-4 | **保留 LLM** | 需要综合权衡 |

**判断标准**：如果一个操作的输入和规则都是结构化的，且存在明确的对错判断标准，则必须代码化。

### 原则二：LLM 最小职责（Minimal LLM Scope）

> **LLM 只回答一个问题："从这些合格候选人中，选哪几个最好？"**

LLM 不再需要：
- ❌ 理解规则文本（引擎已检查）
- ❌ 过滤不合格人员（引擎已过滤）
- ❌ 校验排班结果（引擎已校验）
- ❌ 解析任务目标（代码已解析）

LLM 只需要：
- ✅ 从合格候选人表中选择指定数量的人
- ✅ 参考偏好评分和约束余量信息
- ✅ 给出选择理由（用于 debug）

### 原则三：结构化传递（Structured Communication）

> **任何传递给 LLM 的信息，必须是结构化格式，不允许自然语言规则文本。**

```
❌ V3: "1. 夜班连续限制: 夜班最多连续2天，之后必须休息至少1天"
✅ V4: "| S1 | 张三 | 0.9 | 2/5 | 0 | 偏好周末休息 |"（表格化候选人）
```

### 原则四：录入时解析，排班时执行（Parse Once, Execute Many）

> **规则解析在录入阶段完成，排班时直接使用结构化数据。**

- 录入：用户输入自然语言 → LLM 解析 → 三层验证 → 结构化存储
- 排班：加载结构化规则 → 代码引擎执行 → 无 LLM 参与

### 原则五：可观测性（Observability）

> **每个排除决策都必须有明确的原因链。**

```go
// ❌ V3: 直接从候选列表中移除，LLM 不知道原因
// ✅ V4: 记录排除原因，并传递给 LLM 作为参考
type ExclusionRecord struct {
    StaffID    string
    StaffName  string
    Reason     string   // "请假: 2026-02-12~2026-02-14"
    RuleID     string   // 触发排除的规则ID（如有）
    RuleName   string
}
```

### 原则六：渐进兼容（Progressive Compatibility）

> **V4 与 V3 共存，通过配置切换，不破坏现有功能。**

- V3 工作流保留，不做任何修改
- V4 工作流独立注册
- 用户通过设置选择 V3 或 V4
- 数据模型向后兼容（新增字段有默认值）

---

## 2. 代码规范

### 2.1 包命名与组织

```
agents/rostering/internal/engine/       # 确定性规则引擎
agents/rostering/internal/workflow/schedule_v4/  # V4 工作流
```

- 引擎包（`engine`）与工作流包（`schedule_v4`）解耦
- 引擎不依赖工作流，工作流依赖引擎
- 引擎不依赖 AI 包（`pkg/ai`）

### 2.2 类型定义规范

**使用 SDK model 作为基础**：
```go
// ✅ 正确：扩展 SDK model
type RuleV4 struct {
    sdk_model.Rule
    Category    string `json:"category"`
    SubCategory string `json:"subCategory"`
}

// ❌ 错误：重新定义完整结构
type RuleV4 struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    // ... 重复定义
}
```

**引擎内部类型独立定义**：
```go
// 引擎内部类型，不直接暴露 domain model
package engine

type CandidateStatus struct {
    StaffID     string
    IsEligible  bool
    Violations  []*RuleViolation
    // ...
}
```

### 2.3 接口定义规范

**每个模块提供接口**：
```go
// 规则引擎接口（供工作流调用）
type IRuleEngine interface {
    PrepareSchedulingContext(ctx context.Context, input *SchedulingInput) (*SchedulingContext, error)
    ValidateSchedule(ctx context.Context, schedule *ScheduleResult, rules *MatchedRules) (*ValidationResult, error)
}

// 依赖解析器接口
type IDependencyResolver interface {
    ResolveShiftOrder(ctx context.Context, shifts []*Shift, deps []*ShiftDependency) ([]string, error)
}
```

### 2.4 错误处理规范

```go
// 使用 domain error 包
import "jusha/mcp/pkg/errors"

// 规则引擎错误
var (
    ErrCircularDependency  = errors.New("CIRCULAR_DEPENDENCY", "规则存在循环依赖")
    ErrNoEligibleCandidate = errors.New("NO_ELIGIBLE_CANDIDATE", "无合格候选人")
    ErrRuleConflict        = errors.New("RULE_CONFLICT", "规则冲突")
)
```

### 2.5 日志规范

```go
// 结构化日志，包含完整上下文
e.logger.Info("Constraint check completed",
    "shiftID", shiftID,
    "date", date,
    "totalCandidates", len(candidates),
    "eligible", len(result.EligibleCandidates),
    "excluded", len(result.ExcludedCandidates),
    "duration", time.Since(startTime),
)
```

### 2.6 测试规范

```
engine/
├── engine.go
├── engine_test.go              # 集成测试
├── candidate_filter.go
├── candidate_filter_test.go    # 单元测试
├── constraint_checker.go
├── constraint_checker_test.go  # 单元测试（每种规则类型一个 test case）
├── testdata/                   # 测试数据
│   ├── rules_basic.json
│   ├── rules_complex.json
│   └── expected_results.json
```

**测试要求**：
- 每种规则类型（`maxCount`/`consecutiveMax`/`exclusive`...）至少 3 个 test case
- 边界值测试（0, 1, max, max+1）
- 组合规则测试（多条规则同时生效）
- 循环依赖检测测试

---

## 3. V4 与 V3 代码共存规则

### 3.1 共享组件（不修改）

| 组件 | 路径 | V4 使用方式 |
|------|------|-----------|
| SDK Model | `sdk/rostering/model/` | 扩展（新增字段），不修改已有字段 |
| Domain Model | `agents/rostering/domain/model/` | 新增 V4 专用文件，不修改已有文件 |
| 工作流引擎 | `pkg/workflow/engine/` | 直接使用 |
| AI Provider | `pkg/ai/` | V4 executor 内使用 |
| Domain Service | `agents/rostering/domain/service/` | 扩展接口（新增方法） |

### 3.2 V4 独有组件（新建）

| 组件 | 路径 | 说明 |
|------|------|------|
| 规则引擎 | `agents/rostering/internal/engine/` | 全新包 |
| V4 工作流 | `agents/rostering/internal/workflow/schedule_v4/` | 全新包 |
| 规则解析服务 | `services/management-service/internal/service/rule_parser_service.go` | 全新文件 |

### 3.3 需要扩展的组件

| 组件 | 变更 |
|------|------|
| `sdk/rostering/model/rule.go` | Rule 结构体新增 `Category`、`SubCategory`、`OriginalRuleID` 字段 |
| `sdk/rostering/model/rule.go` | RuleAssociation 结构体新增 `Role` 字段 |
| `agents/rostering/domain/service/rostering.go` | IRosteringService 新增依赖/冲突关系查询方法 |
| `agents/rostering/internal/wiring/` | 注册 V4 工作流和规则引擎 |

---

## 4. Agent 开发指引

本文档体系为多 agent 并行开发设计。每个 agent 负责一个模块：

| Agent | 负责模块 | 入口文档 | 依赖 |
|-------|---------|---------|------|
| Agent-1 | 数据模型扩展 | [03_data_model.md](03_data_model.md) | 无 |
| Agent-2 | 确定性规则引擎 | [04_rule_engine.md](04_rule_engine.md) | Agent-1 |
| Agent-3 | V4 工作流 | [05_workflow.md](05_workflow.md) | Agent-1, Agent-2 |
| Agent-4 | 规则解析服务 | [06_rule_parser.md](06_rule_parser.md) | Agent-1 |
| Agent-5 | API 层 | [07_api_design.md](07_api_design.md) | Agent-1, Agent-4 |
| Agent-6 | 前端改造 | [08_frontend.md](08_frontend.md) | Agent-5 |

**并行开发建议**：
- Agent-1 最先开始（数据模型是所有模块的基础）
- Agent-2 和 Agent-4 可以在 Agent-1 完成后并行
- Agent-3 在 Agent-2 完成后开始
- Agent-5 和 Agent-6 可以在接口定义确认后并行

---

## 5. 质量门禁

### 5.1 代码提交前

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 所有规则类型有测试用例
- [ ] 无 `any` 类型（`map[string]any` 不允许出现在新代码中）
- [ ] 无直接调用 LLM 做确定性计算
- [ ] 日志包含完整上下文

### 5.2 模块完成前

- [ ] 接口文档完整
- [ ] 与上下游模块的集成测试通过
- [ ] 性能基准测试（规则引擎: 100条规则×50人 < 100ms）
- [ ] V3 回归测试通过（确保未破坏已有功能）
