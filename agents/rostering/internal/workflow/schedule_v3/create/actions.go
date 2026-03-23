// Package create 排班创建工作流 V3 - Actions 实现
//
// 渐进式排班流程：LLM评估所有需求，生成渐进式任务计划，分阶段执行
package create

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	"jusha/agent/rostering/internal/workflow/common"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/workflow/wsbridge"

	. "jusha/agent/rostering/internal/workflow/state/schedule"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/agent/rostering/internal/workflow/schedule_v3/executor"
	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// ============================================================
// 会话数据 Keys
// ============================================================

const (
	// KeyCreateV3Context 创建工作流 V3 上下文
	KeyCreateV3Context = "create_v3_context"
)

// ============================================================
// 工作流元数据辅助函数
// ============================================================

// setAllowUserInput 设置是否允许用户输入
func setAllowUserInput(ctx context.Context, wctx engine.Context, sessionID string, allow bool) error {
	_, err := wctx.SessionService().UpdateWorkflowMeta(ctx, sessionID, func(meta *session.WorkflowMeta) error {
		if meta.Extra == nil {
			meta.Extra = make(map[string]any)
		}
		meta.Extra["allowUserInput"] = allow
		return nil
	})
	return err
}

// buildScheduleDetailPreview 构建排班详情预览（显示任务相关的班次安排）
// 返回清晰的多班次排班格式（用于MultiShiftScheduleDialog）
// 重要：输出格式必须与前端 MultiShiftScheduleDialog.vue 的接口匹配
func buildScheduleDetailPreview(
	draft *d_model.ScheduleDraft,
	task *d_model.ProgressiveTask,
	batch *d_model.ScheduleChangeBatch,
	shifts []*d_model.Shift,
) map[string]any {
	if draft == nil || draft.Shifts == nil {
		return map[string]any{
			"shifts":        map[string]any{},
			"shiftInfoList": []map[string]any{},
		}
	}

	// 构建班次ID到信息的映射（包含 priority）
	shiftInfoMap := make(map[string]*d_model.Shift)
	for _, shift := range shifts {
		shiftInfoMap[shift.ID] = shift
	}

	shiftData := make(map[string]any)
	shiftInfoList := make([]map[string]any, 0)

	// 遍历任务涉及的班次
	for _, shiftID := range task.TargetShifts {
		shiftDraft, exists := draft.Shifts[shiftID]
		if !exists || shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		// 构建该班次的排班数据（格式：{ date: { staff, staffIds, requiredCount, actualCount } }）
		days := make(map[string]map[string]any)
		for _, date := range task.TargetDates {
			dayShift, dateExists := shiftDraft.Days[date]
			if dateExists && dayShift != nil {
				// 即使没有人员也添加该日期（显示"未安排"）
				days[date] = map[string]any{
					"staff":         dayShift.Staff,
					"staffIds":      dayShift.StaffIDs,
					"requiredCount": dayShift.RequiredCount,
					"actualCount":   dayShift.ActualCount,
				}
			} else {
				// 日期存在但没有排班数据，显示空状态
				days[date] = map[string]any{
					"staff":         []string{},
					"staffIds":      []string{},
					"requiredCount": 0,
					"actualCount":   0,
				}
			}
		}

		// 获取班次信息
		shift := shiftInfoMap[shiftID]
		priority := 0
		shiftName := shiftID
		if shift != nil {
			priority = shift.SchedulingPriority
			shiftName = shift.Name
		}

		// 只要有目标日期就添加班次（即使所有日期都无排班）
		if len(days) > 0 {
			shiftData[shiftID] = map[string]any{
				"shiftId":  shiftID,
				"priority": priority,
				"days":     days,
			}
			shiftInfoList = append(shiftInfoList, map[string]any{
				"id":   shiftID,
				"name": shiftName,
			})
		}
	}

	// 找出日期范围
	startDate := ""
	endDate := ""
	if len(task.TargetDates) > 0 {
		startDate = task.TargetDates[0]
		endDate = task.TargetDates[len(task.TargetDates)-1]
	}

	return map[string]any{
		"title":         task.Title, // 任务标题，用于对话框显示
		"shifts":        shiftData,
		"shiftInfoList": shiftInfoList,
		"startDate":     startDate,
		"endDate":       endDate,
	}
}

// convertBatchToChangeDetailPreview 将ScheduleChangeBatch转换为ChangeDetailPreview
func convertBatchToChangeDetailPreview(
	batch *d_model.ScheduleChangeBatch,
	shifts []*d_model.Shift,
) *d_model.ChangeDetailPreview {
	if batch == nil || len(batch.Changes) == 0 {
		return &d_model.ChangeDetailPreview{
			TaskID:    batch.TaskID,
			TaskTitle: batch.TaskTitle,
			TaskIndex: batch.TaskIndex,
			Timestamp: batch.Timestamp,
			Shifts:    make([]*d_model.ShiftChangePreview, 0),
		}
	}

	// 构建班次ID到名称的映射
	shiftNameMap := make(map[string]string)
	for _, shift := range shifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	// 按班次组织变更数据
	shiftChangesMap := make(map[string][]*d_model.DateChangePreview)

	for _, change := range batch.Changes {
		if change == nil {
			continue
		}

		// 构建日期变更预览
		dateChange := &d_model.DateChangePreview{
			Date:        change.Date,
			ChangeType:  change.ChangeType,
			BeforeIDs:   change.BeforeIDs,
			AfterIDs:    change.AfterIDs,
			BeforeNames: change.BeforeNames,
			AfterNames:  change.AfterNames,
		}

		// 按班次归类
		shiftChangesMap[change.ShiftID] = append(shiftChangesMap[change.ShiftID], dateChange)
	}

	// 转换为 ShiftChangePreview 列表
	shiftPreviews := make([]*d_model.ShiftChangePreview, 0, len(shiftChangesMap))
	for shiftID, changes := range shiftChangesMap {
		shiftPreview := &d_model.ShiftChangePreview{
			ShiftID:   shiftID,
			ShiftName: shiftNameMap[shiftID],
			Changes:   changes,
		}
		shiftPreviews = append(shiftPreviews, shiftPreview)
	}

	return &d_model.ChangeDetailPreview{
		TaskID:    batch.TaskID,
		TaskTitle: batch.TaskTitle,
		TaskIndex: batch.TaskIndex,
		Timestamp: batch.Timestamp,
		Shifts:    shiftPreviews,
	}
}

// buildSchedulePreviewData 构建完整排班预览数据
func buildSchedulePreviewData(createCtx *CreateV3Context) map[string]any {
	if createCtx.WorkingDraft == nil {
		return make(map[string]any)
	}

	// 使用上下文中的日期范围
	return map[string]any{
		"draftSchedule": createCtx.WorkingDraft,
		"startDate":     createCtx.StartDate,
		"endDate":       createCtx.EndDate,
	}
}

// isContinuousSchedulingEnabled 检查是否启用连续排班（默认关闭）
func isContinuousSchedulingEnabled(ctx context.Context, wctx engine.Context, orgID string) bool {
	logger := wctx.Logger()
	rosteringService, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !serviceOk {
		logger.Error("Rostering service not available in service registry",
			"orgID", orgID,
			"serviceKey", engine.ServiceKeyRostering)
		return false
	}

	continuousSetting, err := rosteringService.GetSystemSetting(ctx, orgID, "continuous_scheduling")
	if err != nil {
		logger.Error("Failed to get continuous_scheduling setting",
			"error", err,
			"orgID", orgID)
		return false
	}

	return continuousSetting == "true"
}

// serializePayload 将结构体序列化为 map[string]any
func serializePayload(payload any) map[string]any {
	if payload == nil {
		return nil
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}
	return result
}

// buildProgressiveTaskPlan 生成渐进式任务计划
func buildProgressiveTaskPlan(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) (*d_model.ProgressiveTaskPlan, error) {
	logger := wctx.Logger()

	// 1. 先直接填充固定排班（不需要LLM参与）
	if err := fillFixedShiftSchedules(ctx, wctx, createCtx); err != nil {
		logger.Warn("Failed to fill fixed shift schedules", "error", err)
		// 不影响后续流程
	}

	// 【新增】保存基准排班（深拷贝 WorkingDraft）
	if createCtx.WorkingDraft != nil {
		baselineCopy, err := utils.DeepCopyScheduleDraft(createCtx.WorkingDraft)
		if err != nil {
			logger.Warn("Failed to deep copy baseline schedule", "error", err)
		} else {
			createCtx.BaselineSchedule = baselineCopy
			logger.Info("Baseline schedule saved",
				"shiftsCount", len(createCtx.BaselineSchedule.Shifts))
		}
	}

	// 【新增】初始化变更历史
	createCtx.ChangeBatches = make([]*d_model.ScheduleChangeBatch, 0)

	// 2. 调用需求评估服务（从 ServiceRegistry 获取）
	progressiveService, ok := engine.GetService[utils.IProgressiveSchedulingService](wctx, engine.ServiceKeyProgressiveScheduling)
	if !ok {
		return nil, fmt.Errorf("progressive scheduling service not available")
	}

	// 转换个人需求格式：从 create.PersonalNeed 转换为 d_model.PersonalNeed
	personalNeedsMap := convertPersonalNeedsToModel(createCtx.PersonalNeeds)

	// 【P1优化】获取固定排班配置并缓存（用于AI评估和后续传递给子工作流）
	fixedShiftAssignments, err := getFixedShiftAssignments(ctx, wctx, createCtx)
	if err != nil {
		logger.Warn("Failed to get fixed shift assignments", "error", err)
		// 如果获取失败，使用空 slice，不影响后续流程
		fixedShiftAssignments = []d_model.CtxFixedShiftAssignment{}
	}
	// 缓存到上下文，避免 spawnCurrentTask 中重复获取
	createCtx.FixedAssignments = fixedShiftAssignments

	// 【重要】AssessRequirementsAndPlanTasks 需要合并格式（date -> staffIDs）
	// 用于 AI 评估时了解哪些人员在哪些日期是固定排班的
	fixedShiftAssignmentsMerged := mergeFixedShiftAssignmentsForAI(fixedShiftAssignments)

	// 转换 StaffRequirements 为旧格式（AssessRequirementsAndPlanTasks 尚未更新）
	staffRequirementsMap := d_model.ConvertRequirementsToMap(createCtx.StaffRequirements)

	return progressiveService.AssessRequirementsAndPlanTasks(
		ctx,
		createCtx.SelectedShifts,
		createCtx.Rules,
		personalNeedsMap,
		fixedShiftAssignmentsMerged,
		staffRequirementsMap,
		createCtx.AllStaff,
		createCtx.StartDate,
		createCtx.EndDate,
	)
}

// mergeFixedShiftAssignmentsForAI 合并所有班次的固定排班数据（用于AI评估）
// 输入: []d_model.CtxFixedShiftAssignment
// 输出: date -> staffIDs（所有班次合并，去重）
func mergeFixedShiftAssignmentsForAI(fixedShiftAssignments []d_model.CtxFixedShiftAssignment) map[string][]string {
	result := make(map[string][]string)
	for _, assignment := range fixedShiftAssignments {
		// 合并同一天的固定排班人员（去重）
		existingStaffIDs := result[assignment.Date]
		staffIDMap := make(map[string]bool)
		for _, id := range existingStaffIDs {
			staffIDMap[id] = true
		}
		for _, id := range assignment.StaffIDs {
			if !staffIDMap[id] {
				result[assignment.Date] = append(result[assignment.Date], id)
				staffIDMap[id] = true
			}
		}
	}
	return result
}

// fillFixedShiftSchedules 直接填充固定排班（不通过LLM任务）
func fillFixedShiftSchedules(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()

	// 获取 rosteringService
	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rostering service not available")
	}

	// 提取所有班次ID
	shiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		shiftIDs = append(shiftIDs, shift.ID)
	}

	if len(shiftIDs) == 0 {
		return nil
	}

	logger.Info("Filling fixed shift schedules", "shiftCount", len(shiftIDs))

	// 调用 IRosteringService 获取固定排班数据（保留班次信息）
	allFixedSchedules, err := rosteringService.CalculateMultipleFixedSchedules(
		ctx,
		shiftIDs,
		createCtx.StartDate,
		createCtx.EndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to calculate fixed schedules: %w", err)
	}

	if len(allFixedSchedules) == 0 {
		logger.Info("No fixed shift schedules found")
		return nil
	}

	// 初始化 WorkingDraft
	if createCtx.WorkingDraft == nil {
		createCtx.WorkingDraft = &d_model.ScheduleDraft{
			StartDate: createCtx.StartDate,
			EndDate:   createCtx.EndDate,
			Shifts:    make(map[string]*d_model.ShiftDraft),
		}
	}
	if createCtx.WorkingDraft.Shifts == nil {
		createCtx.WorkingDraft.Shifts = make(map[string]*d_model.ShiftDraft)
	}

	// 构建人员ID到姓名的映射
	staffNames := make(map[string]string)
	for _, s := range createCtx.AllStaff {
		staffNames[s.ID] = s.Name
	}

	// 直接填充固定排班到 WorkingDraft
	filledCount := 0
	occupiedCount := 0 // 【P2优化】统计占位信息更新数量
	for shiftID, schedule := range allFixedSchedules {
		if len(schedule) == 0 {
			continue
		}

		// 确保班次存在
		shiftDraft := createCtx.WorkingDraft.Shifts[shiftID]
		if shiftDraft == nil {
			shiftDraft = &d_model.ShiftDraft{
				ShiftID: shiftID,
				Days:    make(map[string]*d_model.DayShift),
			}
			createCtx.WorkingDraft.Shifts[shiftID] = shiftDraft
		}
		if shiftDraft.Days == nil {
			shiftDraft.Days = make(map[string]*d_model.DayShift)
		}

		// 填充每一天的固定排班
		for date, staffIDs := range schedule {
			// 将人员ID转换为姓名
			staff := make([]string, 0, len(staffIDs))
			for _, id := range staffIDs {
				if name, ok := staffNames[id]; ok {
					staff = append(staff, name)
				} else {
					staff = append(staff, id)
				}
			}

			// 获取该日期该班次的需求人数
			requiredCount := createCtx.GetRequirement(shiftID, date)

			shiftDraft.Days[date] = &d_model.DayShift{
				Staff:         staff,
				StaffIDs:      staffIDs,
				RequiredCount: requiredCount,
				ActualCount:   len(staffIDs),
				IsFixed:       true, // 【P1修复】标记为固定排班
			}

			// 【P2优化】更新占位信息：将固定排班的人员标记为已占用（紧跟数据填充，保证原子性）
			for _, staffID := range staffIDs {
				createCtx.OccupySlot(staffID, date, shiftID)
				occupiedCount++
			}
		}

		filledCount++
	}

	logger.Info("Fixed shift schedules filled",
		"shiftsWithFixed", filledCount,
		"occupiedSlots", occupiedCount)
	return nil
}

// validateDataConsistency 验证排班数据一致性（V3改进：实现完整校验逻辑）
// 检查以下几个方面：
// 1. OccupiedSlots与WorkingDraft是否一致
// 2. 是否有无效的人员ID
// 3. 是否有超出需求数量的排班
// 注意：不再检查"同一人员同一天多个班次"，因为某些场景下这是合理的（如跨夜班、值班叠加等）
// 返回警告信息列表，空数组表示无问题
func validateDataConsistency(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) []string {
	warnings := make([]string, 0)

	if createCtx.WorkingDraft == nil {
		return warnings
	}

	// 构建有效人员ID集合
	validStaffIDs := make(map[string]bool)
	for _, staff := range createCtx.AllStaff {
		validStaffIDs[staff.ID] = true
	}

	// 从WorkingDraft重建占位表，与OccupiedSlots对比
	// 注意：这里允许同一人员同一天有多个班次（用数组而非单一值）
	rebuiltSlots := make(map[string]map[string][]string) // staffID -> date -> []shiftIDs
	for shiftID, shiftDraft := range createCtx.WorkingDraft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}

		for date, dayShift := range shiftDraft.Days {
			if dayShift == nil {
				continue
			}

			for _, staffID := range dayShift.StaffIDs {
				// 检查人员ID有效性
				if !validStaffIDs[staffID] {
					warnings = append(warnings, fmt.Sprintf(
						"无效人员ID: %s (班次=%s, 日期=%s)", staffID, shiftID, date))
					continue
				}

				// 记录占位（允许多个班次）
				if rebuiltSlots[staffID] == nil {
					rebuiltSlots[staffID] = make(map[string][]string)
				}
				rebuiltSlots[staffID][date] = append(rebuiltSlots[staffID][date], shiftID)
			}

			// 检查是否超出需求数量
			requiredCount := createCtx.GetRequirement(shiftID, date)
			if requiredCount > 0 && len(dayShift.StaffIDs) > requiredCount {
				warnings = append(warnings, fmt.Sprintf(
					"超出需求: 班次 %s 在 %s 需要 %d 人，实际安排 %d 人",
					shiftID, date, requiredCount, len(dayShift.StaffIDs)))
			}
		}
	}

	// 对比重建的占位表与实际占位表
	// 注意：新的 OccupiedSlots 是数组结构，需要转换为 map 进行对比
	// 这里只检查最基本的不一致性，不强制要求完全匹配
	occupiedSlotsMap := d_model.ConvertOccupiedSlotsToMap(createCtx.OccupiedSlots)
	if len(occupiedSlotsMap) > 0 {
		for staffID, dates := range rebuiltSlots {
			if occupiedSlotsMap[staffID] == nil {
				warnings = append(warnings, fmt.Sprintf(
					"占位表不一致: 人员 %s 在WorkingDraft中有排班，但OccupiedSlots中缺失", staffID))
				continue
			}

			for date, shiftIDs := range dates {
				occupiedShift := occupiedSlotsMap[staffID][date]
				// 检查OccupiedSlots中的班次是否在WorkingDraft的班次列表中
				found := false
				for _, sid := range shiftIDs {
					if sid == occupiedShift {
						found = true
						break
					}
				}
				if !found && occupiedShift != "" {
					warnings = append(warnings, fmt.Sprintf(
						"占位表不一致: 人员 %s 在 %s 的OccupiedSlots记录(%s)不在WorkingDraft的班次列表中(%v)",
						staffID, date, occupiedShift, shiftIDs))
				}
			}
		}

		// 检查OccupiedSlots中是否有WorkingDraft中没有的记录
		for staffID, dates := range occupiedSlotsMap {
			if rebuiltSlots[staffID] == nil {
				warnings = append(warnings, fmt.Sprintf(
					"占位表不一致: 人员 %s 在OccupiedSlots中有记录，但WorkingDraft中缺失", staffID))
				continue
			}

			for date := range dates {
				if _, ok := rebuiltSlots[staffID][date]; !ok {
					warnings = append(warnings, fmt.Sprintf(
						"占位表不一致: 人员 %s 在 %s 的记录存在于OccupiedSlots但不在WorkingDraft中",
						staffID, date))
				}
			}
		}
	}

	return warnings
}

