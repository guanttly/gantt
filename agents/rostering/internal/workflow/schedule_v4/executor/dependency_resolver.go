package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// DependencyResolver — 班次依赖协调器
//
// 纯编排逻辑，不直接操作候选人过滤/评分。
// 所有实际排人操作都委托给 SingleShiftScheduler。
//
// 核心流程：
//  1. 调用 SingleShiftScheduler 为 A 正排
//  2. 若 A 的规则要求依赖 B（required_together），检查 B 当前排班中是否有满足 A 的人
//  3. 若 B 中无合格人 → 计算 A∩B 成员池交集作为白名单
//  4. 检查 B 是否有空位；若已满，收集 B 中 Priority=low 人员作为 Replaceable
//  5. 调用 SingleShiftScheduler 为 B 补排（传入白名单 + Source=dependency + Replaceable）
//  6. B 补排成功 → 立即再次调用 SingleShiftScheduler 为 B 补位（若有人被置换导致欠编）
//  7. 回到 A，从 B 的更新结果中获取所需人员，完成 A 排班
//  8. 任一步骤 SingleShiftScheduler 返回 error → 立即终止，将 error 转为 ScheduleConflict
// ============================================================

// DependencyResolver 班次依赖协调器
type DependencyResolver struct {
	logger    logging.ILogger
	scheduler *SingleShiftScheduler
}

// NewDependencyResolver 创建依赖协调器
func NewDependencyResolver(
	logger logging.ILogger,
	scheduler *SingleShiftScheduler,
) *DependencyResolver {
	return &DependencyResolver{
		logger:    logger,
		scheduler: scheduler,
	}
}

// ResolveResult 依赖解析结果
type ResolveResult struct {
	// Assignments 当前班次（A）的最终人员分配
	Assignments []*model.StaffAssignment

	// DependencyFills 依赖补排记录：shiftID -> date -> 补入的人员
	// 调用方需要将这些人员写入对应班次的 draft
	DependencyFills map[string]map[string][]*model.StaffAssignment

	// Conflicts 依赖解析过程中产生的冲突（非致命）
	Conflicts []*model.ScheduleConflict
}

// Resolve 为班次 A 在日期 D 进行排班，自动解析班次依赖
//
// 参数:
//   - shiftID: 目标班次ID
//   - date: 目标日期 YYYY-MM-DD
//   - requiredCount: 需要排入的人数
//   - source: 分配来源标记
//   - execInput: 排班执行输入（含 CurrentDraft, Rules, Shifts 等全局上下文）
func (r *DependencyResolver) Resolve(
	ctx context.Context,
	shiftID string,
	date string,
	requiredCount int,
	source model.AssignmentSource,
	execInput *SchedulingExecutionInput,
) (*ResolveResult, error) {
	result := &ResolveResult{
		DependencyFills: make(map[string]map[string][]*model.StaffAssignment),
	}

	// 防御性检查：维护访问集，防止循环依赖
	visiting := make(map[string]bool)

	assignments, err := r.resolveWithDependency(ctx, shiftID, date, requiredCount, source, execInput, visiting, result)
	if err != nil {
		return nil, err
	}

	result.Assignments = assignments
	return result, nil
}

