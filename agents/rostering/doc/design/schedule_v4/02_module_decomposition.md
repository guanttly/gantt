# 02. 模块拆分与依赖关系

## 1. 模块总览

V4 系统由 6 个核心模块组成：

```
┌─────────────────────────────────────────────────────────┐
│                    模块依赖全景图                         │
│                                                         │
│  M1 数据模型层                                           │
│  ├── sdk/rostering/model/  (SDK 模型扩展)                │
│  ├── agents/rostering/domain/model/  (Agent 领域模型)     │
│  └── agents/rostering/domain/repository/  (仓储接口)     │
│       │                                                  │
│       ▼                                                  │
│  M2 确定性规则引擎                                        │
│  └── agents/rostering/internal/engine/                   │
│       │                                                  │
│       ├──────────────────────────┐                       │
│       ▼                         ▼                       │
│  M3 V4排班工作流              M4 规则解析服务             │
│  └── agents/.../schedule_v4/  └── services/.../          │
│       │                         rule_parser_service.go   │
│       │                         │                       │
│       ▼                         ▼                       │
│  M5 API层（Agent HTTP + Management Service API）         │
│       │                                                  │
│       ▼                                                  │
│  M6 前端                                                 │
│  └── frontend/web/src/                                   │
└─────────────────────────────────────────────────────────┘
```

## 2. M1: 数据模型层

### 2.1 职责

- 定义 V4 规则扩展字段（Category, SubCategory, Role）
- 定义规则依赖/冲突/班次依赖关系模型
- 定义仓储接口
- 定义数据库 DDL

### 2.2 文件清单

| 文件 | 类型 | 说明 |
|------|------|------|
| `sdk/rostering/model/rule.go` | 修改 | Rule 新增 Category/SubCategory/OriginalRuleID 字段 |
| `sdk/rostering/model/rule.go` | 修改 | RuleAssociation 新增 Role 字段 |
| `agents/rostering/domain/model/rule_v4.go` | 新建 | V4 规则分类常量、RuleCategory 枚举 |
| `agents/rostering/domain/model/rule_dependency.go` | 新建 | RuleDependency 模型 |
| `agents/rostering/domain/model/rule_conflict.go` | 新建 | RuleConflict 模型 |
| `agents/rostering/domain/model/shift_dependency.go` | 新建 | ShiftDependency 模型 |
| `agents/rostering/domain/repository/rule_dependency_repository.go` | 新建 | 依赖关系仓储接口 |
| `agents/rostering/domain/repository/rule_conflict_repository.go` | 新建 | 冲突关系仓储接口 |
| `agents/rostering/domain/repository/shift_dependency_repository.go` | 新建 | 班次依赖仓储接口 |

### 2.3 对外接口

```go
// 被 M2/M3/M4/M5 依赖的核心类型
type RuleCategory string      // "constraint" / "preference" / "dependency"
type RuleSubCategory string   // "forbid" / "limit" / "must" / "prefer" / "source" / ...
type AssociationRole string   // "target" / "source" / "reference"

type RuleDependency struct { ... }
type RuleConflict struct { ... }
type ShiftDependency struct { ... }
```

### 2.4 开发约束

- SDK model 字段新增必须有 `omitempty` 标签（向后兼容）
- 新增字段在数据库中必须有默认值
- 不修改任何已有字段的类型或 JSON 名

---

## 3. M2: 确定性规则引擎

### 3.1 职责

- 候选人过滤（替代 LLM-1）
- 规则匹配（替代 LLM-2）
- 约束检查（替代 LLM-3）
- 偏好评分
- 排班校验（替代 LLM-5）
- 依赖关系解析 + 拓扑排序

### 3.2 文件清单

| 文件 | 说明 |
|------|------|
| `internal/engine/types.go` | 引擎类型定义（SchedulingInput, SchedulingContext, LLMBrief 等） |
| `internal/engine/engine.go` | 引擎入口（IRuleEngine 接口 + 实现） |
| `internal/engine/candidate_filter.go` | 候选人过滤器 |
| `internal/engine/rule_matcher.go` | 规则匹配器 |
| `internal/engine/constraint_checker.go` | 约束检查器 |
| `internal/engine/preference_scorer.go` | 偏好评分器 |
| `internal/engine/schedule_validator.go` | 排班校验器 |
| `internal/engine/dependency_resolver.go` | 依赖解析 + 拓扑排序 |
| `internal/engine/*_test.go` | 各组件单元测试 |

