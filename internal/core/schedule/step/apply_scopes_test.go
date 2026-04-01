package step

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"gantt-saas/internal/core/rule"
)

func mustJSONForApplyScopes(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

type stubRuleScopeGroupProvider struct {
	members map[string][]string
}

func (s stubRuleScopeGroupProvider) GetMemberEmployeeIDs(_ context.Context, groupID string) ([]string, error) {
	return s.members[groupID], nil
}

func TestFilterCandidatesStep_LoadRuleScopedGroups(t *testing.T) {
	groupID := "grp-1"
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", nil)
	state.EffectiveRules = []rule.Rule{{
		ApplyScopes: []rule.RuleApplyScope{{
			ScopeType: rule.ScopeTypeGroup,
			ScopeID:   &groupID,
		}},
	}}

	step := &FilterCandidatesStep{GroupMemberProvider: stubRuleScopeGroupProvider{members: map[string][]string{groupID: {"e1", "e2"}}}}
	if err := step.loadRuleScopedGroups(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.EmployeeGroupIDs["e1"]
	if got == nil || !got[groupID] {
		t.Fatalf("expected e1 to belong to %s", groupID)
	}
	got = state.EmployeeGroupIDs["e2"]
	if got == nil || !got[groupID] {
		t.Fatalf("expected e2 to belong to %s", groupID)
	}
}

func TestPhaseOneStep_ExclusiveRuleRespectsEmployeeScope(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
		{EmployeeID: "e2", ShiftID: "day", Date: "2026-03-23"},
	}
	employeeID := "e1"
	state.EffectiveRules = []rule.Rule{{
		Category:  rule.CategoryConstraint,
		SubType:   rule.SubTypeForbid,
		IsEnabled: true,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
		ApplyScopes: []rule.RuleApplyScope{{
			ScopeType: rule.ScopeTypeEmployee,
			ScopeID:   &employeeID,
		}},
	}}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	want := []string{"e2", "e3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}
}

func TestFullValidationStep_SourceRuleRespectsGroupScope(t *testing.T) {
	groupID := "grp-1"
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", nil)
	state.EmployeeGroupIDs["e1"] = map[string]bool{groupID: true}
	state.Assignments = []Assignment{
		{ID: "a1", EmployeeID: "e1", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
		{ID: "a2", EmployeeID: "e2", ShiftID: "night", Date: "2026-03-23", Source: SourceFill},
	}
	state.EffectiveRules = []rule.Rule{{
		ID:        "r-source-scope",
		Name:      "指定组来源规则",
		Category:  rule.CategoryDependency,
		SubType:   rule.SubTypeSource,
		IsEnabled: true,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
		ApplyScopes: []rule.RuleApplyScope{{
			ScopeType: rule.ScopeTypeGroup,
			ScopeID:   &groupID,
		}},
	}}

	if err := (&FullValidationStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(state.Violations))
	}
	if state.Violations[0].EmployeeID != "e1" {
		t.Fatalf("expected violation for e1, got %s", state.Violations[0].EmployeeID)
	}
}

func TestPhaseOneStep_ExclusiveRuleSkipsLowerPriorityConflictInCurrentShiftContext(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night", "oncall"},
		Requirements: map[string]map[string]int{
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-23", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	state.Assignments = []Assignment{{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"}}
	state.EffectiveRules = []rule.Rule{
		{
			ID:        "rule-high",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeForbid,
			IsEnabled: true,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "oncall", Role: rule.AssociationRoleObject},
			},
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
		{
			ID:        "rule-low",
			Priority:  20,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeForbid,
			IsEnabled: true,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
	}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	want := []string{"e1", "e2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}
}

func TestPhaseOneStep_RequiredTogetherSkipsLowerPriorityConflictInCurrentShiftContext(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"prep", "day", "night"},
		Requirements: map[string]map[string]int{
			"prep":  {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("sch-1", "org-1", "", "2026-03-22", "2026-03-23", "user-1", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["prep|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{
		{
			ID:             "rule-high",
			Priority:       10,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "prep", Role: rule.AssociationRoleObject},
			},
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
		{
			ID:             "rule-low",
			Priority:       20,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
	}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	want := []string{"e1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("night candidates mismatch, want %v, got %v", want, got)
	}
}
