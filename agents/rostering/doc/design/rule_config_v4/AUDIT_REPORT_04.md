# V4 评审报告（四）— 编译阻断修复验证 & 全量追踪

> **评审日期**: 2026-02-13  
> **评审范围**: management-service(10 文件/+490 行) + SDK(1 文件/+36 行) + MCP(6 文件/+132 行) + Frontend(5 文件/+764 行)  
> **结论**: management-service **5 个编译阻断全部修复** + AddAssociations Role 映射修复 ✅；MCP domain model 已补齐但 **引入 2 个新编译阻断**；前端/SDK 无新变更

---

## 总体进度

| 设计目标 | 上次完成度 | 本次完成度 | 变化 |
|---------|-----------|-----------|------|
| ① LLM 辅助生成 | ~75% | **~75%** | → MCP 桩未变、SDK 未变 |
| ② 人工审核确认 | ~60% | **~60%** | → 前端未变 |
| ③ 手动编辑 V4 字段 | ~70% | **~75%** | ⬆ management-service 编译修复、MCP update.go V4 对齐 |
| ④ V3 规则迁移 | ~50% | **~50%** | → 无变化 |
| ⑤ 枚举统一 | ~80% | **~85%** | ⬆ MCP update.go 枚举从 V3 全量替换为 V4 |
| ⑥ V3 兼容隔离 | ~70% | **~70%** | → 无变化 |

---

## 第一部分：Management-Service — 上次 5 个 P0 + 1 个 P1 追踪

### ✅ 全部修复（6/6）

| # | 上次问题 | 本次状态 | 说明 |
|---|---------|---------|------|
| P0-1 | `handler.go` 路由引用 `BatchSaveRules`/`OrganizeRules` 未实现 | ✅ **已修复** | 两个方法已移到独立文件 `rule_batch_handler.go` 实现 |
| P0-2 | `scheduling_rule_service.go` `mapToModel` 函数重复声明 | ✅ **已修复** | `mapToModel` 已删除，转换逻辑内联到 handler |
| P0-3 | `scheduling_rule_service.go` `SchedulingRuleServiceImpl` 重复声明 | ✅ **已修复** | 只保留 service 文件中的唯一声明 |
| P0-4 | `wiring/container.go` 未使用的 import | ✅ **已修复** | import 已清理，所有包均被引用 |
| P0-5 | 缺少 import | ✅ **已修复** | 已补全 |
| P1-8 | `AddAssociations` Repository **Role 字段未映射**（连续 3 次评审未修复） | ✅ **已修复** | 代码中已添加 `Role: assoc.Role` |

#### AddAssociations Role 完整链路验证 ✅

```
请求层 (assoc.Role) → Handler (默认值 "target") → Model → Repository (Role: assoc.Role) → Entity → DB
                                                              ↕ Mapper (双向映射 Role)
```

### 🔴 新发现编译阻断（1 个）

| # | 文件 | 问题 | 修复方式 |
|---|------|------|---------|
| **NEW-1** | `rule_batch_handler.go` | 导入了 `"jusha/gantt/service/management/domain/service"` 但**从未通过 `service.` 前缀引用该包** — Go 编译器会报 `imported and not used` | 删除该 import 行 |

### ⚠️ 仍存在的建议项

- 缺少 `SourceType` 常量定义（`SourceTypeManual = "manual"` 等），各处硬编码字符串
- `BatchUpdateVersion` 逐条查询+更新，建议改为批量 SQL

---

## 第二部分：MCP Server — 重要更新

### ✅ 本轮修复（2 个 P1 问题解决）

| # | 上次问题 | 本次状态 | 说明 |
|---|---------|---------|------|
| P1-6 | `domain/model/rule.go` **完全缺失 V4 字段** | ✅ **已修复** | `Rule`/`UpdateRuleRequest`/`RuleAssociation` 全部补齐 V4 字段 |
| P1-7 | `update.go` 枚举完全过时，无 V4 字段 | ✅ **已修复** | InputSchema 枚举替换为 V4 值 + 新增 category/subCategory/sourceType/version 字段 |

#### MCP Domain Model V4 字段覆盖

| 结构体 | Category | SubCategory | OriginalRuleID | SourceType | ParseConfidence | Version | Role |
|--------|:--------:|:-----------:|:--------------:|:----------:|:---------------:|:-------:|:----:|
| `Rule` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| `CreateRuleRequest` | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | — |
| `UpdateRuleRequest` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| `ListRulesRequest` | ❌ | ❌ | — | ❌ | — | ❌ | — |
| `RuleAssociation` | — | — | — | — | — | — | ✅ |

