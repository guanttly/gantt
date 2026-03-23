# 05. 前后端枚举统一方案

> **问题根源**: V3 前后端使用不同的枚举值，导致数据映射混乱  
> **目标**: V4 统一为一套枚举值，消除前后端不一致  
> **策略**: 以**后端枚举值为准**，前端全部切换到后端枚举值

## 1. 现状问题分析

### 1.1 RuleType 不一致

| 前端枚举值 | 后端枚举值 | 语义 | 差异分析 |
|-----------|-----------|------|---------|
| `max_shifts` | `maxCount` | 最大班次数 | 前端用蛇形，后端用驼峰 |
| `consecutive_shifts` | `maxCount` + `consecutiveMax` | 连续班次限制 | 前端当独立类型，后端用同一类型+参数区分 |
| `rest_days` | `maxCount` + `minRestDays` | 休息日要求 | 同上 |
| `forbidden_pattern` | `exclusive` | 禁止模式 | 完全不同的命名 |
| `preferred_pattern` | `preferred` | 偏好模式 | 前端多了 `_pattern` 后缀 |
| — | `combinable` | 可组合 | 前端缺少 |
| — | `required_together` | 必须同排 | 前端缺少 |
| — | `periodic` | 周期性 | 前端缺少 |
| — | `forbidden_day` | 禁止日期 | 前端缺少 |

### 1.2 ApplyScope 不一致

| 前端枚举值 | 后端枚举值 | 差异 |
|-----------|-----------|------|
| `global` | `global` | ✅ 一致 |
| `shift` | `specific` | ❌ 前端按对象类型细分，后端只有 global/specific |
| `group` | `specific` | ❌ |
| `employee` | `specific` | ❌ |

### 1.3 TimeScope 不一致

| 前端枚举值 | 后端枚举值 | 差异 |
|-----------|-----------|------|
| `daily` | `same_day` | ❌ 命名不同 |
| `weekly` | `same_week` | ❌ |
| `monthly` | `same_month` | ❌ |
| `custom` | `custom` | ✅ 一致 |

---

## 2. V4 统一枚举标准

### 2.1 原则

1. **以后端值为准**: 前端切换到后端的枚举值
2. **不改后端**: 后端枚举值保持不变，不引入新的值
3. **过渡性兼容**: 后端接口过渡期同时接受旧值，兼容代码隔离到 `v3_compat.go`，全量切换后移除
4. **前端废弃旧值**: 前端代码只使用新值，但显示文本可以保持中文

### 2.2 V4 统一枚举值

#### RuleType（统一为后端值）

```typescript
// V4 前端统一使用后端枚举值
type RuleType =
  | 'exclusive'         // 互斥（原 forbidden_pattern）
  | 'combinable'        // 可组合（新增）
  | 'required_together' // 必须同排（新增）
  | 'periodic'          // 周期性（新增）
  | 'maxCount'          // 数量限制（原 max_shifts + consecutive_shifts + rest_days）
  | 'forbidden_day'     // 禁止日期（新增）
  | 'preferred'         // 偏好（原 preferred_pattern）
```

#### ApplyScope（统一为后端值）

```typescript
// V4 前端统一使用后端枚举值
type ApplyScope = 'global' | 'specific'

// 前端原来的 shift/group/employee 通过关联关系的 targetType 体现
// 不再作为 applyScope 的值
```

#### TimeScope（统一为后端值）

```typescript
// V4 前端统一使用后端枚举值
type TimeScope = 'same_day' | 'same_week' | 'same_month' | 'custom'
```

---

## 3. 前端改造清单

### 3.1 logic.ts 改造

**改动对比**:

```typescript
// ==================== V3 旧值 → V4 新值 ====================

// --- RuleType ---
// 旧: { label: '最大班次数', value: 'max_shifts' }
// 新: { label: '数量限制', value: 'maxCount', description: '限制排班次数/天数' }
//
// 旧: { label: '连续班次', value: 'consecutive_shifts' }
// 新: 合并到 maxCount, 通过 consecutiveMax 参数区分
//
// 旧: { label: '休息日', value: 'rest_days' }
// 新: 合并到 maxCount, 通过 minRestDays 参数区分
//
// 旧: { label: '禁止模式', value: 'forbidden_pattern' }
// 新: { label: '互斥', value: 'exclusive' }
//
// 旧: { label: '偏好模式', value: 'preferred_pattern' }
// 新: { label: '偏好', value: 'preferred' }

// --- ApplyScope ---
// 旧: 'global' | 'shift' | 'group' | 'employee'
// 新: 'global' | 'specific'

// --- TimeScope ---
// 旧: 'daily' | 'weekly' | 'monthly' | 'custom'
// 新: 'same_day' | 'same_week' | 'same_month' | 'custom'
```

### 3.2 model.d.ts 改造

**RuleType 映射**:

