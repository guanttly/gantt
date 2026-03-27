package step

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gantt-saas/internal/core/rule"
)

type sharedCompatFixture struct {
	FixedAssignments struct {
		ShiftID                  string `json:"shift_id"`
		ExpectedTotalAssignments int    `json:"expected_total_assignments"`
		ExpectedUniqueEmployees  int    `json:"expected_unique_employees"`
		Dates                    []struct {
			Date          string   `json:"date"`
			StaffIDs      []string `json:"staff_ids"`
			RequiredCount int      `json:"required_count"`
		} `json:"dates"`
	} `json:"fixed_assignments"`
	Underfill struct {
		Shift struct {
			ID string `json:"id"`
		} `json:"shift"`
		Date             string `json:"date"`
		RequiredCount    int    `json:"required_count"`
		ExpectedAssigned int    `json:"expected_assigned"`
		ExpectedUnfilled int    `json:"expected_unfilled"`
		Staff            []struct {
			ID string `json:"id"`
		} `json:"staff"`
	} `json:"underfill"`
	SourceValidation struct {
		SourceShiftID        string `json:"source_shift_id"`
		TargetShiftID        string `json:"target_shift_id"`
		Date                 string `json:"date"`
		AllowedEmployeeID    string `json:"allowed_employee_id"`
		DisallowedEmployeeID string `json:"disallowed_employee_id"`
		RuleID               string `json:"rule_id"`
		RuleName             string `json:"rule_name"`
	} `json:"source_validation"`
	ExclusiveSemantics struct {
		Date     string   `json:"date"`
		ShiftIDs []string `json:"shift_ids"`
		Staff    []struct {
			ID string `json:"id"`
		} `json:"staff"`
		Requirements             map[string]int `json:"requirements"`
		RuleID                   string         `json:"rule_id"`
		RuleName                 string         `json:"rule_name"`
		ExpectedTotalAssignments int            `json:"expected_total_assignments"`
		ExpectedUniqueEmployees  int            `json:"expected_unique_employees"`
		ExpectedViolations       int            `json:"expected_violations"`
	} `json:"exclusive_semantics"`
}

func loadSharedCompatFixture(t *testing.T) sharedCompatFixture {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "testdata", "schedule_compat", "basic_semantics.json")
		data, err := os.ReadFile(candidate)
		if err == nil {
			var fixture sharedCompatFixture
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			return fixture
		}
	}

	t.Fatal("shared fixture not found")
	return sharedCompatFixture{}
}