### 🔴 新编译阻断（2 个）

| # | 文件 | 问题 | 严重度 |
|---|------|------|--------|
| **MCP-1** | `domain/model/rule.go` — `CreateRuleRequest` | **缺失 6 个 V4 字段**。但 `tool/rule/create.go` Execute 方法中赋值了 `req.Category`、`req.SubCategory` 等 → **编译失败** | 🔴 |
| **MCP-2** | `domain/model/rule.go` — `ListRulesRequest` | **缺失 4 个 V4 筛选字段**。但 `tool/rule/list.go` Execute 方法中赋值了 `req.Category` 等 → **编译失败** | 🔴 |

### 🟠 逻辑 Bug（1 个）

| # | 文件 | 问题 | 严重度 |
|---|------|------|--------|
| **MCP-3** | `tool/rule/update.go` Execute 方法 | InputSchema 已声明 V4 字段（category/subCategory/sourceType/version），但 **Execute 方法构建 `UpdateRuleRequest` 时未读取这些字段**，V4 输入被静默丢弃 | 🟠 |

#### update.go Execute 当前代码：
```go
req := &model.UpdateRuleRequest{
    OrgID:       common.GetString(input, "orgId"),
    Name:        common.GetString(input, "name"),
    RuleType:    common.GetString(input, "ruleType"),    // ✅
    ApplyScope:  common.GetString(input, "applyScope"),  // ✅
    TimeScope:   common.GetString(input, "timeScope"),   // ✅
    Description: common.GetString(input, "description"), // ✅
    // ❌ 缺少: Category, SubCategory, OriginalRuleID, SourceType, ParseConfidence, Version
}
```

### 🟡 工具桩代码（8 个，未变）

| 工具 | 状态 | 阻塞原因 |
|------|------|---------|
| `parse_rule.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `batch_parse_rules.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `get_statistics.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `get_dependencies.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `add_dependency.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `get_conflicts.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `preview_migration.go` | ❌ TODO 桩 | SDK Client 缺少方法 |
| `execute_migration.go` | ❌ TODO 桩 | SDK Client 缺少方法 |

> 8 个新工具已在 `tool/manager.go` 中注册 ✅，但全部返回固定/空数据。

---

## 第三部分：SDK — 无变更

### ✅ Model 层完备

- `Rule`/`CreateRuleRequest`/`UpdateRuleRequest`/`ListRulesRequest` 四个 struct 的 V4 字段齐全
- `RuleAssociation.Role` 已添加

### ❌ 仍缺失（与上次相同）

| 缺失项 | 影响 | 工时 |
|--------|------|------|
| `IRuleService` 接口无 V4 方法 | MCP 8 个新工具无法调用后端 | 1天 |
| Client 实现无 V4 方法 | 同上 | 1天 |
| 缺少 `RuleDependency`/`RuleConflict`/`RuleStatistics`/`MigrationPreview`/`ParseRequest`/`ParseResponse` 类型 | MCP 工具无法序列化 V4 关系数据 | 0.5天 |

> **这仍是当前 V4 数据链路的最大断层**。

---

## 第四部分：Frontend — 无变更

### ✅ 保持良好

| 功能 | 状态 |
|------|------|
| V4 类型定义 (model.d.ts) | ✅ 完备 |
| V4 枚举 (logic.ts) | ✅ 与后端一致 |
| V3 兼容映射 | ✅ 双向映射 + @deprecated |
| 分类 Tab | ✅ 5 个标签页 |
| 统计卡片 | ✅ 本地计算 |
| 规则解析对话框 | ✅ 3 步流程 |
| 依赖关系图 | ✅ ECharts |
| 规则迁移对话框 | ⚠️ API 注释 |
| API 函数 — parse/batch/organize | ✅ 已实现 |

### 🟡 仍存在问题（与上次相同）

| # | 文件 | 问题 | 严重度 |
|---|------|------|--------|
| F-1 | `RuleFormDialog.vue` | 无 category/subCategory UI 控件 | 🟡 |
| F-2 | `RuleFormDialog.vue` | 更新请求未传 V4 字段（category/subCategory/sourceType/version） | 🟡 |
| F-3 | `RuleFormDialog.vue` | `ruleDataPlaceholder` 用 V3 枚举匹配 V4 → placeholder 永远走 default | 🟡 |
| F-4 | `RuleFormDialog.vue` | Create 请求缺少 category/subCategory | 🟡 |
| F-5 | `api/index.ts` | 缺少 statistics/dependencies/conflicts/previewMigration/executeMigration API 函数 | 🟢 |
| F-6 | 统计卡片 | 基于当前页计算，非全量 API | 🟢 |

