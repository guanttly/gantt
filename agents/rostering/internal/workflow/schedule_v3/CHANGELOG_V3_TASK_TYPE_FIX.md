# V3排班任务类型修复 - 变更日志

**修复日期**: 2026-01-27  
**问题描述**: task_2和task_3被误判为规则校验任务，导致不调用LLM执行排班

---

## 问题根因

### 1. 任务类型判断机制缺陷
- 原逻辑通过**关键词匹配**判断任务类型（`isRuleValidationTask`方法）
- 检查任务标题/描述中是否包含"校验"、"验证"等关键词
- 容易误判：LLM生成的任务描述中偶然包含这些词就会被识别为规则校验任务

### 2. 规则校验任务是空实现
- `executeRuleValidationTask`方法只是复制当前草案并返回
- 不调用LLM，不执行任何实际操作
- 注释说明："规则级校验已经在任务执行后自动执行"

### 3. 执行流程
```
ExecuteProgressiveTask
  ├─ if isRuleValidationTask(task)  [✓ task_2/task_3因包含"校验"被匹配]
  │   └─ executeRuleValidationTask  [空实现，直接返回]
  │
  ├─ else if aiFactory != nil       [✓ task_1正常执行]
  │   └─ executeAITask              [调用LLM生成排班]
  │
  └─ else
      └─ executeRemainingFillTask   [回退逻辑]
```

---

## 修复方案

### 1. 为ProgressiveTask模型添加Type字段
**文件**: `domain/model/progressive_scheduling.go`

```go
type ProgressiveTask struct {
    // ... 其他字段
    
    // Type 任务类型: "ai"(AI执行), "fill"(填充逻辑), "validation"(规则校验-已废弃)
    Type string `json:"type"`
    
    // ... 其他字段
}
```

**优势**:
- 显式标记任务类型，避免运行时通过文本匹配判断
- 支持向后兼容（Type为空时自动推断）

### 2. 移除规则校验任务类型
**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/progressive_task_executor.go`

**删除的方法**:
- `isRuleValidationTask()` - 关键词匹配判断逻辑
- `executeRuleValidationTask()` - 空实现方法

**修改的逻辑**:
```go
// 优先使用显式的任务类型字段，如果没有则根据aiFactory判断
if task.Type == "validation" {
    // 规则校验任务类型已废弃，统一使用AI或填充逻辑
    logger.Warn("Task type 'validation' is deprecated, treating as AI task")
}

if aiFactory != nil && (task.Type == "" || task.Type == "ai") {
    // AI执行任务
    executeAITask(...)
} else if task.Type == "fill" || aiFactory == nil {
    // 填充逻辑（显式指定或AI不可用时回退）
    executeRemainingFillTask(...)
} else {
    // 未知任务类型，返回错误
    return error("unsupported task type")
}
```

### 3. 更新任务生成逻辑
**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/progressive_scheduling.go`

在`parseRequirementAssessmentResult`方法中添加Type字段处理：

```go
// 设置默认任务类型（如果未指定）
if task.Type == "" {
    // 根据标题和描述推断类型（向后兼容旧数据）
    title := strings.ToLower(task.Title)
    description := strings.ToLower(task.Description)
    
    // 检查是否包含"校验"、"验证"等关键词
    validationKeywords := []string{"校验", "验证", "validation", "检查规则"}
    isValidation := false
    for _, keyword := range validationKeywords {
        if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
            isValidation = true
            break
        }
    }
    
    if isValidation {
        // 废弃的validation类型，默认改为ai
        task.Type = "ai"
        logger.Warn("Task contains validation keywords, setting type to 'ai'")
    } else {
        // 默认使用AI执行
        task.Type = "ai"
    }
}
```

**特性**:
- 向后兼容：对于没有Type字段的旧任务，自动推断并设置
- 废弃警告：如果检测到validation关键词，记录警告并转为ai类型

### 4. 添加LLM调用监控日志
在所有LLM调用点添加详细的监控日志：

**监控信息**:
- 调用时间点和耗时
- 提示词长度和响应长度
- 任务ID、班次ID等上下文信息
- 错误信息（如果失败）

**修改位置**:
1. `parseTaskTargetShifts` - 任务解析调用
2. `executeAITaskForSingleShift` - 排班生成调用
3. `analyzeFailureWithAI` - 失败分析调用

**日志格式**:
```go
// 调用前
logger.Info("[LLM Call] Parsing task to identify target shifts",
    "taskID", task.ID,
    "taskTitle", task.Title,
    "promptLength", len(userPrompt))

// 调用后
logger.Info("[LLM Call] Task parsing completed",
    "taskID", task.ID,
    "duration", llmCallDuration.Seconds(),
    "responseLength", len(resp.Content))

// 失败时
logger.Error("[LLM Call] Task parsing failed",
    "taskID", task.ID,
    "duration", llmCallDuration.Seconds(),
    "error", err)
```

---

## 影响范围

### 修改的文件
1. `domain/model/progressive_scheduling.go` - 添加Type字段
2. `agents/rostering/internal/workflow/schedule_v3/utils/progressive_task_executor.go` - 重构任务执行逻辑
3. `agents/rostering/internal/workflow/schedule_v3/utils/progressive_scheduling.go` - 更新任务生成逻辑

### 兼容性
- ✅ **向后兼容**: 对于没有Type字段的旧任务，自动推断并设置
- ✅ **数据库兼容**: Type字段为可选，不影响现有数据
- ✅ **API兼容**: 前端可以继续不传Type字段

### 行为变更
- ❌ **废弃**: 规则校验任务类型（validation）已废弃
- ✅ **新增**: 支持显式指定任务类型（ai/fill）
- ✅ **改进**: 所有任务统一通过AI或填充逻辑执行

---

## 验证方法

### 1. 检查日志中的LLM调用
运行排班任务后，搜索日志中的`[LLM Call]`标记：
```bash
grep "\[LLM Call\]" rostering-agent.log
```

应该看到：
- `[LLM Call] Parsing task to identify target shifts` - 任务解析
- `[LLM Call] Executing AI task for single shift` - 排班生成
- 每次调用都有对应的`completed`或`failed`日志

### 2. 验证任务执行流程
检查日志，确认任务不再走规则校验路径：
```bash
# 不应该再看到这条日志
grep "Executing rule validation task" rostering-agent.log

# 应该看到AI任务执行
grep "AI task completed" rostering-agent.log
```

### 3. 检查任务Type字段
在任务生成后，检查任务的Type字段是否正确设置：
```bash
grep "Task type" rostering-agent.log
```

---

## 后续改进建议

### 1. 在任务生成阶段指定Type
修改任务生成的Prompt，要求LLM明确返回任务类型：
```json
{
  "tasks": [
    {
      "id": "task_1",
      "title": "...",
      "description": "...",
      "type": "ai"  // 明确指定类型
    }
  ]
}
```

### 2. 移除validation关键词检测
一旦确认所有任务都有Type字段，可以移除向后兼容的关键词检测逻辑。

### 3. 添加任务类型统计
在日志或监控中添加任务类型的统计信息，便于分析任务分布。

---

## 总结

本次修复彻底解决了任务类型误判导致的LLM调用缺失问题：

✅ **根本原因已修复**: 移除关键词匹配，使用显式Type字段  
✅ **废弃空实现**: 移除规则校验任务类型  
✅ **向后兼容**: 支持旧数据自动推断类型  
✅ **监控增强**: 添加LLM调用详细日志  

现在所有任务都会正确地通过AI或填充逻辑执行，不会再出现"跳过执行"的情况。
