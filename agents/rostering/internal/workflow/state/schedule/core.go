// Package schedule 定义排班相关工作流的状态和事件
package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// 排班核心子工作流定义
// 可被 Create 和 Adjust 工作流调用
// ============================================================

const (
	// WorkflowSchedulingCore 排班核心子工作流
	WorkflowSchedulingCore engine.Workflow = "schedule.core"
)

// ============================================================
// schedule.core 子工作流 - 状态定义
// 命名规范: _[workflow]_[sub]_[state]_
// ============================================================

const (
	// 阶段1: 验证中
	CoreStateValidating engine.State = "_schedule_core_validating_"

	// 阶段2: 生成 Todo 计划
	CoreStateGeneratingTodo engine.State = "_schedule_core_generating_todo_"

	// 阶段3: 执行 Todo 计划
	CoreStateExecutingTodo engine.State = "_schedule_core_executing_todo_"

	// 阶段4: 优化结果
	CoreStateRefiningResult engine.State = "_schedule_core_refining_result_"

	// 终态
	CoreStateCompleted engine.State = "_schedule_core_completed_"
	CoreStateFailed    engine.State = "_schedule_core_failed_"

	// 新状态梳理重构
	CoreStatePreValidate     engine.State = "_schedule_core_pre_validate_"
	CoreStateUserInteraction engine.State = "_schedule_core_user_interaction_"
)

// ============================================================
// schedule.core 子工作流 - 事件定义
// ============================================================

const (
	// 启动事件
	CoreEventStart engine.Event = "_schedule_core_start_"

	// 生成 Todo 计划相关
	CoreEventTodoGenerated engine.Event = "_schedule_core_todo_generated_"

	// 执行 Todo 相关
	CoreEventExecuteTodo  engine.Event = "_schedule_core_execute_todo_"
	CoreEventTodoComplete engine.Event = "_schedule_core_todo_complete_"

	// 优化相关
	CoreEventRefine  engine.Event = "_schedule_core_refine_"
	CoreEventRefined engine.Event = "_schedule_core_refined_"

	// 错误和返回
	CoreEventError  engine.Event = "_schedule_core_error_"
	CoreEventReturn engine.Event = "_schedule_core_return_"

	// 错误恢复事件（新增）
	CoreEventRetry engine.Event = "_schedule_core_retry_" // 重试当前班次
	CoreEventSkip  engine.Event = "_schedule_core_skip_"  // 跳过当前班次

	// new
	CoreEventUserConfirmed engine.Event = engine.EventConfirm
	CoreEventUserCancelled engine.Event = engine.EventCancel
	CoreEventValidate      engine.Event = "_schedule_core_validate_"
	CoreEventAdjustPlan    engine.Event = "_schedule_core_adjust_plan_"
	CoreEventUserRequest   engine.Event = "_schedule_core_user_request_"
)

// ============================================================
// schedule.core 子工作流 - 状态描述
// ============================================================

var CoreStateDescriptions = map[engine.State]string{
	CoreStateGeneratingTodo: "生成排班计划",
	CoreStateExecutingTodo:  "执行排班任务",
	CoreStateRefiningResult: "优化排班结果",
	CoreStateCompleted:      "排班核心完成",
	CoreStateFailed:         "排班核心失败",
}

// GetCoreStateDescription 获取核心子工作流状态描述
func GetCoreStateDescription(state engine.State) string {
	if desc, ok := CoreStateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}

// ============================================================
// 子工作流调用配置
// ============================================================

// CoreSubWorkflowConfig 核心子工作流调用配置
// 用于父工作流调用核心子工作流时的标准配置
type CoreSubWorkflowConfig struct {
	// 输入数据
	ShiftID    string `json:"shift_id"`    // 班次ID
	DateRange  string `json:"date_range"`  // 日期范围
	StaffCount int    `json:"staff_count"` // 所需人数

	// 回调事件
	OnComplete engine.Event `json:"on_complete"` // 成功回调事件
	OnError    engine.Event `json:"on_error"`    // 失败回调事件
}

// DefaultCoreConfig 默认核心子工作流配置
func DefaultCoreConfig(onComplete, onError engine.Event) *engine.SubWorkflowConfig {
	return &engine.SubWorkflowConfig{
		WorkflowName: WorkflowSchedulingCore,
		OnComplete:   onComplete,
		OnError:      onError,
		Timeout:      0, // 无限等待
		SnapshotKeys: []string{
			"shift_scheduling_context", // 快照排班上下文以便回滚
		},
	}
}
