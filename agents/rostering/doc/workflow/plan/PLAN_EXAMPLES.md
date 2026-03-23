# 通用 Plan 框架使用示例

本文档提供了如何在不同工作流中使用通用执行计划框架的具体示例。

## 示例 1: Schedule 工作流

### 1. 定义意图执行器

创建文件 `internal/workflow/schedule/plan_executor_impl.go`:

```go
package schedule

import (
	"context"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
	. "jusha/agent/rostering/domain/workflow"
	"jusha/agent/rostering/internal/workflow/common"
)

// ScheduleIntentExecutor 排班工作流的意图执行器
type ScheduleIntentExecutor struct{}

// ExecuteIntent 执行排班相关的意图
func (e *ScheduleIntentExecutor) ExecuteIntent(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	logger := wfCtx.Logger()

	// 根据意图类型执行相应的操作
	switch intent.Type {
	case d_model.IntentScheduleCreate:
		return e.executeScheduleCreate(ctx, dto, wfCtx, intent)
	case d_model.IntentScheduleUpdate:
		return e.executeScheduleUpdate(ctx, dto, wfCtx, intent)
	case d_model.IntentScheduleDelete:
		return e.executeScheduleDelete(ctx, dto, wfCtx, intent)
	case d_model.IntentScheduleQuery:
		return e.executeScheduleQuery(ctx, dto, wfCtx, intent)
	case d_model.IntentScheduleOptimize:
		return e.executeScheduleOptimize(ctx, dto, wfCtx, intent)
	case d_model.IntentScheduleValidate:
		return e.executeScheduleValidate(ctx, dto, wfCtx, intent)
	default:
		logger.Error("unsupported intent type", "type", intent.Type)
		return fmt.Errorf("unsupported intent type: %s", intent.Type)
	}
}

// GetWorkflowType 返回工作流类型
func (e *ScheduleIntentExecutor) GetWorkflowType() Workflow {
	return Workflow_Schedule
}

// 具体执行方法
func (e *ScheduleIntentExecutor) executeScheduleCreate(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule create", "intent", intent.Type)
	
	// 触发排班创建子工作流
	err := wfCtx.Send(ctx, Event_Schedule_Create, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule create workflow: %w", err)
	}
	
	return nil
}

func (e *ScheduleIntentExecutor) executeScheduleUpdate(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule update", "intent", intent.Type)
	
	err := wfCtx.Send(ctx, Event_Schedule_Update, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule update workflow: %w", err)
	}
	
	return nil
}

func (e *ScheduleIntentExecutor) executeScheduleDelete(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule delete", "intent", intent.Type)
	
	err := wfCtx.Send(ctx, Event_Schedule_Delete, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule delete workflow: %w", err)
	}
	
	return nil
}

func (e *ScheduleIntentExecutor) executeScheduleQuery(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule query", "intent", intent.Type)
	
	err := wfCtx.Send(ctx, Event_Schedule_Query, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule query workflow: %w", err)
	}
	
	return nil
}

func (e *ScheduleIntentExecutor) executeScheduleOptimize(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule optimize", "intent", intent.Type)
	
	err := wfCtx.Send(ctx, Event_Schedule_Optimize, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule optimize workflow: %w", err)
	}
	
	return nil
}

func (e *ScheduleIntentExecutor) executeScheduleValidate(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing schedule validate", "intent", intent.Type)
	
	err := wfCtx.Send(ctx, Event_Schedule_Validate, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger schedule validate workflow: %w", err)
	}
	
	return nil
}

// initPlanTransitions 初始化计划相关的状态转换
func initPlanTransitions() []Transition {
	// 创建排班意图执行器
	executor := &ScheduleIntentExecutor{}

	// 配置计划执行器
	config := common.PlanExecutorConfig{
		IntentExecutor:       executor,
		EventExecuteNext:     Event_Schedule_ExecuteNext,
		EventAllCompleted:    Event_Schedule_AllCompleted,
		EventExecutionFailed: Event_Schedule_ExecutionFailed,
		EventPlanReady:       Event_Schedule_PlanReady,
		StateCompleted:       State_Schedule_Completed,
		StateFailed:          State_Schedule_Failed,
		StateConfirming:      State_Schedule_PlanConfirming,
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid plan executor config: %v", err))
	}

	// 创建计划转换构建器
	builder := common.NewPlanTransitionBuilder(config)

	// 子工作流完成状态列表
	completedStates := []State{
		State_Schedule_Create_Completed,
		State_Schedule_Update_Completed,
		State_Schedule_Delete_Completed,
		State_Schedule_Query_Completed,
		State_Schedule_Optimize_Completed,
		State_Schedule_Validate_Completed,
	}

	// 构建计划转换
	return builder.BuildPlanTransitions(
		State_Schedule_Idle,
		State_Schedule_IntentRecognizing,
		State_Schedule_PlanConfirming,
		State_Schedule_Executing,
		Event_Schedule_Start,
		Event_Schedule_IntentRecognized,
		Event_Schedule_PlanReady,
		completedStates,
	)
}
```