// resolveWithDependency 内部递归解析（带访问集保护）
func (r *DependencyResolver) resolveWithDependency(
	ctx context.Context,
	shiftID string,
	date string,
	requiredCount int,
	source model.AssignmentSource,
	execInput *SchedulingExecutionInput,
	visiting map[string]bool,
	result *ResolveResult,
) ([]*model.StaffAssignment, error) {
	// ── 循环依赖检测 ──
	visitKey := shiftID + ":" + date
	if visiting[visitKey] {
		shift := findShiftByID(execInput.Shifts, shiftID)
		shiftName := shiftID
		if shift != nil {
			shiftName = shift.Name
		}
		return nil, &SchedulingError{
			ShiftID:   shiftID,
			ShiftName: shiftName,
			Date:      date,
			Required:  requiredCount,
			Reason:    "检测到循环依赖，请检查规则配置",
		}
	}
	visiting[visitKey] = true
	defer func() { delete(visiting, visitKey) }()

	// ── 1. 查找当前班次是否有 required_together 依赖关系 ──
	dependencies := r.findRequiredTogetherDeps(shiftID, execInput)
	if len(dependencies) == 0 {
		// 无依赖：直接调用 SingleShiftScheduler 正排
		return r.scheduleDirect(ctx, shiftID, date, requiredCount, source, nil, nil, execInput)
	}

	// ── 2. 汇总所有依赖班次中已有的合格人员（OR 语义：任一依赖班次满足即可）──
	// 正确语义：主体班次(A)的人员必须来自某个依赖班次(B1 或 B2 ...)
	// 先把所有依赖班次中已排且在 A 成员池里的人并集起来
	allAvailable := make([]string, 0)
	availableSet := make(map[string]bool)
	for _, dep := range dependencies {
		relatedShiftID := dep.relatedShiftID
		targetDate := r.computeTargetDate(date, dep.offsetDays)
		r.logger.Info("DependencyResolver: 检查依赖班次",
			"currentShift", shiftID, "currentDate", date,
			"relatedShift", relatedShiftID, "relatedDate", targetDate)
		existing := r.findAvailableFromDependency(shiftID, relatedShiftID, targetDate, execInput)
		for _, id := range existing {
			if !availableSet[id] {
				allAvailable = append(allAvailable, id)
				availableSet[id] = true
			}
		}
	}

	if len(allAvailable) >= requiredCount {
		// Scenario B：依赖班次中已有足够的合格人员，直接复用
		r.logger.Info("DependencyResolver: Scenario B - 复用依赖班次已排人员",
			"currentShift", shiftID, "available", len(allAvailable), "required", requiredCount)
		return r.scheduleDirect(ctx, shiftID, date, requiredCount, source, allAvailable, nil, execInput)
	}

	// ── 3. 人员不足，尝试向各依赖班次补排 ──
	// 对每个依赖班次逐一尝试（交集为空则跳过，不报错），
	// 补排数量不超过该依赖班次自身的剩余名额，避免超标。
	for _, dep := range dependencies {
		if len(allAvailable) >= requiredCount {
			break // 已凑够，提前退出
		}

		relatedShiftID := dep.relatedShiftID
		targetDate := r.computeTargetDate(date, dep.offsetDays)
		relatedShift := findShiftByID(execInput.Shifts, relatedShiftID)
		relatedShiftName := relatedShiftID
		if relatedShift != nil {
			relatedShiftName = relatedShift.Name
		}

		// 计算 A∩B 成员池交集
		whitelist := r.computeIntersection(shiftID, relatedShiftID, execInput)
		if len(whitelist) == 0 {
			// 此依赖班次与主体无交集 → 跳过，继续下一个，不立即报错
			r.logger.Info("DependencyResolver: 依赖班次与主体无交集，跳过",
				"currentShift", shiftID, "relatedShift", relatedShiftID)
			continue
		}

		// 排除已计入 allAvailable 的人员（避免重复计数）
		filteredWhitelist := make([]string, 0, len(whitelist))
		for _, id := range whitelist {
			if !availableSet[id] {
				filteredWhitelist = append(filteredWhitelist, id)
			}
		}
		if len(filteredWhitelist) == 0 {
			continue
		}

		// ── 名额保护：仅在依赖班次有明确的需求配置且已满时跳过 ──
		depRequired := r.getShiftRequiredCount(relatedShiftID, targetDate, execInput)
		depScheduled := r.getShiftScheduledCount(relatedShiftID, targetDate, execInput)

		stillNeeded := requiredCount - len(allAvailable)
		toFill := stillNeeded

		if depRequired > 0 {
			// 有明确需求配置：不超过剩余名额
			depRemaining := depRequired - depScheduled
			if depRemaining <= 0 {
				r.logger.Info("DependencyResolver: 依赖班次已满员，跳过",
					"relatedShift", relatedShiftID, "required", depRequired, "scheduled", depScheduled)
				continue
			}
			if depRemaining < toFill {
				toFill = depRemaining
			}
		}
		// depRequired == 0 意味着依赖班次还未被处理或无独立需求配置，
		// 此时不做名额限制，让 SingleShiftScheduler 自行处理

		// 排除在主体班次(A)上被 forbidden 的人员：
		// 补排到依赖班次的人最终要回流到主体班次白名单，如果被 forbidden 则白名单名额浪费。
		// 只有当排除后剩余人数仍 >= toFill 时才执行排除，否则保留全部以尽可能满足需求。
		var nonForbiddenWhitelist []string
		for _, id := range filteredWhitelist {
			if !isForbiddenOnShift(id, shiftID, date, execInput) {
				nonForbiddenWhitelist = append(nonForbiddenWhitelist, id)
			}
		}
		effectiveWhitelist := filteredWhitelist
		if len(nonForbiddenWhitelist) >= toFill {
			effectiveWhitelist = nonForbiddenWhitelist
		}

		replaceable := r.findReplaceableInShift(relatedShiftID, targetDate, effectiveWhitelist, execInput)

		r.logger.Info("DependencyResolver: 向依赖班次补排",
			"currentShift", shiftID, "relatedShift", relatedShiftID,
			"toFill", toFill, "depRequired", depRequired, "depScheduled", depScheduled)

		depResult, err := r.scheduler.Schedule(ctx, &SingleShiftInput{
			ShiftID:       relatedShiftID,
			Date:          targetDate,
			RequiredCount: toFill,
			Source:        model.AssignmentSourceDependency,
			Whitelist:     effectiveWhitelist,
			Replaceable:   replaceable,
		}, execInput)

		if err != nil {
			// 此依赖班次补排失败 → 记录日志但继续尝试其他依赖，不立即报错
			r.logger.Warn("DependencyResolver: 依赖班次补排失败，继续尝试其他依赖",
				"relatedShift", relatedShiftName, "err", err)
			continue
		}

		// 写入依赖班次 draft
		r.applyDependencyFill(execInput.CurrentDraft, relatedShiftID, targetDate, depResult, execInput)

		if result.DependencyFills[relatedShiftID] == nil {
			result.DependencyFills[relatedShiftID] = make(map[string][]*model.StaffAssignment)
		}
		result.DependencyFills[relatedShiftID][targetDate] = depResult.Assignments

		r.logger.Info("DependencyResolver: 依赖班次补排成功",
			"relatedShift", relatedShiftID, "filledCount", len(depResult.Assignments))

		// 如果有人被置换，补位
		if len(depResult.Replaced) > 0 {
			r.handleBackfill(ctx, relatedShiftID, targetDate, depResult.Replaced, execInput, result)
		}

		// 重新收集该依赖班次中的合格人员并并入总池
		newly := r.findAvailableFromDependency(shiftID, relatedShiftID, targetDate, execInput)
		for _, id := range newly {
			if !availableSet[id] {
				allAvailable = append(allAvailable, id)
				availableSet[id] = true
			}
		}
	}

	// ── 4. 检查并集是否满足需求 ──
	if len(allAvailable) < requiredCount {
		shift := findShiftByID(execInput.Shifts, shiftID)
		shiftName := shiftID
		if shift != nil {
			shiftName = shift.Name
		}
		return nil, &SchedulingError{
			ShiftID:   shiftID,
			ShiftName: shiftName,
			Date:      date,
			Required:  requiredCount,
			Available: len(allAvailable),
			Deficit:   requiredCount - len(allAvailable),
			Reason: fmt.Sprintf("在所有依赖班次中合计只能找到 %d 人（需要 %d 人），请检查各依赖班次的人员配置及成员池交集",
				len(allAvailable), requiredCount),
			Detail: r.buildDiagnosticDetail(shiftID, shiftName, date, dependencies, execInput),
		}
	}

	// ── 5. 从合并后的可用池中为主体班次排班 ──
	return r.scheduleDirect(ctx, shiftID, date, requiredCount, source, allAvailable, nil, execInput)
}

