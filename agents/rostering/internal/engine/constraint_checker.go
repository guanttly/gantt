package engine

import (
	"fmt"
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
	"time"
)

// ConstraintChecker 约束检查器（替代 LLM-3）
type ConstraintChecker struct {
	logger logging.ILogger
}

// NewConstraintChecker 创建约束检查器
func NewConstraintChecker(logger logging.ILogger) *ConstraintChecker {
	return &ConstraintChecker{logger: logger}
}

// CheckAll 检查所有候选人的所有约束
func (c *ConstraintChecker) CheckAll(
	candidates []*model.Staff,
	rules *MatchedRules,
	currentDraft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
	allStaff ...[]*model.Staff,
) *ConstraintCheckResult {
	// 提取可选的 allStaff 参数（用于 ApplyScopes 判断）
	var staffList []*model.Staff
	if len(allStaff) > 0 {
		staffList = allStaff[0]
	}

	result := &ConstraintCheckResult{
		EligibleCandidates: make([]*CandidateStatus, 0),
		ExcludedCandidates: make([]*CandidateStatus, 0),
		Details:            make([]*ConstraintDetail, 0),
	}

	for _, staff := range candidates {
		status := &CandidateStatus{
			StaffID:          staff.ID,
			StaffName:        staff.Name,
			IsEligible:       true,
			ViolatedRules:    make([]*RuleViolation, 0),
			Warnings:         make([]*RuleWarning, 0),
			ConstraintScores: make(map[string]float64),
		}

		// 检查所有约束型规则
		for _, rule := range rules.ConstraintRules {
			violation := c.checkSingleConstraint(staff, rule, currentDraft, shiftID, date, staffList)
			if violation != nil {
				if violation.IsHard {
					status.IsEligible = false
					status.ViolatedRules = append(status.ViolatedRules, violation)
				} else {
					status.Warnings = append(status.Warnings, &RuleWarning{
						RuleID:   rule.Rule.ID,
						RuleName: rule.Rule.Name,
						Message:  violation.Message,
					})
				}
			}
			// 计算"剩余空间"评分
			score := c.computeConstraintScore(staff, rule, currentDraft, shiftID, date)
			status.ConstraintScores[rule.Rule.ID] = score
		}

		// 检查所有依赖型规则（作为前置班次约束的补充校验）
		for _, rule := range rules.DependencyRules {
			violation := c.checkSingleDependency(staff, rule, currentDraft, shiftID, date)
			if violation != nil {
				if violation.IsHard {
					status.IsEligible = false
					status.ViolatedRules = append(status.ViolatedRules, violation)
				} else {
					status.Warnings = append(status.Warnings, &RuleWarning{
						RuleID:   rule.Rule.ID,
						RuleName: rule.Rule.Name,
						Message:  violation.Message,
					})
				}
			}
			// 依赖规则暂时不影响评分
			status.ConstraintScores[rule.Rule.ID] = 1.0
		}

		if status.IsEligible {
			result.EligibleCandidates = append(result.EligibleCandidates, status)
		} else {
			result.ExcludedCandidates = append(result.ExcludedCandidates, status)
		}
	}

	return result
}

// checkSingleConstraint 检查单个约束规则
func (c *ConstraintChecker) checkSingleConstraint(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
	allStaff []*model.Staff,
) *RuleViolation {
	switch rule.Rule.RuleType {
	case "maxCount":
		return c.checkMaxCount(staff, rule, draft, shiftID, date)
	case "consecutiveMax":
		return c.checkConsecutiveMax(staff, rule, draft, shiftID, date)
	case "minRestDays":
		return c.checkMinRestDays(staff, rule, draft, shiftID, date)
	case "exclusive":
		return c.checkExclusive(staff, rule, draft, shiftID, date)
	case "forbidden_day":
		return c.checkForbiddenDay(staff, rule, shiftID, date, allStaff)
	case "required_together":
		return c.checkRequiredTogether(staff, rule, draft, shiftID, date)
	default:
		return nil
	}
}

