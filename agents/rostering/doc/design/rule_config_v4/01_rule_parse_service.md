# 01. LLM 规则解析服务设计

> **部署位置**: `services/management-service/internal/rule_parser/`  
> **依赖**: `pkg/ai` (AI Provider)  
> **被依赖**: `scheduling_rule_handler.go` (新增 /rules/parse 端点)

## 1. 设计概述

规则解析服务是 V4 规则配置管理的核心，负责将用户的**自然语言规则描述**一次性解析为 V4 **结构化规则字段**。

### 1.1 核心原则

- **解析一次，执行多次**: LLM 只在规则录入时调用一次，解析结果保存为结构化数据
- **解析 ≠ 保存**: 解析结果返回给前端展示，人工确认后才调用创建接口保存
- **回译验证**: LLM 输出结构化结果的同时，输出自然语言回译，供用户对比确认
- **批量支持**: 支持一段文字描述多条规则，LLM 拆解后返回多条解析结果

### 1.2 调用链路

```
前端 RuleParseDialog
  → POST /v1/rules/parse
    → management-service RuleParseHandler
      → RuleParserService.Parse()
        → AI Provider (LLM)
        → Validator (三层验证)
      → Response: ParseResult[]
  → 前端展示解析结果
  → 用户审核/编辑
  → POST /v1/scheduling-rules (创建)
```

---

## 2. 类型定义

### 2.1 请求/响应类型

**文件**: `services/management-service/internal/rule_parser/types.go`

```go
package rule_parser

// ============================================================
// 请求类型
// ============================================================

// ParseRequest 单条规则解析请求
type ParseRequest struct {
    OrgID      string   `json:"orgId" binding:"required"`
    RuleText   string   `json:"ruleText" binding:"required"` // 自然语言规则描述
    ShiftNames []string `json:"shiftNames"`                  // 当前组织的班次名称列表
    GroupNames []string `json:"groupNames"`                  // 当前组织的分组名称列表
}

// BatchParseRequest 批量解析请求（一段文字含多条规则）
type BatchParseRequest struct {
    OrgID      string   `json:"orgId" binding:"required"`
    RuleTexts  []string `json:"ruleTexts" binding:"required"` // 多段规则描述
    ShiftNames []string `json:"shiftNames"`
    GroupNames []string `json:"groupNames"`
}

// ============================================================
// 响应类型
// ============================================================

// ParseResponse 解析响应
type ParseResponse struct {
    Success bool           `json:"success"`
    Results []*ParseResult `json:"results"` // 可能一段文字解析出多条规则
    Error   string         `json:"error,omitempty"`
}

// ParseResult 单条规则解析结果
type ParseResult struct {
    // 解析状态
    Confidence float64 `json:"confidence"` // 解析置信度 0.0-1.0

    // === V4 结构化字段 ===
    Name        string `json:"name"`        // 建议的规则名称
    Description string `json:"description"` // 规则描述
    RuleType    string `json:"ruleType"`    // exclusive/combinable/required_together/periodic/maxCount/forbidden_day/preferred
    Category    string `json:"category"`    // constraint/preference/dependency
    SubCategory string `json:"subCategory"` // forbid/limit/must/prefer/suggest/combinable/source/resource/order
    ApplyScope  string `json:"applyScope"`  // global/specific
    TimeScope   string `json:"timeScope"`   // same_day/same_week/same_month/custom
    Priority    int    `json:"priority"`    // 建议优先级 1-10

    // 量化参数（根据规则类型填充）
    MaxCount       *int `json:"maxCount,omitempty"`
    ConsecutiveMax *int `json:"consecutiveMax,omitempty"`
    IntervalDays   *int `json:"intervalDays,omitempty"`
    MinRestDays    *int `json:"minRestDays,omitempty"`

    // 规则语义文本（保留原始意图）
    RuleData string `json:"ruleData"`

    // 关联对象
    Associations []*ParsedAssociation `json:"associations,omitempty"`

    // 回译文本（LLM 将结构化结果翻译回自然语言，供用户对比确认）
    BackTranslation string `json:"backTranslation"`

    // 验证结果
    Validation *ValidationResult `json:"validation"`

    // 原始输入（方便前端展示对比）
    OriginalText string `json:"originalText"`
}

// ParsedAssociation 解析出的关联对象
type ParsedAssociation struct {
    Type   string `json:"type"`   // shift / group / employee
    Name   string `json:"name"`   // LLM 识别出的名称
    ID     string `json:"id"`     // 后端匹配的真实 ID（模糊匹配）
    Role   string `json:"role"`   // target / source / reference
    Matched bool  `json:"matched"` // 是否成功匹配到真实对象
}

// ValidationResult 三层验证结果
type ValidationResult struct {
    IsValid     bool              `json:"isValid"`
    Errors      []*ValidationItem `json:"errors,omitempty"`
    Warnings    []*ValidationItem `json:"warnings,omitempty"`
    Suggestions []*ValidationItem `json:"suggestions,omitempty"`
}

// ValidationItem 单条验证项
type ValidationItem struct {
    Field   string `json:"field"`   // 相关字段名
    Code    string `json:"code"`    // 验证码
    Message string `json:"message"` // 说明
}
```

