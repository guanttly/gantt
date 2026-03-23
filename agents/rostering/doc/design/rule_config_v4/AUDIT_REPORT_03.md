# V4 评审报告（三）— 全量变更评审

> **评审日期**: 2026-02-12  
> **评审范围**: management-service(9 文件/+489 行) + SDK(1 文件/+36 行) + MCP(4 文件/+86 行) + Frontend(5 文件/+764 行)  
> **结论**: 大幅进步，核心链路基本打通，但存在 **编译阻断 Bug** 和 **MCP domain model 断层**

---

## 总体进度

| 设计目标 | 上次完成度 | 本次完成度 | 变化 |
|---------|-----------|-----------|------|
| ① LLM 辅助生成 | ~50% | **~75%** | ⬆ 后端 parse/batch-parse API + 前端 RuleParseDialog 3步流程 |
| ② 人工审核确认 | ~20% | **~60%** | ⬆ 前端审核流程 + 置信度展示 + 勾选保存 |
| ③ 手动编辑 V4 字段 | ~30% | **~70%** | ⬆ Handler CRUD 穿透 + 前端分类Tab/筛选/列展示 |
| ④ V3 规则迁移 | 0% | **~50%** | ⬆ 后端迁移服务 + 前端迁移对话框（API 未接通） |
| ⑤ 枚举统一 | 0% | **~80%** | ⬆ 前端切换 V4 值 + v3_compat.go + logic.ts 兼容映射 |
| ⑥ V3 兼容隔离 | 0% | **~70%** | ⬆ v3_compat.go 创建 + 前端兼容函数标记 @deprecated |

---

## 第一部分：Management-Service (9 文件, +489 行)

### ✅ 已完成

| 模块 | 内容 | 状态 |
|------|------|------|
| 领域模型 | V4 字段(6个) + Filter(4个) + 常量(分类/子分类/角色) | ✅ |
| Entity 层 | V4 GORM 字段 + 索引 + 3个新表 | ✅ |
| Mapper | 正反映射完整 + Role 默认值逻辑 | ✅ |
| Handler CRUD | Create/Update/List 穿透 V4 字段 + 默认值 | ✅ |
| Handler V3 兼容 | CreateRule/UpdateRule/ListRules 中调用 normalizeXxx | ✅ |
| Filter 查询 | Repository List 方法支持 V4 筛选 | ✅ |
| v3_compat.go | normalizeRuleType/ApplyScope/TimeScope + FillV3Defaults | ✅ |
| 服务接口 | ListRulesByCategory + GetV3Rules + BatchUpdateVersion | ✅ |
| 新增 API 端点 | parse/batch-parse/migration(4个)/dependencies(3个)/conflicts(3个)/statistics | ✅ |
| DI 容器 | 4个新服务 + 3个新仓储 + AutoMigrate 3张新表 | ✅ |

### 🔴 编译阻断 Bug (5个)

| # | 文件 | 问题 | 修复方式 |
|---|------|------|---------|
| 1 | `handler.go` | `BatchSaveRules` 和 `OrganizeRules` 方法路由引用但**未在 handler 中实现** | 实现方法或暂时注释路由 |
| 2 | `scheduling_rule_service.go` | `mapToModel` 函数与同 package 其他文件重复声明 | 提取到 `helpers.go` 或重命名 |
| 3 | `scheduling_rule_service.go` | `SchedulingRuleServiceImpl` 结构体重复声明 | 合并到一个文件 |
| 4 | `name_matcher.go` (推测) | `repository` 包未导入 | 添加 import |
| 5 | `wiring/container.go` | 未使用的 import | 删除多余 import |

### 🟡 数据 Bug (1个，仍未修复)

| # | 文件 | 问题 |
|---|------|------|
| 6 | `scheduling_rule_repository.go` — `AddAssociations` | **Role 字段仍未映射**。手动构建 Entity 时遗漏 `Role: assoc.Role`，导致通过独立 API 添加的关联 Role 始终为空串 |

### ⚠️ 建议改进

- 缺少 `SourceType` 常量（`SourceTypeManual = "manual"` 等），各处硬编码字符串
- `BatchUpdateVersion` 逐条查询+更新，大量规则时性能差，建议批量 SQL

---

## 第二部分：SDK (1 文件, +36 行)

### ✅ 已完成

- `Rule` 结构体：6 个 V4 字段全部添加
- `RuleAssociation`：`Role` 字段已添加
- `CreateRuleRequest`/`UpdateRuleRequest`：V4 字段齐全
- `ListRulesRequest`：4 个 V4 筛选字段齐全

