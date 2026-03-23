# Rule Workflow Implementation

## 概述

本目录包含规则相关的工作流实现，规则工作流负责处理排班规则的提取、查询、更新和删除等操作，实际的存储和读取通过relational-graph-server的MCP工具完成。

**✨ 新特性：支持多意图顺序执行**

规则工作流现已集成通用执行计划（Plan）框架，支持一次处理多个规则操作。例如：
- "创建值班规则A，然后删除规则B"
- "查询当前所有规则，然后更新规则C"

详见 [计划执行](#计划执行plan) 章节。

## 文件结构

- `definition.go` - 工作流定义和注册
- `plan_executor_impl.go` - **计划执行器实现（新增）**
- `actions_extract.go` - 规则提取工作流实现
- `actions_query.go` - 规则查询工作流实现
- `actions_delete.go` - 规则删除工作流实现

## 工作流说明

### 1. 规则提取工作流 (Extract)

负责从自然语言文本中提取排班规则，并存储到图数据库中。

**状态流转:**
```
Idle -> Extracting -> Checking -> Confirming -> Executing -> Completed
                         |              |
                         |              +-> Failed (取消)
                         +-> ConflictFound (发现冲突)
```

**关键步骤:**
1. 接收用户输入的规则文本
2. 调用relational-graph-server的`scheduling_rules_upsert`工具提取规则
3. 检查是否存在冲突
4. 如有冲突，提示用户选择是否强制更新
5. 用户确认后存储规则

**使用的MCP工具:**
- `scheduling_rules_upsert` - 提取并存储规则

### 2. 规则查询工作流 (Query)

负责从图数据库中查询排班规则。

**状态流转:**
```
Idle -> Querying -> Completed
           |
           +-> Failed
```

**关键步骤:**
1. 接收查询参数（查询类型、实体等）
2. 调用relational-graph-server的`scheduling_query`工具查询规则
3. 返回查询结果

**使用的MCP工具:**
- `scheduling_query` - 查询规则

### 3. 规则删除工作流 (Delete)

负责从图数据库中删除排班规则。

**状态流转:**
```
Idle -> Checking -> Confirming -> Executing -> Completed
                        |              |
                        |              +-> Failed
                        +-> Failed (取消)
```

**关键步骤:**
1. 接收要删除的规则ID或名称
2. 检查规则是否存在
3. 提示用户确认删除
4. 用户确认后调用relational-graph-server的`scheduling_rules_delete`工具删除规则
5. 返回删除结果

**使用的MCP工具:**
- `scheduling_rules_delete` - 删除规则

## 待完成事项 (TODO)

### 1. RelationalGraphGateway注入 ⚠️

当前`getRelGraphGateway`函数返回nil，需要实现真实的gateway注入逻辑。

**建议方案:**

#### 方案A: 修改IWFContext接口（推荐）
```go
// 在actor_system.go中
type IWFContext interface {
    // ... 现有方法
    IRelationalGraphGateway() IRelationalGraphGateway  // 新增
}

// 在actor.go中
type Actor struct {
    // ... 现有字段
    relGraphGateway IRelationalGraphGateway  // 新增
}

func NewActor(..., relGraphGateway IRelationalGraphGateway, ...) *Actor {
    // ... 注入relGraphGateway
}

func (a *Actor) IRelationalGraphGateway() IRelationalGraphGateway {
    return a.relGraphGateway
}
```

然后在actions文件中修改：
```go
func getRelGraphGateway(wfCtx IWFContext) mcp.IRelationalGraphGateway {
    return wfCtx.IRelationalGraphGateway()
}
```

#### 方案B: 使用全局服务定位器
创建一个全局的服务注册表，在启动时注册gateway，在使用时获取。

#### 方案C: 通过Context传递
在workflow的Extra字段中传递gateway引用（不推荐，容易引起内存泄漏）

### 2. 规则更新工作流

实现规则更新功能，允许修改已存在的规则。

### 3. 规则验证工作流

实现规则验证功能，检查规则是否合法、是否存在逻辑冲突等。

### 4. 单元测试

为各个工作流添加单元测试。

### 5. 集成测试

添加与relational-graph-server的集成测试。

## 依赖关系

```
rule workflow
    |
    +-- relational-graph-server (MCP Server)
    |       |
    |       +-- scheduling_rules_upsert
    |       +-- scheduling_rules_delete  
    |       +-- scheduling_query
    |
    +-- workflow engine
    |       |
    |       +-- Actor
    |       +-- MessageBuilder
    |
    +-- domain models
            |
            +-- SessionDTO
            +-- IntentResult
            +-- WorkflowMeta
```

## 使用示例

### 规则提取
```
用户: "周末至少要有2个人值班"
系统: [触发规则提取工作流]
      [调用scheduling_rules_upsert提取规则]
      [检查冲突]
      [提示用户确认]
用户: [确认]
系统: [存储规则到图数据库]
      [返回提取结果]
```

### 规则查询
```
用户: "查询所有周末相关的规则"
系统: [触发规则查询工作流]
      [调用scheduling_query查询]
      [返回查询结果]
```

### 规则删除
```
用户: "删除规则ID为rule-123的规则"
系统: [触发规则删除工作流]
      [检查规则存在性]
      [提示用户确认]
用户: [确认]
系统: [调用scheduling_rules_delete删除]
      [返回删除结果]
```

## 注意事项

1. **错误处理**: 所有与relational-graph-server的交互都应该有完善的错误处理
2. **用户确认**: 对于可能影响数据的操作（删除、强制更新等），必须先获取用户确认
3. **日志记录**: 关键操作都应记录日志，便于问题排查
4. **异步处理**: 耗时操作应该异步执行，避免阻塞工作流

## 计划执行（Plan）

### 概述

规则工作流现已支持**多意图顺序执行**功能，通过通用执行计划框架实现。当用户一次性提出多个规则操作请求时，系统会：

1. 识别所有子意图
2. 生成执行计划
3. 请求用户确认
4. 顺序执行每个操作
5. 返回汇总结果

### 支持的意图类型

- `rule.create` - 创建规则（通过提取工作流）
- `rule.update` - 更新规则
- `rule.query` - 查询规则
- `rule.delete` - 删除规则

### 执行流程

```
用户输入多个操作
    ↓
意图识别 (AnalyzeSubIntents)
    ↓
生成执行计划 (ExecutionPlan)
    ↓
等待用户确认 (PlanConfirming)
    ↓
用户确认 / 取消
    ↓
顺序执行意图 (Executing)
    ↓  ↓  ↓  ↓
   意图1 意图2 意图3 意图4
    ↓
完成 / 失败
```

### 状态说明

- **Idle**: 空闲状态，等待触发
- **IntentRecognizing**: 正在识别子意图
- **PlanConfirming**: 等待用户确认执行计划
- **Executing**: 正在顺序执行意图
- **Completed**: 所有意图执行完成
- **Failed**: 执行失败

### 使用示例

#### 示例 1: 批量创建规则

```
用户: "创建两个规则：1. 周末至少2人值班 2. 夜班必须有1名主治医师"

系统: [识别到2个意图: rule.create, rule.create]
      [生成执行计划]
      
      检测到 2 个操作需要执行:
      1. rule.create - 周末至少2人值班 (置信度: 0.95)
      2. rule.create - 夜班必须有1名主治医师 (置信度: 0.92)
      
      是否确认执行这些操作?
      [确认执行] [取消]

用户: [点击确认执行]

系统: [开始执行]
      执行第1个操作: 周末至少2人值班 ✓
      执行第2个操作: 夜班必须有1名主治医师 ✓
      
      执行计划完成！成功执行 2/2 个操作。
```

#### 示例 2: 查询后删除

```
用户: "查询所有节假日规则，然后删除rule-123"

系统: [识别到2个意图: rule.query, rule.delete]
      [生成执行计划]
      
      检测到 2 个操作需要执行:
      1. rule.query - 查询所有节假日规则 (置信度: 0.90)
      2. rule.delete - 删除rule-123 (置信度: 0.88)
      
      是否确认执行这些操作?
      [确认执行] [取消]

用户: [点击确认执行]

系统: [开始执行]
      执行第1个操作: 查询所有节假日规则 ✓
      [显示查询结果]
      
      执行第2个操作: 删除rule-123
      [触发删除确认流程]
      ...
```

#### 示例 3: 用户取消

```
用户: "创建规则A、更新规则B、删除规则C"

系统: [识别到3个意图]
      [生成执行计划]
      
      检测到 3 个操作需要执行:
      1. rule.create - 创建规则A (置信度: 0.93)
      2. rule.update - 更新规则B (置信度: 0.89)
      3. rule.delete - 删除规则C (置信度: 0.91)
      
      是否确认执行这些操作?
      [确认执行] [取消]

用户: [点击取消]

系统: 已取消执行计划
```

### 实现细节

#### RuleIntentExecutor

执行器实现在 `plan_executor_impl.go` 中：

```go
type RuleIntentExecutor struct{}

func (e *RuleIntentExecutor) ExecuteIntent(ctx, dto, wfCtx, intent) error {
    switch intent.Type {
    case d_model.IntentRuleCreate:
        return e.executeRuleCreate(...)
    case d_model.IntentRuleUpdate:
        return e.executeRuleUpdate(...)
    case d_model.IntentRuleQuery:
        return e.executeRuleQuery(...)
    case d_model.IntentRuleDelete:
        return e.executeRuleDelete(...)
    }
}
```

每个意图执行方法会触发相应的子工作流：
- `executeRuleCreate` → 触发 `Event_Rule_Extract`
- `executeRuleUpdate` → 触发 `Event_Rule_Update`
- `executeRuleQuery` → 触发 `Event_Rule_Query`
- `executeRuleDelete` → 触发 `Event_Rule_Delete`

#### 配置

```go
config := common.PlanExecutorConfig{
    IntentExecutor:       &RuleIntentExecutor{},
    EventExecuteNext:     Event_Rule_ExecuteNext,
    EventAllCompleted:    Event_Rule_AllCompleted,
    EventExecutionFailed: Event_Rule_ExecutionFailed,
    EventPlanReady:       Event_Rule_PlanReady,
    StateCompleted:       State_Rule_Completed,
    StateFailed:          State_Rule_Failed,
    StateConfirming:      State_Rule_PlanConfirming,
}
```

### 与单意图工作流的关系

- **单意图场景**: 直接触发相应的子工作流（Extract、Query、Delete等）
- **多意图场景**: 通过计划执行框架，顺序触发多个子工作流

两者可以共存，不会相互影响。

### 扩展指南

如果需要添加新的规则操作类型：

1. **定义意图类型** (`domain/model/intent.go`)
   ```go
   IntentRuleValidate IntentType = "rule.validate"
   ```

2. **添加事件和状态** (`domain/workflow/event_rule.go`)
   ```go
   Event_Rule_Validate = "_event_rule_validate_"
   State_Rule_Validate_Completed State = "_state_rule_validate_completed_"
   ```

3. **实现执行方法** (`plan_executor_impl.go`)
   ```go
   func (e *RuleIntentExecutor) executeRuleValidate(...) error {
       return wfCtx.Send(ctx, Event_Rule_Validate, intent)
   }
   ```

4. **更新 switch 语句**
   ```go
   case d_model.IntentRuleValidate:
       return e.executeRuleValidate(...)
   ```

5. **添加完成状态到列表**
   ```go
   completedStates := []State{
       State_Rule_Validate_Completed,
       // ...
   }
   ```

### 相关文档

- [通用计划框架快速参考](../common/PLAN_QUICK_REFERENCE.md)
- [计划框架架构说明](../common/PLAN_ARCHITECTURE.md)
- [计划框架使用示例](../common/PLAN_EXAMPLES.md)
- [重构总结](../common/PLAN_REFACTORING_SUMMARY.md)
5. **状态管理**: 使用UpdateSessionCAS确保状态更新的原子性

## 相关文档

- [Dept Workflow](../dept/README.md) - 部门工作流参考实现
- [Workflow Engine](../engine/README.md) - 工作流引擎文档
- [MCP Gateway](../../infrastructure/mcp/README.md) - MCP网关文档
- [Relational Graph Server](../../../../mcp-servers/relational-graph-server/README.md) - 关系图服务器文档
