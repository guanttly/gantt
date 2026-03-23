package create

import (
	"context"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// ============================================================
// V3辅助函数
// ============================================================

// ExtractPersonalNeeds 从规则中提取个人需求（复用V2的逻辑）
// 【关键修复】只提取候选人员（staffList）的个人需求，过滤掉非候选人员
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
	result := make(map[string][]*PersonalNeed)

	// 构建候选人员ID集合（用于过滤）
	candidateStaffIDs := make(map[string]bool)
	staffNameMap := make(map[string]string)
	for _, staff := range staffList {
		candidateStaffIDs[staff.ID] = true // ← 添加到候选集合
		staffNameMap[staff.ID] = staff.Name
	}

	// 从规则中提取个人需求
	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}

		// 查找规则关联的人员
		if len(rule.Associations) > 0 {
			for _, assoc := range rule.Associations {
				if assoc.AssociationType == "staff" {
					staffID := assoc.AssociationID

					// 【关键修复】只提取候选人员的需求，过滤掉非候选人员
					if !candidateStaffIDs[staffID] {
						continue
					}

					need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
					if need != nil {
						if result[staffID] == nil {
							result[staffID] = make([]*PersonalNeed, 0)
						}
						result[staffID] = append(result[staffID], need)
					}
				}
			}
		}
	}

	return result
}

// parseRuleToPersonalNeed 根据规则类型和内容，转换为标准化的个人需求结构
func parseRuleToPersonalNeed(rule *d_model.Rule, staffID string, staffNameMap map[string]string) *PersonalNeed {
	if staffID == "" {
		return nil
	}

	staffName := staffNameMap[staffID]
	if staffName == "" {
		staffName = "未知员工"
	}

	// 根据规则类型判断需求类型
	needType := "permanent" // 默认常态化
	if rule.ValidFrom != nil && rule.ValidTo != nil {
		needType = "temporary"
	}

	// 根据规则优先级判断请求类型
	requestType := "prefer" // 默认偏好
	if rule.Priority <= 3 {
		requestType = "must" // 高优先级视为必须
	}

	// 构建需求描述
	description := rule.Description
	if description == "" {
		if rule.RuleData != "" {
			description = rule.RuleData
		} else {
			description = rule.Name
		}
	}

	need := &PersonalNeed{
		StaffID:     staffID,
		StaffName:   staffName,
		NeedType:    needType,
		RequestType: requestType,
		Description: description,
		Priority:    rule.Priority,
		RuleID:      rule.ID,
		Source:      "rule",
		Confirmed:   false,
	}

	return need
}

// ============================================================
// V3上下文传播函数（CreateV3 ↔ CoreV3）
// ============================================================

// BuildTaskExecutionContext 从 CreateV3Context 构建 TaskExecutionContext (L2)
// 用于任务执行前准备任务级上下文
func BuildTaskExecutionContext(
	createCtx *CreateV3Context,
	task *d_model.ProgressiveTask,
) *utils.TaskExecutionContext {
	// 1. 过滤任务相关的班次
	taskShifts := filterShiftsByTask(createCtx.SelectedShifts, task)

	// 2. 过滤任务相关的需求
	taskRequirements := filterRequirementsByTask(createCtx.StaffRequirements, task)

	// 3. 过滤任务相关的固定排班
	taskFixedAssignments := filterFixedAssignmentsByTask(createCtx.FixedAssignments, task)

	// 4. 深拷贝占位数据（任务执行时的快照）
	occupiedSlotsCopy := make([]d_model.StaffOccupiedSlot, len(createCtx.OccupiedSlots))
	copy(occupiedSlotsCopy, createCtx.OccupiedSlots)

	// 5. 构建任务执行上下文 (L2)
	taskCtx := &utils.TaskExecutionContext{
		// 从L1继承的只读数据
		AllStaff:      createCtx.AllStaff,
		Rules:         createCtx.Rules,
		PersonalNeeds: convertPersonalNeedsToModel(createCtx.PersonalNeeds),

		// 任务范围数据
		Task:         task,
		TargetShifts: taskShifts,

		// 深拷贝L1的可写数据（任务执行时的快照）
		OccupiedSlots:     occupiedSlotsCopy,
		StaffRequirements: taskRequirements,
		FixedAssignments:  taskFixedAssignments,

		// 引用L1的WorkingDraft
		WorkingDraft: createCtx.WorkingDraft,
	}

	return taskCtx
}

