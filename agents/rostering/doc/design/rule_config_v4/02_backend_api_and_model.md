# 02. 后端 API 与数据模型改造方案

> **改造范围**: management-service、SDK、MCP Server  
> **原则**: V4 字段过渡期 optional，V3 兼容代码集中隔离、标记 `@deprecated`，全量迁移后一键清理

## 1. 数据模型改造

### 1.1 management-service 领域模型扩展

**文件**: `services/management-service/domain/model/scheduling_rule.go`

#### 1.1.1 SchedulingRule 新增字段

```go
type SchedulingRule struct {
    // === 现有字段（不修改）===
    ID          string     `json:"id"`
    OrgID       string     `json:"orgId"`
    Name        string     `json:"name"`
    Description string     `json:"description"`
    RuleType    RuleType   `json:"ruleType"`
    ApplyScope  ApplyScope `json:"applyScope"`
    TimeScope   TimeScope  `json:"timeScope"`
    RuleData    string     `json:"ruleData"`
    MaxCount       *int `json:"maxCount,omitempty"`
    ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
    IntervalDays   *int `json:"intervalDays,omitempty"`
    MinRestDays    *int `json:"minRestDays,omitempty"`
    Priority  int        `json:"priority"`
    IsActive  bool       `json:"isActive"`
    ValidFrom *time.Time `json:"validFrom,omitempty"`
    ValidTo   *time.Time `json:"validTo,omitempty"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
    DeletedAt *time.Time `json:"deletedAt,omitempty"`
    Associations []RuleAssociation `json:"associations,omitempty"`

    // === V4 新增字段 ===
    
    // Category 规则分类: constraint / preference / dependency
    // 空字符串表示 V3 规则（未分类）
    Category string `json:"category,omitempty"`
    
    // SubCategory 规则子分类
    // constraint: forbid / limit / must
    // preference: prefer / suggest / combinable
    // dependency: source / resource / order
    SubCategory string `json:"subCategory,omitempty"`
    
    // OriginalRuleID 原始规则ID
    // 如果是从一段自然语言描述中解析出的多条规则，共享同一个 OriginalRuleID
    OriginalRuleID string `json:"originalRuleId,omitempty"`
    
    // SourceType 规则来源类型
    // manual: 手动创建
    // llm_parsed: LLM 解析生成
    // migrated: V3 迁移 (@deprecated V3: 迁移完成后可考虑将 migrated 统一为 manual)
    SourceType string `json:"sourceType,omitempty"`
    
    // ParseConfidence LLM 解析置信度 (0.0-1.0)
    // 仅 SourceType=llm_parsed 时有值
    ParseConfidence *float64 `json:"parseConfidence,omitempty"`
    
    // Version 规则版本号（V3=空或"v3", V4="v4"）
    // @deprecated V3: 全量迁移完成后此字段固定为"v4"，可移除版本判断逻辑
    Version string `json:"version,omitempty"`
}
```

#### 1.1.2 RuleAssociation 新增字段

```go
type RuleAssociation struct {
    // === 现有字段（不修改）===
    ID              string          `json:"id"`
    RuleID          string          `json:"ruleId"`
    AssociationType AssociationType `json:"associationType"`
    AssociationID   string          `json:"associationId"`
    CreatedAt       time.Time       `json:"createdAt"`

    // === V4 新增字段 ===
    
    // Role 关联角色
    // target: 被约束的对象（默认）
    // source: 数据来源
    // reference: 引用对象
    Role string `json:"role,omitempty"` // 默认 "target"
}
```

#### 1.1.3 新增枚举常量

```go
// V4 规则分类
const (
    CategoryConstraint  = "constraint"
    CategoryPreference  = "preference"
    CategoryDependency  = "dependency"
)

// V4 规则子分类
const (
    SubCategoryForbid     = "forbid"
    SubCategoryLimit      = "limit"
    SubCategoryMust       = "must"
    SubCategoryPrefer     = "prefer"
    SubCategorySuggest    = "suggest"
    SubCategoryCombinable = "combinable"
    SubCategorySource     = "source"
    SubCategoryResource   = "resource"
    SubCategoryOrder      = "order"
)

// V4 关联角色
const (
    AssociationRoleTarget    = "target"
    AssociationRoleSource    = "source"
    AssociationRoleReference = "reference"
)