// checkSingleDependency 检查单个依赖规则（例如：下半夜的人必须从前一天的上半夜排班中选取）
func (c *ConstraintChecker) checkSingleDependency(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	// 判断当前班次在该依赖关系中是主体还是客体
	var isSubject = true
	hasAssociations := false
	for _, assoc := range rule.Rule.Associations {
		if assoc.AssociationType == "shift" && assoc.AssociationID == shiftID {
			hasAssociations = true
			if assoc.Role == "object" || assoc.Role == "target" || assoc.Role == "reference" {
				isSubject = false
			}
		}
	}

	// 如果当前班次不是主体（或者并不是给当前班次制定的规则），则在安排客体时不限制选人。
	// 下半夜的约束：主体是下半夜（后排），客体是上半夜（先排）。所以上半夜选人时很自由，下半夜选人时必须从前一天的客观结果中挑。
	if !isSubject || !hasAssociations {
		return nil
	}

	var hasValidSource = false
	var sourceCount = 0

	// 遍历所有客体班次
	for _, assoc := range rule.Rule.Associations {
		if assoc.AssociationType == "shift" && assoc.AssociationID != shiftID {
			sourceCount++
			offsetDays := 0
			if rule.Rule.TimeOffsetDays != nil {
				offsetDays = *rule.Rule.TimeOffsetDays
			}
			// 计算出前置班次应该发生在哪一天
			targetDate := date.AddDate(0, 0, offsetDays)

			// 只要 staff 在任意一个客体班次（比如本部/江北均可）当天被排了，就算合规
			if c.hasStaffShiftOnDate(draft, staff.ID, assoc.AssociationID, targetDate) {
				hasValidSource = true
				break
			}
		}
	}

	// 如果配置了来源班次，但该员工却未在任何一个来源班次被排班
	if sourceCount > 0 && !hasValidSource {
		return &RuleViolation{
			RuleID:   rule.Rule.ID,
			RuleName: rule.Rule.Name,
			IsHard:   true,
			Message:  "未在必需的前置班次中被排班",
		}
	}

	return nil
}

// checkMaxCount 检查最大次数约束
func (c *ConstraintChecker) checkMaxCount(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	if rule.Rule.MaxCount == nil {
		return nil
	}

	maxCount := *rule.Rule.MaxCount
	// 根据 TimeScope 确定统计范围
	startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)

	// 从草稿中统计该员工在时间范围内的排班次数
	currentCount := c.countStaffShiftInRange(draft, staff.ID, shiftID, startDate, endDate)

	if currentCount >= maxCount {
		return &RuleViolation{
			RuleID:   rule.Rule.ID,
			RuleName: rule.Rule.Name,
			IsHard:   true,
			Message:  fmt.Sprintf("已达到最大次数限制(%d/%d)", currentCount, maxCount),
		}
	}
	return nil
}

