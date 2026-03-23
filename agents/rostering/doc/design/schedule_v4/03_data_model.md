# 03. 数据模型与数据库设计

> **开发负责人**: Agent-1  
> **依赖**: 无  
> **被依赖**: Agent-2(规则引擎), Agent-3(工作流), Agent-4(规则解析), Agent-5(API)

## 1. SDK Model 扩展

### 1.1 Rule 结构体扩展

**文件**: `sdk/rostering/model/rule.go`

在现有 `Rule` 结构体中新增以下字段：

```go
type Rule struct {
    // === 现有字段（不修改）===
    ID          string `json:"id"`
    OrgID       string `json:"orgId"`
    Name        string `json:"name"`
    Description string `json:"description"`
    RuleType    string `json:"ruleType"`
    ApplyScope  string `json:"applyScope,omitempty"`
    TimeScope   string `json:"timeScope,omitempty"`
    RuleData    string `json:"ruleData,omitempty"`
    MaxCount       *int `json:"maxCount,omitempty"`
    ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
    IntervalDays   *int `json:"intervalDays,omitempty"`
    MinRestDays    *int `json:"minRestDays,omitempty"`
    Priority  int        `json:"priority"`
    IsActive  bool       `json:"isActive,omitempty"`
    Status    string     `json:"status,omitempty"`
    Config    string     `json:"config,omitempty"`
    ValidFrom *time.Time `json:"validFrom,omitempty"`
    ValidTo   *time.Time `json:"validTo,omitempty"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
    DeletedAt *time.Time `json:"deletedAt,omitempty"`
    Associations []RuleAssociation `json:"associations,omitempty"`

    // === V4 新增字段 ===
    
    // Category 规则分类: constraint(约束型) / preference(偏好型) / dependency(依赖型)
    // 默认值: "" (空，向后兼容V3)
    Category string `json:"category,omitempty"`
    
    // SubCategory 规则子分类
    // constraint: forbid / limit / must
    // preference: prefer / suggest / combinable
    // dependency: source / resource / order
    SubCategory string `json:"subCategory,omitempty"`
    
    // OriginalRuleID 原始规则ID（如果是从语义化输入拆解出来的）
    // 多条解析出的规则共享同一个 OriginalRuleID
    OriginalRuleID string `json:"originalRuleId,omitempty"`
}
```

### 1.2 RuleAssociation 结构体扩展

**文件**: `sdk/rostering/model/rule.go`

```go
type RuleAssociation struct {
    // === 现有字段（不修改）===
    ID              string `json:"id,omitempty"`
    RuleID          string `json:"ruleId"`
    AssociationType string `json:"associationType"` // employee, shift, group
    AssociationID   string `json:"associationId"`
    TargetType      string `json:"targetType,omitempty"`
    TargetID        string `json:"targetId,omitempty"`

    // === V4 新增字段 ===
    
    // Role 关联角色，定义该对象在规则中扮演的角色
    // "target"    - 被约束的对象（规则作用目标），如"下夜班不能超过3次"中的"下夜班"
    // "source"    - 数据来源（依赖型规则），如"下夜班人员必须来自上半夜班"中的"上半夜班"
    // "reference" - 引用对象（排他规则），如"排了A班不能排B班"中的"B班"
    // 默认值: "target" (向后兼容V3，所有现有关联都视为 target)
    Role string `json:"role,omitempty"`
}
```

### 1.3 CreateRuleRequest / UpdateRuleRequest 扩展

同样新增 `Category`、`SubCategory`、`OriginalRuleID` 字段。

### 1.4 ListRulesRequest 扩展

```go
type ListRulesRequest struct {
    // === 现有字段 ===
    OrgID      string `json:"orgId"`
    RuleType   string `json:"type"`
    ApplyScope string `json:"applyScope"`
    TimeScope  string `json:"timeScope"`
    IsActive   *bool  `json:"isActive"`
    Keyword    string `json:"keyword"`
    Status     string `json:"status"`
    Page       int    `json:"page"`
    PageSize   int    `json:"pageSize"`

    // === V4 新增 ===
    Category    string `json:"category,omitempty"`    // 按分类筛选
    SubCategory string `json:"subCategory,omitempty"` // 按子分类筛选
}
```

---

## 2. Agent Domain Model 新增

### 2.1 V4 规则常量与枚举

**新建文件**: `agents/rostering/domain/model/rule_v4.go`

```go
package model