// buildFixedSchedulePreviewData 构建固定排班预览数据
func buildFixedSchedulePreviewData(createCtx *CreateV3Context) map[string]any {
	if createCtx.WorkingDraft == nil || len(createCtx.WorkingDraft.Shifts) == 0 {
		return nil
	}

	// 构建班次信息列表
	shiftInfoList := make([]map[string]any, 0)
	for _, shift := range createCtx.SelectedShifts {
		shiftInfoList = append(shiftInfoList, map[string]any{
			"id":       shift.ID,
			"name":     shift.Name,
			"priority": shift.SchedulingPriority,
		})
	}

	// 构建班次排班数据
	shiftsData := make(map[string]any)
	for _, shift := range createCtx.SelectedShifts {
		shiftDraft := createCtx.WorkingDraft.Shifts[shift.ID]

		// 构建每日排班数据
		days := make(map[string]any)
		if shiftDraft != nil && len(shiftDraft.Days) > 0 {
			for date, dayShift := range shiftDraft.Days {
				days[date] = map[string]any{
					"staff":         dayShift.Staff,
					"staffIds":      dayShift.StaffIDs,
					"requiredCount": dayShift.RequiredCount,
					"actualCount":   dayShift.ActualCount,
				}
			}
		}

		shiftsData[shift.ID] = map[string]any{
			"shiftId":  shift.ID,
			"priority": shift.SchedulingPriority,
			"days":     days,
		}
	}

	return map[string]any{
		"shifts":        shiftsData,
		"startDate":     createCtx.StartDate,
		"endDate":       createCtx.EndDate,
		"shiftInfoList": shiftInfoList,
	}
}

// buildFixedSchedulePreviewMessage 构建固定排班预览消息
func buildFixedSchedulePreviewMessage(previewData map[string]any) string {
	if previewData == nil {
		return ""
	}

	var message strings.Builder
	message.WriteString("📊 **固定排班预览**\n\n")

	// 统计信息
	shiftsWithData := 0
	totalDays := 0
	totalAssignments := 0

	if shifts, ok := previewData["shifts"].(map[string]any); ok {
		for _, shiftValue := range shifts {
			if shiftData, ok := shiftValue.(map[string]any); ok {
				if days, ok := shiftData["days"].(map[string]any); ok {
					if len(days) > 0 {
						shiftsWithData++
						totalDays += len(days)
						for _, dayValue := range days {
							if day, ok := dayValue.(map[string]any); ok {
								if actualCount, ok := day["actualCount"].(int); ok {
									totalAssignments += actualCount
								}
							}
						}
					}
				}
			}
		}
	}

	if shiftsWithData > 0 {
		message.WriteString(fmt.Sprintf("已为 **%d** 个班次填充固定排班，", shiftsWithData))
		message.WriteString(fmt.Sprintf("共 **%d** 天，**%d** 人次。\n\n", totalDays, totalAssignments))
		message.WriteString("✅ 这些班次的固定人员配置已自动填充，无需AI生成。\n")
	} else {
		message.WriteString("暂无固定排班数据。\n")
	}

	return message.String()
}

// buildPersonalNeedsPreviewMessage 构建个人需求预览消息
func buildPersonalNeedsPreviewMessage(createCtx *CreateV3Context) string {
	if createCtx == nil || len(createCtx.PersonalNeeds) == 0 {
		return "📋 **需求预览**\n\n未解析到任何个人需求。\n\n请确认是否继续生成排班计划。"
	}

	var message strings.Builder
	message.WriteString("📋 **需求预览**\n\n")
	message.WriteString("已解析出以下个人需求：\n\n")

	// 统计正向和负向需求数量
	positiveCount := 0
	negativeCount := 0

	for staffID, needs := range createCtx.PersonalNeeds {
		if len(needs) == 0 {
			continue
		}

		// 获取员工名称（如果有）
		staffName := staffID
		for _, staff := range createCtx.AllStaff {
			if staff.ID == staffID {
				staffName = staff.Name
				break
			}
		}
		if staffName == staffID {
			for _, staff := range createCtx.AllStaff {
				if staff.ID == staffID {
					staffName = staff.Name
					break
				}
			}
		}

		message.WriteString(fmt.Sprintf("**%s**：\n", staffName))
		for i, need := range needs {
			// 区分正向和负向需求
			// 正向需求：明确要求在指定日期上指定班次（RequestType为prefer/must且指定了TargetShiftID）
			// 负向需求：回避某日期/某班次，或要求休息（RequestType为avoid，或未指定TargetShiftID）
			// 注意：RequestType为prefer/must但未指定TargetShiftID视为负向需求（表示休息或回避）
			isPositive := (need.RequestType == "prefer" || need.RequestType == "must") && need.TargetShiftID != ""
			if isPositive {
				positiveCount++
				message.WriteString(fmt.Sprintf("  %d. ✅ **[正向需求]** %s", i+1, need.Description))
			} else {
				negativeCount++
				message.WriteString(fmt.Sprintf("  %d. 🚫 **[负向需求]** %s", i+1, need.Description))
			}

			if len(need.TargetDates) > 0 {
				message.WriteString(fmt.Sprintf(" (日期: %s)", strings.Join(need.TargetDates, ", ")))
			}
			message.WriteString("\n")
		}
		message.WriteString("\n")
	}

	totalNeeds := positiveCount + negativeCount
	message.WriteString(fmt.Sprintf("**共计**：%d 个需求（正向需求 %d 个，负向需求 %d 个）\n\n", totalNeeds, positiveCount, negativeCount))
	message.WriteString("**说明**：\n")
	message.WriteString("- ✅ **正向需求**（要求在指定日期上指定班次）：将在固定排班后优先处理\n")
	message.WriteString("- 🚫 **负向需求**（要求不在指定日期上班）：将在班次填充时考虑，避开这些人员和日期\n\n")
	message.WriteString("请确认需求是否正确。如需修改，可以重新填写需求文本。")

	return message.String()
}

// buildPlanPreviewMessage 构建任务计划预览消息
func buildPlanPreviewMessage(taskPlan *d_model.ProgressiveTaskPlan) string {
	if taskPlan == nil {
		return "⚠️ 未生成任务计划，请重试。"
	}
	message := fmt.Sprintf("📌 **任务计划预览**\n\n共 %d 个任务：\n\n", len(taskPlan.Tasks))
	if taskPlan.Summary != "" {
		message += fmt.Sprintf("**规划说明**：%s\n\n", taskPlan.Summary)
	}
	message += "**任务列表**：\n"
	for i, task := range taskPlan.Tasks {
		message += fmt.Sprintf("%d. %s\n", i+1, task.Title)
	}
	message += "\n请选择是否开始执行计划，或提出修改意见。"
	return message
}

// ============================================================
// 阶段 0: 初始化
// ============================================================

// actStartInfoCollect 启动信息收集子工作流
func actStartInfoCollect(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting workflow", "sessionID", sess.ID)

	// 初始化工作流上下文
	createCtx := NewCreateV3Context()
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 禁用用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to set allowUserInput", "error", err)
	}

	// 发送流程转换消息
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID,
		"🚀 开始创建渐进式排班方案（V3）"); err != nil {
		logger.Warn("Failed to send welcome message", "error", err)
	}

	// 触发信息收集完成事件
	return wctx.Send(ctx, CreateV3EventInfoCollected, nil)
}

// ============================================================
// 阶段 1: 信息收集完成处理
// ============================================================

// actOnInfoCollected 处理信息收集完成事件
func actOnInfoCollected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Info collection completed", "sessionID", sess.ID)

	// 加载上下文
	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从 InfoCollect 子工作流的输出中提取数据
	if err := populateInfoFromSubWorkflow(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to populate info: %w", err)
	}

	// 从规则中提取个人需求
	personalNeeds := ExtractPersonalNeeds(createCtx.Rules, createCtx.AllStaff)
	createCtx.PersonalNeeds = personalNeeds

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 开始时间确认阶段
	return startPeriodConfirmPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 1.5: 确认排班时间
// ============================================================

// startPeriodConfirmPhase 开始时间确认阶段
func startPeriodConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting period confirmation phase", "sessionID", sess.ID)

	dateRangeDisplay := common.FormatDateRangeForDisplay(createCtx.StartDate, createCtx.EndDate)
	message := fmt.Sprintf("📅 **请确认排班时间范围**\n\n当前时间范围：**%s**\n\n", dateRangeDisplay)
	message += "如需修改，请点击「修改时间」按钮。"

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认时间",
			Event: session.WorkflowEvent(CreateV3EventPeriodConfirmed),
			Style: session.ActionStylePrimary,
			Payload: serializePayload(&PeriodConfirmPayload{
				StartDate: createCtx.StartDate,
				EndDate:   createCtx.EndDate,
			}),
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改时间",
			Event: session.WorkflowEvent(CreateV3EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actOnPeriodConfirmed 处理排班时间确认
func actOnPeriodConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Period confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 解析payload（如果有）
	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			if startDate, ok := payloadMap["startDate"].(string); ok {
				createCtx.StartDate = startDate
			}
			if endDate, ok := payloadMap["endDate"].(string); ok {
				createCtx.EndDate = endDate
			}
		}
	}

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 进入班次确认阶段
	return startShiftsConfirmPhase(ctx, wctx, createCtx)
}

// actModifyPeriod 修改排班时间
func actModifyPeriod(ctx context.Context, wctx engine.Context, payload any) error {
	// 重新显示时间确认界面
	return startPeriodConfirmPhase(ctx, wctx, nil)
}

// ============================================================
// 阶段 1.6: 确认班次选择
// ============================================================

// startShiftsConfirmPhase 开始班次确认阶段
func startShiftsConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting shifts confirmation phase", "sessionID", sess.ID)

	if createCtx == nil {
		var err error
		createCtx, err = loadCreateV3Context(ctx, wctx)
		if err != nil {
			return fmt.Errorf("failed to load context: %w", err)
		}
	}

	// 如果还没有加载班次信息，先加载
	if len(createCtx.SelectedShifts) == 0 {
		if err := populateInfoFromSubWorkflow(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to populate info: %w", err)
		}
		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}
	}

	// 构建班次确认消息
	message := "🏢 **请确认排班班次**\n\n"
	message += fmt.Sprintf("共 **%d** 个班次需要排班：\n\n", len(createCtx.SelectedShifts))

	// 显示每个班次的基本信息
	for i, shift := range createCtx.SelectedShifts {
		if i >= 10 { // 最多显示10个
			message += fmt.Sprintf("\n... 还有 %d 个班次\n", len(createCtx.SelectedShifts)-10)
			break
		}

		message += fmt.Sprintf("  • **%s**", shift.Name)
		description := fmt.Sprintf("%s-%s", shift.StartTime, shift.EndTime)
		if shift.IsOvernight {
			description += " (跨夜)"
		}
		message += fmt.Sprintf("：%s\n", description)
	}

	message += "\n确认班次后，将进入人数配置阶段。如需修改班次选择，请点击「修改班次」按钮。"

	// 获取服务以加载班次分组人员数据
	service, serviceOk := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !serviceOk {
		return fmt.Errorf("rosteringService not found")
	}

	// 加载每个班次的分组人员数据（用于预览）
	shiftGroupsData := make([]map[string]any, 0, len(createCtx.SelectedShifts))
	totalStaff := 0
	allStaffMap := make(map[string]bool) // 用于统计总人数（去重）

	for _, shift := range createCtx.SelectedShifts {
		members, err := service.GetShiftGroupMembers(ctx, shift.ID)
		if err != nil {
			logger.Warn("Failed to get members for shift", "shiftID", shift.ID, "error", err)
			continue
		}

		// 统计去重后的总人数
		for _, m := range members {
			if !allStaffMap[m.ID] {
				allStaffMap[m.ID] = true
				totalStaff++
			}
		}

		// 构建班次人员数据
		staffList := make([]map[string]any, 0, len(members))
		for _, m := range members {
			staffItem := map[string]any{
				"id":   m.ID,
				"name": m.Name,
			}
			if m.Position != "" {
				staffItem["position"] = m.Position
			}
			if m.DepartmentID != "" {
				staffItem["departmentId"] = m.DepartmentID
			}
			staffList = append(staffList, staffItem)
		}

		shiftGroupsData = append(shiftGroupsData, map[string]any{
			"shiftId":    shift.ID,
			"shiftName":  shift.Name,
			"startTime":  shift.StartTime,
			"endTime":    shift.EndTime,
			"staffCount": len(staffList),
			"staffList":  staffList,
		})
	}

	message += fmt.Sprintf("\n\n涉及人员总数：**%d** 人", totalStaff)

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认班次",
			Event: session.WorkflowEvent(CreateV3EventShiftsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改班次",
			Event: session.WorkflowEvent(CreateV3EventUserModify),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 发送消息
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":            "shiftsConfirm",
			"shiftGroupsData": shiftGroupsData,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send shifts confirm message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnShiftsConfirmed 处理班次确认
func actOnShiftsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Shifts confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 解析payload（如果有）
	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			if shiftIDs, ok := payloadMap["shiftIds"].([]any); ok {
				// 更新选定的班次
				// 获取所有可用班次
				rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
				if !ok {
					return fmt.Errorf("rosteringService not found")
				}

				allShifts, err := rosteringService.ListShifts(ctx, sess.OrgID, "")
				if err != nil {
					return fmt.Errorf("failed to list shifts: %w", err)
				}

				// 构建 shiftID -> shift 映射
				shiftMap := make(map[string]*d_model.Shift)
				for _, shift := range allShifts {
					shiftMap[shift.ID] = shift
				}

				// 根据 payload 中的 shiftIDs 更新 SelectedShifts
				newSelectedShifts := make([]*d_model.Shift, 0, len(shiftIDs))
				for _, idVal := range shiftIDs {
					shiftID, ok := idVal.(string)
					if !ok {
						continue
					}
					if shift, exists := shiftMap[shiftID]; exists {
						newSelectedShifts = append(newSelectedShifts, shift)
					}
				}

				// 如果用户选择了班次，更新 SelectedShifts
				if len(newSelectedShifts) > 0 {
					createCtx.SelectedShifts = newSelectedShifts
					logger.Info("Updated selected shifts", "count", len(newSelectedShifts))
				}
			}
		}
	}

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 进入人数配置阶段
	return startStaffCountConfirmPhase(ctx, wctx, createCtx)
}

// actModifyShifts 修改班次选择
func actModifyShifts(ctx context.Context, wctx engine.Context, payload any) error {
	// 重新显示班次确认界面
	return startShiftsConfirmPhase(ctx, wctx, nil)
}

// ============================================================
// 阶段 1.7: 确认人数配置
// ============================================================

// startStaffCountConfirmPhase 开始人数配置确认阶段
func startStaffCountConfirmPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting staff count confirmation phase", "sessionID", sess.ID)

	if createCtx == nil {
		var err error
		createCtx, err = loadCreateV3Context(ctx, wctx)
		if err != nil {
			return fmt.Errorf("failed to load context: %w", err)
		}
	}

	// 构建人数配置表单字段
	staffCountFields := buildStaffCountFieldsV3(createCtx)
	if len(staffCountFields) == 0 {
		if err := populateInfoFromSubWorkflow(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to populate info: %w", err)
		}
		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}
		staffCountFields = buildStaffCountFieldsV3(createCtx)
	}

	// 构建消息
	message := "📊 **配置每个班次每天需要的人数**\n\n"
	message += fmt.Sprintf("排班周期：**%s** 至 **%s**\n\n", createCtx.StartDate, createCtx.EndDate)
	message += fmt.Sprintf("共 **%d** 个班次需要配置人数。\n\n", len(createCtx.SelectedShifts))
	message += "请为每个班次配置每天需要的人数，可以按日期分别设置，也可以统一设置。\n\n"
	message += "💡 **提示**：如果不配置，系统将使用默认人数要求。"

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认人数配置",
			Event: session.WorkflowEvent(CreateV3EventStaffCountConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:   session.ActionTypeWorkflow,
			Label:  "修改人数配置",
			Event:  session.WorkflowEvent(CreateV3EventStaffCountConfirmed),
			Style:  session.ActionStyleSecondary,
			Fields: staffCountFields,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 发送消息
	mainMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: nil, // 工作流操作按钮不放在消息上
		Metadata: map[string]any{
			"type":   "staffCountConfirm",
			"fields": staffCountFields,
		},
	}
	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, mainMsg); err != nil {
		logger.Warn("Failed to send staff count confirm message", "error", err)
	}

	// 设置工作流 meta（包含工作流操作按钮和表单字段）
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnStaffCountConfirmed 处理人数配置确认
func actOnStaffCountConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Staff count confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 解析payload（如果有）
	if payload != nil {
		var payloadMap map[string]any
		if err := parsePayloadToMap(payload, &payloadMap); err == nil {
			// 更新人数配置
			if err := parseStaffCountPayloadV3(payloadMap, createCtx); err != nil {
				logger.Warn("Failed to parse staff count payload", "error", err)
			} else {
				logger.Info("Parsed staff count payload", "shiftsCount", len(createCtx.SelectedShifts))
			}
		}
	}

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 进入个人需求阶段
	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// actModifyStaffCount 修改人数配置
