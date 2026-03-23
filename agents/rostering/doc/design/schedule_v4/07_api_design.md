# 07. API 接口设计

> **开发负责人**: Agent-5  
> **依赖**: Agent-1 (数据模型), Agent-4 (规则解析服务)  
> **被依赖**: Agent-6 (前端)  
> **包路径**: `services/management-service/internal/handler/` & `mcp-servers/rostering/`

## 1. API 总览

### 新增 API (V4)

| 方法 | 路径 | 说明 | 服务位置 |
|------|------|------|---------|
| POST | `/api/v1/rules/parse` | 解析自然语言规则 | management-service |
| POST | `/api/v1/rules/parse/batch` | 批量解析规则 | management-service |
| POST | `/api/v1/rules/validate` | 三层验证规则 | management-service |
| GET | `/api/v1/rules/{id}/dependencies` | 获取规则依赖 | management-service |
| POST | `/api/v1/rules/dependencies` | 创建规则依赖 | management-service |
| DELETE | `/api/v1/rules/dependencies/{id}` | 删除规则依赖 | management-service |
| GET | `/api/v1/rules/{id}/conflicts` | 获取规则冲突 | management-service |
| POST | `/api/v1/rules/conflicts` | 创建规则冲突 | management-service |
| DELETE | `/api/v1/rules/conflicts/{id}` | 删除规则冲突 | management-service |
| GET | `/api/v1/shifts/dependencies` | 获取班次依赖 | management-service |
| POST | `/api/v1/shifts/dependencies` | 创建班次依赖 | management-service |
| DELETE | `/api/v1/shifts/dependencies/{id}` | 删除班次依赖 | management-service |
| GET | `/api/v1/rules/organization` | 获取规则组织视图 | management-service |

### 已有 API (保持不变)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/rules` | 获取规则列表 |
| POST | `/api/v1/rules` | 创建规则 |
| PUT | `/api/v1/rules/{id}` | 更新规则 |
| DELETE | `/api/v1/rules/{id}` | 删除规则 |

## 2. 规则解析 API

### 2.1 单条规则解析

**POST** `/api/v1/rules/parse`

Request:
```json
{
  "orgId": "org_001",
  "ruleText": "早班每人每周最多排3次",
  "shiftNames": ["早班", "中班", "夜班"],
  "groupNames": ["护理组", "急诊组"]
}
```

Response (200):
```json
{
  "code": 0,
  "data": {
    "success": true,
    "ruleName": "早班周频次限制",
    "ruleType": "maxCount",
    "category": "constraint",
    "subCategory": "frequency",
    "description": "早班每人每周最多排3次",
    "maxCount": 3,
    "timeScope": "same_week",
    "applyScope": "shift",
    "associationTargets": [
      {"type": "shift", "name": "早班", "id": "shift_001", "role": "target"}
    ],
    "suggestedPriority": 8,
    "backTranslation": "早班每位员工每周最多排班3次",
    "validation": {
      "structuralValid": true,
      "backTransValid": true,
      "simulationValid": true
    }
  }
}
```

### 2.2 批量规则解析

**POST** `/api/v1/rules/parse/batch`

Request:
```json
{
  "orgId": "org_001",
  "ruleTexts": [
    "早班每人每周最多排3次",
    "夜班后至少休息1天",
    "连续排班不超过5天"
  ],
  "shiftNames": ["早班", "中班", "夜班"],
  "groupNames": ["护理组"]
}
```

Response (200):
```json
{
  "code": 0,
  "data": {
    "totalCount": 3,
    "successCount": 3,
    "failedCount": 0,
    "results": [
      {"success": true, "ruleName": "早班周频次限制", "ruleType": "maxCount", "...": "..."},
      {"success": true, "ruleName": "夜班后休息", "ruleType": "minRestDays", "...": "..."},
      {"success": true, "ruleName": "连续排班限制", "ruleType": "consecutiveMax", "...": "..."}
    ]
  }
}
```

## 3. 规则依赖 API

### 3.1 获取规则依赖

**GET** `/api/v1/rules/{ruleId}/dependencies`

Response:
```json
{
  "code": 0,
  "data": {
    "dependsOn": [
      {
        "id": "dep_001",
        "ruleId": "rule_001",
        "dependsOnRuleId": "rule_002",
        "dependencyType": "prerequisite",
        "description": "需先满足 rule_002",
        "dependsOnRuleName": "夜班后休息规则"
      }
    ],
    "dependedBy": [
      {
        "id": "dep_002",
        "ruleId": "rule_003",
        "dependsOnRuleId": "rule_001",
        "dependencyType": "prerequisite",
        "description": "rule_003 依赖本规则",
        "dependedByRuleName": "连续排班限制"
      }
    ]
  }
}
```

### 3.2 创建规则依赖

**POST** `/api/v1/rules/dependencies`

Request:
```json
{
  "ruleId": "rule_001",
  "dependsOnRuleId": "rule_002",
  "dependencyType": "prerequisite",
  "description": "频次限制需先满足休息规则"
}
```

Response (201):
```json
{
  "code": 0,
  "data": {
    "id": "dep_001",
    "ruleId": "rule_001",
    "dependsOnRuleId": "rule_002",
    "dependencyType": "prerequisite"
  }
}
```

## 4. 规则冲突 API

### 4.1 获取规则冲突

**GET** `/api/v1/rules/{ruleId}/conflicts`

Response:
```json
{
  "code": 0,
  "data": [
    {
      "id": "conflict_001",
      "ruleAId": "rule_001",
      "ruleBId": "rule_005",
      "conflictType": "mutual_exclusive",
      "resolution": "priority",
      "description": "频次限制与均衡偏好可能冲突",
      "ruleAName": "早班周频次限制",
      "ruleBName": "排班均衡"
    }
  ]
}
```