// 规则来源
const (
    SourceTypeManual    = "manual"
    SourceTypeLLMParsed = "llm_parsed"
    SourceTypeMigrated  = "migrated"
)
```

#### 1.1.4 SchedulingRuleFilter 扩展

```go
type SchedulingRuleFilter struct {
    // === 现有字段 ===
    OrgID      string
    RuleType   *RuleType
    ApplyScope *ApplyScope
    TimeScope  *TimeScope
    IsActive   *bool
    Keyword    string
    Page       int
    PageSize   int

    // === V4 新增 ===
    Category    string // 按分类筛选
    SubCategory string // 按子分类筛选
    SourceType  string // 按来源筛选
    Version     string // 按版本筛选 (v3/v4)
}
```

---

### 1.2 数据库 DDL 改造

#### 1.2.1 scheduling_rules 表新增列

```sql
-- V4 扩展字段
ALTER TABLE scheduling_rules 
    ADD COLUMN category VARCHAR(20) DEFAULT '' COMMENT '规则分类: constraint/preference/dependency',
    ADD COLUMN sub_category VARCHAR(20) DEFAULT '' COMMENT '规则子分类',
    ADD COLUMN original_rule_id VARCHAR(36) DEFAULT '' COMMENT '原始规则ID(LLM解析出多条时共享)',
    ADD COLUMN source_type VARCHAR(20) DEFAULT 'manual' COMMENT '来源类型: manual/llm_parsed/migrated',
    ADD COLUMN parse_confidence DECIMAL(3,2) DEFAULT NULL COMMENT 'LLM解析置信度 0.00-1.00',
    ADD COLUMN version VARCHAR(10) DEFAULT '' COMMENT '规则版本: 空或v3=V3, v4=V4';

-- 索引
CREATE INDEX idx_scheduling_rules_category ON scheduling_rules(category);
CREATE INDEX idx_scheduling_rules_version ON scheduling_rules(version);
CREATE INDEX idx_scheduling_rules_source_type ON scheduling_rules(source_type);
CREATE INDEX idx_scheduling_rules_original_rule_id ON scheduling_rules(original_rule_id);
```

#### 1.2.2 rule_associations 表新增列

```sql
-- V4 关联角色
ALTER TABLE rule_associations 
    ADD COLUMN role VARCHAR(20) DEFAULT 'target' COMMENT '关联角色: target/source/reference';