// BuildCoreV3TaskContext 从 CreateV3Context 构建 CoreV3TaskContext
// 用于启动子工作流前准备子工作流上下文
// Deprecated: 请使用 BuildTaskExecutionContext + SplitIntoShiftContexts 代替
func BuildCoreV3TaskContext(
	ctx context.Context,
	createCtx *CreateV3Context,
	task *d_model.ProgressiveTask,
	orgID string,
) *utils.CoreV3TaskContext {

	// 1. 过滤任务相关的班次
	taskShifts := filterShiftsByTask(createCtx.SelectedShifts, task)

	// 2. 过滤任务相关的需求
	taskRequirements := filterRequirementsByTask(createCtx.StaffRequirements, task)

	// 3. 过滤任务相关的固定排班
	taskFixedAssignments := filterFixedAssignmentsByTask(createCtx.FixedAssignments, task)

	// 4. 深拷贝占位数据（子工作流可修改）
	occupiedSlotsCopy := make([]d_model.StaffOccupiedSlot, len(createCtx.OccupiedSlots))
	copy(occupiedSlotsCopy, createCtx.OccupiedSlots)

	// 5. 构建当前班次草稿（如果有）
	var currentDraft *d_model.ShiftScheduleDraft
	if createCtx.WorkingDraft != nil && len(task.TargetShifts) > 0 {
		shiftID := task.TargetShifts[0]
		currentDraft = extractShiftDraft(createCtx.WorkingDraft, shiftID)
	}

	// 6. 构建任务上下文
	taskCtx := &utils.CoreV3TaskContext{
		OrgID:             orgID,
		Task:              task,
		Shifts:            taskShifts,
		Rules:             createCtx.Rules,
		CandidateStaff:    createCtx.AllStaff, // 使用全员列表，在L3动态过滤
		AllStaff:          createCtx.AllStaff,
		StaffRequirements: taskRequirements,
		FixedAssignments:  taskFixedAssignments,
		OccupiedSlots:     occupiedSlotsCopy,
		PersonalNeeds:     convertPersonalNeedsToModel(createCtx.PersonalNeeds),
		CurrentDraft:      currentDraft,
		WorkingDraft:      createCtx.WorkingDraft,
	}

	// 7. 预构建LLM缓存
	taskCtx.BuildLLMCache()

	return taskCtx
}

// MergeTaskResultToCreate 将任务执行结果合并回父上下文 (L2 → L1)
// 用于任务执行完成后更新父上下文状态
func MergeTaskResultToCreate(
	createCtx *CreateV3Context,
	taskCtx *utils.TaskExecutionContext,
) error {
	if taskCtx.TaskResult == nil {
		return nil
	}

	// 1. 保存任务结果
	if createCtx.TaskResults == nil {
		createCtx.TaskResults = make(map[string]*d_model.TaskResult)
	}
	createCtx.TaskResults[taskCtx.Task.ID] = taskCtx.TaskResult

	// 2. 合并占位信息（追加新增的占位）
	createCtx.OccupiedSlots = mergeOccupiedSlots(
		createCtx.OccupiedSlots,
		taskCtx.OccupiedSlots,
	)

	// 3. 合并排班草稿
	if taskCtx.TaskResult.ShiftSchedules != nil {
		for shiftID, shiftDraft := range taskCtx.TaskResult.ShiftSchedules {
			mergeShiftDraftToWorking(createCtx.WorkingDraft, shiftID, shiftDraft)
		}
	}

	// 4. 计算并记录变更批次
	if len(taskCtx.TaskResult.ShiftSchedules) > 0 {
		batch := utils.ComputeChangeBatch(
			taskCtx.Task.ID,
			taskCtx.Task.Title,
			createCtx.CurrentTaskIndex+1,
			createCtx.WorkingDraft,
			taskCtx.TaskResult.ShiftSchedules,
			createCtx.AllStaff,
			createCtx.AllStaff,
			createCtx.SelectedShifts,
		)
		if batch != nil && len(batch.Changes) > 0 {
			createCtx.ChangeBatches = append(createCtx.ChangeBatches, batch)
		}
	}

	return nil
}

