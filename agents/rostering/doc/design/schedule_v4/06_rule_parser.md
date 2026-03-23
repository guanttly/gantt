# 06. 规则语义解析服务详细设计

> **开发负责人**: Agent-4  
> **依赖**: Agent-1 (数据模型)  
> **被依赖**: Agent-5 (前端), Agent-6 (API)  
> **部署位置**: management-service（管理端）  
> **包路径**: `services/management-service/internal/rule_parser/`

## 1. 设计目标

将用户的自然语言规则描述，通过 LLM **一次性解析**为结构化的 V4 规则字段，实现 **"解析一次，执行多次"** 的模式。

### 核心流程

```
用户输入: "早班每人每周最多排3次"
         ↓
    [LLM 解析] ← 只在规则录入时调用一次
         ↓
    结构化结果:
    {
      "ruleType": "maxCount",
      "category": "constraint",
      "subCategory": "frequency",
      "maxCount": 3,
      "timeScope": "same_week",
      "applyScope": "shift",
      "associationTargets": [{"type": "shift", "name": "早班"}]
    }
         ↓
    [三层验证]
         ↓
    保存到数据库（V4 规则）
```

## 2. 类型定义

**文件**: `services/management-service/internal/rule_parser/types.go`

```go
package rule_parser

import (
    "time"
    d_model "jusha/sdk/rostering/model"
)

// ParseRequest 规则解析请求
type ParseRequest struct {
    OrgID       string   `json:"orgId"`
    RuleText    string   `json:"ruleText"`    // 自然语言规则描述
    ShiftNames  []string `json:"shiftNames"`  // 当前组织的班次名称列表（供LLM匹配）
    GroupNames  []string `json:"groupNames"`  // 当前组织的分组名称列表
}

// ParseResult 规则解析结果
type ParseResult struct {
    Success           bool                  `json:"success"`
    
    // 解析出的结构化字段
    RuleName          string                `json:"ruleName"`
    RuleType          string                `json:"ruleType"`
    Category          d_model.RuleCategory     `json:"category"`
    SubCategory       d_model.RuleSubCategory  `json:"subCategory"`
    Description       string                `json:"description"`
    
    // 量化参数
    MaxCount          *int                  `json:"maxCount,omitempty"`
    ConsecutiveMax    *int                  `json:"consecutiveMax,omitempty"`
    IntervalDays      *int                  `json:"intervalDays,omitempty"`
    MinRestDays       *int                  `json:"minRestDays,omitempty"`
    
    // 时间/范围
    TimeScope         string                `json:"timeScope"`       // same_day/same_week/same_month
    ApplyScope        string                `json:"applyScope"`      // global/shift/group
    
    // 关联对象
    AssociationTargets []ParsedAssociation  `json:"associationTargets"`
    
    // 优先级建议
    SuggestedPriority int                   `json:"suggestedPriority"`
    
    // 回译文本（用于用户确认）
    BackTranslation   string                `json:"backTranslation"`
    
    // 验证结果
    Validation        *ValidationStatus     `json:"validation"`
}

// ParsedAssociation 解析出的关联对象
type ParsedAssociation struct {
    Type   string `json:"type"`    // shift / group / employee
    Name   string `json:"name"`    // 名称（LLM识别出的）
    ID     string `json:"id"`      // 匹配的ID（后端填充）
    Role   string `json:"role"`    // target / source / reference
}

// BatchParseRequest 批量解析请求
type BatchParseRequest struct {
    OrgID      string   `json:"orgId"`
    RuleTexts  []string `json:"ruleTexts"`   // 多条规则文本（每行一条）
    ShiftNames []string `json:"shiftNames"`
    GroupNames []string `json:"groupNames"`
}

// BatchParseResult 批量解析结果
type BatchParseResult struct {
    Results    []*ParseResult `json:"results"`
    TotalCount int            `json:"totalCount"`
    SuccessCount int          `json:"successCount"`
    FailedCount  int          `json:"failedCount"`
}

// ============================================================
// 三层验证
// ============================================================

// ValidationStatus 验证状态
type ValidationStatus struct {
    StructuralValid   bool     `json:"structuralValid"`   // 结构层验证
    BackTransValid    bool     `json:"backTransValid"`    // 回译层验证
    SimulationValid   bool     `json:"simulationValid"`   // 模拟层验证
    StructuralErrors  []string `json:"structuralErrors,omitempty"`
    SimulationWarnings []string `json:"simulationWarnings,omitempty"`
}

// SimulationInput 模拟验证输入
type SimulationInput struct {
    ParsedRule   *ParseResult
    MockStaff    []*d_model.Employee     // 模拟人员
    MockShifts   []*d_model.Shift        // 模拟班次
    MockSchedule map[string][]string     // date -> staffIDs
}

// SimulationResult 模拟验证结果
type SimulationResult struct {
    IsValid         bool     `json:"isValid"`
    TriggerCount    int      `json:"triggerCount"`    // 规则触发次数
    ExclusionCount  int      `json:"exclusionCount"`  // 排除人员次数
    Warnings        []string `json:"warnings"`
}
```