### ❌ 仍缺失

| 缺失项 | 影响 |
|--------|------|
| SDK Client 接口无 V4 新方法 | Agent 无法通过 SDK 调用 parse/migration/dependencies/conflicts/statistics |
| 无 `RuleDependency`/`RuleConflict`/`RuleStatistics` 类型 | MCP 工具无法序列化 V4 关系数据 |
| 无 `MigrationPreview`/`ParseRequest`/`ParseResponse` 类型 | MCP 工具无法调用解析/迁移 API |

> **影响范围**: SDK client 接口缺失导致 **MCP 8 个新工具全部只能是桩代码**——即使后端 API 已就绪，MCP 工具也无法调用。这是当前 V4 数据链路中最关键的断层。

---

## 第三部分：MCP 工具 (4 文件, +86 行)

### ✅ 已完成

| 工具 | InputSchema V4 字段 | Execute V4 字段 |
|------|-------------------|----------------|
| `create.go` | ✅ 6个字段+enum约束 | ✅ 读取并赋值 |
| `list.go` | ✅ 4个筛选字段 | ✅ 读取并赋值 |
| `add_associations.go` | ✅ Role 字段 | ✅ 读取并赋值，默认 target |
| `manager.go` | ✅ 注册 8 个 V4 新工具 | — |

### 🔴 关键问题

**问题 1：MCP Domain Model 未更新**

MCP 自身的 domain model（`mcp-servers/rostering/domain/model/rule.go`）**完全没有 V4 字段**。`create.go`/`list.go`/`add_associations.go` 的 Execute 方法在代码中引用了 V4 字段（如 `req.Category`），但 MCP 的 `CreateRuleRequest` 结构体**不包含这些字段**。

这会导致两种结果：
- 若 MCP 使用 SDK 的 model → OK（但需确认 import 路径）
- 若 MCP 使用自己的 domain model → **编译失败**

> 需要确认 MCP create/list/add_associations 的 Execute 方法引用的是哪个 model 包。

**问题 2：`update.go` 完全未更新**

`update.go` 的 InputSchema 使用**完全过时的枚举值**（`MaxConsecutiveDays`/`All`/`Daily` 等），且**无任何 V4 字段**。与 `create.go` 形成严重不一致。

**问题 3：8 个新工具全为桩代码**

| 工具 | 实现状态 |
|------|---------|
| `parse_rule.go` | ❌ TODO — 返回固定字符串 |
| `batch_parse_rules.go` | ❌ TODO — 返回空数组 |
| `get_statistics.go` | ❌ TODO — 返回全 0 |
| `get_dependencies.go` | ❌ TODO — 返回空数组 |
| `add_dependency.go` | ❌ TODO — 返回固定 success |
| `get_conflicts.go` | ❌ TODO — 返回空数组 |
| `preview_migration.go` | ❌ TODO — 返回全 0 |
| `execute_migration.go` | ❌ TODO — 返回模拟数据 |

> **根因**: SDK client 缺少 V4 方法，MCP 工具无法调用后端 API。

---

## 第四部分：Frontend (5 文件, +764 行)

### ✅ 重大进展

| 功能 | 状态 | 说明 |
|------|------|------|
| **V4 类型定义** (model.d.ts) | ✅ 完备 | 全部 V4 类型/接口已定义 |
| **枚举切换** (logic.ts) | ✅ 完成 | 主枚举使用 V4 值 (`exclusive`/`specific`/`same_day` 等) |
| **V3 兼容映射** (logic.ts) | ✅ 完成 | 双向映射 + @deprecated 标记 |
| **分类 Tab** (index.vue) | ✅ 实现 | 全部/约束/偏好/依赖/未分类V3 |
| **统计卡片** (RuleStatisticsCard) | ✅ 实现 | 8 格网格（总/约束/偏好/依赖/V3/V4/启用/禁用） |
| **V4 表格列** (index.vue) | ✅ 实现 | 分类/子分类/来源/版本 列 |
| **V4 筛选** (index.vue) | ✅ 实现 | category/sourceType/version 下拉 |
| **规则解析对话框** (RuleParseDialog.vue) | ✅ 实现 | 3步流程：输入→解析→审核确认 |
| **规则迁移对话框** (RuleMigrationDialog.vue) | ⚠️ UI 完成 | API 调用被注释（TODO） |
| **依赖关系图** (RuleDependencyGraph.vue) | ✅ 实现 | ECharts 力导向/环形布局 |
| **规则组织结果** (index.vue) | ✅ 实现 | Tab 展示约束/偏好/依赖/关系/冲突/执行顺序/依赖图 |