// MergeTaskResultToCreateContext 将任务结果合并回父上下文
// 用于子工作流完成后更新父上下文状态
// Deprecated: 请使用 MergeTaskResultToCreate 代替
func MergeTaskResultToCreateContext(
	ctx context.Context,
	createCtx *CreateV3Context,
	taskCtx *utils.CoreV3TaskContext,
) error {
	if taskCtx.TaskResult == nil {
		return nil
	}

	// 1. 保存任务结果
	if createCtx.TaskResults == nil {
		createCtx.TaskResults = make(map[string]*d_model.TaskResult)
	}
	createCtx.TaskResults[taskCtx.Task.ID] = taskCtx.TaskResult

	// 2. 合并占位信息（追加新增的占位）
	createCtx.OccupiedSlots = mergeOccupiedSlots(
		createCtx.OccupiedSlots,
		taskCtx.OccupiedSlots,
	)

	// 3. 合并排班草稿
	if taskCtx.TaskResult.ShiftSchedules != nil {
		for shiftID, shiftDraft := range taskCtx.TaskResult.ShiftSchedules {
			mergeShiftDraftToWorking(createCtx.WorkingDraft, shiftID, shiftDraft)
		}
	}

	// 4. 计算并记录变更批次
	if len(taskCtx.TaskResult.ShiftSchedules) > 0 {
		batch := utils.ComputeChangeBatch(
			taskCtx.Task.ID,
			taskCtx.Task.Title,
			createCtx.CurrentTaskIndex+1,
			createCtx.WorkingDraft,
			taskCtx.TaskResult.ShiftSchedules,
			createCtx.AllStaff,
			createCtx.AllStaff,
			createCtx.SelectedShifts,
		)
		if batch != nil && len(batch.Changes) > 0 {
			createCtx.ChangeBatches = append(createCtx.ChangeBatches, batch)
		}
	}

	return nil
}

// ============================================================
// 内部辅助函数
// ============================================================

// filterShiftsByTask 过滤任务相关的班次
func filterShiftsByTask(shifts []*d_model.Shift, task *d_model.ProgressiveTask) []*d_model.Shift {
	if len(task.TargetShifts) == 0 {
		return shifts
	}

	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		shiftMap[shift.ID] = shift
	}

	result := make([]*d_model.Shift, 0, len(task.TargetShifts))
	for _, shiftID := range task.TargetShifts {
		if shift, ok := shiftMap[shiftID]; ok {
			result = append(result, shift)
		}
	}

	return result
}

// filterRequirementsByTask 过滤任务相关的需求
func filterRequirementsByTask(reqs []d_model.ShiftDateRequirement, task *d_model.ProgressiveTask) []d_model.ShiftDateRequirement {
	if len(task.TargetShifts) == 0 && len(task.TargetDates) == 0 {
		return reqs
	}

	// 构建过滤条件
	targetShifts := make(map[string]bool)
	for _, shiftID := range task.TargetShifts {
		targetShifts[shiftID] = true
	}

	targetDates := make(map[string]bool)
	for _, date := range task.TargetDates {
		targetDates[date] = true
	}

	// 过滤
	result := make([]d_model.ShiftDateRequirement, 0)
	for _, req := range reqs {
		matchShift := len(targetShifts) == 0 || targetShifts[req.ShiftID]
		matchDate := len(targetDates) == 0 || targetDates[req.Date]
		if matchShift && matchDate {
			result = append(result, req)
		}
	}

	return result
}

