# V4排班规则梳理与组织系统设计方案

## 1. 概述

### 1.1 设计目标

1. **规则语义化录入**：用户只需输入自然语言描述的规则，系统自动解析并拆解为结构化规则
2. **规则预组织**：在规则录入时完成规则分类、依赖关系分析，避免每次排班时重复处理
3. **依赖关系驱动**：基于规则间的依赖关系和冲突关系组织排班流程，确保被依赖的班次和人员优先排班
4. **V4排班逻辑**：全新的排班工作流，充分利用规则组织结果，提升排班质量和效率

### 1.2 核心设计理念

- **录入时解析，排班时应用**：规则解析和分类在录入阶段完成，存储到数据库，排班时直接使用
- **依赖关系优先**：按照依赖关系确定班次和规则的执行顺序
- **分类体系清晰**：不强制正向/负向，而是按作用方式（约束型/偏好型/依赖型）分类

## 2. 规则分类体系

### 2.1 规则分类定义

#### 2.1.1 约束型规则（Constraint Rules）
**特点**：必须严格遵守，违反会导致排班失败

| 子类型 | 规则类型 | 说明 | 示例 |
|--------|---------|------|------|
| 禁止型 | exclusive | 排他规则 | "排了A班就不能排B班" |
| 禁止型 | forbidden_day | 禁止日期 | "禁止在周末排班" |
| 限制型 | maxCount | 最大次数 | "每周最多3次" |
| 限制型 | consecutiveMax | 连续天数限制 | "连续不超过2天" |
| 限制型 | MinRestDays | 最少休息天数 | "夜班后至少休息1天" |
| 必须型 | required_together | 必须同时 | "排了A班必须排B班" |
| 必须型 | periodic | 周期性要求 | "隔周上夜班" |

#### 2.1.2 偏好型规则（Preference Rules）
**特点**：尽量满足，但不强制，冲突时可放宽

| 子类型 | 规则类型 | 说明 | 示例 |
|--------|---------|------|------|
| 优先型 | preferred | 偏好规则 | "优先安排周末休息" |
| 建议型 | preferred (低优先级) | 建议规则 | "建议避免连续夜班" |
| 可合并型 | combinable | 可合并 | "A班和B班可以同时排同一人" |

#### 2.1.3 依赖型规则（Dependency Rules）
**特点**：定义了班次间或规则间的依赖关系

| 子类型 | 说明 | 示例 |
|--------|------|------|
| 来源依赖 | 人员必须来自前一日某班次 | "下夜班人员必须来自前一日的上半夜班" |
| 资源预留 | 当日班次人员需保留给次日 | "当日上半夜班人员需保留给次日下夜班" |
| 顺序依赖 | 规则A必须在规则B之前执行 | "来源限制规则需要前一日数据" |

### 2.2 规则分类存储

在数据库中添加分类字段：

```sql
ALTER TABLE scheduling_rules ADD COLUMN category VARCHAR(32) COMMENT '规则分类: constraint/preference/dependency';
ALTER TABLE scheduling_rules ADD COLUMN sub_category VARCHAR(32) COMMENT '规则子分类: forbid/must/limit/prefer/suggest/source/resource/order';
```

## 3. 规则录入与解析系统

### 3.1 用户输入界面

#### 3.1.1 简化表单设计

**前端页面**：`frontend/web/src/pages/management/scheduling-rule/components/RuleInputDialog.vue`

```vue
<template>
  <el-dialog title="新增排班规则" width="800px">
    <el-form :model="formData" :rules="rules">
      <!-- 规则名称 -->
      <el-form-item label="规则名称" prop="name">
        <el-input v-model="formData.name" placeholder="例如：夜班连续限制规则" />
      </el-form-item>

      <!-- 语义化规则描述（核心字段） -->
      <el-form-item label="规则描述" prop="ruleDescription">
        <el-input
          v-model="formData.ruleDescription"
          type="textarea"
          :rows="6"
          placeholder="请用自然语言描述规则，例如：&#10;1. 夜班最多连续2天&#10;2. 下夜班人员必须来自前一日的上半夜班&#10;3. 每周最多工作5天"
        />
        <div class="form-tip">
          <div>💡 提示：可以输入多条规则，系统会自动解析并拆解</div>
          <div>示例：</div>
          <ul>
            <li>"夜班最多连续2天，之后必须休息至少1天"</li>
            <li>"下夜班人员必须来自前一日的上半夜班"</li>
            <li>"张三每周最多上3次夜班"</li>
          </ul>
        </div>
      </el-form-item>

      <!-- 应用范围（可选，系统可自动识别） -->
      <el-form-item label="应用范围" prop="applyScope">
        <el-select v-model="formData.applyScope" placeholder="系统将自动识别，也可手动指定">
          <el-option label="全局规则" value="global" />
          <el-option label="特定对象" value="specific" />
        </el-select>
      </el-form-item>

      <!-- 优先级 -->
      <el-form-item label="优先级" prop="priority">
        <el-slider v-model="formData.priority" :min="1" :max="10" />
        <div class="form-tip">1-10，数字越小优先级越高</div>
      </el-form-item>

      <!-- 生效时间（可选） -->
      <el-form-item label="生效时间">
        <el-date-picker
          v-model="formData.validDateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
        />
      </el-form-item>
    </el-form>

    <!-- 解析预览（解析后显示） -->
    <el-divider v-if="parseResult">解析结果预览</el-divider>
    <div v-if="parseResult" class="parse-preview">
      <el-alert
        :title="`共解析出 ${parseResult.rules.length} 条规则`"
        type="success"
        :closable="false"
      />
      <el-table :data="parseResult.rules" border>
        <el-table-column prop="name" label="规则名称" />
        <el-table-column prop="category" label="分类" />
        <el-table-column prop="ruleType" label="规则类型" />
        <el-table-column prop="applyScope" label="应用范围" />
        <el-table-column prop="description" label="规则说明" show-overflow-tooltip />
      </el-table>
    </div>

    <template #footer>
      <el-button @click="handleCancel">取消</el-button>
      <el-button @click="handleParse" :loading="parsing">解析规则</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="saving" :disabled="!parseResult">
        保存规则
      </el-button>
    </template>
  </el-dialog>
</template>
```

### 3.2 规则解析服务

#### 3.2.1 后端API设计

**文件**：`services/management-service/internal/service/rule_parser_service.go`