### 2. 更新工作流定义

在 `internal/workflow/schedule/definition.go` 中：

```go
func New() *ScheduleWorkflow {
	transitions := []Transition{}
	
	// 添加计划执行相关的转换
	transitions = append(transitions, initPlanTransitions()...)
	
	// 添加其他业务逻辑的转换
	transitions = append(transitions, initScheduleTransitions()...)
	transitions = append(transitions, initQueryTransitions()...)
	transitions = append(transitions, initOptimizeTransitions()...)
	
	return &ScheduleWorkflow{
		Definition: WorkflowDefinition{
			Name:         Workflow_Schedule,
			InitialState: State_Schedule_Idle,
			Transitions:  transitions,
		},
	}
}
```

### 3. 添加必要的事件和状态定义

在 `domain/workflow/event_schedule.go` 中添加：

```go
const (
	// Plan 相关事件
	Event_Schedule_Start             Event = "_event_schedule_start_"
	Event_Schedule_IntentRecognized  Event = "_event_schedule_intent_recognized_"
	Event_Schedule_PlanReady         Event = "_event_schedule_plan_ready_"
	Event_Schedule_ExecuteNext       Event = "_event_schedule_execute_next_"
	Event_Schedule_AllCompleted      Event = "_event_schedule_all_completed_"
	Event_Schedule_ExecutionFailed   Event = "_event_schedule_execution_failed_"
	
	// 子操作事件
	Event_Schedule_Create   Event = "_event_schedule_create_"
	Event_Schedule_Update   Event = "_event_schedule_update_"
	Event_Schedule_Delete   Event = "_event_schedule_delete_"
	Event_Schedule_Query    Event = "_event_schedule_query_"
	Event_Schedule_Optimize Event = "_event_schedule_optimize_"
	Event_Schedule_Validate Event = "_event_schedule_validate_"
)

const (
	// Plan 相关状态
	State_Schedule_Idle               State = "_state_schedule_idle_"
	State_Schedule_IntentRecognizing  State = "_state_schedule_intent_recognizing_"
	State_Schedule_PlanConfirming     State = "_state_schedule_plan_confirming_"
	State_Schedule_Executing          State = "_state_schedule_executing_"
	State_Schedule_Completed          State = "_state_schedule_completed_"
	State_Schedule_Failed             State = "_state_schedule_failed_"
	
	// 子操作完成状态
	State_Schedule_Create_Completed   State = "_state_schedule_create_completed_"
	State_Schedule_Update_Completed   State = "_state_schedule_update_completed_"
	State_Schedule_Delete_Completed   State = "_state_schedule_delete_completed_"
	State_Schedule_Query_Completed    State = "_state_schedule_query_completed_"
	State_Schedule_Optimize_Completed State = "_state_schedule_optimize_completed_"
	State_Schedule_Validate_Completed State = "_state_schedule_validate_completed_"
)
```

## 示例 2: Rule 工作流

Rule 工作流已经有部分 plan 支持，可以通过以下方式进行重构：

### 1. 创建意图执行器

创建文件 `internal/workflow/rule/plan_executor_impl.go`:

