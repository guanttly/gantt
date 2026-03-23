# V3排班系统完整执行流程图

## 概览

V3排班系统采用**渐进式任务执行**模式，将复杂的排班需求拆分为多个小任务，逐步完成排班。

```
用户请求
   ↓
生成任务计划 (LLM)
   ↓
循环执行任务
   ↓
合并结果 & 校验
   ↓
返回排班结果
```

---

## 详细流程

### 1. 工作流启动阶段

```
用户发起排班请求
   ↓
CreateV3工作流启动
   ├─ 解析排班需求
   ├─ 加载基础数据（人员、班次、规则）
   └─ 调用LLM生成任务计划
      ↓
   parseRequirementAssessmentResult
      ├─ 解析LLM返回的JSON
      ├─ 验证任务质量
      └─ 设置任务Type字段（ai/fill）
```

**任务计划示例**:
```json
{
  "tasks": [
    {
      "id": "task_1",
      "order": 1,
      "title": "正向需求填充：安排指定人员到要求的班次",
      "description": "...",
      "type": "ai",
      "targetShifts": ["shift_1", "shift_2"],
      "priority": 1,
      "status": "pending"
    },
    {
      "id": "task_2",
      "order": 2,
      "title": "特殊班次填充：完成夜班和跨夜班排班",
      "description": "...",
      "type": "ai",
      "targetShifts": ["shift_4", "shift_5", "shift_6"],
      "priority": 1,
      "status": "pending"
    }
  ],
  "summary": "整体排班策略说明",
  "reasoning": "AI的思考过程"
}
```

---

### 2. 任务执行循环

```
FOR EACH task IN taskPlan:
   ↓
CreateV3: spawnCurrentTask
   ├─ 更新任务状态为"executing"
   └─ 启动Core子工作流
      ↓
   CoreV3子工作流
      ├─ Pre-validate（前置校验）
      ├─ Execute Task（执行任务）★
      ├─ Validate Result（结果校验）
      └─ Return to Parent
         ↓
   CreateV3: 任务审查
      ├─ 展示任务结果
      ├─ 等待用户确认
      └─ 继续下一个任务
```

---

### 3. 任务执行核心流程（重点）★

```
CoreV3: actExecuteTask
   ↓
创建 ProgressiveTaskExecutor 实例
   ↓
ExecuteProgressiveTask
   ↓
任务类型判断:
   │
   ├─ task.Type == "validation"?
   │   └─ [已废弃] 输出警告，转为AI任务
   │
   ├─ aiFactory != nil && (task.Type == "" || task.Type == "ai")?
   │   └─ YES → executeAITask ★★
   │       ├─ [LLM Call 1] parseTaskTargetShifts
   │       │   ├─ 构建任务解析提示词
   │       │   ├─ 调用LLM解析任务描述
   │       │   ├─ 识别目标班次
   │       │   └─ 为每个班次生成任务说明
   │       │
   │       ├─ splitTaskByShiftsWithSpecs
   │       │   └─ 将任务拆分为多个单班次子任务
   │       │
   │       ├─ FOR EACH shift IN targetShifts:
   │       │   └─ executeAITaskForSingleShift ★★★
   │       │       ├─ 构建排班上下文 (V3SchedulingContext)
   │       │       ├─ 构建系统提示词 (buildAITaskSystemPrompt)
   │       │       ├─ 构建用户提示词 (buildAITaskUserPromptWithContext)
   │       │       │   ├─ 包含当前排班草案
   │       │       │   ├─ 包含已占位信息
   │       │       │   ├─ 包含个人需求
   │       │       │   └─ 包含人员调度上下文
   │       │       │
   │       │       ├─ [LLM Call 2] aiFactory.CallDefault
   │       │       │   ├─ 记录调用开始时间
   │       │       │   ├─ 执行LLM调用
   │       │       │   └─ 记录调用结果（耗时、响应长度）
   │       │       │
   │       │       ├─ parseAIResponse (解析LLM返回的排班方案)
   │       │       │
   │       │       ├─ validateSingleShift (校验排班结果)
   │       │       │   ├─ 规则级校验 (ValidateAll)
   │       │       │   └─ 数据一致性检查
   │       │       │
   │       │       └─ 如果失败 && 配置了自动重试:
   │       │           ├─ [LLM Call 3] analyzeFailureWithAI
   │       │           │   └─ 分析失败原因并提供改进建议
   │       │           │
   │       │           └─ 重试 executeAITaskForSingleShift
   │       │               (最多重试N次，默认3次)
   │       │
   │       └─ 合并所有班次的排班结果
   │
   └─ task.Type == "fill" || aiFactory == nil?
       └─ YES → executeRemainingFillTask
           └─ 使用非AI逻辑填充人员
```

---

### 4. LLM调用详细流程