```go
package service

import (
    "context"
    "fmt"
    "jusha/mcp/pkg/ai"
    "jusha/mcp/pkg/logging"
    "services/management-service/domain/model"
)

// RuleParserService 规则解析服务
type RuleParserService struct {
    logger    logging.ILogger
    aiFactory *ai.AIProviderFactory
    ruleRepo  repository.ISchedulingRuleRepository
}

// ParseRuleRequest 规则解析请求
type ParseRuleRequest struct {
    OrgID            string     `json:"orgId"`
    Name             string     `json:"name"`
    RuleDescription  string     `json:"ruleDescription"`  // 用户输入的语义化规则
    ApplyScope       string     `json:"applyScope,omitempty"` // 可选，系统可自动识别
    Priority         int        `json:"priority"`
    ValidFrom        *time.Time `json:"validFrom,omitempty"`
    ValidTo          *time.Time `json:"validTo,omitempty"`
}

// ParseRuleResponse 规则解析响应
type ParseRuleResponse struct {
    OriginalRule     string              `json:"originalRule"`     // 原始规则描述
    ParsedRules      []*ParsedRule       `json:"parsedRules"`      // 解析后的规则列表
    Dependencies     []RuleDependency    `json:"dependencies"`     // 识别出的依赖关系
    Conflicts        []RuleConflict      `json:"conflicts"`        // 识别出的冲突关系
    Reasoning        string              `json:"reasoning"`        // 解析说明
}

// ParsedRule 解析后的规则
type ParsedRule struct {
    Name             string              `json:"name"`
    Category          string              `json:"category"`          // constraint/preference/dependency
    SubCategory       string              `json:"subCategory"`       // forbid/must/limit/prefer/suggest/source/resource
    RuleType         model.RuleType      `json:"ruleType"`
    ApplyScope       model.ApplyScope    `json:"applyScope"`
    TimeScope        model.TimeScope     `json:"timeScope"`
    Description      string              `json:"description"`
    RuleData         string              `json:"ruleData"`
    
    // 数值型参数
    MaxCount         *int                `json:"maxCount,omitempty"`
    ConsecutiveMax   *int                `json:"consecutiveMax,omitempty"`
    IntervalDays     *int                `json:"intervalDays,omitempty"`
    MinRestDays      *int                `json:"minRestDays,omitempty"`
    
    Priority         int                 `json:"priority"`
    ValidFrom        *time.Time          `json:"validFrom,omitempty"`
    ValidTo          *time.Time          `json:"validTo,omitempty"`
    
    // 关联信息
    Associations     []model.RuleAssociation `json:"associations,omitempty"`
    
    // 依赖和冲突关系
    Dependencies     []string            `json:"dependencies,omitempty"`  // 依赖的其他规则ID（解析时为空，保存后填充）
    Conflicts        []string            `json:"conflicts,omitempty"`   // 冲突的其他规则ID
}

// RuleDependency 规则依赖关系
type RuleDependency struct {
    DependentRuleName   string  `json:"dependentRuleName"`   // 被依赖的规则（需要先执行）
    DependentOnRuleName string  `json:"dependentOnRuleName"` // 依赖的规则（后执行）
    DependencyType      string  `json:"dependencyType"`      // time/source/resource/order
    Description         string  `json:"description"`
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
    RuleName1    string  `json:"ruleName1"`
    RuleName2    string  `json:"ruleName2"`
    ConflictType string  `json:"conflictType"`  // exclusive/resource/time/frequency
    Description  string  `json:"description"`
}

// ParseRule 解析语义化规则
func (s *RuleParserService) ParseRule(ctx context.Context, req *ParseRuleRequest) (*ParseRuleResponse, error) {
    // 1. 构建LLM提示词
    systemPrompt := s.buildParseSystemPrompt()
    userPrompt := s.buildParseUserPrompt(req)
    
    // 2. 调用LLM解析
    resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
    if err != nil {
        return nil, fmt.Errorf("LLM解析失败: %w", err)
    }
    
    // 3. 解析LLM响应
    parseResult, err := s.parseLLMResponse(resp.Content)
    if err != nil {
        return nil, fmt.Errorf("解析LLM响应失败: %w", err)
    }
    
    // 4. 检查与现有规则的冲突
    conflicts, err := s.checkConflictsWithExisting(ctx, req.OrgID, parseResult.ParsedRules)
    if err != nil {
        return nil, fmt.Errorf("检查冲突失败: %w", err)
    }
    
    return &ParseRuleResponse{
        OriginalRule: req.RuleDescription,
        ParsedRules:  parseResult.ParsedRules,
        Dependencies: parseResult.Dependencies,
        Conflicts:    conflicts,
        Reasoning:    parseResult.Reasoning,
    }, nil
}

// SaveParsedRules 保存解析后的规则
func (s *RuleParserService) SaveParsedRules(ctx context.Context, orgID string, parsedRules []*ParsedRule) ([]*model.SchedulingRule, error) {
    savedRules := make([]*model.SchedulingRule, 0, len(parsedRules))
    
    for _, parsed := range parsedRules {
        rule := &model.SchedulingRule{
            ID:          uuid.New().String(),
            OrgID:       orgID,
            Name:        parsed.Name,
            Description: parsed.Description,
            RuleType:    parsed.RuleType,
            ApplyScope:  parsed.ApplyScope,
            TimeScope:   parsed.TimeScope,
            RuleData:    parsed.RuleData,
            MaxCount:    parsed.MaxCount,
            ConsecutiveMax: parsed.ConsecutiveMax,
            IntervalDays:   parsed.IntervalDays,
            MinRestDays:    parsed.MinRestDays,
            Priority:    parsed.Priority,
            ValidFrom:   parsed.ValidFrom,
            ValidTo:     parsed.ValidTo,
            IsActive:    true,
            Associations: parsed.Associations,
        }
        
        // 保存规则
        if err := s.ruleRepo.Create(ctx, rule); err != nil {
            return nil, fmt.Errorf("保存规则失败: %w", err)
        }
        
        savedRules = append(savedRules, rule)
    }
    
    // 保存依赖关系（存储到新表 rule_dependencies）
    // 保存冲突关系（存储到新表 rule_conflicts）
    
    return savedRules, nil
}
```

#### 3.2.2 LLM解析提示词

```go
func (s *RuleParserService) buildParseSystemPrompt() string {
    return `你是排班规则解析专家。你的任务是将用户输入的自然语言规则描述，解析并拆解为多条结构化的排班规则。

## 规则分类体系

### 1. 约束型规则（Constraint Rules）
- **禁止型**：exclusive（排他）、forbidden_day（禁止日期）
- **限制型**：maxCount（最大次数）、consecutiveMax（连续天数限制）、MinRestDays（最少休息天数）
- **必须型**：required_together（必须同时）、periodic（周期性要求）

### 2. 偏好型规则（Preference Rules）
- **优先型**：preferred（偏好）
- **可合并型**：combinable（可合并）

### 3. 依赖型规则（Dependency Rules）
- **来源依赖**：人员必须来自前一日某班次
- **资源预留**：当日班次人员需保留给次日
- **顺序依赖**：规则A必须在规则B之前执行

## 解析要求

1. **识别规则类型**：根据语义判断规则类型（exclusive/maxCount/periodic等）
2. **提取数值参数**：识别并提取数值型参数（如"最多3次"中的3）
3. **识别应用范围**：判断是全局规则还是针对特定对象（员工/班次/分组）
4. **识别时间范围**：判断是same_day/same_week/same_month/custom
5. **识别依赖关系**：识别规则间的依赖关系（如"下夜班依赖前一日上半夜班"）
6. **识别冲突关系**：识别规则间的冲突关系（如互斥规则）

## 输出格式

请返回JSON格式：
{
  "parsedRules": [
    {
      "name": "规则名称",
      "category": "constraint/preference/dependency",
      "subCategory": "forbid/must/limit/prefer/suggest/source/resource/order",
      "ruleType": "exclusive/maxCount/periodic/...",
      "applyScope": "global/specific",
      "timeScope": "same_day/same_week/same_month/custom",
      "description": "规则说明",
      "ruleData": "规则数据（语义化描述）",
      "maxCount": 3,  // 如果有
      "consecutiveMax": 2,  // 如果有
      "intervalDays": 7,  // 如果有
      "minRestDays": 1,  // 如果有
      "priority": 5,
      "associations": [
        {
          "associationType": "employee/shift/group",
          "associationId": "对象ID"
        }
      ]
    }
  ],
  "dependencies": [
    {
      "dependentRuleName": "被依赖的规则名",
      "dependentOnRuleName": "依赖的规则名",
      "dependencyType": "time/source/resource/order",
      "description": "依赖关系描述"
    }
  ],
  "conflicts": [
    {
      "ruleName1": "规则1名称",
      "ruleName2": "规则2名称",
      "conflictType": "exclusive/resource/time/frequency",
      "description": "冲突描述"
    }
  ],
  "reasoning": "解析思路说明"
}`
}
```

### 3.3 数据库设计

#### 3.3.1 规则表扩展

```sql
-- 扩展规则表，添加分类字段
ALTER TABLE scheduling_rules 
ADD COLUMN category VARCHAR(32) COMMENT '规则分类: constraint/preference/dependency',
ADD COLUMN sub_category VARCHAR(32) COMMENT '规则子分类: forbid/must/limit/prefer/suggest/source/resource/order',
ADD COLUMN original_rule_id VARCHAR(64) COMMENT '原始规则ID（如果是从语义化规则解析出来的）',
ADD INDEX idx_category (category),
ADD INDEX idx_sub_category (sub_category);
```

#### 3.3.2 规则依赖关系表（新增）

```sql
CREATE TABLE rule_dependencies (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    dependent_rule_id VARCHAR(64) NOT NULL COMMENT '被依赖的规则ID（需要先执行）',
    dependent_on_rule_id VARCHAR(64) NOT NULL COMMENT '依赖的规则ID（后执行）',
    dependency_type VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource/order',
    description TEXT COMMENT '依赖关系描述',
    created_at DATETIME NOT NULL,
    INDEX idx_dependent_rule (dependent_rule_id),
    INDEX idx_dependent_on_rule (dependent_on_rule_id),
    INDEX idx_org (org_id)
) COMMENT='规则依赖关系表';
```

#### 3.3.3 规则冲突关系表（新增）

```sql
CREATE TABLE rule_conflicts (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    rule_id_1 VARCHAR(64) NOT NULL,
    rule_id_2 VARCHAR(64) NOT NULL,
    conflict_type VARCHAR(32) NOT NULL COMMENT '冲突类型: exclusive/resource/time/frequency',
    description TEXT COMMENT '冲突描述',
    resolution_priority INT COMMENT '解决优先级（数字越小越优先）',
    created_at DATETIME NOT NULL,
    INDEX idx_rule_1 (rule_id_1),
    INDEX idx_rule_2 (rule_id_2),
    INDEX idx_org (org_id)
) COMMENT='规则冲突关系表';
```