---

## 3. LLM 解析 Prompt 设计

### 3.1 System Prompt

```go
const SystemPrompt = `你是一个排班规则解析器。你的任务是将用户的自然语言排班规则描述，
解析为结构化的JSON格式。

## 可用的字段值

### ruleType（规则类型）
- exclusive: 互斥规则（两个班次/人员不能同时排）
- combinable: 可组合规则（可以同时排）
- required_together: 必须同时排
- periodic: 周期性规则（隔周/隔天等）
- maxCount: 数量限制（最多X次）
- forbidden_day: 禁止日期/时段
- preferred: 偏好

### category（规则分类）
- constraint: 约束型（必须遵守的硬规则）
- preference: 偏好型（尽量满足的软规则）
- dependency: 依赖型（定义执行顺序或来源关系）

### subCategory（规则子分类）
constraint 下:
  - forbid: 禁止型（绝对不允许）
  - limit: 限制型（有数量/频率上限）
  - must: 必须型（必须满足）
preference 下:
  - prefer: 优先型（优先考虑）
  - suggest: 建议型（可以不满足）
  - combinable: 可合并型
dependency 下:
  - source: 来源依赖（人员必须来自某个班次）
  - resource: 资源预留（需要预留人员给后续班次）
  - order: 顺序依赖（必须先排某个班次）

### applyScope（应用范围）
- global: 全局规则
- specific: 特定对象（需要关联具体班次/分组/员工）

### timeScope（时间范围）
- same_day: 同一天内
- same_week: 同一周内
- same_month: 同一月内
- custom: 自定义

### 关联对象角色 (role)
- target: 被约束的对象（规则作用目标）
- source: 数据来源（依赖型规则中的来源）
- reference: 引用对象（如排他规则中的"另一方"）

## 当前组织可用的班次名称
{shiftNames}

## 当前组织可用的分组名称
{groupNames}

## 输出格式

输出一个JSON数组，每个元素代表一条解析出的规则：

[
  {
    "name": "规则名称",
    "description": "规则描述",
    "ruleType": "maxCount",
    "category": "constraint",
    "subCategory": "limit",
    "applyScope": "specific",
    "timeScope": "same_week",
    "priority": 7,
    "maxCount": 3,
    "consecutiveMax": null,
    "intervalDays": null,
    "minRestDays": null,
    "ruleData": "保留原始语义描述",
    "associations": [
      {"type": "shift", "name": "早班", "role": "target"}
    ],
    "backTranslation": "约束规则：每周内，每位员工在【早班】上最多排3次"
  }
]

## 重要注意事项

1. 一段描述可能包含多条规则，请全部解析出来
2. 如果描述模糊，设置合理的默认值并在 backTranslation 中明确说明
3. backTranslation 必须用清晰的结构化自然语言描述，方便用户对比确认
4. 量化参数只填写与 ruleType 相关的字段，其他设为 null
5. associations 中的 name 必须从可用班次/分组列表中匹配
6. priority 建议值: constraint 类 7-9, preference 类 3-5, dependency 类 5-7
`
```

### 3.2 User Prompt 模板

```go
const UserPromptTemplate = `请解析以下排班规则描述:

%s

请输出JSON数组格式的解析结果。`
```

### 3.3 解析示例

**输入**:
```
早班每人每周最多排3次，夜班和早班不能连续排，尽量安排高年资护士上夜班
```