```typescript
// V3 → V4 规则类型映射（用于显示历史数据）
export const RULE_TYPE_V3_TO_V4: Record<string, string> = {
  'max_shifts': 'maxCount',
  'consecutive_shifts': 'maxCount',
  'rest_days': 'maxCount',
  'forbidden_pattern': 'exclusive',
  'preferred_pattern': 'preferred',
}

// V3 → V4 时间范围映射
export const TIME_SCOPE_V3_TO_V4: Record<string, string> = {
  'daily': 'same_day',
  'weekly': 'same_week',
  'monthly': 'same_month',
  'custom': 'custom',
}

// V3 → V4 应用范围映射
export const APPLY_SCOPE_V3_TO_V4: Record<string, string> = {
  'global': 'global',
  'shift': 'specific',
  'group': 'specific',
  'employee': 'specific',
}
```

### 3.3 RuleFormDialog.vue 改造

```typescript
// 提交时统一转换为后端枚举值
function normalizeFormData(data: FormData): CreateRequest {
  return {
    ...data,
    ruleType: RULE_TYPE_V3_TO_V4[data.ruleType] || data.ruleType,
    applyScope: APPLY_SCOPE_V3_TO_V4[data.applyScope] || data.applyScope,
    timeScope: TIME_SCOPE_V3_TO_V4[data.timeScope] || data.timeScope,
  }
}
```

---

## 4. 后端兼容层

### 4.1 后端接收兼容

为避免前端升级期间出现问题，后端需要同时接受前端旧值和后端值。