### 🟡 前端问题

| # | 文件 | 问题 | 严重度 |
|---|------|------|--------|
| 1 | `RuleFormDialog.vue` | `ruleDataPlaceholder` 用 V3 枚举值判断（`max_shifts` 等），但表单 ruleType 已是 V4 值 → **placeholder 永远走 default** | 🟡 |
| 2 | `RuleFormDialog.vue` | 表单模板中**无 category/subCategory 选择控件**，V4 字段只在 FormData 中声明但无 UI | 🟡 |
| 3 | `RuleFormDialog.vue` | 提交时硬编码 `version: 'v4'`, `sourceType: 'manual'`，但**未传 category/subCategory** | 🟡 |
| 4 | `RuleFormDialog.vue` | 更新请求只传基础字段，**V4 字段全部丢失** | 🟡 |
| 5 | `RuleMigrationDialog.vue` | `loadPreview()` API 调用被注释，显示 "API 尚未实现" toast | 🟡 |
| 6 | `api/index.ts` | 缺少 `getStatistics`/`getDependencies`/`getConflicts`/`previewMigration`/`executeMigration` API 调用函数 | 🟢 |
| 7 | `index.vue` 统计卡片 | 统计数据基于当前页列表本地计算，非后端全量统计 API | 🟢 |

---

## 第五部分：全栈数据链路分析

### V4 基础 CRUD 链路

```
前端 (V4枚举) → Handler (V4字段+V3兼容) → Model → Entity → DB
                                              ↕
                                           Mapper
                                              ↕
                                           SDK Model (V4字段✅)
                                              ↕
                                        SDK Client (无V4方法❌)
                                              ↕
                                        MCP Domain Model (无V4字段❌) ← 断层
                                              ↕
                                        MCP Tools (桩代码❌)
```

**断层位置**: SDK Client → MCP Domain Model 之间。后端已就绪，前端已就绪，但 **MCP/Agent 通道不通**。

### 各能力端到端状态

| 能力 | 后端API | SDK Model | SDK Client | MCP Tool | 前端UI | 端到端 |
|------|:-------:|:---------:|:----------:|:--------:|:------:|:------:|
| V4 CRUD(含分类) | ✅ | ✅ | ⚠️¹ | ⚠️² | ⚠️³ | **⚠️ 大部分通** |
| V3 枚举兼容 | ✅ | — | — | ❌⁴ | ✅ | **⚠️** |
| 规则解析 (LLM) | ✅ | ❌ | ❌ | ❌桩 | ✅ | **前端→后端 通** |
| 规则组织 | ✅ | ❌ | ❌ | — | ✅ | **前端→后端 通** |
| 规则迁移 | ✅ | ❌ | ❌ | ❌桩 | ⚠️⁵ | **❌ 不通** |
| 依赖/冲突 | ✅ | ❌ | ❌ | ❌桩 | ✅⁶ | **前端→后端 通** |
| 统计 | ✅ | ❌ | ❌ | ❌桩 | ⚠️⁷ | **本地计算，非API** |

> ¹ SDK 基础 CRUD 方法已有，V4 字段可传递  
> ² MCP create/list V4 字段代码已写，但依赖 domain model 更新  
> ³ RuleFormDialog 缺少 category/subCategory 表单控件，更新请求丢失 V4 字段  
> ⁴ MCP update.go 枚举完全过时  
> ⁵ 前端迁移对话框 API 调用被注释  
> ⁶ 前端通过规则组织接口获取依赖/冲突数据  
> ⁷ 前端本地计算统计，非后端 API

---

## 第六部分：问题汇总 (按优先级)

### 🔴 P0 — 编译阻断 (5个)

| # | 位置 | 问题 | 修复 |
|---|------|------|------|
| 1 | `handler.go` 路由 | `BatchSaveRules`/`OrganizeRules` 方法未实现 | 实现或注释路由 |
| 2 | `scheduling_rule_service.go` | `mapToModel` 函数重复声明 | 提取到 helpers.go |
| 3 | `scheduling_rule_service.go` | `SchedulingRuleServiceImpl` 结构体重复声明 | 合并文件 |
| 4 | 新增文件 | 缺少 import | 补全 |
| 5 | `wiring/container.go` | 未使用 import | 清理 |