func actModifyStaffCount(ctx context.Context, wctx engine.Context, payload any) error {
	// 重新显示人数配置界面
	return startStaffCountConfirmPhase(ctx, wctx, nil)
}

// ============================================================
// 阶段 2: 个人需求确认
// ============================================================

// actOnPersonalNeedsConfirmed 处理个人需求确认
func actOnPersonalNeedsConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Personal needs confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	createCtx.PersonalNeedsConfirmed = true

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 显示"正在生成排班计划"的消息
	planningMsg := "🔄 正在生成排班计划，请稍候..."
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, planningMsg); err != nil {
		logger.Warn("Failed to send planning message", "error", err)
	}

	// 进入需求评估阶段（生成计划）
	return wctx.Send(ctx, CreateV3EventRequirementAssessed, nil)
}

// actOnTemporaryNeedsTextSubmitted 处理临时需求文本提交
func actOnTemporaryNeedsTextSubmitted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Temporary needs text submitted", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 提取文本
	var requirementText string
	switch p := payload.(type) {
	case map[string]any:
		if v, ok := p["requirementText"].(string); ok {
			requirementText = v
		} else if v, ok := p["text"].(string); ok {
			requirementText = v
		}
	case string:
		requirementText = p
	}
	requirementText = strings.TrimSpace(requirementText)
	if requirementText == "" {
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "未收到需求文本，请重新填写。"); err != nil {
			logger.Warn("Failed to send empty requirement text message", "error", err)
		}
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	// 立即发送"正在解析需求"的消息
	parsingMsg := "📝 正在解析临时需求..."
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, parsingMsg); err != nil {
		logger.Warn("Failed to send parsing message", "error", err)
	}

	// 调用LLM解析需求
	if err := applyTemporaryNeedsText(ctx, wctx, createCtx, requirementText); err != nil {
		errorMsg := fmt.Sprintf("❌ 解析需求失败：%v\n\n请重新填写需求。", err)
		if _, msgErr := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); msgErr != nil {
			logger.Warn("Failed to send error message", "error", msgErr)
		}
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 解析完成，显示需求预览
	successMsg := "✅ 需求解析完成！"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, successMsg); err != nil {
		logger.Warn("Failed to send success message", "error", err)
	}

	// 显示解析出来的需求预览
	previewMsg := buildPersonalNeedsPreviewMessage(createCtx)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, previewMsg); err != nil {
		logger.Warn("Failed to send needs preview message", "error", err)
	}

	// 设置工作流按钮：确认需求、重新填写、取消
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认需求并生成计划",
			Event: session.WorkflowEvent(CreateV3EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "重新填写需求",
			Event: session.WorkflowEvent(CreateV3EventTemporaryNeedsTextSubmitted),
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "requirementText",
					Label:       "临时需求文本",
					Type:        session.FieldTypeTextarea,
					Required:    true,
					Placeholder: "请粘贴需求文本，例如：\n1. 张三 1月12-14日出差，不能排班\n2. 李四 1月16日希望上早班\n3. 1月20日夜班需增加1人",
					Extra: map[string]any{
						"markdown": true,
						"rows":     8,
					},
				},
			},
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, workflowActions)
}

// applyTemporaryNeedsText 解析并应用临时需求文本（更新个人需求）
func applyTemporaryNeedsText(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context, requirementText string) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	createCtx.TemporaryNeedsText = requirementText

	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		return fmt.Errorf("schedulingAIService not found")
	}

	allStaff := createCtx.AllStaff
	if len(allStaff) == 0 {
		return fmt.Errorf("no staff available for scheduling")
	}

	temporaryNeeds, err := aiService.ExtractTemporaryNeeds(
		ctx,
		requirementText,
		allStaff,
		createCtx.StartDate,
		createCtx.EndDate,
		sess.Messages,
	)
	if err != nil {
		logger.Warn("CreateV3: Failed to extract temporary needs", "error", err)
		temporaryNeeds = []*d_model.PersonalNeed{}
	}

	// 清除已有的用户临时需求，避免重复叠加
	for staffID, needs := range createCtx.PersonalNeeds {
		filteredNeeds := make([]*PersonalNeed, 0)
		for _, need := range needs {
			if need.NeedType == "temporary" && need.Source == "user" {
				continue
			}
			filteredNeeds = append(filteredNeeds, need)
		}
		if len(filteredNeeds) > 0 {
			createCtx.PersonalNeeds[staffID] = filteredNeeds
		} else {
			delete(createCtx.PersonalNeeds, staffID)
		}
	}

	// 添加提取的临时需求
	for _, need := range temporaryNeeds {
		if need == nil {
			continue
		}
		createCtx.AddPersonalNeed(&PersonalNeed{
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
			Source:          "user",
			Confirmed:       true,
		})
	}

	createCtx.PersonalNeedsConfirmed = true
	return nil
}

// startPersonalNeedsPhase 开始个人需求确认阶段
func startPersonalNeedsPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting personal needs phase", "sessionID", sess.ID)

	// 构建个人需求消息
	message := "📋 **请添加临时需求**\n\n"
	message += "您可以：\n"
	message += "1. 直接开始生成排班计划（跳过临时需求）\n"
	message += "2. 添加临时需求（粘贴文本描述），系统将基于这些需求生成计划\n\n"
	message += "**提示**：临时需求支持 Markdown 格式，多行粘贴即可。"

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "直接生成计划",
			Event: session.WorkflowEvent(CreateV3EventPersonalNeedsConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "添加临时需求",
			Event: session.WorkflowEvent(CreateV3EventTemporaryNeedsTextSubmitted),
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "requirementText",
					Label:       "临时需求文本",
					Type:        session.FieldTypeTextarea,
					Required:    true,
					Placeholder: "请粘贴需求文本，例如：\n1. 张三 1月12-14日出差，不能排班\n2. 李四 1月16日希望上早班\n3. 1月20日夜班需增加1人",
					Extra: map[string]any{
						"markdown": true,
						"rows":     8,
					},
				},
			},
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, message, workflowActions)
}

// actModifyPersonalNeeds 修改个人需求（兼容保留）
func actModifyPersonalNeeds(ctx context.Context, wctx engine.Context, payload any) error {
	// 重新显示个人需求界面
	return startPersonalNeedsPhase(ctx, wctx, nil)
}

// actReturnToPersonalNeedsConfirm 返回到个人需求确认
func actReturnToPersonalNeedsConfirm(ctx context.Context, wctx engine.Context, payload any) error {
	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	return startPersonalNeedsPhase(ctx, wctx, createCtx)
}

// ============================================================
// 阶段 3: 需求评估（V3核心）
// ============================================================

// actOnRequirementAssessed 处理需求评估完成
func actOnRequirementAssessed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting requirement assessment", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 调用LLM生成任务计划
	taskPlan, err := buildProgressiveTaskPlan(ctx, wctx, createCtx)
	if err != nil {
		logger.Error("Failed to build progressive task plan", "error", err)
		// 显示错误消息并返回个人需求阶段
		errorMsg := fmt.Sprintf("❌ 生成排班计划失败：%v\n\n请检查需求后重试。", err)
		if _, msgErr := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); msgErr != nil {
			logger.Warn("Failed to send error message", "error", msgErr)
		}
		return startPersonalNeedsPhase(ctx, wctx, createCtx)
	}

	createCtx.ProgressiveTaskPlan = taskPlan
	createCtx.CurrentTaskIndex = 0
	createCtx.TaskResults = make(map[string]*d_model.TaskResult)
	createCtx.CompletedTaskCount = 0
	createCtx.FailedTaskCount = 0
	createCtx.SkippedTaskCount = 0
	// 注意：不要清空 WorkingDraft，它已经包含了固定排班数据
	// createCtx.WorkingDraft = nil
	createCtx.AdjustmentScope = ""

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 禁用用户输入，等待用户确认计划
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to disable user input", "error", err)
	}

	// 发送计划生成完成的系统消息
	completionMsg := "✅ 排班计划已生成！"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, completionMsg); err != nil {
		logger.Warn("Failed to send completion message", "error", err)
	}

	// 发送任务计划预览消息（包含详细计划内容）
	message := buildPlanPreviewMessage(taskPlan)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send task plan preview message", "error", err)
	}

	// 如果已经填充了固定排班，发送固定排班预览
	if createCtx.WorkingDraft != nil && len(createCtx.WorkingDraft.Shifts) > 0 {
		fixedSchedulePreview := buildFixedSchedulePreviewData(createCtx)
		if fixedSchedulePreview != nil {
			fixedMsgText := buildFixedSchedulePreviewMessage(fixedSchedulePreview)

			// 使用 AddMessage 发送带数据的消息
			fixedScheduleMsg := session.Message{
				Role:    session.RoleAssistant,
				Content: fixedMsgText,
				Actions: []session.WorkflowAction{
					{
						ID:      "view_fixed_shifts_detail",
						Type:    session.ActionTypeQuery,
						Label:   "📊 查看固定排班详情",
						Payload: fixedSchedulePreview,
						Style:   session.ActionStyleSuccess,
					},
				},
				Metadata: map[string]any{
					"type":         "fixedSchedulePreview",
					"scheduleData": fixedSchedulePreview,
				},
			}

			if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, fixedScheduleMsg); err != nil {
				logger.Warn("Failed to send fixed schedule preview message", "error", err)
			}
		}
	}

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "开始执行计划",
			Event: session.WorkflowEvent(CreateV3EventPlanConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "修改计划",
			Event: session.WorkflowEvent(CreateV3EventPlanAdjust),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	return session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, workflowActions)
}

// actOnPlanConfirmed 处理任务计划确认
func actOnPlanConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Plan confirmed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	if createCtx.ProgressiveTaskPlan == nil || len(createCtx.ProgressiveTaskPlan.Tasks) == 0 {
		return fmt.Errorf("progressive task plan is empty")
	}

	// 禁用用户输入，进入执行阶段
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to disable user input", "error", err)
	}

	// 清空计划确认按钮
	if err := session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, nil); err != nil {
		logger.Warn("Failed to clear plan actions", "error", err)
	}

	return nil
}

// actAfterPlanConfirmed 计划确认后启动第一个任务（在状态已更新后执行）
func actAfterPlanConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	logger.Info("CreateV3: Starting first task after plan confirmed", "sessionID", wctx.Session().ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	return spawnCurrentTask(ctx, wctx, createCtx)
}

// actOnPlanAdjust 处理任务计划调整请求
func actOnPlanAdjust(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Plan adjustment requested", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	createCtx.AdjustmentScope = "plan"
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 启用用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	message := "📝 **请描述需要调整的计划**\n\n"
	message += "可以补充或修改临时需求，例如：\n"
	message += "- \"张三 1月12-14日出差，不能排班\"\n"
	message += "- \"1月20日夜班需增加1人\""

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send plan adjustment prompt", "error", err)
	}

	return nil
}

// actOnPlanAdjusted 处理任务计划调整完成
func actOnPlanAdjusted(ctx context.Context, wctx engine.Context, payload any) error {
	// 计划调整完成后的消息已在 actOnUserAdjustmentMessage 中发送
	return nil
}

// ============================================================
// 阶段 4: 渐进式任务执行
// ============================================================

// spawnCurrentTask 启动当前任务
func spawnCurrentTask(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	if createCtx.ProgressiveTaskPlan == nil || createCtx.CurrentTaskIndex >= len(createCtx.ProgressiveTaskPlan.Tasks) {
		// 所有任务完成
		return wctx.Send(ctx, CreateV3EventAllTasksComplete, nil)
	}

	// 【数据一致性验证】暂时禁用，后续自行考虑如何添加
	// if createCtx.CurrentTaskIndex == 0 {
	// 	validateDataConsistency(ctx, wctx, createCtx)
	// }

	task := createCtx.ProgressiveTaskPlan.Tasks[createCtx.CurrentTaskIndex]
	logger.Info("CreateV3: Starting task", "taskID", task.ID, "taskTitle", task.Title, "taskIndex", createCtx.CurrentTaskIndex+1, "totalTasks", len(createCtx.ProgressiveTaskPlan.Tasks))

	// 更新任务状态
	task.Status = "executing"
	task.ExecutedAt = time.Now().Format(time.RFC3339)

	// 发送任务开始消息（将人员ID、班次ID、规则ID转换为名称）
	// 获取rosteringService用于动态查询固定排班人员
	rosteringService, _ := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	description := replaceIDsWithNames(task.Description, createCtx.AllStaff, createCtx.AllStaff, createCtx.SelectedShifts, createCtx.Rules, rosteringService, sess.OrgID, ctx)
	message := fmt.Sprintf("⚙️ **执行任务 %d/%d**：%s\n\n%s",
		createCtx.CurrentTaskIndex+1,
		len(createCtx.ProgressiveTaskPlan.Tasks),
		task.Title,
		description)
	// 主任务消息使用 Assistant 消息类型，不是 System 消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send task start message", "error", err)
	}

	// 注意：这里不再直接执行任务，而是通过 Core 子工作流执行
	// 这个函数已经被替换为调用 Core 子工作流

	// 只提取动态排班数据，不传递固定排班
	// 任务执行器只应该处理动态排班，固定排班已经在fillFixedShiftSchedules中处理完成
	var currentDraftShiftSchedule *d_model.ShiftScheduleDraft
	if createCtx.WorkingDraft != nil && createCtx.WorkingDraft.Shifts != nil {
		currentDraftShiftSchedule = d_model.NewShiftScheduleDraft()

		// 确定要处理的班次列表
		targetShiftIDs := task.TargetShifts
		if len(targetShiftIDs) == 0 {
			// 如果任务未指定班次，提取所有班次（但只提取动态排班）
			for shiftID := range createCtx.WorkingDraft.Shifts {
				targetShiftIDs = append(targetShiftIDs, shiftID)
			}
		}

		// 只提取动态排班数据（IsFixed == false）
		for _, shiftID := range targetShiftIDs {
			shiftDraftData := createCtx.WorkingDraft.Shifts[shiftID]
			if shiftDraftData != nil && shiftDraftData.Days != nil {
				for date, dayShift := range shiftDraftData.Days {
					// 【关键】只提取动态排班，排除固定排班
					if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
						if currentDraftShiftSchedule.Schedule == nil {
							currentDraftShiftSchedule.Schedule = make(map[string][]string)
						}
						// 合并到currentDraftShiftSchedule（同一日期可能有多个班次的数据）
						existing := currentDraftShiftSchedule.Schedule[date]
						staffMap := make(map[string]bool)
						for _, id := range existing {
							staffMap[id] = true
						}
						for _, id := range dayShift.StaffIDs {
							if !staffMap[id] {
								currentDraftShiftSchedule.Schedule[date] = append(currentDraftShiftSchedule.Schedule[date], id)
								staffMap[id] = true
							}
						}
					}
				}
			}
		}
	} else {
		// 如果没有WorkingDraft，创建空的ShiftScheduleDraft
		currentDraftShiftSchedule = d_model.NewShiftScheduleDraft()
	}

	// 【P1优化】从缓存获取固定排班配置（传递给子工作流，用于校验器豁免和LLM QC识别）
	fixedAssignments := createCtx.FixedAssignments
	if len(fixedAssignments) == 0 {
		logger.Warn("FixedAssignments not cached, fetching again")
		var err error
		fixedAssignments, err = getFixedShiftAssignments(ctx, wctx, createCtx)
		if err != nil {
			logger.Warn("Failed to get fixed shift assignments for task context", "error", err)
			fixedAssignments = []d_model.CtxFixedShiftAssignment{}
		}
	}

	// 【V3改进】使用新的L2/L3结构
	// 注意：虽然我们构建了L2/L3结构，但为了向后兼容子工作流接口，
	// 我们仍然需要创建 CoreV3TaskContext 传递给子工作流
	// actExecuteTask 会从 CoreV3TaskContext 重新构建L2/L3结构并执行

	// 构建 CoreV3TaskContext（包含所有必要信息，供 actExecuteTask 重建L2/L3结构）
	taskCtx := &utils.CoreV3TaskContext{
		OrgID:             sess.OrgID,
		Task:              task,
		Shifts:            createCtx.SelectedShifts, // 包含所有班次，actExecuteTask 会过滤
		Rules:             createCtx.Rules,
		CandidateStaff:    createCtx.AllStaff, // 使用全员列表，在L3动态过滤（受 ShiftMembersMap 约束）
		AllStaff:          createCtx.AllStaff,
		ShiftMembersMap:   createCtx.ShiftMembersMap, // 各班次专属人员（L3候选人过滤依据）
		StaffRequirements: createCtx.StaffRequirements,
		OccupiedSlots:     createCtx.OccupiedSlots,
		CurrentDraft:      currentDraftShiftSchedule,
		WorkingDraft:      createCtx.WorkingDraft,
		FixedAssignments:  fixedAssignments,
		PersonalNeeds:     convertPersonalNeedsToModel(createCtx.PersonalNeeds),
	}

	// 【重要】预构建LLM调用缓存，避免每次调用时重复转换
	taskCtx.BuildLLMCache()

	// 保存任务上下文到 session
	if err := utils.SaveCoreV3TaskContext(ctx, wctx, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}

	// 调用 Core 子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor, cannot spawn sub-workflow")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowSchedulingCoreV3,
		Input:        nil, // Core 从 session 读取 CoreV3TaskContext
		OnComplete:   CreateV3EventTaskCompleted,
		OnError:      CreateV3EventTaskFailed,
		Timeout:      30 * 60 * 1e9, // 30 分钟超时 (纳秒)，任务可能涉及多个班次和多次LLM调用
		SnapshotKeys: []string{
			KeyCreateV3Context,
			utils.KeyCoreV3TaskContext,
		},
	}

	logger.Info("CreateV3: Spawning CoreV3 sub-workflow",
		"taskID", task.ID,
		"taskTitle", task.Title,
	)

	return actor.SpawnSubWorkflow(ctx, config)
}

