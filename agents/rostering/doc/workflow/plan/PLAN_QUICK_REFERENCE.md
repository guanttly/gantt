# 通用执行计划框架 - 快速参考

## 30 秒快速上手

```go
// Step 1: 实现执行器
type MyExecutor struct{}

func (e *MyExecutor) ExecuteIntent(ctx context.Context, dto *d_model.SessionDTO, 
    wfCtx IWFContext, intent *d_model.IntentResult) error {
    switch intent.Type {
    case d_model.IntentMyAction:
        return wfCtx.Send(ctx, Event_My_Action, intent)
    default:
        return fmt.Errorf("unsupported intent: %s", intent.Type)
    }
}

func (e *MyExecutor) GetWorkflowType() Workflow {
    return Workflow_MyWorkflow
}

// Step 2: 配置并生成转换
func initPlanTransitions() []Transition {
    config := common.PlanExecutorConfig{
        IntentExecutor:       &MyExecutor{},
        EventExecuteNext:     Event_My_ExecuteNext,
        EventAllCompleted:    Event_My_AllCompleted,
        EventExecutionFailed: Event_My_ExecutionFailed,
        EventPlanReady:       Event_My_PlanReady,
        StateCompleted:       State_My_Completed,
        StateFailed:          State_My_Failed,
        StateConfirming:      State_My_PlanConfirming,
    }
    
    builder := common.NewPlanTransitionBuilder(config)
    
    return builder.BuildPlanTransitions(
        State_My_Idle,           // 初始状态
        State_My_Recognizing,    // 识别状态
        State_My_PlanConfirming, // 确认状态
        State_My_Executing,      // 执行状态
        Event_My_Start,          // 启动事件
        Event_My_IntentRecognized, // 识别完成事件
        Event_My_PlanReady,      // 计划就绪事件
        []State{                 // 子流程完成状态
            State_My_Action_Completed,
        },
    )
}
```

## 核心概念

### 1. IntentExecutor 接口
负责执行具体的意图操作。

**方法**:
- `ExecuteIntent(...)`: 执行单个意图
- `GetWorkflowType()`: 返回工作流类型

### 2. PlanExecutorConfig 配置
定义工作流特定的事件和状态。

**必填字段**:
| 字段 | 说明 |
|------|------|
| IntentExecutor | 执行器实现 |
| EventExecuteNext | 执行下一个意图 |
| EventAllCompleted | 全部完成 |
| EventExecutionFailed | 执行失败 |
| EventPlanReady | 计划就绪 |
| StateCompleted | 完成状态 |
| StateFailed | 失败状态 |
| StateConfirming | 确认状态 |

### 3. PlanTransitionBuilder 构建器
根据配置生成标准的状态转换。

**方法**:
- `BuildPlanTransitions(...)`: 构建转换列表

## 标准执行流程

```
Idle → IntentRecognizing → PlanGenerating → PlanConfirming
                                                ↓
                                          用户确认/取消
                                                ↓
                                            Executing
                                         (循环执行意图)
                                                ↓
                                        Completed/Failed
```

## 提供的自动化功能

✅ 意图识别和分析  
✅ 执行计划生成  
✅ 用户确认界面  
✅ 顺序执行控制  
✅ 错误处理和回滚  
✅ 执行状态追踪  
✅ 结果收集和报告  

## 你只需要实现

❗ `ExecuteIntent` 方法 - 定义如何执行每种意图类型

```go
func (e *MyExecutor) ExecuteIntent(...) error {
    // 1. 可选：执行前验证
    if err := validate(intent); err != nil {
        return err
    }
    
    // 2. 根据类型执行操作
    switch intent.Type {
    case TypeA:
        return e.doA(...)
    case TypeB:
        return e.doB(...)
    default:
        return fmt.Errorf("unsupported type")
    }
}
```

## 常见执行模式

### 模式 1: 触发子工作流
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    return wfCtx.Send(ctx, Event_SubWorkflow, intent)
}
```

### 模式 2: 直接调用服务
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    svc := wfCtx.IDataService()
    _, err := svc.CreateItem(ctx, buildRequest(intent))
    return err
}
```