// ============================================================
// V4 规则分类常量
// ============================================================

// RuleCategory 规则分类
type RuleCategory string

const (
    RuleCategoryConstraint  RuleCategory = "constraint"  // 约束型：必须遵守
    RuleCategoryPreference  RuleCategory = "preference"  // 偏好型：尽量满足
    RuleCategoryDependency  RuleCategory = "dependency"  // 依赖型：定义执行顺序
)

// RuleSubCategory 规则子分类
type RuleSubCategory string

const (
    // 约束型子分类
    RuleSubCategoryForbid RuleSubCategory = "forbid"  // 禁止型
    RuleSubCategoryLimit  RuleSubCategory = "limit"   // 限制型
    RuleSubCategoryMust   RuleSubCategory = "must"    // 必须型
    
    // 偏好型子分类
    RuleSubCategoryPrefer     RuleSubCategory = "prefer"     // 优先型
    RuleSubCategorySuggest    RuleSubCategory = "suggest"    // 建议型
    RuleSubCategoryCombinable RuleSubCategory = "combinable" // 可合并型
    
    // 依赖型子分类
    RuleSubCategorySource   RuleSubCategory = "source"   // 来源依赖
    RuleSubCategoryResource RuleSubCategory = "resource" // 资源预留
    RuleSubCategoryOrder    RuleSubCategory = "order"    // 顺序依赖
)

// AssociationRole 关联角色
type AssociationRole string

const (
    AssociationRoleTarget    AssociationRole = "target"    // 约束目标
    AssociationRoleSource    AssociationRole = "source"    // 数据来源
    AssociationRoleReference AssociationRole = "reference" // 引用对象
)

// IsValidCategory 校验分类是否有效
func IsValidCategory(cat string) bool {
    switch RuleCategory(cat) {
    case RuleCategoryConstraint, RuleCategoryPreference, RuleCategoryDependency:
        return true
    }
    return false
}

// IsValidSubCategory 校验子分类是否有效
func IsValidSubCategory(cat, subCat string) bool {
    switch RuleCategory(cat) {
    case RuleCategoryConstraint:
        switch RuleSubCategory(subCat) {
        case RuleSubCategoryForbid, RuleSubCategoryLimit, RuleSubCategoryMust:
            return true
        }
    case RuleCategoryPreference:
        switch RuleSubCategory(subCat) {
        case RuleSubCategoryPrefer, RuleSubCategorySuggest, RuleSubCategoryCombinable:
            return true
        }
    case RuleCategoryDependency:
        switch RuleSubCategory(subCat) {
        case RuleSubCategorySource, RuleSubCategoryResource, RuleSubCategoryOrder:
            return true
        }
    }
    return false
}

// IsValidAssociationRole 校验关联角色是否有效
func IsValidAssociationRole(role string) bool {
    switch AssociationRole(role) {
    case AssociationRoleTarget, AssociationRoleSource, AssociationRoleReference:
        return true
    }
    return false
}

// RuleTypeToDefaultCategory 根据规则类型推断默认分类（用于V3数据迁移）
func RuleTypeToDefaultCategory(ruleType string) (RuleCategory, RuleSubCategory) {
    switch ruleType {
    case "exclusive", "forbidden_day":
        return RuleCategoryConstraint, RuleSubCategoryForbid
    case "maxCount", "consecutiveMax", "minRestDays":
        return RuleCategoryConstraint, RuleSubCategoryLimit
    case "required_together", "periodic":
        return RuleCategoryConstraint, RuleSubCategoryMust
    case "preferred":
        return RuleCategoryPreference, RuleSubCategoryPrefer
    case "combinable":
        return RuleCategoryPreference, RuleSubCategoryCombinable
    default:
        return RuleCategoryConstraint, RuleSubCategoryLimit // 默认当约束处理
    }
}
```

### 2.2 规则依赖关系模型

**新建文件**: `agents/rostering/domain/model/rule_dependency.go`

```go
package model