#### 3.3.4 班次依赖关系表（新增）

```sql
CREATE TABLE shift_dependencies (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    dependent_shift_id VARCHAR(64) NOT NULL COMMENT '被依赖的班次ID（需要先排）',
    dependent_on_shift_id VARCHAR(64) NOT NULL COMMENT '依赖的班次ID（后排）',
    dependency_type VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource',
    rule_id VARCHAR(64) COMMENT '产生此依赖关系的规则ID',
    description TEXT COMMENT '依赖关系描述',
    created_at DATETIME NOT NULL,
    INDEX idx_dependent_shift (dependent_shift_id),
    INDEX idx_dependent_on_shift (dependent_on_shift_id),
    INDEX idx_rule (rule_id),
    INDEX idx_org (org_id)
) COMMENT='班次依赖关系表';
```

## 4. V4排班工作流设计

### 4.1 工作流架构

**文件结构**：
```
agents/rostering/internal/workflow/schedule_v4/
├── main.go                    # 工作流注册
├── create/
│   ├── definition.go         # 工作流定义
│   ├── actions.go            # 动作实现
│   ├── context.go            # 上下文定义
│   └── helpers.go            # 辅助函数
└── executor/
    ├── rule_organizer.go     # 规则组织器（新增）
    ├── dependency_analyzer.go # 依赖关系分析器（新增）
    └── v4_executor.go        # V4执行器
```

### 4.2 规则组织器

**文件**：`agents/rostering/internal/workflow/schedule_v4/executor/rule_organizer.go`

```go
package executor

import (
    "context"
    "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// RuleOrganization 规则组织结果
type RuleOrganization struct {
    // 分类后的规则
    ConstraintRules []*ClassifiedRule  // 约束型规则
    PreferenceRules []*ClassifiedRule  // 偏好型规则
    DependencyRules []*ClassifiedRule  // 依赖型规则
    
    // 依赖关系
    ShiftDependencies []ShiftDependency  // 班次依赖关系
    RuleDependencies  []RuleDependency  // 规则依赖关系
    
    // 冲突关系
    RuleConflicts     []RuleConflict    // 规则冲突关系
    
    // 执行顺序
    ShiftExecutionOrder []string         // 班次执行顺序（按依赖关系排序）
    RuleExecutionOrder  []string         // 规则执行顺序（按依赖关系排序）
}

// ClassifiedRule 分类后的规则
type ClassifiedRule struct {
    Rule            *model.Rule
    Category        string   // constraint/preference/dependency
    SubCategory     string   // forbid/must/limit/prefer/suggest/source/resource/order
    Dependencies    []string // 依赖的其他规则ID
    Conflicts       []string // 冲突的其他规则ID
    ExecutionOrder  int      // 执行顺序（数字越小越先执行）
}

// ShiftDependency 班次依赖关系
type ShiftDependency struct {
    DependentShiftID   string  // 被依赖的班次（需要先排）
    DependentOnShiftID string  // 依赖的班次（后排）
    DependencyType     string  // time/source/resource
    RuleID             string  // 产生此依赖关系的规则ID
    Description        string  // 依赖关系描述
}

// RuleDependency 规则依赖关系
type RuleDependency struct {
    DependentRuleID   string  // 被依赖的规则（需要先执行）
    DependentOnRuleID string  // 依赖的规则（后执行）
    DependencyType    string  // time/source/resource/order
    Description       string  // 依赖关系描述
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
    RuleID1       string  // 冲突的规则1
    RuleID2       string  // 冲突的规则2
    ConflictType  string  // exclusive/resource/time/frequency
    Description   string  // 冲突描述
    ResolutionPriority int // 解决优先级
}

// RuleOrganizer 规则组织器
type RuleOrganizer struct {
    logger    logging.ILogger
    ruleRepo  repository.ISchedulingRuleRepository
}

// OrganizeRules 组织规则
func (o *RuleOrganizer) OrganizeRules(
    ctx context.Context,
    rules []*model.Rule,
    shifts []*model.Shift,
) (*RuleOrganization, error) {
    // 1. 从数据库加载规则分类、依赖关系、冲突关系
    // 2. 构建规则分类映射
    // 3. 构建依赖关系图
    // 4. 拓扑排序计算执行顺序
    // 5. 返回组织结果
}
```

### 4.3 V4排班执行流程

```
1. 规则组织阶段（排班前）
   ├─ 加载所有规则
   ├─ 从数据库加载规则分类、依赖关系、冲突关系
   ├─ 构建依赖关系图
   └─ 计算执行顺序

2. 班次排序阶段
   ├─ 根据班次依赖关系排序
   ├─ 结合SchedulingPriority
   └─ 生成班次执行顺序

3. 排班执行阶段
   ├─ 按班次执行顺序逐个排班
   ├─ 优先处理约束型规则
   ├─ 然后处理依赖型规则
   └─ 最后处理偏好型规则

4. 冲突解决阶段
   ├─ 检测规则冲突
   ├─ 按优先级解决冲突
   └─ 生成冲突报告
```

## 5. 前端页面改造

### 5.1 规则录入页面

**文件**：`frontend/web/src/pages/management/scheduling-rule/components/RuleInputDialog.vue`

主要改动：
1. 简化表单，只保留"规则描述"核心字段
2. 添加"解析规则"按钮，调用解析API
3. 显示解析结果预览
4. 支持批量保存多条解析后的规则

### 5.2 规则列表页面

**文件**：`frontend/web/src/pages/management/scheduling-rule/index.vue`

主要改动：
1. 显示规则分类（约束型/偏好型/依赖型）
2. 显示规则依赖关系
3. 显示规则冲突关系
4. 支持按分类筛选

### 5.3 依赖关系可视化

**新增文件**：`frontend/web/src/pages/management/scheduling-rule/components/DependencyGraph.vue`

使用图形库（如D3.js或ECharts）可视化：
- 班次依赖关系图
- 规则依赖关系图
- 规则冲突关系图

## 6. API设计

### 6.1 规则解析API

```
POST /api/v1/scheduling-rules/parse
Request:
{
  "orgId": "org_xxx",
  "name": "规则名称",
  "ruleDescription": "语义化规则描述",
  "priority": 5,
  "validFrom": "2025-01-01",
  "validTo": "2025-12-31"
}

Response:
{
  "originalRule": "原始规则描述",
  "parsedRules": [...],
  "dependencies": [...],
  "conflicts": [...],
  "reasoning": "解析说明"
}
```

### 6.2 规则保存API

```
POST /api/v1/scheduling-rules/batch
Request:
{
  "orgId": "org_xxx",
  "rules": [
    {
      "name": "规则名称",
      "category": "constraint",
      "subCategory": "limit",
      "ruleType": "maxCount",
      ...
    }
  ],
  "dependencies": [...],
  "conflicts": [...]
}

Response:
{
  "savedRules": [...],
  "dependencies": [...],
  "conflicts": [...]
}
```

### 6.3 规则组织API

```
GET /api/v1/scheduling-rules/organize?orgId=xxx
Response:
{
  "constraintRules": [...],
  "preferenceRules": [...],
  "dependencyRules": [...],
  "shiftDependencies": [...],
  "ruleDependencies": [...],
  "ruleConflicts": [...],
  "shiftExecutionOrder": [...],
  "ruleExecutionOrder": [...]
}
```

## 7. 实施计划

### 阶段1：数据库和模型设计（1周）
- [ ] 扩展规则表，添加分类字段
- [ ] 创建规则依赖关系表
- [ ] 创建规则冲突关系表
- [ ] 创建班次依赖关系表
- [ ] 更新领域模型

### 阶段2：规则解析服务（2周）
- [ ] 实现规则解析服务
- [ ] 实现LLM解析提示词
- [ ] 实现规则保存逻辑
- [ ] 实现依赖关系和冲突关系检测

### 阶段3：前端页面改造（2周）
- [ ] 改造规则录入页面
- [ ] 改造规则列表页面
- [ ] 实现依赖关系可视化
- [ ] 实现解析结果预览

### 阶段4：V4排班工作流（3周）
- [ ] 创建V4工作流框架
- [ ] 实现规则组织器
- [ ] 实现依赖关系分析器
- [ ] 实现V4执行器
- [ ] 集成到排班流程

### 阶段5：测试和优化（2周）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 文档完善

## 8. 技术要点

