// Package core 渐进式任务执行核心子工作流
//
// 处理单个渐进式任务的完整执行流程：验证 -> 执行 -> 校验 -> LLMQC -> 完成
package core

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班核心子工作流 V3 定义
// ============================================================

/*
工作流流程：
PreValidate -> Executing -> Validating -> LLMQC -> Completed
                ↓              ↓            ↓
              Failed        Failed      Failed
*/

// init 函数在包被导入时自动注册子工作流
func init() {
	engine.Register(GetSchedulingCoreV3WorkflowDefinition())
}

// GetSchedulingCoreV3WorkflowDefinition 获取排班核心子工作流 V3 定义
func GetSchedulingCoreV3WorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowSchedulingCoreV3,
		InitialState: CoreV3StatePreValidate,
		Transitions:  buildCoreV3Transitions(),

		// 子工作流标记
		IsSubWorkflow: true,

		// 子工作流生命周期钩子
		OnSubWorkflowEnter: onCoreV3Enter,
		OnSubWorkflowExit:  onCoreV3Exit,
	}
}

// buildCoreV3Transitions 构建核心子工作流 V3 的状态转换
func buildCoreV3Transitions() []engine.Transition {
	return []engine.Transition{
		// ============================================================
		// 阶段 1: 预验证
		// ============================================================
		{
			From:       CoreV3StatePreValidate,
			Event:      CoreV3EventStart,
			To:         CoreV3StateExecuting,
			Act:        actPreValidate,
			StateLabel: "验证任务数据",
		},

		// ============================================================
		// 阶段 2: 执行任务
		// ============================================================
		{
			From:       CoreV3StateExecuting,
			Event:      CoreV3EventTaskExecuted,
			To:         CoreV3StateValidating,
			Act:        actExecuteTask,
			StateLabel: "执行渐进式任务",
		},
		{
			From:  CoreV3StateExecuting,
			Event: CoreV3EventFailed,
			To:    CoreV3StateFailed,
			Act:   actOnTaskFailed,
		},

		// ============================================================
		// 阶段 3: 规则级校验
		// ============================================================
		{
			From:       CoreV3StateValidating,
			Event:      CoreV3EventValidationComplete,
			To:         CoreV3StateCompleted, // 完全成功时进入完成状态
			Act:        actValidateResult,
			StateLabel: "规则级校验",
		},
		{
			From:       CoreV3StateValidating,
			Event:      CoreV3EventPartialSuccess,
			To:         CoreV3StatePartialSuccess, // 部分成功时进入部分成功状态
			Act:        actOnPartialSuccess,
			StateLabel: "部分成功处理",
		},
		{
			From:  CoreV3StateValidating,
			Event: CoreV3EventFailed,
			To:    CoreV3StateFailed,
			Act:   actOnValidationFailed,
		},

		// ============================================================
		// 部分成功处理
		// ============================================================
		{
			From:       CoreV3StatePartialSuccess,
			Event:      CoreV3EventRetryFailed,
			To:         CoreV3StatePartialSuccess, // 重试后可能仍部分成功
			Act:        actRetryFailedShifts,
			StateLabel: "重试失败班次",
		},
		{
			From:       CoreV3StatePartialSuccess,
			Event:      CoreV3EventSkipFailed,
			To:         CoreV3StateCompleted,
			Act:        actSkipFailedShifts,
			StateLabel: "跳过失败班次",
		},
		{
			From:       CoreV3StatePartialSuccess,
			Event:      CoreV3EventCancelTask,
			To:         CoreV3StateFailed,
			Act:        actCancelPartialTask,
			StateLabel: "取消任务",
		},

		// ============================================================
		// 阶段 4: LLMQC校验（已注释，暂时不使用）
		// ============================================================
		/*
			{
				From:     CoreV3StateLLMQC,
				Event:    CoreV3EventLLMQCComplete,
				To:       CoreV3StateCompleted,
				Act:      actLLMQC,
				StateLabel: "LLMQC质量检查",
			},
			{
				From:  CoreV3StateLLMQC,
				Event: CoreV3EventFailed,
				To:    CoreV3StateFailed,
				Act:   actOnLLMQCFailed,
			},
		*/

		// ============================================================
		// 完成和失败处理
		// ============================================================
		{
			From:  CoreV3StateCompleted,
			Event: CoreV3EventCompleted,
			To:    CoreV3StateCompleted, // 终态
			Act:   actOnComplete,
		},
		{
			From:  CoreV3StateFailed,
			Event: CoreV3EventRetry,
			To:    CoreV3StatePreValidate,
			Act:   actRetryTask,
		},
		{
			From:  CoreV3StateFailed,
			Event: CoreV3EventSkip,
			To:    CoreV3StateCompleted, // 跳过任务，标记为完成
			Act:   actSkipTask,
		},
	}
}
