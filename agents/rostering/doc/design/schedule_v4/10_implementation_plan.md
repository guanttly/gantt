# 10. 实施计划与任务拆分

## 1. Agent 分工总览

| Agent | 职责 | 核心交付物 | 预估工期 |
|-------|------|-----------|---------|
| Agent-1 | 数据模型 & SDK 扩展 | SDK model 扩展、DDL、Repository | 3天 |
| Agent-2 | 确定性规则引擎 | `internal/engine/` 全部代码 + 单元测试 | 5天 |
| Agent-3 | V4 工作流 | `internal/workflow/schedule_v4/` 全部代码 | 5天 |
| Agent-4 | 规则解析服务 | `management-service/internal/rule_parser/` | 3天 |
| Agent-5 | API 接口 | Handler + MCP Tool 扩展 | 3天 |
| Agent-6 | 前端 | 规则解析/组织视图/依赖管理页面 | 5天 |
| Agent-7 | 迁移 & 测试 | 迁移脚本、集成测试、监控 | 3天 |

## 2. 依赖关系图

```
                    Agent-1 (数据模型)
                   /       |        \
                 /         |          \
               ↓           ↓            ↓
        Agent-2        Agent-4       Agent-5
      (规则引擎)     (规则解析)      (API)
           |              |            |
           ↓              |            |
        Agent-3           |            ↓
       (V4工作流)         ↓         Agent-6
           |         Agent-5         (前端)
           |          (API)            |
           ↓              |            |
        Agent-7 ←─────────┘────────────┘
      (迁移&测试)
```

**并行路径**:
- 路径 A: Agent-1 → Agent-2 → Agent-3 → Agent-7 （后端核心链路）
- 路径 B: Agent-1 → Agent-4 → Agent-5 → Agent-6 （管理端链路）
- 路径 A 和 B 可以 **并行** 执行

## 3. 各 Agent 详细任务

### Agent-1: 数据模型（3天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 1.1 | SDK Rule model 扩展 | `sdk/rostering/model/rule.go` | 2h |
| 1.2 | SDK RuleAssociation 扩展 | `sdk/rostering/model/rule.go` | 1h |
| 1.3 | 新增 RuleDependency model | `sdk/rostering/model/rule_dependency.go` | 2h |
| 1.4 | 新增 RuleConflict model | `sdk/rostering/model/rule_conflict.go` | 2h |
| 1.5 | 新增 ShiftDependency model | `sdk/rostering/model/shift_dependency.go` | 2h |
| 1.6 | Domain model 类型别名 | `agents/rostering/domain/model/rule.go` | 1h |
| 1.7 | Repository 接口定义 | `agents/rostering/domain/repo/rule_v4.go` | 3h |
| 1.8 | Repository 实现 | `agents/rostering/internal/repo/rule_v4_repo.go` | 4h |
| 1.9 | DDL 脚本 | `release/scripts/migration_v4.sql` | 1h |
| 1.10 | 单元测试 | `agents/rostering/internal/repo/rule_v4_repo_test.go` | 4h |

**交付标准**:
- 所有 model 编译通过
- DDL 可在 MySQL 执行
- Repository 接口和实现完整
- 单元测试覆盖率 > 80%

---

### Agent-2: 确定性规则引擎（5天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 2.1 | 引擎类型定义 | `internal/engine/types.go` | 4h |
| 2.2 | 引擎入口 | `internal/engine/engine.go` | 4h |
| 2.3 | 候选人过滤器 | `internal/engine/candidate_filter.go` | 4h |
| 2.4 | 规则匹配器 | `internal/engine/rule_matcher.go` | 4h |
| 2.5 | 约束检查器 | `internal/engine/constraint_checker.go` | 8h |
| 2.6 | 偏好评分器 | `internal/engine/preference_scorer.go` | 3h |
| 2.7 | 排班校验器 | `internal/engine/schedule_validator.go` | 4h |
| 2.8 | 依赖解析器 | `internal/engine/dependency_resolver.go` | 3h |
| 2.9 | CandidateFilter 测试 | `internal/engine/candidate_filter_test.go` | 3h |
| 2.10 | RuleMatcher 测试 | `internal/engine/rule_matcher_test.go` | 3h |
| 2.11 | ConstraintChecker 测试 | `internal/engine/constraint_checker_test.go` | 4h |
| 2.12 | DependencyResolver 测试 | `internal/engine/dependency_resolver_test.go` | 2h |
| 2.13 | 集成测试 | `internal/engine/engine_integration_test.go` | 4h |