// scheduleDirect 直接调用 SingleShiftScheduler 排班
func (r *DependencyResolver) scheduleDirect(
	ctx context.Context,
	shiftID string,
	date string,
	requiredCount int,
	source model.AssignmentSource,
	whitelist []string,
	blacklist []string,
	execInput *SchedulingExecutionInput,
) ([]*model.StaffAssignment, error) {
	result, err := r.scheduler.Schedule(ctx, &SingleShiftInput{
		ShiftID:       shiftID,
		Date:          date,
		RequiredCount: requiredCount,
		Source:        source,
		Whitelist:     whitelist,
		Blacklist:     blacklist,
	}, execInput)
	if err != nil {
		return nil, err
	}
	return result.Assignments, nil
}

// ============================================================
// 依赖关系查找与分析
// ============================================================

// depInfo 依赖关系信息
type depInfo struct {
	relatedShiftID string // 关联班次ID（被依赖的）
	offsetDays     int    // 日期偏移
	depType        string // 依赖类型
	ruleID         string // 关联规则ID
}

// findRequiredTogetherDeps 查找班次的 required_together 依赖关系
// 仅当当前班次是主体（DependentOnShiftID）时，返回其依赖的客体班次列表。
// 客体班次不会被返回任何依赖，确保它走正常调度路径。
func (r *DependencyResolver) findRequiredTogetherDeps(
	shiftID string,
	execInput *SchedulingExecutionInput,
) []depInfo {
	if execInput.RuleOrganization == nil {
		return nil
	}

	var deps []depInfo

	// 直接使用 ShiftDependencies（有方向信息），不再使用 ShiftGroups（无方向）
	// DependentOnShiftID = 主体班次（排了它就必须排客体）
	// DependentShiftID   = 客体班次（被依赖的目标池）
	for _, dep := range execInput.RuleOrganization.ShiftDependencies {
		if dep.DependencyType != RuleTypeRequiredTogether {
			continue
		}
		// 只有当前班次是主体时，才返回其依赖的客体班次
		if dep.DependentOnShiftID == shiftID {
			deps = append(deps, depInfo{
				relatedShiftID: dep.DependentShiftID,
				offsetDays:     dep.TimeOffsetDays,
				depType:        RuleTypeRequiredTogether,
				ruleID:         dep.RuleID,
			})
		}
	}

	return deps
}

