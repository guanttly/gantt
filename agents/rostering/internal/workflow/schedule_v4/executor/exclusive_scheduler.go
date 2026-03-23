package executor

import (
	"context"
	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
	"time"
)

// ExclusiveScheduler 排他规则调度器
// 处理由 exclusive 规则关联的班次，由于底层 ConstraintChecker 已经能够阻断互斥情况，
// 这里主要是确保“关联的排他班次中最难排的（候选人最少、分数最低）先排”，然后依赖底层规则防止另个班次选同样的人。
type ExclusiveScheduler struct {
	logger     logging.ILogger
	ruleEngine *engine.RuleEngine
}

func NewExclusiveScheduler(
	logger logging.ILogger,
	ruleEngine *engine.RuleEngine,
) *ExclusiveScheduler {
	return &ExclusiveScheduler{
		logger:     logger,
		ruleEngine: ruleEngine,
	}
}

func (s *ExclusiveScheduler) Name() string {
	return "ExclusiveScheduler"
}

func (s *ExclusiveScheduler) Priority() int {
	return 30 // 优先级排在明确的人员规则和“必须同时”之后
}

func (s *ExclusiveScheduler) CanHandle(
	ctx context.Context,
	input *SchedulingExecutionInput,
	rule *model.Rule,
	shiftID string,
	date string,
) bool {
	// 直接从规则列表中查找 exclusive 规则，检查其关联班次是否包含当前班次
	// 不再依赖 ShiftGroup（排他规则已从依赖分析管道中剥离）
	for _, r := range input.Rules {
		if r.RuleType == RuleTypeExclusive {
			for _, assoc := range r.Associations {
				if assoc.AssociationType == model.AssociationTypeShift && assoc.AssociationID == shiftID {
					return true
				}
			}
		}
	}
	return false
}

func (s *ExclusiveScheduler) Schedule(
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

	// Exclusive 排班策略：正常构建上下文，让底层 ConstraintChecker 将已被排到排他班次的人滤掉
	// （底层 checkExclusive 通过 rule.Associations 直接判断互斥关系，不依赖 ShiftGroup）

	shift := s.findShiftByID(input.Shifts, shiftID)

	// 准备基础池
	baseStaff := input.AllStaff
	if members, ok := input.ShiftMembersMap[shiftID]; ok && len(members) > 0 {
		baseStaff = members
	}

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

	// 规则引擎底层的 checkExclusive 会挡住冲突人员
	schedulingCtx, err := s.ruleEngine.PrepareSchedulingContext(ctx, schedulingInput)
	if err != nil {
		return nil, false, err
	}

	// 选人
	selected := make([]string, 0, requiredCount)
	for i := 0; i < requiredCount && i < len(schedulingCtx.EligibleCandidates); i++ {
		selected = append(selected, schedulingCtx.EligibleCandidates[i].StaffID)
	}

	return selected, false, nil
}

func (s *ExclusiveScheduler) findShiftByID(shifts []*model.Shift, shiftID string) *model.Shift {
	for _, shift := range shifts {
		if shift.ID == shiftID {
			return shift
		}
	}
	return nil
}