**交付标准**:
- `PrepareSchedulingContext()` 正确替代 LLM-1/2/3
- `ValidateSchedule()` 正确替代 LLM-5
- 单元测试覆盖率 > 85%
- 所有约束类型有边界测试
- 拓扑排序能处理菱形/线性/无依赖/循环场景

---

### Agent-3: V4 工作流（5天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 3.1 | 状态/事件常量 | `internal/workflow/schedule_v4/state/constants.go` | 2h |
| 3.2 | V4 上下文 | `internal/workflow/schedule_v4/create/context.go` | 3h |
| 3.3 | 状态机定义 | `internal/workflow/schedule_v4/create/definition.go` | 3h |
| 3.4 | 初始化 Actions | `internal/workflow/schedule_v4/create/actions_init.go` | 3h |
| 3.5 | 信息收集 Actions（复用V3） | `internal/workflow/schedule_v4/create/actions_collect.go` | 4h |
| 3.6 | 规则校验 Action（V4新增） | `internal/workflow/schedule_v4/create/actions_plan.go` | 4h |
| 3.7 | 任务执行 Actions（V4核心） | `internal/workflow/schedule_v4/create/actions_execute.go` | 8h |
| 3.8 | 审核/保存 Actions | `internal/workflow/schedule_v4/create/actions_review.go` | 3h |
| 3.9 | V4 执行器 | `internal/workflow/schedule_v4/executor/executor.go` | 6h |
| 3.10 | Prompt 构建器 | `internal/workflow/schedule_v4/executor/prompt_builder.go` | 4h |
| 3.11 | 结果解析器 | `internal/workflow/schedule_v4/executor/result_parser.go` | 3h |
| 3.12 | 工作流路由 | `internal/workflow/router.go` | 2h |
| 3.13 | 工作流测试 | `internal/workflow/schedule_v4/create/*_test.go` | 6h |

**交付标准**:
- V4 工作流可独立运行
- 与 V3 通过 Feature Flag 切换
- 信息收集阶段复用 V3 代码
- 任务执行阶段使用规则引擎

---

### Agent-4: 规则解析服务（3天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 4.1 | 解析器类型 | `rule_parser/types.go` | 2h |
| 4.2 | 解析器实现 | `rule_parser/parser.go` | 4h |
| 4.3 | 三层验证器 | `rule_parser/validator.go` | 4h |
| 4.4 | Prompt 模板 | `rule_parser/prompts.go` | 3h |
| 4.5 | 单元测试 | `rule_parser/*_test.go` | 4h |
| 4.6 | LLM Mock 测试 | `rule_parser/parser_mock_test.go` | 3h |

**交付标准**:
- 常见规则类型解析成功率 > 90%
- 三层验证全部实现
- 支持单条和批量解析

---

### Agent-5: API 接口（3天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 5.1 | 规则解析 Handler | `handler/rule_parse.go` | 4h |
| 5.2 | 规则依赖 Handler | `handler/rule_dependency.go` | 4h |
| 5.3 | 规则冲突 Handler | `handler/rule_conflict.go` | 3h |
| 5.4 | 班次依赖 Handler | `handler/shift_dependency.go` | 3h |
| 5.5 | 规则组织视图 Handler | `handler/rule_organization.go` | 3h |
| 5.6 | MCP Tool 扩展 | `mcp-servers/rostering/tool/` | 4h |
| 5.7 | API 路由注册 | `handler/routes.go` | 1h |
| 5.8 | API 测试 | `handler/*_test.go` | 4h |

---

