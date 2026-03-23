# V4 规则配置管理 — 实现 vs 设计评审报告

> **评审日期**: 2026-02-11  
> **评审范围**: rule_config_v4/ 全部 6 份设计文档 vs 代码实现  
> **结论**: ⚠️ **设计目标尚未达成**

---

## 总体评估

核心数据链路（Model → Entity → SDK → MCP → Handler → Frontend）中间有多个断层，V4 字段传递不完整，前后端枚举未统一，关键模块缺失。

| 设计目标 | 完成度 | 判定 |
|---------|--------|------|
| ① LLM 辅助生成 | ~50% | ⚠️ 后端核心链路已通，前端/类型设计偏离 |
| ② 人工审核确认 | ~20% | ❌ 前端缺组件，无回译对比、置信度展示 |
| ③ 手动编辑 V4 字段 | ~30% | ❌ Handler/SDK/前端均未对接 V4 字段的读写 |
| ④ V3 规则迁移 | 0% | ❌ migration 包不存在、前端无迁移组件 |
| ⑤ 枚举统一 | 0% | ❌ 前端仍全部使用 V3 枚举值，v3_compat 未创建 |
| ⑥ V3 兼容隔离 | 0% | ❌ v3_compat.go / v3-compat.ts 均不存在 |

---

## 一、数据模型层

### 1.1 领域模型 (management-service)

**文件**: `services/management-service/domain/model/scheduling_rule.go`

| 设计字段 | 实现 | 问题 |
|---------|------|------|
| `Category` | ✅ | — |
| `SubCategory` | ✅ | — |
| `OriginalRuleID` | ✅ | — |
| `SourceType` | ❌ | **缺失**，无法区分 manual/llm_parsed/migrated |
| `ParseConfidence` | ❌ | **缺失**，LLM 解析结果无置信度存储 |
| `Version` | ❌ | **缺失**，无法区分 V3/V4 规则，迁移方案无法落地 |
| `RuleAssociation.Role` | ✅ | — |

> **影响**: `Version` 缺失直接阻断了迁移方案（04_migration）的核心前提——按 version 筛选 V3 规则。`SourceType` 缺失导致规则来源不可追踪。

### 1.2 GORM Entity 层

**文件**: `services/management-service/infrastructure/persistence/entity/scheduling_rule_entity.go`

与领域模型一致：`Category`/`SubCategory`/`OriginalRuleID`/`Role` 已有，`Version`/`SourceType`/`ParseConfidence` 缺失。

额外 V4 实体已实现：
- ✅ `RuleDependencyEntity` — V4 新表 `rule_dependencies`
- ✅ `RuleConflictEntity` — V4 新表 `rule_conflicts`
- ✅ `ShiftDependencyEntity`

### 1.3 SDK 模型 — 🔴 最大空白

**文件**: `sdk/rostering/model/rule.go`

SDK 的 `Rule` 结构体 **完全缺失所有 V4 字段**：

| V4 设计字段 | 实现 |
|------------|------|
| `Category` | ❌ |
| `SubCategory` | ❌ |
| `OriginalRuleID` | ❌ |
| `Version` | ❌ |
| `SourceType` | ❌ |
| `ParseConfidence` | ❌ |
| `RuleAssociation.Role` | ❌ |

`CreateRuleRequest`/`UpdateRuleRequest`/`ListRulesRequest` 也均无 V4 字段。

> **影响**: MCP Server 通过 SDK 与 management-service 通信，SDK 缺失 V4 字段意味着 **MCP 工具无法传递 V4 数据**，Agent 侧完全看不到规则分类。这是当前 V4 覆盖的 **最大空白**。

### 1.4 SchedulingRuleFilter

❌ 缺失 `Category`/`SubCategory`/`SourceType`/`Version` 筛选条件。无法按 V4 维度过滤规则。

---

## 二、后端 API 层

### 2.1 Handler CRUD — V4 字段未穿透

**文件**: `services/management-service/internal/port/http/scheduling_rule_handler.go`

| 接口 | 问题 |
|------|------|
| `CreateRule` | Request 结构体无 V4 字段，创建规则时不设置 category/subCategory/version 等 |
| `UpdateRule` | Request 结构体无 V4 字段，无法更新分类 |
| `ListRules` | 不解析 category/version 等查询参数，不传递到 Filter |
| `AddAssociations` | 不接收 `role` 字段 |

