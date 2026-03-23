# 通用执行计划（Plan）架构

## 概述

执行计划（Execution Plan）是一个通用的多意图顺序执行框架，适用于所有需要分步执行多个操作的工作流。

## 架构组件

### 1. 核心接口和类型

**位置**: `internal/workflow/common/`

- **`IntentExecutor`**: 意图执行器接口
  - 每个工作流需要实现此接口来定义如何执行特定类型的意图
  - 方法:
    - `ExecuteIntent()`: 执行单个意图
    - `GetWorkflowType()`: 返回工作流类型

- **`PlanExecutorConfig`**: 计划执行器配置
  - 定义工作流特定的事件和状态
  - 关联意图执行器

- **`PlanTransitionBuilder`**: 计划转换构建器
  - 根据配置自动生成标准的状态转换
  - 提供通用的 action 函数

### 2. 标准执行流程

```
Idle → IntentRecognizing → PlanGenerating → PlanConfirming → Executing → Completed
                                                    ↓
                                                 Failed/Cancelled
```

### 3. 状态说明

- **IntentRecognizing**: 识别用户意图，提取子意图列表
- **PlanGenerating**: 生成执行计划，初始化执行状态
- **PlanConfirming**: 等待用户确认执行计划
- **Executing**: 顺序执行计划中的每个意图
- **Completed**: 所有意图执行完成
- **Failed**: 执行失败
- **Cancelled**: 用户取消

### 4. 事件说明

- **Start**: 启动工作流
- **IntentRecognized**: 意图识别完成
- **PlanReady**: 计划生成完成
- **UserConfirm**: 用户确认执行
- **UserCancel**: 用户取消
- **ExecuteNext**: 执行下一个意图
- **AllCompleted**: 所有意图完成
- **ExecutionFailed**: 执行失败

## 使用方法

### 步骤 1: 实现 IntentExecutor

为你的工作流创建一个执行器：

```go
type MyWorkflowExecutor struct{}

func (e *MyWorkflowExecutor) ExecuteIntent(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
    switch intent.Type {
    case d_model.IntentMyWorkflowAction1:
        return e.executeAction1(ctx, dto, wfCtx, intent)
    case d_model.IntentMyWorkflowAction2:
        return e.executeAction2(ctx, dto, wfCtx, intent)
    default:
        return fmt.Errorf("unsupported intent type: %s", intent.Type)
    }
}

func (e *MyWorkflowExecutor) GetWorkflowType() Workflow {
    return Workflow_MyWorkflow
}
```

### 步骤 2: 配置并构建转换

在工作流定义文件中配置计划执行器：

```go
func initPlanTransitions() []Transition {
    // 创建执行器
    executor := &MyWorkflowExecutor{}

    // 配置
    config := common.PlanExecutorConfig{
        IntentExecutor:       executor,
        EventExecuteNext:     Event_MyWorkflow_ExecuteNext,
        EventAllCompleted:    Event_MyWorkflow_AllCompleted,
        EventExecutionFailed: Event_MyWorkflow_ExecutionFailed,
        StateCompleted:       State_MyWorkflow_Completed,
        StateFailed:          State_MyWorkflow_Failed,
        StateConfirming:      State_MyWorkflow_PlanConfirming,
    }

    // 验证配置
    if err := config.Validate(); err != nil {
        panic(fmt.Sprintf("invalid plan executor config: %v", err))
    }

    // 创建构建器
    builder := common.NewPlanTransitionBuilder(config)

    // 定义子工作流完成状态（可选）
    completedStates := []State{
        State_MyWorkflow_SubAction1_Completed,
        State_MyWorkflow_SubAction2_Completed,
    }

    // 构建转换
    return builder.BuildPlanTransitions(
        State_MyWorkflow_Idle,
        State_MyWorkflow_IntentRecognizing,
        State_MyWorkflow_PlanConfirming,
        State_MyWorkflow_Executing,
        Event_MyWorkflow_Start,
        Event_MyWorkflow_IntentRecognized,
        Event_MyWorkflow_PlanReady,
        completedStates,
    )
}
```

### 步骤 3: 在工作流定义中引用

将生成的转换添加到工作流定义中：

```go
func New() *DeptWorkflow {
    transitions := []Transition{}
    
    // 添加计划执行相关的转换
    transitions = append(transitions, initPlanTransitions()...)
    
    // 添加其他转换
    transitions = append(transitions, initOtherTransitions()...)
    
    return &DeptWorkflow{
        Transitions: transitions,
    }
}
```

## 示例实现

参考 `dept` 工作流的实现：

- **执行器实现**: `internal/workflow/dept/plan_executor_impl.go`
- **工作流定义**: `internal/workflow/dept/definition.go`

## 数据模型

执行计划使用以下数据模型（定义在 `domain/model/session.go`）：

```go
type ExecutionPlan struct {
    PlanID      string              // 计划ID
    Intents     []*IntentResult     // 待执行的意图列表
    Current     int                 // 当前执行到的意图索引
    Status      ExecutionPlanStatus // 状态
    Results     []*IntentExecResult // 执行结果
    FailedIndex int                 // 失败的意图索引
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type IntentExecResult struct {
    IntentIndex int                 // 意图索引
    IntentType  IntentType          // 意图类型
    Status      IntentExecStatus    // 执行状态
    StartedAt   *time.Time          // 开始时间
    CompletedAt *time.Time          // 完成时间
    Error       string              // 错误信息
}
```

## 扩展性

### 自定义执行逻辑

可以在执行器中实现任何自定义逻辑：

- 调用外部服务
- 触发子工作流
- 更新数据库
- 发送通知
- 等待异步操作完成

### 并行执行

当前实现是顺序执行。如需并行执行，可以：

1. 在 `ExecuteIntent` 中启动 goroutine
2. 使用 WaitGroup 等待所有并行任务完成
3. 自定义状态转换逻辑

### 重试机制

可以在执行器中实现重试逻辑：

```go
func (e *MyExecutor) ExecuteIntent(...) error {
    const maxRetries = 3
    var err error
    
    for i := 0; i < maxRetries; i++ {
        err = e.doExecute(...)
        if err == nil {
            return nil
        }
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    return err
}
```

## 最佳实践

1. **意图粒度**: 将复杂操作拆分为多个小意图，便于错误处理和重试
2. **错误处理**: 在执行器中提供详细的错误信息，便于调试
3. **日志记录**: 充分记录执行过程，便于追踪问题
4. **幂等性**: 确保意图执行是幂等的，支持重试
5. **状态同步**: 使用 CAS 操作更新 session 状态，避免并发问题

## 迁移指南

如果你的工作流已经有类似的计划执行逻辑，可以按以下步骤迁移：

1. 创建 `IntentExecutor` 实现，将原有的执行逻辑移到其中
2. 使用 `PlanTransitionBuilder` 替换手工定义的转换
3. 移除重复的 action 函数
4. 测试验证功能正常

## 已支持的工作流

- ✅ Dept (部门管理)
- ✅ Rule (规则管理) - 部分支持
- 🚧 Schedule (排班) - 待实现

## 注意事项

- 通用框架不会自动触发意图识别，需要工作流提供初始输入
- 执行器的实现需要处理所有可能的意图类型
- 状态转换配置必须与工作流的事件定义匹配