```

#### 1.2.3 新增 rule_dependencies 表

```sql
CREATE TABLE rule_dependencies (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    dependent_rule_id VARCHAR(36) NOT NULL COMMENT '被依赖的规则(先执行)',
    depends_on_rule_id VARCHAR(36) NOT NULL COMMENT '依赖者规则(后执行)',
    dependency_type VARCHAR(20) NOT NULL COMMENT 'time/source/resource/order',
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_rule_deps_org (org_id),
    INDEX idx_rule_deps_dependent (dependent_rule_id),
    INDEX idx_rule_deps_depends_on (depends_on_rule_id),
    UNIQUE KEY uk_rule_dependency (dependent_rule_id, depends_on_rule_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='规则依赖关系表';
```

#### 1.2.4 新增 rule_conflicts 表

```sql
CREATE TABLE rule_conflicts (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    rule_id_1 VARCHAR(36) NOT NULL,
    rule_id_2 VARCHAR(36) NOT NULL,
    conflict_type VARCHAR(20) NOT NULL COMMENT 'exclusive/resource/time/frequency',
    description TEXT,
    resolution_priority INT DEFAULT 0 COMMENT '冲突时优先保留的规则(值小优先)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_rule_conflicts_org (org_id),
    INDEX idx_rule_conflicts_rule1 (rule_id_1),
    INDEX idx_rule_conflicts_rule2 (rule_id_2),
    UNIQUE KEY uk_rule_conflict (rule_id_1, rule_id_2)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='规则冲突关系表';
```

---

## 2. 服务接口扩展

### 2.1 ISchedulingRuleService 新增方法

**文件**: `services/management-service/domain/service/scheduling_rule.go`

```go
type ISchedulingRuleService interface {
    // === 现有方法（不修改）===
    CreateRule(ctx context.Context, rule *model.SchedulingRule) error
    UpdateRule(ctx context.Context, rule *model.SchedulingRule) error
    DeleteRule(ctx context.Context, orgID, ruleID string) error
    GetRule(ctx context.Context, orgID, ruleID string) (*model.SchedulingRule, error)
    ListRules(ctx context.Context, filter *model.SchedulingRuleFilter) (*model.SchedulingRuleListResult, error)
    GetActiveRules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)
    ToggleRuleStatus(ctx context.Context, orgID, ruleID string, isActive bool) error
    AddRuleAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error
    RemoveRuleAssociations(ctx context.Context, orgID, ruleID string, associationIDs []string) error
    RemoveRuleAssociationByTarget(ctx context.Context, orgID, ruleID, targetType, targetID string) error
    GetRuleAssociations(ctx context.Context, orgID, ruleID string) ([]model.RuleAssociation, error)
    GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.SchedulingRule, error)
    GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.SchedulingRule, error)
    GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.SchedulingRule, error)
    GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.SchedulingRule, error)
    GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.SchedulingRule, error)
    GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.SchedulingRule, error)
    UpdateRuleAssociations(ctx context.Context, orgID, ruleID string, associations []model.RuleAssociation) error
    ValidateRule(ctx context.Context, rule *model.SchedulingRule) error
    CheckRuleConflicts(ctx context.Context, orgID string, rule *model.SchedulingRule) ([]string, error)

    // === V4 新增方法 ===

    // ListRulesByCategory 按分类获取规则
    ListRulesByCategory(ctx context.Context, orgID, category string) ([]*model.SchedulingRule, error)

    // GetRuleDependencies 获取规则依赖关系
    GetRuleDependencies(ctx context.Context, orgID string) ([]*model.RuleDependency, error)

    // AddRuleDependency 添加规则依赖
    AddRuleDependency(ctx context.Context, dep *model.RuleDependency) error

    // RemoveRuleDependency 删除规则依赖
    RemoveRuleDependency(ctx context.Context, id string) error

    // GetRuleConflicts 获取规则冲突关系
    GetRuleConflicts(ctx context.Context, orgID string) ([]*model.RuleConflict, error)

    // AddRuleConflict 添加规则冲突
    AddRuleConflict(ctx context.Context, conflict *model.RuleConflict) error

    // RemoveRuleConflict 删除规则冲突
    RemoveRuleConflict(ctx context.Context, id string) error

    // BatchUpdateVersion 批量更新规则版本
    BatchUpdateVersion(ctx context.Context, orgID string, ruleIDs []string, version string) error

    // GetV3Rules 获取所有 V3 规则（待迁移）
    GetV3Rules(ctx context.Context, orgID string) ([]*model.SchedulingRule, error)

    // GetRuleStatistics 获取规则统计信息
    GetRuleStatistics(ctx context.Context, orgID string) (*model.RuleStatistics, error)
}
```

### 2.2 新增模型类型

```go
// RuleDependency 规则依赖关系
type RuleDependency struct {
    ID               string    `json:"id"`
    OrgID            string    `json:"orgId"`
    DependentRuleID  string    `json:"dependentRuleId"`
    DependsOnRuleID  string    `json:"dependsOnRuleId"`
    DependencyType   string    `json:"dependencyType"`
    Description      string    `json:"description"`
    CreatedAt        time.Time `json:"createdAt"`
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
    ID                 string    `json:"id"`
    OrgID              string    `json:"orgId"`
    RuleID1            string    `json:"ruleId1"`
    RuleID2            string    `json:"ruleId2"`
    ConflictType       string    `json:"conflictType"`
    Description        string    `json:"description"`
    ResolutionPriority int       `json:"resolutionPriority"`
    CreatedAt          time.Time `json:"createdAt"`
}