### 4.2 创建规则冲突

**POST** `/api/v1/rules/conflicts`

Request:
```json
{
  "ruleAId": "rule_001",
  "ruleBId": "rule_005",
  "conflictType": "mutual_exclusive",
  "resolution": "priority",
  "description": "频次限制与均衡偏好可能冲突"
}
```

## 5. 班次依赖 API

### 5.1 获取班次依赖

**GET** `/api/v1/shifts/dependencies`

Query: `orgId=org_001`

Response:
```json
{
  "code": 0,
  "data": [
    {
      "id": "sdep_001",
      "dependentShiftId": "shift_001",
      "dependsOnShiftId": "shift_003",
      "dependencyType": "schedule_after",
      "description": "夜班排完后再排早班",
      "dependentShiftName": "早班",
      "dependsOnShiftName": "夜班"
    }
  ]
}
```

### 5.2 创建班次依赖

**POST** `/api/v1/shifts/dependencies`

Request:
```json
{
  "dependentShiftId": "shift_001",
  "dependsOnShiftId": "shift_003",
  "dependencyType": "schedule_after",
  "description": "夜班排完后再排早班"
}
```

Response (201):
```json
{
  "code": 0,
  "data": {
    "id": "sdep_001"
  }
}
```

## 6. 规则组织视图 API

### 6.1 获取组织视图

**GET** `/api/v1/rules/organization`

Query: `orgId=org_001`

Response:
```json
{
  "code": 0,
  "data": {
    "categories": {
      "constraint": {
        "label": "约束型规则",
        "count": 8,
        "subCategories": {
          "frequency": {
            "label": "频次限制",
            "rules": [
              {"id": "rule_001", "name": "早班周频次限制", "ruleType": "maxCount", "priority": 8}
            ]
          },
          "continuity": {
            "label": "连续性限制",
            "rules": [
              {"id": "rule_003", "name": "连续排班限制", "ruleType": "consecutiveMax", "priority": 9}
            ]
          }
        }
      },
      "preference": {
        "label": "偏好型规则",
        "count": 3,
        "subCategories": {}
      }
    },
    "dependencies": [
      {"from": "rule_001", "to": "rule_002", "type": "prerequisite"}
    ],
    "conflicts": [
      {"ruleA": "rule_001", "ruleB": "rule_005", "type": "mutual_exclusive", "resolution": "priority"}
    ],
    "shiftDependencies": [
      {"dependent": "shift_001", "dependsOn": "shift_003", "type": "schedule_after"}
    ],
    "statistics": {
      "totalRules": 11,
      "parsedRules": 9,
      "unparsedRules": 2,
      "constraintCount": 8,
      "preferenceCount": 3,
      "dependencyCount": 0
    }
  }
}
```

## 7. 规则创建/更新扩展

### 7.1 创建规则（V4 扩展字段）

**POST** `/api/v1/rules`

Request（在已有字段基础上新增 V4 字段）:
```json
{
  "orgId": "org_001",
  "name": "早班周频次限制",
  "description": "早班每人每周最多排3次",
  "ruleType": "maxCount",
  "maxCount": 3,
  "timeScope": "same_week",
  "applyScope": "shift",
  "priority": 8,
  "isActive": true,
  "associations": [
    {"associationType": "shift", "associationId": "shift_001", "role": "target"}
  ],
  
  "category": "constraint",
  "subCategory": "frequency",
  "originalRuleId": null
}
```

### 7.2 从解析结果保存规则

**POST** `/api/v1/rules/from-parse`

Request:
```json
{
  "orgId": "org_001",
  "parseResult": {
    "ruleName": "早班周频次限制",
    "ruleType": "maxCount",
    "category": "constraint",
    "subCategory": "frequency",
    "maxCount": 3,
    "timeScope": "same_week",
    "applyScope": "shift",
    "associationTargets": [
      {"type": "shift", "name": "早班", "id": "shift_001", "role": "target"}
    ],
    "suggestedPriority": 8
  }
}
```

Response (201):
```json
{
  "code": 0,
  "data": {
    "id": "rule_new_001",
    "name": "早班周频次限制",
    "ruleType": "maxCount",
    "category": "constraint",
    "subCategory": "frequency"
  }
}
```

## 8. 错误码定义

| 错误码 | 说明 |
|-------|------|
| 40001 | 规则文本为空 |
| 40002 | 规则解析失败 |
| 40003 | 结构验证失败 |
| 40004 | 关联对象未找到 |
| 40005 | 循环依赖检测到 |
| 40006 | 重复冲突声明 |
| 40007 | 自依赖/自冲突 |
| 40008 | 班次不存在 |
| 40009 | 规则不存在 |
| 50001 | LLM 调用失败 |
| 50002 | 内部服务错误 |

## 9. MCP Server 扩展

rostering-server 需新增以下 MCP Tool，供 V4 工作流的 Agent 调用：

```go
// Tool: get_rule_dependencies
// 描述: 获取指定规则的依赖关系
// 输入: {ruleId: string}
// 输出: {dependsOn: [...], dependedBy: [...]}

// Tool: get_shift_execution_order
// 描述: 获取考虑依赖关系后的班次执行顺序
// 输入: {orgId: string, shiftIds: []string}
// 输出: {order: []string, hasCycles: bool}

// Tool: get_rule_organization_view
// 描述: 获取规则组织视图（分类 + 依赖 + 冲突）
// 输入: {orgId: string}
// 输出: {categories: {...}, dependencies: [...], conflicts: [...]}
```
