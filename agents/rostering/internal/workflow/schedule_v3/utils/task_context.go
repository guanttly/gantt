package utils

import (
	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// L2: TaskExecutionContext（任务执行上下文）
// 一个任务可能涉及多个班次，需要拆分成多个 ShiftTaskContext
// ============================================================

// TaskExecutionContext 任务执行上下文（L2层）
type TaskExecutionContext struct {
	// ========== 从L1继承的只读数据 ==========

	// AllStaff 所有员工（继承自 L1）
	AllStaff []*d_model.Employee

	// ShiftMembersMap 各班次专属人员映射（从 L1 继承，shiftID → 成员列表，用于 L3 候选人过滤）
	ShiftMembersMap map[string][]*d_model.Employee

	// Rules 规则列表（继承自 L1）
	Rules []*d_model.Rule

	// PersonalNeeds 个人需求（继承自 L1）
	PersonalNeeds map[string][]*d_model.PersonalNeed

	// ========== 任务范围数据（从L1过滤/切片） ==========

	// Task 当前任务定义
	Task *d_model.ProgressiveTask

	// TargetShifts 该任务涉及的班次列表（可能多个）
	TargetShifts []*d_model.Shift

	// OccupiedSlots 当前占位状态（任务开始时的快照，深拷贝）
	OccupiedSlots []d_model.StaffOccupiedSlot

	// StaffRequirements 该任务相关的人员需求（过滤后）
	StaffRequirements []d_model.ShiftDateRequirement

	// FixedAssignments 该任务相关的固定排班（过滤后）
	FixedAssignments []d_model.CtxFixedShiftAssignment

	// WorkingDraft 当前工作草稿（引用 L1 的 WorkingDraft）
	WorkingDraft *d_model.ScheduleDraft

	// ========== 任务执行结果 ==========

	// TaskResult 任务执行结果
	TaskResult *d_model.TaskResult
}

// SplitIntoShiftContexts 将任务上下文拆分成多个单班次上下文
func (tctx *TaskExecutionContext) SplitIntoShiftContexts() []*ShiftTaskContext {
	contexts := make([]*ShiftTaskContext, 0, len(tctx.TargetShifts))

	for _, shift := range tctx.TargetShifts {
		// 为每个班次创建独立的上下文
		shiftCtx := &ShiftTaskContext{
			// 继承只读数据
			AllStaff:      tctx.AllStaff,
			EligibleStaff: tctx.ShiftMembersMap[shift.ID], // 该班次专属候选人员（用于基础候选池）
			PersonalNeeds: tctx.PersonalNeeds,

			// 班次特定数据
			Task:        tctx.Task,
			TargetShift: shift,

			// 相关班次（任务中的其他班次 + 规则关联的班次）
			RelatedShifts: buildRelatedShifts(shift, tctx.TargetShifts, tctx.Rules),

			// 过滤与该班次相关的规则
			FilteredRules: filterRulesByShift(tctx.Rules, shift.ID),

			// 深拷贝当前占位状态（避免并发修改）
			OccupiedSlots: deepCopyOccupiedSlots(tctx.OccupiedSlots),

			// 过滤出该班次的需求和固定排班
			StaffRequirements: filterRequirementsByShift(tctx.StaffRequirements, shift.ID),
			FixedAssignments:  filterAssignmentsByShift(tctx.FixedAssignments, shift.ID),

			// 提取该班次的当前草稿
			CurrentDraft: extractShiftDraft(tctx.WorkingDraft, shift.ID),
		}

		// ★ 关键：动态计算该班次的候选人员
		shiftCtx.CandidateStaff = shiftCtx.ComputeCandidateStaff()

		// 预构建LLM缓存
		shiftCtx.BuildLLMCache()

		contexts = append(contexts, shiftCtx)
	}

	return contexts
}

// MergeShiftResults 合并所有单班次结果到任务结果
func (tctx *TaskExecutionContext) MergeShiftResults(shiftResults []*ShiftTaskContext) error {
	tctx.TaskResult = &d_model.TaskResult{
		TaskID:         tctx.Task.ID,
		Success:        true,
		ShiftSchedules: make(map[string]*d_model.ShiftScheduleDraft),
	}

	for _, shiftCtx := range shiftResults {
		if shiftCtx.Result != nil {
			// 合并班次结果
			tctx.TaskResult.ShiftSchedules[shiftCtx.TargetShift.ID] = shiftCtx.Result

			// 从排班结果中提取占位信息并合并
			newOccupiedSlots := ExtractOccupiedSlotsFromDraft(shiftCtx.Result, shiftCtx.TargetShift.ID)
			tctx.OccupiedSlots = mergeOccupiedSlots(tctx.OccupiedSlots, newOccupiedSlots)
		}
	}

	return nil
}

// ExtractOccupiedSlotsFromDraft 从排班草稿中提取占位信息（导出函数）
func ExtractOccupiedSlotsFromDraft(draft *d_model.ShiftScheduleDraft, shiftID string) []d_model.StaffOccupiedSlot {
	if draft == nil || draft.Schedule == nil {
		return []d_model.StaffOccupiedSlot{}
	}

	slots := make([]d_model.StaffOccupiedSlot, 0)
	for date, staffIDs := range draft.Schedule {
		for _, staffID := range staffIDs {
			slots = append(slots, d_model.StaffOccupiedSlot{
				StaffID: staffID,
				Date:    date,
				ShiftID: shiftID,
				Source:  "draft",
			})
		}
	}

	return slots
}

// extractOccupiedSlotsFromDraft 从排班草稿中提取占位信息（内部函数）
func extractOccupiedSlotsFromDraft(draft *d_model.ShiftScheduleDraft, shiftID string) []d_model.StaffOccupiedSlot {
	return ExtractOccupiedSlotsFromDraft(draft, shiftID)
}

// UpdateOccupiedSlotsForNextShifts 更新后续班次的占位信息（导出函数）
// 当某个班次执行完成后，需要更新后续班次的占位信息，以便它们能正确计算候选人员
func UpdateOccupiedSlotsForNextShifts(
	nextShiftContexts []*ShiftTaskContext,
	completedResult *d_model.ShiftScheduleDraft,
	completedShiftID string,
) {
	if completedResult == nil || completedResult.Schedule == nil {
		return
	}

	// 从完成的结果中提取占位信息
	newSlots := ExtractOccupiedSlotsFromDraft(completedResult, completedShiftID)

	// 更新后续班次的占位信息
	for _, shiftCtx := range nextShiftContexts {
		// 合并新的占位信息到后续班次的占位列表中
		shiftCtx.OccupiedSlots = mergeOccupiedSlots(shiftCtx.OccupiedSlots, newSlots)

		// 重新计算候选人员（因为占位信息已更新）
		shiftCtx.CandidateStaff = shiftCtx.ComputeCandidateStaff()

		// 重新构建LLM缓存（因为候选人员已更新）
		shiftCtx.BuildLLMCache()
	}
}

// updateOccupiedSlotsForNextShifts 更新后续班次的占位信息（内部函数）
func updateOccupiedSlotsForNextShifts(
	nextShiftContexts []*ShiftTaskContext,
	completedResult *d_model.ShiftScheduleDraft,
	completedShiftID string,
) {
	UpdateOccupiedSlotsForNextShifts(nextShiftContexts, completedResult, completedShiftID)
}

// ============================================================
// 辅助函数：构建相关班次和过滤规则
// ============================================================

// buildRelatedShifts 构建相关班次列表
// 包含：1) 任务中的其他班次  2) 规则关联的班次
func buildRelatedShifts(
	targetShift *d_model.Shift,
	taskShifts []*d_model.Shift,
	rules []*d_model.Rule,
) []*d_model.Shift {
	relatedMap := make(map[string]*d_model.Shift)

	// 1. 添加任务中的其他班次
	for _, shift := range taskShifts {
		if shift.ID != targetShift.ID {
			relatedMap[shift.ID] = shift
		}
	}

	// 2. 从规则中提取关联班次
	for _, rule := range rules {
		// 获取规则关联的班次ID列表
		relatedShiftIDs := getAssociatedShiftIDs(rule.Associations)

		// 如果规则涉及当前班次
		if containsShiftID(relatedShiftIDs, targetShift.ID) {
			// 添加规则中的其他关联班次
			for _, relatedID := range relatedShiftIDs {
				if relatedID != targetShift.ID {
					// 从任务班次列表中查找
					if shift := findShiftByID(taskShifts, relatedID); shift != nil {
						relatedMap[relatedID] = shift
					}
				}
			}
		}
	}

	// 转换为数组
	result := make([]*d_model.Shift, 0, len(relatedMap))
	for _, shift := range relatedMap {
		result = append(result, shift)
	}
	return result
}

// filterRulesByShift 过滤与班次相关的规则（支持多维度关联）
// 过滤逻辑：
//   - ApplyScope=="global" → 全局规则，始终包含
//   - 无关联（数据异常兜底） → 保守包含
//   - 有班次关联 → 关联到当前班次才包含
//   - 仅有员工/分组关联（无班次关联） → 保守包含（无法在此层判断员工是否属于该班次）
func filterRulesByShift(rules []*d_model.Rule, shiftID string) []*d_model.Rule {
	result := make([]*d_model.Rule, 0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		// 全局规则始终包含
		if rule.ApplyScope == "global" {
			result = append(result, rule)
			continue
		}

		// 无关联（数据异常兜底），保守包含
		if len(rule.Associations) == 0 {
			result = append(result, rule)
			continue
		}

		// 检查是否有班次关联
		hasShiftAssoc := false
		shiftMatched := false
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == "shift" {
				hasShiftAssoc = true
				if assoc.AssociationID == shiftID {
					shiftMatched = true
				}
			}
		}

		if shiftMatched {
			// 班次关联匹配
			result = append(result, rule)
		} else if !hasShiftAssoc {
			// 无班次关联（仅员工/分组关联），保守包含
			result = append(result, rule)
		}
		// 有班次关联但不匹配当前班次 → 排除
	}
	return result
}

