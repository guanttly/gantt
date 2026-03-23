# 04. V3 → V4 规则迁移方案

> **迁移目标**: 将现有 V3 规则无缝升级到 V4 格式  
> **核心原则**: 不改变 V3 规则的任何现有字段值，只填充 V4 新增字段  
> **回滚保证**: 迁移可逆，随时可回滚到 V3 状态  
> **清理设计**: 迁移服务本身也是过渡性代码，全量迁移完成后整个 `internal/migration/` 包可删除

## 1. 迁移策略

### 1.1 三阶段迁移

```
阶段一: 自动推断     阶段二: LLM 辅助     阶段三: 人工审核
━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━
                    
V3 RuleType         V3 RuleData          管理员逐条
  → Category          → LLM 解析            审核确认
  → SubCategory       → 更精确的分类        → 最终保存
                      → 关联 Role 推断
                    
 自动完成             可选步骤              可选步骤
 100% 覆盖           提高准确度            保证正确性
```

### 1.2 迁移不修改的字段

| 字段 | 说明 |
|------|------|
| `id` | 保持不变 |
| `orgId` | 保持不变 |
| `name` | 保持不变 |
| `description` | 保持不变 |
| `ruleType` | **保持不变**（不转换枚举值） |
| `applyScope` | **保持不变** |
| `timeScope` | **保持不变** |
| `ruleData` | 保持不变 |
| `maxCount/consecutiveMax/intervalDays/minRestDays` | 保持不变 |
| `priority` | 保持不变 |
| `isActive` | 保持不变 |
| `associations` | 保持不变（仅补充 Role） |

### 1.3 迁移填充的字段

| 字段 | 填充策略 | 说明 |
|------|---------|------|
| `category` | 自动推断 | 根据 ruleType 映射 |
| `subCategory` | 自动推断 | 根据 ruleType 映射 |
| `version` | 设为 `"v4"` | 标记已迁移 |
| `sourceType` | 设为 `"migrated"` | 标记来源 |
| `associations[].role` | 全部设为 `"target"` | V3 没有 Role 概念，默认都是 target |

---

## 2. 自动推断映射表

### 2.1 RuleType → Category + SubCategory 映射

```
后端 RuleType          →  Category       SubCategory    推断逻辑
━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━  ━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━
exclusive              constraint     forbid          互斥 = 禁止型约束
combinable             preference     combinable      可组合 = 偏好型可合并
required_together      constraint     must            必须同排 = 必须型约束
periodic               constraint     limit           周期性 = 限制型约束
maxCount               constraint     limit           数量限制 = 限制型约束
forbidden_day          constraint     forbid          禁止日期 = 禁止型约束
preferred              preference     prefer          偏好 = 偏好型优先
```

### 2.2 Go 实现

```go
// RuleTypeToV4Category 根据 V3 规则类型推断 V4 分类
// 这是确定性映射，不需要 LLM
func RuleTypeToV4Category(ruleType string) (category, subCategory string) {
    switch ruleType {
    case "exclusive":
        return "constraint", "forbid"
    case "combinable":
        return "preference", "combinable"
    case "required_together":
        return "constraint", "must"
    case "periodic":
        return "constraint", "limit"
    case "maxCount":
        return "constraint", "limit"
    case "forbidden_day":
        return "constraint", "forbid"
    case "preferred":
        return "preference", "prefer"
    default:
        return "constraint", "limit" // 未知类型默认当约束处理
    }
}
```

### 2.3 前端 RuleType 映射（处理前端历史数据）

由于前端曾使用不同的枚举值，如果数据库中存了前端的枚举值，需要额外处理：

```go
// FrontendRuleTypeToBackend 前端枚举值转后端枚举值
// 如果数据库中存了前端的 ruleType，用此函数转换
func FrontendRuleTypeToBackend(frontendType string) string {
    switch frontendType {
    case "max_shifts":
        return "maxCount"
    case "consecutive_shifts":
        return "maxCount" // 用 consecutiveMax 参数区分
    case "rest_days":
        return "maxCount" // 用 minRestDays 参数区分
    case "forbidden_pattern":
        return "exclusive"
    case "preferred_pattern":
        return "preferred"
    default:
        return frontendType // 已经是后端枚举值
    }
}
```