#### Call 1: 任务解析 (parseTaskTargetShifts)

**目的**: 识别任务涉及的班次，并为每个班次生成专门的任务说明

```
输入:
├─ 任务标题和描述
├─ 可用班次列表
└─ 系统提示词

↓ [LLM Call]

输出:
├─ 目标班次列表 [{shiftId, shiftName, description}]
└─ 解析思路说明
```

**日志标记**: `[LLM Call] Parsing task to identify target shifts`

**示例**:
```
任务: "特殊班次填充：完成夜班和跨夜班排班"
↓
LLM识别: 
- shift_4 (本部夜班)
- shift_5 (江北夜班)
- shift_6 (下夜班)
```

#### Call 2: 排班生成 (executeAITaskForSingleShift)

**目的**: 为指定班次生成排班方案

```
输入:
├─ 系统提示词 (任务要求 + 排班规则 + 重试上下文)
├─ 用户提示词 (当前排班状态 + 人员信息 + 约束条件)
└─ 排班上下文 (V3SchedulingContext)
    ├─ 已占位信息 (occupiedSlots)
    ├─ 个人需求 (personalNeeds)
    └─ 人员调度上下文 (每个人的近期排班情况)

↓ [LLM Call]

输出:
├─ 排班方案 JSON
├─ AI推理过程
└─ 排班说明
```

**日志标记**: `[LLM Call] Executing AI task for single shift`

**排班方案格式**:
```json
{
  "schedule": {
    "2026-01-20": {
      "shift_1": ["emp_001", "emp_002"]
    },
    "2026-01-21": {
      "shift_1": ["emp_003", "emp_004"]
    }
  },
  "reasoning": "AI推理说明",
  "explanation": "排班决策说明"
}
```

#### Call 3: 失败分析 (analyzeFailureWithAI)

**目的**: 分析排班失败原因，提供改进建议

```
输入:
├─ 失败的排班草案
├─ 校验失败详情 (人数不足、时间冲突、规则违反等)
└─ 历史失败记录

↓ [LLM Call]

输出:
├─ 失败根本原因分析
└─ 具体改进建议
```

**日志标记**: `[LLM Call] Analyzing failure with AI`

---

### 5. 校验流程

每个任务执行后都会进行多层校验：

```
TaskResult
   ↓
1. 数据一致性检查
   ├─ 检查日期范围
   ├─ 检查班次ID有效性
   └─ 检查人员ID有效性
   ↓
2. 规则级校验 (ValidateAll)
   ├─ 人数约束校验
   ├─ 时间冲突检查
   ├─ 连班规则检查
   ├─ 禁止模式检查
   └─ 个人需求检查
   ↓
3. LLMQC校验 (可选)
   └─ 使用LLM进行语义级质量检查
   ↓
如果校验失败:
   ├─ 配置了自动重试?
   │   ├─ YES → 调用analyzeFailureWithAI
   │   │        └─ 重新执行任务（最多N次）
   │   └─ NO → 返回失败结果
   └─ 用户可以选择修改或跳过
```

---

### 6. 任务结果合并

```
所有任务执行完成后:
   ↓
合并所有TaskResult
   ├─ 按班次组织排班数据
   ├─ 处理冲突（后执行的任务优先）
   └─ 生成最终排班表
      ↓
转换为前端格式
   └─ 返回给用户
```

---

## 关键数据结构

### V3SchedulingContext

用于传递排班上下文信息给LLM：

```go
type V3SchedulingContext struct {
    ShiftID   string                        // 班次ID
    Date      string                        // 日期 (YYYY-MM-DD)
    ShiftName string                        // 班次名称
    Required  int                           // 需要人数
    
    // 人员调度上下文（每个人的近期排班情况）
    StaffSchedules []V3StaffScheduleContext
}

type V3StaffScheduleContext struct {
    StaffID    string                // 人员ID
    StaffName  string                // 人员姓名
    RecentDays []V3StaffRecentDay    // 近期排班情况
}

type V3StaffRecentDay struct {
    Date       string   // 日期
    ShiftNames []string // 当天排班的班次名称
}
```

**示例**:
```json
{
  "shiftID": "shift_1",
  "date": "2026-01-20",
  "shiftName": "本部穿刺",
  "required": 2,
  "staffSchedules": [
    {
      "staffID": "emp_001",
      "staffName": "张三",
      "recentDays": [
        {
          "date": "2026-01-19",
          "shiftNames": ["本部夜班"]
        },
        {
          "date": "2026-01-18",
          "shiftNames": ["本部穿刺"]
        }
      ]
    }
  ]
}
```

---

## 任务类型说明

### AI任务 (type: "ai")

**特点**:
- 使用LLM智能生成排班方案
- 适用于复杂排班场景（需求填充、特殊班次等）
- 支持自动重试和失败分析