**LLM 输出**:
```json
[
  {
    "name": "早班周排班次数限制",
    "description": "早班每人每周最多排3次",
    "ruleType": "maxCount",
    "category": "constraint",
    "subCategory": "limit",
    "applyScope": "specific",
    "timeScope": "same_week",
    "priority": 7,
    "maxCount": 3,
    "ruleData": "早班每人每周最多排3次",
    "associations": [
      {"type": "shift", "name": "早班", "role": "target"}
    ],
    "backTranslation": "【约束-限制】每周内，每位员工在「早班」上最多排3次"
  },
  {
    "name": "夜班早班互斥",
    "description": "夜班后不能紧接早班",
    "ruleType": "exclusive",
    "category": "constraint",
    "subCategory": "forbid",
    "applyScope": "specific",
    "timeScope": "same_day",
    "priority": 9,
    "ruleData": "夜班和早班不能连续排",
    "associations": [
      {"type": "shift", "name": "夜班", "role": "target"},
      {"type": "shift", "name": "早班", "role": "reference"}
    ],
    "backTranslation": "【约束-禁止】同一天内，排了「夜班」的员工不能再排「早班」（反之亦然）"
  },
  {
    "name": "高年资护士夜班偏好",
    "description": "优先安排高年资护士上夜班",
    "ruleType": "preferred",
    "category": "preference",
    "subCategory": "prefer",
    "applyScope": "specific",
    "timeScope": "same_day",
    "priority": 4,
    "ruleData": "尽量安排高年资护士上夜班",
    "associations": [
      {"type": "shift", "name": "夜班", "role": "target"},
      {"type": "group", "name": "高年资护士", "role": "source"}
    ],
    "backTranslation": "【偏好-优先】排「夜班」时，优先从「高年资护士」分组中选人"
  }
]
```

---

## 4. 解析服务实现

### 4.1 服务接口

**文件**: `services/management-service/internal/rule_parser/parser.go`

```go
package rule_parser

import (
    "context"
)

// IRuleParserService 规则解析服务接口
type IRuleParserService interface {
    // Parse 解析单段自然语言规则描述
    // 一段描述可能解析出多条规则
    Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)

    // BatchParse 批量解析多段规则描述
    BatchParse(ctx context.Context, req *BatchParseRequest) (*ParseResponse, error)
}
```

### 4.2 服务实现

```go
package rule_parser

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "jusha/gantt/pkg/ai"
    "jusha/gantt/pkg/logging"
)

// RuleParserService 规则解析服务实现
type RuleParserService struct {
    aiProvider   ai.AIProvider          // LLM 调用
    validator    *RuleValidator         // 三层验证器
    nameMatcher  *NameMatcher           // 名称模糊匹配器
    logger       *logging.Logger
}

// NewRuleParserService 创建规则解析服务
func NewRuleParserService(aiProvider ai.AIProvider, logger *logging.Logger) *RuleParserService {
    return &RuleParserService{
        aiProvider:  aiProvider,
        validator:   NewRuleValidator(),
        nameMatcher: NewNameMatcher(),
        logger:      logger,
    }
}

// Parse 解析规则
func (s *RuleParserService) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
    // 1. 构建 Prompt
    prompt := s.buildPrompt(req)

    // 2. 调用 LLM
    llmResponse, err := s.aiProvider.Chat(ctx, ai.ChatRequest{
        Messages: []ai.Message{
            {Role: "system", Content: s.buildSystemPrompt(req.ShiftNames, req.GroupNames)},
            {Role: "user", Content: prompt},
        },
        Temperature: 0.1, // 低温度，追求确定性
    })
    if err != nil {
        return &ParseResponse{
            Success: false,
            Error:   fmt.Sprintf("LLM 调用失败: %v", err),
        }, nil
    }

    // 3. 解析 LLM 输出 JSON
    results, err := s.parseLLMOutput(llmResponse.Content)
    if err != nil {
        return &ParseResponse{
            Success: false,
            Error:   fmt.Sprintf("LLM 输出解析失败: %v", err),
        }, nil
    }

    // 4. 后处理: 名称匹配 + ID 填充
    for _, result := range results {
        result.OriginalText = req.RuleText
        s.matchAssociations(ctx, req.OrgID, result, req.ShiftNames, req.GroupNames)
    }

    // 5. 三层验证
    for _, result := range results {
        result.Validation = s.validator.Validate(result)
    }

    return &ParseResponse{
        Success: true,
        Results: results,
    }, nil
}

// BatchParse 批量解析
func (s *RuleParserService) BatchParse(ctx context.Context, req *BatchParseRequest) (*ParseResponse, error) {
    var allResults []*ParseResult
    
    for _, text := range req.RuleTexts {
        resp, err := s.Parse(ctx, &ParseRequest{
            OrgID:      req.OrgID,
            RuleText:   text,
            ShiftNames: req.ShiftNames,
            GroupNames: req.GroupNames,
        })
        if err != nil {
            return nil, err
        }
        if resp.Success {
            allResults = append(allResults, resp.Results...)
        }
    }

    return &ParseResponse{
        Success: true,
        Results: allResults,
    }, nil
}

// buildSystemPrompt 构建系统提示词
func (s *RuleParserService) buildSystemPrompt(shiftNames, groupNames []string) string {
    prompt := SystemPrompt
    prompt = strings.ReplaceAll(prompt, "{shiftNames}", strings.Join(shiftNames, "、"))
    prompt = strings.ReplaceAll(prompt, "{groupNames}", strings.Join(groupNames, "、"))
    return prompt
}

// buildPrompt 构建用户提示词
func (s *RuleParserService) buildPrompt(req *ParseRequest) string {
    return fmt.Sprintf(UserPromptTemplate, req.RuleText)
}

// parseLLMOutput 解析 LLM JSON 输出
func (s *RuleParserService) parseLLMOutput(content string) ([]*ParseResult, error) {
    // 去除可能的 markdown 代码块标记
    content = strings.TrimSpace(content)
    content = strings.TrimPrefix(content, "```json")
    content = strings.TrimPrefix(content, "```")
    content = strings.TrimSuffix(content, "```")
    content = strings.TrimSpace(content)

    var results []*ParseResult
    if err := json.Unmarshal([]byte(content), &results); err != nil {
        // 尝试单对象解析（LLM 可能返回单个对象而非数组）
        var single ParseResult
        if err2 := json.Unmarshal([]byte(content), &single); err2 != nil {
            return nil, fmt.Errorf("JSON 解析失败: %v", err)
        }
        results = []*ParseResult{&single}
    }
    return results, nil
}

