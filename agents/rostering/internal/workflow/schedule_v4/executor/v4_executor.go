package executor

import (
	"context"
	"fmt"
	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
	"sort"
	"time"
)

// OnConflictCallback 排班冲突回调（用于向前端推送结构化冲突信息）
type OnConflictCallback func(conflict *model.ScheduleConflict)

// V4Executor V4排班执行器
type V4Executor struct {
	logger        logging.ILogger
	ruleEngine    *engine.RuleEngine
	ruleOrganizer *RuleOrganizer

	// 新架构：原子排班 + 依赖协调
	singleScheduler    *SingleShiftScheduler
	dependencyResolver *DependencyResolver

	// 调度器链 (按优先级排序) — 保留用于 PersonnelRule/Exclusive 等非依赖场景
	schedulers []IScheduler

	// OnConflict 冲突回调（可选），排班遇到依赖不满足等错误时触发
	OnConflict OnConflictCallback
}

// NewV4Executor 创建V4执行器
func NewV4Executor(
	logger logging.ILogger,
	ruleEngine *engine.RuleEngine,
	ruleOrganizer *RuleOrganizer,
) *V4Executor {
	// 初始化并按优先级注册调度器（保留用于非依赖场景）
	schedulers := []IScheduler{
		NewPersonnelRuleScheduler(logger, ruleEngine),
		NewRequiredTogetherScheduler(logger, ruleEngine),
		NewExclusiveScheduler(logger, ruleEngine),
		NewDefaultScheduler(logger, ruleEngine),
	}

	// 新架构组件
	singleScheduler := NewSingleShiftScheduler(logger, ruleEngine)
	depResolver := NewDependencyResolver(logger, singleScheduler)

	return &V4Executor{
		logger:             logger,
		ruleEngine:         ruleEngine,
		ruleOrganizer:      ruleOrganizer,
		singleScheduler:    singleScheduler,
		dependencyResolver: depResolver,
		schedulers:         schedulers,
	}
}