func TestSharedFixture_PhaseZeroMatchesFixedAssignmentSemantics(t *testing.T) {
	fixture := loadSharedCompatFixture(t)

	requirements := make(map[string]map[string]int)
	requirements[fixture.FixedAssignments.ShiftID] = make(map[string]int)
	staffIDs := fixture.FixedAssignments.Dates[0].StaffIDs
	for _, item := range fixture.FixedAssignments.Dates {
		requirements[fixture.FixedAssignments.ShiftID][item.Date] = item.RequiredCount
	}

	state := NewScheduleState("sch-1", "org-1", "", fixture.FixedAssignments.Dates[0].Date, fixture.FixedAssignments.Dates[len(fixture.FixedAssignments.Dates)-1].Date, "user-1", &ScheduleConfig{
		ShiftIDs:     []string{fixture.FixedAssignments.ShiftID},
		Requirements: requirements,
	})
	state.ShiftOrder = makeShifts(fixture.FixedAssignments.ShiftID)
	state.EffectiveRules = []rule.Rule{{
		Category: rule.CategoryConstraint,
		SubType:  rule.SubTypeMust,
		Config: mustJSON(t, rule.RequiredTogetherConfig{
			Type:        "fixed_schedule",
			EmployeeIDs: staffIDs,
			ShiftID:     fixture.FixedAssignments.ShiftID,
		}),
	}}

	if err := (&PhaseZeroStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != fixture.FixedAssignments.ExpectedTotalAssignments {
		t.Fatalf("expected %d fixed assignments, got %d", fixture.FixedAssignments.ExpectedTotalAssignments, len(state.Assignments))
	}
	uniqueEmployees := make(map[string]bool)
	for _, assignment := range state.Assignments {
		uniqueEmployees[assignment.EmployeeID] = true
	}
	if len(uniqueEmployees) != fixture.FixedAssignments.ExpectedUniqueEmployees {
		t.Fatalf("expected %d unique employees, got %d", fixture.FixedAssignments.ExpectedUniqueEmployees, len(uniqueEmployees))
	}
}

func TestSharedFixture_PhaseTwoAllowsUnderfill(t *testing.T) {
	fixture := loadSharedCompatFixture(t)

	state := NewScheduleState("sch-2", "org-1", "", fixture.Underfill.Date, fixture.Underfill.Date, "user-1", &ScheduleConfig{
		ShiftIDs: []string{fixture.Underfill.Shift.ID},
		Requirements: map[string]map[string]int{
			fixture.Underfill.Shift.ID: {
				fixture.Underfill.Date: fixture.Underfill.RequiredCount,
			},
		},
	})
	state.ShiftOrder = makeShifts(fixture.Underfill.Shift.ID)

	candidates := make([]string, 0, len(fixture.Underfill.Staff))
	for _, staff := range fixture.Underfill.Staff {
		candidates = append(candidates, staff.ID)
	}
	state.Candidates[fixture.Underfill.Shift.ID+"|"+fixture.Underfill.Date] = candidates

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := state.CountAssigned(fixture.Underfill.Shift.ID, fixture.Underfill.Date); got != fixture.Underfill.ExpectedAssigned {
		t.Fatalf("expected %d partial assignments, got %d", fixture.Underfill.ExpectedAssigned, got)
	}
	if unfilled := fixture.Underfill.RequiredCount - len(state.Assignments); unfilled != fixture.Underfill.ExpectedUnfilled {
		t.Fatalf("expected %d unfilled slots, got %d", fixture.Underfill.ExpectedUnfilled, unfilled)
	}
}

func TestSharedFixture_FullValidationChecksSourceRule(t *testing.T) {
	fixture := loadSharedCompatFixture(t)

	state := NewScheduleState("sch-3", "org-1", "", fixture.SourceValidation.Date, fixture.SourceValidation.Date, "user-1", nil)
	state.Assignments = []Assignment{
		{
			ID:         "a-source",
			EmployeeID: fixture.SourceValidation.AllowedEmployeeID,
			ShiftID:    fixture.SourceValidation.SourceShiftID,
			Date:       fixture.SourceValidation.Date,
			Source:     SourceRule,
		},
		{
			ID:         "a-target-valid",
			EmployeeID: fixture.SourceValidation.AllowedEmployeeID,
			ShiftID:    fixture.SourceValidation.TargetShiftID,
			Date:       fixture.SourceValidation.Date,
			Source:     SourceFill,
		},
		{
			ID:         "a-target-invalid",
			EmployeeID: fixture.SourceValidation.DisallowedEmployeeID,
			ShiftID:    fixture.SourceValidation.TargetShiftID,
			Date:       fixture.SourceValidation.Date,
			Source:     SourceFill,
		},
	}
	state.EffectiveRules = []rule.Rule{{
		ID:        fixture.SourceValidation.RuleID,
		Name:      fixture.SourceValidation.RuleName,
		Category:  rule.CategoryDependency,
		SubType:   rule.SubTypeSource,
		IsEnabled: true,
		Config: mustJSON(t, rule.StaffSourceConfig{
			Type:          "staff_source",
			TargetShiftID: fixture.SourceValidation.TargetShiftID,
			SourceShiftID: fixture.SourceValidation.SourceShiftID,
		}),
	}}

	if err := (&FullValidationStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 1 {
		t.Fatalf("expected 1 source-rule violation, got %d", len(state.Violations))
	}

	v := state.Violations[0]
	if v.EmployeeID != fixture.SourceValidation.DisallowedEmployeeID {
		t.Fatalf("expected violation for %s, got %s", fixture.SourceValidation.DisallowedEmployeeID, v.EmployeeID)
	}
	if v.RuleID != fixture.SourceValidation.RuleID {
		t.Fatalf("expected rule id %s, got %s", fixture.SourceValidation.RuleID, v.RuleID)
	}
}

func TestSharedFixture_ExclusiveSemanticsAvoidDoubleBooking(t *testing.T) {
	fixture := loadSharedCompatFixture(t)

	state := NewScheduleState("sch-4", "org-1", "", fixture.ExclusiveSemantics.Date, fixture.ExclusiveSemantics.Date, "user-1", &ScheduleConfig{
		ShiftIDs: fixture.ExclusiveSemantics.ShiftIDs,
		Requirements: map[string]map[string]int{
			fixture.ExclusiveSemantics.ShiftIDs[0]: {
				fixture.ExclusiveSemantics.Date: fixture.ExclusiveSemantics.Requirements[fixture.ExclusiveSemantics.ShiftIDs[0]],
			},
			fixture.ExclusiveSemantics.ShiftIDs[1]: {
				fixture.ExclusiveSemantics.Date: fixture.ExclusiveSemantics.Requirements[fixture.ExclusiveSemantics.ShiftIDs[1]],
			},
		},
	})
	state.ShiftOrder = makeShifts(fixture.ExclusiveSemantics.ShiftIDs...)
	for _, shiftID := range fixture.ExclusiveSemantics.ShiftIDs {
		key := shiftID + "|" + fixture.ExclusiveSemantics.Date
		for _, staff := range fixture.ExclusiveSemantics.Staff {
			state.Candidates[key] = append(state.Candidates[key], staff.ID)
		}
	}
	state.EffectiveRules = []rule.Rule{{
		ID:        fixture.ExclusiveSemantics.RuleID,
		Name:      fixture.ExclusiveSemantics.RuleName,
		Category:  rule.CategoryConstraint,
		SubType:   rule.SubTypeForbid,
		IsEnabled: true,
		Config: mustJSON(t, rule.ExclusiveShiftsConfig{
			Type:     "exclusive_shifts",
			ShiftIDs: fixture.ExclusiveSemantics.ShiftIDs,
			Scope:    "same_day",
		}),
	}}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("phase two error: %v", err)
	}
	if err := (&FullValidationStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if len(state.Assignments) != fixture.ExclusiveSemantics.ExpectedTotalAssignments {
		t.Fatalf("expected %d assignments, got %d", fixture.ExclusiveSemantics.ExpectedTotalAssignments, len(state.Assignments))
	}
	uniqueEmployees := make(map[string]bool)
	for _, assignment := range state.Assignments {
		uniqueEmployees[assignment.EmployeeID] = true
	}
	if len(uniqueEmployees) != fixture.ExclusiveSemantics.ExpectedUniqueEmployees {
		t.Fatalf("expected %d unique employees, got %d", fixture.ExclusiveSemantics.ExpectedUniqueEmployees, len(uniqueEmployees))
	}
	if len(state.Violations) != fixture.ExclusiveSemantics.ExpectedViolations {
		t.Fatalf("expected %d exclusive violations after fill, got %d", fixture.ExclusiveSemantics.ExpectedViolations, len(state.Violations))
	}
}
