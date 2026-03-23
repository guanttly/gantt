# Workflow Common Components

此目录包含工作流系统的通用组件和工具。

## 目录结构

```
common/
├── README.md                       # 本文件
├── reasons.go                      # 失败原因常量定义
├── plan_executor.go                # 通用执行计划框架 - 接口和类型
├── plan_actions.go                 # 通用执行计划框架 - 动作实现
├── PLAN_QUICK_REFERENCE.md         # 执行计划快速参考（⭐ 推荐新手阅读）
├── PLAN_ARCHITECTURE.md            # 执行计划架构说明
├── PLAN_EXAMPLES.md                # 执行计划使用示例
└── PLAN_REFACTORING_SUMMARY.md     # 重构总结文档
```

## 组件说明

### 1. 执行计划框架 (Plan Framework)

**文件**: `plan_executor.go`, `plan_actions.go`

通用的多意图顺序执行框架，适用于所有需要分步执行多个操作的工作流。

**核心接口**:
- `IntentExecutor`: 意图执行器接口，每个工作流需要实现
- `PlanExecutorConfig`: 执行器配置
- `PlanTransitionBuilder`: 状态转换构建器

**主要功能**:
- 意图识别和计划生成
- 用户确认机制
- 顺序执行多个意图
- 统一的错误处理
- 执行状态跟踪

**使用场景**:
- 批量操作（如批量创建、更新、删除）
- 复杂的多步骤流程
- 需要用户确认的操作序列

**快速开始**:
```go
// 1. 实现执行器
type MyExecutor struct{}
func (e *MyExecutor) ExecuteIntent(ctx, dto, wfCtx, intent) error { ... }
func (e *MyExecutor) GetWorkflowType() Workflow { return Workflow_MyWorkflow }

// 2. 配置和构建
config := common.PlanExecutorConfig{
    IntentExecutor: &MyExecutor{},
    // ... 其他配置
}
builder := common.NewPlanTransitionBuilder(config)
transitions := builder.BuildPlanTransitions(...)

// 3. 添加到工作流
func New() *MyWorkflow {
    transitions := initPlanTransitions()
    // ...
}
```

**详细文档**:
- [快速参考](./PLAN_QUICK_REFERENCE.md) ⭐ **推荐新手阅读**
- [架构说明](./PLAN_ARCHITECTURE.md)
- [使用示例](./PLAN_EXAMPLES.md)

### 2. 失败原因常量 (Failure Reasons)

**文件**: `reasons.go`

定义了工作流终止和失败的标准原因常量。

**常量列表**:
- `FinalizeFailReasonUnknown`: 未知原因
- `FinalizeFailReasonTimeout`: 超时
- `FinalizeFailReasonUserCancel`: 用户取消
- `FinalizeFailReasonMissingDependency`: 依赖缺失
- `FinalizeFailReasonStorage`: 存储错误
- `FinalizeFailReasonNetwork`: 网络错误
- `FinalizeFailReasonPermission`: 权限不足
- `FinalizeFailReasonExecError`: 执行错误

**使用示例**:
```go
if err := doSomething(); err != nil {
    return finalize(common.FinalizeFailReasonExecError, err.Error())
}
```

## 已支持的工作流

### 使用执行计划框架的工作流

- ✅ **Dept (部门管理)**: 完全支持，参考实现在 `dept/plan_executor_impl.go`
- 🚧 **Rule (规则管理)**: 部分支持，可迁移
- 🚧 **Schedule (排班)**: 待实现

## 添加新组件

如果你要添加新的通用组件到此目录：

1. **命名规范**
   - 使用小写字母和下划线
   - 文件名应清晰表达组件用途
   - 避免使用缩写

2. **文档要求**
   - 在文件顶部添加包说明
   - 为导出的函数、类型、常量添加注释
   - 提供使用示例
   - 更新本 README

3. **测试要求**
   - 添加单元测试
   - 测试覆盖率 > 80%
   - 添加集成测试（如适用）

4. **示例**
   ```go
   // package_name.go
   package common

   // MyComponent 组件说明
   // 
   // 使用场景：...
   // 
   // 示例:
   //   comp := NewMyComponent()
   //   result := comp.DoSomething()
   type MyComponent struct {
       // ...
   }
   ```

## 设计原则

1. **可复用性**: 组件应该是通用的，适用于多个工作流
2. **可扩展性**: 通过接口和配置支持定制
3. **易用性**: 提供清晰的 API 和完善的文档
4. **可测试性**: 便于单元测试和集成测试
5. **向后兼容**: 修改时考虑现有使用方的兼容性

## 最佳实践

1. **接口设计**
   - 使用接口定义契约
   - 保持接口简洁
   - 遵循 SOLID 原则

2. **错误处理**
   - 返回有意义的错误信息
   - 使用 error wrapping
   - 记录详细日志

3. **性能考虑**
   - 避免不必要的内存分配
   - 使用 sync.Pool 复用对象
   - 考虑并发安全性

4. **文档编写**
   - 代码即文档
   - 提供使用示例
   - 说明设计决策

## 问题反馈

如有问题或建议：
1. 查阅相关文档
2. 检查现有实现
3. 联系架构团队

## 版本历史

- **v1.0** (2025-10-13): 初始版本
  - 添加执行计划框架
  - 添加失败原因常量
  - 完善文档体系