// ExecuteScheduling 执行排班（三阶段）
//
// 原单阶段模式的问题：
//
//	班次按顺序逐个处理，每个班次内部直接运行包括 DefaultScheduler 在内的完整调度器链。
//	若班次 B 同时与 A、C 存在 required_together 关系，且 A、C 的候选人完全不重叠，
//	则 B 的 DefaultScheduler 会在 C 的规则调度运行之前就填满剩余名额，
//	抢占 C 所需的候选人，导致 C 无人可排。
//
// 三阶段解决方案：
//
//	阶段零（固定排班占位）：遍历所有班次，将 FixedAssignments 中配置的固定人员
//	  直接写入 CurrentDraft，标记为 IsFixed=true。
//	  此阶段纯粹是数据写入，不执行任何调度器逻辑。
//	  阶段一和阶段二均会感知这些固定占位，不再重复选人。
//
//	阶段一（规则性占位）：遍历所有班次，只运行规则类调度器
//	  （PersonnelRule / RequiredTogether / Exclusive），跳过 DefaultScheduler。
//	  此阶段开始时已从 CurrentDraft 读取阶段零的固定人员，剩余名额 = 总需求 - 固定人数。
//	  此阶段结束后，所有班次的「规则约束人员」已全部写入 CurrentDraft。
//
//	阶段二（兜底填充）：遍历所有班次，只运行 DefaultScheduler，
//	  补充阶段零+阶段一后未填满的剩余名额。
//	  由于阶段零已固定人员、阶段一已确定全局规则占位，
//	  DefaultScheduler 填充时不会抢占其他班次的固定/规则预留人员。
func (e *V4Executor) ExecuteScheduling(
	ctx context.Context,
	input *SchedulingExecutionInput,
) (*SchedulingExecutionResult, error) {
	// 1. 规则组织（如果尚未组织）
	if input.RuleOrganization == nil {
		org, err := e.ruleOrganizer.OrganizeRules(
			ctx,
			input.OrgID,
			input.Rules,
			input.Shifts,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("规则组织失败: %w", err)
		}
		input.RuleOrganization = org
	}

	// 2. 初始化结果
	result := &SchedulingExecutionResult{
		Schedule: &model.ScheduleDraft{
			StartDate:  input.StartDate,
			EndDate:    input.EndDate,
			Shifts:     make(map[string]*model.ShiftDraft),
			StaffStats: make(map[string]*model.StaffStats),
			Conflicts:  make([]*model.ScheduleConflict, 0),
		},
		ShiftResults: make(map[string]*ShiftSchedulingResult),
	}

	// 按依赖关系排序的班次执行顺序
	shiftOrder := input.RuleOrganization.ShiftExecutionOrder
	if len(shiftOrder) == 0 {
		for _, shift := range input.Shifts {
			shiftOrder = append(shiftOrder, shift.ID)
		}
	}

	// 构建员工名称映射
	staffNames := make(map[string]string)
	for _, staff := range input.AllStaff {
		staffNames[staff.ID] = staff.Name
	}

	// 初始化工作草稿（用于跟踪已排班状态）
	if input.CurrentDraft == nil {
		input.CurrentDraft = result.Schedule
	}

	// 3. 分离调度器：规则类（阶段一使用，阶段二走 SingleShiftScheduler）
	var ruleSchedulers []IScheduler
	for _, s := range e.schedulers {
		if s.Name() != "DefaultScheduler" {
			ruleSchedulers = append(ruleSchedulers, s)
		}
	}

	// ── 阶段零：固定排班占位 ──
	// 将 FixedAssignments 中配置的固定人员直接写入 CurrentDraft（IsFixed=true）。
	// 后续阶段一、阶段二均从 CurrentDraft 读取已占位数量，不会重复安排固定人员。
	e.applyFixedAssignments(input, shiftOrder, staffNames)

	// ── 阶段一：规则性占位 + 依赖解析 ──
	// 使用 DependencyResolver 处理 required_together 依赖关系，
	// 使用原有 PersonnelRule/Exclusive 调度器处理非依赖规则。
	// 依赖不满足时直接返回错误，不再静默降级。
	for _, shiftID := range shiftOrder {
		shiftResult, err := e.executePhaseOne(ctx, input, shiftID, staffNames, ruleSchedulers)
		if err != nil {
			// 检查是否为结构化排班错误
			if schedErr, ok := err.(*SchedulingError); ok {
				shift := findShiftByID(input.Shifts, shiftID)
				shiftName := shiftID
				if shift != nil {
					shiftName = shift.Name
				}
				conflict := &model.ScheduleConflict{
					Date:           schedErr.Date,
					Shift:          shiftName,
					Issue:          schedErr.Reason,
					Severity:       "error",
					ConflictType:   model.ConflictTypeDependencyUnresolved,
					RelatedRuleIDs: schedErr.RelatedRules,
					Detail:         schedErr.Detail,
				}
				result.Schedule.Conflicts = append(result.Schedule.Conflicts, conflict)
				if e.OnConflict != nil {
					e.OnConflict(conflict)
				}
			}
			return result, fmt.Errorf("阶段一班次规则排班失败 %s: %w", shiftID, err)
		}
		result.ShiftResults[shiftID] = shiftResult
		e.writeShiftResultToSchedule(result.Schedule, shiftID, shiftResult, staffNames, input)
	}

	// ── 阶段二：兜底填充（Source=default, Priority=low）──
	// 使用 SingleShiftScheduler 补充剩余名额，标记为低优先级（可被后续依赖补排置换）
	// 重要：按候选人稀缺度排序——候选人池越小的班次越先填充，避免大班次抢占小池班次的全部候选人。
	// 例如：本部穿刺（7候选人，需1人）应先于 CT/MRI报告上（45候选人，需22人）。
	phaseTwoOrder := e.buildPhaseTwoOrder(shiftOrder, input)
	for _, shiftID := range phaseTwoOrder {
		if err := e.executePhaseTwo(
			ctx, input, shiftID, staffNames,
			result.ShiftResults[shiftID],
		); err != nil {
			return nil, fmt.Errorf("阶段二班次填充失败 %s: %w", shiftID, err)
		}
	}

	// ── 阶段二完成后：将完整结果同步到 result.Schedule ──
	for _, shiftID := range shiftOrder {
		e.writeShiftResultToSchedule(result.Schedule, shiftID, result.ShiftResults[shiftID], staffNames, input)
	}

	return result, nil
}