// actOnTaskCompleted 处理任务完成事件
func actOnTaskCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Task completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	taskCtx, err := utils.GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		logger.Warn("Failed to get core task context", "error", err)
		return nil
	}

	// 【增强日志】记录任务上下文状态
	logger.Info("CreateV3: Processing task completion",
		"hasTaskCtx", taskCtx != nil,
		"hasTask", taskCtx != nil && taskCtx.Task != nil,
		"hasTaskResult", taskCtx != nil && taskCtx.TaskResult != nil)

	if taskCtx.Task != nil {
		logger.Info("CreateV3: Task details",
			"taskID", taskCtx.Task.ID,
			"taskTitle", taskCtx.Task.Title,
			"taskStatus", taskCtx.Task.Status)

		if createCtx.TaskResults == nil {
			createCtx.TaskResults = make(map[string]*d_model.TaskResult)
			logger.Info("CreateV3: Initialized TaskResults map")
		}

		// 防御性检查：即使 TaskResult 为 nil 也创建默认失败结果
		if taskCtx.TaskResult == nil {
			logger.Warn("TaskResult is nil, creating default failed result",
				"taskID", taskCtx.Task.ID,
				"taskTitle", taskCtx.Task.Title)
			taskCtx.TaskResult = &d_model.TaskResult{
				TaskID:  taskCtx.Task.ID,
				Success: false,
				Error:   "Task result was not created by executor",
			}
		}

		// 1. 保存任务结果
		createCtx.TaskResults[taskCtx.Task.ID] = taskCtx.TaskResult
		createCtx.CompletedTaskCount++

		logger.Info("TaskResult saved to context",
			"taskID", taskCtx.Task.ID,
			"taskResultID", taskCtx.TaskResult.TaskID,
			"success", taskCtx.TaskResult.Success,
			"shiftSchedulesCount", len(taskCtx.TaskResult.ShiftSchedules),
			"totalTaskResults", len(createCtx.TaskResults))

		// 2. 计算变更（使用当前 WorkingDraft 作为 before）
		if len(taskCtx.TaskResult.ShiftSchedules) > 0 {
			batch := utils.ComputeChangeBatch(
				taskCtx.Task.ID,
				taskCtx.Task.Title,
				createCtx.CurrentTaskIndex+1, // 任务序号从1开始
				createCtx.WorkingDraft,       // 关键：用当前状态作对比
				taskCtx.TaskResult.ShiftSchedules,
				createCtx.AllStaff,
				createCtx.AllStaff,
				createCtx.SelectedShifts,
			)

			// 3. 应用变更
			if err := utils.ApplyChangeBatch(
				createCtx.WorkingDraft,
				taskCtx.TaskResult.ShiftSchedules,
				createCtx.AllStaff,
				createCtx.AllStaff,
				createCtx.OccupySlot,
				createCtx.OccupiedSlots, // 直接传递强类型数组
			); err != nil {
				logger.Error("Failed to apply change batch", "error", err)
				return fmt.Errorf("failed to apply change batch: %w", err)
			}

			// 4. 保存变更批次
			createCtx.ChangeBatches = append(createCtx.ChangeBatches, batch)

			// 5. 日志
			stats := batch.GetStats()
			logger.Info("Changes applied",
				"taskID", taskCtx.Task.ID,
				"changes", len(batch.Changes),
				"add", stats.AddCount,
				"modify", stats.ModifyCount,
				"remove", stats.RemoveCount,
				"totalStaffSlots", stats.TotalStaffSlots)
		} else {
			logger.Info("No shift schedules in task result, skipping change computation")
		}

		// 6. 【V3改进】执行数据一致性校验
		warnings := validateDataConsistency(ctx, wctx, createCtx)
		if len(warnings) > 0 {
			logger.Warn("Data consistency warnings detected",
				"taskID", taskCtx.Task.ID,
				"warningCount", len(warnings),
				"warnings", warnings)
			// 可选：将警告记录到任务结果的Metadata中
			if taskCtx.TaskResult.Metadata == nil {
				taskCtx.TaskResult.Metadata = make(map[string]any)
			}
			taskCtx.TaskResult.Metadata["consistencyWarnings"] = warnings
		} else {
			logger.Info("Data consistency check passed", "taskID", taskCtx.Task.ID)
		}

		// 7. 【V3增强】执行时间约束校验（时间冲突和每日时长）
		timeErrors := utils.ValidateTimeConstraints(
			createCtx.WorkingDraft,
			createCtx.SelectedShifts,
			12.0,               // 每日最大工作时长（小时）
			createCtx.AllStaff, // 传入人员列表用于显示姓名
		)
		if len(timeErrors) > 0 {
			logger.Warn("Time constraint violations detected",
				"taskID", taskCtx.Task.ID,
				"errorCount", len(timeErrors),
				"errors", timeErrors)
			// 记录时间约束错误到任务结果的Metadata中
			if taskCtx.TaskResult.Metadata == nil {
				taskCtx.TaskResult.Metadata = make(map[string]any)
			}
			taskCtx.TaskResult.Metadata["timeConstraintErrors"] = timeErrors
		} else {
			logger.Info("Time constraint check passed", "taskID", taskCtx.Task.ID)
		}
	}

	// 【关键修复】先保存 context，再发送消息
	// 这样可以避免 AddSystemMessage 导致的 CAS 版本冲突覆盖 context 数据
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送时间约束警告消息（在 context 保存之后）
	if taskCtx != nil && taskCtx.TaskResult != nil && taskCtx.TaskResult.Metadata != nil {
		if timeErrors, ok := taskCtx.TaskResult.Metadata["timeConstraintErrors"].([]string); ok && len(timeErrors) > 0 {
			warningMsg := fmt.Sprintf("⚠️ 检测到时间约束违规（共%d项）：\n", len(timeErrors))
			for i, err := range timeErrors {
				if i < 5 { // 只显示前5个错误
					warningMsg += fmt.Sprintf("  %d. %s\n", i+1, err)
				}
			}
			if len(timeErrors) > 5 {
				warningMsg += fmt.Sprintf("  ... 还有 %d 项错误\n", len(timeErrors)-5)
			}
			if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, warningMsg); err != nil {
				logger.Warn("Failed to send time constraint warning message", "error", err)
			}
		}
	}

	return nil
}

// actAfterTaskCompleted 处理任务完成后的下一步（根据连续排班配置）
func actAfterTaskCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: AfterTaskCompleted called", "sessionID", sess.ID)

	// 检查当前任务是否有实际排班结果
	// 如果没有排班结果（本次无需排班），直接跳过审核继续下一个任务
	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		logger.Error("CreateV3: Failed to load context for task check", "error", err)
		// 加载失败时，继续原有逻辑
	} else {
		task := createCtx.GetCurrentTask()
		if task != nil {
			taskResult := createCtx.TaskResults[task.ID]
			hasScheduleChanges := false

			// 检查是否有变更批次
			if len(createCtx.ChangeBatches) > 0 {
				for _, batch := range createCtx.ChangeBatches {
					if batch.TaskID == task.ID && len(batch.Changes) > 0 {
						hasScheduleChanges = true
						break
					}
				}
			}

			// 检查是否有ShiftSchedules
			if !hasScheduleChanges && taskResult != nil && len(taskResult.ShiftSchedules) > 0 {
				for _, draft := range taskResult.ShiftSchedules {
					if draft != nil && len(draft.Schedule) > 0 {
						hasScheduleChanges = true
						break
					}
				}
			}

			// 如果没有排班变更，直接跳过审核继续下一个任务
			if !hasScheduleChanges {
				logger.Info("CreateV3: Task has no schedule changes, skipping review",
					"taskID", task.ID,
					"taskTitle", task.Title)
				return actSpawnNextTaskOrComplete(ctx, wctx, payload)
			}

			// 【V3改进】任务完成后立即校验人数需求
			rosteringService := engine.MustGetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
			schedulingAIService := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)

			// 获取规则校验器（可选，用于最终校验）
			var ruleValidator d_service.IRuleLevelValidator
			if validator, ok := engine.GetService[d_service.IRuleLevelValidator](wctx, "ruleValidator"); ok {
				ruleValidator = validator
			}

			// 获取配置器
			var configurator config.IRosteringConfigurator
			if cfg, ok := engine.GetService[config.IRosteringConfigurator](wctx, "configurator"); ok {
				configurator = cfg
			}

			// 创建任务执行器（用于校验）
			taskExecutor := executor.NewProgressiveTaskExecutor(
				logger,
				schedulingAIService,
				ruleValidator,
				rosteringService,
				nil, // aiFactory 不需要用于校验
				configurator,
			)

			// 执行任务级人数校验
			taskValidation := taskExecutor.ValidateTaskStaffCount(
				ctx,
				task,
				createCtx.WorkingDraft,
				createCtx.StaffRequirements,
				createCtx.SelectedShifts,
			)

			// 如果发现缺员，立即生成补充任务
			if !taskValidation.Passed && len(taskValidation.ShortageDetails) > 0 {
				logger.Info("Task validation found shortages, generating immediate supplement tasks",
					"taskID", task.ID,
					"shortageCount", len(taskValidation.ShortageDetails))

				// 生成补充任务
				supplementTasks := generateSupplementTasksForShortages(
					ctx,
					wctx,
					taskValidation.ShortageDetails,
					createCtx.SelectedShifts,
					createCtx.Rules,
					createCtx.AllStaff,
					taskExecutor,
				)

				if len(supplementTasks) > 0 {
					// 插入到任务计划中（在当前任务之后）
					insertSupplementTasks(createCtx, supplementTasks, createCtx.CurrentTaskIndex+1)

					// 发送通知
					sendShortageNotification(ctx, wctx, taskValidation.ShortageDetails, len(supplementTasks))

					logger.Info("Supplement tasks inserted",
						"taskID", task.ID,
						"supplementTaskCount", len(supplementTasks),
						"insertPosition", createCtx.CurrentTaskIndex+1)
				}
			}
		}
	}

	continuousEnabled := isContinuousSchedulingEnabled(ctx, wctx, sess.OrgID)

	if !continuousEnabled {
		if err := wctx.Send(ctx, CreateV3EventEnterTaskReview, nil); err != nil {
			logger.Error("CreateV3: Failed to send EnterTaskReview event", "error", err, "sessionID", sess.ID)
			return fmt.Errorf("failed to send EnterTaskReview event: %w", err)
		}
		return nil
	}

	return actSpawnNextTaskOrComplete(ctx, wctx, payload)
}

// actSpawnNextTaskOrComplete 启动下一个任务或完成所有任务（AfterAct）
func actSpawnNextTaskOrComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 移动到下一个任务
	createCtx.IncrementTaskProgress()

	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 检查是否所有任务完成
	if createCtx.IsAllTasksComplete() {
		logger.Info("CreateV3: All tasks completed")
		return wctx.Send(ctx, CreateV3EventAllTasksComplete, nil)
	}

	// 启动下一个任务
	return spawnCurrentTask(ctx, wctx, createCtx)
}

// actOnAllTasksComplete 处理所有任务完成
// 执行最终严格人数校验，如果存在缺员则自动生成 LLM 补充任务
func actOnAllTasksComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: All tasks completed, running final validation", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// ============================================================
	// 【最终校验】执行严格人数校验
	// ============================================================
	// 获取配置
	var maxFillRounds int = 2 // 默认最大补充2轮
	if configurator, ok := engine.GetService[interface{ GetConfig() interface{} }](wctx, "configurator"); ok {
		// 尝试从配置中读取 MaxFillRetries
		if cfg := configurator.GetConfig(); cfg != nil {
			// 配置结构可能需要类型断言
			logger.Debug("Configurator available for final validation")
		}
	}

	// 检查是否已达到最大补充轮次
	if createCtx.SupplementRound >= maxFillRounds {
		logger.Warn("CreateV3: Max supplement rounds reached, needs manual intervention",
			"supplementRound", createCtx.SupplementRound,
			"maxFillRounds", maxFillRounds)

		// 已达到最大补充轮次，发送需人工介入的消息
		message := "⚠️ **排班任务已完成，但存在人数缺口**\n\n"
		message += fmt.Sprintf("已执行 %d 轮自动补充，仍有部分班次人数不足。\n", createCtx.SupplementRound)
		message += fmt.Sprintf("完成任务数：%d\n", createCtx.CompletedTaskCount)
		message += fmt.Sprintf("失败任务数：%d\n", createCtx.FailedTaskCount)
		message += "\n请人工检查并调整排班，或确认保存当前结果。"

		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send manual intervention message", "error", err)
		}

		// 标记需要人工介入
		createCtx.NeedsManualIntervention = true
		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			logger.Warn("Failed to save context", "error", err)
		}

		// 进入确认保存阶段
		return wctx.Send(ctx, CreateV3EventSaveCompleted, nil)
	}

	// 执行最终校验
	rosteringService := engine.MustGetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)

	// 使用辅助函数转换 staffRequirements 格式
	allStaffRequirements := d_model.ConvertRequirementsToMap(createCtx.StaffRequirements)

	// 【关键修复】过滤 staffRequirements，只保留 SelectedShifts 中的班次
	// 原问题：createCtx.StaffRequirements 包含所有班次（如10个）的需求，
	// 但 SelectedShifts 可能只有用户选择的部分班次（如5个）。
	// 这会导致最终校验为未参与排班的班次报告缺员，生成无意义的补充任务。
	selectedShiftIDs := make(map[string]bool, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		selectedShiftIDs[shift.ID] = true
	}
	staffRequirements := make(map[string]map[string]int)
	for shiftID, dateReqs := range allStaffRequirements {
		if selectedShiftIDs[shiftID] {
			staffRequirements[shiftID] = dateReqs
		}
	}
	if len(allStaffRequirements) != len(staffRequirements) {
		logger.Info("Filtered staffRequirements to selected shifts only",
			"totalShifts", len(allStaffRequirements),
			"selectedShifts", len(staffRequirements),
			"skipped", len(allStaffRequirements)-len(staffRequirements))
	}

	// 创建校验器执行最终校验
	validator := executor.NewProgressiveTaskExecutor(
		logger,
		nil, // schedulingAIService - 校验时不需要
		nil, // ruleValidator - 使用内置校验
		rosteringService,
		nil, // 不需要 aiFactory 进行校验
		nil, // 不需要 configurator
	)

	finalResult := validator.ValidateFinalSchedule(
		ctx,
		createCtx.WorkingDraft,
		staffRequirements,
		createCtx.SelectedShifts,
		createCtx.Rules,
		createCtx.AllStaff,
		maxFillRounds,
	)

	if finalResult.Passed {
		// 校验通过，发送完成消息
		message := "🎉 **所有渐进式任务已完成**\n\n"
		message += "✅ 最终人数校验通过：所有班次人数严格满足需求\n\n"
		message += fmt.Sprintf("完成任务数：%d\n", createCtx.CompletedTaskCount)
		message += fmt.Sprintf("失败任务数：%d\n", createCtx.FailedTaskCount)
		if createCtx.SupplementRound > 0 {
			message += fmt.Sprintf("补充轮次：%d\n", createCtx.SupplementRound)
		}
		message += "\n请确认并保存排班结果。"

		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send completion message", "error", err)
		}

		// 进入确认保存阶段
		return wctx.Send(ctx, CreateV3EventSaveCompleted, nil)
	}

	// 存在缺员，需要执行补充任务
	logger.Info("CreateV3: Final validation found shortages, generating supplement tasks",
		"shortageCount", len(finalResult.ShortageDetails),
		"supplementRound", createCtx.SupplementRound+1)

	// 发送缺员通知
	message := "⚠️ **最终校验发现人数缺口**\n\n"
	message += fmt.Sprintf("%s\n\n", finalResult.Summary)
	message += "**缺员详情**：\n"
	for i, shortage := range finalResult.ShortageDetails {
		if i >= 5 {
			message += fmt.Sprintf("... 还有 %d 处缺员\n", len(finalResult.ShortageDetails)-5)
			break
		}
		message += fmt.Sprintf("- %s %s：需要 %d 人，当前 %d 人，缺少 %d 人\n",
			shortage.Date, shortage.ShiftName, shortage.RequiredCount, shortage.ActualCount, shortage.ShortageCount)
	}
	message += fmt.Sprintf("\n🔄 正在执行第 %d 轮自动补充...\n", createCtx.SupplementRound+1)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send shortage message", "error", err)
	}

	// 更新补充轮次
	createCtx.SupplementRound++

	// 将补充任务添加到任务计划中
	if len(finalResult.SupplementTasks) > 0 {
		// 重置任务索引，准备执行补充任务
		createCtx.CurrentTaskIndex = len(createCtx.ProgressiveTaskPlan.Tasks) // 指向新任务的起始位置
		createCtx.ProgressiveTaskPlan.Tasks = append(createCtx.ProgressiveTaskPlan.Tasks, finalResult.SupplementTasks...)

		logger.Info("CreateV3: Added supplement tasks to plan",
			"supplementTaskCount", len(finalResult.SupplementTasks),
			"totalTaskCount", len(createCtx.ProgressiveTaskPlan.Tasks))
	} else {
		// 无法生成补充任务，标记需要人工介入
		createCtx.NeedsManualIntervention = true

		message := "⚠️ **无法自动生成补充任务**\n\n"
		message += "请人工检查缺员情况并手动调整排班。\n"

		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send manual intervention message", "error", err)
		}

		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			logger.Warn("Failed to save context", "error", err)
		}

		// 进入确认保存阶段
		return wctx.Send(ctx, CreateV3EventSaveCompleted, nil)
	}

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 继续执行补充任务（回到任务执行循环）
	return spawnCurrentTask(ctx, wctx, createCtx)
}

