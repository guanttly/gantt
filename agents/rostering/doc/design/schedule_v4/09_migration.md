# 09. V3→V4 迁移方案

> **开发负责人**: Agent-7  
> **依赖**: 所有 Agent（迁移在功能开发完成后执行）  
> **优先级**: 中（可与功能开发并行设计，但执行在后）

## 1. 迁移原则

1. **零停机**: V3 和 V4 共存，通过 Feature Flag 控制
2. **数据兼容**: V4 新增字段全部可空，V3 数据无需强制迁移
3. **渐进迁移**: 可以逐个组织切换到 V4
4. **可回滚**: 任何阶段可以回退到 V3

## 2. 迁移阶段

### 阶段 1: 数据库迁移（Day 1）

执行 DDL 变更，不影响 V3 运行：

```sql
-- 1. 扩展 scheduling_rules 表（新增V4字段，全部可空）
ALTER TABLE scheduling_rules 
    ADD COLUMN category VARCHAR(32) DEFAULT NULL COMMENT '规则大类',
    ADD COLUMN sub_category VARCHAR(32) DEFAULT NULL COMMENT '规则子类',
    ADD COLUMN original_rule_id VARCHAR(64) DEFAULT NULL COMMENT '原始规则ID';

-- 2. 扩展 rule_associations 表（新增role字段）
ALTER TABLE rule_associations
    ADD COLUMN role VARCHAR(16) DEFAULT 'target' COMMENT '关联角色';

-- 3. 创建新表
-- (见 03_data_model.md 中的 DDL)
-- rule_dependencies
-- rule_conflicts  
-- shift_dependencies
```

### 阶段 2: V4 代码部署（Week 2-4）

部署 V4 代码，但默认不启用：

```yaml
# config/agents/rostering-agent.yml
scheduling:
  version: "v3"        # 默认仍使用 V3
  v4_enabled: false     # V4 功能总开关
  v4_orgs: []           # 启用 V4 的组织ID列表（白名单模式）
```

### 阶段 3: 规则数据补充（Week 3-5）

为已有 V3 规则补充 V4 字段（自动 + 人工）：

```sql
-- 自动补充: 根据 ruleType 推断 category/subCategory
UPDATE scheduling_rules SET category='constraint', sub_category='frequency' 
    WHERE rule_type='maxCount' AND category IS NULL;

UPDATE scheduling_rules SET category='constraint', sub_category='continuity' 
    WHERE rule_type='consecutiveMax' AND category IS NULL;

UPDATE scheduling_rules SET category='constraint', sub_category='rest' 
    WHERE rule_type='minRestDays' AND category IS NULL;

UPDATE scheduling_rules SET category='constraint', sub_category='exclusive' 
    WHERE rule_type='exclusive' AND category IS NULL;

UPDATE scheduling_rules SET category='constraint', sub_category='forbidden' 
    WHERE rule_type='forbidden_day' AND category IS NULL;

-- 检查未覆盖的规则
SELECT id, name, rule_type FROM scheduling_rules WHERE category IS NULL;
```

### 阶段 4: 灰度发布（Week 5-6）

按组织逐步启用 V4：

```yaml
scheduling:
  version: "v3"
  v4_enabled: true
  v4_orgs: ["org_pilot_001"]  # 先用试点组织
```

### 阶段 5: 全量切换（Week 7+）

```yaml
scheduling:
  version: "v4"
  v4_enabled: true
  v3_fallback: true   # 保留 V3 回退能力
```

## 3. Feature Flag 实现

```go
// pkg/config/scheduling.go

type SchedulingConfig struct {
    Version     string   `yaml:"version"`     // "v3" / "v4"
    V4Enabled   bool     `yaml:"v4_enabled"`
    V4Orgs      []string `yaml:"v4_orgs"`     // 白名单
    V3Fallback  bool     `yaml:"v3_fallback"` // V4 失败时回退到 V3
}

// ShouldUseV4 判断是否使用 V4
func (c *SchedulingConfig) ShouldUseV4(orgID string) bool {
    if !c.V4Enabled {
        return false
    }
    if c.Version == "v4" {
        return true
    }
    // 白名单模式
    for _, id := range c.V4Orgs {
        if id == orgID {
            return true
        }
    }
    return false
}
```

### 工作流路由

```go
// agents/rostering/internal/workflow/router.go

func GetScheduleWorkflow(orgID string, config *SchedulingConfig) string {
    if config.ShouldUseV4(orgID) {
        return state.WorkflowScheduleCreateV4
    }
    return schedule.WorkflowScheduleCreateV3
}
```

## 4. 数据迁移脚本

### 4.1 V3 规则 → V4 规则字段补充