// executePhaseOne 阶段一：规则性占位 + 依赖解析
// 对每个班次-日期组合：
//  1. 先运行 PersonnelRule 调度器（抢占强制排班人员）
//  2. 检查是否有 required_together 依赖 → 如有，使用 DependencyResolver
//  3. 运行 Exclusive 调度器
func (e *V4Executor) executePhaseOne(
	ctx context.Context,
	input *SchedulingExecutionInput,
	shiftID string,
	staffNames map[string]string,
	ruleSchedulers []IScheduler,
) (*ShiftSchedulingResult, error) {
	shift := findShiftByID(input.Shifts, shiftID)
	if shift == nil {
		return nil, fmt.Errorf("班次不存在: %s", shiftID)
	}

	resultSched := &ShiftSchedulingResult{
		ShiftID:  shiftID,
		Schedule: make(map[string][]string),
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("解析开始日期失败: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		return nil, fmt.Errorf("解析结束日期失败: %w", err)
	}

	// 检查当前班次是否有 required_together 依赖
	hasDependency := e.hasRequiredTogetherDep(shiftID, input)

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		requiredCount := 0
		if input.ShiftRequirements != nil {
			if dateReqs, ok := input.ShiftRequirements[shiftID]; ok {
				requiredCount = dateReqs[dateStr]
			}
		}
		if requiredCount == 0 {
			resultSched.Schedule[dateStr] = []string{}
			continue
		}

		// 读取阶段零写入的固定人员
		var finalSelected []string
		if input.CurrentDraft != nil && input.CurrentDraft.Shifts != nil {
			if shiftDraft, ok := input.CurrentDraft.Shifts[shiftID]; ok && shiftDraft != nil && shiftDraft.Days != nil {
				if dayDraft, ok := shiftDraft.Days[dateStr]; ok && dayDraft != nil {
					finalSelected = append(finalSelected, dayDraft.StaffIDs...)
				}
			}
		}

		remainingCount := requiredCount - len(finalSelected)
		if remainingCount <= 0 {
			resultSched.Schedule[dateStr] = finalSelected
			continue
		}

		// ── 运行 PersonnelRule 调度器（必须排班人员） ──
		for _, scheduler := range ruleSchedulers {
			if scheduler.Name() == "PersonnelRuleScheduler" {
				targetRule := e.resolveRuleForShift(input, shiftID)
				if scheduler.CanHandle(ctx, input, targetRule, shiftID, dateStr) {
					selected, _, err := scheduler.Schedule(ctx, input, shiftID, dateStr, remainingCount)
					if err != nil {
						e.logger.Error("Phase1 PersonnelRule error", "err", err)
					} else if len(selected) > 0 {
						existingSet := make(map[string]bool, len(finalSelected))
						for _, id := range finalSelected {
							existingSet[id] = true
						}
						for _, id := range selected {
							if !existingSet[id] {
								finalSelected = append(finalSelected, id)
								existingSet[id] = true
							}
						}
						remainingCount = requiredCount - len(finalSelected)
						e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)
					}
				}
				break
			}
		}

		if remainingCount <= 0 {
			resultSched.Schedule[dateStr] = finalSelected
			continue
		}

		// ── 依赖解析路径 vs 非依赖路径 ──
		if hasDependency {
			// 使用 DependencyResolver（内部调用 SingleShiftScheduler）
			resolveResult, err := e.dependencyResolver.Resolve(
				ctx, shiftID, dateStr, remainingCount,
				model.AssignmentSourceRule, input,
			)
			if err != nil {
				return nil, err
			}

			// 将依赖解析选出的人员追加
			existingSet := make(map[string]bool, len(finalSelected))
			for _, id := range finalSelected {
				existingSet[id] = true
			}
			for _, a := range resolveResult.Assignments {
				if !existingSet[a.StaffID] {
					finalSelected = append(finalSelected, a.StaffID)
					existingSet[a.StaffID] = true
				}
			}
			remainingCount = requiredCount - len(finalSelected)
			e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)

			// 收集依赖补排产生的冲突
			for _, c := range resolveResult.Conflicts {
				if e.OnConflict != nil {
					e.OnConflict(c)
				}
			}
		} else {
			// 非依赖班次：运行 Exclusive 等规则调度器
			for _, scheduler := range ruleSchedulers {
				if scheduler.Name() == "PersonnelRuleScheduler" {
					continue // 已在上面运行过
				}
				if scheduler.Name() == "RequiredTogetherScheduler" {
					continue // 已被 DependencyResolver 替代
				}
				if remainingCount <= 0 {
					break
				}
				targetRule := e.resolveRuleForShift(input, shiftID)
				if scheduler.CanHandle(ctx, input, targetRule, shiftID, dateStr) {
					selected, terminateChain, err := scheduler.Schedule(ctx, input, shiftID, dateStr, remainingCount)
					if err != nil {
						e.logger.Error("Phase1 scheduler error", "name", scheduler.Name(), "err", err)
						continue
					}
					if len(selected) > 0 {
						existingSet := make(map[string]bool, len(finalSelected))
						for _, id := range finalSelected {
							existingSet[id] = true
						}
						for _, id := range selected {
							if !existingSet[id] {
								finalSelected = append(finalSelected, id)
								existingSet[id] = true
							}
						}
						remainingCount = requiredCount - len(finalSelected)
						e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)
					}
					if terminateChain {
						break
					}
				}
			}

			// ── Phase1 就地兜底：如果所有规则调度器都未匹配，直接使用 SingleShiftScheduler 填充 ──
			// 避免小候选池班次（如本部穿刺 7人）被完全推迟到 Phase2，导致大班次在 Phase1 中
			// 先行消耗掉全部候选人造成饥饿。
			if remainingCount > 0 {
				fillResult, err := e.singleScheduler.Schedule(ctx, &SingleShiftInput{
					ShiftID:       shiftID,
					Date:          dateStr,
					RequiredCount: remainingCount,
					Source:        model.AssignmentSourceDefault,
				}, input)
				if err != nil {
					e.logger.Warn("Phase1 inline fill error", "shiftID", shiftID, "date", dateStr, "err", err)
				} else if fillResult != nil && len(fillResult.Assignments) > 0 {
					existingSet := make(map[string]bool, len(finalSelected))
					for _, id := range finalSelected {
						existingSet[id] = true
					}
					for _, a := range fillResult.Assignments {
						if !existingSet[a.StaffID] {
							finalSelected = append(finalSelected, a.StaffID)
							existingSet[a.StaffID] = true
						}
					}
					remainingCount = requiredCount - len(finalSelected)
					e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)
				}
			}
		}

		resultSched.Schedule[dateStr] = finalSelected
		e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)
	}

	return resultSched, nil
}