**执行流程**:
1. 解析任务，识别目标班次
2. 按班次拆分为子任务
3. 为每个班次调用LLM生成排班
4. 校验结果，失败则重试
5. 合并所有班次结果

### 填充任务 (type: "fill")

**特点**:
- 使用规则引擎填充人员
- 适用于简单场景（剩余人员填充）
- 不依赖LLM

**执行流程**:
1. 识别未排班的槽位
2. 根据规则筛选可用人员
3. 按优先级分配人员
4. 校验结果

### 规则校验任务 (type: "validation") - 已废弃

**原因**:
- `executeRuleValidationTask`方法是空实现
- 规则校验已在每个任务执行后自动完成
- 独立的校验任务类型是冗余的

**处理方式**:
- 如果检测到`type="validation"`，输出警告
- 自动转为`type="ai"`执行

---

## 监控与日志

### LLM调用监控

所有LLM调用都会记录以下信息：

```
[LLM Call] <操作名称>
├─ 调用前:
│   ├─ taskID
│   ├─ shiftID/shiftName
│   ├─ promptLength
│   └─ timestamp
│
├─ 调用后:
│   ├─ duration (秒)
│   ├─ responseLength
│   └─ success/error
│
└─ 失败时:
    └─ error详情
```

**搜索方法**:
```bash
# 查看所有LLM调用
grep "\[LLM Call\]" rostering-agent.log

# 查看失败的调用
grep "\[LLM Call\].*failed" rostering-agent.log

# 统计调用次数
grep "\[LLM Call\].*completed" rostering-agent.log | wc -l
```

### 任务执行监控

```
任务开始:
├─ "Starting task" (taskID, taskTitle)

任务执行中:
├─ "Parsing task to identify target shifts"
├─ "Task parsing returned N shifts"
├─ "Executing AI task for single shift" (×N次)

任务完成:
├─ "Progressive task executed successfully"
├─ "AI task completed" (shiftCount)
└─ "TaskResult saved to context"
```

---

## 常见问题排查

### 问题1: 任务不调用LLM

**症状**: 日志中看不到`[LLM Call]`标记

**可能原因**:
1. ✅ **任务Type被错误设置为"validation"** (已修复)
2. ✅ **任务标题包含"校验"等关键词** (已修复)
3. aiFactory为nil（AI服务未初始化）
4. 任务被跳过执行

**排查步骤**:
```bash
# 1. 检查任务Type字段
grep "Task type" rostering-agent.log

# 2. 检查任务执行流程
grep "Executing progressive task" rostering-agent.log

# 3. 检查是否有警告
grep "WARN" rostering-agent.log | grep -i task
```

### 问题2: 任务执行失败

**症状**: 日志显示"AI execution failed"

**可能原因**:
1. LLM服务不可用
2. 提示词格式错误
3. LLM返回格式不符合预期

**排查步骤**:
```bash
# 1. 查看错误详情
grep "AI execution failed" rostering-agent.log

# 2. 查看LLM响应
grep "AI execution completed" rostering-agent.log

# 3. 查看解析错误
grep "Failed to parse AI response" rostering-agent.log
```

### 问题3: 校验失败循环重试

**症状**: 同一个任务重复执行多次

**可能原因**:
1. 排班方案违反规则
2. 人数不足
3. LLM生成的方案不符合要求

**排查步骤**:
```bash
# 1. 查看重试日志
grep "Retry attempt" rostering-agent.log

# 2. 查看失败分析
grep "AI failure analysis" rostering-agent.log

# 3. 查看校验详情
grep "Validation failed" rostering-agent.log
```

---

## 性能优化建议

### 1. 减少LLM调用次数

- **合并相似任务**: 将多个小任务合并为一个大任务
- **优化任务拆分**: 避免将简单任务拆分成过多子任务
- **使用缓存**: 对相似场景的提示词和响应进行缓存

### 2. 优化提示词

- **精简上下文**: 只传递必要的信息
- **压缩人员列表**: 只传递相关人员
- **简化规则描述**: 使用简洁的规则描述

### 3. 并行执行

- **多班次并行**: 不同班次的子任务可以并行执行
- **多任务并行**: 独立的任务可以并行执行

---

## 总结

V3排班系统通过**渐进式任务执行**和**AI智能排班**，实现了灵活、高效的排班方案生成。

**核心优势**:
✅ 任务拆分：复杂需求分解为小任务  
✅ AI驱动：智能理解需求并生成方案  
✅ 自动重试：失败后分析原因并重试  
✅ 监控完善：所有LLM调用都有详细日志  
✅ 类型明确：显式Type字段避免误判  

**执行保证**:
- 每个任务都会正确执行（AI或填充逻辑）
- 所有LLM调用都有监控日志
- 失败任务会自动分析并重试
- 用户可以在每个任务后审查结果
