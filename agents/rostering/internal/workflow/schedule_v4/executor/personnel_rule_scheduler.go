package executor

import (
	"context"
	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
	"time"
)

// PersonnelRuleScheduler 人员与临时规则调度器
// 处理由于人员规则（如 required_day）或临时需求（如 forbidden_day/请假）带来的锁定逻辑
type PersonnelRuleScheduler struct {
	logger     logging.ILogger
	ruleEngine *engine.RuleEngine
}

func NewPersonnelRuleScheduler(logger logging.ILogger, ruleEngine *engine.RuleEngine) *PersonnelRuleScheduler {
	return &PersonnelRuleScheduler{
		logger:     logger,
		ruleEngine: ruleEngine,
	}
}

func (s *PersonnelRuleScheduler) Name() string {
	return "PersonnelRuleScheduler"
}

func (s *PersonnelRuleScheduler) Priority() int {
	return 10 // 最高优先级
}

func (s *PersonnelRuleScheduler) CanHandle(ctx context.Context, input *SchedulingExecutionInput, rule *model.Rule, shiftID string, date string) bool {
	// 这是一个全局性质的调度器，只要输入中有明确的个人诉求或针对特定人的日期规则，它就接管部分候选人的锁定/排除工作。
	// 但它的运行模式比较特殊，它不是为了填满班次，而是为了“预先安排满足必须排班条件的人员”。
	// 实际在执行时，我们可以让它总是“尝试执行”（返回true），如果没有需要锁定的，它会很快返回空。
	return true
}

func (s *PersonnelRuleScheduler) Schedule(
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

	selected := make([]string, 0)

	// 1. 处理 required_day (必须排班/指定日期) 规则或极高优先级的 PersonalNeeds(要求排某个班)
	// 在此处，我们可以遍历输入中的规则或人员需求配置，主动寻找“要求今天排该班”的人
	for _, staff := range input.AllStaff {
		if isAlreadyScheduledOnShift(staff.ID, shiftID, dateStr, input) {
			continue
		}

		// 排除 UnavailableStaffMap 中标记的人员（用于局部重排）
		if input.UnavailableStaffMap != nil &&
			input.UnavailableStaffMap[dateStr] != nil &&
			input.UnavailableStaffMap[dateStr][shiftID] != nil &&
			input.UnavailableStaffMap[dateStr][shiftID][staff.ID] {
			continue // 该人员被黑名单排除
		}
		if s.isRequiredOnShift(staff, shiftID, date, input) {
			// 在添加到 selected 前，最好通过规则引擎简单鉴权一下，避免硬性违规
			if s.isEligibleForShift(ctx, staff, shiftID, date, input) {
				selected = append(selected, staff.ID)
			} else {
				s.logger.Warn("Staff requested shift but violates hard constraints", "staffID", staff.ID, "shiftID", shiftID, "date", dateStr)
			}
			if len(selected) >= requiredCount {
				break
			}
		}
	}

	// 2. 对于 forbidden_day（禁止排班）或者请假，不需要在这里选人，
	// 因为后续构建 Context 和 selectStaff 时，规则引擎的 ConstraintChecker
	// 会自动将违反 forbidden_day 或 PersonalNeeds(请假) 的人过滤掉（IsEligible = false）。
	// 所以这个调度器的主要职责是“抢占式分配被强制要求排班的人员”。

	return selected, false, nil
}

// isRequiredOnShift 判断指定员工是否被强制要求在某天排某班
func (s *PersonnelRuleScheduler) isRequiredOnShift(staff *model.Staff, shiftID string, date time.Time, input *SchedulingExecutionInput) bool {
	dateStr := date.Format("2006-01-02")

	// 1. 检查有没有该员工对于此班次此时的 PersonalNeed (意愿排班) 并且优先级极高
	for _, pn := range input.PersonalNeeds {
		if pn.StaffID == staff.ID && pn.RequestType == "must" {
			if pn.TargetShiftID == shiftID {
				// 检查日期是否匹配 (如果没有限制 TargetDates，或是包含今天)
				if len(pn.TargetDates) == 0 {
					return true
				}
				for _, d := range pn.TargetDates {
					if d == dateStr {
						return true
					}
				}
			}
		}
	}

	// 2. 检查是否有 RuleType = "required_day" 且绑定了某人某班的全局规则
	for _, rule := range input.Rules {
		if rule.RuleType == "required_day" && rule.IsActive {
			// 如果规则绑定了这个人，并且绑定了这个班，并且应用日期包含今天
			holdsForStaff := false
			holdsForShift := false
			for _, assoc := range rule.Associations {
				if assoc.AssociationType == "employee" && assoc.AssociationID == staff.ID {
					holdsForStaff = true
				}
				if assoc.AssociationType == "shift" && assoc.AssociationID == shiftID {
					holdsForShift = true
				}
			}

			// FIXME: 简单日期匹配，实际规则中的日期可能在 RuleData 里
			// 这里假设如果有这个规则切命中了员工和班次，我们就尝试强排
			// 在V4中，"required_day" 或特定日期排定实际上会被提炼为 FixedAssignments 或在预处理中处理
			if holdsForStaff && holdsForShift {
				return true
			}
		}
	}

	return false
}

// isEligibleForShift 检查员工是否硬性满足排班条件（防止把正在请假的人强排进去）
func (s *PersonnelRuleScheduler) isEligibleForShift(ctx context.Context, staff *model.Staff, shiftID string, date time.Time, input *SchedulingExecutionInput) bool {
	// 借用 RuleEngine 的能力，只丢这一个人进去跑一下准备。
	// 或者简单粗暴返回 true，依赖后面的兜底或由于是强需求所以突破一些软约束。
	// 这里为了严谨，我们先通过简单的状态位判断，如果没有休假就行。
	dateStr := date.Format("2006-01-02")
	for _, pn := range input.PersonalNeeds {
		if pn.StaffID == staff.ID && pn.RequestType == "avoid" {
			// 如果是强烈要求回避该日期的请假等情况
			if len(pn.TargetDates) == 0 {
				return false
			}
			for _, d := range pn.TargetDates {
				if d == dateStr {
					return false
				}
			}
		}
	}

	return true
}
