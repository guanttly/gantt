// Package create 排班创建工作流（重构版）
//
// 新架构：Create 作为父工作流，编排三个子工作流
// 1. InfoCollect 子工作流 - 信息收集（周期、班次、人数、人员、规则）
// 2. Core 子工作流 - 核心排班（循环执行每个班次）
// 3. ConfirmSave 子工作流 - 确认保存（预览、确认、保存）
//
// 工作流程:
// [Init] --Start--> [InfoCollecting] --InfoCollected--> [CoreScheduling]
//
//	                                        |
//	循环处理每个班次 <--ShiftCompleted--------+
//	                                        |
//	      --AllShiftsDone--> [ConfirmSaving] --SaveCompleted--> [Completed]
package create

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班创建工作流定义（父工作流）
// ============================================================

// init 函数在包被导入时自动注册工作流
func init() {
	engine.Register(GetScheduleCreateWorkflowDefinition())
}

// GetScheduleCreateWorkflowDefinition 获取排班创建工作流定义
func GetScheduleCreateWorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowScheduleCreate,
		InitialState: CreateStateInit,
		Transitions:  buildCreateTransitions(),
	}
}

// buildCreateTransitions 构建所有状态转换
func buildCreateTransitions() []engine.Transition {
	return []engine.Transition{
		// ========== 初始化阶段 ==========
		// 使用通用 _start_ 事件（与意图映射系统统一）
		{
			From:       CreateStateInit,
			Event:      engine.EventStart,
			To:         CreateStateInfoCollecting,
			StateLabel: "正在收集排班信息",
			Act:        actCreateStartPrepare,
			AfterAct:   actCreateStartSpawnSubWorkflow,
		},
		{
			From:  CreateStateInit,
			Event: CreateEventUserCancel,
			To:    StateCancelled,
			Act:   actCreateCancel,
		},
		// ========== 信息收集阶段（InfoCollect 子工作流） ==========
		{
			From:       CreateStateInfoCollecting,
			Event:      CreateEventInfoCollected,
			To:         CreateStateCoreScheduling,
			StateLabel: "正在生成排班方案",
			Act:        actCreateOnInfoCollected,
			AfterAct:   actCreateSpawnCoreWorkflow, // 状态转换后启动子工作流
		},
		{
			From:  CreateStateInfoCollecting,
			Event: CreateEventSubCancelled,
			To:    StateCancelled,
			Act:   actCreateOnSubCancelled,
		},
		{
			From:  CreateStateInfoCollecting,
			Event: CreateEventSubFailed,
			To:    StateFailed,
			Act:   actCreateOnSubFailed,
		},

		// ========== 核心排班阶段（Core 子工作流循环） ==========
		// 单个班次排班完成，继续下一个班次
		{
			From:       CreateStateCoreScheduling,
			Event:      CreateEventShiftCompleted,
			To:         CreateStateCoreScheduling, // 循环在同一状态
			StateLabel: "正在处理下一个班次",
			Act:        actCreateOnShiftCompleted,
			AfterAct:   actCreateSpawnNextCoreOrComplete, // 状态转换后启动下一个子工作流或完成
		},
		// 所有班次排班完成，进入确认保存阶段
		{
			From:       CreateStateCoreScheduling,
			Event:      CreateEventAllShiftsDone,
			To:         CreateStateConfirmSaving,
			StateLabel: "排班生成完成，请确认",
			Act:        actCreateOnAllShiftsDone,
			AfterAct:   actCreateSpawnConfirmSaveWorkflow, // 状态转换后启动子工作流
		},
		{
			From:  CreateStateCoreScheduling,
			Event: CreateEventSubCancelled,
			To:    StateCancelled,
			Act:   actCreateOnSubCancelled,
		},
		{
			From:  CreateStateCoreScheduling,
			Event: CreateEventSubFailed,
			To:    StateFailed,
			Act:   actCreateOnSubFailed,
		},

		// ========== 确认保存阶段（ConfirmSave 子工作流） ==========
		{
			From:       CreateStateConfirmSaving,
			Event:      CreateEventSaveCompleted,
			To:         StateCompleted,
			StateLabel: "排班保存成功 ✅",
			Act:        actCreateOnSaveCompleted,
		},
		{
			From:  CreateStateConfirmSaving,
			Event: CreateEventSubCancelled,
			To:    StateCancelled,
			Act:   actCreateOnSubCancelled,
		},
		{
			From:  CreateStateConfirmSaving,
			Event: CreateEventSubFailed,
			To:    StateFailed,
			Act:   actCreateOnSubFailed,
		},
	}
}