### 2.2 缺失的 API 端点

| 设计端点 | 状态 | 设计文档 |
|---------|------|---------|
| `GET /v1/rules/migration/preview` | ❌ | 04_migration_plan |
| `POST /v1/rules/migration/execute` | ❌ | 04_migration_plan |
| `POST /v1/rules/migration/rollback` | ❌ | 04_migration_plan |
| `GET /v1/rules/migration/status` | ❌ | 04_migration_plan |
| `GET /v1/rules/dependencies` | ❌ | 02_backend_api |
| `POST /v1/rules/dependencies` | ❌ | 02_backend_api |
| `DELETE /v1/rules/dependencies/:id` | ❌ | 02_backend_api |
| `GET /v1/rules/conflicts` | ❌ | 02_backend_api |
| `POST /v1/rules/conflicts` | ❌ | 02_backend_api |
| `GET /v1/rules/statistics` | ❌ | 02_backend_api |

> **注意**: `RuleDependency`/`RuleConflict` 的 Repository 层已实现并注入 DI，但 **没有通过 Handler 暴露 API**，仅在 `RuleParserService` 内部使用。

### 2.3 Service 接口

`ISchedulingRuleService` 的 V4 新增方法（`ListRulesByCategory`/`GetV3Rules`/`GetRuleStatistics`/`BatchUpdateVersion` 等）**全部缺失**。

---

## 三、前端实现

### 3.1 枚举值 — 🔴 全部仍为 V3

**文件**: `frontend/web/src/pages/management/scheduling-rule/logic.ts`

| 枚举 | 前端当前值 | 设计目标(后端值) |
|------|-----------|-----------------|
| RuleType | `max_shifts, consecutive_shifts, rest_days, forbidden_pattern, preferred_pattern` | `exclusive, combinable, required_together, periodic, maxCount, forbidden_day, preferred` |
| ApplyScope | `global, group, employee, shift` | `global, specific` |
| TimeScope | `daily, weekly, monthly, custom` | `same_day, same_week, same_month, custom` |

> 这是设计文档 05_enum_unification 要解决的 **核心问题**，完全未实施。

### 3.2 类型定义 (model.d.ts) — 部分完成

**文件**: `frontend/web/src/api/scheduling-rule/model.d.ts`

已定义 V4 新增类型（`RuleCategory`/`RuleSubCategory`/`SemanticRuleRequest`/`OrganizeResult` 等），但 **基础 `RuleInfo` 接口未扩展 V4 字段**，与 V4 新增类型形成两套并行体系，未融合。

### 3.3 页面结构 (index.vue)

**文件**: `frontend/web/src/pages/management/scheduling-rule/index.vue`

| 设计要求 | 状态 |
|---------|------|
| 分类 Tab (全部/约束/偏好/依赖/未分类V3) | ❌ 未实现 |
| 统计卡片 (RuleStatisticsCard) | ❌ 未创建 |
| 表格 category/subCategory/sourceType/version 列 | ⚠️ 仅有 category 列但类型映射不对 |
| 搜索栏 sourceType/category 筛选 | ❌ 未实现 |

### 3.4 组件

| 设计要求的组件 | 状态 | 说明 |
|--------------|------|------|
| `RuleParseDialog.vue` | ⚠️ 存在但偏离 | 用旧版接口，缺少回译对比/置信度/多规则勾选/步骤切换 |
| `RuleDependencyGraph.vue` | ✅ 已实现 | ECharts 力导向图，设计要求 `RuleDependencyPanel` 但功能等价 |
| `ParseResultReview.vue` | ❌ 未创建 | 设计要求独立审核组件 |
| `RuleStatisticsCard.vue` | ❌ 未创建 | |
| `RuleMigrationDialog.vue` | ❌ 未创建 | |
| `v3-compat.ts` | ❌ 未创建 | V3 兼容隔离文件 |

---

## 四、LLM 规则解析服务

**文件**: `services/management-service/domain/service/rule_parser_service.go`

### 已实现 ✅

- `RuleParserServiceImpl` 核心结构，含 LLM 调用链路
- `buildParseSystemPrompt()` LLM 提示词（含 V4 分类体系）
- `ParseRule()` + `SaveParsedRules()` 方法
- `validateStructure()` 基础验证
- 依赖 `RuleDependencyRepo`/`RuleConflictRepo` 注入
- DI 配置完成