// filterRequirementsByShift 过滤某班次的需求
func filterRequirementsByShift(reqs []d_model.ShiftDateRequirement, shiftID string) []d_model.ShiftDateRequirement {
	return d_model.FilterRequirementsByShift(reqs, shiftID)
}

// filterAssignmentsByShift 过滤某班次的固定排班
func filterAssignmentsByShift(assigns []d_model.CtxFixedShiftAssignment, shiftID string) []d_model.CtxFixedShiftAssignment {
	result := make([]d_model.CtxFixedShiftAssignment, 0)
	for _, a := range assigns {
		if a.ShiftID == shiftID {
			result = append(result, a)
		}
	}
	return result
}

// extractShiftDraft 从总草稿中提取某班次的草稿
func extractShiftDraft(workingDraft *d_model.ScheduleDraft, shiftID string) *d_model.ShiftScheduleDraft {
	if workingDraft == nil || workingDraft.Shifts == nil {
		return d_model.NewShiftScheduleDraft()
	}

	if shiftDraft, ok := workingDraft.Shifts[shiftID]; ok && shiftDraft != nil {
		// 转换 DayShift 到员工ID列表
		result := d_model.NewShiftScheduleDraft()
		for date, dayShift := range shiftDraft.Days {
			if dayShift != nil {
				result.Schedule[date] = dayShift.StaffIDs
			}
		}
		return result
	}

	return d_model.NewShiftScheduleDraft()
}