### 模式 3: 带重试逻辑
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    var err error
    for i := 0; i < 3; i++ {
        err = e.doExecute(...)
        if err == nil {
            return nil
        }
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return err
}
```

## 数据模型

框架使用 `domain/model/session.go` 中的模型：

```go
type ExecutionPlan struct {
    PlanID      string              // 计划ID
    Intents     []*IntentResult     // 意图列表
    Current     int                 // 当前索引
    Status      ExecutionPlanStatus // 状态
    Results     []*IntentExecResult // 结果列表
    FailedIndex int                 // 失败索引
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 状态码

### ExecutionPlanStatus
- `pending`: 待确认
- `executing`: 执行中
- `completed`: 已完成
- `failed`: 执行失败
- `cancelled`: 已取消

### IntentExecStatus
- `pending`: 待执行
- `executing`: 执行中
- `completed`: 已完成
- `failed`: 执行失败

## 调试技巧

### 1. 添加日志
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    logger := wfCtx.Logger()
    logger.Info("executing intent", 
        "type", intent.Type,
        "confidence", intent.Confidence)
    
    // 执行逻辑...
    
    logger.Info("intent completed")
    return nil
}
```

### 2. 检查执行计划
```go
if dto.ExecutionPlan != nil {
    logger.Info("plan status",
        "planID", dto.ExecutionPlan.PlanID,
        "current", dto.ExecutionPlan.Current,
        "total", len(dto.ExecutionPlan.Intents))
}
```

### 3. 查看执行结果
```go
for i, result := range dto.ExecutionPlan.Results {
    logger.Info("result", 
        "index", i,
        "type", result.IntentType,
        "status", result.Status,
        "error", result.Error)
}
```

## 完整示例参考

查看 `dept/plan_executor_impl.go` 获取完整实现示例。

## 需要帮助？

1. 📖 阅读 [PLAN_ARCHITECTURE.md](./PLAN_ARCHITECTURE.md)
2. 💡 查看 [PLAN_EXAMPLES.md](./PLAN_EXAMPLES.md)
3. 📝 参考 [PLAN_REFACTORING_SUMMARY.md](./PLAN_REFACTORING_SUMMARY.md)
4. 🔍 检查 dept 工作流的实现
5. 💬 联系架构团队

## 检查清单

在完成实现后，检查：

- [ ] 实现了 `IntentExecutor` 接口
- [ ] 处理了所有可能的意图类型
- [ ] 配置了所有必填字段
- [ ] 调用了 `Validate()` 验证配置
- [ ] 添加了适当的日志记录
- [ ] 处理了错误情况
- [ ] 添加了单元测试
- [ ] 更新了工作流定义
- [ ] 添加了事件和状态定义
- [ ] 更新了状态描述映射

## 常见错误

### ❌ 忘记验证配置
```go
config := common.PlanExecutorConfig{...}
// 缺少验证！
builder := common.NewPlanTransitionBuilder(config)
```

**✅ 正确做法**:
```go
config := common.PlanExecutorConfig{...}
if err := config.Validate(); err != nil {
    panic(fmt.Sprintf("invalid config: %v", err))
}
builder := common.NewPlanTransitionBuilder(config)
```

### ❌ 未处理所有意图类型
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    switch intent.Type {
    case TypeA:
        return e.doA(...)
    // 缺少 default 分支！
    }
}
```

**✅ 正确做法**:
```go
func (e *MyExecutor) ExecuteIntent(...) error {
    switch intent.Type {
    case TypeA:
        return e.doA(...)
    default:
        return fmt.Errorf("unsupported intent: %s", intent.Type)
    }
}
```

### ❌ 忘记添加状态描述
```go
// 定义了状态但没有添加描述映射
State_My_PlanConfirming State = "_state_my_plan_confirming_"
```

**✅ 正确做法**:
```go
// 在状态描述映射中添加
var stateDescriptions = map[string]map[string]string{
    string(State_My_PlanConfirming): {
        "zh-CN": "待确认执行计划",
        "en-US": "Confirming Plan",
    },
}
```

---

**版本**: v1.0  
**更新**: 2025-10-13  
**维护**: 架构团队