---

## 3. 迁移服务实现

### 3.1 服务接口

**文件**: `services/management-service/internal/migration/rule_migration.go`

> ⚠️ **整个 `internal/migration/` 包为 V3 过渡代码**，全量迁移完成后整包删除。

```go
package migration

import "context"

// IRuleMigrationService 规则迁移服务
type IRuleMigrationService interface {
    // PreviewMigration 预览迁移结果（不执行）
    PreviewMigration(ctx context.Context, orgID string) (*MigrationPreview, error)

    // ExecuteMigration 执行迁移
    ExecuteMigration(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)

    // RollbackMigration 回滚迁移
    RollbackMigration(ctx context.Context, orgID string, ruleIDs []string) error

    // GetMigrationStatus 获取迁移状态
    GetMigrationStatus(ctx context.Context, orgID string) (*MigrationStatus, error)
}
```

### 3.2 类型定义

```go
// MigrationPreview 迁移预览结果
type MigrationPreview struct {
    TotalV3Rules    int              `json:"totalV3Rules"`
    AutoMigratable  []*MigrationItem `json:"autoMigratable"`  // 可自动迁移
    NeedsReview     []*MigrationItem `json:"needsReview"`     // 需人工审核
}

// MigrationItem 单条迁移项
type MigrationItem struct {
    RuleID               string  `json:"ruleId"`
    RuleName             string  `json:"ruleName"`
    CurrentRuleType      string  `json:"currentRuleType"`
    CurrentRuleData      string  `json:"currentRuleData"`
    SuggestedCategory    string  `json:"suggestedCategory"`
    SuggestedSubCategory string  `json:"suggestedSubCategory"`
    Confidence           float64 `json:"confidence"`
    Reason               string  `json:"reason"`
    NeedsReview          bool    `json:"needsReview"`
}

// ExecuteRequest 执行迁移请求
type ExecuteRequest struct {
    OrgID     string   `json:"orgId"`
    RuleIDs   []string `json:"ruleIds"`   // 指定迁移的规则ID列表
    AutoApply bool     `json:"autoApply"` // 是否自动应用（true=不审核直接迁移）
}

// ExecuteResult 迁移执行结果
type ExecuteResult struct {
    SuccessCount int      `json:"successCount"`
    FailedCount  int      `json:"failedCount"`
    FailedRules  []string `json:"failedRules"` // 失败的规则ID
}

// MigrationStatus 迁移状态
type MigrationStatus struct {
    TotalRules     int `json:"totalRules"`
    V3Count        int `json:"v3Count"`        // 未迁移
    V4Count        int `json:"v4Count"`        // 已迁移
    MigratedCount  int `json:"migratedCount"`  // sourceType=migrated 的数量
}
```

### 3.3 服务实现