```go
// scripts/migrate_rules_v4.go

func MigrateRulesToV4(db *gorm.DB) error {
    // 1. 自动推断 category/subCategory
    ruleTypeMapping := map[string][2]string{
        "maxCount":       {"constraint", "frequency"},
        "consecutiveMax": {"constraint", "continuity"},
        "minRestDays":    {"constraint", "rest"},
        "exclusive":      {"constraint", "exclusive"},
        "forbidden_day":  {"constraint", "forbidden"},
        "required_together": {"constraint", "headcount"},
        "preferred":      {"preference", "balance"},
    }
    
    for ruleType, cats := range ruleTypeMapping {
        result := db.Model(&Rule{}).
            Where("rule_type = ? AND category IS NULL", ruleType).
            Updates(map[string]interface{}{
                "category":     cats[0],
                "sub_category": cats[1],
            })
        log.Printf("Updated %d rules: %s -> %s/%s", 
            result.RowsAffected, ruleType, cats[0], cats[1])
    }
    
    // 2. 为所有 association 设置默认 role
    db.Model(&RuleAssociation{}).
        Where("role IS NULL OR role = ''").
        Update("role", "target")
    
    // 3. 统计未迁移的规则
    var unmigrated int64
    db.Model(&Rule{}).Where("category IS NULL").Count(&unmigrated)
    if unmigrated > 0 {
        log.Printf("WARNING: %d rules still have no category, need manual review", unmigrated)
    }
    
    return nil
}
```

### 4.2 LLM 辅助迁移（针对无法自动推断的规则）

```go
// scripts/migrate_rules_llm.go

func MigrateUnclassifiedRules(db *gorm.DB, parser *rule_parser.RuleParser) error {
    var rules []Rule
    db.Where("category IS NULL").Find(&rules)
    
    for _, rule := range rules {
        result, err := parser.Parse(context.Background(), &rule_parser.ParseRequest{
            OrgID:    rule.OrgID,
            RuleText: rule.Description,
        })
        if err != nil {
            log.Printf("Failed to parse rule %s: %v", rule.ID, err)
            continue
        }
        
        db.Model(&rule).Updates(map[string]interface{}{
            "category":     result.Category,
            "sub_category": result.SubCategory,
        })
    }
    
    return nil
}
```

## 5. 回滚方案

### 快速回滚（< 5 分钟）

```yaml
# 直接切换 version
scheduling:
  version: "v3"
  v4_enabled: false
```

### 数据回滚（如需）

```sql
-- V4 新增字段不影响 V3 运行，通常无需清理
-- 如确需回滚数据：

-- 清理 V4 专有表
TRUNCATE TABLE rule_dependencies;
TRUNCATE TABLE rule_conflicts;
TRUNCATE TABLE shift_dependencies;

-- V4 字段置空（不影响 V3）
UPDATE scheduling_rules SET category = NULL, sub_category = NULL, original_rule_id = NULL;
UPDATE rule_associations SET role = 'target';
```

## 6. 兼容性矩阵

| 组件 | V3 Only | V3+V4 共存 | V4 Only |
|------|---------|-----------|---------|
| scheduling_rules 表 | ✅ 正常 | ✅ V4 字段可空 | ✅ 正常 |
| rule_associations 表 | ✅ role 有默认值 | ✅ 兼容 | ✅ 正常 |
| V3 工作流 | ✅ 正常 | ✅ 按 orgID 路由 | ❌ 不使用 |
| V4 工作流 | ❌ 不使用 | ✅ 按 orgID 路由 | ✅ 正常 |
| 规则引擎 | ❌ 不存在 | ✅ V4 专用 | ✅ 正常 |
| 规则解析器 | ❌ 不存在 | ✅ 管理端使用 | ✅ 正常 |
| 前端规则管理 | ✅ 旧版 | ✅ 新旧并存 | ✅ V4 版 |
| LLM 调用 | 5次/班次/日 | 混合 | 1次/班次/日 |

## 7. 监控指标

迁移期间需要监控的关键指标：

| 指标 | V3 基准 | V4 目标 | 报警阈值 |
|------|--------|---------|---------|
| LLM 调用次数/次排班 | ~720 | ~160 | > 200 |
| 单班次排班延迟 | ~11s | ~5s | > 8s |
| 规则校验通过率 | ~85% | > 95% | < 90% |
| Token 消耗/次排班 | ~300K | ~60K | > 100K |
| 排班一致性（重跑一致率） | ~60% | > 90% | < 80% |
| 规则解析成功率 | N/A | > 95% | < 90% |

## 8. 迁移检查清单

- [ ] DDL 变更已在测试环境验证
- [ ] V3 在新 DDL 下运行正常（回归测试）
- [ ] V4 Feature Flag 默认关闭
- [ ] 自动规则分类脚本已准备
- [ ] 试点组织已选定
- [ ] 监控面板已配置
- [ ] 回滚脚本已准备并测试
- [ ] 前端兼容性已验证（V3 规则在 V4 页面正常展示）
