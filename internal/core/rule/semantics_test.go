package rule

import (
	"reflect"
	"testing"
)

func TestOrderRulesForExecution_SortsByDependencies(t *testing.T) {
	rules := []Rule{
		{ID: "rule-b", Name: "B", Priority: 100, Dependencies: []RuleDependency{{DependentRuleID: "rule-b", DependentOnRuleID: "rule-a"}}},
		{ID: "rule-a", Name: "A", Priority: 200},
		{ID: "rule-c", Name: "C", Priority: 50},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-c", "rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected rule order, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_FallsBackOnDependencyCycle(t *testing.T) {
	rules := []Rule{
		{ID: "rule-a", Name: "A", Priority: 10, Dependencies: []RuleDependency{{DependentRuleID: "rule-a", DependentOnRuleID: "rule-b"}}},
		{ID: "rule-b", Name: "B", Priority: 20, Dependencies: []RuleDependency{{DependentRuleID: "rule-b", DependentOnRuleID: "rule-a"}}},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fallback order, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_ResolvesConflictsByPriority(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive", ResolutionPriority: 0}
	rules := []Rule{
		{ID: "rule-b", Name: "B", Priority: 100, Conflicts: []RuleConflict{conflict}},
		{ID: "rule-a", Name: "A", Priority: 10, Conflicts: []RuleConflict{conflict}},
		{ID: "rule-c", Name: "C", Priority: 50},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected conflict-resolved rule order, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingRulesWithDisjointShiftContext(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive"}
	rules := []Rule{
		{
			ID:        "rule-a",
			Name:      "A",
			Priority:  10,
			SubType:   SubTypeMust,
			Conflicts: []RuleConflict{conflict},
			Associations: []RuleAssociation{
				{TargetType: TargetTypeShift, TargetID: "night", Role: AssociationRoleSubject},
			},
		},
		{
			ID:        "rule-b",
			Name:      "B",
			Priority:  20,
			SubType:   SubTypeMust,
			Conflicts: []RuleConflict{conflict},
			Associations: []RuleAssociation{
				{TargetType: TargetTypeShift, TargetID: "day", Role: AssociationRoleSubject},
			},
		},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected shift-context conflict result, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingRulesWithDisjointEmployeeScopes(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive"}
	e1 := "emp-1"
	e2 := "emp-2"
	rules := []Rule{
		{
			ID:        "rule-a",
			Name:      "A",
			Priority:  10,
			Conflicts: []RuleConflict{conflict},
			ApplyScopes: []RuleApplyScope{{
				ScopeType: ScopeTypeEmployee,
				ScopeID:   &e1,
			}},
		},
		{
			ID:        "rule-b",
			Name:      "B",
			Priority:  20,
			Conflicts: []RuleConflict{conflict},
			ApplyScopes: []RuleApplyScope{{
				ScopeType: ScopeTypeEmployee,
				ScopeID:   &e2,
			}},
		},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected employee-scope conflict result, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingRulesWithDisjointGroupScopes(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive"}
	g1 := "grp-1"
	g2 := "grp-2"
	rules := []Rule{
		{
			ID:        "rule-a",
			Name:      "A",
			Priority:  10,
			Conflicts: []RuleConflict{conflict},
			ApplyScopes: []RuleApplyScope{{
				ScopeType: ScopeTypeGroup,
				ScopeID:   &g1,
			}},
		},
		{
			ID:        "rule-b",
			Name:      "B",
			Priority:  20,
			Conflicts: []RuleConflict{conflict},
			ApplyScopes: []RuleApplyScope{{
				ScopeType: ScopeTypeGroup,
				ScopeID:   &g2,
			}},
		},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected group-scope conflict result, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingRulesWhenOnlyOneIsScoped(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive"}
	e1 := "emp-1"
	rules := []Rule{
		{ID: "rule-a", Name: "A", Priority: 10, Conflicts: []RuleConflict{conflict}},
		{
			ID:        "rule-b",
			Name:      "B",
			Priority:  20,
			Conflicts: []RuleConflict{conflict},
			ApplyScopes: []RuleApplyScope{{
				ScopeType: ScopeTypeEmployee,
				ScopeID:   &e1,
			}},
		},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected scoped-vs-global conflict result, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingRulesWithDifferentTimeContext(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-a", RuleID2: "rule-b", ConflictType: "exclusive"}
	minusOne := -1
	rules := []Rule{
		{ID: "rule-a", Name: "A", Priority: 10, Conflicts: []RuleConflict{conflict}, TimeScope: TimeScopeSameDay},
		{ID: "rule-b", Name: "B", Priority: 20, Conflicts: []RuleConflict{conflict}, TimeScope: TimeScopeSameDay, TimeOffsetDays: &minusOne},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-a", "rule-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected time-context conflict result, want %v, got %v", want, got)
	}
}

func TestOrderRulesForExecution_KeepsConflictingPreferenceRulesWithDifferentShiftTargets(t *testing.T) {
	conflict := RuleConflict{RuleID1: "rule-night", RuleID2: "rule-day", ConflictType: "exclusive"}
	rules := []Rule{
		{
			ID:        "rule-night",
			Name:      "night",
			Priority:  10,
			SubType:   SubTypePrefer,
			Conflicts: []RuleConflict{conflict},
			Config:    []byte(`{"type":"prefer_employee","employee_id":"e1","shift_id":"night","weight":20}`),
		},
		{
			ID:        "rule-day",
			Name:      "day",
			Priority:  20,
			SubType:   SubTypePrefer,
			Conflicts: []RuleConflict{conflict},
			Config:    []byte(`{"type":"prefer_employee","employee_id":"e1","shift_id":"day","weight":80}`),
		},
	}

	ordered := OrderRulesForExecution(rules)
	got := make([]string, 0, len(ordered))
	for _, currentRule := range ordered {
		got = append(got, currentRule.ID)
	}
	want := []string{"rule-night", "rule-day"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected preference shift-context conflict result, want %v, got %v", want, got)
	}
}