```go
package migration

import (
    "context"
    "jusha/gantt/service/management/domain/model"
    "jusha/gantt/service/management/domain/service"
)

type RuleMigrationService struct {
    ruleService service.ISchedulingRuleService
}

func NewRuleMigrationService(ruleService service.ISchedulingRuleService) *RuleMigrationService {
    return &RuleMigrationService{ruleService: ruleService}
}

// PreviewMigration 预览迁移
func (s *RuleMigrationService) PreviewMigration(ctx context.Context, orgID string) (*MigrationPreview, error) {
    // 获取所有 V3 规则（version 为空或 "v3"）
    v3Rules, err := s.ruleService.GetV3Rules(ctx, orgID)
    if err != nil {
        return nil, err
    }

    preview := &MigrationPreview{
        TotalV3Rules: len(v3Rules),
    }

    for _, rule := range v3Rules {
        category, subCategory := RuleTypeToV4Category(string(rule.RuleType))
        
        item := &MigrationItem{
            RuleID:               rule.ID,
            RuleName:             rule.Name,
            CurrentRuleType:      string(rule.RuleType),
            CurrentRuleData:      rule.RuleData,
            SuggestedCategory:    category,
            SuggestedSubCategory: subCategory,
        }

        // 判断是否需要人工审核
        switch {
        case rule.RuleData != "" && isAmbiguous(rule):
            // ruleData 中的描述可能暗示不同的分类
            item.NeedsReview = true
            item.Confidence = 0.6
            item.Reason = "规则描述可能暗示不同的分类，建议人工确认"
            preview.NeedsReview = append(preview.NeedsReview, item)
        case rule.RuleType == "":
            // 没有 ruleType 的规则
            item.NeedsReview = true
            item.Confidence = 0.3
            item.Reason = "规则类型为空，无法自动推断"
            preview.NeedsReview = append(preview.NeedsReview, item)
        default:
            // 确定性映射，高置信度
            item.Confidence = 0.95
            item.Reason = "基于规则类型 " + string(rule.RuleType) + " 自动推断"
            preview.AutoMigratable = append(preview.AutoMigratable, item)
        }
    }

    return preview, nil
}

// ExecuteMigration 执行迁移
func (s *RuleMigrationService) ExecuteMigration(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
    result := &ExecuteResult{}

    for _, ruleID := range req.RuleIDs {
        rule, err := s.ruleService.GetRule(ctx, req.OrgID, ruleID)
        if err != nil {
            result.FailedCount++
            result.FailedRules = append(result.FailedRules, ruleID)
            continue
        }

        // 推断 V4 分类
        category, subCategory := RuleTypeToV4Category(string(rule.RuleType))

        // 更新 V4 字段
        rule.Category = category
        rule.SubCategory = subCategory
        rule.Version = "v4"
        rule.SourceType = "migrated"

        // 补充关联 Role
        for i := range rule.Associations {
            if rule.Associations[i].Role == "" {
                rule.Associations[i].Role = "target" // 默认 target
            }
        }

        if err := s.ruleService.UpdateRule(ctx, rule); err != nil {
            result.FailedCount++
            result.FailedRules = append(result.FailedRules, ruleID)
        } else {
            result.SuccessCount++
        }
    }

    return result, nil
}

// RollbackMigration 回滚迁移
func (s *RuleMigrationService) RollbackMigration(ctx context.Context, orgID string, ruleIDs []string) error {
    for _, ruleID := range ruleIDs {
        rule, err := s.ruleService.GetRule(ctx, orgID, ruleID)
        if err != nil {
            continue
        }

        // 清空 V4 字段
        rule.Category = ""
        rule.SubCategory = ""
        rule.Version = ""
        rule.SourceType = ""

        // 清空关联 Role
        for i := range rule.Associations {
            rule.Associations[i].Role = ""
        }

        _ = s.ruleService.UpdateRule(ctx, rule)
    }
    return nil
}

// isAmbiguous 判断规则是否模糊（需要人工审核）
func isAmbiguous(rule *model.SchedulingRule) bool {
    // 如果 ruleData 包含多种可能的语义
    // 例如 ruleType 是 maxCount 但 ruleData 描述的像是偏好
    if rule.RuleData == "" {
        return false
    }
    
    // 简单的关键词检测
    // 如果 ruleType 是约束型但 ruleData 含"尽量"/"优先"/"建议"等偏好词
    if isConstraintType(string(rule.RuleType)) {
        keywords := []string{"尽量", "优先", "建议", "最好", "如果可能"}
        for _, kw := range keywords {
            if strings.Contains(rule.RuleData, kw) {
                return true
            }
        }
    }
    
    return false
}

func isConstraintType(ruleType string) bool {
    switch ruleType {
    case "exclusive", "required_together", "maxCount", "forbidden_day", "periodic":
        return true
    }
    return false
}
```

---

## 4. LLM 辅助迁移（可选增强）

对于 `NeedsReview` 的规则，可以通过 LLM 进一步分析 `ruleData` 来推荐更精确的分类。