### 8.1 拓扑排序算法

用于计算班次和规则的执行顺序：

```go
// TopologicalSort 拓扑排序
func TopologicalSort(nodes []string, edges []DependencyEdge) ([]string, error) {
    // 构建入度图
    inDegree := make(map[string]int)
    graph := make(map[string][]string)
    
    for _, edge := range edges {
        inDegree[edge.To]++
        graph[edge.From] = append(graph[edge.From], edge.To)
    }
    
    // 找到所有入度为0的节点
    queue := []string{}
    for _, node := range nodes {
        if inDegree[node] == 0 {
            queue = append(queue, node)
        }
    }
    
    // 拓扑排序
    result := []string{}
    for len(queue) > 0 {
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)
        
        for _, neighbor := range graph[node] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                queue = append(queue, neighbor)
            }
        }
    }
    
    // 检查是否有环
    if len(result) != len(nodes) {
        return nil, fmt.Errorf("存在循环依赖")
    }
    
    return result, nil
}
```

### 8.2 依赖关系检测

在规则解析时，自动检测规则间的依赖关系：

```go
func detectDependencies(parsedRules []*ParsedRule) []RuleDependency {
    dependencies := []RuleDependency{}
    
    for i, rule1 := range parsedRules {
        for j, rule2 := range parsedRules {
            if i == j {
                continue
            }
            
            // 检测来源依赖
            if isSourceDependency(rule1, rule2) {
                dependencies = append(dependencies, RuleDependency{
                    DependentRuleName: rule2.Name,
                    DependentOnRuleName: rule1.Name,
                    DependencyType: "source",
                    Description: fmt.Sprintf("%s的人员必须来自%s", rule2.Name, rule1.Name),
                })
            }
            
            // 检测资源预留
            if isResourceReservation(rule1, rule2) {
                dependencies = append(dependencies, RuleDependency{
                    DependentRuleName: rule1.Name,
                    DependentOnRuleName: rule2.Name,
                    DependencyType: "resource",
                    Description: fmt.Sprintf("%s的人员需保留给%s", rule1.Name, rule2.Name),
                })
            }
        }
    }
    
    return dependencies
}
```

## 9. 预期效果

1. **规则录入简化**：用户只需输入自然语言，系统自动解析
2. **规则组织清晰**：规则按分类、依赖关系组织，不再混乱
3. **排班质量提升**：基于依赖关系确定执行顺序，避免逻辑混乱
4. **性能优化**：规则解析在录入时完成，排班时直接使用，避免重复处理
5. **可维护性提升**：规则分类清晰，依赖关系明确，便于维护和扩展

## 10. 风险与应对

### 10.1 LLM解析准确性
**风险**：LLM可能解析错误
**应对**：
- 提供解析结果预览，用户可手动调整
- 支持编辑解析后的规则
- 建立解析结果验证机制

### 10.2 依赖关系循环
**风险**：可能存在循环依赖
**应对**：
- 使用拓扑排序检测循环依赖
- 提示用户解决循环依赖
- 提供依赖关系可视化，便于发现循环

### 10.3 向后兼容
**风险**：V4与V3不兼容
**应对**：
- 保留V3工作流，支持切换
- 提供数据迁移工具
- 逐步迁移，不强制切换

---

## 11. 设计方案评审意见

### 11.1 评审总结

| 维度 | 评分 | 评价 |
|------|------|------|
| **V3问题覆盖度** | ⭐⭐⭐ (3/5) | 覆盖了规则分类和依赖关系两个核心问题，但未触及 V3 最严重的「LLM 滥用」反模式 |
| **输出稳定性改善** | ⭐⭐ (2/5) | 规则预组织有帮助，但排班执行阶段仍依赖 LLM 理解规则文本，稳定性瓶颈未解 |
| **一致性保障** | ⭐⭐ (2/5) | 拓扑排序保证了执行顺序一致性，但缺乏确定性校验引擎，LLM 输出一致性无法保证 |
| **可落地性** | ⭐⭐⭐ (3/5) | 数据库设计和 API 设计完整，但缺少最关键的「确定性规则引擎」设计 |
| **成本优化** | ⭐⭐ (2/5) | 未解决 V3 中每班次每天 5-7 次 LLM 调用的成本问题 |

### 11.2 V3 核心问题诊断与 V4 覆盖度分析

#### 11.2.1 V3 核心问题清单

经过对 V3 代码的深入分析，识别出以下核心问题：

| # | 问题 | 严重程度 | V4方案是否覆盖 | 说明 |
|---|------|---------|---------------|------|
| P1 | **3-LLM 预分析反模式**：用 LLM 做人员过滤(LLM-1)、规则过滤(LLM-2)、冲突检测(LLM-3)，这些本质上是确定性计算 | 🔴 致命 | ❌ 未覆盖 | V4 方案未改变排班执行阶段的 LLM 调用模式 |
| P2 | **规则纯文本传递**：规则的数值参数(`MaxCount=3`, `ConsecutiveMax=2`)只在 `RuleData` 自然语言中，未结构化传递给 LLM | 🔴 致命 | 🟡 部分覆盖 | V4 增加了分类但未设计结构化规则传递格式 |
| P3 | **无规则分类体系**：约束/偏好/依赖不分，LLM 无法区分"必须遵守"和"尽量满足" | 🟠 严重 | ✅ 已覆盖 | V4 核心改进 |
| P4 | **无班次依赖关系**：只靠 `SchedulingPriority` 整数排序，不知道"谁依赖谁" | 🟠 严重 | ✅ 已覆盖 | V4 核心改进 |
| P5 | **LLM 间信息丢失**：LLM-2 过滤规则后传给 LLM-4 的是压缩后的 `{name,constraint}` 键值对，原始规则参数丢失 | 🟠 严重 | ❌ 未覆盖 | V4 方案未设计新的信息传递机制 |
| P6 | **LLM 幻觉叠加**：3个 LLM 各自独立判断，错误逐层叠加（error compounding） | 🟠 严重 | ❌ 未覆盖 | 3-LLM 模式未改变 |
| P7 | **无确定性校验引擎**：校验也用 LLM，结果不确定 | 🟠 严重 | ❌ 未覆盖 | V4 方案缺少确定性校验器设计 |
| P8 | **LLM 调用成本高**：20个班次×7天≈700+次 LLM 调用，其中 58% 是可代码化的预分析 | 🟡 中等 | ❌ 未覆盖 | 未优化 LLM 调用次数 |
| P9 | **规则间冲突关系未建模** | 🟡 中等 | ✅ 已覆盖 | V4 核心改进 |
| P10 | **候选人员排除原因不透明**：LLM-3 过滤人员后静默移除，排班 LLM 不知道为什么某些人不在列表中 | 🟡 中等 | ❌ 未覆盖 | |

**结论：V4 方案覆盖了 P3/P4/P9 三个问题（规则分类 + 依赖关系 + 冲突关系），但最致命的 P1（LLM 滥用反模式）和 P2（规则非结构化传递）未触及。**

#### 11.2.2 V4 方案的核心缺陷

**缺陷一：只改了"规则录入侧"，未改"规则应用侧"**

V4 方案的规则组织器（`RuleOrganizer`）在排班前将规则分类并排序，但排班执行阶段（`progressive_scheduling.go`）的核心流程未改变：
- 仍然用 LLM-1 过滤人员（应该用代码做日期匹配）
- 仍然用 LLM-2 过滤规则（有了 `Associations` 和分类信息后完全可以代码化）
- 仍然用 LLM-3 检测冲突（频次统计、连班检查是纯数学运算）
- 规则仍然以自然语言传递给排班 LLM

**缺陷二：缺少「确定性规则引擎」**

V4 最需要但缺失的组件：一个**不依赖 LLM 的确定性规则引擎**。当规则已经结构化存储（有 `category`、`ruleType`、`maxCount` 等字段），大多数规则约束可以用代码精确执行：

| 规则类型 | 是否可代码化 | 说明 |
|---------|------------|------|
| `maxCount` | ✅ 100% 代码化 | 统计当前排班次数 ≤ maxCount |
| `consecutiveMax` | ✅ 100% 代码化 | 检查连续排班天数 ≤ consecutiveMax |
| `minRestDays` | ✅ 100% 代码化 | 检查两次排班间隔 ≥ minRestDays |
| `exclusive` | ✅ 100% 代码化 | 检查同日/同时段是否有排他班次 |
| `forbidden_day` | ✅ 100% 代码化 | 日期匹配 |
| `required_together` | ✅ 100% 代码化 | 检查关联班次是否同时存在 |
| `periodic` | ✅ 100% 代码化 | 周期性模式匹配 |
| `preferred` | 🟡 部分代码化 | 可计算偏好得分，不需 LLM |
| 复杂语义规则 | ❌ 需要 LLM | 仅不可结构化的自由文本规则 |