// checkConsecutiveMax 检查连续天数约束
func (c *ConstraintChecker) checkConsecutiveMax(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	if rule.Rule.ConsecutiveMax == nil {
		return nil
	}

	maxConsecutive := *rule.Rule.ConsecutiveMax
	// 从当前日期往前回溯，计算连续排班天数
	consecutiveDays := 0
	checkDate := date.AddDate(0, 0, -1)

	for {
		if c.hasStaffShiftOnDate(draft, staff.ID, shiftID, checkDate) {
			consecutiveDays++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	if consecutiveDays >= maxConsecutive {
		return &RuleViolation{
			RuleID:   rule.Rule.ID,
			RuleName: rule.Rule.Name,
			IsHard:   true,
			Message:  fmt.Sprintf("已连续排班%d天，超过限制(%d天)", consecutiveDays, maxConsecutive),
		}
	}
	return nil
}

// checkMinRestDays 检查最少休息天数约束
func (c *ConstraintChecker) checkMinRestDays(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	if rule.Rule.MinRestDays == nil {
		return nil
	}

	minRest := *rule.Rule.MinRestDays
	// 检查前 minRest 天内是否有关联班次的排班
	for i := 1; i <= minRest; i++ {
		checkDate := date.AddDate(0, 0, -i)
		// 检查关联班次（通过 Associations 获取）
		for _, assoc := range rule.Rule.Associations {
			if assoc.AssociationType == "shift" {
				if c.hasStaffShiftOnDate(draft, staff.ID, assoc.AssociationID, checkDate) {
					return &RuleViolation{
						RuleID:   rule.Rule.ID,
						RuleName: rule.Rule.Name,
						IsHard:   true,
						Message:  fmt.Sprintf("距上次该班次排班仅%d天，未满足最少休息%d天", i, minRest),
					}
				}
			}
		}
	}
	return nil
}

// checkExclusive 检查排他约束
// 规则语义：排了主体班次（subject）后，在 offsetDays 天后不能再排客体班次（object）。
// 当前排的班次可能是主体也可能是客体，需根据 assoc.Role 判断并反向计算偏移。
func (c *ConstraintChecker) checkExclusive(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	offsetDays := 0
	if rule.Rule.TimeOffsetDays != nil {
		offsetDays = *rule.Rule.TimeOffsetDays
	}

	// 判断当前班次在规则中是主体还是客体
	currentIsSubject := true
	for _, assoc := range rule.Rule.Associations {
		if assoc.AssociationType == "shift" && assoc.AssociationID == shiftID {
			if assoc.Role == "object" || assoc.Role == "target" {
				currentIsSubject = false
			}
			break
		}
	}

	for _, assoc := range rule.Rule.Associations {
		if assoc.AssociationType != "shift" || assoc.AssociationID == shiftID {
			continue
		}
		var targetDate time.Time
		if currentIsSubject {
			// 当前是主体（subject），客体在 date+offsetDays 处
			targetDate = date.AddDate(0, 0, offsetDays)
		} else {
			// 当前是客体（object），主体在 date-offsetDays 处
			targetDate = date.AddDate(0, 0, -offsetDays)
		}

		if c.hasStaffShiftOnDate(draft, staff.ID, assoc.AssociationID, targetDate) {
			return &RuleViolation{
				RuleID:   rule.Rule.ID,
				RuleName: rule.Rule.Name,
				IsHard:   true,
				Message:  "已排排他班次，与当前班次冲突",
			}
		}
	}
	return nil
}

// checkForbiddenDay 检查禁止日期约束
func (c *ConstraintChecker) checkForbiddenDay(
	staff *model.Staff,
	rule *ClassifiedRule,
	shiftID string,
	date time.Time,
	allStaff []*model.Staff,
) *RuleViolation {
	// V4.1: 先检查 ApplyScopes，确保规则只应用于目标员工
	if !c.isRuleApplicableToStaff(rule.Rule, staff, allStaff) {
		return nil
	}

	dateStr := date.Format("2006-01-02")

	// 检查具体日期禁止（targetDates）
	for _, fd := range parseForbiddenDates(rule.Rule.RuleData) {
		if fd == dateStr {
			return &RuleViolation{
				RuleID:   rule.Rule.ID,
				RuleName: rule.Rule.Name,
				IsHard:   true,
				Message:  fmt.Sprintf("%s 在禁止日期%s不可排班", staff.Name, dateStr),
			}
		}
	}

	// 检查星期几禁止（forbiddenWeekdays）
	weekday := int(date.Weekday())
	for _, fw := range parseForbiddenWeekdays(rule.Rule.RuleData) {
		if fw == weekday {
			return &RuleViolation{
				RuleID:   rule.Rule.ID,
				RuleName: rule.Rule.Name,
				IsHard:   true,
				Message:  fmt.Sprintf("%s 在禁止的星期(%s)不可排班", staff.Name, date.Format("Monday")),
			}
		}
	}

	// 检查每月内日期禁止（forbiddenMonthDays）
	day := date.Day()
	for _, fd := range parseForbiddenMonthDays(rule.Rule.RuleData) {
		if fd == day {
			return &RuleViolation{
				RuleID:   rule.Rule.ID,
				RuleName: rule.Rule.Name,
				IsHard:   true,
				Message:  fmt.Sprintf("%s 在每月%d日禁止排班", staff.Name, day),
			}
		}
	}

	return nil
}

// checkRequiredTogether 检查必须同时约束 (v4.2新增)
//
// 设计说明：此约束以【软约束】（IsHard: false）形式返回违规，而非硬排除。
//
// 原因：required_together 班次组的主调度逻辑由 Phase 1 的 RequiredTogetherScheduler
// 负责（通过成员池交集保证同组人员一致）。Phase 2 的 DefaultScheduler 作为兜底填充，
// 若此处设为硬约束，会导致如下死循环：
//
//	同组的两个班次 A、B 成员池不相交时 →
//	Phase 1 交集为空，两个班次均无人 →
//	Phase 2 先排的班次（设为 A）可以自由填充 →
//	后排的班次（B）触发硬约束"候选人必须也在 A 中"→ 候选人全部被排除 → B 永远为空
//
// 改为软约束后：Phase 2 两个班次均可独立填充；
// 实际违反 required_together 的情况由 LLM 规则调整器（Phase 7.5）检测并修复。
func (c *ConstraintChecker) checkRequiredTogether(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) *RuleViolation {
	// 检查关联的班次
	// required_together 的逻辑 (OR逻辑支持)：如果关联的多个班次中有任何一个排了人，当前员工必须至少存在于其中一个班次中。
	var hasScheduledRelatedShifts bool
	var foundInAnyRelatedShift bool

	for _, assoc := range rule.Rule.Associations {
		if assoc.AssociationType == "shift" && assoc.AssociationID != shiftID {
			targetShiftID := assoc.AssociationID

			// 计算带偏移量的目标日期 (以当前 shift 为主体源，关联班次为目标)
			offsetDays := 0
			if rule.Rule.TimeOffsetDays != nil {
				offsetDays = *rule.Rule.TimeOffsetDays
			}
			targetDate := date.AddDate(0, 0, offsetDays)
			dateStr := targetDate.Format("2006-01-02")

			// 检查目标班次是否已经有排班草稿
			if draft != nil && draft.Shifts != nil {
				if shiftDraft, ok := draft.Shifts[targetShiftID]; ok {
					if dayShift, ok := shiftDraft.Days[dateStr]; ok && len(dayShift.StaffIDs) > 0 {
						hasScheduledRelatedShifts = true

						for _, sid := range dayShift.StaffIDs {
							if sid == staff.ID {
								foundInAnyRelatedShift = true
								break
							}
						}
					}
				}
			}
		}
	}

	if hasScheduledRelatedShifts && !foundInAnyRelatedShift {
		// 软约束：记录违规但不硬排除候选人。
		// Phase 2 兜底填充阶段不应因此阻塞排班；
		// 违规情况由 LLM 调整器在后续阶段处理。
		return &RuleViolation{
			RuleID:   rule.Rule.ID,
			RuleName: rule.Rule.Name,
			IsHard:   false,
			Message:  "required_together未满足：未被安排至任何关联班次",
		}
	}

	return nil
}

// computeConstraintScore 计算约束"剩余空间"评分（0.0~1.0，越高越宽松）
func (c *ConstraintChecker) computeConstraintScore(
	staff *model.Staff,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	shiftID string,
	date time.Time,
) float64 {
	switch rule.Rule.RuleType {
	case "maxCount":
		if rule.Rule.MaxCount == nil {
			return 1.0
		}
		max := float64(*rule.Rule.MaxCount)
		startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)
		current := float64(c.countStaffShiftInRange(draft, staff.ID, shiftID, startDate, endDate))
		if max == 0 {
			return 0.0
		}
		return (max - current) / max
	default:
		return 1.0
	}
}

// 辅助方法
func (c *ConstraintChecker) getTimeScopeRange(timeScope string, date time.Time) (time.Time, time.Time) {
	switch timeScope {
	case "same_day":
		return date, date
	case "same_week":
		// 计算周的开始和结束
		weekday := int(date.Weekday())
		if weekday == 0 {
			weekday = 7 // 周日
		}
		start := date.AddDate(0, 0, -(weekday - 1))
		end := start.AddDate(0, 0, 6)
		return start, end
	case "same_month":
		start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		end := start.AddDate(0, 1, -1)
		return start, end
	default:
		return date, date
	}
}

func (c *ConstraintChecker) countStaffShiftInRange(draft *model.ScheduleDraft, staffID, shiftID string, start, end time.Time) int {
	if draft == nil {
		return 0
	}
	count := 0
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok {
		return 0
	}
	for dateStr, dayShift := range shiftDraft.Days {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end)) {
			for _, sid := range dayShift.StaffIDs {
				if sid == staffID {
					count++
					break
				}
			}
		}
	}
	return count
}