// executePhaseTwo 阶段二：使用 SingleShiftScheduler 兜底填充
// 标记为 Source=default, Priority=low（可被后续依赖补排置换）
func (e *V4Executor) executePhaseTwo(
	ctx context.Context,
	input *SchedulingExecutionInput,
	shiftID string,
	staffNames map[string]string,
	shiftResult *ShiftSchedulingResult,
) error {
	// 纯客体班次不参与兜底填充
	if input.RuleOrganization != nil && input.RuleOrganization.PureObjectShiftIDs != nil &&
		input.RuleOrganization.PureObjectShiftIDs[shiftID] {
		return nil
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return fmt.Errorf("解析开始日期失败: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		return fmt.Errorf("解析结束日期失败: %w", err)
	}

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		requiredCount := 0
		if input.ShiftRequirements != nil {
			if dateReqs, ok := input.ShiftRequirements[shiftID]; ok {
				requiredCount = dateReqs[dateStr]
			}
		}
		if requiredCount == 0 {
			continue
		}

		// 读取阶段一已排人数
		alreadyScheduledCount := 0
		if input.CurrentDraft != nil && input.CurrentDraft.Shifts != nil {
			if shiftDraft, ok := input.CurrentDraft.Shifts[shiftID]; ok && shiftDraft != nil && shiftDraft.Days != nil {
				if dayDraft, ok := shiftDraft.Days[dateStr]; ok && dayDraft != nil {
					alreadyScheduledCount = len(dayDraft.StaffIDs)
				}
			}
		}

		remainingCount := requiredCount - alreadyScheduledCount
		if remainingCount <= 0 {
			continue
		}

		// 使用 SingleShiftScheduler 兜底填充（Source=default, Priority=low）
		fillResult, err := e.singleScheduler.Schedule(ctx, &SingleShiftInput{
			ShiftID:       shiftID,
			Date:          dateStr,
			RequiredCount: remainingCount,
			Source:        model.AssignmentSourceDefault,
		}, input)

		if err != nil {
			e.logger.Warn("Phase2 SingleShiftScheduler fill error",
				"shiftID", shiftID, "date", dateStr, "err", err)
			continue
		}

		if fillResult == nil || len(fillResult.Assignments) == 0 {
			if remainingCount > 0 {
				e.logger.Warn("Phase2: under-staffed after fill attempt",
					"shiftID", shiftID, "date", dateStr,
					"required", requiredCount, "alreadyScheduled", alreadyScheduledCount,
					"stillNeeded", remainingCount)
			}
			continue
		}

		// 将填充结果追加到 CurrentDraft 和 ShiftResult
		fillStaffIDs := make([]string, 0, len(fillResult.Assignments))
		for _, a := range fillResult.Assignments {
			fillStaffIDs = append(fillStaffIDs, a.StaffID)
		}
		e.appendToCurrentDraft(input.CurrentDraft, shiftID, dateStr, fillStaffIDs, staffNames)

		if shiftResult != nil {
			if existing, ok := shiftResult.Schedule[dateStr]; ok {
				shiftResult.Schedule[dateStr] = append(existing, fillStaffIDs...)
			} else {
				shiftResult.Schedule[dateStr] = fillStaffIDs
			}
		}
	}

	return nil
}

