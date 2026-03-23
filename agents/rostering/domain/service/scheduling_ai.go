package service

import (
	"context"

	"jusha/mcp/pkg/workflow/session"

	"jusha/agent/rostering/domain/model"
)

// ISchedulingAIService 排班 AI 服务接口
// 提供 AI 驱动的排班相关功能
type ISchedulingAIService interface {
	// GenerateShiftTodoPlan AI生成单个班次的Todo计划
	// 输入本班次所有规则和信息，让AI有序安排排班计划（Todo列表），并对每个计划进行说明
	// shiftInfo: 班次基本信息（ID、名称、时间段等）
	// staffList: 可用人员列表
	// rules: 适用于该班次的所有规则（全局+班次+分组+人员）
	// staffRequirements: 人数需求（日期->人数）
	// previousDraft: 之前班次的排班结果（用于避免冲突）
	// fixedShiftAssignments: 固定排班人员（date -> staffIds），这些人员已经在固定班次中安排，绝对不能从当前班次中调整
	// temporaryNeeds: 临时需求列表（从用户调整需求中提取的临时需求），确保不会安排已出差、请假或有事的员工
	// 返回: Todo计划（有序的任务列表+整体说明）
	GenerateShiftTodoPlan(ctx context.Context, shiftInfo *model.ShiftInfo, staffList []*model.StaffInfoForAI, rules []*model.RuleInfo, staffRequirements map[string]int, previousDraft *model.ShiftScheduleDraft, fixedShiftAssignments map[string][]string, temporaryNeeds []*model.PersonalNeed) (*model.TodoPlanResult, error)

	// ExecuteTodoTask AI执行单个Todo任务（V3增强版）
	// 输入是本班次基本情况及Todo说明以及可用人员列表
	// todoTask: 当前要执行的Todo任务信息
	// shiftInfo: 班次基本信息
	// availableStaff: 当前可用人员列表（包含已排班标记信息，AI可自行判断时段冲突）
	// currentDraft: 当前排班草案（强类型）
	// staffRequirements: 每日人数需求上限（日期->人数），AI必须严格遵守
	// fixedShiftAssignments: 固定排班人员（date -> staffIds），这些人员已经在固定班次中安排，绝对不能从当前班次中调整
	// temporaryNeeds: 临时需求列表（从用户调整需求中提取的临时需求），确保不会安排已出差、请假或有事的员工
	// allStaffList: 所有员工列表（用于姓名映射，确保显示正确的姓名而不是UUID）
	// allShifts: 【V3新增】所有班次列表（用于时间信息展示）
	// workingDraft: 【V3新增】当前工作排班草案（用于构建人员排班状态）
	// 返回: 执行结果（更新后的排班草案+执行说明），schedule中使用内部ID
	ExecuteTodoTask(
		ctx context.Context,
		todoTask *model.SchedulingTodo,
		shiftInfo *model.ShiftInfo,
		availableStaff []*model.StaffInfoForAI,
		currentDraft *model.ShiftScheduleDraft,
		staffRequirements map[string]int,
		fixedShiftAssignments map[string][]string,
		temporaryNeeds []*model.PersonalNeed,
		allStaffList []*model.Employee,
		allShifts []*model.Shift,
		workingDraft *model.ScheduleDraft,
	) (*model.TodoExecutionResult, error)

	// ValidateAndAdjustShiftSchedule AI校验并调整班次排班
	// 完成Todo列表后，对排班结果和总体要求，让AI再进行校验和总结
	// 如果没问题则结束本班次，如果存在问题，则进行修改
	// shiftDraft: 完成的班次排班草案（强类型）
	// shiftInfo: 班次基本信息
	// rules: 所有适用规则
	// staffRequirements: 原始人数需求
	// staffList: 人员列表（用于姓名-ID转换）
	// taskInfo: 任务信息（用于理解任务目标和进行针对性校验）
	// 返回: 校验结果（是否通过+问题列表+调整后的草案+总结），adjustedSchedule中使用内部ID
	ValidateAndAdjustShiftSchedule(ctx context.Context, shiftDraft *model.ShiftScheduleDraft, shiftInfo *model.ShiftInfo, rules []*model.RuleInfo, staffRequirements map[string]int, staffList []*model.StaffInfoForAI, taskInfo *model.ProgressiveTask) (*model.ValidationResult, error)

	// ============================================================
	// 排班调整工作流 AI 方法
	// ============================================================

	// AnalyzeAdjustIntent 分析用户的排班调整意图
	// 在 schedule.adjust 工作流中使用，识别用户想要进行的调整操作类型
	// userInput: 用户输入的调整描述
	// messages: 会话消息历史（用于上下文理解）
	// 返回: 解析后的调整意图（类型、目标日期、人员等）
	AnalyzeAdjustIntent(ctx context.Context, userInput string, messages []session.Message) (*model.AdjustIntent, error)

	// AdjustShiftSchedule 直接根据用户需求调整排班
	// 用于 adjust 工作流的修改模式，直接调用 AI 根据用户需求、原始排班、规则等信息生成调整后的排班
	// userRequirement: 用户的调整需求描述
	// originalDraft: 原始排班草案
	// shiftInfo: 班次基本信息
	// staffList: 可用人员列表
	// allStaffList: 所有员工列表（用于姓名映射，确保显示正确的姓名而不是UUID）
	// rules: 所有适用规则
	// staffRequirements: 每日人数需求
	// existingScheduleMarks: 已占位信息（其他班次的排班，用于避免冲突）
	// fixedShiftAssignments: 固定排班人员（date -> staffIds），这些人员绝对不能调整
	// 返回: 调整结果（包含完整排班、AI总结和变化列表）
	AdjustShiftSchedule(ctx context.Context, userRequirement string, originalDraft *model.ShiftScheduleDraft, shiftInfo *model.ShiftInfo, staffList []*model.StaffInfoForAI, allStaffList []*model.Employee, rules []*model.RuleInfo, staffRequirements map[string]int, existingScheduleMarks map[string]map[string]bool, fixedShiftAssignments map[string][]string) (*model.AdjustScheduleResult, error)

	// ExtractTemporaryNeeds 从用户消息中提取临时需求
	// 用于识别用户消息中提到的临时需求（如某人出差、某天有事等），这些需求应该被添加到临时需求列表中
	// userMessage: 用户输入的调整需求消息
	// allStaffList: 所有员工列表（用于姓名匹配）
	// startDate: 排班周期开始日期
	// endDate: 排班周期结束日期
	// messages: 会话消息历史（用于上下文理解）
	// 返回: 提取的临时需求列表（PersonalNeed 结构，NeedType 为 "temporary"）
	ExtractTemporaryNeeds(ctx context.Context, userMessage string, allStaffList []*model.Employee, startDate, endDate string, messages []session.Message) ([]*model.PersonalNeed, error)

	// ============================================================
	// 全局评审人工处理 AI 方法
	// ============================================================

	// ProcessManualReviewModification 处理人工评审修改
	// 根据用户输入的修改意图，理解用户需求并应用到排班草案
	// userMessage: 用户输入的修改需求
	// manualContext: 人工评审上下文（包含需处理的项目列表）
	// currentDraft: 当前排班草案
	// staffList: 人员列表
	// shifts: 班次列表
	// 返回: 修改结果（包含修改后的草案、应用的变更列表和摘要）
	ProcessManualReviewModification(
		ctx context.Context,
		userMessage string,
		manualContext *model.ManualReviewContext,
		currentDraft *model.ScheduleDraft,
		staffList []*model.Employee,
		shifts []*model.Shift,
	) (*model.ManualReviewModifyResult, error)
}