```go
package rule

import (
	"context"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
	. "jusha/agent/rostering/domain/workflow"
	"jusha/agent/rostering/internal/workflow/common"
)

// RuleIntentExecutor 规则工作流的意图执行器
type RuleIntentExecutor struct{}

// ExecuteIntent 执行规则相关的意图
func (e *RuleIntentExecutor) ExecuteIntent(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	logger := wfCtx.Logger()

	switch intent.Type {
	case d_model.IntentRuleCreate:
		return e.executeRuleCreate(ctx, dto, wfCtx, intent)
	case d_model.IntentRuleUpdate:
		return e.executeRuleUpdate(ctx, dto, wfCtx, intent)
	case d_model.IntentRuleDelete:
		return e.executeRuleDelete(ctx, dto, wfCtx, intent)
	case d_model.IntentRuleQuery:
		return e.executeRuleQuery(ctx, dto, wfCtx, intent)
	case d_model.IntentRuleExtract:
		return e.executeRuleExtract(ctx, dto, wfCtx, intent)
	default:
		logger.Error("unsupported intent type", "type", intent.Type)
		return fmt.Errorf("unsupported intent type: %s", intent.Type)
	}
}

// GetWorkflowType 返回工作流类型
func (e *RuleIntentExecutor) GetWorkflowType() Workflow {
	return Workflow_Rule
}

func (e *RuleIntentExecutor) executeRuleCreate(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing rule create", "intent", intent.Type)
	err := wfCtx.Send(ctx, Event_Rule_Create, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger rule create workflow: %w", err)
	}
	return nil
}

func (e *RuleIntentExecutor) executeRuleUpdate(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing rule update", "intent", intent.Type)
	err := wfCtx.Send(ctx, Event_Rule_Update, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger rule update workflow: %w", err)
	}
	return nil
}

func (e *RuleIntentExecutor) executeRuleDelete(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing rule delete", "intent", intent.Type)
	err := wfCtx.Send(ctx, Event_Rule_Delete, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger rule delete workflow: %w", err)
	}
	return nil
}

func (e *RuleIntentExecutor) executeRuleQuery(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing rule query", "intent", intent.Type)
	err := wfCtx.Send(ctx, Event_Rule_Query, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger rule query workflow: %w", err)
	}
	return nil
}

func (e *RuleIntentExecutor) executeRuleExtract(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	wfCtx.Logger().Info("executing rule extract", "intent", intent.Type)
	err := wfCtx.Send(ctx, Event_Rule_Extract, intent)
	if err != nil {
		return fmt.Errorf("failed to trigger rule extract workflow: %w", err)
	}
	return nil
}

// initPlanTransitions 初始化计划相关的状态转换
func initPlanTransitions() []Transition {
	executor := &RuleIntentExecutor{}

	config := common.PlanExecutorConfig{
		IntentExecutor:       executor,
		EventExecuteNext:     Event_Rule_ExecuteNext,
		EventAllCompleted:    Event_Rule_AllCompleted,
		EventExecutionFailed: Event_Rule_ExecutionFailed,
		EventPlanReady:       Event_Rule_PlanReady,
		StateCompleted:       State_Rule_Completed,
		StateFailed:          State_Rule_Failed,
		StateConfirming:      State_Rule_PlanConfirming,
	}

	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid plan executor config: %v", err))
	}

	builder := common.NewPlanTransitionBuilder(config)

	completedStates := []State{
		State_Rule_Create_Completed,
		State_Rule_Update_Completed,
		State_Rule_Delete_Completed,
		State_Rule_Query_Completed,
		State_Rule_Extract_Completed,
	}

	return builder.BuildPlanTransitions(
		State_Rule_Idle,
		State_Rule_IntentRecognizing,
		State_Rule_PlanConfirming,
		State_Rule_Executing,
		Event_Rule_Start,
		Event_Rule_IntentRecognized,
		Event_Rule_PlanReady,
		completedStates,
	)
}
```

## 示例 3: 自定义执行逻辑

有时你可能需要在执行意图时添加额外的业务逻辑：

