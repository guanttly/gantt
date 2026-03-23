// Package create 排班创建工作流 Actions（重构版）
//
// 父工作流职责：
// 1. 启动并协调三个子工作流
// 2. 处理子工作流返回结果
// 3. 管理整体排班进度和状态
package create

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 会话数据 Keys
// ============================================================

const (
	// KeyCreateContext 创建工作流上下文
	KeyCreateContext = "create_context"
	// KeyCurrentShiftIndex 当前处理的班次索引
	KeyCurrentShiftIndex = "current_shift_index"
	// KeyShiftResults 所有班次的排班结果
	KeyShiftResults = "shift_results"
	// KeyTotalShifts 总班次数
	KeyTotalShifts = "total_shifts"
)

// ============================================================
// CreateContext 创建工作流上下文（从 InfoCollect 输出构建）
// ============================================================

// CreateContext 创建工作流的核心上下文
type CreateContext struct {
	// 排班周期
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`

	// 选定的班次
	SelectedShifts []*d_model.Shift `json:"selected_shifts"`

	// 人员配置（shift_id -> date -> 人数）
	StaffRequirements map[string]map[string]int `json:"staff_requirements"`

	// 已检索的数据
	StaffList   []*d_model.Employee `json:"staff_list"`   // 可用人员
	GlobalRules []*d_model.Rule     `json:"global_rules"` // 全局规则
	ShiftRules  []*d_model.Rule     `json:"shift_rules"`  // 班次规则

	// 进度跟踪
	CompletedShiftCount int `json:"completed_shift_count"`
	SkippedShiftCount   int `json:"skipped_shift_count"`
}

// ============================================================
// 初始化阶段 Actions
// ============================================================

// actCreateStartPrepare 准备启动创建工作流（Act 阶段）
// 职责：发送欢迎消息，准备数据
func actCreateStartPrepare(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: Starting create workflow", "sessionID", sess.ID)

	// 发送欢迎消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "🚀 开始创建排班，首先需要收集一些信息..."); err != nil {
		logger.Warn("Failed to send welcome message", "error", err)
	}

	return nil
}

// actCreateStartSpawnSubWorkflow 启动 InfoCollect 子工作流（AfterAct 阶段）
// 职责：在状态转换完成后启动子工作流，确保返回时状态正确
func actCreateStartSpawnSubWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()

	// 构建 InfoCollect 子工作流的输入
	input := InfoCollectInput{
		SourceType: "create",
	}

	// 检查 payload 中是否有预设值
	if p, ok := payload.(map[string]any); ok {
		if startDate, ok := p["start_date"].(string); ok {
			input.PresetStartDate = startDate
		}
		if endDate, ok := p["end_date"].(string); ok {
			input.PresetEndDate = endDate
		}
		if shiftIDs, ok := p["shift_ids"].([]string); ok {
			input.PresetShiftIDs = shiftIDs
		}
		if skipPhases, ok := p["skip_phases"].([]string); ok {
			input.SkipPhases = skipPhases
		}
	}

	// 构建输入数据 map
	inputMap := map[string]any{
		"source_type": input.SourceType,
	}
	if input.PresetStartDate != "" {
		inputMap["preset_start_date"] = input.PresetStartDate
	}
	if input.PresetEndDate != "" {
		inputMap["preset_end_date"] = input.PresetEndDate
	}
	if len(input.PresetShiftIDs) > 0 {
		inputMap["preset_shift_ids"] = input.PresetShiftIDs
	}
	if len(input.SkipPhases) > 0 {
		inputMap["skip_phases"] = input.SkipPhases
	}

	// 获取 Actor 并启动子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor")
	}

	// 配置子工作流
	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowInfoCollect,
		Input:        inputMap,
		OnComplete:   CreateEventInfoCollected,
		OnError:      CreateEventSubFailed,
		Timeout:      0, // 无超时，等待用户输入
	}

	logger.Info("Create: Spawning InfoCollect sub-workflow")

	return actor.SpawnSubWorkflow(ctx, config)
}

// actCreateCancel 用户取消工作流
func actCreateCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: User cancelled workflow", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "❌ 排班创建已取消"); err != nil {
		logger.Warn("Failed to send cancel message", "error", err)
	}

	return nil
}

// ============================================================
// 信息收集阶段 Actions
// ============================================================

// actCreateOnInfoCollected 处理信息收集子工作流完成
func actCreateOnInfoCollected(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: InfoCollect sub-workflow completed", "sessionID", sess.ID)

	// 从 payload 解析子工作流结果
	result, ok := payload.(*engine.SubWorkflowResult)
	if !ok {
		return fmt.Errorf("invalid payload type for info collected event")
	}

	if !result.Success {
		return fmt.Errorf("info collect sub-workflow failed: %v", result.Error)
	}

	// 从 session.Data 读取 ScheduleCreateContext（由 InfoCollect 子工作流填充）
	scheduleCtx := GetOrCreateScheduleContext(sess)

	// 构建创建上下文
	createCtx := &CreateContext{
		StartDate:         scheduleCtx.StartDate,
		EndDate:           scheduleCtx.EndDate,
		SelectedShifts:    scheduleCtx.SelectedShifts,
		StaffRequirements: scheduleCtx.ShiftStaffRequirements, // 直接传递完整的 map[shiftID]map[date]count
		StaffList:         scheduleCtx.StaffList,
		GlobalRules:       scheduleCtx.GlobalRules,
		ShiftRules:        flattenShiftRules(scheduleCtx.ShiftRules),
	}

	// 保存上下文到 session
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCreateContext, createCtx); err != nil {
		return fmt.Errorf("failed to save create context: %w", err)
	}

	// 初始化进度
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCurrentShiftIndex, 0); err != nil {
		return fmt.Errorf("failed to init shift index: %w", err)
	}
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyTotalShifts, len(createCtx.SelectedShifts)); err != nil {
		return fmt.Errorf("failed to save total shifts: %w", err)
	}
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyShiftResults, make(map[string]*d_model.ShiftScheduleDraft)); err != nil {
		return fmt.Errorf("failed to init shift results: %w", err)
	}

	logger.Info("Create: Starting core scheduling phase",
		"totalShifts", len(createCtx.SelectedShifts),
		"startDate", createCtx.StartDate,
		"endDate", createCtx.EndDate,
	)

	// 发送进度消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		fmt.Sprintf("📊 信息收集完成！共选择 %d 个班次，开始生成排班...", len(createCtx.SelectedShifts))); err != nil {
		logger.Warn("Failed to send progress message", "error", err)
	}

	// SpawnSubWorkflow 移到 AfterAct 执行，确保状态已转换
	return nil
}

// actCreateSpawnCoreWorkflow 在状态转换后启动第一个 Core 子工作流
func actCreateSpawnCoreWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
	return startCoreForNextShift(ctx, wctx)
}

// ============================================================
// 核心排班阶段 Actions
// ============================================================

// actCreateOnShiftCompleted 处理单个班次排班完成
func actCreateOnShiftCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: Shift scheduling completed", "sessionID", sess.ID)

	// 解析子工作流结果
	result, ok := payload.(*engine.SubWorkflowResult)
	if !ok {
		return fmt.Errorf("invalid payload type for shift completed event")
	}

	// 获取当前上下文
	createCtx, err := getCreateContext(sess)
	if err != nil {
		return err
	}

	// 获取当前班次索引
	currentIndex := getIntFromSession(sess, KeyCurrentShiftIndex)
	if currentIndex >= len(createCtx.SelectedShifts) {
		// 已经没有更多班次了
		return wctx.Send(ctx, CreateEventAllShiftsDone, nil)
	}

	currentShift := createCtx.SelectedShifts[currentIndex]

	// 处理结果
	if result.Success {
		// 保存排班结果
		if resultDraft, ok := result.Output["result_draft"].(*d_model.ShiftScheduleDraft); ok && resultDraft != nil {
			shiftResults := getShiftResults(sess)
			shiftResults[currentShift.ID] = resultDraft
			if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyShiftResults, shiftResults); err != nil {
				logger.Warn("Failed to save shift result", "error", err)
			}
		}
		createCtx.CompletedShiftCount++
	} else {
		// 检查是否是跳过
		if skipped, ok := result.Output["skipped"].(bool); ok && skipped {
			createCtx.SkippedShiftCount++
			logger.Info("Create: Shift was skipped", "shiftName", currentShift.Name)
		}
	}

	// 更新上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCreateContext, createCtx); err != nil {
		logger.Warn("Failed to update create context", "error", err)
	}

	// 更新索引，处理下一个班次
	currentIndex++
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCurrentShiftIndex, currentIndex); err != nil {
		return fmt.Errorf("failed to update shift index: %w", err)
	}

	// SpawnSubWorkflow 移到 AfterAct 执行
	return nil
}

// actCreateSpawnNextCoreOrComplete 在状态转换后启动下一个 Core 子工作流或完成
func actCreateSpawnNextCoreOrComplete(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取当前上下文
	createCtx, err := getCreateContext(sess)
	if err != nil {
		return err
	}

	// 获取当前班次索引
	currentIndex := getIntFromSession(sess, KeyCurrentShiftIndex)

	// 检查是否还有更多班次
	if currentIndex >= len(createCtx.SelectedShifts) {
		logger.Info("Create: All shifts processed",
			"completed", createCtx.CompletedShiftCount,
			"skipped", createCtx.SkippedShiftCount,
		)
		return wctx.Send(ctx, CreateEventAllShiftsDone, nil)
	}

	// 继续处理下一个班次
	return startCoreForNextShift(ctx, wctx)
}

// convertShiftResultsToScheduleDraft 将 shiftResults 转换为 ScheduleDraft
// shiftResults: map[shiftID]*ShiftScheduleDraft (每个班次的 Todo 执行结果)
// 转换为: ScheduleDraft (ConfirmSave 需要的格式)
func convertShiftResultsToScheduleDraft(
	createCtx *CreateContext,
	shiftResults map[string]*d_model.ShiftScheduleDraft,
) *d_model.ScheduleDraft {

	draft := &d_model.ScheduleDraft{
		StartDate:  createCtx.StartDate,
		EndDate:    createCtx.EndDate,
		Shifts:     make(map[string]*d_model.ShiftDraft),
		StaffStats: make(map[string]*d_model.StaffStats),
		Conflicts:  make([]*d_model.ScheduleConflict, 0),
	}

	// 构建班次名称映射
	shiftNameMap := make(map[string]string)
	shiftPriorityMap := make(map[string]int)
	for _, shift := range createCtx.SelectedShifts {
		shiftNameMap[shift.ID] = shift.Name
		shiftPriorityMap[shift.ID] = shift.Priority
	}

	// 构建人员名称映射
	staffNameMap := make(map[string]string)
	for _, staff := range createCtx.StaffList {
		staffNameMap[staff.ID] = staff.Name
	}

	// 转换每个班次的结果
	for shiftID, shiftDraft := range shiftResults {
		if shiftDraft == nil || shiftDraft.Schedule == nil {
			continue
		}

		// 创建 ShiftDraft
		sd := &d_model.ShiftDraft{
			ShiftID:  shiftID,
			Priority: shiftPriorityMap[shiftID],
			Days:     make(map[string]*d_model.DayShift),
		}

		// 转换每天的排班 (ShiftScheduleDraft.Schedule: date -> []staffID)
		for date, staffIDs := range shiftDraft.Schedule {
			// 获取人员姓名
			staffNames := make([]string, 0, len(staffIDs))
			for _, staffID := range staffIDs {
				if name, ok := staffNameMap[staffID]; ok {
					staffNames = append(staffNames, name)
				} else {
					staffNames = append(staffNames, staffID) // 回退使用 ID
				}
			}

			// 获取需求人数（从按天配置中读取）
			requiredCount := 1
			if dateReqs, ok := createCtx.StaffRequirements[shiftID]; ok {
				if count, ok := dateReqs[date]; ok && count > 0 {
					requiredCount = count
				}
			}

			sd.Days[date] = &d_model.DayShift{
				Staff:         staffNames,
				StaffIDs:      staffIDs,
				RequiredCount: requiredCount,
				ActualCount:   len(staffIDs),
			}

			// 统计人员工作信息
			for _, staffID := range staffIDs {
				if _, exists := draft.StaffStats[staffID]; !exists {
					draft.StaffStats[staffID] = &d_model.StaffStats{
						StaffID:   staffID,
						StaffName: staffNameMap[staffID],
						WorkDays:  0,
						Shifts:    make([]string, 0),
					}
				}
				stats := draft.StaffStats[staffID]
				stats.WorkDays++
				stats.Shifts = append(stats.Shifts, shiftNameMap[shiftID])
			}
		}

		draft.Shifts[shiftID] = sd

		// 暂不收集遗留问题为冲突（校验功能已禁用）
		// for _, issue := range shiftDraft.RemainingIssues {
		// 	draft.Conflicts = append(draft.Conflicts, &d_model.ScheduleConflict{
		// 		Shift:    shiftNameMap[shiftID],
		// 		Issue:    issue,
		// 		Severity: "warning",
		// 	})
		// }
	}

	return draft
}

// actCreateOnAllShiftsDone 所有班次排班完成
func actCreateOnAllShiftsDone(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: All shifts scheduling completed", "sessionID", sess.ID)

	// 获取上下文
	createCtx, err := getCreateContext(sess)
	if err != nil {
		return err
	}

	// 发送完成消息
	completionMsg := fmt.Sprintf("✅ 排班生成完成！共处理 %d 个班次，成功 %d 个，跳过 %d 个",
		len(createCtx.SelectedShifts),
		createCtx.CompletedShiftCount,
		createCtx.SkippedShiftCount,
	)
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, completionMsg); err != nil {
		logger.Warn("Failed to send completion message", "error", err)
	}

	// 获取所有排班结果
	shiftResults := getShiftResults(sess)

	// ★ 关键：将 shiftResults 转换为 ScheduleDraft 并保存到 ScheduleCreateContext
	scheduleDraft := convertShiftResultsToScheduleDraft(createCtx, shiftResults)

	// 保存到 ScheduleCreateContext.DraftSchedule
	scheduleCtx := GetOrCreateScheduleContext(sess)
	scheduleCtx.DraftSchedule = scheduleDraft
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleCreateContext, scheduleCtx); err != nil {
		return fmt.Errorf("failed to save schedule context with draft: %w", err)
	}

	logger.Info("Create: Converted shift results to ScheduleDraft",
		"shiftCount", len(scheduleDraft.Shifts),
		"staffCount", len(scheduleDraft.StaffStats),
		"conflictCount", len(scheduleDraft.Conflicts),
	)

	// 触发状态转换（子工作流启动在 AfterAct 中执行）
	return nil
}

// actCreateSpawnConfirmSaveWorkflow 状态转换后启动 ConfirmSave 子工作流
func actCreateSpawnConfirmSaveWorkflow(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取上下文
	createCtx, err := getCreateContext(sess)
	if err != nil {
		return err
	}

	// 构建 ConfirmSave 子工作流输入
	inputMap := map[string]any{
		"source_type":   "create",
		"start_date":    createCtx.StartDate,
		"end_date":      createCtx.EndDate,
		"total_shifts":  len(createCtx.SelectedShifts),
		"skipped_count": createCtx.SkippedShiftCount,
	}

	// 启动确认保存子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowConfirmSave,
		Input:        inputMap,
		OnComplete:   CreateEventSaveCompleted,
		OnError:      CreateEventSubFailed,
		Timeout:      0, // 无超时，等待用户确认
	}

	logger.Info("Create: Spawning ConfirmSave sub-workflow")

	return actor.SpawnSubWorkflow(ctx, config)
}

// ============================================================
// 确认保存阶段 Actions
// ============================================================

// actCreateOnSaveCompleted 保存完成
func actCreateOnSaveCompleted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: Save completed", "sessionID", sess.ID)

	// 解析子工作流结果
	result, ok := payload.(*engine.SubWorkflowResult)
	if !ok {
		return fmt.Errorf("invalid payload type for save completed event")
	}

	if !result.Success {
		return fmt.Errorf("save failed: %v", result.Error)
	}

	// 获取保存的 schedule_id
	scheduleID := ""
	if id, ok := result.Output["schedule_id"].(string); ok {
		scheduleID = id
	}

	// 发送最终完成消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		fmt.Sprintf("🎉 排班已成功保存！排班ID: %s", scheduleID)); err != nil {
		logger.Warn("Failed to send final completion message", "error", err)
	}

	return nil
}

// ============================================================
// 通用错误处理 Actions
// ============================================================

// actCreateOnSubCancelled 子工作流被取消
func actCreateOnSubCancelled(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Create: Sub-workflow cancelled", "sessionID", sess.ID)

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, "❌ 排班创建已取消"); err != nil {
		logger.Warn("Failed to send cancelled message", "error", err)
	}

	return nil
}

// actCreateOnSubFailed 子工作流失败
func actCreateOnSubFailed(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("Create: Sub-workflow failed", "sessionID", sess.ID)

	var errMsg string
	if result, ok := payload.(*engine.SubWorkflowResult); ok && result.Error != nil {
		errMsg = result.Error.Error()
	} else {
		errMsg = "未知错误"
	}

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		fmt.Sprintf("❌ 排班创建失败：%s", errMsg)); err != nil {
		logger.Warn("Failed to send error message", "error", err)
	}

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// startCoreForNextShift 启动下一个班次的核心排班子工作流
func startCoreForNextShift(ctx context.Context, wctx engine.Context) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	// 获取创建上下文
	createCtx, err := getCreateContext(sess)
	if err != nil {
		return err
	}

	// 获取当前班次索引
	currentIndex := getIntFromSession(sess, KeyCurrentShiftIndex)
	totalShifts := len(createCtx.SelectedShifts)

	if currentIndex >= totalShifts {
		// 所有班次已处理
		return wctx.Send(ctx, CreateEventAllShiftsDone, nil)
	}

	currentShift := createCtx.SelectedShifts[currentIndex]

	// 发送进度消息
	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID,
		fmt.Sprintf("📅 正在处理班次 (%d/%d): 【%s】", currentIndex+1, totalShifts, currentShift.Name)); err != nil {
		logger.Warn("Failed to send progress message", "error", err)
	}

	// 构建 ShiftSchedulingContext
	shiftCtx := d_model.NewShiftSchedulingContext(
		currentShift,
		createCtx.StartDate,
		createCtx.EndDate,
		"create",
	)

	// 设置共享数据
	shiftCtx.StaffList = createCtx.StaffList
	shiftCtx.GlobalRules = createCtx.GlobalRules
	shiftCtx.ShiftRules = createCtx.ShiftRules

	// 设置人数需求(直接使用用户配置的每日人数)
	if dateReqs, ok := createCtx.StaffRequirements[currentShift.ID]; ok {
		shiftCtx.StaffRequirements = dateReqs
	} else {
		// 如果没有配置,使用默认值(每天1人)
		shiftCtx.StaffRequirements = buildDateRequirements(
			createCtx.StartDate,
			createCtx.EndDate,
			1,
		)
	}

	// ★ 关键:传递之前已排班次的人员标记,避免重复安排
	// 从已完成的班次结果构建 ExistingScheduleMarks
	if currentIndex > 0 {
		// 获取已保存的班次结果
		shiftResults := getShiftResults(sess)
		if len(shiftResults) > 0 {
			// 将 shiftResults 转换为临时的 ScheduleDraft
			tempDraft := &d_model.ScheduleDraft{
				Shifts: make(map[string]*d_model.ShiftDraft),
			}
			for shiftID, shiftScheduleDraft := range shiftResults {
				// 构建 ShiftDraft
				sd := &d_model.ShiftDraft{
					ShiftID: shiftID,
					Days:    make(map[string]*d_model.DayShift),
				}
				for date, staffIDs := range shiftScheduleDraft.Schedule {
					sd.Days[date] = &d_model.DayShift{
						StaffIDs: staffIDs,
					}
				}
				tempDraft.Shifts[shiftID] = sd
			}
			// 构建已排班标记(排除当前班次)
			// 需要传递所有班次信息以获取时间段
			shiftCtx.ExistingScheduleMarks = buildExistingScheduleMarksFromResults(tempDraft, currentShift.ID, createCtx.SelectedShifts)
		}
	}

	// 保存 ShiftSchedulingContext 到 session（Core 子工作流会读取）
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
		return fmt.Errorf("failed to save shift context: %w", err)
	}

	// 启动 Core 子工作流
	actor, ok := wctx.(*engine.Actor)
	if !ok {
		return fmt.Errorf("context is not an Actor")
	}

	config := engine.SubWorkflowConfig{
		WorkflowName: WorkflowSchedulingCore,
		Input:        nil, // Core 从 session 读取 ShiftSchedulingContext
		OnComplete:   CreateEventShiftCompleted,
		OnError:      CreateEventShiftCompleted, // 失败也触发完成事件，由父工作流处理
		Timeout:      10 * 60 * 1e9,             // 10 分钟超时 (纳秒)
	}

	logger.Info("Create: Spawning Core sub-workflow",
		"shiftIndex", currentIndex,
		"shiftName", currentShift.Name,
	)

	return actor.SpawnSubWorkflow(ctx, config)
}

// getCreateContext 从 session 获取创建上下文
func getCreateContext(sess *session.Session) (*CreateContext, error) {
	data, ok := sess.Data[KeyCreateContext]
	if !ok {
		return nil, fmt.Errorf("create context not found in session")
	}

	createCtx, ok := data.(*CreateContext)
	if !ok {
		// 尝试从 map 转换
		if m, ok := data.(map[string]any); ok {
			bytes, _ := json.Marshal(m)
			createCtx = &CreateContext{}
			if err := json.Unmarshal(bytes, createCtx); err != nil {
				return nil, fmt.Errorf("failed to parse create context: %w", err)
			}
			return createCtx, nil
		}
		return nil, fmt.Errorf("invalid create context type")
	}

	return createCtx, nil
}

// getIntFromSession 从 session 获取整数值
func getIntFromSession(sess *session.Session, key string) int {
	if data, ok := sess.Data[key]; ok {
		switch v := data.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

// getShiftResults 从 session 获取班次结果映射
func getShiftResults(sess *session.Session) map[string]*d_model.ShiftScheduleDraft {
	if data, ok := sess.Data[KeyShiftResults]; ok {
		if results, ok := data.(map[string]*d_model.ShiftScheduleDraft); ok {
			return results
		}
	}
	return make(map[string]*d_model.ShiftScheduleDraft)
}

// flattenShiftRules 将 map[shiftID][]Rule 合并为单一的 []Rule
func flattenShiftRules(shiftRules map[string][]*d_model.Rule) []*d_model.Rule {
	var result []*d_model.Rule
	for _, rules := range shiftRules {
		result = append(result, rules...)
	}
	return result
}

// buildDateRequirements 将每日人数需求展开为 map[date]count
func buildDateRequirements(startDate, endDate string, dailyCount int) map[string]int {
	result := make(map[string]int)

	if dailyCount <= 0 {
		dailyCount = 1 // 默认至少1人
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return result
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return result
	}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		result[d.Format("2006-01-02")] = dailyCount
	}

	return result
}

// buildExistingScheduleMarksFromResults 从已完成的班次结果构建排班标记
// 用于创建工作流中传递前面班次的排班情况给后续班次
func buildExistingScheduleMarksFromResults(draft *d_model.ScheduleDraft, excludeShiftID string, shifts []*d_model.Shift) map[string]map[string][]*d_model.ShiftMark {
	marks := make(map[string]map[string][]*d_model.ShiftMark)

	if draft == nil || draft.Shifts == nil {
		return marks
	}

	shiftMap := make(map[string]*d_model.Shift)
	for _, s := range shifts {
		shiftMap[s.ID] = s
	}

	for shiftID, shiftDraft := range draft.Shifts {
		if shiftID == excludeShiftID {
			continue
		}

		shift := shiftMap[shiftID]
		if shift == nil {
			continue
		}

		for date, dayShift := range shiftDraft.Days {
			for _, staffID := range dayShift.StaffIDs {
				if marks[staffID] == nil {
					marks[staffID] = make(map[string][]*d_model.ShiftMark)
				}
				marks[staffID][date] = append(marks[staffID][date], &d_model.ShiftMark{
					ShiftID:   shiftID,
					ShiftName: shift.Name,
					StartTime: shift.StartTime,
					EndTime:   shift.EndTime,
				})
			}
		}
	}

	return marks
}