### 未实现 ❌

| 设计项 | 问题 |
|--------|------|
| `ParseRequest` 类型 | 字段不匹配设计（缺 `ruleText/shiftNames/groupNames`，多了 `Name/Description` 等） |
| `BatchParse` 接口 | 未实现 |
| 三层验证器 | 仅有 `validateStructure` 一个简单函数，缺语义一致性/业务合理性验证 |
| 名称模糊匹配器 `NameMatcher` | 完全缺失，LLM 返回的班次/员工名称无法回填为 ID |
| `checkConflictsWithExisting()` | 方法体为 TODO，返回 nil |
| Prompt 中 `{shiftNames}/{groupNames}` 注入 | 未实现，Handler 未从 ShiftService 获取名称列表 |
| `ParseConfidence` / `BackTranslation` 返回 | LLM 输出格式未要求这些字段 |

---

## 五、MCP Server 工具

**目录**: `mcp-servers/rostering/tool/rule/`

现有 12 个工具全部为 V3 基础 CRUD：

```
add_associations.go  create.go  delete.go  get.go
get_for_employee.go  get_for_employees.go  get_for_group.go
get_for_groups.go  get_for_shift.go  get_for_shifts.go
list.go  update.go
```

设计要求的 V4 新增工具 **全部缺失**：

| 工具 | 状态 |
|------|------|
| `parse_rule.go` | ❌ |
| `batch_parse_rules.go` | ❌ |
| `get_dependencies.go` | ❌ |
| `add_dependency.go` | ❌ |
| `get_conflicts.go` | ❌ |
| `get_statistics.go` | ❌ |
| `preview_migration.go` | ❌ |
| `execute_migration.go` | ❌ |

现有 `create.go` 和 `list.go` 的 InputSchema 中 **无 V4 字段**（category/subCategory/version 等），Agent 无法通过 MCP 操作 V4 数据。

---

## 六、V3 兼容隔离 (设计 §5)

| 设计要求 | 状态 |
|---------|------|
| `v3_compat.go` (normalizeXxx/FillV3Defaults) | ❌ 文件不存在 |
| `v3-compat.ts` (V3 枚举映射) | ❌ 文件不存在 |
| `@deprecated V3` 标记 | ❌ 代码中无任何标记 |
| `internal/migration/` 包 | ❌ 目录不存在 |
| `scripts/cleanup_v3_compat.sh` | ❌ 不存在 |

---

## 七、改进方案

### 🔴 P0 — 数据链路打通（阻断所有后续功能）

#### 1. 补全领域模型 3 个缺失字段 (0.5 天)

**文件**: `services/management-service/domain/model/scheduling_rule.go`

```go
// 在 SchedulingRule 结构体 OriginalRuleID 后面添加:

// SourceType 规则来源类型
// manual: 手动创建 / llm_parsed: LLM 解析 / migrated: V3 迁移
// @deprecated V3: 迁移完成后 migrated 统一为 manual
SourceType string `json:"sourceType,omitempty"`

// ParseConfidence LLM 解析置信度 (0.0-1.0)
ParseConfidence *float64 `json:"parseConfidence,omitempty"`

// Version 规则版本号（V3=空或"v3", V4="v4"）
// @deprecated V3: 全量迁移完成后此字段固定为"v4"，可移除版本判断逻辑
Version string `json:"version,omitempty"`
```

同步更新：
- Entity 层添加对应 GORM 字段
- Mapper 层添加映射
- AutoMigrate 自动生效

#### 2. SDK 模型全量补 V4 字段 (1 天)

**文件**: `sdk/rostering/model/rule.go`

`Rule` 结构体补充：
- `Category`, `SubCategory`, `OriginalRuleID`
- `Version`, `SourceType`, `ParseConfidence`

`RuleAssociation` 补充 `Role` 字段。

`CreateRuleRequest`/`UpdateRuleRequest` 补充 V4 字段。
`ListRulesRequest` 补充 `Category`/`Version`/`SourceType` 筛选参数。

#### 3. Handler CRUD 穿透 V4 字段 (1 天)

**文件**: `services/management-service/internal/port/http/scheduling_rule_handler.go`