## 3. 解析器实现

**文件**: `services/management-service/internal/rule_parser/parser.go`

```go
package rule_parser

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "jusha/mcp/pkg/ai"
    "jusha/mcp/pkg/logging"
)

// IRuleParser 规则解析器接口
type IRuleParser interface {
    // Parse 解析单条规则
    Parse(ctx context.Context, req *ParseRequest) (*ParseResult, error)
    
    // ParseBatch 批量解析规则
    ParseBatch(ctx context.Context, req *BatchParseRequest) (*BatchParseResult, error)
    
    // Validate 三层验证
    Validate(ctx context.Context, result *ParseResult, req *ParseRequest) (*ValidationStatus, error)
}

// RuleParser 规则解析器
type RuleParser struct {
    logger    logging.ILogger
    aiFactory *ai.AIProviderFactory
    validator *RuleValidator
}

// NewRuleParser 创建规则解析器
func NewRuleParser(
    logger logging.ILogger,
    aiFactory *ai.AIProviderFactory,
) *RuleParser {
    return &RuleParser{
        logger:    logger.With("component", "RuleParser"),
        aiFactory: aiFactory,
        validator: NewRuleValidator(logger),
    }
}

// Parse 解析单条规则
func (p *RuleParser) Parse(ctx context.Context, req *ParseRequest) (*ParseResult, error) {
    // 1. 构建解析 Prompt
    prompt := p.buildParsePrompt(req)
    
    // 2. 调用 LLM
    provider := p.aiFactory.GetDefaultProvider()
    resp, err := provider.Chat(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("LLM 调用失败: %w", err)
    }
    
    // 3. 解析 LLM 响应
    result, err := p.parseLLMResponse(resp)
    if err != nil {
        return nil, fmt.Errorf("响应解析失败: %w", err)
    }
    
    // 4. 关联对象名称→ID 匹配
    p.resolveAssociationIDs(result, req)
    
    // 5. 三层验证
    validation, err := p.validator.Validate(ctx, result, req)
    if err != nil {
        p.logger.Warn("Validation error", "error", err)
    }
    result.Validation = validation
    
    return result, nil
}

// ParseBatch 批量解析
func (p *RuleParser) ParseBatch(ctx context.Context, req *BatchParseRequest) (*BatchParseResult, error) {
    batchResult := &BatchParseResult{
        TotalCount: len(req.RuleTexts),
    }
    
    for _, text := range req.RuleTexts {
        text = strings.TrimSpace(text)
        if text == "" {
            continue
        }
        
        singleReq := &ParseRequest{
            OrgID:      req.OrgID,
            RuleText:   text,
            ShiftNames: req.ShiftNames,
            GroupNames:  req.GroupNames,
        }
        
        result, err := p.Parse(ctx, singleReq)
        if err != nil {
            batchResult.Results = append(batchResult.Results, &ParseResult{
                Success:     false,
                Description: fmt.Sprintf("解析失败: %s", err.Error()),
            })
            batchResult.FailedCount++
        } else {
            batchResult.Results = append(batchResult.Results, result)
            if result.Success {
                batchResult.SuccessCount++
            } else {
                batchResult.FailedCount++
            }
        }
    }
    
    return batchResult, nil
}

// buildParsePrompt 构建解析 Prompt
func (p *RuleParser) buildParsePrompt(req *ParseRequest) string {
    var sb strings.Builder
    
    sb.WriteString(`你是排班规则解析器。将自然语言规则描述解析为结构化 JSON。