// filterFixedAssignmentsByTask 过滤任务相关的固定排班
func filterFixedAssignmentsByTask(assigns []d_model.CtxFixedShiftAssignment, task *d_model.ProgressiveTask) []d_model.CtxFixedShiftAssignment {
	if len(task.TargetShifts) == 0 && len(task.TargetDates) == 0 {
		return assigns
	}

	targetShifts := make(map[string]bool)
	for _, shiftID := range task.TargetShifts {
		targetShifts[shiftID] = true
	}

	targetDates := make(map[string]bool)
	for _, date := range task.TargetDates {
		targetDates[date] = true
	}

	result := make([]d_model.CtxFixedShiftAssignment, 0)
	for _, assign := range assigns {
		matchShift := len(targetShifts) == 0 || targetShifts[assign.ShiftID]
		matchDate := len(targetDates) == 0 || targetDates[assign.Date]
		if matchShift && matchDate {
			result = append(result, assign)
		}
	}

	return result
}

// extractShiftDraft 从 ScheduleDraft 中提取单个班次的草稿
func extractShiftDraft(draft *d_model.ScheduleDraft, shiftID string) *d_model.ShiftScheduleDraft {
	result := d_model.NewShiftScheduleDraft()
	if draft == nil || draft.Shifts == nil {
		return result
	}

	shiftDraftData, ok := draft.Shifts[shiftID]
	if !ok || shiftDraftData == nil {
		return result
	}

	// 转换为 ShiftScheduleDraft 格式
	for date, dayShift := range shiftDraftData.Days {
		if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
			result.Schedule[date] = dayShift.StaffIDs
		}
	}

	return result
}

// mergeOccupiedSlots 合并占位信息（去重）
func mergeOccupiedSlots(base, new []d_model.StaffOccupiedSlot) []d_model.StaffOccupiedSlot {
	// 构建已有占位的索引
	existing := make(map[string]bool)
	for _, slot := range base {
		key := slot.StaffID + "|" + slot.Date
		existing[key] = true
	}

	// 添加新占位（去重）
	result := make([]d_model.StaffOccupiedSlot, len(base))
	copy(result, base)

	for _, slot := range new {
		key := slot.StaffID + "|" + slot.Date
		if !existing[key] {
			result = append(result, slot)
			existing[key] = true
		}
	}

	return result
}

// mergeShiftDraftToWorking 将班次草稿合并到工作草稿中
func mergeShiftDraftToWorking(working *d_model.ScheduleDraft, shiftID string, shiftDraft *d_model.ShiftScheduleDraft) {
	if working == nil || working.Shifts == nil || shiftDraft == nil {
		return
	}

	// 确保班次草稿存在
	if working.Shifts[shiftID] == nil {
		working.Shifts[shiftID] = &d_model.ShiftDraft{
			ShiftID: shiftID,
			Days:    make(map[string]*d_model.DayShift),
		}
	}

	// 合并排班数据
	for date, staffIDs := range shiftDraft.Schedule {
		working.Shifts[shiftID].Days[date] = &d_model.DayShift{
			StaffIDs:    staffIDs,
			ActualCount: len(staffIDs),
			IsFixed:     false,
		}
	}
}

// convertPersonalNeedsToModel 转换个人需求类型（create.PersonalNeed -> d_model.PersonalNeed）
func convertPersonalNeedsToModel(needs map[string][]*PersonalNeed) map[string][]*d_model.PersonalNeed {
	result := make(map[string][]*d_model.PersonalNeed)
	for staffID, staffNeeds := range needs {
		modelNeeds := make([]*d_model.PersonalNeed, 0, len(staffNeeds))
		for _, need := range staffNeeds {
			if need == nil {
				continue
			}
			modelNeeds = append(modelNeeds, &d_model.PersonalNeed{
				StaffID:         need.StaffID,
				StaffName:       need.StaffName,
				NeedType:        need.NeedType,
				RequestType:     need.RequestType,
				TargetShiftID:   need.TargetShiftID,
				TargetShiftName: need.TargetShiftName,
				TargetDates:     need.TargetDates,
				Description:     need.Description,
				Priority:        need.Priority,
				RuleID:          need.RuleID,
				Source:          need.Source,
				Confirmed:       need.Confirmed,
			})
		}
		result[staffID] = modelNeeds
	}
	return result
}