**缺陷三：LLM 语义化录入的"稳定性悖论"**

V4 用 LLM 解析规则输入是好的方向，但存在悖论：
- 用 LLM 解析规则以避免每次排班时 LLM 理解规则 → ✅ 正确
- 但如果 LLM 解析时就解析错了 → 错误被永久存储到数据库 → 每次排班都用错误规则
- 解析一次 vs 理解多次：错误放大效应

需要增加**人工确认 + 结构化验证**机制。

---

## 12. 补充设计：确定性规则引擎（核心增补）

### 12.1 设计理念：LLM 职责最小化

**核心原则：LLM 只做"创造性决策"，确定性计算全部代码化。**

```
V3 模式（当前）：
  LLM-1 人员过滤 → LLM-2 规则过滤 → LLM-3 冲突检测 → LLM-4 排班 → LLM-5 校验
  （5次 LLM 调用，全链路不确定）

V4 模式（目标）：
  代码 人员过滤 → 代码 规则过滤 → 代码 约束计算 → LLM 排班 → 代码 校验
  （1次 LLM 调用，仅排班决策不确定）
```

### 12.2 确定性规则引擎架构

**文件结构**：
```
agents/rostering/internal/engine/
├── rule_engine.go              # 规则引擎入口
├── constraint_checker.go       # 约束检查器（替代 LLM-3）
├── candidate_filter.go         # 候选人过滤器（替代 LLM-1）
├── rule_matcher.go             # 规则匹配器（替代 LLM-2）
├── schedule_validator.go       # 排班校验器（替代 LLM-5）
├── preference_scorer.go        # 偏好评分器
└── types.go                    # 类型定义
```

#### 12.2.1 规则引擎入口

**文件**：`agents/rostering/internal/engine/rule_engine.go`

```go
package engine

import (
    "context"
    "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// RuleEngine 确定性规则引擎
// 替代 V3 中的 LLM-1/LLM-2/LLM-3/LLM-5，用代码实现确定性规则计算
type RuleEngine struct {
    logger             logging.ILogger
    candidateFilter    *CandidateFilter    // 替代 LLM-1
    ruleMatcher        *RuleMatcher        // 替代 LLM-2
    constraintChecker  *ConstraintChecker  // 替代 LLM-3
    validator          *ScheduleValidator  // 替代 LLM-5
    preferenceScorer   *PreferenceScorer   // 偏好评分
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(logger logging.ILogger) *RuleEngine {
    return &RuleEngine{
        logger:            logger,
        candidateFilter:   NewCandidateFilter(logger),
        ruleMatcher:       NewRuleMatcher(logger),
        constraintChecker: NewConstraintChecker(logger),
        validator:         NewScheduleValidator(logger),
        preferenceScorer:  NewPreferenceScorer(logger),
    }
}

// PrepareSchedulingContext 为单个班次单个日期准备排班上下文
// 替代 V3 的 3-LLM 预分析，全部用代码完成
func (e *RuleEngine) PrepareSchedulingContext(
    ctx context.Context,
    input *SchedulingInput,
) (*SchedulingContext, error) {
    
    // 1. 规则匹配（替代 LLM-2）：根据 Associations + Category 精确匹配
    matchedRules := e.ruleMatcher.MatchRules(input.AllRules, input.ShiftID, input.Date)
    
    // 2. 候选人过滤（替代 LLM-1）：根据请假、固定排班等确定性数据过滤
    candidates, exclusionReasons := e.candidateFilter.Filter(
        input.AllStaff,
        input.PersonalNeeds,
        input.FixedAssignments,
        input.ShiftID,
        input.Date,
    )
    
    // 3. 约束检查（替代 LLM-3）：根据结构化规则参数计算每个候选人的约束状态
    constraintResults := e.constraintChecker.CheckAll(
        candidates,
        matchedRules,
        input.CurrentDraft,
        input.ShiftID,
        input.Date,
    )
    
    // 4. 偏好评分：为每个候选人计算偏好得分
    preferenceScores := e.preferenceScorer.Score(
        constraintResults.EligibleCandidates,
        matchedRules.PreferenceRules,
        input.CurrentDraft,
        input.Date,
    )
    
    return &SchedulingContext{
        ShiftID:            input.ShiftID,
        Date:               input.Date,
        RequiredCount:       input.RequiredCount,
        MatchedRules:        matchedRules,
        EligibleCandidates:  constraintResults.EligibleCandidates,
        ExcludedCandidates:  constraintResults.ExcludedCandidates,
        ExclusionReasons:    exclusionReasons,
        ConstraintDetails:   constraintResults.Details,
        PreferenceScores:    preferenceScores,
        // 传递给 LLM 的结构化摘要
        LLMBrief:            e.buildLLMBrief(constraintResults, preferenceScores, matchedRules),
    }, nil
}

// ValidateSchedule 确定性校验排班结果（替代 LLM-5）
func (e *RuleEngine) ValidateSchedule(
    ctx context.Context,
    schedule *model.ScheduleDraft,
    rules *MatchedRules,
    allDraft *model.ScheduleDraft, // 全局草稿
) (*ValidationResult, error) {
    return e.validator.Validate(schedule, rules, allDraft)
}

// buildLLMBrief 构建传递给 LLM 的结构化摘要
// 只包含 LLM 做决策所需的最少信息，格式固定，避免歧义
func (e *RuleEngine) buildLLMBrief(
    constraints *ConstraintCheckResult,
    preferences *PreferenceScoreResult,
    rules *MatchedRules,
) *LLMBrief {
    return &LLMBrief{
        // 可选人员（已通过所有硬约束），附带偏好评分
        Candidates: buildCandidateBriefs(constraints.EligibleCandidates, preferences),
        // 硬约束摘要（结构化，非自然语言）
        HardConstraints: buildConstraintBriefs(rules.ConstraintRules),
        // 软偏好摘要
        SoftPreferences: buildPreferenceBriefs(rules.PreferenceRules),
        // 排除人员及原因（透明化，帮助 LLM 理解可选范围）
        ExcludedWithReasons: buildExclusionBriefs(constraints.ExcludedCandidates),
    }
}
```

#### 12.2.2 约束检查器

**文件**：`agents/rostering/internal/engine/constraint_checker.go`