// buildPhaseTwoOrder 构建阶段二的班次排序：按候选人稀缺度优先
// 候选人池越小的班次越先填充，避免大池班次抢占小池班次的全部候选人。
// 例如：本部穿刺（7候选人）应先于 CT/MRI报告上（45候选人）。
func (e *V4Executor) buildPhaseTwoOrder(shiftOrder []string, input *SchedulingExecutionInput) []string {
	type shiftScarcity struct {
		shiftID  string
		poolSize int // 候选人池大小
	}

	items := make([]shiftScarcity, 0, len(shiftOrder))
	for _, shiftID := range shiftOrder {
		poolSize := len(input.AllStaff) // 默认使用全体人员数量
		if members, ok := input.ShiftMembersMap[shiftID]; ok && len(members) > 0 {
			poolSize = len(members)
		}
		items = append(items, shiftScarcity{shiftID: shiftID, poolSize: poolSize})
	}

	// 按候选人池大小升序排序（稀缺的先排）；池大小相同时保持原顺序
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].poolSize < items[j].poolSize
	})

	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.shiftID
	}
	return result
}

// hasRequiredTogetherDep 检查班次是否有 required_together 依赖
// 仅当当前班次是依赖关系中的主体（DependentOnShiftID）时返回 true。
// 客体班次走正常调度路径，不进入 DependencyResolver。
func (e *V4Executor) hasRequiredTogetherDep(shiftID string, input *SchedulingExecutionInput) bool {
	if input.RuleOrganization == nil {
		return false
	}
	for _, dep := range input.RuleOrganization.ShiftDependencies {
		if dep.DependencyType == RuleTypeRequiredTogether && dep.DependentOnShiftID == shiftID {
			return true
		}
	}
	return false
}