## 规则分类体系

### category (大类)
- constraint: 约束型（必须遵守）
- preference: 偏好型（尽量满足）
- dependency: 依赖型（影响执行顺序）

### subCategory (子类)
约束型:
- frequency: 频次限制（每周/每月最多几次）
- continuity: 连续性限制（连续排班天数上限）
- rest: 休息要求（最少休息天数/小时）
- exclusive: 排他约束（班次互斥）
- forbidden: 禁止约束（禁止某日/某时段）
- headcount: 人数约束（最少/最多人数）

偏好型:
- balance: 均衡偏好（均匀分配排班）
- personnel: 人员偏好（特定人员偏好特定班次）
- time: 时间偏好（特定时间段偏好）

依赖型:
- prerequisite: 前置依赖（排A班前必须排过B班）
- sequence: 顺序依赖（A班排完才排B班）

### ruleType (规则引擎类型)
- maxCount: 频次上限
- consecutiveMax: 连续天数上限
- minRestDays: 最少休息天数
- exclusive: 排他（同日不可同排）
- forbidden_day: 禁止日期
- required_together: 必须同排
- preferred: 偏好

### timeScope (时间范围)
- same_day: 同一天
- same_week: 同一周
- same_month: 同一月

### applyScope (适用范围)
- global: 全局（所有班次）
- shift: 特定班次
- group: 特定分组

## 当前组织数据
`)
    
    if len(req.ShiftNames) > 0 {
        sb.WriteString(fmt.Sprintf("班次列表: %s\n", strings.Join(req.ShiftNames, ", ")))
    }
    if len(req.GroupNames) > 0 {
        sb.WriteString(fmt.Sprintf("分组列表: %s\n", strings.Join(req.GroupNames, ", ")))
    }
    
    sb.WriteString(fmt.Sprintf("\n## 需要解析的规则\n\"%s\"\n\n", req.RuleText))
    
    sb.WriteString(`## 输出格式
请输出 JSON:
` + "```json" + `
{
  "success": true,
  "ruleName": "简短规则名",
  "ruleType": "maxCount",
  "category": "constraint",
  "subCategory": "frequency",
  "description": "原始描述",
  "maxCount": 3,
  "timeScope": "same_week",
  "applyScope": "shift",
  "associationTargets": [
    {"type": "shift", "name": "早班", "role": "target"}
  ],
  "suggestedPriority": 8,
  "backTranslation": "早班每位员工每周最多排班3次"
}
` + "```" + `

注意:
1. backTranslation 必须精确反映解析结果，用于用户确认
2. 量化参数只填与 ruleType 匹配的字段
3. associationTargets.name 必须从组织数据中匹配
4. suggestedPriority 范围 1-10，约束型默认 8-10，偏好型默认 3-5
`)
    
    return sb.String()
}

// parseLLMResponse 解析 LLM 响应
func (p *RuleParser) parseLLMResponse(response string) (*ParseResult, error) {
    // 提取 JSON
    jsonStr := extractJSONBlock(response)
    if jsonStr == "" {
        return nil, fmt.Errorf("未找到 JSON 块")
    }
    
    var result ParseResult
    if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
        return nil, fmt.Errorf("JSON 解析失败: %w", err)
    }
    
    return &result, nil
}

// resolveAssociationIDs 匹配关联对象名称到ID
// 在实际实现中需要查数据库
func (p *RuleParser) resolveAssociationIDs(result *ParseResult, req *ParseRequest) {
    // TODO: 查询数据库匹配 shift name → shift ID, group name → group ID
    // 这里先留接口
    for i, assoc := range result.AssociationTargets {
        _ = assoc
        result.AssociationTargets[i].ID = "" // 待填充
    }
}

// extractJSONBlock 提取 JSON 块
func extractJSONBlock(text string) string {
    // 与 executor/result_parser.go 中的逻辑相同
    // TODO: 提取到 pkg/utils/ 作为公共工具
    import "regexp"
    
    re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return strings.TrimSpace(matches[1])
    }
    
    re2 := regexp.MustCompile(`(?s)\{.*\}`)
    match := re2.FindString(text)
    return match
}
```