```go
package engine

import (
    "time"
    "jusha/agent/rostering/domain/model"
)

// ConstraintChecker 约束检查器（替代 LLM-3）
type ConstraintChecker struct {
    logger logging.ILogger
}

// ConstraintCheckResult 约束检查结果
type ConstraintCheckResult struct {
    EligibleCandidates []*CandidateStatus  // 通过所有硬约束的候选人
    ExcludedCandidates []*CandidateStatus  // 被排除的候选人（附原因）
    Details            []*ConstraintDetail // 详细约束检查记录
}

// CandidateStatus 候选人约束状态
type CandidateStatus struct {
    StaffID          string
    StaffName        string
    IsEligible       bool                  // 是否通过所有硬约束
    ViolatedRules    []*RuleViolation      // 违反的规则列表
    Warnings         []*RuleWarning        // 警告（接近限制）
    ConstraintScores map[string]float64    // 各约束的"剩余空间"评分
}

// CheckAll 检查所有候选人的所有约束
func (c *ConstraintChecker) CheckAll(
    candidates []*model.Staff,
    rules *MatchedRules,
    currentDraft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *ConstraintCheckResult {
    result := &ConstraintCheckResult{}
    
    for _, staff := range candidates {
        status := &CandidateStatus{
            StaffID:   staff.ID,
            StaffName: staff.Name,
            IsEligible: true,
            ConstraintScores: make(map[string]float64),
        }
        
        for _, rule := range rules.ConstraintRules {
            violation := c.checkSingleConstraint(staff, rule, currentDraft, shiftID, date)
            if violation != nil {
                if violation.IsHard {
                    status.IsEligible = false
                    status.ViolatedRules = append(status.ViolatedRules, violation)
                } else {
                    status.Warnings = append(status.Warnings, &RuleWarning{
                        RuleID:   rule.Rule.ID,
                        RuleName: rule.Rule.Name,
                        Message:  violation.Message,
                    })
                }
            }
            // 计算"剩余空间"评分（如 maxCount=5 已用 3 次，剩余空间=0.4）
            score := c.computeConstraintScore(staff, rule, currentDraft, shiftID, date)
            status.ConstraintScores[rule.Rule.ID] = score
        }
        
        if status.IsEligible {
            result.EligibleCandidates = append(result.EligibleCandidates, status)
        } else {
            result.ExcludedCandidates = append(result.ExcludedCandidates, status)
        }
    }
    
    return result
}

// checkSingleConstraint 检查单个约束规则
func (c *ConstraintChecker) checkSingleConstraint(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    switch rule.Rule.RuleType {
    case "maxCount":
        return c.checkMaxCount(staff, rule, draft, shiftID, date)
    case "consecutiveMax":
        return c.checkConsecutiveMax(staff, rule, draft, shiftID, date)
    case "minRestDays":
        return c.checkMinRestDays(staff, rule, draft, shiftID, date)
    case "exclusive":
        return c.checkExclusive(staff, rule, draft, shiftID, date)
    case "forbidden_day":
        return c.checkForbiddenDay(staff, rule, shiftID, date)
    case "required_together":
        return c.checkRequiredTogether(staff, rule, draft, shiftID, date)
    default:
        return nil // 未知规则类型，不做硬约束检查
    }
}

// checkMaxCount 检查最大次数约束
func (c *ConstraintChecker) checkMaxCount(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    if rule.Rule.MaxCount == nil {
        return nil
    }
    
    maxCount := *rule.Rule.MaxCount
    // 根据 TimeScope 确定统计范围
    startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)
    
    // 从草稿中统计该员工在时间范围内的排班次数
    currentCount := draft.CountStaffShiftInRange(staff.ID, shiftID, startDate, endDate)
    
    if currentCount >= maxCount {
        return &RuleViolation{
            RuleID:   rule.Rule.ID,
            RuleName: rule.Rule.Name,
            IsHard:   true,
            Message:  fmt.Sprintf("已达到最大次数限制(%d/%d)", currentCount, maxCount),
        }
    }
    return nil
}

// checkConsecutiveMax 检查连续天数约束
func (c *ConstraintChecker) checkConsecutiveMax(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    if rule.Rule.ConsecutiveMax == nil {
        return nil
    }
    
    maxConsecutive := *rule.Rule.ConsecutiveMax
    // 从当前日期往前回溯，计算连续排班天数
    consecutiveDays := 0
    checkDate := date.AddDate(0, 0, -1) // 从前一天开始检查
    
    for {
        if draft.HasStaffShiftOnDate(staff.ID, shiftID, checkDate) {
            consecutiveDays++
            checkDate = checkDate.AddDate(0, 0, -1)
        } else {
            break
        }
    }
    
    if consecutiveDays >= maxConsecutive {
        return &RuleViolation{
            RuleID:   rule.Rule.ID,
            RuleName: rule.Rule.Name,
            IsHard:   true,
            Message:  fmt.Sprintf("已连续排班%d天，超过限制(%d天)", consecutiveDays, maxConsecutive),
        }
    }
    return nil
}

// checkMinRestDays 检查最少休息天数约束
func (c *ConstraintChecker) checkMinRestDays(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    if rule.Rule.MinRestDays == nil {
        return nil
    }
    
    minRest := *rule.Rule.MinRestDays
    // 检查前 minRest 天内是否有关联班次的排班
    for i := 1; i <= minRest; i++ {
        checkDate := date.AddDate(0, 0, -i)
        // 检查关联班次（通过 Associations 获取）
        for _, assoc := range rule.Rule.Associations {
            if assoc.AssociationType == "shift" {
                if draft.HasStaffShiftOnDate(staff.ID, assoc.AssociationID, checkDate) {
                    return &RuleViolation{
                        RuleID:   rule.Rule.ID,
                        RuleName: rule.Rule.Name,
                        IsHard:   true,
                        Message:  fmt.Sprintf("距上次该班次排班仅%d天，未满足最少休息%d天", i, minRest),
                    }
                }
            }
        }
    }
    return nil
}

// checkExclusive 检查排他约束
func (c *ConstraintChecker) checkExclusive(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    // 获取排他的班次列表
    for _, assoc := range rule.Rule.Associations {
        if assoc.AssociationType == "shift" && assoc.AssociationID != shiftID {
            if draft.HasStaffShiftOnDate(staff.ID, assoc.AssociationID, date) {
                return &RuleViolation{
                    RuleID:   rule.Rule.ID,
                    RuleName: rule.Rule.Name,
                    IsHard:   true,
                    Message:  fmt.Sprintf("同日已排排他班次，与当前班次互斥"),
                }
            }
        }
    }
    return nil
}

// computeConstraintScore 计算约束"剩余空间"评分（0.0~1.0，越高越宽松）
func (c *ConstraintChecker) computeConstraintScore(
    staff *model.Staff,
    rule *ClassifiedRule,
    draft *model.ScheduleDraft,
    shiftID string,
    date time.Time,
) float64 {
    switch rule.Rule.RuleType {
    case "maxCount":
        if rule.Rule.MaxCount == nil { return 1.0 }
        max := float64(*rule.Rule.MaxCount)
        startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)
        current := float64(draft.CountStaffShiftInRange(staff.ID, shiftID, startDate, endDate))
        if max == 0 { return 0.0 }
        return (max - current) / max
    case "consecutiveMax":
        if rule.Rule.ConsecutiveMax == nil { return 1.0 }
        // 类似逻辑...
        return 1.0
    default:
        return 1.0
    }
}
```

#### 12.2.3 排班校验器

**文件**：`agents/rostering/internal/engine/schedule_validator.go`

```go
package engine

// ScheduleValidator 确定性排班校验器（替代 LLM-5）
type ScheduleValidator struct {
    logger logging.ILogger
}

// ValidationResult 校验结果
type ValidationResult struct {
    IsValid      bool                `json:"isValid"`
    Violations   []*ValidationItem   `json:"violations"`   // 硬约束违反
    Warnings     []*ValidationItem   `json:"warnings"`     // 软约束/偏好未满足
    Score        float64             `json:"score"`         // 排班质量评分 (0-100)
    Summary      string              `json:"summary"`       // 可读摘要
}

// ValidationItem 校验项
type ValidationItem struct {
    RuleID       string   `json:"ruleId"`
    RuleName     string   `json:"ruleName"`
    RuleType     string   `json:"ruleType"`
    Category     string   `json:"category"`     // constraint/preference
    StaffIDs     []string `json:"staffIds"`      // 涉及的人员
    Date         string   `json:"date"`          // 涉及的日期
    ShiftID      string   `json:"shiftId"`       // 涉及的班次
    Message      string   `json:"message"`       // 违反描述
    Severity     string   `json:"severity"`      // error/warning/info
    AutoFixable  bool     `json:"autoFixable"`   // 是否可自动修复
}

// Validate 校验排班结果
func (v *ScheduleValidator) Validate(
    schedule *model.ScheduleDraft,
    rules *MatchedRules,
    allDraft *model.ScheduleDraft,
) (*ValidationResult, error) {
    result := &ValidationResult{IsValid: true, Score: 100.0}
    
    // 1. 逐条检查约束型规则
    for _, rule := range rules.ConstraintRules {
        items := v.checkConstraintRule(rule, schedule, allDraft)
        for _, item := range items {
            if item.Severity == "error" {
                result.IsValid = false
                result.Violations = append(result.Violations, item)
                result.Score -= 10.0 // 每个硬约束违反扣10分
            } else {
                result.Warnings = append(result.Warnings, item)
                result.Score -= 2.0  // 每个警告扣2分
            }
        }
    }
    
    // 2. 检查偏好型规则（不影响 IsValid，只影响评分）
    for _, rule := range rules.PreferenceRules {
        items := v.checkPreferenceRule(rule, schedule, allDraft)
        for _, item := range items {
            result.Warnings = append(result.Warnings, item)
            result.Score -= 1.0 // 偏好未满足扣1分
        }
    }
    
    // 3. 检查人数约束（不超配不欠配）
    countItems := v.checkStaffCount(schedule)
    for _, item := range countItems {
        if item.Severity == "error" {
            result.IsValid = false
            result.Violations = append(result.Violations, item)
            result.Score -= 15.0
        }
    }
    
    // 4. 检查时间冲突（同一人同一时段多个班次）
    timeConflictItems := v.checkTimeConflicts(schedule, allDraft)
    for _, item := range timeConflictItems {
        result.IsValid = false
        result.Violations = append(result.Violations, item)
        result.Score -= 20.0
    }
    
    if result.Score < 0 {
        result.Score = 0
    }
    
    result.Summary = v.buildSummary(result)
    return result, nil
}
```