// matchAssociations 匹配关联对象名称到真实 ID
func (s *RuleParserService) matchAssociations(
    ctx context.Context, orgID string,
    result *ParseResult,
    shiftNames, groupNames []string,
) {
    for _, assoc := range result.Associations {
        switch assoc.Type {
        case "shift":
            // 模糊匹配班次名称
            matched, id := s.nameMatcher.MatchShift(ctx, orgID, assoc.Name, shiftNames)
            assoc.Matched = matched
            assoc.ID = id
        case "group":
            // 模糊匹配分组名称
            matched, id := s.nameMatcher.MatchGroup(ctx, orgID, assoc.Name, groupNames)
            assoc.Matched = matched
            assoc.ID = id
        case "employee":
            // 员工名称匹配（需查数据库）
            matched, id := s.nameMatcher.MatchEmployee(ctx, orgID, assoc.Name)
            assoc.Matched = matched
            assoc.ID = id
        }
    }
}
```

---

## 5. 三层验证器

### 5.1 验证器实现

**文件**: `services/management-service/internal/rule_parser/validator.go`

```go
package rule_parser

// RuleValidator 规则验证器
type RuleValidator struct{}

func NewRuleValidator() *RuleValidator {
    return &RuleValidator{}
}

// Validate 执行三层验证
func (v *RuleValidator) Validate(result *ParseResult) *ValidationResult {
    vr := &ValidationResult{IsValid: true}

    // === 第一层: 结构完整性验证 ===
    v.validateStructure(result, vr)

    // === 第二层: 语义一致性验证 ===
    v.validateSemantics(result, vr)

    // === 第三层: 业务合理性验证 ===
    v.validateBusiness(result, vr)

    return vr
}