### 4.1 LLM 迁移 Prompt

```go
const MigrationPrompt = `你是排班规则分类专家。请根据以下V3规则信息，推荐V4分类。

规则名称: %s
规则类型(V3): %s
规则描述: %s
当前关联: %s

请从以下分类中选择最合适的：

category:
- constraint (约束型): 必须遵守的硬规则
- preference (偏好型): 尽量满足的软规则
- dependency (依赖型): 定义执行顺序或来源关系

subCategory:
constraint下: forbid(禁止) / limit(限制) / must(必须)
preference下: prefer(优先) / suggest(建议) / combinable(可合并)
dependency下: source(来源依赖) / resource(资源预留) / order(顺序依赖)

如果规则描述中提到"来自"、"必须从...中选"，可能是dependency/source。
如果描述中有"尽量"、"优先"，是preference。
如果描述中有"不能"、"禁止"、"最多"、"至少"，是constraint。

请输出JSON:
{
  "category": "...",
  "subCategory": "...",
  "confidence": 0.0-1.0,
  "reason": "推理理由"
}
`
```

### 4.2 调用时机

```
迁移预览时，对 NeedsReview 的规则:
  1. 先用确定性映射给出默认值
  2. 可选地调用 LLM 分析 ruleData 获取更精确的推荐
  3. 将两个结果都展示给用户，用户选择最终值
```

---

## 5. 迁移 API

### 5.1 端点定义

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules/migration/preview` | 预览迁移结果 |
| POST | `/v1/rules/migration/execute` | 执行迁移 |
| POST | `/v1/rules/migration/rollback` | 回滚迁移 |
| GET | `/v1/rules/migration/status` | 迁移状态 |

### 5.2 请求/响应示例

```
GET /v1/rules/migration/preview?orgId=org-001

Response:
{
  "totalV3Rules": 15,
  "autoMigratable": [
    {
      "ruleId": "rule-001",
      "ruleName": "早班数量限制",
      "currentRuleType": "maxCount",
      "suggestedCategory": "constraint",
      "suggestedSubCategory": "limit",
      "confidence": 0.95,
      "reason": "基于规则类型 maxCount 自动推断"
    }
  ],
  "needsReview": [
    {
      "ruleId": "rule-005",
      "ruleName": "尽量少排夜班",
      "currentRuleType": "maxCount",
      "currentRuleData": "尽量不要连续排夜班",
      "suggestedCategory": "constraint",
      "suggestedSubCategory": "limit",
      "confidence": 0.6,
      "reason": "规则类型为maxCount(约束)但描述含'尽量'(偏好)，建议人工确认"
    }
  ]
}
```

```
POST /v1/rules/migration/execute
{
  "orgId": "org-001",
  "ruleIds": ["rule-001", "rule-002", "rule-003"],
  "autoApply": true
}

