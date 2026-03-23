# IntentService 重构说明

## 概述

本次重构将 IntentService 从复杂的多策略设计简化为单一职责设计，实现了关注点分离。

## 重构前的问题

1. **职责混乱**: IntentService 同时负责初始意图识别和工作流内意图分析
2. **策略选择复杂**: `selectStrategy()` 方法根据会话状态动态选择策略，逻辑复杂易出错
3. **`{currentDate}` 未替换**: 在 `scheduleAdjust` 策略中使用的日期占位符没有被替换
4. **类型不一致**: YAML 配置中使用 `unknown`，但代码中使用 `other`

## 重构后的架构

### 1. IntentService (简化)

**文件**: `internal/service/intent.go`

**职责**: 仅负责初始意图识别（用户首次进入时）

**关键变更**:
- 删除 `selectStrategy()` 方法
- 始终使用 `initial` 策略
- `MapIntentToEvent()` 使用静态映射表
- 代码量从 ~650 行减少到 ~300 行

```go
// 简化后的 Recognize 方法
func (s *intentService) Recognize(ctx context.Context, req session.IntentRecognizeRequest) (*session.IntentRecognizeResponse, error) {
    // 始终使用 "initial" 策略
    strategy, ok := s.cfg.Intent.Strategies["initial"]
    // ...
}
```

### 2. ISchedulingAIService (扩展)

**文件**: `domain/service/scheduling_ai.go`

**新增方法**:

```go
// AnalyzeAdjustIntent 分析用户的排班调整意图
// 在 schedule.adjust 工作流中使用，识别用户想要进行的调整操作类型
AnalyzeAdjustIntent(ctx context.Context, userInput string, messages []session.Message) (*model.AdjustIntent, error)
```

**实现文件**: `internal/service/scheduling_ai.go`

**特点**:
- 使用 `scheduleAdjust` 策略
- 自动替换 `{currentDate}` 占位符
- 直接返回 `*AdjustIntent`，无需类型转换

### 3. 意图-工作流映射表

**文件**: `domain/model/intent.go`

**新增**:

```go
type IntentWorkflowMapping struct {
    WorkflowName string // 工作流名称 (e.g., "schedule.create")
    Event        string // 触发事件 (统一使用 "start")
    Implemented  bool   // 是否已实现
}

var intentWorkflowMappings = map[IntentType]*IntentWorkflowMapping{
    IntentScheduleCreate: {"schedule.create", "start", true},
    IntentScheduleAdjust: {"schedule.adjust", "start", true},
    IntentScheduleView:   {"schedule.view", "start", false},
    IntentRuleManage:     {"rule.manage", "start", false},
    IntentDeptManage:     {"dept.manage", "start", false},
    IntentHelp:           {"general.help", "start", false},
    IntentOther:          {"", "", false},
}

func GetWorkflowMapping(intentType IntentType) *IntentWorkflowMapping
```

### 4. 工作流调用变更

**文件**: `internal/workflow/schedule/adjust/phase_intent.go`

**变更**: `actScheduleAdjustAnalyzeIntent` 函数

```go
// 旧代码
intentSvc, ok := wctx.Services().Get("intentService")
svc := intentSvc.(d_service.IIntentService)
resp, err := svc.Recognize(ctx, req)
adjustCtx.ParsedIntent = ConvertIntentToAdjustIntent(resp.Intent)

// 新代码
schedulingAISvc, ok := wctx.Services().Get("schedulingAIService")
svc := schedulingAISvc.(d_service.ISchedulingAIService)
adjustIntent, err := svc.AnalyzeAdjustIntent(ctx, userIntent, sess.Messages)
adjustCtx.ParsedIntent = adjustIntent  // 直接使用，无需转换
```

### 5. YAML 配置简化

**文件**: `test/data/config/scheduling-service.yml`

**保留的策略**:
- `initial`: 初始意图识别
- `scheduleAdjust`: 排班调整意图分析

**删除的策略**:
- `inWorkflow`: 工作流内意图识别（不再需要）
- `confirmation`: 确认阶段意图识别（不再需要）

**统一术语**:
- `unknown` → `other`

## 调用流程

### 初始意图识别（用户首次输入）

```
用户输入
    ↓
IntentService.Recognize() [使用 initial 策略]
    ↓
返回 IntentType (schedule/rule/dept/general/other)
    ↓
IntentService.MapIntentToEvent() [使用静态映射表]
    ↓
返回 WorkflowName + Event
    ↓
启动对应工作流
```

### 排班调整意图分析（工作流内）

```
用户在 schedule.adjust 工作流中输入
    ↓
actScheduleAdjustAnalyzeIntent()
    ↓
SchedulingAIService.AnalyzeAdjustIntent() [使用 scheduleAdjust 策略]
    ↓
返回 *AdjustIntent (swap/replace/add/remove/batch/regenerate/custom/other)
    ↓
继续工作流处理
```

## 删除的代码

1. `internal/service/intent.go`:
   - `selectStrategy()` 方法
   - 复杂的 `buildUserPrompt()` 方法（包含工作流状态注入）

2. `internal/workflow/schedule/adjust/helpers.go`:
   - `ConvertIntentToAdjustIntent()` 函数

3. YAML 配置:
   - `inWorkflow` 策略
   - `confirmation` 策略

## 优点

1. **关注点分离**: IntentService 只负责初始识别，SchedulingAIService 负责工作流内分析
2. **代码更简洁**: 删除了复杂的策略选择逻辑
3. **类型安全**: 直接返回正确的类型，无需转换
4. **可扩展性**: 新增工作流只需更新映射表
5. **可维护性**: 每个服务职责明确，易于理解和修改

## 迁移注意事项

1. 确保 `schedulingAIService` 已注册到工作流的 Services 容器
2. 其他使用 IntentService 的地方不受影响（仅用于初始识别）
3. YAML 配置中的 `unknown` 已统一改为 `other`