// validateStructure 结构完整性验证
func (v *RuleValidator) validateStructure(result *ParseResult, vr *ValidationResult) {
    // 必填字段检查
    if result.Name == "" {
        vr.addError("name", "MISSING_NAME", "规则名称不能为空")
    }
    if result.RuleType == "" {
        vr.addError("ruleType", "MISSING_RULE_TYPE", "规则类型不能为空")
    }
    if result.Category == "" {
        vr.addError("category", "MISSING_CATEGORY", "规则分类不能为空")
    }
    if result.SubCategory == "" {
        vr.addError("subCategory", "MISSING_SUB_CATEGORY", "规则子分类不能为空")
    }
    if result.TimeScope == "" {
        vr.addError("timeScope", "MISSING_TIME_SCOPE", "时间范围不能为空")
    }

    // 枚举值合法性检查
    validRuleTypes := map[string]bool{
        "exclusive": true, "combinable": true, "required_together": true,
        "periodic": true, "maxCount": true, "forbidden_day": true, "preferred": true,
    }
    if result.RuleType != "" && !validRuleTypes[result.RuleType] {
        vr.addError("ruleType", "INVALID_RULE_TYPE", "无效的规则类型: "+result.RuleType)
    }

    validCategories := map[string]bool{
        "constraint": true, "preference": true, "dependency": true,
    }
    if result.Category != "" && !validCategories[result.Category] {
        vr.addError("category", "INVALID_CATEGORY", "无效的规则分类: "+result.Category)
    }

    // 关联对象 Role 检查
    validRoles := map[string]bool{"target": true, "source": true, "reference": true}
    for i, assoc := range result.Associations {
        if assoc.Role == "" {
            vr.addWarning("associations", "MISSING_ROLE",
                formatMsg("第%d个关联对象缺少角色(role)，将默认为target", i+1))
            assoc.Role = "target"
        }
        if !validRoles[assoc.Role] {
            vr.addError("associations", "INVALID_ROLE",
                formatMsg("第%d个关联对象角色无效: %s", i+1, assoc.Role))
        }
    }
}

// validateSemantics 语义一致性验证
func (v *RuleValidator) validateSemantics(result *ParseResult, vr *ValidationResult) {
    // Category 与 SubCategory 匹配性检查
    validSubCategories := map[string][]string{
        "constraint": {"forbid", "limit", "must"},
        "preference": {"prefer", "suggest", "combinable"},
        "dependency": {"source", "resource", "order"},
    }
    if subs, ok := validSubCategories[result.Category]; ok {
        found := false
        for _, s := range subs {
            if s == result.SubCategory {
                found = true
                break
            }
        }
        if !found {
            vr.addError("subCategory", "CATEGORY_SUBCATEGORY_MISMATCH",
                formatMsg("子分类 '%s' 与分类 '%s' 不匹配", result.SubCategory, result.Category))
        }
    }

    // RuleType 与量化参数匹配检查
    switch result.RuleType {
    case "maxCount":
        if result.MaxCount == nil || *result.MaxCount <= 0 {
            vr.addError("maxCount", "MISSING_MAX_COUNT", "maxCount 规则必须设置 maxCount 参数且大于0")
        }
    case "exclusive":
        if len(result.Associations) < 2 {
            vr.addWarning("associations", "EXCLUSIVE_NEEDS_TWO",
                "互斥规则通常需要至少2个关联对象(target和reference)")
        }
    case "periodic":
        if result.IntervalDays == nil || *result.IntervalDays <= 0 {
            vr.addWarning("intervalDays", "MISSING_INTERVAL",
                "周期性规则建议设置 intervalDays 参数")
        }
    }

    // ApplyScope 与 Associations 匹配检查
    if result.ApplyScope == "specific" && len(result.Associations) == 0 {
        vr.addWarning("associations", "SPECIFIC_NO_ASSOCIATIONS",
            "应用范围为 specific 但没有关联对象，可能需要添加关联")
    }
    if result.ApplyScope == "global" && len(result.Associations) > 0 {
        vr.addWarning("applyScope", "GLOBAL_WITH_ASSOCIATIONS",
            "应用范围为 global 但有关联对象，请确认是否应为 specific")
    }

    // 优先级范围检查
    if result.Priority < 1 || result.Priority > 10 {
        vr.addWarning("priority", "PRIORITY_OUT_OF_RANGE", "优先级应在1-10之间")
        if result.Priority < 1 {
            result.Priority = 1
        }
        if result.Priority > 10 {
            result.Priority = 10
        }
    }
}