// RuleStatistics 规则统计
type RuleStatistics struct {
    Total          int            `json:"total"`
    ActiveCount    int            `json:"activeCount"`
    InactiveCount  int            `json:"inactiveCount"`
    ByCategory     map[string]int `json:"byCategory"`     // category -> count
    ByVersion      map[string]int `json:"byVersion"`      // version -> count
    BySourceType   map[string]int `json:"bySourceType"`   // sourceType -> count
    V3Count        int            `json:"v3Count"`         // 待迁移的V3规则数
    DependencyCount int           `json:"dependencyCount"` // 依赖关系数
    ConflictCount   int           `json:"conflictCount"`   // 冲突关系数
}
```

---

## 3. HTTP API 改造

### 3.1 现有 API 扩展（向后兼容）

#### 3.1.1 GET /v1/scheduling-rules 列表接口

**新增查询参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| `category` | string | 按分类筛选: constraint/preference/dependency |
| `subCategory` | string | 按子分类筛选 |
| `sourceType` | string | 按来源筛选: manual/llm_parsed/migrated |
| `version` | string | 按版本筛选: v3/v4 |

**响应新增字段**:

```json
{
  "items": [
    {
      "id": "rule-001",
      "name": "早班周排班限制",
      "ruleType": "maxCount",
      "applyScope": "specific",
      "timeScope": "same_week",
      "// ... 现有字段 ...": "",
      
      "category": "constraint",
      "subCategory": "limit",
      "originalRuleId": "",
      "sourceType": "llm_parsed",
      "parseConfidence": 0.95,
      "version": "v4"
    }
  ],
  "total": 42
}
```

#### 3.1.2 POST /v1/scheduling-rules 创建接口

**请求体新增可选字段**:

```json
{
  "orgId": "org-001",
  "name": "早班周排班限制",
  "ruleType": "maxCount",
  "applyScope": "specific",
  "timeScope": "same_week",
  "maxCount": 3,
  "priority": 7,
  "ruleData": "早班每人每周最多排3次",
  "description": "限制早班每周排班次数",
  
  "category": "constraint",
  "subCategory": "limit",
  "originalRuleId": "parse-batch-001",
  "sourceType": "llm_parsed",
  "parseConfidence": 0.95,
  "version": "v4"
}
```

> **过渡兼容**: V4 新增字段过渡期可选，不传则视为 V3 规则。全量迁移完成后，`category`/`subCategory`/`version` 将变为**必填**，届时移除空值兜底逻辑。

#### 3.1.3 POST /v1/scheduling-rules/associations 关联接口

**请求体新增字段**:

```json
{
  "orgId": "org-001",
  "ruleId": "rule-001",
  "targetType": "shift",
  "targetId": "shift-001",
  "role": "target"
}
```

> **过渡兼容**: `role` 过渡期不传时默认为 `"target"`。全量迁移完成后 `role` 将变为**必填**字段。

### 3.2 新增 API

#### 3.2.1 规则解析

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/rules/parse` | 单条规则自然语言解析 |
| POST | `/v1/rules/batch-parse` | 批量规则解析 |

详见 [01_rule_parse_service.md](01_rule_parse_service.md)

#### 3.2.2 规则迁移

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules/migration/preview` | 预览 V3→V4 迁移结果 |
| POST | `/v1/rules/migration/execute` | 执行迁移 |
| GET | `/v1/rules/migration/status` | 查询迁移状态 |

详见 [04_migration_plan.md](04_migration_plan.md)

#### 3.2.3 规则依赖管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules/dependencies` | 获取组织的规则依赖关系 |
| POST | `/v1/rules/dependencies` | 创建规则依赖 |
| DELETE | `/v1/rules/dependencies/:id` | 删除规则依赖 |

**请求/响应示例**:

```json
// POST /v1/rules/dependencies
{
  "orgId": "org-001",
  "dependentRuleId": "rule-A",
  "dependsOnRuleId": "rule-B",
  "dependencyType": "source",
  "description": "下夜班人员必须来自上半夜班"
}

// GET /v1/rules/dependencies?orgId=org-001
{
  "dependencies": [
    {
      "id": "dep-001",
      "dependentRuleId": "rule-A",
      "dependsOnRuleId": "rule-B",
      "dependencyType": "source",
      "description": "下夜班人员必须来自上半夜班"
    }
  ]
}
```

#### 3.2.4 规则冲突管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules/conflicts` | 获取组织的规则冲突关系 |
| POST | `/v1/rules/conflicts` | 创建规则冲突 |
| DELETE | `/v1/rules/conflicts/:id` | 删除规则冲突 |

#### 3.2.5 规则统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules/statistics` | 获取规则统计概览 |

**响应**:

```json
{
  "total": 42,
  "activeCount": 35,
  "inactiveCount": 7,
  "byCategory": {
    "constraint": 25,
    "preference": 12,
    "dependency": 5,
    "": 0
  },
  "byVersion": {
    "v4": 30,
    "": 12
  },
  "bySourceType": {
    "manual": 10,
    "llm_parsed": 20,
    "migrated": 12
  },
  "v3Count": 0,
  "dependencyCount": 8,
  "conflictCount": 3
}
```