---

## 第五部分：全栈数据链路（更新）

### V4 基础 CRUD 链路

```
前端 (V4枚举) → Handler (V4字段+V3兼容) → Model → Entity → DB
                                              ↕
                                           Mapper (Role ✅)
                                              ↕
                                        SDK Model (V4字段✅)
                                              ↕
                                        SDK Client (无V4方法❌) ← 最大断层
                                              ↕
                                        MCP Domain Model (V4字段⚠️部分) ← CreateRuleRequest/ListRulesRequest 仍缺
                                              ↕
                                        MCP Tools (update Execute 漏传V4 ⚠️; 8个新工具桩❌)
```

### 各能力端到端状态

| 能力 | 后端API | SDK Model | SDK Client | MCP Tool | 前端UI | 端到端 |
|------|:-------:|:---------:|:----------:|:--------:|:------:|:------:|
| V4 CRUD(含分类) | ✅ | ✅ | ⚠️¹ | ⚠️² | ⚠️³ | **⚠️ 基础通，细节漏** |
| V3 枚举兼容 | ✅ | — | — | ✅⁴ | ✅ | **✅ 基本通** |
| 规则解析 (LLM) | ✅ | ❌ | ❌ | ❌桩 | ✅ | **前端→后端 通** |
| 规则组织 | ✅ | ❌ | ❌ | — | ✅ | **前端→后端 通** |
| 规则迁移 | ✅ | ❌ | ❌ | ❌桩 | ⚠️ | **❌ 不通** |
| 依赖/冲突 | ✅ | ❌ | ❌ | ❌桩 | ✅ | **前端→后端 通** |
| 统计 | ✅ | ❌ | ❌ | ❌桩 | ⚠️⁵ | **本地计算** |

> ¹ SDK 基础 CRUD 可传 V4 字段  
> ² MCP create/list 编译失败；update Execute 漏传 V4  
> ³ RuleFormDialog 缺 category/subCategory、更新丢失 V4  
> ⁴ MCP update.go 枚举已更新为 V4 ✅（本轮修复）  
> ⁵ 前端本地计算统计

---

## 第六部分：问题汇总（按优先级）

### 🔴 P0 — 编译阻断（3 个）

| # | 位置 | 问题 | 修复方式 | 工时 |
|---|------|------|---------|------|
| **NEW-1** | management-service `rule_batch_handler.go` | 未使用的 import `domain/service` | 删除该 import | 1分钟 |
| **MCP-1** | MCP `domain/model/rule.go` `CreateRuleRequest` | 缺失 6 个 V4 字段，导致 `create.go` 编译失败 | 补齐 Category/SubCategory/OriginalRuleID/SourceType/ParseConfidence/Version | 5分钟 |
| **MCP-2** | MCP `domain/model/rule.go` `ListRulesRequest` | 缺失 4 个 V4 字段，导致 `list.go` 编译失败 | 补齐 Category/SubCategory/SourceType/Version | 5分钟 |

### 🟠 P1 — 数据丢失（1 个）

| # | 位置 | 问题 | 修复方式 | 工时 |
|---|------|------|---------|------|
| **MCP-3** | MCP `tool/rule/update.go` Execute | V4 字段在 InputSchema 声明但 Execute 未读取赋值 → 用户传入的 V4 值被静默丢弃 | 在构建 `UpdateRuleRequest` 时补充读取 Category/SubCategory/OriginalRuleID/SourceType/ParseConfidence/Version | 10分钟 |

### 🟡 P2 — 功能缺失（仍存在，与上次相同）

| # | 位置 | 问题 | 工时 |
|---|------|------|------|
| 9 | SDK IRuleService + Client | 零 V4 新方法，阻断 MCP 8 个新工具 | 2天 |
| 10 | MCP 8 个新工具 | 全部桩代码 | 1天(SDK后) |
| 11 | 前端 `RuleFormDialog.vue` | 缺 category/subCategory 控件 + 更新丢失 V4 字段 | 0.5天 |
| 12 | 前端 `RuleMigrationDialog.vue` | API 调用被注释 | 0.5天(后端就绪后) |
| 13 | 前端 `api/index.ts` | 缺 statistics/dependencies/conflicts/migration API 函数 | 0.5天 |

### 🟢 P3 — 代码质量（仍存在）