// validateBusiness 业务合理性验证
func (v *RuleValidator) validateBusiness(result *ParseResult, vr *ValidationResult) {
    // 关联对象匹配情况检查
    for _, assoc := range result.Associations {
        if !assoc.Matched {
            vr.addWarning("associations", "UNMATCHED_ASSOCIATION",
                formatMsg("关联对象 '%s'(%s) 未能匹配到系统中的真实对象，请手动确认", assoc.Name, assoc.Type))
        }
    }

    // 约束型规则建议高优先级
    if result.Category == "constraint" && result.Priority < 5 {
        vr.addSuggestion("priority", "LOW_PRIORITY_CONSTRAINT",
            "约束型规则建议设置较高优先级(>=7)，当前为"+formatMsg("%d", result.Priority))
    }

    // 偏好型规则建议低优先级
    if result.Category == "preference" && result.Priority > 7 {
        vr.addSuggestion("priority", "HIGH_PRIORITY_PREFERENCE",
            "偏好型规则建议设置较低优先级(<=5)，当前为"+formatMsg("%d", result.Priority))
    }

    // 依赖型规则必须有 source 角色关联
    if result.Category == "dependency" {
        hasSource := false
        for _, assoc := range result.Associations {
            if assoc.Role == "source" {
                hasSource = true
                break
            }
        }
        if !hasSource {
            vr.addWarning("associations", "DEPENDENCY_NO_SOURCE",
                "依赖型规则通常需要一个 role=source 的关联对象")
        }
    }
}

// ============================================================
// ValidationResult 辅助方法
// ============================================================

func (vr *ValidationResult) addError(field, code, message string) {
    vr.IsValid = false
    vr.Errors = append(vr.Errors, &ValidationItem{
        Field: field, Code: code, Message: message,
    })
}

func (vr *ValidationResult) addWarning(field, code, message string) {
    vr.Warnings = append(vr.Warnings, &ValidationItem{
        Field: field, Code: code, Message: message,
    })
}

func (vr *ValidationResult) addSuggestion(field, code, message string) {
    vr.Suggestions = append(vr.Suggestions, &ValidationItem{
        Field: field, Code: code, Message: message,
    })
}

func formatMsg(format string, args ...interface{}) string {
    return fmt.Sprintf(format, args...)
}
```

---

## 6. 名称模糊匹配器

### 6.1 实现

**文件**: `services/management-service/internal/rule_parser/name_matcher.go`

```go
package rule_parser

import (
    "context"
    "strings"
    "unicode/utf8"
)

// NameMatcher 名称模糊匹配器
type NameMatcher struct {
    // 可注入 ShiftService, GroupService, EmployeeService 等
}

func NewNameMatcher() *NameMatcher {
    return &NameMatcher{}
}

// MatchShift 模糊匹配班次名称
func (m *NameMatcher) MatchShift(ctx context.Context, orgID, name string, availableNames []string) (bool, string) {
    // 精确匹配
    for _, n := range availableNames {
        if n == name {
            return true, "" // ID 需要通过 ShiftService 查询
        }
    }
    // 包含匹配
    for _, n := range availableNames {
        if strings.Contains(n, name) || strings.Contains(name, n) {
            return true, ""
        }
    }
    // 编辑距离匹配（容忍1个字符差异）
    for _, n := range availableNames {
        if levenshteinDistance(n, name) <= 1 {
            return true, ""
        }
    }
    return false, ""
}

// MatchGroup 模糊匹配分组名称
func (m *NameMatcher) MatchGroup(ctx context.Context, orgID, name string, availableNames []string) (bool, string) {
    // 与 MatchShift 类似的匹配逻辑
    for _, n := range availableNames {
        if n == name || strings.Contains(n, name) || strings.Contains(name, n) {
            return true, ""
        }
    }
    return false, ""
}

// MatchEmployee 匹配员工名称（需要查数据库）
func (m *NameMatcher) MatchEmployee(ctx context.Context, orgID, name string) (bool, string) {
    // TODO: 通过 EmployeeService 查询
    return false, ""
}

// levenshteinDistance 计算编辑距离
func levenshteinDistance(a, b string) int {
    la := utf8.RuneCountInString(a)
    lb := utf8.RuneCountInString(b)
    ra := []rune(a)
    rb := []rune(b)

    d := make([][]int, la+1)
    for i := range d {
        d[i] = make([]int, lb+1)
        d[i][0] = i
    }
    for j := 0; j <= lb; j++ {
        d[0][j] = j
    }

    for i := 1; i <= la; i++ {
        for j := 1; j <= lb; j++ {
            cost := 0
            if ra[i-1] != rb[j-1] {
                cost = 1
            }
            d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
        }
    }
    return d[la][lb]
}