## 4. 三层验证器

**文件**: `services/management-service/internal/rule_parser/validator.go`

```go
package rule_parser

import (
    "context"
    "fmt"
    "jusha/mcp/pkg/logging"
)

// RuleValidator 规则三层验证器
type RuleValidator struct {
    logger logging.ILogger
}

func NewRuleValidator(logger logging.ILogger) *RuleValidator {
    return &RuleValidator{logger: logger.With("sub", "RuleValidator")}
}

// Validate 执行三层验证
func (v *RuleValidator) Validate(
    ctx context.Context,
    result *ParseResult,
    req *ParseRequest,
) (*ValidationStatus, error) {
    status := &ValidationStatus{
        StructuralValid:  true,
        BackTransValid:   true,
        SimulationValid:  true,
    }
    
    // === 第一层: 结构层验证 ===
    v.validateStructural(result, status)
    
    // === 第二层: 回译层验证 ===
    // 回译文本是否准确反映了解析结果？
    // 由前端展示给用户确认，此处只检查是否存在
    if result.BackTranslation == "" {
        status.BackTransValid = false
    }
    
    // === 第三层: 模拟层验证（可选） ===
    // 使用规则引擎模拟执行，检查规则是否能正常工作
    // 实际实现中需要调用 engine.ConstraintChecker
    
    return status, nil
}

// validateStructural 结构层验证
func (v *RuleValidator) validateStructural(result *ParseResult, status *ValidationStatus) {
    errors := make([]string, 0)
    
    // 1. 必填字段检查
    if result.RuleType == "" {
        errors = append(errors, "ruleType 不能为空")
    }
    if result.Category == "" {
        errors = append(errors, "category 不能为空")
    }
    if result.SubCategory == "" {
        errors = append(errors, "subCategory 不能为空")
    }
    
    // 2. ruleType 与量化参数一致性
    switch result.RuleType {
    case "maxCount":
        if result.MaxCount == nil || *result.MaxCount <= 0 {
            errors = append(errors, "maxCount 规则必须有有效的 maxCount 值")
        }
    case "consecutiveMax":
        if result.ConsecutiveMax == nil || *result.ConsecutiveMax <= 0 {
            errors = append(errors, "consecutiveMax 规则必须有有效的 consecutiveMax 值")
        }
    case "minRestDays":
        if result.MinRestDays == nil || *result.MinRestDays <= 0 {
            errors = append(errors, "minRestDays 规则必须有有效的 minRestDays 值")
        }
    case "exclusive":
        // 排他规则必须有至少2个关联班次
        shiftAssocCount := 0
        for _, a := range result.AssociationTargets {
            if a.Type == "shift" {
                shiftAssocCount++
            }
        }
        if shiftAssocCount < 2 {
            errors = append(errors, "exclusive 规则需要至少关联2个班次")
        }
    }
    
    // 3. category 与 subCategory 一致性
    validSubCategories := map[string][]string{
        "constraint": {"frequency", "continuity", "rest", "exclusive", "forbidden", "headcount"},
        "preference": {"balance", "personnel", "time"},
        "dependency": {"prerequisite", "sequence"},
    }
    if subs, ok := validSubCategories[string(result.Category)]; ok {
        found := false
        for _, s := range subs {
            if string(result.SubCategory) == s {
                found = true
                break
            }
        }
        if !found {
            errors = append(errors, fmt.Sprintf("subCategory '%s' 不属于 category '%s'", result.SubCategory, result.Category))
        }
    }
    
    // 4. timeScope 合法性
    validTimeScopes := map[string]bool{
        "same_day": true, "same_week": true, "same_month": true, "": true,
    }
    if !validTimeScopes[result.TimeScope] {
        errors = append(errors, fmt.Sprintf("无效的 timeScope: %s", result.TimeScope))
    }
    
    // 5. 关联对象验证
    for _, assoc := range result.AssociationTargets {
        if assoc.Type == "" {
            errors = append(errors, "关联对象 type 不能为空")
        }
        validTypes := map[string]bool{"shift": true, "group": true, "employee": true}
        if !validTypes[assoc.Type] {
            errors = append(errors, fmt.Sprintf("无效的关联对象 type: %s", assoc.Type))
        }
        if assoc.Role == "" {
            errors = append(errors, "关联对象 role 不能为空")
        }
        validRoles := map[string]bool{"target": true, "source": true, "reference": true}
        if !validRoles[assoc.Role] {
            errors = append(errors, fmt.Sprintf("无效的关联对象 role: %s", assoc.Role))
        }
    }
    
    if len(errors) > 0 {
        status.StructuralValid = false
        status.StructuralErrors = errors
    }
}
```