// computeTargetDate 计算偏移后的目标日期
func (r *DependencyResolver) computeTargetDate(date string, offsetDays int) string {
	if offsetDays == 0 {
		return date
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, offsetDays).Format("2006-01-02")
}

// findAvailableFromDependency 查找依赖班次中属于当前班次成员池的人员
func (r *DependencyResolver) findAvailableFromDependency(
	currentShiftID string,
	relatedShiftID string,
	relatedDate string,
	execInput *SchedulingExecutionInput,
) []string {
	if execInput.CurrentDraft == nil || execInput.CurrentDraft.Shifts == nil {
		return nil
	}

	shiftDraft, ok := execInput.CurrentDraft.Shifts[relatedShiftID]
	if !ok || shiftDraft == nil || shiftDraft.Days == nil {
		return nil
	}
	dayShift, ok := shiftDraft.Days[relatedDate]
	if !ok || dayShift == nil {
		return nil
	}

	// 获取当前班次的成员池
	currentMembers, hasPool := execInput.ShiftMembersMap[currentShiftID]
	if !hasPool || len(currentMembers) == 0 {
		// 无成员池限制，依赖班次的所有人都可用
		return dayShift.StaffIDs
	}

	// 取交集：依赖班次已排人员 ∩ 当前班次成员池
	memberSet := make(map[string]bool, len(currentMembers))
	for _, m := range currentMembers {
		memberSet[m.ID] = true
	}

	var available []string
	for _, staffID := range dayShift.StaffIDs {
		if memberSet[staffID] {
			available = append(available, staffID)
		}
	}
	return available
}

// computeIntersection 计算两个班次成员池的交集（返回员工ID列表）
func (r *DependencyResolver) computeIntersection(
	shiftA string,
	shiftB string,
	execInput *SchedulingExecutionInput,
) []string {
	membersA, hasA := execInput.ShiftMembersMap[shiftA]
	membersB, hasB := execInput.ShiftMembersMap[shiftB]

	if !hasA || len(membersA) == 0 {
		// A 无成员池限制 → 交集 = B 的成员
		if !hasB || len(membersB) == 0 {
			// 双方都无限制 → 返回全员ID
			ids := make([]string, 0, len(execInput.AllStaff))
			for _, s := range execInput.AllStaff {
				ids = append(ids, s.ID)
			}
			return ids
		}
		ids := make([]string, 0, len(membersB))
		for _, s := range membersB {
			ids = append(ids, s.ID)
		}
		return ids
	}

	if !hasB || len(membersB) == 0 {
		// B 无成员池限制 → 交集 = A 的成员
		ids := make([]string, 0, len(membersA))
		for _, s := range membersA {
			ids = append(ids, s.ID)
		}
		return ids
	}

	// 双方都有成员池 → 求交集
	setB := make(map[string]bool, len(membersB))
	for _, s := range membersB {
		setB[s.ID] = true
	}

	var intersection []string
	for _, s := range membersA {
		if setB[s.ID] {
			intersection = append(intersection, s.ID)
		}
	}
	return intersection
}