### 12.3 V4 排班执行流程（修正版）

```
V4 排班执行流程（修正版）：

1. 规则预组织阶段（录入时已完成，排班时加载）
   ├─ 加载规则分类、依赖关系、冲突关系
   ├─ 拓扑排序计算班次执行顺序
   └─ 输出：RuleOrganization（V4 方案原设计 ✅）

2. 确定性预计算阶段（代码执行，零 LLM 调用）
   ├─ 📌 候选人过滤（替代 LLM-1）
   │   ├─ 请假/休假人员过滤（日期精确匹配）
   │   ├─ 固定排班人员过滤
   │   └─ 输出：可用候选人列表 + 排除原因
   ├─ 📌 规则匹配（替代 LLM-2）
   │   ├─ 根据 Associations + Category 精确匹配
   │   ├─ 分离约束型/偏好型/依赖型
   │   └─ 输出：MatchedRules
   ├─ 📌 约束检查（替代 LLM-3）
   │   ├─ 逐人逐规则检查硬约束
   │   ├─ 计算"约束剩余空间"评分
   │   └─ 输出：EligibleCandidates + ExcludedCandidates + Reasons
   └─ 📌 偏好评分
       ├─ 为每个候选人计算偏好匹配度
       └─ 输出：PreferenceScores

3. LLM 排班决策阶段（仅 1 次 LLM 调用）
   ├─ 输入：结构化的 SchedulingContext（不是自然语言规则文本）
   │   ├─ 候选人列表（已标注偏好评分和约束剩余空间）
   │   ├─ 硬约束摘要（结构化格式）
   │   ├─ 软偏好摘要（结构化格式）
   │   └─ 排除人员及原因（透明化）
   ├─ LLM 职责：从合规候选人中选择最优人选
   └─ 输出：排班分配方案

4. 确定性校验阶段（代码执行，零 LLM 调用）
   ├─ 📌 硬约束违反检查（替代 LLM-5）
   ├─ 📌 偏好满足度评分
   ├─ 📌 时间冲突检查
   ├─ 📌 人数约束检查
   └─ 输出：ValidationResult（含评分和修复建议）

5. 自动修复/重试阶段
   ├─ 如有硬约束违反：代码层面自动替换（从候选人中选下一个）
   ├─ 仅复杂冲突：回到 LLM 重新决策（携带错误信息）
   └─ 最多 2 次 LLM 重试（V3 是 3 次，但 V4 前置校验更严格）
```

### 12.4 结构化 Prompt 设计（替代纯文本规则传递）

**核心改变：不再将规则作为自然语言文本传递给 LLM，而是传递结构化的决策摘要。**

```go
// buildStructuredPrompt 构建 V4 结构化 prompt
func buildStructuredPrompt(sCtx *SchedulingContext) string {
    var b strings.Builder
    
    // 1. 任务说明（固定模板）
    b.WriteString(fmt.Sprintf("## 排班任务\n"))
    b.WriteString(fmt.Sprintf("- 班次: %s (%s~%s)\n", sCtx.ShiftName, sCtx.ShiftStart, sCtx.ShiftEnd))
    b.WriteString(fmt.Sprintf("- 日期: %s\n", sCtx.Date))
    b.WriteString(fmt.Sprintf("- 需求人数: %d\n\n", sCtx.RequiredCount))
    
    // 2. 候选人列表（已通过所有硬约束，附结构化信息）
    b.WriteString("## 合格候选人\n")
    b.WriteString("| ID | 姓名 | 偏好评分 | 约束余量 | 本周已排次数 | 备注 |\n")
    b.WriteString("|-----|------|---------|---------|------------|------|\n")
    for _, c := range sCtx.LLMBrief.Candidates {
        b.WriteString(fmt.Sprintf("| %s | %s | %.1f | %.1f | %d | %s |\n",
            c.ShortID, c.Name, c.PreferenceScore, c.ConstraintMargin,
            c.WeeklyCount, c.Note))
    }
    
    // 3. 硬约束摘要（结构化，不是自然语言）
    b.WriteString("\n## 必须遵守的约束\n")
    for _, c := range sCtx.LLMBrief.HardConstraints {
        b.WriteString(fmt.Sprintf("- [%s] %s（类型: %s，限制值: %v）\n",
            c.RuleID, c.Description, c.Type, c.Limit))
    }
    
    // 4. 偏好参考
    b.WriteString("\n## 偏好参考（尽量满足）\n")
    for _, p := range sCtx.LLMBrief.SoftPreferences {
        b.WriteString(fmt.Sprintf("- %s（权重: %d）\n", p.Description, p.Weight))
    }
    
    // 5. 排除人员及原因（透明化）
    if len(sCtx.LLMBrief.ExcludedWithReasons) > 0 {
        b.WriteString("\n## 已排除人员（信息参考，无需处理）\n")
        for _, e := range sCtx.LLMBrief.ExcludedWithReasons {
            b.WriteString(fmt.Sprintf("- %s: %s\n", e.Name, e.Reason))
        }
    }
    
    return b.String()
}
```

### 12.5 LLM 调用次数对比

| 场景（20个班次 × 7天） | V3 | V4 | 减少比例 |
|------------------------|-----|-----|---------|
| 任务解析 | 2 | 0（代码化） | -100% |
| 人员过滤（LLM-1） | 140 | 0（代码化） | -100% |
| 规则过滤（LLM-2） | 140 | 0（代码化） | -100% |
| 冲突检测（LLM-3） | 140 | 0（代码化） | -100% |
| 排班决策（LLM-4） | 140 | 140 | 0% |
| 校验（LLM-5） | 140 | 0（代码化） | -100% |
| 冲突调整 | ≤20 | ≤20（仅复杂场景） | 0% |
| **合计** | **~720+** | **~160** | **-78%** |

---

## 13. 补充设计：规则结构化传递协议

### 13.1 问题回顾

V3 中规则传递给 LLM 的方式：
```
1. 夜班连续限制: 夜班最多连续2天，之后必须休息至少1天
2. 排他规则: 排了A班就不能排B班
```

问题：LLM 需要从自然语言中"理解"数值约束，理解结果不稳定。

### 13.2 V4 规则传递格式

**规则不再作为自然语言传给 LLM，而是通过确定性引擎转化为"决策摘要"传递。**

LLM 看到的不是规则列表，而是：

```
## 合格候选人（已通过所有约束检查）

| ID | 姓名 | 推荐度 | 本周已排 | 连续天数 | 备注 |
|-----|------|-------|---------|---------|------|
| S1 | 张三 | 0.9   | 2/5     | 0       | 偏好周末休息 |
| S2 | 李四 | 0.7   | 3/5     | 1       | 接近频次上限 |
| S3 | 王五 | 0.8   | 1/5     | 0       | - |

## 约束边界（系统已验证，请在此范围内选择）
- 每人本周最多排5次（系统已过滤超限人员）
- 连续排班不超过2天（系统已过滤超限人员）
- 不可同时排 A班 和 B班（系统已过滤冲突人员）

需求人数: 2人
请从合格候选人中选择2人，优先选择推荐度高的。
```

**核心区别**：
- V3：传规则文本 → LLM 理解规则 → LLM 检查每个人 → LLM 选人（4步，每步都可能出错）
- V4：代码检查规则 → 传合格人员表 → LLM 选人（1步 LLM，只做选择题）

---

## 14. 补充设计：规则解析结果验证机制

### 14.1 问题回顾（V4 方案缺陷三：稳定性悖论）

LLM 语义化解析规则时可能解析错误，错误被永久存储到数据库。

### 14.2 三层验证机制

```
用户输入自然语言规则
       ↓
  LLM 语义化解析
       ↓
 ┌─────────────────────┐
 │ 第1层：结构化验证     │ ← 代码自动检查
 │ - 必填字段完整性      │
 │ - 数值参数合理性      │
 │ - 枚举值有效性        │
 │ - 关联对象存在性      │
 └─────────────────────┘
       ↓
 ┌─────────────────────┐
 │ 第2层：语义一致性验证  │ ← 代码 + 第二次 LLM 交叉验证
 │ - 原始描述 vs 解析结果 │
 │ - 回译验证（将结构化   │
 │   规则重新生成自然语言  │
 │   描述，让用户确认）   │
 └─────────────────────┘
       ↓
 ┌─────────────────────┐
 │ 第3层：模拟验证       │ ← 代码执行
 │ - 用测试数据跑一次    │
 │   确定性规则引擎      │
 │ - 验证规则是否产生    │
 │   预期的约束效果      │
 │ - 检查与现有规则的    │
 │   冲突/矛盾           │
 └─────────────────────┘
       ↓
  用户确认保存
```

