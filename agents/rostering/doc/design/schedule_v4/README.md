# V4 排班系统设计文档

## 文档索引

```
schedule_v4/
├── README.md                           # 本文件：文档索引与全局概览
├── 00_design_principles.md             # 设计原则与开发规范
├── 01_architecture_overview.md         # 总体架构设计
├── 02_module_decomposition.md          # 模块拆分与依赖关系
├── 03_data_model.md                    # 数据模型与数据库设计
├── 04_rule_engine.md                   # 确定性规则引擎详细设计
├── 05_workflow.md                      # V4 排班工作流详细设计
├── 06_rule_parser.md                   # 规则语义化解析服务设计
├── 07_api_design.md                    # API 接口设计
├── 08_frontend.md                      # 前端改造设计
├── 09_migration.md                     # V3→V4 迁移方案
└── 10_implementation_plan.md           # 实施计划与任务拆分
```

## V4 核心理念（一句话）

> **LLM 只做"从合格候选人中选人"这一个决策，其余全部代码化。**

## V3 → V4 关键变化

| 维度 | V3 | V4 |
|------|-----|-----|
| 规则执行 | LLM 理解自然语言规则 | 代码精确执行结构化规则 |
| 预分析 | 3×LLM 并行调用（LLM-1/2/3） | 0×LLM，全部代码化 |
| 校验 | LLM 校验 | 代码校验 + 量化评分 |
| LLM 职责 | 理解规则 + 过滤人员 + 排班 + 校验 | 仅排班选人 |
| 单次排班LLM调用 | ~720 次（20班×7天） | ~160 次 |
| 规则模型 | 纯文本 RuleData | 结构化分类 + 方向性关联 |
| 班次排序 | SchedulingPriority 整数 | 拓扑排序依赖图 |
| 质量保证 | LLM 输出不稳定 | 代码前置校验 + 代码后置校验 |

## 代码目录规划

```
agents/rostering/
├── domain/model/
│   ├── rule_v4.go                 # V4 规则扩展模型
│   ├── rule_dependency.go         # 规则依赖关系模型
│   ├── rule_conflict.go           # 规则冲突关系模型
│   └── shift_dependency.go        # 班次依赖关系模型
├── domain/repository/
│   ├── rule_dependency_repository.go
│   ├── rule_conflict_repository.go
│   └── shift_dependency_repository.go
├── internal/
│   ├── engine/                    # ★ 确定性规则引擎（全新）
│   │   ├── engine.go              # 引擎入口
│   │   ├── candidate_filter.go    # 候选人过滤器
│   │   ├── rule_matcher.go        # 规则匹配器
│   │   ├── constraint_checker.go  # 约束检查器
│   │   ├── preference_scorer.go   # 偏好评分器
│   │   ├── schedule_validator.go  # 排班校验器
│   │   ├── dependency_resolver.go # 依赖解析器
│   │   └── types.go               # 类型定义
│   └── workflow/
│       └── schedule_v4/           # ★ V4 工作流（全新）
│           ├── main.go
│           ├── create/
│           │   ├── definition.go
│           │   ├── context.go
│           │   ├── actions.go
│           │   └── helpers.go
│           ├── executor/
│           │   ├── executor.go
│           │   ├── prompt_builder.go
│           │   └── types.go
│           └── utils/
│               ├── task_context.go
│               └── shift_task_context.go
services/management-service/
├── internal/service/
│   ├── rule_parser_service.go     # ★ 规则语义化解析服务（全新）
│   └── rule_validator_service.go  # ★ 规则验证服务（全新）
sdk/rostering/model/
├── rule.go                        # 扩展 Rule 模型（添加 Category 等字段）
└── rule_association.go            # 扩展 RuleAssociation（添加 Role 字段）
frontend/web/src/
├── pages/management/scheduling-rule/
│   ├── components/
│   │   ├── RuleInputDialog.vue    # 规则语义化录入
│   │   └── DependencyGraph.vue    # 依赖关系可视化
│   └── index.vue                  # 规则列表改造
```

## 阅读顺序建议

1. [设计原则](00_design_principles.md) — 理解设计哲学
2. [总体架构](01_architecture_overview.md) — 把握全局
3. [模块拆分](02_module_decomposition.md) — 理解模块边界
4. [数据模型](03_data_model.md) — 了解数据结构
5. [规则引擎](04_rule_engine.md) — 核心组件详细设计
6. [工作流](05_workflow.md) — 排班流程详细设计
7. [规则解析](06_rule_parser.md) — 规则录入侧设计
8. [API 设计](07_api_design.md) — 接口规范
9. [前端改造](08_frontend.md) — 前端变更
10. [迁移方案](09_migration.md) — 平滑过渡
11. [实施计划](10_implementation_plan.md) — 开发排期