// writeShiftResultToSchedule 将阶段一班次结果写入排班草稿
//
// 相比 updateCurrentDraft，此方法额外设置了 RequiredCount，并整体替换
// shiftDraft 条目（而非逐日追加），确保最终草稿数据完整。
// 由于 result.Schedule == input.CurrentDraft，此操作同时更新两者。
func (e *V4Executor) writeShiftResultToSchedule(
	schedule *model.ScheduleDraft,
	shiftID string,
	shiftResult *ShiftSchedulingResult,
	staffNames map[string]string,
	input *SchedulingExecutionInput,
) {
	shiftDraft := &model.ShiftDraft{
		ShiftID: shiftID,
		Days:    make(map[string]*model.DayShift),
	}
	for date, staffIDs := range shiftResult.Schedule {
		staffNameList := make([]string, 0, len(staffIDs))
		for _, sid := range staffIDs {
			if name, ok := staffNames[sid]; ok {
				staffNameList = append(staffNameList, name)
			} else {
				staffNameList = append(staffNameList, sid)
			}
		}
		requiredCount := 0
		if input.ShiftRequirements != nil && input.ShiftRequirements[shiftID] != nil {
			requiredCount = input.ShiftRequirements[shiftID][date]
		}
		shiftDraft.Days[date] = &model.DayShift{
			Staff:         staffNameList,
			StaffIDs:      staffIDs,
			RequiredCount: requiredCount,
			ActualCount:   len(staffIDs),
			IsFixed:       false,
		}
	}
	schedule.Shifts[shiftID] = shiftDraft
}

// appendToCurrentDraft 将新选出的人员追加到 CurrentDraft 中对应的班次-日期条目
//
// 用于阶段二：将 DefaultScheduler 选出的填充人员追加到阶段一已有的
// 规则占位人员之后，保持 Draft 数据的完整性和连续性。
func (e *V4Executor) appendToCurrentDraft(
	draft *model.ScheduleDraft,
	shiftID string,
	date string,
	newStaffIDs []string,
	staffNames map[string]string,
) {
	if draft == nil || len(newStaffIDs) == 0 {
		return
	}
	if draft.Shifts == nil {
		draft.Shifts = make(map[string]*model.ShiftDraft)
	}

	shiftDraft, exists := draft.Shifts[shiftID]
	if !exists || shiftDraft == nil {
		shiftDraft = &model.ShiftDraft{
			ShiftID: shiftID,
			Days:    make(map[string]*model.DayShift),
		}
		draft.Shifts[shiftID] = shiftDraft
	}
	if shiftDraft.Days == nil {
		shiftDraft.Days = make(map[string]*model.DayShift)
	}

	if dayDraft, ok := shiftDraft.Days[date]; ok && dayDraft != nil {
		// 追加新人员到已有条目（去重）
		existingSet := make(map[string]bool, len(dayDraft.StaffIDs))
		for _, sid := range dayDraft.StaffIDs {
			existingSet[sid] = true
		}
		combined := make([]string, len(dayDraft.StaffIDs))
		copy(combined, dayDraft.StaffIDs)
		for _, sid := range newStaffIDs {
			if !existingSet[sid] {
				combined = append(combined, sid)
				existingSet[sid] = true
			}
		}
		dayDraft.StaffIDs = combined
		// 重建名称列表
		nameList := make([]string, 0, len(combined))
		for _, sid := range combined {
			if name, ok := staffNames[sid]; ok {
				nameList = append(nameList, name)
			} else {
				nameList = append(nameList, sid)
			}
		}
		dayDraft.Staff = nameList
		dayDraft.ActualCount = len(combined)
	} else {
		// 阶段一未产生任何占位的日期（防御性处理）
		nameList := make([]string, 0, len(newStaffIDs))
		for _, sid := range newStaffIDs {
			if name, ok := staffNames[sid]; ok {
				nameList = append(nameList, name)
			} else {
				nameList = append(nameList, sid)
			}
		}
		shiftDraft.Days[date] = &model.DayShift{
			Staff:       nameList,
			StaffIDs:    newStaffIDs,
			ActualCount: len(newStaffIDs),
			IsFixed:     false,
		}
	}
}