- `CreateRuleRequest` 结构体添加 `Category`/`SubCategory`/`SourceType`/`Version` 等
- `UpdateRuleRequest` 同步扩展
- `ListRules` 解析 V4 查询参数 → 传递到 Filter
- `AddAssociations` 接收 `role`

#### 4. SchedulingRuleFilter 补 V4 筛选 (0.5 天)

添加 `Category`/`SubCategory`/`SourceType`/`Version` 四个筛选字段，Repository 层对应更新查询条件。

---

### 🔴 P1 — 核心功能模块

#### 5. 前端枚举统一 + v3-compat.ts 隔离 (2 天)

- `logic.ts` 枚举切换为后端值
- 创建 `v3-compat.ts` 隔离 V3 旧值映射
- `model.d.ts` 基础类型 RuleType/ApplyScope/TimeScope 切换为 V4 值
- 合并 `RuleInfo` 与 V4 字段为一体

#### 6. 创建 v3_compat.go 后端兼容层 (0.5 天)

**新建**: `services/management-service/internal/port/http/v3_compat.go`

实现：
- `normalizeRuleType()` — V3 前端旧枚举 → 后端值
- `normalizeApplyScope()` — 同上
- `normalizeTimeScope()` — 同上
- `FillV3Defaults()` — 为 V3 规则填充 V4 默认值

Handler 中添加调用，标记 `@deprecated V3`。

#### 7. 迁移服务实现 (2 天)

**新建**: `services/management-service/internal/migration/`

实现：
- `IRuleMigrationService` 接口
- `PreviewMigration` / `ExecuteMigration` / `RollbackMigration` / `GetMigrationStatus`
- 注册 4 个迁移 API 端点

#### 8. 前端页面改造 (4 天)

- index.vue 添加分类 Tab + 统计卡片
- 创建 `RuleStatisticsCard.vue` / `RuleMigrationDialog.vue` / `ParseResultReview.vue`
- 改造 `RuleParseDialog.vue` 对接 V4 接口（回译对比/置信度/多规则勾选/步骤切换）

---

### 🟡 P2 — 增强功能

#### 9. LLM 解析服务完善 (3 天)

- 重构 `ParseRequest`/`ParseResponse` 匹配设计类型
- 实现三层验证器（结构完整性 → 语义一致性 → 业务合理性）
- 实现 `NameMatcher` 名称模糊匹配
- Prompt 注入 shiftNames/groupNames
- 补全 `checkConflictsWithExisting`

#### 10. 依赖/冲突/统计 API (1.5 天)

- Service 层补 V4 新方法
- Handler 注册 dependencies/conflicts/statistics 端点
- 现有 Repository 已实现，仅需暴露上层 API

#### 11. MCP 工具扩展 (2 天)

- `create.go`/`list.go` InputSchema 添加 V4 字段
- 新增 parse/migration/statistics 工具

---

## 八、工时总结

| 优先级 | 范围 | 工时 | 说明 |
|--------|------|------|------|
| **P0** | 数据链路打通 (项1-4) | **3 天** | 阻断所有后续功能，必须最先完成 |
| **P1** | 核心功能模块 (项5-8) | **8.5 天** | 枚举统一 + 迁移 + 前端改造 |
| **P2** | 增强功能 (项9-11) | **6.5 天** | LLM 完善 + API 补全 + MCP 扩展 |
| | **合计** | **~18 人天** | |

---

## 九、建议实施路径

```
Week 1: P0 全部 + P1.5(v3_compat.go) + P1.6(v3-compat.ts)
         → V4 数据端到端流通 + 枚举统一完成
         → 验收标准: 创建规则可传 V4 字段，列表可按 category 筛选

Week 2: P1.7(迁移服务) + P1.8(前端改造前半)
         → V3→V4 迁移闭环
         → 验收标准: 可预览迁移、执行迁移、查看迁移状态

Week 3: P1.8(前端改造后半) + P2.9(LLM完善)
         → 前端 V4 体验完整
         → 验收标准: 分类Tab/统计卡片/LLM回译/置信度全部可用

Week 4: P2.10(API补全) + P2.11(MCP扩展)
         → Agent 侧 V4 能力完整
         → 验收标准: MCP 工具可传递 V4 字段，Agent 可调用迁移工具
```
