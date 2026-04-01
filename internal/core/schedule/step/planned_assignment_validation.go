package step

import (
	"context"
	"time"

	"gantt-saas/internal/core/rule/checker"
)

func validatePlannedAssignments(
	ctx context.Context,
	state *ScheduleState,
	checkerAssignments []checker.Assignment,
	plannedAssignments []Assignment,
) ([]checker.Assignment, []Violation) {
	working := append([]checker.Assignment(nil), checkerAssignments...)
	for _, assignment := range plannedAssignments {
		d, _ := time.Parse("2006-01-02", assignment.Date)
		working = append(working, checker.Assignment{
			EmployeeID: assignment.EmployeeID,
			ShiftID:    assignment.ShiftID,
			Date:       d,
		})
	}
	violations := make([]Violation, 0)

	for _, assignment := range plannedAssignments {
		d, _ := time.Parse("2006-01-02", assignment.Date)
		checkCtx := &checker.CheckContext{
			EmployeeID:       assignment.EmployeeID,
			ShiftID:          assignment.ShiftID,
			Date:             d,
			Assignments:      working,
			Candidates:       state.Candidates[assignment.ShiftID+"|"+assignment.Date],
			EmployeeGroupIDs: state.EmployeeGroupIDs[assignment.EmployeeID],
		}

		results := checker.ValidateAll(ctx, state.EffectiveRules, checkCtx)
		failed := false
		for _, result := range results {
			if result.Pass {
				continue
			}
			failed = true
			violations = append(violations, Violation{
				EmployeeID: assignment.EmployeeID,
				ShiftID:    assignment.ShiftID,
				Date:       assignment.Date,
				RuleID:     result.RuleID,
				RuleName:   result.RuleName,
				Reason:     result.Reason,
			})
		}
		if failed {
			return checkerAssignments, violations
		}
	}

	return working, nil
}
