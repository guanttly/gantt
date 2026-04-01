package step

import (
	"sort"

	"gantt-saas/internal/core/rule"
)

type linkedAssignmentPlan struct {
	ShiftID string
	Date    string
}

func planRequiredTogetherAssignments(state *ScheduleState, employeeID, targetShiftID, targetDate string) ([]linkedAssignmentPlan, bool) {
	if state == nil {
		return nil, false
	}
	targetKey := targetShiftID + "|" + targetDate

	occupiedDates := make(map[string]string)
	for _, assignment := range state.Assignments {
		if assignment.EmployeeID == employeeID {
			occupiedDates[assignment.Date] = assignment.ShiftID
		}
	}
	occupiedDates[targetDate] = targetShiftID
	plannedCounts := make(map[string]int)
	plannedKeys := make(map[string]linkedAssignmentPlan)
	plans := make([]linkedAssignmentPlan, 0)

	resolving := make(map[string]bool)
	resolved := make(map[string]bool)

	if !collectRequiredTogetherPlans(state, employeeID, targetShiftID, targetDate, targetKey, occupiedDates, plannedCounts, plannedKeys, resolving, resolved, &plans) {
		return nil, false
	}

	sort.Slice(plans, func(i, j int) bool {
		if plans[i].Date != plans[j].Date {
			return plans[i].Date < plans[j].Date
		}
		return plans[i].ShiftID < plans[j].ShiftID
	})

	return plans, true
}

func collectRequiredTogetherPlans(
	state *ScheduleState,
	employeeID, shiftID, date string,
	targetKey string,
	occupiedDates map[string]string,
	plannedCounts map[string]int,
	plannedKeys map[string]linkedAssignmentPlan,
	resolving map[string]bool,
	resolved map[string]bool,
	plans *[]linkedAssignmentPlan,
) bool {
	nodeKey := shiftID + "|" + date
	if resolved[nodeKey] {
		return true
	}
	if resolving[nodeKey] {
		return true
	}
	resolving[nodeKey] = true
	defer func() {
		delete(resolving, nodeKey)
		resolved[nodeKey] = true
	}()

	for _, currentRule := range rule.ActiveRulesForShiftContext(state.EffectiveRules, employeeID, state.EmployeeGroupIDs[employeeID], shiftID) {
		if currentRule.SubType != rule.SubTypeMust {
			continue
		}

		requiredShiftIDs, offsetDays, matched := currentRule.RequiredTogetherSemantics().RequiredShiftIDsForTarget(shiftID)
		if !matched || len(requiredShiftIDs) == 0 {
			continue
		}

		requiredDate := rule.DateStringWithOffset(date, offsetDays)
		for _, requiredShiftID := range requiredShiftIDs {
			if requiredShiftID == shiftID && requiredDate == date {
				continue
			}

			planKey := requiredShiftID + "|" + requiredDate
			if planKey == targetKey {
				continue
			}
			if _, exists := plannedKeys[planKey]; exists {
				if !collectRequiredTogetherPlans(state, employeeID, requiredShiftID, requiredDate, targetKey, occupiedDates, plannedCounts, plannedKeys, resolving, resolved, plans) {
					return false
				}
				continue
			}
			if !canPlanRequiredShift(state, employeeID, requiredShiftID, requiredDate, occupiedDates, plannedCounts, plannedKeys) {
				return false
			}

			if state.IsOccupiedForShift(employeeID, requiredShiftID, requiredDate) {
				if !collectRequiredTogetherPlans(state, employeeID, requiredShiftID, requiredDate, targetKey, occupiedDates, plannedCounts, plannedKeys, resolving, resolved, plans) {
					return false
				}
				continue
			}

			if _, exists := plannedKeys[planKey]; !exists {
				plannedKeys[planKey] = linkedAssignmentPlan{ShiftID: requiredShiftID, Date: requiredDate}
				plannedCounts[planKey]++
				occupiedDates[requiredDate] = requiredShiftID
				*plans = append(*plans, linkedAssignmentPlan{ShiftID: requiredShiftID, Date: requiredDate})
			}

			if !collectRequiredTogetherPlans(state, employeeID, requiredShiftID, requiredDate, targetKey, occupiedDates, plannedCounts, plannedKeys, resolving, resolved, plans) {
				return false
			}
		}
	}

	return true
}

func canPlanRequiredShift(
	state *ScheduleState,
	employeeID, requiredShiftID, requiredDate string,
	occupiedDates map[string]string,
	plannedCounts map[string]int,
	plannedKeys map[string]linkedAssignmentPlan,
) bool {
	planKey := requiredShiftID + "|" + requiredDate
	if state.IsOccupiedForShift(employeeID, requiredShiftID, requiredDate) {
		return true
	}
	if existingShiftID, occupied := occupiedDates[requiredDate]; occupied && existingShiftID != requiredShiftID {
		return false
	}
	if _, exists := plannedKeys[planKey]; exists {
		return true
	}
	if !containsCandidate(state.Candidates[planKey], employeeID) {
		return false
	}

	needed := requiredCount(state, requiredShiftID, requiredDate)
	if needed == 0 {
		return false
	}
	if state.CountAssigned(requiredShiftID, requiredDate)+plannedCounts[planKey] >= needed {
		return false
	}
	return true
}

func requiredCount(state *ScheduleState, shiftID, date string) int {
	if state == nil || state.Config == nil || state.Config.Requirements == nil {
		return 0
	}
	return state.Config.Requirements[shiftID][date]
}

func containsCandidate(candidates []string, employeeID string) bool {
	for _, candidateID := range candidates {
		if candidateID == employeeID {
			return true
		}
	}
	return false
}
