package step

import (
	"reflect"
	"testing"

	"gantt-saas/internal/core/rule"
)

func TestSortShiftOrderByRuleDependencies_UsesSourceAndOrderRules(t *testing.T) {
	shiftOrder := makeShifts("night", "report", "day")
	rules := []rule.Rule{
		{
			SubType: rule.SubTypeSource,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
		},
		{
			SubType: rule.SubTypeOrder,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "report", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleObject},
			},
		},
	}

	sorted := sortShiftOrderByRuleDependencies(shiftOrder, rules)
	got := make([]string, 0, len(sorted))
	for _, sh := range sorted {
		got = append(got, sh.ID)
	}
	want := []string{"day", "night", "report"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected shift order, want %v, got %v", want, got)
	}
}

func TestSortShiftOrderByRuleDependencies_FallsBackOnCycle(t *testing.T) {
	shiftOrder := makeShifts("a", "b")
	rules := []rule.Rule{
		{
			SubType: rule.SubTypeOrder,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "a", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "b", Role: rule.AssociationRoleObject},
			},
		},
		{
			SubType: rule.SubTypeOrder,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "b", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "a", Role: rule.AssociationRoleObject},
			},
		},
	}

	sorted := sortShiftOrderByRuleDependencies(shiftOrder, rules)
	got := make([]string, 0, len(sorted))
	for _, sh := range sorted {
		got = append(got, sh.ID)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fallback order, want %v, got %v", want, got)
	}
}

func TestSortShiftOrderByRuleDependencies_UsesRequiredTogetherRules(t *testing.T) {
	shiftOrder := makeShifts("target", "required")
	rules := []rule.Rule{
		{
			SubType: rule.SubTypeMust,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "target", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "required", Role: rule.AssociationRoleObject},
			},
		},
	}

	sorted := sortShiftOrderByRuleDependencies(shiftOrder, rules)
	got := make([]string, 0, len(sorted))
	for _, sh := range sorted {
		got = append(got, sh.ID)
	}
	want := []string{"required", "target"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected required_together order, want %v, got %v", want, got)
	}
}