import "time"

// RuleDependency 规则依赖关系
// 表示 DependentRuleID 的执行依赖于 DependsOnRuleID 的结果
type RuleDependency struct {
    ID               string    `json:"id"`
    OrgID            string    `json:"orgId"`
    DependentRuleID  string    `json:"dependentRuleId"`  // 被依赖的规则（需要先执行）
    DependsOnRuleID  string    `json:"dependsOnRuleId"`  // 依赖者（后执行）
    DependencyType   string    `json:"dependencyType"`   // time / source / resource / order
    Description      string    `json:"description"`
    CreatedAt        time.Time `json:"createdAt"`
}

// DependencyType 依赖类型常量
const (
    DependencyTypeTime     = "time"     // 时间依赖（前一日/前一周）
    DependencyTypeSource   = "source"   // 人员来源依赖
    DependencyTypeResource = "resource" // 资源预留依赖
    DependencyTypeOrder    = "order"    // 顺序依赖
)
```

### 2.3 规则冲突关系模型

**新建文件**: `agents/rostering/domain/model/rule_conflict.go`

```go
package model

import "time"

// RuleConflict 规则冲突关系
// 表示两条规则之间存在潜在冲突
type RuleConflict struct {
    ID                 string    `json:"id"`
    OrgID              string    `json:"orgId"`
    RuleID1            string    `json:"ruleId1"`
    RuleID2            string    `json:"ruleId2"`
    ConflictType       string    `json:"conflictType"`       // exclusive / resource / time / frequency
    Description        string    `json:"description"`
    ResolutionPriority int       `json:"resolutionPriority"` // 冲突时，优先保留的规则（值小优先）
    CreatedAt          time.Time `json:"createdAt"`
}

// ConflictType 冲突类型常量
const (
    ConflictTypeExclusive = "exclusive" // 互斥冲突（两条规则不能同时满足）
    ConflictTypeResource  = "resource"  // 资源竞争（争抢同一批人员）
    ConflictTypeTime      = "time"      // 时间冲突（约束的时间段重叠）
    ConflictTypeFrequency = "frequency" // 频次矛盾（一个要求多排，一个要求少排）
)
```

### 2.4 班次依赖关系模型

**新建文件**: `agents/rostering/domain/model/shift_dependency.go`

```go
package model

import "time"

// ShiftDependency 班次依赖关系
// 表示排班时 DependentShiftID 必须在 DependsOnShiftID 之后排
type ShiftDependency struct {
    ID                 string    `json:"id"`
    OrgID              string    `json:"orgId"`
    DependentShiftID   string    `json:"dependentShiftId"`   // 被依赖的班次（先排）
    DependsOnShiftID   string    `json:"dependsOnShiftId"`   // 依赖者班次（后排）
    DependencyType     string    `json:"dependencyType"`     // time / source / resource
    RuleID             string    `json:"ruleId,omitempty"`   // 产生此依赖的规则ID
    Description        string    `json:"description"`
    CreatedAt          time.Time `json:"createdAt"`
}
```

---

## 3. 仓储接口定义

### 3.1 规则依赖仓储

**新建文件**: `agents/rostering/domain/repository/rule_dependency_repository.go`

```go
package repository

import (
    "context"
    d_model "jusha/agent/rostering/domain/model"
)

// IRuleDependencyRepository 规则依赖关系仓储接口
type IRuleDependencyRepository interface {
    // Create 创建规则依赖关系
    Create(ctx context.Context, dep *d_model.RuleDependency) error
    
    // BatchCreate 批量创建
    BatchCreate(ctx context.Context, deps []*d_model.RuleDependency) error
    
    // Delete 删除
    Delete(ctx context.Context, id string) error
    
    // DeleteByRuleID 删除指定规则的所有依赖关系
    DeleteByRuleID(ctx context.Context, ruleID string) error
    
    // GetByOrgID 获取组织下所有依赖关系
    GetByOrgID(ctx context.Context, orgID string) ([]*d_model.RuleDependency, error)
    
    // GetByRuleID 获取指定规则的依赖关系
    GetByRuleID(ctx context.Context, ruleID string) ([]*d_model.RuleDependency, error)
}
```

### 3.2 规则冲突仓储

**新建文件**: `agents/rostering/domain/repository/rule_conflict_repository.go`

```go
package repository