---

## 4. SDK Model 改造

### 4.1 Rule 结构体扩展

**文件**: `sdk/rostering/model/rule.go`

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
    Category        string   `json:"category,omitempty"`
    SubCategory     string   `json:"subCategory,omitempty"`
    OriginalRuleID  string   `json:"originalRuleId,omitempty"`
    SourceType      string   `json:"sourceType,omitempty"`
    ParseConfidence *float64 `json:"parseConfidence,omitempty"`
    Version         string   `json:"version,omitempty"`
}

type RuleAssociation struct {
    // === 现有字段（不修改）===
    ID              string `json:"id,omitempty"`
    RuleID          string `json:"ruleId"`
    AssociationType string `json:"associationType"`
    AssociationID   string `json:"associationId"`
    TargetType      string `json:"targetType,omitempty"`
    TargetID        string `json:"targetId,omitempty"`

    // === V4 新增字段 ===
    Role string `json:"role,omitempty"` // target/source/reference, 默认 target
}

// CreateRuleRequest 扩展
type CreateRuleRequest struct {
    // === 现有字段 ===
    // ... (不列出)

    // === V4 新增 ===
    Category        string   `json:"category,omitempty"`
    SubCategory     string   `json:"subCategory,omitempty"`
    OriginalRuleID  string   `json:"originalRuleId,omitempty"`
    SourceType      string   `json:"sourceType,omitempty"`
    ParseConfidence *float64 `json:"parseConfidence,omitempty"`
    Version         string   `json:"version,omitempty"`
}

// UpdateRuleRequest 扩展
type UpdateRuleRequest struct {
    // === 现有字段 ===
    // ... (不列出)

    // === V4 新增 ===
    Category    string `json:"category,omitempty"`
    SubCategory string `json:"subCategory,omitempty"`
    Version     string `json:"version,omitempty"`
}

// ListRulesRequest 扩展
type ListRulesRequest struct {
    // === 现有字段 ===
    // ... (不列出)

    // === V4 新增 ===
    Category    string `json:"category,omitempty"`
    SubCategory string `json:"subCategory,omitempty"`
    SourceType  string `json:"sourceType,omitempty"`
    Version     string `json:"version,omitempty"`
}
```

---

## 5. SDK Client 扩展

### 5.1 新增方法

**文件**: `sdk/rostering/client/rule.go`

```go
// ParseRule 调用规则解析服务
func (c *RuleClient) ParseRule(ctx context.Context, orgID, ruleText string) (*ParseRuleResponse, error) {
    return c.callTool(ctx, "parse_rule", map[string]interface{}{
        "orgId":    orgID,
        "ruleText": ruleText,
    })
}

// GetRuleDependencies 获取规则依赖关系
func (c *RuleClient) GetRuleDependencies(ctx context.Context, orgID string) ([]*model.RuleDependency, error) {
    return c.callTool(ctx, "get_rule_dependencies", map[string]interface{}{
        "orgId": orgID,
    })
}

// GetRuleConflicts 获取规则冲突关系
func (c *RuleClient) GetRuleConflicts(ctx context.Context, orgID string) ([]*model.RuleConflict, error) {
    return c.callTool(ctx, "get_rule_conflicts", map[string]interface{}{
        "orgId": orgID,
    })
}

