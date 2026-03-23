package executor

import (
	"context"
	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
	"time"
)

// DefaultScheduler 默认兜底调度器
// 负责没有被任何特殊调度器接管的剩余选人需求
type DefaultScheduler struct {
	logger     logging.ILogger
	ruleEngine *engine.RuleEngine
}

func NewDefaultScheduler(logger logging.ILogger, ruleEngine *engine.RuleEngine) *DefaultScheduler {
	return &DefaultScheduler{
		logger:     logger,
		ruleEngine: ruleEngine,
	}
}

func (s *DefaultScheduler) Name() string {
	return "DefaultScheduler"
}

func (s *DefaultScheduler) Priority() int {
	return 100 // 最低优先级，作为兜底
}

func (s *DefaultScheduler) CanHandle(ctx context.Context, input *SchedulingExecutionInput, rule *model.Rule, shiftID string, date string) bool {
	// 作为兜底，总是返回 true
	return true
}

func (s *DefaultScheduler) Schedule(
	ctx context.Context,
	input *SchedulingExecutionInput,
	shiftID string,
	dateStr string,
	requiredCount int,
) ([]string, bool, error) {
	if requiredCount <= 0 {
		return []string{}, false, nil
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, false, err
	}

	shift := s.findShiftByID(input.Shifts, shiftID)

	// 准备基础池
	baseStaff := input.AllStaff
	usesShiftMemberPool := false
	if members, ok := input.ShiftMembersMap[shiftID]; ok && len(members) > 0 {
		baseStaff = members
		usesShiftMemberPool = true
	}
	originalBaseSize := len(baseStaff)

	// 排除 UnavailableStaffMap 中标记的人员（用于局部重排）以及“禁止排班”的人员
	var filteredBaseStaff []*model.Staff
	for _, staff := range baseStaff {
		if isAlreadyScheduledOnShift(staff.ID, shiftID, dateStr, input) {
			continue
		}

		// 局部重排黑名单
		if input.UnavailableStaffMap != nil &&
			input.UnavailableStaffMap[dateStr] != nil &&
			input.UnavailableStaffMap[dateStr][shiftID] != nil &&
			input.UnavailableStaffMap[dateStr][shiftID][staff.ID] {
			continue
		}

		// 个人/规则层面的“禁止排班”
		if isForbiddenOnShift(staff.ID, shiftID, dateStr, input) {
			continue
		}
		filteredBaseStaff = append(filteredBaseStaff, staff)
	}
	baseStaff = filteredBaseStaff

	var fixedAssignments []*model.CtxFixedShiftAssignment
	for _, fa := range input.FixedAssignments {
		if fa.Date == dateStr {
			fixedAssignments = append(fixedAssignments, fa)
		}
	}

	schedulingInput := &engine.SchedulingInput{
		AllStaff:         baseStaff,
		AllRules:         input.Rules,
		PersonalNeeds:    input.PersonalNeeds,
		FixedAssignments: fixedAssignments,
		CurrentDraft:     input.CurrentDraft,
		ShiftID:          shiftID,
		Date:             date,
		RequiredCount:    requiredCount,
		AllShifts:        input.Shifts,
		TargetShift:      shift,
	}

	// 准备上下文，里面包含了基于硬约束过滤和软规则评分的候选人排序
	schedulingCtx, err := s.ruleEngine.PrepareSchedulingContext(ctx, schedulingInput)
	if err != nil {
		return nil, false, err
	}

	selected := make([]string, 0, requiredCount)
	candidates := schedulingCtx.EligibleCandidates

	if len(candidates) == 0 {
		// 汇总排除原因（只记数量，不记 UUID 列表）
		exclusionCount := make(map[string]int)
		for _, reason := range schedulingCtx.ExclusionReasons {
			exclusionCount[reason]++
		}
		for _, excluded := range schedulingCtx.ExcludedCandidates {
			for _, v := range excluded.ViolatedRules {
				key := "约束违反:" + v.RuleName + "(" + v.Message + ")"
				exclusionCount[key]++
			}
		}
		if len(schedulingInput.AllStaff) > 0 {
			// 基础池非空但经 RuleEngine 过滤后无候选人——升级为 WARN
			s.logger.Warn("DefaultScheduler: No eligible candidates despite non-empty pool",
				"shiftID", shiftID,
				"date", dateStr,
				"requiredCount", requiredCount,
				"basePoolSize", len(schedulingInput.AllStaff),
				"exclusionCount", exclusionCount)
		} else if originalBaseSize > 0 {
			// 原始池有人但被前置过滤（已排班/禁止/不可用）全部剔除——升级为 WARN
			s.logger.Warn("DefaultScheduler: No eligible candidates (all pre-filtered out)",
				"shiftID", shiftID,
				"date", dateStr,
				"requiredCount", requiredCount,
				"originalBaseSize", originalBaseSize,
				"usesShiftMemberPool", usesShiftMemberPool)
		} else {
			// 原始池本身为空（ShiftMembersMap 无配置 且 AllStaff 为空）
			s.logger.Warn("DefaultScheduler: No eligible candidates (pool is empty)",
				"shiftID", shiftID,
				"date", dateStr,
				"requiredCount", requiredCount,
				"usesShiftMemberPool", usesShiftMemberPool,
				"hint", "请检查该班次是否配置了排班成员")
		}
		return selected, false, nil
	}

	// 直接从按分数排好序的头部选取需要的数量（去重保护）
	selectedSet := make(map[string]bool, requiredCount)
	for i := 0; len(selected) < requiredCount && i < len(candidates); i++ {
		staffID := candidates[i].StaffID
		if !selectedSet[staffID] {
			selected = append(selected, staffID)
			selectedSet[staffID] = true
		}
	}

	if len(selected) < requiredCount {
		s.logger.Debug("DefaultScheduler: Insufficient candidates",
			"shiftID", shiftID,
			"date", dateStr,
			"required", requiredCount,
			"available", len(selected))
	}

	return selected, false, nil
}

func (s *DefaultScheduler) findShiftByID(shifts []*model.Shift, shiftID string) *model.Shift {
	for _, shift := range shifts {
		if shift.ID == shiftID {
			return shift
		}
	}
	return nil
}