#### 14.2.1 结构化验证器

```go
// StructuralValidator 结构化验证器（第1层）
type StructuralValidator struct{}

func (v *StructuralValidator) Validate(parsed *ParsedRule) []ValidationError {
    var errors []ValidationError
    
    // 1. 必填字段
    if parsed.Name == "" {
        errors = append(errors, ValidationError{Field: "name", Message: "规则名称不能为空"})
    }
    if parsed.Category == "" || !isValidCategory(parsed.Category) {
        errors = append(errors, ValidationError{Field: "category", Message: "无效的规则分类"})
    }
    
    // 2. 数值合理性
    if parsed.RuleType == "maxCount" && (parsed.MaxCount == nil || *parsed.MaxCount <= 0) {
        errors = append(errors, ValidationError{Field: "maxCount", Message: "maxCount类型规则必须指定正整数"})
    }
    if parsed.RuleType == "consecutiveMax" && (parsed.ConsecutiveMax == nil || *parsed.ConsecutiveMax <= 0) {
        errors = append(errors, ValidationError{Field: "consecutiveMax", Message: "consecutiveMax类型规则必须指定正整数"})
    }
    
    // 3. 关联对象
    if parsed.ApplyScope == "specific" && len(parsed.Associations) == 0 {
        errors = append(errors, ValidationError{Field: "associations", Message: "特定范围规则必须指定关联对象"})
    }
    
    return errors
}
```

#### 14.2.2 回译验证（解决稳定性悖论的关键）

```go
// BackTranslationValidator 回译验证器（第2层核心）
// 将解析后的结构化规则重新生成自然语言，让用户对比确认
type BackTranslationValidator struct{}

func (v *BackTranslationValidator) BackTranslate(parsed *ParsedRule) string {
    var desc strings.Builder
    
    switch parsed.RuleType {
    case "maxCount":
        scope := v.translateTimeScope(parsed.TimeScope)
        target := v.translateAssociations(parsed.Associations)
        desc.WriteString(fmt.Sprintf("%s%s最多排%d次", scope, target, *parsed.MaxCount))
        
    case "consecutiveMax":
        target := v.translateAssociations(parsed.Associations)
        desc.WriteString(fmt.Sprintf("%s连续排班不超过%d天", target, *parsed.ConsecutiveMax))
        if parsed.MinRestDays != nil {
            desc.WriteString(fmt.Sprintf("，之后至少休息%d天", *parsed.MinRestDays))
        }
        
    case "exclusive":
        shifts := v.getShiftNames(parsed.Associations)
        desc.WriteString(fmt.Sprintf("%s 互斥，同一天不能同时排", strings.Join(shifts, "和")))
        
    case "minRestDays":
        target := v.translateAssociations(parsed.Associations)
        desc.WriteString(fmt.Sprintf("%s排班后至少休息%d天", target, *parsed.MinRestDays))
    }
    
    return desc.String()
}
```

**前端展示效果**：
```
原始描述: "夜班最多连续2天，之后必须休息至少1天"
     ↓ LLM 解析
解析结果: 
  规则1: consecutiveMax=2, 关联班次=夜班
  规则2: minRestDays=1, 关联班次=夜班
     ↓ 回译
回译描述: "夜班连续排班不超过2天，之后至少休息1天"
     ↓
用户确认: ✅ 含义一致 / ❌ 需要调整
```

---

## 15. 补充设计：规则 Association 方向性改造

### 15.1 问题描述

V3 中 `RuleAssociation` 只记录关联关系，不区分方向：

```go
// 当前结构（V3）
type RuleAssociation struct {
    AssociationType string  // "shift" / "employee" / "group"
    AssociationID   string  // 班次/员工/分组 ID
}
```

问题：规则"下夜班人员必须来自前一日的上半夜班"关联了两个班次（下夜班、上半夜班），但 `Associations` 无法区分"被约束方"（下夜班）和"数据来源"（上半夜班）。导致 V3 需要 LLM-2 靠"看主语"来推断。

### 15.2 改造方案

```go
// V4 改造后的结构
type RuleAssociation struct {
    AssociationType string  `json:"associationType"`  // "shift" / "employee" / "group"
    AssociationID   string  `json:"associationId"`     // 班次/员工/分组 ID
    Role            string  `json:"role"`              // 🆕 "target" / "source" / "reference"
    // target: 被约束的对象（规则作用目标）
    // source: 数据来源（依赖型规则的前置对象）
    // reference: 引用对象（规则中提到但不直接约束的对象）
}
```

```sql
-- 数据库扩展
ALTER TABLE rule_associations 
ADD COLUMN role VARCHAR(32) DEFAULT 'target' 
COMMENT '关联角色: target(约束目标)/source(数据来源)/reference(引用对象)';
```

**示例**：

规则：*"下夜班人员必须来自前一日的上半夜班"*

```json
{
  "name": "下夜班来源限制",
  "ruleType": "source_dependency",
  "category": "dependency",
  "associations": [
    {"associationType": "shift", "associationId": "shift_下夜班", "role": "target"},
    {"associationType": "shift", "associationId": "shift_上半夜班", "role": "source"}
  ]
}
```

**好处**：
- 规则匹配器（`RuleMatcher`）可以精确判断"当前排的是哪个班次，这条规则是否约束它"
- 依赖分析器可以自动构建 DAG：`source → target`
- 完全不需要 LLM-2 的"看主语"推断

---

## 16. 修订后的实施计划

### 阶段1：确定性规则引擎（2周）⭐ 最高优先级
- [ ] 实现 `RuleEngine` 入口
- [ ] 实现 `CandidateFilter`（替代 LLM-1）
- [ ] 实现 `RuleMatcher`（替代 LLM-2）
- [ ] 实现 `ConstraintChecker`（替代 LLM-3）
- [ ] 实现 `ScheduleValidator`（替代 LLM-5）
- [ ] 实现 `PreferenceScorer`
- [ ] 单元测试覆盖所有规则类型

### 阶段2：数据模型改造（1周）
- [ ] `RuleAssociation` 添加 `Role` 字段
- [ ] 扩展规则表添加 `category`/`sub_category` 字段
- [ ] 创建 `rule_dependencies` 表
- [ ] 创建 `rule_conflicts` 表
- [ ] 创建 `shift_dependencies` 表
- [ ] 数据迁移脚本（现有规则补充分类和角色）

### 阶段3：规则解析与验证（2周）
- [ ] 实现 LLM 语义化规则解析服务
- [ ] 实现三层验证机制（结构化验证 + 回译验证 + 模拟验证）
- [ ] 实现规则冲突检测
- [ ] 前端规则录入页面改造

### 阶段4：V4 排班工作流（2周）
- [ ] 创建 V4 工作流框架
- [ ] 实现规则组织器（基于原设计 + 修正）
- [ ] 实现结构化 Prompt 构建
- [ ] 集成确定性规则引擎到排班流程
- [ ] LLM 调用减少为仅排班决策 + 异常处理

### 阶段5：集成测试与灰度（2周）
- [ ] V3/V4 双模式切换支持
- [ ] 对比测试（同一组规则、人员，V3 vs V4 输出质量）
- [ ] 性能测试（LLM 调用次数、响应时间）
- [ ] 灰度上线

**总周期：9周**（原方案 10 周，因为阶段 1 的确定性引擎会减少后续集成的复杂度）

---

## 17. V3 vs V4 预期效果对比

| 指标 | V3 现状 | V4 目标 | 改善方式 |
|------|---------|---------|---------|
| **单次排班 LLM 调用数** | ~720次（20班次×7天） | ~160次 | 代码化 LLM-1/2/3/5 |
| **硬约束违反率** | ~15%（LLM 理解偏差） | <1%（代码精确检查） | 确定性规则引擎 |
| **规则理解一致性** | 不稳定（LLM 每次理解不同） | 100%一致 | 结构化参数 + 代码执行 |
| **幻觉出错率** | ~10%（虚构频次/来源限制） | <1%（LLM 只做选择题） | 缩小 LLM 职责范围 |
| **排班质量评分** | 无量化 | 0-100 分量化评分 | 确定性校验器 |
| **排班耗时** | 长（多次 LLM 调用） | 减少 60-70% | 代码化预处理 |
| **规则录入体验** | 手动填写多字段 | 自然语言 + 自动解析 | LLM 解析 + 三层验证 |
| **规则维护成本** | 高（无分类无关系） | 低（分类 + 依赖图 + 可视化） | 规则组织体系 |