> ⚠️ **以下所有 `normalizeXxx` 函数必须放在独立文件 `v3_compat.go` 中**，不得内联在 Handler 主逻辑里。详见 [02_backend_api_and_model.md](./02_backend_api_and_model.md#8-v3-兼容隔离与清理设计)。

**文件**: `services/management-service/internal/port/http/v3_compat.go`

```go
// @deprecated V3 兼容层 — 前端全部切换到后端枚举值后删除此文件

// normalizeRuleType 兼容前端旧枚举值
// @deprecated V3: 前端切换完成后移除
func normalizeRuleType(ruleType string) model.RuleType {
    switch ruleType {
    // 前端旧值 → 后端值
    case "max_shifts", "consecutive_shifts", "rest_days":
        return model.RuleTypeMaxCount
    case "forbidden_pattern":
        return model.RuleTypeExclusive
    case "preferred_pattern":
        return model.RuleTypePreferred
    // 后端值直接使用
    default:
        return model.RuleType(ruleType)
    }
}

// normalizeApplyScope 兼容前端旧枚举值
func normalizeApplyScope(scope string) model.ApplyScope {
    switch scope {
    case "shift", "group", "employee":
        return model.ApplyScopeSpecific
    default:
        return model.ApplyScope(scope)
    }
}

// normalizeTimeScope 兼容前端旧枚举值
func normalizeTimeScope(scope string) model.TimeScope {
    switch scope {
    case "daily":
        return model.TimeScopeSameDay
    case "weekly":
        return model.TimeScopeSameWeek
    case "monthly":
        return model.TimeScopeSameMonth
    default:
        return model.TimeScope(scope)
    }
}
```

### 4.2 后端查询兼容

```go
// ListRules 列表查询时，同时支持新旧枚举值
func (h *SchedulingRuleHandler) ListRules(c *gin.Context) {
    filter := &model.SchedulingRuleFilter{
        OrgID: c.Query("orgId"),
    }

    // 兼容 ruleType 参数
    if rt := c.Query("ruleType"); rt != "" {
        normalized := normalizeRuleType(rt)
        filter.RuleType = &normalized
    }

    // 兼容 applyScope 参数
    if as := c.Query("applyScope"); as != "" {
        normalized := normalizeApplyScope(as)
        filter.ApplyScope = &normalized
    }

    // 兼容 timeScope 参数
    if ts := c.Query("timeScope"); ts != "" {
        normalized := normalizeTimeScope(ts)
        filter.TimeScope = &normalized
    }

    // ... 继续处理其他参数
}
```

---

## 5. 数据库兼容处理

### 5.1 现有数据枚举值检查

在迁移前，需要确认数据库中实际存储的枚举值是前端值还是后端值：

```sql
-- 检查 rule_type 字段实际值
SELECT DISTINCT rule_type, COUNT(*) as cnt 
FROM scheduling_rules 
GROUP BY rule_type;

-- 检查 apply_scope 字段实际值
SELECT DISTINCT apply_scope, COUNT(*) as cnt 
FROM scheduling_rules 
GROUP BY apply_scope;

-- 检查 time_scope 字段实际值
SELECT DISTINCT time_scope, COUNT(*) as cnt 
FROM scheduling_rules 
GROUP BY time_scope;
```

### 5.2 数据修复脚本（如果数据库存了前端值）

```sql
-- 如果发现数据库中存了前端枚举值，执行修复

-- rule_type 修复
UPDATE scheduling_rules SET rule_type = 'maxCount' WHERE rule_type = 'max_shifts';
UPDATE scheduling_rules SET rule_type = 'maxCount' WHERE rule_type = 'consecutive_shifts';
UPDATE scheduling_rules SET rule_type = 'maxCount' WHERE rule_type = 'rest_days';
UPDATE scheduling_rules SET rule_type = 'exclusive' WHERE rule_type = 'forbidden_pattern';
UPDATE scheduling_rules SET rule_type = 'preferred' WHERE rule_type = 'preferred_pattern';

-- apply_scope 修复
UPDATE scheduling_rules SET apply_scope = 'specific' WHERE apply_scope IN ('shift', 'group', 'employee');

-- time_scope 修复
UPDATE scheduling_rules SET time_scope = 'same_day' WHERE time_scope = 'daily';
UPDATE scheduling_rules SET time_scope = 'same_week' WHERE time_scope = 'weekly';
UPDATE scheduling_rules SET time_scope = 'same_month' WHERE time_scope = 'monthly';
```

---

## 6. 分步执行计划

### Phase 1: 后端兼容层（Day 1-2）

1. 在 Handler 中添加 `normalizeXxx` 函数
2. 所有接收枚举参数的地方加入兼容转换
3. 测试：前端发旧值，后端能正确处理

### Phase 2: 数据库检查与修复（Day 3）

1. 运行 SQL 检查数据库中的实际枚举值
2. 如需修复，执行修复脚本
3. 验证修复结果

### Phase 3: 前端切换（Day 4-5）

1. 更新 `logic.ts` 枚举选项为后端值
2. 更新 `model.d.ts` 类型定义为后端值
3. 更新 `RuleFormDialog.vue` 表单选项
4. 在提交时确保使用新值
5. 在列表展示时处理新旧值显示

### Phase 4: 清理（全量迁移完成后，预计 1 人天）

> ⚠️ 过渡期结束标志：后端 `v3_compat.go` 中 Warn 日志连续 7 天无输出。

1. 确认前端全部使用新值（后端 Warn 日志无旧值请求）
2. 数据库中无旧枚举值残留：
   ```sql
   SELECT COUNT(*) FROM scheduling_rules WHERE rule_type IN ('max_shifts','consecutive_shifts','rest_days','forbidden_pattern','preferred_pattern');
   SELECT COUNT(*) FROM scheduling_rules WHERE apply_scope IN ('shift','group','employee');
   SELECT COUNT(*) FROM scheduling_rules WHERE time_scope IN ('daily','weekly','monthly');
   -- 以上全部必须为 0
   ```
3. 删除后端兼容文件：
   ```bash
   rm services/management-service/internal/port/http/v3_compat.go
   ```
4. 移除 Handler 中所有 `normalizeXxx` 调用（搜索 `@deprecated V3` 标记）
5. 删除前端 V3 兼容文件：
   ```bash
   rm frontend/web/src/pages/management/scheduling-rule/v3-compat.ts
   ```
6. 移除前端显示映射中所有 `@deprecated V3` 条目
7. 编译验证：`go build ./...` && `pnpm build`

---

## 7. 验证矩阵

| 场景 | 前端发送 | 后端接收 | DB 存储 | 前端展示 |
|------|---------|---------|---------|---------|
| V3前端 + V3后端 | `max_shifts` | `max_shifts` | `max_shifts` | "最大班次数" |
| V4前端 + V4后端（兼容层） | `maxCount` | `maxCount` | `maxCount` | "数量限制" |
| V3前端 + V4后端（兼容层） | `max_shifts` | → `maxCount` | `maxCount` | 需升级前端 |
| V4前端 + V3后端 | `maxCount` | ✅ 已是后端值 | `maxCount` | "数量限制" |
| 读取 V3 历史数据 | — | `maxCount` | `maxCount` | 通过映射表展示 |
| 读取未修复的 V3 数据 | — | `max_shifts` | `max_shifts` | 兼容展示 |

---

## 8. 前端显示文本映射

V4 统一枚举值后，前端需要一个全面的显示文本映射，兼容新旧值：

```typescript
/** 规则类型显示文本（兼容新旧枚举值） */
export const RULE_TYPE_LABELS: Record<string, string> = {
  // V4 后端枚举值
  'exclusive': '互斥',
  'combinable': '可组合',
  'required_together': '必须同排',
  'periodic': '周期性',
  'maxCount': '数量限制',
  'forbidden_day': '禁止日期',
  'preferred': '偏好',
  // @deprecated V3 前端旧枚举值（兼容显示，全量迁移后移除）
  'max_shifts': '数量限制',
  'consecutive_shifts': '连续班次限制',
  'rest_days': '休息日要求',
  'forbidden_pattern': '互斥',
  'preferred_pattern': '偏好',
}

/** 应用范围显示文本 */
export const APPLY_SCOPE_LABELS: Record<string, string> = {
  'global': '全局',
  'specific': '特定对象',
  // @deprecated V3 旧值（全量迁移后移除）
  'shift': '特定班次',
  'group': '特定分组',
  'employee': '特定员工',
}

/** 时间范围显示文本 */
export const TIME_SCOPE_LABELS: Record<string, string> = {
  'same_day': '同一天',
  'same_week': '同一周',
  'same_month': '同一月',
  'custom': '自定义',
  // @deprecated V3 旧值（全量迁移后移除）
  'daily': '同一天',
  'weekly': '同一周',
  'monthly': '同一月',
}
```