// deepCopyOccupiedSlots 深拷贝占位信息（避免并发修改）
func deepCopyOccupiedSlots(slots []d_model.StaffOccupiedSlot) []d_model.StaffOccupiedSlot {
	result := make([]d_model.StaffOccupiedSlot, len(slots))
	copy(result, slots)
	return result
}

// MergeOccupiedSlots 合并占位信息（导出函数）
func MergeOccupiedSlots(existing, updates []d_model.StaffOccupiedSlot) []d_model.StaffOccupiedSlot {
	return mergeOccupiedSlotsInternal(existing, updates)
}

// mergeOccupiedSlots 合并占位信息（内部函数，追加新增的，保留旧的）
func mergeOccupiedSlots(existing, updates []d_model.StaffOccupiedSlot) []d_model.StaffOccupiedSlot {
	return mergeOccupiedSlotsInternal(existing, updates)
}

// mergeOccupiedSlotsInternal 合并占位信息的内部实现
func mergeOccupiedSlotsInternal(existing, updates []d_model.StaffOccupiedSlot) []d_model.StaffOccupiedSlot {
	result := make([]d_model.StaffOccupiedSlot, 0, len(existing)+len(updates))
	result = append(result, existing...)

	// 追加新增的占位（去重）
	for _, newSlot := range updates {
		if !containsOccupiedSlot(result, newSlot) {
			result = append(result, newSlot)
		}
	}

	return result
}