// GetRuleStatistics 获取规则统计
func (c *RuleClient) GetRuleStatistics(ctx context.Context, orgID string) (*model.RuleStatistics, error) {
    return c.callTool(ctx, "get_rule_statistics", map[string]interface{}{
        "orgId": orgID,
    })
}
```

---

## 6. MCP Server 扩展

### 6.1 新增 MCP Tools

**目录**: `mcp-servers/rostering/tool/rule/`

| 工具名 | 文件 | 说明 |
|--------|------|------|
| `parse_rule` | `parse_rule.go` | 自然语言规则解析 |
| `batch_parse_rules` | `batch_parse_rules.go` | 批量规则解析 |
| `get_rule_dependencies` | `get_dependencies.go` | 获取规则依赖 |
| `add_rule_dependency` | `add_dependency.go` | 添加规则依赖 |
| `remove_rule_dependency` | `remove_dependency.go` | 删除规则依赖 |
| `get_rule_conflicts` | `get_conflicts.go` | 获取规则冲突 |
| `add_rule_conflict` | `add_conflict.go` | 添加规则冲突 |
| `remove_rule_conflict` | `remove_conflict.go` | 删除规则冲突 |
| `get_rule_statistics` | `get_statistics.go` | 获取规则统计 |
| `preview_migration` | `preview_migration.go` | 预览规则迁移 |
| `execute_migration` | `execute_migration.go` | 执行规则迁移 |

### 6.2 现有工具改造

`create_rule` 和 `update_rule` 工具的 Input Schema 新增 V4 可选字段：

```go
// create_rule 新增 Input 字段
{
    "category":        {"type": "string", "enum": ["constraint", "preference", "dependency"]},
    "subCategory":     {"type": "string"},
    "originalRuleId":  {"type": "string"},
    "sourceType":      {"type": "string", "enum": ["manual", "llm_parsed", "migrated"]},
    "parseConfidence": {"type": "number"},
    "version":         {"type": "string", "enum": ["v3", "v4"]}
}
```

---

## 7. Handler 改造清单

### 7.1 scheduling_rule_handler.go 改造点

| 方法 | 改造类型 | 说明 |
|------|---------|------|
| `ListRules` | **修改** | 新增 category/subCategory/sourceType/version 查询参数 |
| `CreateRule` | **修改** | 接收 V4 新字段，存入数据库 |
| `UpdateRule` | **修改** | 支持更新 V4 字段 |
| `GetRule` | **修改** | 返回 V4 字段 |
| `ParseRule` | **新增** | 调用 RuleParserService |
| `BatchParseRules` | **新增** | 批量解析 |
| `GetDependencies` | **新增** | 规则依赖 CRUD |
| `AddDependency` | **新增** | |
| `RemoveDependency` | **新增** | |
| `GetConflicts` | **新增** | 规则冲突 CRUD |
| `AddConflict` | **新增** | |
| `RemoveConflict` | **新增** | |
| `GetStatistics` | **新增** | 统计 |
| `PreviewMigration` | **新增** | 迁移预览 |
| `ExecuteMigration` | **新增** | 执行迁移 |
| `GetMigrationStatus` | **新增** | 迁移状态 |
| `AddRuleAssociations` | **修改** | 接收 role 字段 |

### 7.2 改造量评估

| 文件 | 现有行数 | 预估新增 | 说明 |
|------|---------|---------|------|
| `scheduling_rule_handler.go` | 778 | +400 | 新增8个Handler方法 |
| `scheduling_rule_service_impl.go` | ~500 | +250 | 新增服务方法实现 |
| `scheduling_rule_repository.go` | ~300 | +200 | 新增仓储方法 + V4字段查询 |
| `rule_parser/` (新建) | 0 | +500 | 解析服务 + 验证器 + 匹配器 |
| 迁移相关 | 0 | +300 | 迁移服务 |
| **合计** | ~1578 | **+1650** | |

---

## 8. V3 兼容隔离与清理设计

> ⚠️ **V3 兼容是过渡态，不是终态**。所有兼容代码必须隔离到独立文件，不得混入 V4 核心逻辑。

### 8.1 后端兼容代码隔离

所有 V3 → V4 枚举转换函数集中到**一个独立文件**中：

**文件**: `services/management-service/internal/port/http/v3_compat.go`

```go
package http

// ============================================================
// @deprecated V3 兼容层
// 全量迁移完成后删除此文件，并移除 Handler 中所有 normalizeXxx 调用
// 清理条件: scheduling_rules 表中 version='' 的记录数为 0
// ============================================================

import (
    "jusha/gantt/service/management/domain/model"
    "jusha/gantt/pkg/logging"
)

var v3CompatLogger = logging.GetLogger("v3_compat")

// normalizeRuleType 兼容前端旧枚举值
// @deprecated V3: 前端切换到后端枚举值后移除
func normalizeRuleType(ruleType string) model.RuleType {
    switch ruleType {
    case "max_shifts", "consecutive_shifts", "rest_days":
        v3CompatLogger.Warn("V3 deprecated ruleType received", "value", ruleType)
        return model.RuleTypeMaxCount
    case "forbidden_pattern":
        v3CompatLogger.Warn("V3 deprecated ruleType received", "value", ruleType)
        return model.RuleTypeExclusive
    case "preferred_pattern":
        v3CompatLogger.Warn("V3 deprecated ruleType received", "value", ruleType)
        return model.RuleTypePreferred
    default:
        return model.RuleType(ruleType)
    }
}

