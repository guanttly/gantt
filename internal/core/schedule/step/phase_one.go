package step

import (
	"context"

	"gantt-saas/internal/core/rule"
)

// PhaseOneStep 规则性占位：处理排他规则、人员来源规则。
type PhaseOneStep struct{}

// Name 返回步骤名称。
func (s *PhaseOneStep) Name() string { return "PhaseOne" }

// Execute 执行规则性占位。
func (s *PhaseOneStep) Execute(ctx context.Context, state *ScheduleState) error {
	for _, sh := range state.ShiftOrder {
		dates := state.Config.Requirements[sh.ID]
		for dateStr, needed := range dates {
			key := sh.ID + "|" + dateStr
			candidates := state.Candidates[key]
			if len(candidates) == 0 {
				continue
			}

			assigned := state.CountAssigned(sh.ID, dateStr)
			if assigned >= needed {
				continue
			}

			// 处理排他规则
			candidates = s.applyExclusiveRules(state, candidates, sh.ID, dateStr)

			// 处理人员来源规则
			candidates = s.applySourceRules(state, candidates, sh.ID, dateStr)

			// 处理联动依赖规则
			candidates = s.applyRequiredTogetherRules(state, candidates, sh.ID, dateStr)

			// 更新候选人列表
			state.Candidates[key] = candidates
		}
	}
	return nil
}

// applyExclusiveRules 应用排他规则。
func (s *PhaseOneStep) applyExclusiveRules(state *ScheduleState, candidates []string, shiftID, date string) []string {
	var filtered []string
	for _, empID := range candidates {
		excluded := false
		for _, r := range rule.ActiveRulesForShiftContext(state.EffectiveRules, empID, state.EmployeeGroupIDs[empID], shiftID) {
			if r.Category != rule.CategoryConstraint || r.SubType != rule.SubTypeForbid {
				continue
			}
			relatedShiftIDs, offsetDays, matched := r.ExclusiveSemantics().CounterpartShiftIDs(shiftID)
			if !matched || len(relatedShiftIDs) == 0 {
				continue
			}
			relatedSet := make(map[string]bool, len(relatedShiftIDs))
			for _, relatedShiftID := range relatedShiftIDs {
				relatedSet[relatedShiftID] = true
			}
			relatedDate := rule.DateStringWithOffset(date, offsetDays)
			for _, a := range state.Assignments {
				if a.EmployeeID == empID && a.Date == relatedDate && relatedSet[a.ShiftID] {
					excluded = true
					break
				}
			}
			if excluded {
				break
			}
		}
		if !excluded {
			filtered = append(filtered, empID)
		}
	}
	return filtered
}

// applySourceRules 应用人员来源规则。
func (s *PhaseOneStep) applySourceRules(state *ScheduleState, candidates []string, shiftID, date string) []string {
	var filtered []string
	for _, empID := range candidates {
		matchedAnyRule := false
		allowed := false
		for _, r := range rule.ActiveRulesForShiftContext(state.EffectiveRules, empID, state.EmployeeGroupIDs[empID], shiftID) {
			if r.Category != rule.CategoryDependency || r.SubType != rule.SubTypeSource {
				continue
			}
			sourceShiftIDs, offsetDays, matched := r.SourceSemantics().SourceShiftIDsForTarget(shiftID)
			if !matched || len(sourceShiftIDs) == 0 {
				continue
			}
			matchedAnyRule = true
			sourceSet := make(map[string]bool, len(sourceShiftIDs))
			for _, sourceShiftID := range sourceShiftIDs {
				sourceSet[sourceShiftID] = true
			}
			sourceDate := rule.DateStringWithOffset(date, offsetDays)
			for _, a := range state.Assignments {
				if a.EmployeeID == empID && a.Date == sourceDate && sourceSet[a.ShiftID] {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}
		if !matchedAnyRule || allowed {
			filtered = append(filtered, empID)
		}
	}
	return filtered
}

func (s *PhaseOneStep) applyRequiredTogetherRules(state *ScheduleState, candidates []string, shiftID, date string) []string {
	var filtered []string
	for _, empID := range candidates {
		if _, ok := planRequiredTogetherAssignments(state, empID, shiftID, date); ok {
			filtered = append(filtered, empID)
		}
	}
	return filtered
}