// findReplaceableInShift 查找依赖班次中可被置换的低优先级人员
func (r *DependencyResolver) findReplaceableInShift(
	relatedShiftID string,
	relatedDate string,
	whitelist []string,
	execInput *SchedulingExecutionInput,
) []string {
	if execInput.CurrentDraft == nil || execInput.CurrentDraft.Shifts == nil {
		return nil
	}

	shiftDraft, ok := execInput.CurrentDraft.Shifts[relatedShiftID]
	if !ok || shiftDraft == nil || shiftDraft.Days == nil {
		return nil
	}
	dayShift, ok := shiftDraft.Days[relatedDate]
	if !ok || dayShift == nil {
		return nil
	}

	// 检查是否已满
	requiredCount := 0
	if execInput.ShiftRequirements != nil && execInput.ShiftRequirements[relatedShiftID] != nil {
		requiredCount = execInput.ShiftRequirements[relatedShiftID][relatedDate]
	}
	if len(dayShift.StaffIDs) < requiredCount {
		// 还有空位，不需要置换
		return nil
	}

	// 收集可被置换的人员（排除白名单中的人，因为他们即将被补入）
	whiteSet := make(map[string]bool, len(whitelist))
	for _, id := range whitelist {
		whiteSet[id] = true
	}

	// 优先从 Assignments 中识别低优先级人员
	if len(dayShift.Assignments) > 0 {
		return dayShift.GetReplaceableStaffIDs()
	}

	// 兼容模式：无 Assignments 时，非固定排班的人员都视为低优先级（可被置换）
	if dayShift.IsFixed {
		return nil
	}
	var replaceable []string
	for _, staffID := range dayShift.StaffIDs {
		if !whiteSet[staffID] {
			replaceable = append(replaceable, staffID)
		}
	}
	return replaceable
}

// handleBackfill 为被置换人员后的空缺进行补位
func (r *DependencyResolver) handleBackfill(
	ctx context.Context,
	shiftID string,
	date string,
	replaced []*model.StaffAssignment,
	execInput *SchedulingExecutionInput,
	result *ResolveResult,
) {
	// 检查当前是否欠编
	requiredCount := 0
	if execInput.ShiftRequirements != nil && execInput.ShiftRequirements[shiftID] != nil {
		requiredCount = execInput.ShiftRequirements[shiftID][date]
	}

	currentCount := 0
	if execInput.CurrentDraft != nil && execInput.CurrentDraft.Shifts != nil {
		if shiftDraft, ok := execInput.CurrentDraft.Shifts[shiftID]; ok && shiftDraft != nil {
			if dayShift, ok := shiftDraft.Days[date]; ok && dayShift != nil {
				currentCount = len(dayShift.StaffIDs)
			}
		}
	}

	deficit := requiredCount - currentCount
	if deficit <= 0 {
		return
	}

	// 被置换人员加入黑名单（避免又选回来）
	var blacklist []string
	for _, a := range replaced {
		blacklist = append(blacklist, a.StaffID)
	}

	r.logger.Info("DependencyResolver: 为被置换后的空缺补位",
		"shiftID", shiftID, "date", date,
		"deficit", deficit, "blacklist", blacklist)

	// 调用 SingleShiftScheduler 补位（Source=default, Priority=low）
	backfillResult, err := r.scheduler.Schedule(ctx, &SingleShiftInput{
		ShiftID:       shiftID,
		Date:          date,
		RequiredCount: deficit,
		Source:        model.AssignmentSourceDefault,
		Blacklist:     blacklist,
	}, execInput)

	if err != nil {
		// 补位失败不终止整体流程，记录 warning
		shift := findShiftByID(execInput.Shifts, shiftID)
		shiftName := shiftID
		if shift != nil {
			shiftName = shift.Name
		}
		result.Conflicts = append(result.Conflicts, &model.ScheduleConflict{
			Date:         date,
			Shift:        shiftName,
			Issue:        fmt.Sprintf("依赖补排置换后补位失败: %v", err),
			Severity:     "warning",
			ConflictType: model.ConflictTypeUnderstaffed,
		})
		r.logger.Warn("DependencyResolver: 补位失败（非致命）",
			"shiftID", shiftID, "date", date, "error", err)
		return
	}

	// 将补位人员写入 draft
	if backfillResult != nil && len(backfillResult.Assignments) > 0 {
		r.applyDependencyFill(execInput.CurrentDraft, shiftID, date, backfillResult, execInput)
	}
}

