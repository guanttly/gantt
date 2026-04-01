package step

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"gantt-saas/internal/core/rule"
)

type stubFixedAssignmentProvider struct {
	calendar map[string]map[string][]string
}

func (s stubFixedAssignmentProvider) GetFixedAssignmentsForRange(_ context.Context, _ []string, _, _ string) (map[string]map[string][]string, error) {
	return s.calendar, nil
}

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
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeMust,
			IsEnabled: true,
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

func TestPhaseZeroStep_AppliesShiftFixedAssignmentsBeforeRules(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day"},
		Requirements: map[string]map[string]int{
			"day": {
				"2026-03-23": 2,
				"2026-03-24": 1,
			},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-24", "user-1", config)
	state.ShiftOrder = makeShifts("day")

	s := &PhaseZeroStep{FixedAssignmentProvider: stubFixedAssignmentProvider{calendar: map[string]map[string][]string{
		"day": {
			"2026-03-23": {"e1", "e2", "e3"},
			"2026-03-24": {"e1"},
		},
	}}}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 3 {
		t.Fatalf("expected 3 assignments capped by requirement, got %d", got)
	}
	if state.CountAssigned("day", "2026-03-23") != 2 {
		t.Fatalf("expected 2 assignments on 2026-03-23, got %d", state.CountAssigned("day", "2026-03-23"))
	}
	if state.CountAssigned("day", "2026-03-24") != 1 {
		t.Fatalf("expected 1 assignment on 2026-03-24, got %d", state.CountAssigned("day", "2026-03-24"))
	}
	for _, assignment := range state.Assignments {
		if assignment.Source != SourceFixed {
			t.Fatalf("expected source %q, got %q", SourceFixed, assignment.Source)
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
			Category:  rule.CategoryDependency,
			SubType:   rule.SubTypeSource,
			IsEnabled: true,
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

func TestPhaseOneStep_SourceRulesUseAssociationsAndOffset(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"night": {"2026-03-23": 2},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-22", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-22"},
		{EmployeeID: "e3", ShiftID: "day", Date: "2026-03-22"},
	}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{{
		Category:       rule.CategoryDependency,
		SubType:        rule.SubTypeSource,
		IsEnabled:      true,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

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

func TestPhaseOneStep_RequiredTogetherRulesUseAssociationsAndOffset(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"night": {"2026-03-23": 2},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-22", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-22"},
		{EmployeeID: "e3", ShiftID: "day", Date: "2026-03-22"},
	}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{{
		Category:       rule.CategoryConstraint,
		SubType:        rule.SubTypeMust,
		IsEnabled:      true,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

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

func TestFullValidationStep_UsesAssociationSemanticsForSourceRule(t *testing.T) {
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-22", "2026-03-23", "user-1", nil)
	minusOne := -1
	state.Assignments = []Assignment{
		{ID: "a-src", EmployeeID: "e1", ShiftID: "day", Date: "2026-03-22", Source: SourceRule},
		{ID: "a-ok", EmployeeID: "e1", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
		{ID: "a-bad", EmployeeID: "e2", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
	}
	state.EffectiveRules = []rule.Rule{{
		ID:             "r-source-assoc",
		Name:           "关联来源",
		Category:       rule.CategoryDependency,
		SubType:        rule.SubTypeSource,
		IsEnabled:      true,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	s := &FullValidationStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 1 {
		t.Fatalf("expected 1 source-rule violation, got %d", len(state.Violations))
	}
	if state.Violations[0].EmployeeID != "e2" {
		t.Fatalf("expected violation for e2, got %s", state.Violations[0].EmployeeID)
	}
}

func TestFullValidationStep_UsesAssociationSemanticsForRequiredTogetherRule(t *testing.T) {
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-22", "2026-03-23", "user-1", nil)
	minusOne := -1
	state.Assignments = []Assignment{
		{ID: "a-ok-source", EmployeeID: "e1", ShiftID: "day", Date: "2026-03-22", Source: SourceRule},
		{ID: "a-ok-target", EmployeeID: "e1", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
		{ID: "a-bad", EmployeeID: "e2", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
	}
	state.EffectiveRules = []rule.Rule{{
		ID:             "r-must-assoc",
		Name:           "关联必须",
		Category:       rule.CategoryConstraint,
		SubType:        rule.SubTypeMust,
		IsEnabled:      true,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	s := &FullValidationStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 1 {
		t.Fatalf("expected 1 must-rule violation, got %d", len(state.Violations))
	}
	if state.Violations[0].EmployeeID != "e2" {
		t.Fatalf("expected violation for e2, got %s", state.Violations[0].EmployeeID)
	}
}

func TestFullValidationStep_UsesAssociationSemanticsForExclusiveRule(t *testing.T) {
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", nil)
	state.Assignments = []Assignment{
		{ID: "a1", EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23", Source: SourceRule},
		{ID: "a2", EmployeeID: "e1", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
	}
	state.EffectiveRules = []rule.Rule{{
		ID:        "r-exclusive-assoc",
		Name:      "关联排他",
		Category:  rule.CategoryConstraint,
		SubType:   rule.SubTypeForbid,
		IsEnabled: true,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	s := &FullValidationStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 2 {
		t.Fatalf("expected 2 violations for association exclusive conflict, got %d", len(state.Violations))
	}
	for _, v := range state.Violations {
		if v.RuleID != "r-exclusive-assoc" {
			t.Fatalf("expected rule id r-exclusive-assoc, got %s", v.RuleID)
		}
	}
}