func min3(a, b, c int) int {
    if a < b {
        if a < c {
            return a
        }
        return c
    }
    if b < c {
        return b
    }
    return c
}
```

---

## 7. HTTP Handler 扩展

### 7.1 新增端点

**文件**: `services/management-service/internal/port/http/scheduling_rule_handler.go` (新增方法)

```go
// ParseRule 解析自然语言规则
// POST /v1/rules/parse
func (h *SchedulingRuleHandler) ParseRule(c *gin.Context) {
    var req rule_parser.ParseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "请求参数无效: " + err.Error()})
        return
    }

    // 获取组织的班次和分组列表（供LLM匹配用）
    if len(req.ShiftNames) == 0 {
        shifts, err := h.shiftService.ListShifts(c, req.OrgID)
        if err == nil {
            for _, s := range shifts {
                req.ShiftNames = append(req.ShiftNames, s.Name)
            }
        }
    }
    if len(req.GroupNames) == 0 {
        groups, err := h.groupService.ListGroups(c, req.OrgID)
        if err == nil {
            for _, g := range groups {
                req.GroupNames = append(req.GroupNames, g.Name)
            }
        }
    }

    resp, err := h.ruleParser.Parse(c, &req)
    if err != nil {
        c.JSON(500, gin.H{"error": "规则解析服务异常: " + err.Error()})
        return
    }

    c.JSON(200, resp)
}

// BatchParseRules 批量解析规则
// POST /v1/rules/batch-parse
func (h *SchedulingRuleHandler) BatchParseRules(c *gin.Context) {
    var req rule_parser.BatchParseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "请求参数无效: " + err.Error()})
        return
    }

    resp, err := h.ruleParser.BatchParse(c, &req)
    if err != nil {
        c.JSON(500, gin.H{"error": "规则解析服务异常: " + err.Error()})
        return
    }

    c.JSON(200, resp)
}
```

### 7.2 路由注册

```go
// 在路由注册中添加
ruleGroup := v1.Group("/rules")
{
    ruleGroup.POST("/parse", ruleHandler.ParseRule)
    ruleGroup.POST("/batch-parse", ruleHandler.BatchParseRules)
}
```

---

## 8. MCP Tool 扩展

### 8.1 新增工具

**文件**: `mcp-servers/rostering/tool/rule/parse_rule.go`

```go
package rule

// ParseRule MCP Tool - 解析自然语言规则
// Tool Name: parse_rule
// Description: 将自然语言排班规则描述解析为结构化的V4规则格式
// Input:
//   - orgId: string (required) - 组织ID
//   - ruleText: string (required) - 自然语言规则描述
// Output:
//   - ParseResponse JSON

func (t *ParseRuleTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
    orgID := input["orgId"].(string)
    ruleText := input["ruleText"].(string)

    return t.ruleService.ParseRule(ctx, orgID, ruleText)
}
```

---

## 9. 测试用例

### 9.1 解析测试场景

| 序号 | 输入 | 期望解析 | 验证重点 |
|------|------|---------|---------|
| 1 | "早班每人每周最多3次" | ruleType=maxCount, maxCount=3, timeScope=same_week, association=[早班/target] | 量化参数提取 |
| 2 | "夜班和早班不能连续排" | ruleType=exclusive, category=constraint, association=[夜班/target, 早班/reference] | 互斥规则+双关联 |
| 3 | "尽量安排高年资护士上夜班" | category=preference, subCategory=prefer, association=[夜班/target, 高年资/source] | 偏好+来源 |
| 4 | "每月每人休息至少4天" | ruleType=maxCount, minRestDays=4, timeScope=same_month, applyScope=global | 全局规则 |
| 5 | "下夜班人员必须来自上半夜班" | category=dependency, subCategory=source, association=[下夜班/target, 上半夜班/source] | 依赖型 |
| 6 | "周末禁止安排实习护士" | ruleType=forbidden_day, category=constraint, subCategory=forbid | 禁止型 |
| 7 | 混合输入(含3条规则) | 3条ParseResult | 多规则拆解 |
| 8 | 模糊输入"班次别排太多" | 低 confidence + warning | 模糊容错 |

### 9.2 验证器测试

```go
func TestValidator_CategorySubCategoryMismatch(t *testing.T) {
    result := &ParseResult{
        Name:        "测试规则",
        RuleType:    "maxCount",
        Category:    "constraint",
        SubCategory: "prefer", // 不匹配！
        TimeScope:   "same_week",
    }
    
    v := NewRuleValidator()
    vr := v.Validate(result)
    
    assert.False(t, vr.IsValid)
    assert.Contains(t, vr.Errors[0].Code, "CATEGORY_SUBCATEGORY_MISMATCH")
}
```