### Agent-6: 前端（5天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 6.1 | TypeScript 类型 | `src/types/ruleV4.ts` | 2h |
| 6.2 | API 调用层 | `src/api/ruleV4.ts` | 2h |
| 6.3 | RuleParseDialog | `src/views/rules/components/RuleParseDialog.vue` | 8h |
| 6.4 | RuleOrganizationView | `src/views/rules/RuleOrganizationView.vue` | 8h |
| 6.5 | RuleDependencyPanel | `src/views/rules/components/RuleDependencyPanel.vue` | 4h |
| 6.6 | ShiftDependencyConfig | `src/views/shifts/ShiftDependencyConfig.vue` | 6h |
| 6.7 | 国际化文件 | `locales/zh-CN/ruleV4.json` | 1h |
| 6.8 | 路由配置 | `src/router/index.ts` | 1h |
| 6.9 | 组件测试 | `tests/` | 4h |

---

### Agent-7: 迁移 & 测试（3天）

| 编号 | 任务 | 文件 | 估时 |
|------|------|------|------|
| 7.1 | DDL 迁移脚本 | `release/scripts/migration_v4.sql` | 2h |
| 7.2 | 数据迁移脚本 | `scripts/migrate_rules_v4.go` | 4h |
| 7.3 | Feature Flag 配置 | `config/agents/rostering-agent.yml` | 1h |
| 7.4 | V3 回归测试 | `test/regression_v3_test.go` | 4h |
| 7.5 | V4 端到端测试 | `test/e2e_v4_test.go` | 6h |
| 7.6 | 性能对比测试 | `test/benchmark_v3_v4_test.go` | 3h |
| 7.7 | 迁移文档 | 本文档完善 | 2h |

## 4. 时间线（9 周）

```
Week 1  ┃ Agent-1: 数据模型完成
        ┃
Week 2  ┃ Agent-2: 规则引擎开发     ┃ Agent-4: 解析服务开发
        ┃                            ┃
Week 3  ┃ Agent-2: 引擎测试完善     ┃ Agent-4: 完成
        ┃                            ┃ Agent-5: API 开发
Week 4  ┃ Agent-3: V4 工作流开发    ┃ Agent-5: 完成
        ┃                            ┃ Agent-6: 前端开发
Week 5  ┃ Agent-3: 工作流完成       ┃ Agent-6: 前端继续
        ┃
Week 6  ┃ Agent-7: 迁移脚本 & 回归  ┃ Agent-6: 前端完成
        ┃
Week 7  ┃ 集成测试 + Bug 修复
        ┃
Week 8  ┃ 灰度发布 + 试点
        ┃
Week 9  ┃ 全量发布 + 监控
```

## 5. 验收标准

### 功能验收

| 验收项 | 标准 | 验证方法 |
|-------|------|---------|
| V4 规则引擎替代 LLM-1/2/3/5 | LLM 调用从 5 次/班次/日 降到 1 次 | 日志统计 |
| 规则解析 | 常见规则解析成功率 > 90% | 测试用例 |
| 三层验证 | 结构验证 100% + 回译存在 + 模拟可运行 | 测试用例 |
| V4 工作流 | 完整排班流程可跑通 | E2E 测试 |
| Feature Flag | V3/V4 可按组织切换 | 手动验证 |
| 数据迁移 | V3 规则自动分类覆盖 > 80% | SQL 查询 |

### 性能验收

| 指标 | V3 基准 | V4 目标 |
|------|--------|---------|
| 排班延迟（单班次单日） | ~11s | < 6s |
| LLM Token 消耗 | ~4000/班次/日 | < 1000/班次/日 |
| 引擎计算耗时 | N/A | < 10ms |
| 重跑一致性 | ~60% | > 90% |

### 质量验收

| 指标 | 标准 |
|------|------|
| 单元测试覆盖率 | > 80% |
| 引擎核心组件覆盖率 | > 90% |
| 无 P0/P1 Bug | 发布前 |
| 代码 Review | 每个 Agent 至少 1 次 |

## 6. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 规则引擎无法覆盖所有 V3 规则类型 | 中 | 高 | V3 Fallback 机制 |
| LLM 解析准确率不达标 | 中 | 中 | 人工修正 + 迭代 Prompt |
| V4 与 V3 共存时的状态一致性问题 | 低 | 高 | 独立上下文，不共享状态 |
| 前端工作量超预期 | 中 | 低 | 简化首版 UI，后续迭代 |
| 依赖解析器在复杂场景下性能问题 | 低 | 低 | 班次数量有限（< 20），不会成为瓶颈 |