### 🔴 P1 — 数据链路断层 (3个)

| # | 位置 | 问题 | 修复 | 工时 |
|---|------|------|------|------|
| 6 | MCP `domain/model/rule.go` | **完全缺失 V4 字段**，导致 create/list/add_associations 的 V4 代码可能编译失败 | 补齐 V4 字段 | 0.5天 |
| 7 | MCP `update.go` | 枚举完全过时，无 V4 字段，与 create.go 严重不一致 | 对齐 create.go | 0.5天 |
| 8 | `AddAssociations` Repository | **Role 字段仍未映射**（第三次评审仍未修复） | 添加 `Role: assoc.Role` | 1行 |

### 🟡 P2 — 功能不完整 (5个)

| # | 位置 | 问题 | 工时 |
|---|------|------|------|
| 9 | SDK Client | 零 V4 新方法，阻断 MCP 8 个新工具 | 2天 |
| 10 | MCP 8 个新工具 | 全部桩代码，依赖 SDK Client | 1天(SDK后) |
| 11 | 前端 `RuleFormDialog.vue` | 缺 category/subCategory 控件 + 更新丢失 V4 字段 | 0.5天 |
| 12 | 前端 `RuleMigrationDialog.vue` | API 调用被注释 | 0.5天(后端就绪后) |
| 13 | 前端 `api/index.ts` | 缺 statistics/dependencies/conflicts/migration API 函数 | 0.5天 |

### 🟢 P3 — 代码质量 (3个)

| # | 位置 | 问题 |
|---|------|------|
| 14 | 领域模型 | 缺少 SourceType 常量定义 |
| 15 | `RuleFormDialog.vue` | `ruleDataPlaceholder` 用 V3 枚举值判断 |
| 16 | `scheduling_rule_service.go` | `BatchUpdateVersion` 逐条操作，性能差 |

---

## 第七部分：建议实施路径

### 立即修复（Day 1）

```
1. 修复 5 个编译阻断 Bug（~2h）
2. 修复 AddAssociations Role 映射（1行）
3. MCP domain/model/rule.go 补齐 V4 字段（~1h）
4. MCP update.go 对齐 create.go 的 V4 枚举和字段（~1h）
```

### 短期（Day 2-3）

```
5. SDK Client 补充 V4 方法（parse/migration/dependencies/conflicts/statistics）
6. MCP 8 个新工具接入 SDK Client 真实调用
7. 前端 RuleFormDialog 添加 category/subCategory 控件 + 修复更新请求
8. 前端补充缺失的 API 调用函数
```

### 中期（Day 4-5）

```
9. 前端 RuleMigrationDialog 接通后端 API
10. 前端统计卡片改为后端 API 全量统计
11. 添加 SourceType 常量
12. BatchUpdateVersion 优化为批量 SQL
```

---

## 附录：与上次评审对比

| 上次评审 Issue | 本次状态 | 说明 |
|---------------|---------|------|
| ❌ Handler 缺 V4 字段 | ✅ **已修复** | Create/Update/List 全部穿透 |
| ❌ 缺 v3_compat.go | ✅ **已修复** | normalizeXxx + FillV3Defaults 完整 |
| ❌ 服务接口缺 V4 方法 | ✅ **已修复** | 3个新方法 + 4个新服务接口 |
| ❌ 缺依赖/冲突/统计 API | ✅ **已修复** | 完整 CRUD + 统计端点 |
| ❌ 缺迁移服务 | ✅ **已修复** | 在 service 包中实现（非独立包） |
| ❌ 前端枚举仍为 V3 | ✅ **已修复** | 切换 V4 值 + V3 兼容映射 |
| ❌ 前端缺分类 Tab | ✅ **已修复** | 5个Tab完整 |
| ❌ 前端缺统计卡片 | ✅ **已修复** | 8格网格（本地计算） |
| ❌ 前端缺迁移对话框 | ✅ **已修复** | UI 完整（API 未接通） |
| ❌ 前端缺解析审核 | ✅ **已修复** | 3步流程完整 |
| ❌ 前端缺 v3-compat.ts | ✅ **已修复** | 在 logic.ts 中实现兼容映射 |
| 🐛 AddAssociations 缺 Role | 🔴 **仍未修复** | 第三次评审仍然存在 |
| ❌ SDK Client 缺 V4 方法 | ❌ **仍缺失** | 阻断 MCP 工具 |
| ❌ MCP 新工具为桩代码 | ❌ **仍为桩** | 依赖 SDK Client |
