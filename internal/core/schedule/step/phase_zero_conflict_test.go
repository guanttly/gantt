package step

import (
	"context"
	"testing"

	"gantt-saas/internal/core/rule"
)

func TestPhaseZeroStep_FixedScheduleSkipsLowerPriorityConflictInCurrentShiftContext(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("day")
	conflict := rule.RuleConflict{RuleID1: "fixed-high", RuleID2: "fixed-low", ConflictType: "exclusive"}
	state.EffectiveRules = []rule.Rule{
		{
			ID:        "fixed-high",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeMust,
			IsEnabled: true,
			Config: mustJSON(t, rule.RequiredTogetherConfig{
				Type:        "fixed_schedule",
				EmployeeIDs: []string{"e1"},
				ShiftID:     "night",
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
		{
			ID:        "fixed-low",
			Priority:  20,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeMust,
			IsEnabled: true,
			Config: mustJSON(t, rule.RequiredTogetherConfig{
				Type:        "fixed_schedule",
				EmployeeIDs: []string{"e2"},
				ShiftID:     "day",
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
	}

	if err := (&PhaseZeroStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != 1 {
		t.Fatalf("expected one fixed assignment, got %d", len(state.Assignments))
	}
	if state.Assignments[0].EmployeeID != "e2" {
		t.Fatalf("expected day fixed schedule to keep e2, got %s", state.Assignments[0].EmployeeID)
	}
	if state.Assignments[0].ShiftID != "day" {
		t.Fatalf("expected assignment on day shift, got %s", state.Assignments[0].ShiftID)
	}
}