// applyFixedAssignments 阶段零：将 FixedAssignments 中配置的固定人员写入 CurrentDraft
//
// 执行逻辑：
//  1. 遍历 shiftOrder 中的每个班次和排班周期内的每个日期
//  2. 在 input.FixedAssignments 中查找该班次-日期的固定人员列表
//  3. 若有固定人员，则以 IsFixed=true 写入 CurrentDraft
//
// 调用时机：ExecuteScheduling 的阶段零，早于任何调度器执行。
// 阶段一（规则排班）和阶段二（兜底填充）均依赖 CurrentDraft 感知已占位情况，
// 因此阶段零只需写入 CurrentDraft，无需操作 ShiftResults。
func (e *V4Executor) applyFixedAssignments(
	input *SchedulingExecutionInput,
	shiftOrder []string,
	staffNames map[string]string,
) {
	if len(input.FixedAssignments) == 0 {
		return
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		e.logger.Error("applyFixedAssignments: failed to parse start date", "err", err)
		return
	}
	endDate, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		e.logger.Error("applyFixedAssignments: failed to parse end date", "err", err)
		return
	}

	if input.CurrentDraft.Shifts == nil {
		input.CurrentDraft.Shifts = make(map[string]*model.ShiftDraft)
	}

	for _, shiftID := range shiftOrder {
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")

			// 收集该班次-日期的固定人员
			var fixedStaffIDs []string
			for _, fa := range input.FixedAssignments {
				if fa != nil && fa.ShiftID == shiftID && fa.Date == dateStr {
					fixedStaffIDs = append(fixedStaffIDs, fa.StaffIDs...)
				}
			}
			if len(fixedStaffIDs) == 0 {
				continue
			}

			// 构建名称列表
			staffNameList := make([]string, 0, len(fixedStaffIDs))
			for _, sid := range fixedStaffIDs {
				if name, ok := staffNames[sid]; ok {
					staffNameList = append(staffNameList, name)
				} else {
					staffNameList = append(staffNameList, sid)
				}
			}

			// 获取或创建 ShiftDraft
			shiftDraft, ok := input.CurrentDraft.Shifts[shiftID]
			if !ok || shiftDraft == nil {
				shiftDraft = &model.ShiftDraft{
					ShiftID: shiftID,
					Days:    make(map[string]*model.DayShift),
				}
				input.CurrentDraft.Shifts[shiftID] = shiftDraft
			}
			if shiftDraft.Days == nil {
				shiftDraft.Days = make(map[string]*model.DayShift)
			}

			// 计算该日期的需求人数（用于 RequiredCount 冗余字段）
			requiredCount := 0
			if input.ShiftRequirements != nil && input.ShiftRequirements[shiftID] != nil {
				requiredCount = input.ShiftRequirements[shiftID][dateStr]
			}

			// 写入 DayShift，标记为固定排班
			shiftDraft.Days[dateStr] = &model.DayShift{
				Staff:         staffNameList,
				StaffIDs:      fixedStaffIDs,
				ActualCount:   len(fixedStaffIDs),
				RequiredCount: requiredCount,
				IsFixed:       true,
			}
		}
	}
}

// ExecuteSingleShiftDate 针对特定的班次和日期进行排班（用于局部重排）
//
// 使用 DependencyResolver + SingleShiftScheduler 统一路径，
// 确保局部重排与全量排班走同一条依赖解析逻辑。
func (e *V4Executor) ExecuteSingleShiftDate(
	ctx context.Context,
	input *SchedulingExecutionInput,
	shiftID string,
	dateStr string,
) ([]string, error) {
	staffNames := make(map[string]string)
	for _, staff := range input.AllStaff {
		staffNames[staff.ID] = staff.Name
	}

	requiredCount := 0
	if input.ShiftRequirements != nil {
		if dateReqs, ok := input.ShiftRequirements[shiftID]; ok {
			requiredCount = dateReqs[dateStr]
		}
	}
	if requiredCount == 0 {
		return []string{}, nil
	}

	// 使用 DependencyResolver 统一处理（内部判断是否有依赖并自动补排）
	resolveResult, err := e.dependencyResolver.Resolve(
		ctx, shiftID, dateStr, requiredCount,
		model.AssignmentSourceRule, input,
	)
	if err != nil {
		return nil, err
	}

	finalSelected := make([]string, 0, len(resolveResult.Assignments))
	for _, a := range resolveResult.Assignments {
		finalSelected = append(finalSelected, a.StaffID)
	}

	// 更新 draft
	e.updateCurrentDraft(input.CurrentDraft, shiftID, dateStr, finalSelected, staffNames)

	return finalSelected, nil
}