// normalizeApplyScope 兼容前端旧枚举值
// @deprecated V3: 前端切换到后端枚举值后移除
func normalizeApplyScope(scope string) model.ApplyScope {
    switch scope {
    case "shift", "group", "employee":
        v3CompatLogger.Warn("V3 deprecated applyScope received", "value", scope)
        return model.ApplyScopeSpecific
    default:
        return model.ApplyScope(scope)
    }
}

// normalizeTimeScope 兼容前端旧枚举值
// @deprecated V3: 前端切换到后端枚举值后移除
func normalizeTimeScope(scope string) model.TimeScope {
    switch scope {
    case "daily":
        v3CompatLogger.Warn("V3 deprecated timeScope received", "value", scope)
        return model.TimeScopeSameDay
    case "weekly":
        v3CompatLogger.Warn("V3 deprecated timeScope received", "value", scope)
        return model.TimeScopeSameWeek
    case "monthly":
        v3CompatLogger.Warn("V3 deprecated timeScope received", "value", scope)
        return model.TimeScopeSameMonth
    default:
        return model.TimeScope(scope)
    }
}

// FillV3Defaults 为 V3 规则填充 V4 默认值
// @deprecated V3: 全量迁移后移除
func FillV3Defaults(rule *model.SchedulingRule) {
    if rule.Category == "" {
        // 仅在 V3 过渡期自动推断，迁移完成后不再需要
        cat, subCat := model.RuleTypeToDefaultCategory(string(rule.RuleType))
        rule.Category = string(cat)
        rule.SubCategory = string(subCat)
    }
    // role 默认值
    for i := range rule.Associations {
        if rule.Associations[i].Role == "" {
            rule.Associations[i].Role = "target"
        }
    }
}
```

### 8.2 Handler 中的调用方式

Handler 通过调用 `v3_compat.go` 中的函数来处理兼容，**不在 Handler 主逻辑中写任何兼容 switch/case**：

```go
// Handler 中的调用（清理时只需搜索 normalizeXxx 并删除调用即可）
func (h *SchedulingRuleHandler) CreateRule(c *gin.Context) {
    // ...
    rule.RuleType = normalizeRuleType(string(rule.RuleType))   // @deprecated V3
    rule.ApplyScope = normalizeApplyScope(string(rule.ApplyScope)) // @deprecated V3
    rule.TimeScope = normalizeTimeScope(string(rule.TimeScope))   // @deprecated V3
    FillV3Defaults(rule) // @deprecated V3
    // ...
}
```

### 8.3 清理步骤（全量迁移完成后执行）

```
Step 1: 确认前提条件
  $ SELECT COUNT(*) FROM scheduling_rules WHERE version != 'v4' OR version IS NULL;
  → 必须为 0

Step 2: 删除后端兼容文件
  $ rm services/management-service/internal/port/http/v3_compat.go

Step 3: 移除 Handler 中的兼容调用
  $ grep -n 'normalizeRuleType\|normalizeApplyScope\|normalizeTimeScope\|FillV3Defaults\|@deprecated V3' \
      services/management-service/internal/port/http/scheduling_rule_handler.go
  → 逐个删除标记行

Step 4: 将 V4 字段从 optional 改为 required
  - category: omitempty → required
  - subCategory: omitempty → required
  - version: 移除字段（全部为 v4，无需区分）

Step 5: 删除迁移相关代码
  $ rm -rf services/management-service/internal/migration/
  $ rm mcp-servers/rostering/tool/rule/preview_migration.go
  $ rm mcp-servers/rostering/tool/rule/execute_migration.go

Step 6: 数据库清理
  ALTER TABLE scheduling_rules DROP COLUMN version;
  ALTER TABLE scheduling_rules MODIFY COLUMN category VARCHAR(20) NOT NULL;
  ALTER TABLE scheduling_rules MODIFY COLUMN sub_category VARCHAR(20) NOT NULL;
  ALTER TABLE rule_associations MODIFY COLUMN role VARCHAR(20) NOT NULL DEFAULT 'target';

Step 7: 编译验证
  $ go build ./...
```