// applyDependencyFill 将补排/补位结果写入 draft
func (r *DependencyResolver) applyDependencyFill(
	draft *model.ScheduleDraft,
	shiftID string,
	date string,
	fillResult *SingleShiftResult,
	execInput *SchedulingExecutionInput,
) {
	if draft == nil || fillResult == nil || len(fillResult.Assignments) == 0 {
		return
	}
	if draft.Shifts == nil {
		draft.Shifts = make(map[string]*model.ShiftDraft)
	}

	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok || shiftDraft == nil {
		shiftDraft = &model.ShiftDraft{
			ShiftID: shiftID,
			Days:    make(map[string]*model.DayShift),
		}
		draft.Shifts[shiftID] = shiftDraft
	}
	if shiftDraft.Days == nil {
		shiftDraft.Days = make(map[string]*model.DayShift)
	}

	dayShift, ok := shiftDraft.Days[date]
	if !ok || dayShift == nil {
		requiredCount := 0
		if execInput.ShiftRequirements != nil && execInput.ShiftRequirements[shiftID] != nil {
			requiredCount = execInput.ShiftRequirements[shiftID][date]
		}
		dayShift = &model.DayShift{
			RequiredCount: requiredCount,
		}
		shiftDraft.Days[date] = dayShift
	}

	// 追加分配记录
	for _, a := range fillResult.Assignments {
		dayShift.AddAssignment(a)
	}
}

// buildAssignments 从 staffID 列表构建 StaffAssignment
func (r *DependencyResolver) buildAssignments(
	staffIDs []string,
	source model.AssignmentSource,
	execInput *SchedulingExecutionInput,
) []*model.StaffAssignment {
	staffNames := make(map[string]string, len(execInput.AllStaff))
	for _, s := range execInput.AllStaff {
		staffNames[s.ID] = s.Name
	}

	priority := model.SourceToPriority(source)
	assignments := make([]*model.StaffAssignment, 0, len(staffIDs))
	for _, id := range staffIDs {
		name := id
		if n, ok := staffNames[id]; ok {
			name = n
		}
		assignments = append(assignments, &model.StaffAssignment{
			StaffID:   id,
			StaffName: name,
			Source:    source,
			Priority:  priority,
		})
	}
	return assignments
}

// ============================================================
// 工具函数
// ============================================================

// getShiftRequiredCount 获取指定班次在指定日期的需求人数
func (r *DependencyResolver) getShiftRequiredCount(shiftID string, date string, execInput *SchedulingExecutionInput) int {
	if execInput.ShiftRequirements != nil {
		if req, ok := execInput.ShiftRequirements[shiftID]; ok {
			if count, ok := req[date]; ok {
				return count
			}
		}
	}
	return 0
}

// getShiftScheduledCount 获取指定班次在指定日期已排人数（来自 CurrentDraft）
func (r *DependencyResolver) getShiftScheduledCount(shiftID string, date string, execInput *SchedulingExecutionInput) int {
	if execInput.CurrentDraft == nil || execInput.CurrentDraft.Shifts == nil {
		return 0
	}
	sd, ok := execInput.CurrentDraft.Shifts[shiftID]
	if !ok || sd == nil || sd.Days == nil {
		return 0
	}
	dd, ok := sd.Days[date]
	if !ok || dd == nil {
		return 0
	}
	return len(dd.StaffIDs)
}