```go
func (e *MyExecutor) ExecuteIntent(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	logger := wfCtx.Logger()
	
	// 执行前验证
	if err := e.validateIntent(intent); err != nil {
		logger.Error("intent validation failed", "error", err)
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// 记录执行开始
	logger.Info("starting intent execution", 
		"type", intent.Type,
		"confidence", intent.Confidence,
		"summary", intent.Summary)
	
	// 执行意图
	var err error
	switch intent.Type {
	case d_model.IntentMyAction1:
		err = e.executeAction1(ctx, dto, wfCtx, intent)
	case d_model.IntentMyAction2:
		err = e.executeAction2(ctx, dto, wfCtx, intent)
	default:
		return fmt.Errorf("unsupported intent type: %s", intent.Type)
	}
	
	if err != nil {
		// 执行失败，记录详细信息
		logger.Error("intent execution failed",
			"type", intent.Type,
			"error", err)
		
		// 可以在这里添加重试逻辑
		if e.shouldRetry(err) {
			logger.Info("retrying intent execution")
			err = e.retryExecution(ctx, dto, wfCtx, intent)
		}
		
		return err
	}
	
	// 执行成功，添加后续处理
	logger.Info("intent execution completed successfully", "type", intent.Type)
	
	// 可以发送通知或更新相关状态
	e.notifyCompletion(ctx, dto, intent)
	
	return nil
}

func (e *MyExecutor) validateIntent(intent *d_model.IntentResult) error {
	if intent.Confidence < 0.5 {
		return fmt.Errorf("intent confidence too low: %.2f", intent.Confidence)
	}
	return nil
}

func (e *MyExecutor) shouldRetry(err error) bool {
	// 判断错误类型是否应该重试
	return isTemporaryError(err)
}

func (e *MyExecutor) retryExecution(ctx context.Context, dto *d_model.SessionDTO, wfCtx IWFContext, intent *d_model.IntentResult) error {
	// 实现重试逻辑
	time.Sleep(time.Second)
	return e.ExecuteIntent(ctx, dto, wfCtx, intent)
}

func (e *MyExecutor) notifyCompletion(ctx context.Context, dto *d_model.SessionDTO, intent *d_model.IntentResult) {
	// 发送完成通知
	// ...
}
```

## 测试建议

### 单元测试执行器

```go
func TestMyExecutor_ExecuteIntent(t *testing.T) {
	executor := &MyExecutor{}
	
	tests := []struct {
		name        string
		intentType  d_model.IntentType
		expectError bool
	}{
		{
			name:        "valid action 1",
			intentType:  d_model.IntentMyAction1,
			expectError: false,
		},
		{
			name:        "unsupported action",
			intentType:  d_model.IntentType("unknown"),
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := &d_model.IntentResult{
				Type:       tt.intentType,
				Confidence: 0.9,
				Summary:    "test intent",
			}
			
			// 创建模拟的上下文
			mockCtx := &MockWFContext{}
			mockDTO := &d_model.SessionDTO{ID: "test-session"}
			
			err := executor.ExecuteIntent(context.Background(), mockDTO, mockCtx, intent)
			
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
```

### 集成测试

```go
func TestPlanExecution_Integration(t *testing.T) {
	// 设置测试环境
	store := setupTestStore()
	wfCtx := setupTestWFContext(store)
	
	// 创建测试 session
	dto := &d_model.SessionDTO{
		ID:    "test-session",
		State: string(State_MyWorkflow_Idle),
	}
	store.Set(dto)
	
	// 创建意图列表
	intents := []*d_model.IntentResult{
		{Type: d_model.IntentMyAction1, Confidence: 0.9},
		{Type: d_model.IntentMyAction2, Confidence: 0.8},
	}
	
	// 创建执行器和构建器
	executor := &MyExecutor{}
	config := common.PlanExecutorConfig{
		IntentExecutor:    executor,
		// ... 其他配置
	}
	builder := common.NewPlanTransitionBuilder(config)
	
	// 测试计划生成
	err := builder.actGeneratePlan(context.Background(), dto, wfCtx, intents)
	assert.NoError(t, err)
	
	// 验证计划已创建
	updatedDTO := store.Get(dto.ID)
	assert.NotNil(t, updatedDTO.ExecutionPlan)
	assert.Equal(t, 2, len(updatedDTO.ExecutionPlan.Intents))
	
	// 测试计划执行
	// ...
}
```

## 总结

通过使用通用 Plan 框架，你可以：

1. **减少重复代码**: 所有工作流共享相同的计划执行逻辑
2. **提高一致性**: 用户体验在不同工作流中保持一致
3. **易于维护**: 修改一处即可影响所有工作流
4. **灵活扩展**: 通过接口定义，可以轻松添加自定义行为

记住关键步骤：
1. 实现 `IntentExecutor` 接口
2. 配置 `PlanExecutorConfig`
3. 使用 `PlanTransitionBuilder` 生成转换
4. 添加到工作流定义中
