package utils

import (
	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// L3: ShiftTaskContext（单班次任务上下文）
// 这是实际执行 LLM 调用的层级，候选人员在这里动态计算
// ============================================================

// ShiftTaskContext 单班次任务上下文（L3层，递归的最底层）
type ShiftTaskContext struct {
	// ========== 从L2继承的只读数据 ==========

	// AllStaff 所有员工（用于姓名映射）
	AllStaff []*d_model.Employee

	// EligibleStaff 该班次的专属候选人员（班次分组成员）
	// 若不为空，则作为 ComputeCandidateStaff 的基础候选池（替代 AllStaff）
	// 若为空，则退化使用 AllStaff，保持向后兼容
	EligibleStaff []*d_model.Employee

	// PersonalNeeds 个人需求
	PersonalNeeds map[string][]*d_model.PersonalNeed

	// ========== 单班次特定数据 ==========

	// Task 任务定义（可能包含多个班次，但这里只处理一个）
	Task *d_model.ProgressiveTask

	// TargetShift 当前处理的班次（单个）
	TargetShift *d_model.Shift

	// RelatedShifts 相关班次（用于规则判断，如早晚班间隔）
	// 包含：1) 任务涉及的其他班次  2) 规则关联的班次
	RelatedShifts []*d_model.Shift

	// FilteredRules 过滤后的规则（仅与当前班次相关的）
	// 过滤规则：规则的 RelatedShiftIDs 包含 TargetShift.ID
	FilteredRules []*d_model.Rule

	// CandidateStaff 该班次的候选人员（动态计算）★
	// 规则：EligibleStaff（班次分组成员）- 已在目标日期被占位的员工 - 请假的员工
	// 若 EligibleStaff 为空，则退化为 AllStaff 作为基础候选池
	CandidateStaff []*d_model.Employee

	// OccupiedSlots 当前占位状态（任务开始时的快照）
	OccupiedSlots []d_model.StaffOccupiedSlot

	// StaffRequirements 该班次的人员需求（仅该班次）
	StaffRequirements []d_model.ShiftDateRequirement

	// FixedAssignments 该班次的固定排班（仅该班次）
	FixedAssignments []d_model.CtxFixedShiftAssignment

	// CurrentDraft 该班次的当前草稿
	CurrentDraft *d_model.ShiftScheduleDraft

	// ========== LLM调用预构建数据 ==========

	// StaffInfoForAI 候选人员的AI格式（预构建，避免重复转换）
	StaffInfoForAI []*d_model.StaffInfoForAI

	// RequirementsMap 人员需求映射（shiftID -> date -> count）
	RequirementsMap map[string]map[string]int

	// FixedAssignmentsMap 固定排班映射（date -> staffIDs）
	FixedAssignmentsMap map[string][]string

	// ========== 执行结果 ==========

	// Result 该班次的排班结果
	Result *d_model.ShiftScheduleDraft
}

// ComputeCandidateStaff ★ 动态计算该班次的候选人员
// 过滤规则：
//  1. 以 EligibleStaff（班次分组成员）为基础候选池；若为空则退化使用 AllStaff（向后兼容）
//  2. 排除在目标日期已被占位的员工
//  3. 排除在目标日期有请假（avoid/must）的员工
func (sctx *ShiftTaskContext) ComputeCandidateStaff() []*d_model.Employee {
	targetDates := sctx.Task.TargetDates

	// 1. 确定基础候选池：优先使用班次专属人员，若为空则兼容圈底使用全员
	basePool := sctx.EligibleStaff
	if len(basePool) == 0 {
		basePool = sctx.AllStaff
	}

	if len(targetDates) == 0 {
		// 没有指定日期，直接返回基础候选池
		return basePool
	}

	candidates := make([]*d_model.Employee, 0, len(basePool))

	for _, staff := range basePool {
		// 检查该员工在目标日期是否已被占位
		isOccupied := false
		for _, date := range targetDates {
			if d_model.IsStaffOccupiedOnDate(sctx.OccupiedSlots, staff.ID, date) {
				isOccupied = true
				break
			}
		}

		if !isOccupied {
			// 还需检查是否请假
			if !hasLeaveOnDates(sctx.PersonalNeeds[staff.ID], targetDates) {
				candidates = append(candidates, staff)
			}
		}
	}

	return candidates
}

// BuildLLMCache 构建LLM调用缓存
func (sctx *ShiftTaskContext) BuildLLMCache() {
	// 1. 转换候选人员为AI格式
	sctx.StaffInfoForAI = d_model.NewStaffInfoListFromEmployees(sctx.CandidateStaff)

	// 2. 构建需求映射
	sctx.RequirementsMap = d_model.ConvertRequirementsToMap(sctx.StaffRequirements)

	// 3. 构建固定排班映射
	sctx.FixedAssignmentsMap = d_model.ConvertFixedAssignmentsToMap(sctx.FixedAssignments)
}

// ============================================================
// 辅助函数
// ============================================================

// hasLeaveOnDates 检查员工是否在指定日期有请假
func hasLeaveOnDates(needs []*d_model.PersonalNeed, dates []string) bool {
	if len(needs) == 0 {
		return false
	}

	for _, need := range needs {
		if need == nil {
			continue
		}

		// 检查是否是回避类型（避免排班，类似请假）
		if need.RequestType != "avoid" && need.RequestType != "must" {
			continue
		}

		// 检查日期是否重叠
		for _, targetDate := range need.TargetDates {
			for _, date := range dates {
				if targetDate == date {
					return true
				}
			}
		}
	}

	return false
}