func (c *ConstraintChecker) hasStaffShiftOnDate(draft *model.ScheduleDraft, staffID, shiftID string, date time.Time) bool {
	if draft == nil {
		return false
	}
	dateStr := date.Format("2006-01-02")
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok {
		return false
	}
	dayShift, ok := shiftDraft.Days[dateStr]
	if !ok {
		return false
	}
	for _, sid := range dayShift.StaffIDs {
		if sid == staffID {
			return true
		}
	}
	return false
}

// isRuleApplicableToStaff 判断 forbidden_day 等规则是否适用于指定员工
// 逻辑与 personnel_constraints.go 中的 isRuleApplicableToStaff 保持一致：
//   - ApplyScope == "global" -> 适用于所有人
//   - ApplyScopes 有数据    -> 按 ScopeType 精确匹配
//   - 兼容旧模式：Associations 中的 employee 关联判断
func (c *ConstraintChecker) isRuleApplicableToStaff(rule *model.Rule, staff *model.Staff, allStaff []*model.Staff) bool {
	// 全局规则：适用于所有人
	if rule.ApplyScope == "global" {
		return true
	}

	// V4.1: 通过 ApplyScopes 判断
	if len(rule.ApplyScopes) > 0 {
		staffGroupIDs := c.getStaffGroupIDs(staff.ID, allStaff)
		return c.matchApplyScopes(rule.ApplyScopes, staff.ID, staffGroupIDs)
	}

	// 兼容旧模式：ApplyScopes 为空时，通过 Associations 中的 employee 关联判断
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == staff.ID {
			return true
		}
	}

	return false
}