### 3.3 对外接口

```go
// IRuleEngine 规则引擎主接口
type IRuleEngine interface {
    // PrepareSchedulingContext 为单个班次+日期准备排班上下文（替代 LLM-1/2/3）
    PrepareSchedulingContext(ctx context.Context, input *SchedulingInput) (*SchedulingContext, error)
    
    // ValidateSchedule 校验排班结果（替代 LLM-5）
    ValidateSchedule(ctx context.Context, result *ScheduleResult, rules *MatchedRules, globalDraft *ScheduleDraft) (*ValidationResult, error)
    
    // ValidateGlobal 全局校验（跨班次/跨日期）
    ValidateGlobal(ctx context.Context, draft *ScheduleDraft, allRules []*ClassifiedRule) (*GlobalValidationResult, error)
}

// IDependencyResolver 依赖解析接口
type IDependencyResolver interface {
    // ResolveShiftOrder 计算班次执行顺序（拓扑排序）
    ResolveShiftOrder(shifts []*Shift, deps []*ShiftDependency) ([]string, error)
    
    // ResolveRuleOrder 计算规则执行顺序
    ResolveRuleOrder(rules []*ClassifiedRule, deps []*RuleDependency) ([]string, error)
    
    // DetectCircularDependency 检测循环依赖
    DetectCircularDependency(deps []DependencyEdge) ([][]string, error)
}
```

### 3.4 依赖

- 依赖 M1（数据模型）
- 不依赖 M3/M4/M5/M6
- 不依赖 `pkg/ai/`

### 3.5 开发约束

- **零 LLM 调用**
- 所有方法必须是确定性的（相同输入 → 相同输出）
- 性能要求：100 条规则 × 50 人 × 7 天 < 500ms
- 每个公开方法必须有测试

---

## 4. M3: V4 排班工作流

### 4.1 职责

- 定义 V4 工作流状态机
- 管理排班生命周期
- 集成规则引擎和 LLM
- 管理 L1/L2/L3 上下文

### 4.2 文件清单

| 文件 | 说明 |
|------|------|
| `schedule_v4/main.go` | 包入口（注册工作流） |
| `schedule_v4/create/definition.go` | 工作流状态机定义 |
| `schedule_v4/create/context.go` | L1 上下文定义（CreateV4Context） |
| `schedule_v4/create/actions.go` | 状态转换动作 |
| `schedule_v4/create/helpers.go` | 辅助函数 |
| `schedule_v4/executor/executor.go` | V4 执行器 |
| `schedule_v4/executor/prompt_builder.go` | 结构化 Prompt 构建 |
| `schedule_v4/executor/types.go` | 执行器类型 |
| `schedule_v4/utils/task_context.go` | L2 上下文 |
| `schedule_v4/utils/shift_task_context.go` | L3 上下文 |

### 4.3 对外接口

```go
// IProgressiveTaskExecutorV4 V4 执行器接口
type IProgressiveTaskExecutorV4 interface {
    ExecuteTask(ctx context.Context, taskCtx *CoreV4TaskContext) (*TaskResult, error)
    SetProgressCallback(callback ProgressCallback)
}
```

### 4.4 依赖

- 依赖 M1（数据模型）
- 依赖 M2（规则引擎）
- 依赖 `pkg/ai/`（仅 LLM 排班选人）
- 依赖 `pkg/workflow/engine/`

### 4.5 开发约束

- V3 状态名前缀 `CreateV3State*`，V4 前缀 `CreateV4State*`（避免冲突）
- V4 工作流名 `workflow.schedule.create.v4`
- executor 中只有 `prompt_builder.go` 涉及 LLM 调用
- L3 上下文必须包含 `LLMBrief`（由 RuleEngine 生成）