import (
    "context"
    d_model "jusha/agent/rostering/domain/model"
)

// IRuleConflictRepository 规则冲突关系仓储接口
type IRuleConflictRepository interface {
    // Create 创建规则冲突关系
    Create(ctx context.Context, conflict *d_model.RuleConflict) error
    
    // BatchCreate 批量创建
    BatchCreate(ctx context.Context, conflicts []*d_model.RuleConflict) error
    
    // Delete 删除
    Delete(ctx context.Context, id string) error
    
    // DeleteByRuleID 删除指定规则的所有冲突关系
    DeleteByRuleID(ctx context.Context, ruleID string) error
    
    // GetByOrgID 获取组织下所有冲突关系
    GetByOrgID(ctx context.Context, orgID string) ([]*d_model.RuleConflict, error)
    
    // GetByRuleID 获取指定规则的冲突关系
    GetByRuleID(ctx context.Context, ruleID string) ([]*d_model.RuleConflict, error)
}
```

### 3.3 班次依赖仓储

**新建文件**: `agents/rostering/domain/repository/shift_dependency_repository.go`

```go
package repository

import (
    "context"
    d_model "jusha/agent/rostering/domain/model"
)

// IShiftDependencyRepository 班次依赖关系仓储接口
type IShiftDependencyRepository interface {
    // Create 创建班次依赖关系
    Create(ctx context.Context, dep *d_model.ShiftDependency) error
    
    // BatchCreate 批量创建
    BatchCreate(ctx context.Context, deps []*d_model.ShiftDependency) error
    
    // Delete 删除
    Delete(ctx context.Context, id string) error
    
    // DeleteByShiftID 删除指定班次的所有依赖关系
    DeleteByShiftID(ctx context.Context, shiftID string) error
    
    // GetByOrgID 获取组织下所有班次依赖关系
    GetByOrgID(ctx context.Context, orgID string) ([]*d_model.ShiftDependency, error)
    
    // GetByShiftID 获取指定班次的依赖关系
    GetByShiftID(ctx context.Context, shiftID string) ([]*d_model.ShiftDependency, error)
}
```

---

## 4. IRosteringService 接口扩展

**文件**: `agents/rostering/domain/service/rostering.go`

在现有 `IRosteringService` 接口末尾追加：

```go
    // ============================================================
    // V4 新增：规则依赖/冲突/班次依赖查询
    // ============================================================
    
    // GetRuleDependencies 获取组织下所有规则依赖关系
    GetRuleDependencies(ctx context.Context, orgID string) ([]*d_model.RuleDependency, error)
    
    // GetRuleConflicts 获取组织下所有规则冲突关系
    GetRuleConflicts(ctx context.Context, orgID string) ([]*d_model.RuleConflict, error)
    
    // GetShiftDependencies 获取组织下所有班次依赖关系
    GetShiftDependencies(ctx context.Context, orgID string) ([]*d_model.ShiftDependency, error)
    
    // SaveRuleDependencies 保存规则依赖关系（全量覆盖）
    SaveRuleDependencies(ctx context.Context, orgID string, deps []*d_model.RuleDependency) error
    
    // SaveRuleConflicts 保存规则冲突关系（全量覆盖）
    SaveRuleConflicts(ctx context.Context, orgID string, conflicts []*d_model.RuleConflict) error
    
    // SaveShiftDependencies 保存班次依赖关系（全量覆盖）
    SaveShiftDependencies(ctx context.Context, orgID string, deps []*d_model.ShiftDependency) error
```

---

## 5. 数据库 DDL

### 5.1 现有表扩展

```sql
-- scheduling_rules 表扩展
ALTER TABLE scheduling_rules 
ADD COLUMN category VARCHAR(32) DEFAULT '' COMMENT 'V4规则分类: constraint/preference/dependency',
ADD COLUMN sub_category VARCHAR(32) DEFAULT '' COMMENT 'V4规则子分类',
ADD COLUMN original_rule_id VARCHAR(64) DEFAULT '' COMMENT 'V4原始规则ID（语义化解析源）',
ADD INDEX idx_category (category),
ADD INDEX idx_sub_category (sub_category);