## 5. 解析 Prompt 示例

### 输入输出对照表

| 自然语言输入 | ruleType | category | subCategory | 量化参数 | timeScope |
|-------------|----------|----------|-------------|---------|-----------|
| "早班每人每周最多排3次" | maxCount | constraint | frequency | maxCount=3 | same_week |
| "夜班后至少休息1天" | minRestDays | constraint | rest | minRestDays=1 | - |
| "连续排班不超过5天" | consecutiveMax | constraint | continuity | consecutiveMax=5 | - |
| "早班和夜班同日不能排同一个人" | exclusive | constraint | exclusive | - | same_day |
| "尽量让每人排班次数均匀" | preferred | preference | balance | - | same_month |
| "护士A偏好排早班" | preferred | preference | personnel | - | - |
| "周末每班至少2人" | headcount | constraint | headcount | maxCount=2 | - |
| "排中班前必须排过早班" | prerequisite | dependency | prerequisite | - | - |

### 回译验证示例

```
原文: "早班每人每周最多排3次"
解析: {ruleType: "maxCount", maxCount: 3, timeScope: "same_week", 关联: [早班]}
回译: "早班每位员工每周最多排班3次"
→ 用户确认: ✅ 语义一致
```

```
原文: "夜班后至少休息1天才能排早班"
解析: {ruleType: "minRestDays", minRestDays: 1, 关联: [{夜班,source}, {早班,target}]}
回译: "排夜班后至少休息1天才能排早班"
→ 用户确认: ✅ 语义一致
```

## 6. 模拟验证流程

```
                                ┌──────────────┐
                                │ 解析结果      │
                                └──────┬───────┘
                                       │
                          ┌────────────▼────────────┐
                          │  构建模拟场景             │
                          │  - 3个模拟员工            │
                          │  - 7天排班周期            │
                          │  - 相关班次               │
                          └────────────┬────────────┘
                                       │
                          ┌────────────▼────────────┐
                          │  引擎执行约束检查         │
                          │  ConstraintChecker       │
                          │  .checkSingleConstraint  │
                          └────────────┬────────────┘
                                       │
                          ┌────────────▼────────────┐
                          │  统计结果                 │
                          │  - 触发次数               │
                          │  - 排除人员次数           │
                          │  - 是否过于宽松/严格      │
                          └────────────┬────────────┘
                                       │
                          ┌────────────▼────────────┐
                          │  输出 SimulationResult   │
                          │  如果 triggerCount=0     │
                          │  → 警告"规则可能永远不会  │
                          │    生效，请检查"          │
                          └─────────────────────────┘
```

## 7. 测试要求

### 解析准确性
- 频次类规则（maxCount）解析正确率 > 95%
- 连续性规则（consecutiveMax）解析正确率 > 95%
- 排他规则（exclusive）解析正确率 > 90%
- 模糊/歧义规则给出合理默认值

### 结构验证
- ruleType 与量化参数一致性 100%
- category 与 subCategory 一致性 100%
- 关联对象类型和角色合法性 100%

### 回译验证
- 回译文本与原文语义一致率 > 90%
- 用户可通过对比确认解析正确性

### 边界情况
- 空输入处理
- 无法识别的规则类型
- 关联班次名称不存在时的处理
- 批量解析中部分失败的处理
