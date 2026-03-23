package create

import (
	"context"
	"encoding/json"
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/agent/rostering/internal/workflow/common"
	"jusha/agent/rostering/internal/workflow/schedule_v4/executor"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
)

// ============================================================
// 上下文管理
// ============================================================

// loadCreateV4Context 加载V4上下文
func loadCreateV4Context(ctx context.Context, wctx engine.Context) (*CreateV4Context, error) {
	sess := wctx.Session()
	data, found, err := wctx.SessionService().GetData(ctx, sess.ID, KeyCreateV4Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}
	if !found {
		return NewCreateV4Context(), nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context: %w", err)
	}
	var createCtx CreateV4Context
	if err := json.Unmarshal(jsonBytes, &createCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}
	return &createCtx, nil
}

// saveCreateV4Context 保存V4上下文
func saveCreateV4Context(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	sess := wctx.Session()
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCreateV4Context, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}
	return nil
}

// ============================================================
// 工作流辅助
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

// serializePayload 序列化Payload
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

// parsePayloadToMap 解析Payload为map
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

// ============================================================
// 数据加载与初始化
// ============================================================

// populateInfoFromService 从服务加载基础信息
func populateInfoFromService(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Populating info from service", "sessionID", sess.ID)

	basicCtx, err := common.LoadScheduleBasicContext(
		ctx, wctx, sess.OrgID, createCtx.StartDate, createCtx.EndDate, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to load schedule basic context: %w", err)
	}

	createCtx.SelectedShifts = basicCtx.SelectedShifts
	createCtx.AllStaff = basicCtx.AllStaffList
	createCtx.ShiftMembersMap = basicCtx.ShiftMembersMap
	createCtx.Rules = basicCtx.Rules
	createCtx.StaffLeaves = basicCtx.StaffLeaves

	if basicCtx.StaffRequirements != nil {
		createCtx.StaffRequirements = make([]d_model.ShiftDateRequirement, 0)
		for shiftID, dateMap := range basicCtx.StaffRequirements {
			shiftName := ""
			for _, shift := range basicCtx.SelectedShifts {
				if shift.ID == shiftID {
					shiftName = shift.Name
					break
				}
			}
			for date, count := range dateMap {
				createCtx.StaffRequirements = append(createCtx.StaffRequirements, d_model.ShiftDateRequirement{
					ShiftID: shiftID, ShiftName: shiftName, Date: date, Count: count,
				})
			}
		}
	}

	if createCtx.StartDate == "" {
		createCtx.StartDate = basicCtx.StartDate
	}
	if createCtx.EndDate == "" {
		createCtx.EndDate = basicCtx.EndDate
	}

	logger.Info("CreateV4: Info populated",
		"shifts", len(createCtx.SelectedShifts),
		"staff", len(createCtx.AllStaff),
		"rules", len(createCtx.Rules))

	return nil
}

// ============================================================
// 排班执行辅助
// ============================================================

// fillFixedShiftSchedules 填充固定排班
func fillFixedShiftSchedules(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()

	rosteringService, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return fmt.Errorf("rostering service not available")
	}

	shiftIDs := make([]string, 0, len(createCtx.SelectedShifts))
	for _, shift := range createCtx.SelectedShifts {
		shiftIDs = append(shiftIDs, shift.ID)
	}
	if len(shiftIDs) == 0 {
		return nil
	}

	allFixedSchedules, err := rosteringService.CalculateMultipleFixedSchedules(
		ctx, shiftIDs, createCtx.StartDate, createCtx.EndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to calculate fixed schedules: %w", err)
	}
	if len(allFixedSchedules) == 0 {
		logger.Info("No fixed shift schedules found")
		return nil
	}

	createCtx.FixedAssignments = buildFixedAssignmentsFromSchedules(allFixedSchedules, createCtx.SelectedShifts)

	if createCtx.WorkingDraft == nil {
		createCtx.WorkingDraft = &d_model.ScheduleDraft{
			StartDate: createCtx.StartDate, EndDate: createCtx.EndDate,
			Shifts: make(map[string]*d_model.ShiftDraft),
		}
	}
	if createCtx.WorkingDraft.Shifts == nil {
		createCtx.WorkingDraft.Shifts = make(map[string]*d_model.ShiftDraft)
	}

	staffNames := make(map[string]string)
	for _, s := range createCtx.AllStaff {
		staffNames[s.ID] = s.Name
	}

	filledCount := 0
	for shiftID, schedule := range allFixedSchedules {
		if len(schedule) == 0 {
			continue
		}
		shiftDraft := createCtx.WorkingDraft.Shifts[shiftID]
		if shiftDraft == nil {
			shiftDraft = &d_model.ShiftDraft{ShiftID: shiftID, Days: make(map[string]*d_model.DayShift)}
			createCtx.WorkingDraft.Shifts[shiftID] = shiftDraft
		}
		if shiftDraft.Days == nil {
			shiftDraft.Days = make(map[string]*d_model.DayShift)
		}
		for date, staffIDs := range schedule {
			staff := make([]string, 0, len(staffIDs))
			for _, id := range staffIDs {
				if name, ok := staffNames[id]; ok {
					staff = append(staff, name)
				} else {
					staff = append(staff, id)
				}
			}
			requiredCount := createCtx.GetRequirement(shiftID, date)
			shiftDraft.Days[date] = &d_model.DayShift{
				Staff: staff, StaffIDs: staffIDs,
				RequiredCount: requiredCount, ActualCount: len(staffIDs), IsFixed: true,
			}
			for _, staffID := range staffIDs {
				createCtx.OccupySlot(staffID, date, shiftID)
			}
		}
		filledCount++
	}

	logger.Info("Fixed shift schedules filled", "shiftsWithFixed", filledCount)
	return nil
}