-- rule_associations 表扩展
ALTER TABLE rule_associations 
ADD COLUMN role VARCHAR(32) DEFAULT 'target' COMMENT 'V4关联角色: target/source/reference';
```

### 5.2 新增表

```sql
-- 规则依赖关系表
CREATE TABLE IF NOT EXISTS rule_dependencies (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    dependent_rule_id VARCHAR(64) NOT NULL COMMENT '被依赖的规则ID（需要先执行）',
    depends_on_rule_id VARCHAR(64) NOT NULL COMMENT '依赖者规则ID（后执行）',
    dependency_type VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource/order',
    description TEXT COMMENT '依赖关系描述',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_dependent_rule (dependent_rule_id),
    INDEX idx_depends_on_rule (depends_on_rule_id),
    INDEX idx_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='V4规则依赖关系表';

-- 规则冲突关系表
CREATE TABLE IF NOT EXISTS rule_conflicts (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    rule_id_1 VARCHAR(64) NOT NULL,
    rule_id_2 VARCHAR(64) NOT NULL,
    conflict_type VARCHAR(32) NOT NULL COMMENT '冲突类型: exclusive/resource/time/frequency',
    description TEXT COMMENT '冲突描述',
    resolution_priority INT DEFAULT 0 COMMENT '解决优先级（值小优先）',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_rule_1 (rule_id_1),
    INDEX idx_rule_2 (rule_id_2),
    INDEX idx_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='V4规则冲突关系表';

-- 班次依赖关系表
CREATE TABLE IF NOT EXISTS shift_dependencies (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    dependent_shift_id VARCHAR(64) NOT NULL COMMENT '被依赖的班次ID（先排）',
    depends_on_shift_id VARCHAR(64) NOT NULL COMMENT '依赖者班次ID（后排）',
    dependency_type VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource',
    rule_id VARCHAR(64) DEFAULT '' COMMENT '产生此依赖关系的规则ID',
    description TEXT COMMENT '依赖关系描述',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_dependent_shift (dependent_shift_id),
    INDEX idx_depends_on_shift (depends_on_shift_id),
    INDEX idx_rule (rule_id),
    INDEX idx_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='V4班次依赖关系表';
```

---

## 6. V3 数据迁移策略

### 6.1 迁移脚本

```sql
-- 为现有规则补充 category / sub_category（基于 rule_type 推断）
UPDATE scheduling_rules SET 
    category = CASE rule_type
        WHEN 'exclusive' THEN 'constraint'
        WHEN 'forbidden_day' THEN 'constraint'
        WHEN 'maxCount' THEN 'constraint'
        WHEN 'consecutiveMax' THEN 'constraint'
        WHEN 'minRestDays' THEN 'constraint'
        WHEN 'required_together' THEN 'constraint'
        WHEN 'periodic' THEN 'constraint'
        WHEN 'preferred' THEN 'preference'
        WHEN 'combinable' THEN 'preference'
        ELSE 'constraint'
    END,
    sub_category = CASE rule_type
        WHEN 'exclusive' THEN 'forbid'
        WHEN 'forbidden_day' THEN 'forbid'
        WHEN 'maxCount' THEN 'limit'
        WHEN 'consecutiveMax' THEN 'limit'
        WHEN 'minRestDays' THEN 'limit'
        WHEN 'required_together' THEN 'must'
        WHEN 'periodic' THEN 'must'
        WHEN 'preferred' THEN 'prefer'
        WHEN 'combinable' THEN 'combinable'
        ELSE 'limit'
    END
WHERE category = '' OR category IS NULL;

-- 为现有关联补充 role（默认全部为 target）
UPDATE rule_associations SET role = 'target' WHERE role = '' OR role IS NULL;
```

### 6.2 迁移原则

1. **不破坏 V3 数据**：所有新字段有默认值
2. **V3 代码无需改动**：V3 读取数据时忽略新字段（`omitempty`）
3. **迁移可重复执行**：使用 `WHERE category = ''` 条件
4. **迁移时间**：在 Agent-1 完成后、Agent-2 开始前执行