// uniqueStrings 去重字符串切片
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool, len(input))
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// buildDiagnosticDetail 生成排班冲突诊断快照
//
// 快照内容：
//   - 主体班次 (A)：当日已排人员 + 候选人员池
//   - 所有客体/依赖班次 (B1, B2 ...)：目标日期已排人员 + 候选人员池 + 与A的交集
func (r *DependencyResolver) buildDiagnosticDetail(
	shiftAID string, shiftAName string, dateA string,
	deps []depInfo,
	execInput *SchedulingExecutionInput,
) string {
	// 构建 staffID -> Name 查找表
	staffNames := make(map[string]string, len(execInput.AllStaff))
	for _, s := range execInput.AllStaff {
		if s != nil {
			staffNames[s.ID] = s.Name
		}
	}
	resolveNames := func(ids []string) []string {
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := staffNames[id]; ok && name != "" {
				names = append(names, name)
			} else {
				names = append(names, id)
			}
		}
		return names
	}

	// 主体班次 A 的候选人员池
	var poolANames []string
	if members, ok := execInput.ShiftMembersMap[shiftAID]; ok && len(members) > 0 {
		for _, m := range members {
			if m != nil {
				poolANames = append(poolANames, m.Name)
			}
		}
	}

	// 主体班次 A 当日已排人员
	var scheduledANames []string
	if execInput.CurrentDraft != nil && execInput.CurrentDraft.Shifts != nil {
		if sd, ok := execInput.CurrentDraft.Shifts[shiftAID]; ok && sd != nil && sd.Days != nil {
			if dd, ok := sd.Days[dateA]; ok && dd != nil {
				scheduledANames = resolveNames(dd.StaffIDs)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n主体班次 [%s]（%s）\n", shiftAName, dateA))
	sb.WriteString(fmt.Sprintf("  · 已排人员（%d人）: %s\n", len(scheduledANames), formatDiagNameList(scheduledANames)))
	sb.WriteString(fmt.Sprintf("  · 候选人员池（%d人）: %s\n", len(poolANames), formatDiagNameList(poolANames)))

	// 逐个展示所有依赖班次
	for _, dep := range deps {
		relatedShiftID := dep.relatedShiftID
		targetDate := r.computeTargetDate(dateA, dep.offsetDays)
		relatedShift := findShiftByID(execInput.Shifts, relatedShiftID)
		relatedShiftName := relatedShiftID
		if relatedShift != nil {
			relatedShiftName = relatedShift.Name
		}

		// 客体班次的候选人员池
		var poolBNames []string
		if members, ok := execInput.ShiftMembersMap[relatedShiftID]; ok && len(members) > 0 {
			for _, m := range members {
				if m != nil {
					poolBNames = append(poolBNames, m.Name)
				}
			}
		}

		// 客体班次目标日期已排人员
		var scheduledBNames []string
		if execInput.CurrentDraft != nil && execInput.CurrentDraft.Shifts != nil {
			if sd, ok := execInput.CurrentDraft.Shifts[relatedShiftID]; ok && sd != nil && sd.Days != nil {
				if dd, ok := sd.Days[targetDate]; ok && dd != nil {
					scheduledBNames = resolveNames(dd.StaffIDs)
				}
			}
		}

		// 计算与 A 的交集
		intersectionIDs := r.computeIntersection(shiftAID, relatedShiftID, execInput)
		intersectionNames := resolveNames(intersectionIDs)

		sb.WriteString(fmt.Sprintf("依赖班次 [%s]（%s）\n", relatedShiftName, targetDate))
		sb.WriteString(fmt.Sprintf("  · 已排人员（%d人）: %s\n", len(scheduledBNames), formatDiagNameList(scheduledBNames)))
		sb.WriteString(fmt.Sprintf("  · 候选人员池（%d人）: %s\n", len(poolBNames), formatDiagNameList(poolBNames)))
		if len(intersectionIDs) == 0 {
			sb.WriteString("  · 与主体交集候选: （空——两班次成员池无共同人员）\n")
		} else {
			sb.WriteString(fmt.Sprintf("  · 与主体交集候选（%d人）: %s\n", len(intersectionNames), formatDiagNameList(intersectionNames)))
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// formatDiagNameList 将名称列表格式化为逗号分隔字符串，空时返回 "（空）"
func formatDiagNameList(names []string) string {
	if len(names) == 0 {
		return "（空）"
	}
	return strings.Join(names, "、")
}