Response:
{
  "successCount": 3,
  "failedCount": 0,
  "failedRules": []
}
```

---

## 6. 迁移验证检查清单

### 6.1 迁移前检查

- [ ] 备份 `scheduling_rules` 和 `rule_associations` 表
- [ ] 确认 V4 DDL 已执行（新增列 + 新表）
- [ ] 确认前端已部署 V4 版本（支持新字段展示）

### 6.2 迁移中检查

- [ ] 每条规则的 `category` + `subCategory` 与 `ruleType` 逻辑一致
- [ ] `version` 字段已设为 `"v4"`
- [ ] `sourceType` 字段已设为 `"migrated"`
- [ ] 所有 Association 的 `role` 字段已设为 `"target"`
- [ ] 原有字段（ruleType/applyScope/timeScope 等）未被修改

### 6.3 迁移后检查

- [ ] V3 规则列表页过渡期正常显示（过渡性兼容，全部迁移后移除此检查项）
- [ ] V4 分类 Tab 过滤正常工作
- [ ] V4 统计卡片数据正确
- [ ] 排班引擎能正确读取 V4 字段
- [ ] MCP Server 返回的规则包含 V4 字段
- [ ] 回滚测试：清空 V4 字段后系统恢复到 V3 状态（过渡期检查，确认全量迁移后此项可删除）

---

## 7. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 数据库枚举不一致（前端值存入DB） | 中 | 高 | 迁移时统一检测并转换 |
| RuleData 语义与 RuleType 矛盾 | 中 | 低 | NeedsReview 机制，人工确认 |
| 迁移中断导致部分迁移 | 低 | 中 | 事务保护 + 回滚接口 |
| 迁移后排班引擎行为变化 | 低 | 高 | V4 字段过渡期 optional，不影响 V3 逻辑；全量迁移后切换为 required |

---

## 8. Phase 4: V3 全面清理（全量迁移完成后执行）

> ⚠️ 迁移服务本身也是过渡代码，全量迁移完成后迁移服务也要被清理。

### 8.1 清理触发条件

以下条件全部满足时可触发全面清理：

| 条件 | 验证方法 |
|------|----------|
| DB 中无 V3 规则 | `SELECT COUNT(*) FROM scheduling_rules WHERE version != 'v4' OR version IS NULL` → 0 |
| 前端无 V3 枚举请求 | 后端 `v3_compat.go` 中 Warn 日志连续 7 天无输出 |
| 迁移 API 无调用 | `/v1/rules/migration/*` 端点连续 30 天无调用 |
| 产品确认 | V4 已稳定运行 >= 2 周，产品经理确认无回滚需求 |

### 8.2 清理范围清单

#### 后端清理

| 删除目标 | 说明 |
|----------|------|
| `services/management-service/internal/migration/` | 整个迁移包 |
| `services/management-service/internal/port/http/v3_compat.go` | 枚举兼容层 |
| `mcp-servers/rostering/tool/rule/preview_migration.go` | MCP 迁移工具 |
| `mcp-servers/rostering/tool/rule/execute_migration.go` | MCP 迁移工具 |
| `/v1/rules/migration/*` 路由注册 | 迁移 API 端点 |
| Handler 中 `normalizeXxx` / `FillV3Defaults` 调用 | 搜索 `@deprecated V3` 标记 |
| Model 中 `Version` 字段 | 全部为 v4 后无需区分 |
| `sourceType = "migrated"` | 统一为 manual 或保留作为历史记录 |

#### 前端清理

| 删除目标 | 说明 |
|----------|------|
| `v3-compat.ts` | V3 枚举映射文件 |
| `RuleMigrationDialog.vue` | 迁移对话框组件 |
| `logic.ts` 中 `...v3Compat.*` 展开 | V3 标签合并 |
| `index.vue` 中 "未分类/V3" Tab | 过渡分类 Tab |
| 统计卡片 "待迁移" 指标 | 过渡指标 |
| i18n 中 `"migrated": "V3 迁移"` | 过渡文本 |

#### 数据库清理

```sql
-- Phase 4 清理 SQL
ALTER TABLE scheduling_rules DROP COLUMN version;
ALTER TABLE scheduling_rules MODIFY COLUMN category VARCHAR(20) NOT NULL;
ALTER TABLE scheduling_rules MODIFY COLUMN sub_category VARCHAR(20) NOT NULL;
-- 可选：将 source_type='migrated' 统一为 'manual'
UPDATE scheduling_rules SET source_type = 'manual' WHERE source_type = 'migrated';
```

### 8.3 清理验证

```bash
# 1. 编译验证
cd services/management-service && go build ./...
cd mcp-servers/rostering && go build ./...
cd frontend/web && pnpm build

# 2. 功能验证
# - 规则 CRUD 正常
# - 分类过滤正常
# - LLM 解析正常
# - 排班引擎正常读取规则

# 3. 确认无残留引用
grep -rn 'v3_compat\|v3-compat\|@deprecated V3\|normalizeRuleType\|FillV3Defaults\|migration' \
  services/management-service/ mcp-servers/rostering/ frontend/web/src/
```