// actOnTaskFailed 处理任务失败
func actOnTaskFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Task failed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取当前任务
	task := createCtx.GetCurrentTask()
	if task == nil {
		return fmt.Errorf("current task not found")
	}

	// 【关键修复】从子工作流上下文中提取任务结果
	// 这一步在 actOnTaskCompleted 中有，但之前在 actOnTaskFailed 中缺失
	taskCtx, err := utils.GetCoreV3TaskContext(ctx, wctx)
	if err != nil {
		logger.Warn("Failed to get core task context for failed task", "error", err, "taskID", task.ID)
	} else if taskCtx != nil && taskCtx.TaskResult != nil {
		// 初始化 TaskResults map（如果需要）
		if createCtx.TaskResults == nil {
			createCtx.TaskResults = make(map[string]*d_model.TaskResult)
		}
		// 保存任务结果到父上下文
		createCtx.TaskResults[task.ID] = taskCtx.TaskResult
		logger.Info("CreateV3: Saved failed task result to context",
			"taskID", task.ID,
			"taskTitle", task.Title,
			"error", taskCtx.TaskResult.Error,
			"hasShiftSchedules", len(taskCtx.TaskResult.ShiftSchedules) > 0)
	}

	// 更新失败计数
	createCtx.FailedTaskCount++

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 分析失败原因
	var failureReason string
	var failureDetails []string

	// 从任务结果中提取失败信息
	if taskResult, ok := createCtx.TaskResults[task.ID]; ok && taskResult != nil {
		if taskResult.Error != "" {
			failureReason = taskResult.Error
			failureDetails = append(failureDetails, fmt.Sprintf("执行错误：%s", taskResult.Error))
		}

		// 检查规则校验失败
		if taskResult.RuleValidationResult != nil && !taskResult.RuleValidationResult.Passed {
			failureDetails = append(failureDetails, fmt.Sprintf("规则校验失败：%s", taskResult.RuleValidationResult.Summary))

			// 统计各类问题
			issueCount := len(taskResult.RuleValidationResult.StaffCountIssues) +
				len(taskResult.RuleValidationResult.ShiftRuleIssues) +
				len(taskResult.RuleValidationResult.RuleComplianceIssues)

			if issueCount > 0 {
				failureDetails = append(failureDetails, fmt.Sprintf("发现问题：%d个（人数问题：%d，班次规则问题：%d，合规性问题：%d）",
					issueCount,
					len(taskResult.RuleValidationResult.StaffCountIssues),
					len(taskResult.RuleValidationResult.ShiftRuleIssues),
					len(taskResult.RuleValidationResult.RuleComplianceIssues)))
			}
		}

		// 检查LLMQC校验失败
		if taskResult.LLMQCResult != nil && !taskResult.LLMQCResult.Passed {
			failureDetails = append(failureDetails, fmt.Sprintf("LLMQC校验失败：%s", taskResult.LLMQCResult.Summary))

			if len(taskResult.LLMQCResult.Issues) > 0 {
				failureDetails = append(failureDetails, fmt.Sprintf("LLM发现问题：%d个", len(taskResult.LLMQCResult.Issues)))
			}
		}
	}

	// 如果没有明确的失败原因，使用通用描述
	if failureReason == "" && len(failureDetails) == 0 {
		failureReason = "任务执行过程中出现未知错误"
	}

	// 发送失败消息（包含失败原因）
	message := "❌ **任务执行失败**\n\n"
	message += fmt.Sprintf("任务：**%s**\n", task.Title)
	if task.Description != "" {
		// 将人员ID转换为姓名
		// 获取rosteringService用于动态查询固定排班人员
		rosteringService, _ := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
		description := replaceIDsWithNames(task.Description, createCtx.AllStaff, createCtx.AllStaff, createCtx.SelectedShifts, createCtx.Rules, rosteringService, sess.OrgID, ctx)
		message += fmt.Sprintf("任务描述：%s\n", description)
	}

	if failureReason != "" {
		message += fmt.Sprintf("\n**失败原因**：%s\n", failureReason)
	}

	if len(failureDetails) > 0 {
		message += "\n**失败详情**：\n"
		for _, detail := range failureDetails {
			message += fmt.Sprintf("- %s\n", detail)
		}
	}

	message += "\n请选择操作："

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send task failed message", "error", err)
	}

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "重试任务",
			Event: session.WorkflowEvent(CreateV3EventRetry),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "跳过任务",
			Event: session.WorkflowEvent(CreateV3EventSkipPhase),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 设置工作流 meta
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnEnterTaskFailedState 进入任务失败状态
func actOnEnterTaskFailedState(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Entering task failed state", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取当前任务
	task := createCtx.GetCurrentTask()
	if task == nil {
		return fmt.Errorf("current task not found")
	}

	// 【关键修复】确保从子工作流上下文中提取任务结果
	// 某些场景下 actOnTaskFailed 可能未被调用，需要在这里再次检查
	if _, exists := createCtx.TaskResults[task.ID]; !exists {
		taskCtx, err := utils.GetCoreV3TaskContext(ctx, wctx)
		if err != nil {
			logger.Warn("Failed to get core task context for failed task state", "error", err, "taskID", task.ID)
		} else if taskCtx != nil && taskCtx.TaskResult != nil {
			if createCtx.TaskResults == nil {
				createCtx.TaskResults = make(map[string]*d_model.TaskResult)
			}
			createCtx.TaskResults[task.ID] = taskCtx.TaskResult
			logger.Info("CreateV3: Saved failed task result to context (from state entry)",
				"taskID", task.ID,
				"taskTitle", task.Title,
				"error", taskCtx.TaskResult.Error)
			// 保存更新后的上下文
			if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
				logger.Warn("Failed to save context after extracting task result", "error", err)
			}
		}
	}

	// 分析失败原因
	var failureReason string
	var failureDetails []string

	// 从任务结果中提取失败信息
	if taskResult, ok := createCtx.TaskResults[task.ID]; ok && taskResult != nil {
		if taskResult.Error != "" {
			failureReason = taskResult.Error
			failureDetails = append(failureDetails, fmt.Sprintf("执行错误：%s", taskResult.Error))
		}

		// 检查规则校验失败
		if taskResult.RuleValidationResult != nil && !taskResult.RuleValidationResult.Passed {
			failureDetails = append(failureDetails, fmt.Sprintf("规则校验失败：%s", taskResult.RuleValidationResult.Summary))

			// 统计各类问题
			issueCount := len(taskResult.RuleValidationResult.StaffCountIssues) +
				len(taskResult.RuleValidationResult.ShiftRuleIssues) +
				len(taskResult.RuleValidationResult.RuleComplianceIssues)

			if issueCount > 0 {
				failureDetails = append(failureDetails, fmt.Sprintf("发现问题：%d个（人数问题：%d，班次规则问题：%d，合规性问题：%d）",
					issueCount,
					len(taskResult.RuleValidationResult.StaffCountIssues),
					len(taskResult.RuleValidationResult.ShiftRuleIssues),
					len(taskResult.RuleValidationResult.RuleComplianceIssues)))
			}
		}

		// 检查LLMQC校验失败
		if taskResult.LLMQCResult != nil && !taskResult.LLMQCResult.Passed {
			failureDetails = append(failureDetails, fmt.Sprintf("LLMQC校验失败：%s", taskResult.LLMQCResult.Summary))

			if len(taskResult.LLMQCResult.Issues) > 0 {
				failureDetails = append(failureDetails, fmt.Sprintf("LLM发现问题：%d个", len(taskResult.LLMQCResult.Issues)))
			}
		}
	}

	// 如果没有明确的失败原因，使用通用描述
	if failureReason == "" && len(failureDetails) == 0 {
		failureReason = "任务执行过程中出现未知错误"
	}

	// 构建失败信息消息
	message := "❌ **任务执行失败**\n\n"
	message += fmt.Sprintf("任务：**%s**\n", task.Title)
	if task.Description != "" {
		// 将人员ID转换为姓名
		// 获取rosteringService用于动态查询固定排班人员
		rosteringService, _ := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
		description := replaceIDsWithNames(task.Description, createCtx.AllStaff, createCtx.AllStaff, createCtx.SelectedShifts, createCtx.Rules, rosteringService, sess.OrgID, ctx)
		message += fmt.Sprintf("任务描述：%s\n", description)
	}

	// 显示失败原因（关键修复）
	if failureReason != "" {
		message += fmt.Sprintf("\n**失败原因**：%s\n", failureReason)
	}

	if len(failureDetails) > 0 {
		message += "\n**失败详情**：\n"
		for _, detail := range failureDetails {
			message += fmt.Sprintf("- %s\n", detail)
		}
	}

	message += "\n请选择如何处理：\n\n"
	message += "1. **重试任务**：重新执行当前任务\n"
	message += "2. **跳过任务**：跳过当前任务，继续下一个任务\n"
	message += "3. **取消排班**：取消整个排班流程"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send task failed state message", "error", err)
	}

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "重试任务",
			Event: session.WorkflowEvent(CreateV3EventRetry),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "跳过任务",
			Event: session.WorkflowEvent(CreateV3EventTaskFailedContinue),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventTaskFailedCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 设置工作流 meta
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// ============================================================
// 任务审核状态（非连续排班模式）
// ============================================================

// actEnterTaskReviewState 进入任务审核状态
func actEnterTaskReviewState(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Entering task review state", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		logger.Error("CreateV3: Failed to load context", "error", err, "sessionID", sess.ID)
		return fmt.Errorf("failed to load context: %w", err)
	}

	task := createCtx.GetCurrentTask()
	if task == nil {
		logger.Error("CreateV3: Current task not found", "sessionID", sess.ID, "currentTaskIndex", createCtx.CurrentTaskIndex)
		return fmt.Errorf("current task not found")
	}

	// 【增强防御】详细记录TaskResults状态
	logger.Info("CreateV3: Checking task result",
		"taskID", task.ID,
		"taskTitle", task.Title,
		"taskResultsCount", len(createCtx.TaskResults),
		"taskResultExists", createCtx.TaskResults[task.ID] != nil)

	// 记录所有TaskResults的键
	if len(createCtx.TaskResults) > 0 {
		taskResultKeys := make([]string, 0, len(createCtx.TaskResults))
		for key := range createCtx.TaskResults {
			taskResultKeys = append(taskResultKeys, key)
		}
		logger.Info("CreateV3: Available task results", "taskIDs", taskResultKeys)
	}

	taskResult := createCtx.TaskResults[task.ID]
	if taskResult == nil {
		logger.Error("CreateV3: Task result not found",
			"sessionID", sess.ID,
			"taskID", task.ID,
			"taskTitle", task.Title,
			"currentTaskIndex", createCtx.CurrentTaskIndex,
			"totalTasks", len(createCtx.ProgressiveTaskPlan.Tasks),
			"taskResultsCount", len(createCtx.TaskResults))

		// 【防御性处理】创建默认任务结果，允许流程继续
		logger.Warn("CreateV3: Creating default task result to allow workflow to continue",
			"taskID", task.ID)
		taskResult = &d_model.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "Task result was not properly saved during execution",
			Metadata: map[string]any{
				"fallback": true,
				"reason":   "TaskResult missing in context",
			},
		}
		// 保存到context中
		if createCtx.TaskResults == nil {
			createCtx.TaskResults = make(map[string]*d_model.TaskResult)
		}
		createCtx.TaskResults[task.ID] = taskResult
		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			logger.Error("Failed to save fallback task result", "error", err)
		}
	}

	// 获取当前任务的变更批次
	var currentBatch *d_model.ScheduleChangeBatch
	if len(createCtx.ChangeBatches) > 0 {
		// 找到对应任务的变更批次
		for _, batch := range createCtx.ChangeBatches {
			if batch.TaskID == task.ID {
				currentBatch = batch
				break
			}
		}
	}

	// 构建审核消息
	message := fmt.Sprintf("📋 **任务审核**：%s\n\n", task.Title)

	// 添加排班统计信息（使用变更批次）
	if currentBatch == nil || len(currentBatch.Changes) == 0 {
		message += "**排班情况**：本次无需排班\n\n"
	} else {
		stats := currentBatch.GetStats()
		message += "**排班情况**：\n"
		if stats.AddCount > 0 {
			message += fmt.Sprintf("  - 🆕 新增：%d 条\n", stats.AddCount)
		}
		if stats.ModifyCount > 0 {
			message += fmt.Sprintf("  - ✏️ 修改：%d 条\n", stats.ModifyCount)
		}
		if stats.RemoveCount > 0 {
			message += fmt.Sprintf("  - 🗑️ 删除：%d 条\n", stats.RemoveCount)
		}
		message += fmt.Sprintf("  - 📊 涉及：%d 个班次，%d 天\n", len(stats.AffectedShifts), len(stats.AffectedDates))
		message += fmt.Sprintf("  - 👥 总人次：%d\n\n", stats.TotalStaffSlots)
	}

	if taskResult.RuleValidationResult != nil {
		message += fmt.Sprintf("**规则级校验**：%s\n", taskResult.RuleValidationResult.Summary)
		if !taskResult.RuleValidationResult.Passed {
			message += "⚠️ 发现规则违反问题，请检查。\n"
		}
	}
	if taskResult.LLMQCResult != nil {
		message += fmt.Sprintf("**LLMQC校验**：%s\n", taskResult.LLMQCResult.Summary)
	}
	message += "\n请确认是否继续下一个任务，或提出调整意见。"

	// 发送排班预览消息（带查看按钮）
	scheduleMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: []session.WorkflowAction{},
		Metadata: map[string]any{
			"type":       "taskSchedule",
			"taskId":     task.ID,
			"taskTitle":  task.Title,
			"taskIndex":  createCtx.CurrentTaskIndex + 1,
			"totalTasks": len(createCtx.ProgressiveTaskPlan.Tasks),
		},
	}

	// 判断是否为最后一个任务
	isLastTask := createCtx.CurrentTaskIndex >= len(createCtx.ProgressiveTaskPlan.Tasks)-1

	// 添加查看排班详情按钮（渐进式排班每次都展示全班次预览）
	if createCtx.WorkingDraft != nil {
		// 使用上下文的全班次预览方法，展示当前整体排班状态
		fullSchedulePreview := createCtx.BuildFullSchedulePreview()
		scheduleMsg.Actions = append(scheduleMsg.Actions, session.WorkflowAction{
			ID:      "view_task_schedule_detail",
			Type:    session.ActionTypeQuery,
			Label:   "📊 查看排班详情",
			Payload: fullSchedulePreview,
			Style:   session.ActionStyleSuccess,
		})
		scheduleMsg.Metadata["scheduleDetail"] = fullSchedulePreview

		// 添加变更详情按钮（如果有变更）
		if currentBatch != nil && len(currentBatch.Changes) > 0 {
			changeDetailPreview := convertBatchToChangeDetailPreview(currentBatch, createCtx.SelectedShifts)
			scheduleMsg.Actions = append(scheduleMsg.Actions, session.WorkflowAction{
				ID:      "view_changes_detail",
				Type:    session.ActionTypeQuery,
				Label:   "📝 查看本次变更",
				Payload: serializePayload(changeDetailPreview),
				Style:   session.ActionStyleInfo,
			})
		}
	}

	// 仅在最后一个任务时添加预览完整排班按钮
	if isLastTask && createCtx.WorkingDraft != nil {
		schedulePreviewData := buildSchedulePreviewData(createCtx)
		scheduleMsg.Actions = append(scheduleMsg.Actions, session.WorkflowAction{
			ID:      "preview_full_schedule",
			Type:    session.ActionTypeQuery,
			Label:   "📅 预览完整排班",
			Payload: schedulePreviewData,
			Style:   session.ActionStylePrimary,
		})
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, scheduleMsg); err != nil {
		logger.Warn("CreateV3: Failed to send task review message", "error", err, "sessionID", sess.ID)
	}

	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "确认继续",
			Event: session.WorkflowEvent(CreateV3EventTaskReviewConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "提出调整",
			Event: session.WorkflowEvent(CreateV3EventTaskReviewAdjust),
			Style: session.ActionStyleSecondary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消排班",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	logger.Info("CreateV3: Setting workflow actions for task review", "sessionID", sess.ID)
	if err := session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, workflowActions); err != nil {
		logger.Error("CreateV3: Failed to set workflow actions", "error", err, "sessionID", sess.ID)
		return fmt.Errorf("failed to set workflow actions: %w", err)
	}

	logger.Info("CreateV3: Task review state entered successfully", "sessionID", sess.ID)
	return nil
}

// actOnTaskReviewConfirmed 处理任务审核确认
func actOnTaskReviewConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	// 继续下一个任务（AfterAct会处理）
	return nil
}

// actOnTaskReviewAdjust 处理任务审核调整
func actOnTaskReviewAdjust(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: User requested task adjustment", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	createCtx.AdjustmentScope = "task"
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 发送消息提示用户输入调整需求
	message := "📝 **请输入调整需求**\n\n"
	message += "请描述您希望如何调整当前任务的排班结果。\n"
	message += "例如：\n"
	message += "- \"将张三从12月1日移除\"\n"
	message += "- \"12月2日需要增加1人\"\n"
	message += "- \"李四不能安排在12月3日\""

	// 启用用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send adjustment message", "error", err)
	}

	return nil
}

