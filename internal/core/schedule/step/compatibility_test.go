package step

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"gantt-saas/internal/core/rule"
)

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return b
}

func TestPhaseZeroStep_AppliesFixedScheduleRule(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day"},
		Requirements: map[string]map[string]int{
			"day": {
				"2026-03-23": 2,
				"2026-03-24": 2,
			},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-24", "user-1", config)
	state.ShiftOrder = makeShifts("day")
	state.EffectiveRules = []rule.Rule{
		{
			Category: rule.CategoryConstraint,
			SubType:  rule.SubTypeMust,
			Config: mustJSON(t, rule.RequiredTogetherConfig{
				Type:        "fixed_schedule",
				EmployeeIDs: []string{"e1", "e2"},
				ShiftID:     "day",
			}),
		},
	}

	s := &PhaseZeroStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 4 {
		t.Fatalf("expected 4 fixed assignments, got %d", got)
	}

	seen := map[string]bool{}
	for _, a := range state.Assignments {
		if a.Source != SourceFixed {
			t.Fatalf("expected source %q, got %q", SourceFixed, a.Source)
		}
		seen[a.EmployeeID+"|"+a.Date] = true
	}

	for _, key := range []string{
		"e1|2026-03-23",
		"e1|2026-03-24",
		"e2|2026-03-23",
		"e2|2026-03-24",
	} {
		if !seen[key] {
			t.Errorf("missing fixed assignment %s", key)
		}
	}
}

func TestPhaseOneStep_ExclusiveRulesMatchLegacyExpectation(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-23": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("day", "night")
	state.Candidates["day|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Assignments = []Assignment{{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"}}
	state.EffectiveRules = []rule.Rule{
		{
			Category: rule.CategoryConstraint,
			SubType:  rule.SubTypeForbid,
			Config: mustJSON(t, rule.ExclusiveShiftsConfig{
				Type:     "exclusive_shifts",
				ShiftIDs: []string{"day", "night"},
				Scope:    "same_day",
			}),
		},
	}

	s := &PhaseOneStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	want := []string{"e2", "e3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}
}

func TestPhaseOneStep_SourceRulesMatchLegacyExpectation(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-23": 2},
			"night": {"2026-03-23": 2},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("day", "night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
		{EmployeeID: "e3", ShiftID: "day", Date: "2026-03-23"},
	}
	state.EffectiveRules = []rule.Rule{
		{
			Category: rule.CategoryDependency,
			SubType:  rule.SubTypeSource,
			Config: mustJSON(t, rule.StaffSourceConfig{
				Type:          "staff_source",
				TargetShiftID: "night",
				SourceShiftID: "day",
			}),
		},
	}

	s := &PhaseOneStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	want := []string{"e1", "e3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}
}

func TestPhaseTwoStep_AllowsUnderfillWithoutErrorLikeLegacyDefaultScheduler(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"night"},
		Requirements: map[string]map[string]int{
			"night": {"2026-03-23": 3},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}

	s := &PhaseTwoStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := state.CountAssigned("night", "2026-03-23"); got != 2 {
		t.Fatalf("expected partial fill with 2 assignments, got %d", got)
	}

	for _, a := range state.Assignments {
		if a.Source != SourceFill {
			t.Fatalf("expected source %q, got %q", SourceFill, a.Source)
		}
	}
}

func TestFullValidationStep_FindsExclusiveConflictsLikeLegacyValidator(t *testing.T) {
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", nil)
	state.Assignments = []Assignment{
		{ID: "a1", EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23", Source: SourceRule},
		{ID: "a2", EmployeeID: "e1", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
	}
	state.EffectiveRules = []rule.Rule{
		{
			ID:        "r-exclusive",
			Name:      "同日排他",
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeForbid,
			IsEnabled: true,
			Config: mustJSON(t, rule.ExclusiveShiftsConfig{
				Type:     "exclusive_shifts",
				ShiftIDs: []string{"day", "night"},
				Scope:    "same_day",
			}),
		},
	}

	s := &FullValidationStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 2 {
		t.Fatalf("expected 2 violations for mutual exclusive conflict, got %d", len(state.Violations))
	}

	for _, v := range state.Violations {
		if v.RuleID != "r-exclusive" {
			t.Fatalf("expected rule id r-exclusive, got %s", v.RuleID)
		}
	}
}
