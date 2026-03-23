// Package create 排班创建工作流 V4
//
// V4排班架构：基于确定性规则引擎，减少LLM调用
// 流程：信息收集 -> 个人需求 -> 规则组织 -> 确定性预计算 -> LLM排班决策 -> 确定性校验 -> 确认保存
package create

import (
	"jusha/mcp/pkg/workflow/engine"

	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 排班创建工作流 V4 定义（父工作流）
// ============================================================

// init 函数在包被导入时自动注册工作流
func init() {
	engine.Register(GetScheduleCreateV4WorkflowDefinition())
}

// GetScheduleCreateV4WorkflowDefinition 获取排班创建工作流 V4 定义
func GetScheduleCreateV4WorkflowDefinition() *engine.WorkflowDefinition {
	return &engine.WorkflowDefinition{
		Name:         WorkflowScheduleCreateV4,
		InitialState: CreateV4StateInit,
		Transitions:  buildCreateV4Transitions(),
	}
}

// buildCreateV4Transitions 构建所有状态转换
func buildCreateV4Transitions() []engine.Transition {
	return []engine.Transition{
		// ============================================================
		// 阶段 0: 初始化
		// ============================================================
		{
			From:       CreateV4StateInit,
			Event:      CreateV4EventStart,
			To:         CreateV4StateInfoCollecting,
			StateLabel: "收集信息",
			Act:        actStartInfoCollect,
		},
		{
			From:  CreateV4StateInit,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 1: 信息收集（复用V3逻辑）
		// ============================================================
		{
			From:  CreateV4StateInfoCollecting,
			Event: CreateV4EventInfoCollected,
			To:    CreateV4StateConfirmPeriod,
			Act:   actOnInfoCollected,
		},
		{
			From:  CreateV4StateInfoCollecting,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 2: 确认排班时间
		// ============================================================
		{
			From:  CreateV4StateConfirmPeriod,
			Event: CreateV4EventPeriodConfirmed,
			To:    CreateV4StateConfirmShifts,
			Act:   actOnPeriodConfirmed,
		},
		{
			From:  CreateV4StateConfirmPeriod,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 3: 确认班次选择
		// ============================================================
		{
			From:  CreateV4StateConfirmShifts,
			Event: CreateV4EventShiftsConfirmed,
			To:    CreateV4StateConfirmStaffCount,
			Act:   actOnShiftsConfirmed,
		},
		{
			From:  CreateV4StateConfirmShifts,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 4: 确认人数配置
		// ============================================================
		{
			From:  CreateV4StateConfirmStaffCount,
			Event: CreateV4EventStaffCountConfirmed,
			To:    CreateV4StatePersonalNeeds,
			Act:   actOnStaffCountConfirmed,
		},
		{
			From:  CreateV4StateConfirmStaffCount,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 5: 个人需求收集
		// ============================================================
		{
			From:  CreateV4StatePersonalNeeds,
			Event: CreateV4EventPersonalNeedsConfirmed,
			To:    CreateV4StateRuleOrganization,
			Act:   actOnPersonalNeedsConfirmed,
		},
		{
			From:  CreateV4StatePersonalNeeds,
			Event: CreateV4EventTemporaryNeedsTextSubmitted,
			To:    CreateV4StatePersonalNeeds,
			Act:   actOnTemporaryNeedsTextSubmitted,
		},
		{
			From:  CreateV4StatePersonalNeeds,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 6: 规则组织（V4核心新增）
		// ============================================================
		{
			From:       CreateV4StateRuleOrganization,
			Event:      CreateV4EventRulesOrganized,
			To:         CreateV4StateScheduling,
			StateLabel: "正在组织规则...",
			Act:        actOnRulesOrganized,
		},
		{
			From:  CreateV4StateRuleOrganization,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 7: 排班执行（使用确定性规则引擎）
		// 排班完成后直接进入校验阶段，不再经过 LLM 调整
		// ============================================================
		{
			From:       CreateV4StateScheduling,
			Event:      CreateV4EventSchedulingComplete,
			To:         CreateV4StateValidation,
			StateLabel: "正在执行排班...",
			Act:        actOnSchedulingComplete,
		},
		{
			From:  CreateV4StateScheduling,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 8: 确定性校验（只读，不主动修改排班）
		// ============================================================
		{
			From:  CreateV4StateValidation,
			Event: CreateV4EventValidationComplete,
			To:    CreateV4StateReview,
			Act:   actOnValidationComplete,
		},
		{
			From:  CreateV4StateValidation,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},

		// ============================================================
		// 阶段 9: 排班审核
		// ============================================================
		{
			From:  CreateV4StateReview,
			Event: CreateV4EventReviewConfirmed,
			To:    CreateV4StateCompleted,
			Act:   actOnReviewConfirmed,
		},
		{
			From:  CreateV4StateReview,
			Event: CreateV4EventUserModify,
			To:    CreateV4StateScheduling,
			Act:   actOnUserModify,
		},
		{
			From:  CreateV4StateReview,
			Event: CreateV4EventUserCancel,
			To:    CreateV4StateCancelled,
			Act:   actUserCancel,
		},
	}
}