// matchApplyScopes 判断 ApplyScopes 是否匹配指定员工
func (c *ConstraintChecker) matchApplyScopes(scopes []model.RuleApplyScope, staffID string, staffGroupIDs []string) bool {
	for _, scope := range scopes {
		switch scope.ScopeType {
		case model.ScopeTypeAll:
			return true
		case model.ScopeTypeEmployee:
			if scope.ScopeID == staffID {
				return true
			}
		case model.ScopeTypeGroup:
			for _, gid := range staffGroupIDs {
				if scope.ScopeID == gid {
					return true
				}
			}
		case model.ScopeTypeExcludeEmployee:
			if scope.ScopeID == staffID {
				return false
			}
		case model.ScopeTypeExcludeGroup:
			for _, gid := range staffGroupIDs {
				if scope.ScopeID == gid {
					return false
				}
			}
		}
	}

	// 如果只有 exclude 类型且未命中排除，则视为适用
	hasPositiveScope := false
	for _, scope := range scopes {
		if scope.ScopeType == model.ScopeTypeAll ||
			scope.ScopeType == model.ScopeTypeEmployee ||
			scope.ScopeType == model.ScopeTypeGroup {
			hasPositiveScope = true
			break
		}
	}
	if !hasPositiveScope && len(scopes) > 0 {
		return true
	}

	return false
}

// getStaffGroupIDs 获取员工所属的分组ID列表
func (c *ConstraintChecker) getStaffGroupIDs(staffID string, allStaff []*model.Staff) []string {
	for _, staff := range allStaff {
		if staff.ID == staffID {
			groupIDs := make([]string, 0, len(staff.Groups))
			for _, g := range staff.Groups {
				if g != nil {
					groupIDs = append(groupIDs, g.ID)
				}
			}
			return groupIDs
		}
	}
	return nil
}
