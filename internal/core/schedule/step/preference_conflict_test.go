package step

import (
	"context"
	"testing"

	"gantt-saas/internal/core/rule"
)

func TestPhaseTwoStep_PreferenceScoringSkipsLowerPriorityConflict(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day"},
		Requirements: map[string]map[string]int{
			"day": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("day")
	state.Candidates["day|2026-03-23"] = []string{"e1", "e2"}
	conflict := rule.RuleConflict{RuleID1: "prefer-high", RuleID2: "prefer-low", ConflictType: "exclusive"}
	state.EffectiveRules = []rule.Rule{
		{
			ID:        "prefer-high",
			Priority:  10,
			Category:  rule.CategoryPreference,
			SubType:   rule.SubTypePrefer,
			IsEnabled: true,
			Config: mustJSON(t, rule.PreferEmployeeConfig{
				Type:       "prefer_employee",
				EmployeeID: "e1",
				ShiftID:    "day",
				Weight:     20,
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
		{
			ID:        "prefer-low",
			Priority:  20,
			Category:  rule.CategoryPreference,
			SubType:   rule.SubTypePrefer,
			IsEnabled: true,
			Config: mustJSON(t, rule.PreferEmployeeConfig{
				Type:       "prefer_employee",
				EmployeeID: "e2",
				ShiftID:    "day",
				Weight:     100,
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
	}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != 1 {
		t.Fatalf("expected one assignment, got %d", len(state.Assignments))
	}
	if state.Assignments[0].EmployeeID != "e1" {
		t.Fatalf("expected phase two to select e1 after conflict suppression, got %s", state.Assignments[0].EmployeeID)
	}
}