// actOnUserAdjustmentMessage 处理用户调整消息
func actOnUserAdjustmentMessage(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Processing user adjustment message", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	_ = createCtx // 暂时未使用，后续实现调整逻辑

	// 解析用户消息
	var userMessage string
	if payload != nil {
		if payloadMap, ok := payload.(map[string]any); ok {
			if msg, ok := payloadMap["message"].(string); ok {
				userMessage = msg
			}
		}
	}

	if userMessage == "" {
		return fmt.Errorf("user adjustment message is empty")
	}

	logger.Info("User adjustment message received", "message", userMessage)

	if createCtx.AdjustmentScope == "plan" {
		// 计划调整：将用户输入合并到临时需求文本并重新生成计划
		if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
			logger.Warn("Failed to disable user input", "error", err)
		}

		requirementText := userMessage
		if createCtx.TemporaryNeedsText != "" {
			requirementText = strings.TrimSpace(createCtx.TemporaryNeedsText + "\n" + userMessage)
		}

		if err := applyTemporaryNeedsText(ctx, wctx, createCtx, requirementText); err != nil {
			return err
		}

		taskPlan, err := buildProgressiveTaskPlan(ctx, wctx, createCtx)
		if err != nil {
			return fmt.Errorf("failed to assess requirements: %w", err)
		}

		createCtx.ProgressiveTaskPlan = taskPlan
		createCtx.CurrentTaskIndex = 0
		createCtx.TaskResults = make(map[string]*d_model.TaskResult)
		createCtx.CompletedTaskCount = 0
		createCtx.FailedTaskCount = 0
		createCtx.SkippedTaskCount = 0
		createCtx.WorkingDraft = nil
		createCtx.AdjustmentScope = ""

		if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		message := buildPlanPreviewMessage(taskPlan)
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
			logger.Warn("Failed to send task plan preview message", "error", err)
		}

		workflowActions := []session.WorkflowAction{
			{
				Type:  session.ActionTypeWorkflow,
				Label: "开始执行计划",
				Event: session.WorkflowEvent(CreateV3EventPlanConfirmed),
				Style: session.ActionStylePrimary,
			},
			{
				Type:  session.ActionTypeWorkflow,
				Label: "修改计划",
				Event: session.WorkflowEvent(CreateV3EventPlanAdjust),
				Style: session.ActionStyleSecondary,
			},
			{
				Type:  session.ActionTypeWorkflow,
				Label: "取消排班",
				Event: session.WorkflowEvent(CreateV3EventUserCancel),
				Style: session.ActionStyleDanger,
			},
		}

		if err := session.SetWorkflowActions(ctx, wctx.SessionService(), sess.ID, workflowActions); err != nil {
			return err
		}

		return wctx.Send(ctx, CreateV3EventPlanAdjusted, nil)
	}

	// 获取当前任务的排班草案
	currentDraft := createCtx.WorkingDraft
	if currentDraft == nil {
		// 如果没有最终草案，创建一个空的
		createCtx.WorkingDraft = &d_model.ScheduleDraft{
			StartDate: createCtx.StartDate,
			EndDate:   createCtx.EndDate,
			Shifts:    make(map[string]*d_model.ShiftDraft),
		}
		currentDraft = createCtx.WorkingDraft
	}

	if currentDraft == nil {
		return fmt.Errorf("no schedule draft found for adjustment")
	}

	// 禁用用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to disable user input", "error", err)
	}

	// 发送处理中消息
	processingMsg := "🔄 **正在处理您的调整需求...**\n\n"
	processingMsg += "AI 正在分析您的需求并调整排班，请稍候..."
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, processingMsg); err != nil {
		logger.Warn("Failed to send processing message", "error", err)
	}

	// 调用 AI 服务进行排班调整
	// 注意：AdjustShiftSchedule 是针对单个班次的，我们需要对每个班次分别调整
	// 获取当前任务，确定需要调整的班次
	task := createCtx.GetCurrentTask()
	if task == nil {
		return fmt.Errorf("current task not found for adjustment")
	}

	// 获取AI服务
	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		return fmt.Errorf("AI service not available")
	}

	// 确定需要调整的班次（优先使用任务指定的班次，否则使用所有选定的班次）
	targetShiftIDs := task.TargetShifts
	if len(targetShiftIDs) == 0 {
		targetShiftIDs = make([]string, 0, len(createCtx.SelectedShifts))
		for _, shift := range createCtx.SelectedShifts {
			targetShiftIDs = append(targetShiftIDs, shift.ID)
		}
	}

	if len(targetShiftIDs) == 0 {
		return fmt.Errorf("no shifts found for adjustment")
	}

	// 对每个班次分别进行调整
	adjustedShiftsCount := 0
	var adjustmentErrors []string

	for _, shiftID := range targetShiftIDs {
		// 查找班次信息
		var shift *d_model.Shift
		for _, s := range createCtx.SelectedShifts {
			if s.ID == shiftID {
				shift = s
				break
			}
		}
		if shift == nil {
			logger.Warn("Shift not found", "shiftID", shiftID)
			continue
		}

		// 从ScheduleDraft中提取该班次的ShiftScheduleDraft
		shiftDraft := d_model.NewShiftScheduleDraft()
		if currentDraft.Shifts != nil {
			if shiftDraftData, ok := currentDraft.Shifts[shiftID]; ok && shiftDraftData != nil {
				// 转换ShiftDraft到ShiftScheduleDraft
				for date, dayShift := range shiftDraftData.Days {
					if dayShift != nil && len(dayShift.StaffIDs) > 0 {
						shiftDraft.Schedule[date] = dayShift.StaffIDs
					}
				}
			}
		}

		// 构建ShiftInfo
		shiftInfo := &d_model.ShiftInfo{
			ShiftID:   shift.ID,
			ShiftName: shift.Name,
			StartDate: createCtx.StartDate,
			EndDate:   createCtx.EndDate,
		}

		// 获取该班次的人员列表和规则
		staffListForAI := d_model.NewStaffInfoListFromEmployees(createCtx.AllStaff)
		rulesForAI := d_model.NewRuleInfoListFromRules(createCtx.Rules)

		// 获取该班次的人数需求
		staffRequirements := make(map[string]int)
		for _, req := range createCtx.StaffRequirements {
			if req.ShiftID == shiftID {
				staffRequirements[req.Date] = req.Count
			}
		}

		// 构建ExistingScheduleMarks（用于冲突检测）
		existingMarks := make(map[string]map[string]bool)
		for _, mark := range createCtx.ExistingScheduleMarks {
			if existingMarks[mark.Date] == nil {
				existingMarks[mark.Date] = make(map[string]bool)
			}
			existingMarks[mark.Date][mark.StaffID] = true
		}

		// 获取固定排班配置
		fixedShiftAssignments := make(map[string][]string)
		// 从任务结果中提取固定排班（如果有）
		// 这里简化处理，实际应该从createCtx中获取

		// 调用AI服务进行排班调整
		adjustResult, err := aiService.AdjustShiftSchedule(
			ctx,
			userMessage,
			shiftDraft,
			shiftInfo,
			staffListForAI,
			createCtx.AllStaff,
			rulesForAI,
			staffRequirements,
			existingMarks,
			fixedShiftAssignments,
		)
		if err != nil {
			logger.Warn("Failed to adjust shift schedule", "shiftID", shiftID, "error", err)
			adjustmentErrors = append(adjustmentErrors, fmt.Sprintf("班次 %s 调整失败：%v", shift.Name, err))
			continue
		}

		// 将调整结果合并回ScheduleDraft
		if adjustResult != nil && adjustResult.Draft != nil {
			// 确保ShiftDraft存在
			if currentDraft.Shifts == nil {
				currentDraft.Shifts = make(map[string]*d_model.ShiftDraft)
			}
			shiftDraftData := currentDraft.Shifts[shiftID]
			if shiftDraftData == nil {
				shiftDraftData = &d_model.ShiftDraft{
					ShiftID: shiftID,
					Days:    make(map[string]*d_model.DayShift),
				}
				currentDraft.Shifts[shiftID] = shiftDraftData
			}
			if shiftDraftData.Days == nil {
				shiftDraftData.Days = make(map[string]*d_model.DayShift)
			}

			// 更新排班数据
			for date, staffIDs := range adjustResult.Draft.Schedule {
				requiredCount := 0
				if count, ok := staffRequirements[date]; ok {
					requiredCount = count
				}
				shiftDraftData.Days[date] = &d_model.DayShift{
					StaffIDs:      staffIDs,
					ActualCount:   len(staffIDs),
					RequiredCount: requiredCount,
					IsFixed:       false, // 【P0修复】明确标记为动态排班（调整结果）
				}

				// 【P0修复】更新占位信息：将调整后的人员标记为已占用
				for _, staffID := range staffIDs {
					createCtx.OccupySlot(staffID, date, shiftID)
				}
			}

			adjustedShiftsCount++
			logger.Info("Shift schedule adjusted", "shiftID", shiftID, "shiftName", shift.Name, "datesCount", len(adjustResult.Draft.Schedule))
		}
	}

	// 更新WorkingDraft
	createCtx.WorkingDraft = currentDraft

	// 发送调整完成消息
	adjustedMsg := "✅ **排班调整完成**\n\n"
	if adjustedShiftsCount > 0 {
		adjustedMsg += fmt.Sprintf("已成功调整 %d 个班次的排班。\n\n", adjustedShiftsCount)
	}
	if len(adjustmentErrors) > 0 {
		adjustedMsg += "**调整过程中的问题**：\n"
		for _, errMsg := range adjustmentErrors {
			adjustedMsg += fmt.Sprintf("- %s\n", errMsg)
		}
		adjustedMsg += "\n"
	}
	adjustedMsg += "💡 **提示**：调整后的排班将在保存前再次确认。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, adjustedMsg); err != nil {
		logger.Warn("Failed to send adjusted message", "error", err)
	}

	logger.Info("User adjustment message processed", "message", userMessage, "adjustedShifts", adjustedShiftsCount)

	createCtx.AdjustmentScope = ""

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 触发调整完成事件
	return wctx.Send(ctx, CreateV3EventTaskAdjusted, nil)
}

// actOnTaskAdjusted 处理任务调整完成
func actOnTaskAdjusted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Task adjustment completed", "sessionID", sess.ID)

	// 发送调整完成消息
	message := "✅ **任务调整完成**\n\n"
	message += "排班已根据您的需求进行调整。请确认调整结果。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send adjustment complete message", "error", err)
	}

	// 返回到审核状态，让用户确认
	return nil
}

// actOnTaskFailedContinue 处理任务失败后继续
func actOnTaskFailedContinue(ctx context.Context, wctx engine.Context, payload any) error {
	// 继续下一个任务（AfterAct会处理）
	return nil
}

// actOnTaskFailedCancel 处理任务失败后取消
func actOnTaskFailedCancel(ctx context.Context, wctx engine.Context, payload any) error {
	return actUserCancel(ctx, wctx, payload)
}

// ============================================================
// 阶段 5: 确认保存
// ============================================================

// actOnSaveCompleted 处理保存完成
func actOnSaveCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Save completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 检查是否有最终排班草案
	if createCtx.WorkingDraft == nil {
		return fmt.Errorf("final schedule draft is nil")
	}

	// 获取 rostering service
	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rosteringService not found")
	}

	// 将 WorkingDraft 转换为 ScheduleBatch
	batch := convertScheduleDraftToBatch(createCtx.WorkingDraft, sess.OrgID, createCtx.SelectedShifts)

	// 保存排班
	result, err := rosteringService.BatchUpsertSchedules(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to save schedules: %w", err)
	}

	// 保存结果到上下文
	createCtx.SavedScheduleID = "batch_" + fmt.Sprintf("%d", time.Now().Unix())

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		logger.Warn("Failed to save context after schedule save", "error", err)
	}

	// 构建完整排班详情预览数据（用于"查看排班详情"按钮）
	scheduleDetailData := buildSchedulePreviewData(createCtx)

	// 发送成功消息
	message := "✅ **排班保存成功**\n\n"
	message += fmt.Sprintf("成功保存 **%d** 条排班记录。\n", result.Upserted)
	if result.Failed > 0 {
		message += fmt.Sprintf("⚠️ 有 **%d** 条记录保存失败。\n", result.Failed)
	}
	message += "\n您可以查看完整排班详情。"

	// 使用 AddMessage 发送带操作按钮的消息
	successMsg := session.Message{
		Role:    session.RoleAssistant,
		Content: message,
		Actions: []session.WorkflowAction{
			{
				ID:      "preview_full_schedule",
				Type:    session.ActionTypeQuery,
				Label:   "📊 查看完整排班",
				Payload: scheduleDetailData,
				Style:   session.ActionStyleSuccess,
			},
		},
		Metadata: map[string]any{
			"type":           "scheduleCompleted",
			"scheduleDetail": scheduleDetailData,
			"savedCount":     result.Upserted,
			"failedCount":    result.Failed,
		},
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, successMsg); err != nil {
		logger.Warn("Failed to send save success message", "error", err)
	}

	logger.Info("CreateV3: Schedule saved successfully", "total", result.Total, "upserted", result.Upserted, "failed", result.Failed)
	return nil
}

// actModifyBeforeSave 保存前修改
func actModifyBeforeSave(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Modify before save", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 检查是否有最终排班草案
	if createCtx.WorkingDraft == nil {
		return fmt.Errorf("final schedule draft is nil")
	}

	// 发送修改提示消息
	message := "✏️ **保存前修改**\n\n"
	message += "您可以在保存前对排班进行修改。\n\n"
	message += "请描述您想要进行的修改，例如：\n"
	message += "- 将某人的班次调整到其他日期\n"
	message += "- 增加或减少某天的人数\n"
	message += "- 替换某个人员\n\n"
	message += "💡 **提示**：输入您的修改需求，AI 将帮您调整排班。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send modify before save message", "error", err)
	}

	// 启用用户输入，等待用户输入修改需求
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	// 构建工作流操作按钮
	workflowActions := []session.WorkflowAction{
		{
			Type:  session.ActionTypeWorkflow,
			Label: "直接保存",
			Event: session.WorkflowEvent(CreateV3EventSaveCompleted),
			Style: session.ActionStylePrimary,
		},
		{
			Type:  session.ActionTypeWorkflow,
			Label: "取消修改",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleSecondary,
		},
	}

	// 设置工作流 meta
	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// ============================================================
// 通用处理函数
// ============================================================

// actUserCancel 用户取消操作
func actUserCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: User cancelled", "sessionID", sess.ID)

	message := "❌ 排班已取消"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	return nil
}

// actOnSubCancelled 子工作流被取消
func actOnSubCancelled(ctx context.Context, wctx engine.Context, payload any) error {
	return actUserCancel(ctx, wctx, payload)
}

// actOnSubFailed 子工作流失败
func actOnSubFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("CreateV3: Sub workflow failed", "sessionID", sess.ID)

	message := "❌ 子工作流执行失败"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send error message", "error", err)
	}

	return nil
}

// actHandleError 处理错误
func actHandleError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("CreateV3: Error occurred", "sessionID", sess.ID)

	message := "❌ 发生错误，排班流程已终止"
	if _, err := wctx.SessionService().AddSystemMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send error message", "error", err)
	}

	return nil
}

// ============================================================
// 上下文管理函数
// ============================================================

// loadCreateV3Context 加载V3上下文
func loadCreateV3Context(ctx context.Context, wctx engine.Context) (*CreateV3Context, error) {
	logger := wctx.Logger()
	sess := wctx.Session()
	data, found, err := wctx.SessionService().GetData(ctx, sess.ID, KeyCreateV3Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}
	if !found {
		return NewCreateV3Context(), nil
	}

	// 通过JSON序列化/反序列化转换
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context: %w", err)
	}

	// 【调试】检查原始 JSON 中 finalScheduleDraft 的内容
	var rawCheck map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &rawCheck); err == nil {
		if fsd, ok := rawCheck["finalScheduleDraft"].(map[string]interface{}); ok {
			if shifts, ok := fsd["shifts"].(map[string]interface{}); ok {
				totalStaff := 0
				for shiftID, shiftData := range shifts {
					if sd, ok := shiftData.(map[string]interface{}); ok {
						if days, ok := sd["days"].(map[string]interface{}); ok {
							for _, dayData := range days {
								if dd, ok := dayData.(map[string]interface{}); ok {
									if staffIDs, ok := dd["staffIds"].([]interface{}); ok {
										totalStaff += len(staffIDs)
									}
								}
							}
						}
					}
					_ = shiftID
				}
				logger.Info("loadCreateV3Context: Raw JSON check",
					"shiftsCount", len(shifts),
					"totalStaffIDs", totalStaff)
			} else {
				logger.Warn("loadCreateV3Context: finalScheduleDraft.shifts is nil or not a map")
			}
		} else {
			logger.Warn("loadCreateV3Context: finalScheduleDraft is nil or not a map")
		}
	}

	var createCtx CreateV3Context
	if err := json.Unmarshal(jsonBytes, &createCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	// 【调试】检查反序列化后的 WorkingDraft
	if createCtx.WorkingDraft != nil && createCtx.WorkingDraft.Shifts != nil {
		totalStaff := 0
		for _, shiftDraft := range createCtx.WorkingDraft.Shifts {
			if shiftDraft.Days != nil {
				for _, dayShift := range shiftDraft.Days {
					totalStaff += len(dayShift.StaffIDs)
				}
			}
		}
		logger.Info("loadCreateV3Context: After unmarshal",
			"shiftsCount", len(createCtx.WorkingDraft.Shifts),
			"totalStaffIDs", totalStaff)
	} else {
		logger.Warn("loadCreateV3Context: WorkingDraft or Shifts is nil after unmarshal")
	}

	return &createCtx, nil
}

// saveCreateV3Context 保存V3上下文
func saveCreateV3Context(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 【调试】保存前检查 WorkingDraft 状态
	if createCtx.WorkingDraft != nil && createCtx.WorkingDraft.Shifts != nil {
		totalStaff := 0
		for _, shiftDraft := range createCtx.WorkingDraft.Shifts {
			if shiftDraft.Days != nil {
				for _, dayShift := range shiftDraft.Days {
					totalStaff += len(dayShift.StaffIDs)
				}
			}
		}
		logger.Info("saveCreateV3Context: Before save",
			"shiftsCount", len(createCtx.WorkingDraft.Shifts),
			"totalStaffIDs", totalStaff)
	} else {
		logger.Warn("saveCreateV3Context: WorkingDraft or Shifts is nil before save")
	}

	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCreateV3Context, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}
	return nil
}