| # | 位置 | 问题 |
|---|------|------|
| 14 | 领域模型 | 缺少 SourceType 常量定义 |
| 15 | `RuleFormDialog.vue` | `ruleDataPlaceholder` 用 V3 枚举匹配 |
| 16 | `scheduling_rule_service.go` | `BatchUpdateVersion` 逐条操作 |
| 17 | 前端统计卡片 | 基于当前页计算非全量 |

---

## 第七部分：建议实施路径

### 立即修复（30分钟）

```
1. management-service: 删除 rule_batch_handler.go 中未使用的 import
2. MCP domain/model/rule.go: CreateRuleRequest 补齐 6 个 V4 字段
3. MCP domain/model/rule.go: ListRulesRequest 补齐 4 个 V4 筛选字段
4. MCP tool/rule/update.go: Execute 方法补充读取 V4 字段赋值到 request
```

### 短期（Day 1-2）

```
5. SDK: IRuleService 接口添加 V4 方法签名
6. SDK: Client 实现 V4 方法（ParseRule/BatchParseRules/GetStatistics/GetDependencies/AddDependency/GetConflicts/PreviewMigration/ExecuteMigration/BatchSaveRules/OrganizeRules）
7. SDK: 添加 RuleDependency/RuleConflict/RuleStatistics/MigrationPreview/ParseRequest/ParseResponse 类型
8. MCP: 8 个新工具接入 SDK Client 真实调用
```

### 中期（Day 3-4）

```
9. 前端 RuleFormDialog: 添加 category/subCategory 选择控件
10. 前端 RuleFormDialog: Create/Update 请求补齐 V4 字段
11. 前端 RuleFormDialog: ruleDataPlaceholder 改用 V4 枚举值
12. 前端 api/index.ts: 补充缺失的 V4 API 函数
13. 前端 RuleMigrationDialog: 接通后端 API
14. 前端统计卡片: 改为后端 API 全量统计
```

---

## 第八部分：与上次评审对比

### 上次 P0 编译阻断（5 个）→ 本次状态

| # | 问题 | 状态 |
|---|------|------|
| P0-1 | `BatchSaveRules`/`OrganizeRules` 未实现 | ✅ **已修复** → 移到 `rule_batch_handler.go` |
| P0-2 | `mapToModel` 重复声明 | ✅ **已修复** → 已删除，逻辑内联 |
| P0-3 | `SchedulingRuleServiceImpl` 重复声明 | ✅ **已修复** → 合并为唯一声明 |
| P0-4 | `container.go` 未使用 import | ✅ **已修复** → 已清理 |
| P0-5 | 缺少 import | ✅ **已修复** → 已补全 |

### 上次 P1 数据链路断层（3 个）→ 本次状态

| # | 问题 | 状态 |
|---|------|------|
| P1-6 | MCP `domain/model/rule.go` 缺失 V4 字段 | ⚠️ **部分修复** — Rule/UpdateRuleRequest/RuleAssociation ✅，但 CreateRuleRequest/ListRulesRequest 仍缺 |
| P1-7 | MCP `update.go` 枚举过时 | ⚠️ **部分修复** — InputSchema ✅，但 Execute 未传 V4 字段 |
| P1-8 | `AddAssociations` Role 未映射 | ✅ **已修复**（历经 4 轮评审终于修复！） |

### 上次 P2 功能缺失 → 本次状态

| # | 问题 | 状态 |
|---|------|------|
| P2-9 | SDK Client 零 V4 方法 | ❌ **未变** |
| P2-10 | MCP 8 个新工具桩代码 | ❌ **未变** |
| P2-11 | 前端 RuleFormDialog 缺控件 | ❌ **未变** |
| P2-12 | 前端 RuleMigrationDialog API 注释 | ❌ **未变** |
| P2-13 | 前端 api/index.ts 缺 API 函数 | ❌ **未变** |

---

## 总结

### 本轮主要成果 ✅

1. **management-service 5 个编译阻断全部修复** — 后端应该可以正常编译运行
2. **AddAssociations Role 映射修复** — 历经 4 轮评审终于完成，完整链路验证通过
3. **MCP domain model 部分补齐** — Rule/UpdateRuleRequest/RuleAssociation V4 字段到位
4. **MCP update.go 枚举更新** — InputSchema 从 V3 全面切换到 V4

### 本轮遗留问题

1. **3 个新编译阻断**（1 个 management-service + 2 个 MCP）— 均为小问题，30分钟可修复
2. **MCP update.go Execute 漏传 V4 字段** — InputSchema 已声明但 Execute 未读取
3. **SDK Client 仍是最大断层** — 阻断 MCP 8 个新工具从桩变为真实调用
4. **前端 RuleFormDialog 问题未动** — 缺控件 + 请求丢字段