// executeSimplifiedScheduling 执行简化的排班逻辑
func executeSimplifiedScheduling(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context) error {
	logger := wctx.Logger()
	logger.Info("CreateV4: Executing simplified scheduling")

	if createCtx.WorkingDraft == nil {
		createCtx.WorkingDraft = &d_model.ScheduleDraft{
			StartDate: createCtx.StartDate, EndDate: createCtx.EndDate,
			Shifts: make(map[string]*d_model.ShiftDraft),
		}
	}

	staffNames := make(map[string]string)
	for _, s := range createCtx.AllStaff {
		staffNames[s.ID] = s.Name
	}

	dates := createCtx.GetAllDatesInRange()

	for _, shift := range createCtx.SelectedShifts {
		shiftDraft := createCtx.WorkingDraft.Shifts[shift.ID]
		if shiftDraft == nil {
			shiftDraft = &d_model.ShiftDraft{ShiftID: shift.ID, Days: make(map[string]*d_model.DayShift)}
			createCtx.WorkingDraft.Shifts[shift.ID] = shiftDraft
		}
		for _, date := range dates {
			if shiftDraft.Days[date] != nil && shiftDraft.Days[date].IsFixed {
				continue
			}
			requiredCount := createCtx.GetRequirement(shift.ID, date)
			if requiredCount == 0 {
				continue
			}
			selectedStaffIDs := make([]string, 0)
			selectedStaffNames := make([]string, 0)
			for _, staff := range createCtx.AllStaff {
				if len(selectedStaffIDs) >= requiredCount {
					break
				}
				if createCtx.IsStaffOccupied(staff.ID, date) {
					continue
				}
				selectedStaffIDs = append(selectedStaffIDs, staff.ID)
				selectedStaffNames = append(selectedStaffNames, staff.Name)
				createCtx.OccupySlot(staff.ID, date, shift.ID)
			}
			shiftDraft.Days[date] = &d_model.DayShift{
				Staff: selectedStaffNames, StaffIDs: selectedStaffIDs,
				RequiredCount: requiredCount, ActualCount: len(selectedStaffIDs), IsFixed: false,
			}
		}
	}

	createCtx.LLMCallCount = 0
	return nil
}

// applySchedulingResult 应用排班结果（合并到工作草稿，保留固定排班）
func applySchedulingResult(createCtx *CreateV4Context, result *executor.SchedulingExecutionResult) {
	if result == nil || result.Schedule == nil {
		return
	}
	if createCtx.WorkingDraft == nil {
		createCtx.WorkingDraft = &d_model.ScheduleDraft{
			StartDate: createCtx.StartDate, EndDate: createCtx.EndDate,
			Shifts: make(map[string]*d_model.ShiftDraft),
			StaffStats: make(map[string]*d_model.StaffStats),
			Conflicts: make([]*d_model.ScheduleConflict, 0),
		}
	}
	if createCtx.WorkingDraft.Shifts == nil {
		createCtx.WorkingDraft.Shifts = make(map[string]*d_model.ShiftDraft)
	}
	for shiftID, newShiftDraft := range result.Schedule.Shifts {
		if newShiftDraft == nil || newShiftDraft.Days == nil {
			continue
		}
		existingShiftDraft := createCtx.WorkingDraft.Shifts[shiftID]
		if existingShiftDraft == nil {
			existingShiftDraft = &d_model.ShiftDraft{
				ShiftID: shiftID, Days: make(map[string]*d_model.DayShift),
			}
			createCtx.WorkingDraft.Shifts[shiftID] = existingShiftDraft
		}
		if existingShiftDraft.Days == nil {
			existingShiftDraft.Days = make(map[string]*d_model.DayShift)
		}
		for date, newDayShift := range newShiftDraft.Days {
			if existingDay, exists := existingShiftDraft.Days[date]; exists && existingDay != nil && existingDay.IsFixed {
				continue
			}
			existingShiftDraft.Days[date] = newDayShift
		}
	}
	if result.Schedule.StaffStats != nil {
		if createCtx.WorkingDraft.StaffStats == nil {
			createCtx.WorkingDraft.StaffStats = make(map[string]*d_model.StaffStats)
		}
		for k, v := range result.Schedule.StaffStats {
			createCtx.WorkingDraft.StaffStats[k] = v
		}
	}
	if len(result.Schedule.Conflicts) > 0 {
		createCtx.WorkingDraft.Conflicts = append(createCtx.WorkingDraft.Conflicts, result.Schedule.Conflicts...)
	}
}