// populateInfoFromSubWorkflow 从子工作流填充信息
func populateInfoFromSubWorkflow(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Populating info from sub-workflow or direct loading", "sessionID", sess.ID)

	// 使用 common.LoadScheduleBasicContext 加载基础信息
	// 这个函数会加载所有需要的数据：班次、人员、规则、人数需求、请假信息等
	basicCtx, err := common.LoadScheduleBasicContext(
		ctx,
		wctx,
		sess.OrgID,
		createCtx.StartDate,
		createCtx.EndDate,
		nil, // shiftIDs: nil 表示加载所有激活的班次，如果已有选定的班次可以从 createCtx 中获取
	)
	if err != nil {
		return fmt.Errorf("failed to load schedule basic context: %w", err)
	}

	// 填充到 createCtx
	createCtx.SelectedShifts = basicCtx.SelectedShifts
	createCtx.AllStaff = basicCtx.AllStaffList           // 所有员工（用于姓名映射、信息检索）
	createCtx.ShiftMembersMap = basicCtx.ShiftMembersMap // 各班次专属人员（用于L3候选人过滤）
	// 转换 StaffRequirements 从 map 到数组
	if basicCtx.StaffRequirements != nil {
		createCtx.StaffRequirements = make([]d_model.ShiftDateRequirement, 0)
		for shiftID, dateMap := range basicCtx.StaffRequirements {
			for date, count := range dateMap {
				// 查找班次名称
				shiftName := ""
				for _, shift := range basicCtx.SelectedShifts {
					if shift.ID == shiftID {
						shiftName = shift.Name
						break
					}
				}
				createCtx.StaffRequirements = append(createCtx.StaffRequirements, d_model.ShiftDateRequirement{
					ShiftID:   shiftID,
					ShiftName: shiftName,
					Date:      date,
					Count:     count,
				})
			}
		}
	}
	createCtx.Rules = basicCtx.Rules
	createCtx.StaffLeaves = basicCtx.StaffLeaves

	// 如果没有设置时间范围，使用加载的时间范围
	if createCtx.StartDate == "" {
		createCtx.StartDate = basicCtx.StartDate
	}
	if createCtx.EndDate == "" {
		createCtx.EndDate = basicCtx.EndDate
	}

	logger.Info("CreateV3: Info populated successfully",
		"shifts", len(createCtx.SelectedShifts),
		"staff", len(createCtx.AllStaff),
		"allStaff", len(createCtx.AllStaff),
		"rules", len(createCtx.Rules),
		"requirements", len(createCtx.StaffRequirements))

	return nil
}

// getFixedShiftAssignments 获取固定排班配置
// 返回结构体 slice，避免嵌套 map，更清晰易懂
func getFixedShiftAssignments(ctx context.Context, wctx engine.Context, createCtx *CreateV3Context) ([]d_model.CtxFixedShiftAssignment, error) {
	logger := wctx.Logger()

	// 获取 rosteringService
	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return nil, fmt.Errorf("rostering service not available")
	}

	// 提取所有班次ID
	shiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		shiftIDs = append(shiftIDs, shift.ID)
	}

	if len(shiftIDs) == 0 {
		return []d_model.CtxFixedShiftAssignment{}, nil
	}

	logger.Info("Getting fixed shift assignments", "shiftCount", len(shiftIDs))

	// 调用 IRosteringService 的 CalculateMultipleFixedSchedules 方法
	// 返回格式: map[shiftID]map[date][]staffID
	allFixedSchedules, err := rosteringService.CalculateMultipleFixedSchedules(
		ctx,
		shiftIDs,
		createCtx.StartDate,
		createCtx.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fixed schedules: %w", err)
	}

	// 转换为结构体 slice，避免嵌套 map
	result := make([]d_model.CtxFixedShiftAssignment, 0)
	for shiftID, shiftSchedule := range allFixedSchedules {
		for date, staffIDs := range shiftSchedule {
			result = append(result, d_model.CtxFixedShiftAssignment{
				ShiftID:  shiftID,
				Date:     date,
				StaffIDs: staffIDs,
			})
		}
	}

	logger.Info("Fixed shift assignments loaded", "totalEntries", len(result))
	return result, nil
}

// parsePayloadToMap 解析payload为map
func parsePayloadToMap(payload any, result *map[string]any) error {
	if payload == nil {
		*result = nil
		return nil
	}
	if m, ok := payload.(map[string]any); ok {
		*result = m
		return nil
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, result)
}

// buildStaffCountFieldsV3 构建人员数量配置表单字段（V3版本）
func buildStaffCountFieldsV3(createCtx *CreateV3Context) []session.WorkflowActionField {
	fields := make([]session.WorkflowActionField, 0)

	for _, shift := range createCtx.SelectedShifts {
		// 构建默认值（map[date]count 格式）
		defaultValue := make(map[string]int)
		for _, req := range createCtx.StaffRequirements {
			if req.ShiftID == shift.ID {
				defaultValue[req.Date] = req.Count
			}
		}

		// 使用 daily-grid 类型
		fields = append(fields, session.WorkflowActionField{
			Name:         fmt.Sprintf("shift_%s_count", shift.ID),
			Label:        shift.Name,
			Type:         session.FieldTypeDailyGrid,
			Required:     false,
			DefaultValue: defaultValue,
			Options: []session.FieldOption{
				{
					Label: shift.Name,
					Value: shift.ID,
					Extra: map[string]any{
						"shiftId":    shift.ID,
						"shiftName":  shift.Name,
						"startTime":  shift.StartTime,
						"endTime":    shift.EndTime,
						"shiftColor": shift.Color,
						"startDate":  createCtx.StartDate,
						"endDate":    createCtx.EndDate,
					},
				},
			},
		})
	}

	return fields
}

// parseStaffCountPayloadV3 解析人数配置 payload（V3版本）
func parseStaffCountPayloadV3(payloadMap map[string]any, createCtx *CreateV3Context) error {
	// 检查是否有修改
	hasModification := false
	for k := range payloadMap {
		if len(k) > 6 && k[:6] == "shift_" {
			hasModification = true
			break
		}
	}

	if !hasModification {
		return nil
	}

	// 确保 StaffRequirements 已初始化
	if createCtx.StaffRequirements == nil {
		createCtx.StaffRequirements = make([]d_model.ShiftDateRequirement, 0)
	}

	// 解析日期范围
	startDate, startErr := time.Parse("2006-01-02", createCtx.StartDate)
	endDate, endErr := time.Parse("2006-01-02", createCtx.EndDate)

	for _, shift := range createCtx.SelectedShifts {
		fieldName := fmt.Sprintf("shift_%s_count", shift.ID)

		if countVal, exists := payloadMap[fieldName]; exists {
			// 格式1：JSON 对象 {"2024-01-01": 2, "2024-01-02": 3, ...}
			if dailyMap, ok := countVal.(map[string]any); ok {
				// 移除旧的该班次的需求记录
				newReqs := make([]d_model.ShiftDateRequirement, 0)
				for _, req := range createCtx.StaffRequirements {
					if req.ShiftID != shift.ID {
						newReqs = append(newReqs, req)
					}
				}
				// 添加新的需求
				for dateStr, count := range dailyMap {
					var countInt int
					switch v := count.(type) {
					case float64:
						countInt = int(v)
					case int:
						countInt = v
					case int64:
						countInt = int(v)
					default:
						countInt = 1
					}
					newReqs = append(newReqs, d_model.ShiftDateRequirement{
						ShiftID:   shift.ID,
						ShiftName: shift.Name,
						Date:      dateStr,
						Count:     countInt,
					})
				}
				createCtx.StaffRequirements = newReqs
				continue
			}

			// 格式2：单个数字，应用到所有日期
			var staffCount int
			switch v := countVal.(type) {
			case float64:
				staffCount = int(v)
			case int:
				staffCount = v
			case int64:
				staffCount = int(v)
			default:
				continue
			}

			if startErr == nil && endErr == nil && staffCount > 0 {
				// 移除旧的该班次的需求记录
				newReqs := make([]d_model.ShiftDateRequirement, 0)
				for _, req := range createCtx.StaffRequirements {
					if req.ShiftID != shift.ID {
						newReqs = append(newReqs, req)
					}
				}
				// 添加新的需求
				for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
					newReqs = append(newReqs, d_model.ShiftDateRequirement{
						ShiftID:   shift.ID,
						ShiftName: shift.Name,
						Date:      d.Format("2006-01-02"),
						Count:     staffCount,
					})
				}
				createCtx.StaffRequirements = newReqs
			}
		}
	}

	return nil
}

// convertScheduleDraftToBatch 将 ScheduleDraft 转换为 ScheduleBatch
func convertScheduleDraftToBatch(draft *d_model.ScheduleDraft, orgID string, selectedShifts []*d_model.Shift) d_model.ScheduleBatch {
	items := make([]d_model.ScheduleUpsertRequest, 0)

	// 构建 shiftID -> shift 映射
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range selectedShifts {
		shiftMap[shift.ID] = shift
	}

	// 遍历所有班次
	for shiftID, shiftDraft := range draft.Shifts {
		// 查找对应的Shift对象，获取StartTime和EndTime
		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}

		// 遍历该班次的所有日期
		for date, dayShift := range shiftDraft.Days {
			// 为每个人员创建一条排班记录
			for _, staffID := range dayShift.StaffIDs {
				items = append(items, d_model.ScheduleUpsertRequest{
					OrgID:     orgID,
					UserID:    staffID,
					WorkDate:  date,
					ShiftCode: shiftID, // 实际传递 ShiftID，字段名虽然叫 ShiftCode
					StartTime: shift.StartTime,
					EndTime:   shift.EndTime,
					Status:    "active", // 默认状态为激活
				})
			}
		}
	}

	return d_model.ScheduleBatch{
		Items:          items,
		IdempotencyKey: "v3_" + fmt.Sprintf("%d", time.Now().Unix()),
		OnConflict:     "upsert", // 冲突时更新
	}
}