---

## 5. M4: 规则解析服务

### 5.1 职责

- LLM 语义化规则解析
- 三层验证（结构化/回译/模拟）
- 批量规则保存
- 依赖/冲突关系自动检测

### 5.2 文件清单

| 文件 | 说明 |
|------|------|
| `services/management-service/internal/service/rule_parser_service.go` | 规则解析服务 |
| `services/management-service/internal/service/rule_validator_service.go` | 规则验证服务（三层验证） |
| `services/management-service/internal/service/rule_back_translator.go` | 回译器 |

### 5.3 依赖

- 依赖 M1（数据模型）
- 依赖 M2（规则引擎，用于模拟验证）
- 依赖 `pkg/ai/`（LLM 解析）

---

## 6. M5: API 层

### 6.1 职责

- 规则解析 API
- 规则批量保存 API
- 规则组织查询 API
- 依赖/冲突关系 CRUD API

### 6.2 文件清单

| 文件 | 说明 |
|------|------|
| `services/management-service/internal/handler/rule_v4_handler.go` | V4 规则 API Handler |
| `mcp-servers/rostering/tool/rule_v4/` | V4 规则 MCP Tool（如需） |

### 6.3 依赖

- 依赖 M1, M4

---

## 7. M6: 前端

### 7.1 职责

- 规则语义化录入对话框
- 解析结果预览
- 规则依赖关系可视化
- 规则列表增加分类筛选

### 7.2 文件清单

| 文件 | 说明 |
|------|------|
| `frontend/web/src/pages/management/scheduling-rule/components/RuleInputDialog.vue` | 语义化录入 |
| `frontend/web/src/pages/management/scheduling-rule/components/RuleParsePreview.vue` | 解析预览 |
| `frontend/web/src/pages/management/scheduling-rule/components/DependencyGraph.vue` | 依赖可视化 |
| `frontend/web/src/api/rule-v4.ts` | V4 规则 API 封装 |

### 7.3 依赖

- 依赖 M5（API 接口）

---

## 8. 模块间接口契约

### 8.1 M1 → M2: 规则数据

```go
// M2 从 M1 获取的数据
type SchedulingInput struct {
    AllStaff          []*model.Employee
    AllRules          []*model.Rule                  // 包含 Category, SubCategory
    PersonalNeeds     map[string][]*model.PersonalNeed
    FixedAssignments  []model.CtxFixedShiftAssignment
    CurrentDraft      *model.ScheduleDraft
    ShiftID           string
    Date              time.Time
    RequiredCount     int
    // V4 新增
    RuleDependencies  []*model.RuleDependency
    RuleConflicts     []*model.RuleConflict
    ShiftDependencies []*model.ShiftDependency
}
```

### 8.2 M2 → M3: 引擎输出

```go
// M3 从 M2 获取的数据
type SchedulingContext struct {
    ShiftID            string
    Date               time.Time
    RequiredCount      int
    MatchedRules       *MatchedRules
    EligibleCandidates []*CandidateStatus
    ExcludedCandidates []*CandidateStatus
    ExclusionReasons   []*ExclusionRecord
    ConstraintDetails  []*ConstraintDetail
    PreferenceScores   *PreferenceScoreResult
    LLMBrief           *LLMBrief               // 给 LLM 的结构化摘要
}
```

### 8.3 M3 → LLM: 结构化 Prompt

```
## 排班任务
- 班次: {ShiftName} ({StartTime}~{EndTime})
- 日期: {Date}
- 需求人数: {Count}

## 合格候选人
| ID | 姓名 | 推荐度 | 约束余量 | 本周已排 | 备注 |
...

## 必须遵守的约束
- [R1] ...

## 偏好参考
- ...
```

### 8.4 M4 → M1: 解析结果

```go
// M4 输出到 M1 的数据
type ParsedRule struct {
    Name, Category, SubCategory string
    RuleType                    string
    MaxCount, ConsecutiveMax    *int
    Associations                []RuleAssociationWithRole
    Dependencies                []RuleDependencyDef
    Conflicts                   []RuleConflictDef
}
```