// SchedulingExecutionInput 排班执行输入
type SchedulingExecutionInput struct {
	OrgID     string
	StartDate string
	EndDate   string
	AllStaff  []*model.Staff
	// ShiftMembersMap 班次分组成员映射（shiftID -> 该班次的员工列表）
	// 若不为空，则 executeShiftWithSchedulers 优先用此作为候选人基础池（替代 AllStaff）
	ShiftMembersMap   map[string][]*model.Staff
	Rules             []*model.Rule
	Shifts            []*model.Shift
	PersonalNeeds     []*model.PersonalNeed
	FixedAssignments  []*model.CtxFixedShiftAssignment // 固定排班配置
	CurrentDraft      *model.ScheduleDraft
	ShiftRequirements map[string]map[string]int // shiftID -> date -> count
	RuleOrganization  *RuleOrganization
	// UnavailableStaffMap 特定日期特定班次不可用的人员（用于局部重排时的定点排除）: date -> shiftID -> staffID -> bool
	UnavailableStaffMap map[string]map[string]map[string]bool
}

// SchedulingExecutionResult 排班执行结果
type SchedulingExecutionResult struct {
	Schedule     *model.ScheduleDraft
	ShiftResults map[string]*ShiftSchedulingResult
}

// ShiftSchedulingResult 单个班次排班结果（阶段一 + 阶段二合并后的完整结果）
type ShiftSchedulingResult struct {
	ShiftID  string
	Schedule map[string][]string // date -> staffIDs
}

// updateCurrentDraft 更新当前草稿以跟踪已排班状态
func (e *V4Executor) updateCurrentDraft(
	draft *model.ScheduleDraft,
	shiftID string,
	date string,
	staffIDs []string,
	staffNames map[string]string,
) {
	if draft == nil {
		return
	}

	// 确保 Shifts map 已初始化
	if draft.Shifts == nil {
		draft.Shifts = make(map[string]*model.ShiftDraft)
	}

	// 获取或创建班次草稿
	shiftDraft, exists := draft.Shifts[shiftID]
	if !exists {
		shiftDraft = &model.ShiftDraft{
			ShiftID: shiftID,
			Days:    make(map[string]*model.DayShift),
		}
		draft.Shifts[shiftID] = shiftDraft
	}

	// 确保 Days map 已初始化
	if shiftDraft.Days == nil {
		shiftDraft.Days = make(map[string]*model.DayShift)
	}

	// 构建员工名称列表
	staffNameList := make([]string, 0, len(staffIDs))
	for _, sid := range staffIDs {
		if name, ok := staffNames[sid]; ok {
			staffNameList = append(staffNameList, name)
		} else {
			staffNameList = append(staffNameList, sid)
		}
	}

	// 更新该日期的排班
	shiftDraft.Days[date] = &model.DayShift{
		Staff:       staffNameList,
		StaffIDs:    staffIDs,
		ActualCount: len(staffIDs),
		IsFixed:     false,
	}
}

// resolveRuleForShift 解析当前班次在规则组织中关联的任意一条规则，供 CanHandle 等使用。
// 从 RuleOrganization.ShiftGroups 中找到包含 shiftID 的组，再按 RuleID 在 input.Rules 中查规则。
func (e *V4Executor) resolveRuleForShift(input *SchedulingExecutionInput, shiftID string) *model.Rule {
	if input == nil || input.RuleOrganization == nil || len(input.Rules) == 0 {
		return nil
	}
	ruleByID := make(map[string]*model.Rule, len(input.Rules))
	for _, r := range input.Rules {
		if r != nil && r.ID != "" {
			ruleByID[r.ID] = r
		}
	}
	for _, g := range input.RuleOrganization.ShiftGroups {
		for _, id := range g.ShiftIDs {
			if id == shiftID && g.RuleID != "" {
				if r, ok := ruleByID[g.RuleID]; ok {
					return r
				}
				break
			}
		}
	}
	return nil
}

// findShiftByID 根据ID查找班次
func findShiftByID(shifts []*model.Shift, shiftID string) *model.Shift {
	for _, shift := range shifts {
		if shift.ID == shiftID {
			return shift
		}
	}
	return nil
}