// containsOccupiedSlot 检查占位数组是否包含指定占位
func containsOccupiedSlot(slots []d_model.StaffOccupiedSlot, slot d_model.StaffOccupiedSlot) bool {
	for _, s := range slots {
		if s.StaffID == slot.StaffID && s.Date == slot.Date && s.ShiftID == slot.ShiftID {
			return true
		}
	}
	return false
}

// containsShiftID 检查班次ID列表是否包含指定ID
func containsShiftID(shiftIDs []string, shiftID string) bool {
	for _, id := range shiftIDs {
		if id == shiftID {
			return true
		}
	}
	return false
}

// findShiftByID 从班次列表中查找指定ID的班次
func findShiftByID(shifts []*d_model.Shift, shiftID string) *d_model.Shift {
	for _, shift := range shifts {
		if shift != nil && shift.ID == shiftID {
			return shift
		}
	}
	return nil
}

// getAssociatedShiftIDs 从关联列表中提取班次ID
func getAssociatedShiftIDs(associations []d_model.RuleAssociation) []string {
	ids := make([]string, 0)
	for _, assoc := range associations {
		if assoc.AssociationType == "shift" {
			ids = append(ids, assoc.AssociationID)
		}
	}
	return ids
}

// ConvertShiftTaskContextToCoreV3TaskContext 将 ShiftTaskContext 转换为 CoreV3TaskContext
// 用于适配现有的任务执行器接口
func ConvertShiftTaskContextToCoreV3TaskContext(
	shiftCtx *ShiftTaskContext,
	orgID string,
	workingDraft *d_model.ScheduleDraft,
) *CoreV3TaskContext {
	// 构建 CoreV3TaskContext
	coreCtx := &CoreV3TaskContext{
		OrgID:             orgID,
		Task:              shiftCtx.Task,
		Shifts:            append([]*d_model.Shift{shiftCtx.TargetShift}, shiftCtx.RelatedShifts...),
		Rules:             shiftCtx.FilteredRules,
		CandidateStaff:    shiftCtx.CandidateStaff, // ★ 使用动态计算的候选人员
		AllStaff:          shiftCtx.AllStaff,
		StaffRequirements: shiftCtx.StaffRequirements,
		FixedAssignments:  shiftCtx.FixedAssignments,
		OccupiedSlots:     shiftCtx.OccupiedSlots,
		PersonalNeeds:     shiftCtx.PersonalNeeds,
		CurrentDraft:      shiftCtx.CurrentDraft,
		WorkingDraft:      workingDraft,
	}

	// 预构建LLM缓存
	coreCtx.BuildLLMCache()

	return coreCtx
}