// replaceIDsWithNames 将任务描述中的ID（人员、班次、规则）替换为名称
// 查找模式：ID列表：后跟UUID列表，或直接包含UUID列表
// 如果人员不在AllStaffList中，会通过rosteringService动态查询（用于固定排班人员等）
func replaceIDsWithNames(
	description string,
	allStaffList []*d_model.Employee,
	staffList []*d_model.Employee,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	rosteringService d_service.IRosteringService,
	orgID string,
	ctx context.Context,
) string {
	if description == "" {
		return description
	}

	// 构建ID到名称的映射（包含人员、班次、规则）
	idToName := make(map[string]string)

	// 1. 添加班次ID到名称的映射（优先级最高，避免班次UUID泄露）
	for _, shift := range shifts {
		if shift != nil && shift.ID != "" && shift.Name != "" {
			idToName[shift.ID] = shift.Name
		}
	}

	// 2. 添加规则ID到名称的映射
	for _, rule := range rules {
		if rule != nil && rule.ID != "" && rule.Name != "" {
			idToName[rule.ID] = rule.Name
		}
	}

	// 3. 添加人员ID到姓名的映射
	// 优先使用AllStaffList（更完整）
	for _, staff := range allStaffList {
		if staff != nil && staff.ID != "" && staff.Name != "" {
			idToName[staff.ID] = staff.Name
		}
	}

	// 如果AllStaffList中没有，使用StaffList
	for _, staff := range staffList {
		if staff != nil && staff.ID != "" && staff.Name != "" {
			if _, exists := idToName[staff.ID]; !exists {
				idToName[staff.ID] = staff.Name
			}
		}
	}

	// UUID格式：8-4-4-4-12 十六进制字符，用连字符分隔
	// 模式：匹配UUID格式的字符串（不区分大小写）
	uuidPattern := `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

	// 使用正则表达式查找所有UUID
	re := regexp.MustCompile(uuidPattern)

	// 提取描述中所有UUID，对于无法匹配的UUID，通过rosteringService查询
	allUUIDs := re.FindAllString(description, -1)
	if rosteringService != nil && orgID != "" && ctx != nil {
		for _, uuid := range allUUIDs {
			if _, exists := idToName[uuid]; !exists {
				// 尝试查询该员工（用于固定排班人员等不在班次分组中的人员）
				staff, err := rosteringService.GetStaff(ctx, uuid)
				if err == nil && staff != nil && staff.Name != "" {
					idToName[uuid] = staff.Name
				}
			}
		}
	}

	// 处理括号内的人员列表（无论是否有"ID列表："标签）
	// 支持格式：`（姓名1, uuid1, uuid2, 姓名2）` 或 `(name1, uuid1, uuid2, name2)`
	// 这是为了处理LLM生成的混合格式（姓名和UUID混合）
	bracketPattern := regexp.MustCompile(`[（(]([^）)]+)[）)]`)
	result := bracketPattern.ReplaceAllStringFunc(description, func(match string) string {
		// 确定括号类型
		var leftParen, rightParen string
		if strings.HasPrefix(match, "（") {
			leftParen = "（"
			rightParen = "）"
		} else if strings.HasPrefix(match, "(") {
			leftParen = "("
			rightParen = ")"
		} else {
			return match
		}

		// 提取括号内的内容
		content := match[len(leftParen) : len(match)-len(rightParen)]
		content = strings.TrimSpace(content)

		// 检查是否包含"ID列表："标签
		hasIDLabel := strings.Contains(content, "ID列表") || strings.Contains(content, "id列表") ||
			strings.Contains(content, "人员ID") || strings.Contains(content, "人员id")

		// 如果包含"ID列表："标签，跳过（由后续逻辑处理）
		if hasIDLabel {
			return match
		}

		// 检查是否包含UUID（如果包含，说明可能是人员列表）
		hasUUID := re.MatchString(content)
		if !hasUUID {
			return match // 不包含UUID，不是人员列表，保持原样
		}

		// 检查是否包含逗号分隔的列表格式
		if !strings.Contains(content, ",") && !strings.Contains(content, "，") {
			// 单个UUID，直接替换
			if name, ok := idToName[content]; ok {
				return leftParen + name + rightParen
			}
			return match
		}

		// 处理逗号分隔的列表
		// 使用 strings.FieldsFunc 分割，支持中英文逗号
		items := strings.FieldsFunc(content, func(r rune) bool {
			return r == ',' || r == '，'
		})

		// 替换每个UUID为姓名，保留已有姓名
		replacedItems := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue // 跳过空项
			}

			// 检查是否是UUID
			if re.MatchString(item) {
				// 是UUID，尝试替换为姓名
				if name, ok := idToName[item]; ok {
					replacedItems = append(replacedItems, name)
				} else {
					replacedItems = append(replacedItems, item) // 找不到姓名时保持ID
				}
			} else {
				// 不是UUID，可能是姓名，直接保留
				replacedItems = append(replacedItems, item)
			}
		}

		// 重新组合，使用中文顿号作为分隔符
		replacedContent := strings.Join(replacedItems, "、")
		return leftParen + replacedContent + rightParen
	})

	// 处理"ID列表："后的ID列表格式
	// 支持多种格式：
	// 1. "ID列表：uuid1, uuid2, uuid3"
	// 2. "（ID列表：uuid1, uuid2）"
	// 3. "(ID列表: uuid1, uuid2)"
	// 4. 跨行的ID列表（使用多行模式）

	// 首先尝试匹配带括号的格式
	bracketedPattern := regexp.MustCompile(`[（(]\s*(ID列表|id列表|人员ID|人员id)\s*[：:]\s*([^）)]+)[）)]`)
	result = bracketedPattern.ReplaceAllStringFunc(result, func(match string) string {
		// 提取前缀和ID列表部分
		var prefix, idListStr string
		var rightParen string

		// 确定括号类型
		if strings.HasPrefix(match, "（") {
			rightParen = "）"
		} else if strings.HasPrefix(match, "(") {
			rightParen = ")"
		} else {
			return match // 如果没有匹配到括号，返回原字符串
		}

		// 查找冒号位置
		colonIdx := -1
		if idx := strings.Index(match, "："); idx >= 0 {
			colonIdx = idx
		} else if idx := strings.Index(match, ":"); idx >= 0 {
			colonIdx = idx
		}

		if colonIdx < 0 {
			return match
		}

		// 提取前缀（包括左括号和标签）
		prefix = match[:colonIdx+1]

		// 提取ID列表部分（去除右括号）
		idListStr = strings.TrimSpace(match[colonIdx+1 : len(match)-len(rightParen)])

		// 提取所有UUID（支持逗号、空格、换行等分隔符）
		// 先替换所有分隔符为空格，然后提取
		idListStr = regexp.MustCompile(`[,，\s\n]+`).ReplaceAllString(idListStr, " ")
		ids := re.FindAllString(idListStr, -1)
		if len(ids) == 0 {
			return match
		}

		// 转换为姓名列表
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := idToName[id]; ok {
				names = append(names, name)
			} else {
				names = append(names, id) // 找不到姓名时保持ID
			}
		}

		// 替换为姓名列表（prefix已经包含了左括号和标签）
		return prefix + strings.Join(names, "、") + rightParen
	})

	// 然后处理不带括号的格式
	simplePattern := regexp.MustCompile(`(ID列表|id列表|人员ID|人员id)\s*[：:]\s*([^\n]+)`)
	result = simplePattern.ReplaceAllStringFunc(result, func(match string) string {
		// 提取前缀和ID列表部分
		var prefix, idListStr string

		// 查找冒号位置
		colonIdx := -1
		if idx := strings.Index(match, "："); idx >= 0 {
			colonIdx = idx
			prefix = match[:idx+len("：")]
			idListStr = strings.TrimSpace(match[idx+len("："):])
		} else if idx := strings.Index(match, ":"); idx >= 0 {
			colonIdx = idx
			prefix = match[:idx+1]
			idListStr = strings.TrimSpace(match[idx+1:])
		}

		if colonIdx < 0 {
			return match
		}

		// 提取所有UUID（支持逗号、空格等分隔符）
		idListStr = regexp.MustCompile(`[,，\s]+`).ReplaceAllString(idListStr, " ")
		ids := re.FindAllString(idListStr, -1)
		if len(ids) == 0 {
			return match
		}

		// 转换为姓名列表
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := idToName[id]; ok {
				names = append(names, name)
			} else {
				names = append(names, id) // 找不到姓名时保持ID
			}
		}

		// 替换为姓名列表
		return prefix + strings.Join(names, "、")
	})

	// 如果上面没有匹配到"ID列表："格式，直接替换所有UUID为姓名
	if result == description {
		result = re.ReplaceAllStringFunc(description, func(match string) string {
			if name, ok := idToName[match]; ok {
				return name
			}
			// 如果找不到姓名，保持原ID
			return match
		})
	}

	// 【新增】处理shortID格式（rule_1, shift_1, staff_1等），替换为友好名称
	// 构建shortID到友好名称的映射表
	shortIDToName := make(map[string]string)

	// 1. 构建班次shortID到名称的映射
	_, shiftReverseMappings := utils.BuildShiftIDMappings(shifts)
	for shortID, realID := range shiftReverseMappings {
		if name, ok := idToName[realID]; ok {
			shortIDToName[shortID] = name
		}
	}

	// 2. 构建规则shortID到名称的映射
	_, ruleReverseMappings := utils.BuildRuleIDMappings(rules)
	for shortID, realID := range ruleReverseMappings {
		if name, ok := idToName[realID]; ok {
			shortIDToName[shortID] = name
		}
	}

	// 3. 构建人员shortID到名称的映射
	_, staffReverseMappings := utils.BuildStaffIDMappings(allStaffList)
	// 如果allStaffList为空，使用staffList
	if len(allStaffList) == 0 {
		_, staffReverseMappings = utils.BuildStaffIDMappings(staffList)
	}
	for shortID, realID := range staffReverseMappings {
		if name, ok := idToName[realID]; ok {
			shortIDToName[shortID] = name
		}
	}

	// 4. 使用正则表达式匹配shortID格式（shift_1, rule_1, staff_1等）
	shortIDPattern := regexp.MustCompile(`(shift|rule|staff)_\d+`)
	result = shortIDPattern.ReplaceAllStringFunc(result, func(match string) string {
		if name, ok := shortIDToName[match]; ok {
			return name
		}
		// 如果找不到友好名称，保持原shortID
		return match
	})

	return result
}

// ============================================================
// 全局评审相关 Actions
// ============================================================

// actStartGlobalReview 启动全局评审流程
func actStartGlobalReview(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Starting global review", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送开始评审消息
	message := "📋 **开始全局规则评审**\n\n"
	message += "正在逐条检查规则和个人需求是否在排班表中被正确满足...\n"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send global review start message", "error", err)
	}

	// 禁止用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, false); err != nil {
		logger.Warn("Failed to disable user input", "error", err)
	}

	// 保存上下文
	return saveCreateV3Context(ctx, wctx, createCtx)
}

// actAfterGlobalReview 全局评审完成后的处理
func actAfterGlobalReview(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Executing global review", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 获取配置
	configurator := engine.MustGetService[config.IRosteringConfigurator](wctx, "configurator")
	v3Config := configurator.GetScheduleV3Config()

	maxDebateRounds := v3Config.GlobalReview.MaxDebateRounds
	if maxDebateRounds <= 0 {
		maxDebateRounds = 3 // 默认最多3轮对论
	}

	// 检查是否启用全局评审
	if !v3Config.GlobalReview.Enabled {
		logger.Info("CreateV3: Global review is disabled, skipping")
		return wctx.Send(ctx, CreateV3EventGlobalReviewSkip, nil)
	}

	// 收集个人需求（转换为 d_model.PersonalNeed 格式）
	personalNeeds := make(map[string][]*d_model.PersonalNeed)
	for staffID, needs := range createCtx.PersonalNeeds {
		for _, need := range needs {
			personalNeeds[staffID] = append(personalNeeds[staffID], &d_model.PersonalNeed{
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
	}

	// 构建任务上下文
	taskContext := &utils.CoreV3TaskContext{
		OrgID:           createCtx.OrgID,
		AllStaff:        createCtx.AllStaff,
		CandidateStaff:  createCtx.StaffList,
		ShiftMembersMap: createCtx.ShiftMembersMap, // 各班次专属人员（用于候选人过滤）
		Shifts:          createCtx.SelectedShifts,
		Rules:           createCtx.Rules,
		PersonalNeeds:   personalNeeds,
		WorkingDraft:    createCtx.WorkingDraft,
	}
	taskContext.BuildLLMCache()

	// 获取 Bridge 用于实时进度广播
	var bridge wsbridge.IBridge
	if b, ok := engine.GetService[wsbridge.IBridge](wctx, engine.ServiceKeyBridge); ok {
		bridge = b
	}

	// 创建全局评审执行器
	executor, err := utils.NewGlobalReviewExecutor(
		logger,
		configurator,
		taskContext,
		utils.WithMaxDebateRounds(maxDebateRounds),
		utils.WithReviewProgressCallback(func(progress *d_model.GlobalReviewProgress) {
			// 发送进度消息到前端
			var progressMsg string
			var status string
			switch progress.Type {
			case d_model.ReviewProgressItemReviewing:
				progressMsg = fmt.Sprintf("📝 评审中 (%d/%d): %s", progress.CurrentItem, progress.TotalItems, progress.CurrentItemName)
				status = "reviewing"
			case d_model.ReviewProgressItemCompleted:
				progressMsg = fmt.Sprintf("✅ 完成 (%d/%d): %s", progress.CurrentItem, progress.TotalItems, progress.CurrentItemName)
				status = "item_completed"
			case d_model.ReviewProgressDebating:
				progressMsg = fmt.Sprintf("💬 对论迭代中（第%d轮）", progress.DebateRound)
				status = "debating"
			case d_model.ReviewProgressModifying:
				progressMsg = "🔧 正在应用修改意见..."
				status = "modifying"
			case d_model.ReviewProgressNeedsManual:
				progressMsg = "⚠️ 需要人工处理"
				status = "needs_manual"
			case d_model.ReviewProgressCompleted:
				progressMsg = "✅ 全局评审完成"
				status = "completed"
			}
			if progressMsg != "" {
				logger.Debug("Global review progress", "message", progressMsg)
				// 通过 WebSocket 推送进度给前端
				if bridge != nil {
					progressData := map[string]any{
						"type":            string(progress.Type),
						"status":          status,
						"currentItem":     progress.CurrentItem,
						"totalItems":      progress.TotalItems,
						"currentItemName": progress.CurrentItemName,
						"currentItemType": string(progress.CurrentItemType),
						"debateRound":     progress.DebateRound,
						"violatedCount":   progress.ViolatedCount,
						"message":         progressMsg,
					}
					if err := bridge.BroadcastToSession(sess.ID, "global_review_progress", progressData); err != nil {
						logger.Warn("Failed to broadcast global review progress", "error", err)
					}
				}
			}
		}),
	)
	if err != nil {
		logger.Error("Failed to create global review executor", "error", err)
		if v3Config.GlobalReview.SkipOnError {
			// 跳过全局评审，直接进入确认保存
			return wctx.Send(ctx, CreateV3EventGlobalReviewSkip, nil)
		}
		return fmt.Errorf("failed to create global review executor: %w", err)
	}

	// 执行全局评审
	result, err := executor.Execute(ctx)
	if err != nil {
		logger.Error("Global review execution failed", "error", err)
		if v3Config.GlobalReview.SkipOnError {
			// 跳过全局评审，直接进入确认保存
			return wctx.Send(ctx, CreateV3EventGlobalReviewSkip, nil)
		}
		return fmt.Errorf("global review execution failed: %w", err)
	}

	// 保存评审结果到上下文
	createCtx.GlobalReviewResult = result

	// 如果有修改后的草案，更新工作草案
	if result.ModifiedDraft != nil {
		createCtx.WorkingDraft = result.ModifiedDraft
	}

	// 保存上下文
	if err := saveCreateV3Context(ctx, wctx, createCtx); err != nil {
		logger.Warn("Failed to save context after global review", "error", err)
	}

	// 根据结果决定下一步
	if result.NeedsManualReview {
		// 需要人工介入
		return wctx.Send(ctx, CreateV3EventGlobalReviewNeedsManual, result)
	}

	// 评审完成，无需人工介入
	return wctx.Send(ctx, CreateV3EventGlobalReviewCompleted, result)
}

// actOnGlobalReviewCompleted 全局评审完成（无需人工介入）
func actOnGlobalReviewCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Global review completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 构建评审完成消息
	result := createCtx.GlobalReviewResult
	handler := utils.NewGlobalReviewResultHandler(result)

	message := handler.GetReviewSummaryForUser()
	message += "\n\n请确认并保存排班结果。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send global review completion message", "error", err)
	}

	// 设置工作流按钮
	workflowActions := []session.WorkflowAction{
		{
			Label: "保存排班",
			Event: session.WorkflowEvent(CreateV3EventSaveCompleted),
			Style: session.ActionStylePrimary,
		},
		{
			Label: "取消",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 允许用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnGlobalReviewNeedsManual 全局评审需要人工介入
func actOnGlobalReviewNeedsManual(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Global review needs manual intervention", "sessionID", sess.ID)

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 构建需人工处理的消息
	result := createCtx.GlobalReviewResult
	handler := utils.NewGlobalReviewResultHandler(result)

	message := "⚠️ **全局评审需要人工处理**\n\n"
	message += handler.GetReviewSummaryForUser()
	message += "\n\n"
	message += handler.GetManualReviewSummary()
	message += "\n您可以在下方输入修改意见，或选择："

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send manual review message", "error", err)
	}

	// 设置工作流按钮
	workflowActions := []session.WorkflowAction{
		{
			Label: "确认并继续",
			Event: session.WorkflowEvent(CreateV3EventGlobalReviewManualConfirmed),
			Style: session.ActionStylePrimary,
		},
		{
			Label: "跳过直接保存",
			Event: session.WorkflowEvent(CreateV3EventGlobalReviewSkip),
			Style: session.ActionStyleSecondary,
		},
		{
			Label: "取消",
			Event: session.WorkflowEvent(CreateV3EventUserCancel),
			Style: session.ActionStyleDanger,
		},
	}

	// 允许用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	return session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID, "", workflowActions)
}

// actOnGlobalReviewSkip 跳过全局评审
func actOnGlobalReviewSkip(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Skipping global review", "sessionID", sess.ID)

	message := "ℹ️ 已跳过全局规则评审，直接进入保存确认阶段。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send skip message", "error", err)
	}

	// 允许用户输入
	if err := setAllowUserInput(ctx, wctx, sess.ID, true); err != nil {
		logger.Warn("Failed to enable user input", "error", err)
	}

	return nil
}

// actOnGlobalReviewManualConfirmed 人工处理确认完成
func actOnGlobalReviewManualConfirmed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Global review manual confirmed", "sessionID", sess.ID)

	message := "✅ 人工处理已确认，进入保存确认阶段。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send manual confirmed message", "error", err)
	}

	return nil
}

// actModifyGlobalReviewManual 人工处理修改
func actModifyGlobalReviewManual(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV3: Modifying global review manual items", "sessionID", sess.ID)

	// 获取用户输入的修改内容
	userMessage, ok := payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload type for manual modification")
	}

	createCtx, err := loadCreateV3Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 发送处理中消息
	processingMsg := fmt.Sprintf("收到您的修改意见：\n> %s\n\n正在理解并处理...", userMessage)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, processingMsg); err != nil {
		logger.Warn("Failed to send processing message", "error", err)
	}

	// 获取 AI 服务
	schedulingAIService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		return fmt.Errorf("scheduling AI service not found")
	}

	// 构建人工处理上下文
	manualContext := buildManualReviewContext(createCtx)

	// 调用 LLM 理解用户意图并生成修改方案
	modifyResult, err := schedulingAIService.ProcessManualReviewModification(
		ctx,
		userMessage,
		manualContext,
		createCtx.WorkingDraft,
		createCtx.StaffList,
		createCtx.SelectedShifts,
	)
	if err != nil {
		logger.Error("Failed to process manual modification", "error", err)
		errorMsg := fmt.Sprintf("❌ 处理修改意见时出错: %v\n\n请重新描述您的修改需求，或选择跳过。", err)
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); err != nil {
			logger.Warn("Failed to send error message", "error", err)
		}
		return nil
	}

	// 更新草案
	if modifyResult != nil && modifyResult.ModifiedDraft != nil {
		createCtx.WorkingDraft = modifyResult.ModifiedDraft
	}

	// 构建结果消息
	var resultMsg strings.Builder
	resultMsg.WriteString("✅ 已处理您的修改意见\n\n")
	if modifyResult != nil {
		resultMsg.WriteString(fmt.Sprintf("**处理结果**: %s\n", modifyResult.Summary))
		if len(modifyResult.AppliedChanges) > 0 {
			resultMsg.WriteString("\n**应用的修改**:\n")
			for _, change := range modifyResult.AppliedChanges {
				resultMsg.WriteString(fmt.Sprintf("- %s\n", change))
			}
		}
	}
	resultMsg.WriteString("\n您可以继续修改，或确认保存当前结果。")

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, resultMsg.String()); err != nil {
		logger.Warn("Failed to send result message", "error", err)
	}

	// 保存上下文
	return saveCreateV3Context(ctx, wctx, createCtx)
}

// buildManualReviewContext 构建人工评审上下文
func buildManualReviewContext(createCtx *CreateV3Context) *d_model.ManualReviewContext {
	if createCtx.GlobalReviewResult == nil {
		return nil
	}

	result := createCtx.GlobalReviewResult
	context := &d_model.ManualReviewContext{
		ManualReviewItems: make([]*d_model.ManualReviewItem, 0),
	}

	for _, item := range result.ManualReviewItems {
		context.ManualReviewItems = append(context.ManualReviewItems, &d_model.ManualReviewItem{
			OpinionID:            item.ID,
			ReviewItemName:       item.ReviewItemName,
			ReviewItemType:       item.ReviewItemType,
			ViolationDescription: item.ViolationDescription,
			Suggestion:           item.Suggestion,
			Status:               item.Status,
			ConflictReason:       strings.Join(item.ConflictingOpinionIDs, ", "),
		})
	}

	return context
}

// ============================================================
// 任务级人数校验辅助函数
// ============================================================

// generateSupplementTasksForShortages 为缺员生成补充任务
func generateSupplementTasksForShortages(
	ctx context.Context,
	wctx engine.Context,
	shortages []*executor.ShortageDetail,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	taskExecutor executor.IProgressiveTaskExecutor,
) []*d_model.ProgressiveTask {
	logger := wctx.Logger()

	// 由于 generateSupplementTasks 是私有方法，我们使用简单实现
	// 或者可以通过反射调用，但为了代码清晰，我们使用简单实现
	logger.Info("Generating supplement tasks for shortages", "shortageCount", len(shortages))
	return generateSupplementTasksSimple(shortages, shifts)
}

// generateSupplementTasksSimple 简单实现补充任务生成（备用）
func generateSupplementTasksSimple(
	shortages []*executor.ShortageDetail,
	shifts []*d_model.Shift,
) []*d_model.ProgressiveTask {
	// 按班次分组缺员
	shiftShortages := make(map[string][]*executor.ShortageDetail)
	for _, shortage := range shortages {
		shiftShortages[shortage.ShiftID] = append(shiftShortages[shortage.ShiftID], shortage)
	}

	// 构建班次ID到名称的映射
	shiftNameMap := make(map[string]string)
	for _, shift := range shifts {
		shiftNameMap[shift.ID] = shift.Name
	}

	tasks := make([]*d_model.ProgressiveTask, 0)
	taskIndex := 0

	for shiftID, shortageList := range shiftShortages {
		// 收集该班次的缺员日期
		dates := make([]string, 0, len(shortageList))
		totalShortage := 0
		shiftName := shiftNameMap[shiftID]
		if shiftName == "" {
			shiftName = shiftID
		}

		for _, shortage := range shortageList {
			dates = append(dates, shortage.Date)
			totalShortage += shortage.ShortageCount
		}

		// 构建任务描述
		var description strings.Builder
		description.WriteString(fmt.Sprintf("【补充任务】班次 %s 存在缺员，请补充排班：\n", shiftName))
		description.WriteString("\n**缺员详情**：\n")
		for _, shortage := range shortageList {
			description.WriteString(fmt.Sprintf("- %s：需要 %d 人，当前 %d 人，缺少 %d 人\n",
				shortage.Date, shortage.RequiredCount, shortage.ActualCount, shortage.ShortageCount))
		}
		description.WriteString(fmt.Sprintf("\n共计缺少 %d 人，请从可用人员中补充。\n", totalShortage))

		task := &d_model.ProgressiveTask{
			ID:           fmt.Sprintf("supplement_%s_%d", shiftID[:min(8, len(shiftID))], taskIndex),
			Title:        fmt.Sprintf("补充任务：%s 缺员补充", shiftName),
			Description:  description.String(),
			Type:         "ai",
			TargetShifts: []string{shiftID},
			TargetDates:  dates,
			Priority:     1,
		}

		tasks = append(tasks, task)
		taskIndex++
	}

	return tasks
}

// insertSupplementTasks 将补充任务插入到任务计划中
func insertSupplementTasks(createCtx *CreateV3Context, supplementTasks []*d_model.ProgressiveTask, insertPosition int) {
	if len(supplementTasks) == 0 {
		return
	}

	// 确保任务计划存在
	if createCtx.ProgressiveTaskPlan == nil {
		createCtx.ProgressiveTaskPlan = &d_model.ProgressiveTaskPlan{
			Tasks: make([]*d_model.ProgressiveTask, 0),
		}
	}

	// 插入补充任务
	newTasks := make([]*d_model.ProgressiveTask, 0, len(createCtx.ProgressiveTaskPlan.Tasks)+len(supplementTasks))
	newTasks = append(newTasks, createCtx.ProgressiveTaskPlan.Tasks[:insertPosition]...)
	newTasks = append(newTasks, supplementTasks...)
	if insertPosition < len(createCtx.ProgressiveTaskPlan.Tasks) {
		newTasks = append(newTasks, createCtx.ProgressiveTaskPlan.Tasks[insertPosition:]...)
	}

	createCtx.ProgressiveTaskPlan.Tasks = newTasks
}

// sendShortageNotification 发送缺员通知
func sendShortageNotification(
	ctx context.Context,
	wctx engine.Context,
	shortages []*executor.ShortageDetail,
	supplementTaskCount int,
) {
	logger := wctx.Logger()
	sess := wctx.Session()

	message := "⚠️ **任务执行后发现人数缺口**\n\n"
	message += fmt.Sprintf("发现 %d 处缺员，已自动生成 %d 个补充任务。\n\n", len(shortages), supplementTaskCount)
	message += "**缺员详情**：\n"
	for i, shortage := range shortages {
		if i < 5 {
			message += fmt.Sprintf("- %s %s：需要 %d 人，当前 %d 人，缺少 %d 人\n",
				shortage.Date, shortage.ShiftName, shortage.RequiredCount, shortage.ActualCount, shortage.ShortageCount)
		}
	}
	if len(shortages) > 5 {
		message += fmt.Sprintf("... 还有 %d 处缺员\n", len(shortages)-5)
	}
	message += "\n补充任务将在后续自动执行。"

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, message); err != nil {
		logger.Warn("Failed to send shortage notification", "error", err)
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