// ============================================================
// 数据转换辅助
// ============================================================

// convertEmployeesToStaff 转换员工列表为Staff（同一类型别名）
func convertEmployeesToStaff(employees []*d_model.Employee) []*d_model.Staff {
	result := make([]*d_model.Staff, len(employees))
	copy(result, employees)
	return result
}

// convertShiftMembersMap 将班次成员映射从 Employee 切片转换为 Staff 切片
func convertShiftMembersMap(m map[string][]*d_model.Employee) map[string][]*d_model.Staff {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string][]*d_model.Staff, len(m))
	for shiftID, members := range m {
		converted := make([]*d_model.Staff, len(members))
		copy(converted, members)
		result[shiftID] = converted
	}
	return result
}

// convertPersonalNeeds 转换个人需求
func convertPersonalNeeds(needs map[string][]*PersonalNeed) []*d_model.PersonalNeed {
	result := make([]*d_model.PersonalNeed, 0)
	for staffID, staffNeeds := range needs {
		for _, need := range staffNeeds {
			result = append(result, &d_model.PersonalNeed{
				StaffID: staffID, RequestType: need.RequestType,
				TargetShiftID: need.TargetShiftID, TargetDates: need.TargetDates,
				Description: need.Description, Priority: need.Priority,
			})
		}
	}
	return result
}

// convertFixedAssignments 将上下文中的固定排班转换为 executor 需要的指针切片
func convertFixedAssignments(assignments []d_model.CtxFixedShiftAssignment) []*d_model.CtxFixedShiftAssignment {
	result := make([]*d_model.CtxFixedShiftAssignment, 0, len(assignments))
	for i := range assignments {
		result = append(result, &assignments[i])
	}
	return result
}

// buildFixedAssignmentsFromSchedules 将固定排班转为上下文格式
func buildFixedAssignmentsFromSchedules(
	allFixedSchedules map[string]map[string][]string,
	selectedShifts []*d_model.Shift,
) []d_model.CtxFixedShiftAssignment {
	shiftNameByID := make(map[string]string)
	for _, shift := range selectedShifts {
		if shift != nil {
			shiftNameByID[shift.ID] = shift.Name
		}
	}
	out := make([]d_model.CtxFixedShiftAssignment, 0)
	for shiftID, dateToStaff := range allFixedSchedules {
		for date, staffIDs := range dateToStaff {
			if len(staffIDs) == 0 {
				continue
			}
			ids := make([]string, len(staffIDs))
			copy(ids, staffIDs)
			out = append(out, d_model.CtxFixedShiftAssignment{
				ShiftID: shiftID, ShiftName: shiftNameByID[shiftID],
				Date: date, StaffIDs: ids,
			})
		}
	}
	return out
}

// buildShiftRequirements 构建班次需求映射
func buildShiftRequirements(requirements []d_model.ShiftDateRequirement) map[string]map[string]int {
	result := make(map[string]map[string]int)
	for _, req := range requirements {
		if result[req.ShiftID] == nil {
			result[req.ShiftID] = make(map[string]int)
		}
		result[req.ShiftID][req.Date] = req.Count
	}
	return result
}

// saveSchedule 保存排班
func saveSchedule(ctx context.Context, service d_service.IRosteringService, createCtx *CreateV4Context) (string, error) {
	if createCtx.WorkingDraft == nil {
		return "", fmt.Errorf("no schedule to save")
	}
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range createCtx.SelectedShifts {
		shiftMap[shift.ID] = shift
	}
	items := make([]d_model.ScheduleUpsertRequest, 0)
	for shiftID, shiftDraft := range createCtx.WorkingDraft.Shifts {
		if shiftDraft.Days == nil {
			continue
		}
		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}
		for date, dayShift := range shiftDraft.Days {
			for _, staffID := range dayShift.StaffIDs {
				items = append(items, d_model.ScheduleUpsertRequest{
					UserID: staffID, WorkDate: date, ShiftCode: shift.ID,
					StartTime: shift.StartTime, EndTime: shift.EndTime,
					OrgID: createCtx.OrgID, Status: "active",
				})
			}
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no schedule items to save")
	}
	batch := d_model.ScheduleBatch{Items: items, OnConflict: "upsert"}
	result, err := service.BatchUpsertSchedules(ctx, batch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("saved_%d", result.Upserted), nil
}

// ============================================================
// 通用工具
// ============================================================

// firstOrEmpty 取切片第一个元素或空字符串
func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// sortStrings 简单字符串排序
func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
