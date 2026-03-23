package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule_v3.core 子工作流定义
// 用于处理单个渐进式任务的完整执行流程
// ============================================================

const (
	// WorkflowSchedulingCoreV3 排班核心子工作流 V3
	WorkflowSchedulingCoreV3 engine.Workflow = "schedule_v3.core"
)

// ============================================================
// schedule_v3.core 子工作流 - 状态定义
// 命名规范: CoreV3State[StateName]
// ============================================================

const (
	// CoreV3StatePreValidate 预验证状态（子工作流入口）
	CoreV3StatePreValidate engine.State = "_schedule_v3_core_pre_validate_"

	// CoreV3StateExecuting 执行任务状态
	CoreV3StateExecuting engine.State = "_schedule_v3_core_executing_"

	// CoreV3StateValidating 规则级校验状态
	CoreV3StateValidating engine.State = "_schedule_v3_core_validating_"

	// CoreV3StateLLMQC LLMQC校验状态
	CoreV3StateLLMQC engine.State = "_schedule_v3_core_llmqc_"

	// CoreV3StatePartialSuccess 部分成功状态（部分班次成功，部分班次失败）
	CoreV3StatePartialSuccess engine.State = "_schedule_v3_core_partial_success_"

	// CoreV3StateCompleted 完成状态
	CoreV3StateCompleted engine.State = "_schedule_v3_core_completed_"

	// CoreV3StateFailed 失败状态
	CoreV3StateFailed engine.State = "_schedule_v3_core_failed_"
)

// ============================================================
// schedule_v3.core 子工作流 - 事件定义
// 命名规范: CoreV3Event[EventName]
// ============================================================

const (
	// CoreV3EventStart 启动事件（子工作流标准启动事件）
	CoreV3EventStart engine.Event = engine.EventSubWorkflowStart

	// CoreV3EventTaskExecuted 任务执行完成
	CoreV3EventTaskExecuted engine.Event = "_schedule_v3_core_task_executed_"

	// CoreV3EventValidationComplete 规则级校验完成
	CoreV3EventValidationComplete engine.Event = "_schedule_v3_core_validation_complete_"

	// CoreV3EventLLMQCComplete LLMQC校验完成
	CoreV3EventLLMQCComplete engine.Event = "_schedule_v3_core_llmqc_complete_"

	// CoreV3EventCompleted 子工作流完成
	CoreV3EventCompleted engine.Event = "_schedule_v3_core_completed_"

	// CoreV3EventFailed 子工作流失败
	CoreV3EventFailed engine.Event = "_schedule_v3_core_failed_"

	// CoreV3EventRetry 重试任务
	CoreV3EventRetry engine.Event = "_schedule_v3_core_retry_"

	// CoreV3EventSkip 跳过任务
	CoreV3EventSkip engine.Event = "_schedule_v3_core_skip_"

	// CoreV3EventRetryFailed 重试失败的班次
	CoreV3EventRetryFailed engine.Event = "_schedule_v3_core_retry_failed_"

	// CoreV3EventSkipFailed 跳过失败的班次，保存成功部分
	CoreV3EventSkipFailed engine.Event = "_schedule_v3_core_skip_failed_"

	// CoreV3EventCancelTask 取消任务
	CoreV3EventCancelTask engine.Event = "_schedule_v3_core_cancel_task_"

	// CoreV3EventPartialSuccess 任务部分成功（需要用户决策）
	CoreV3EventPartialSuccess engine.Event = "_schedule_v3_core_partial_success_"
)

// ============================================================
// schedule_v3.core 子工作流 - 状态描述（用于前端显示）
// ============================================================

var CoreV3StateDescriptions = map[engine.State]string{
	CoreV3StatePreValidate:    "验证任务数据",
	CoreV3StateExecuting:      "执行渐进式任务",
	CoreV3StateValidating:     "规则级校验",
	CoreV3StateLLMQC:          "LLMQC质量检查",
	CoreV3StatePartialSuccess: "部分班次成功",
	CoreV3StateCompleted:      "任务完成",
	CoreV3StateFailed:         "任务失败",
}

// GetCoreV3StateDescription 获取状态的中文描述
func GetCoreV3StateDescription(state engine.State) string {
	if desc, ok := CoreV3StateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}
