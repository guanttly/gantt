package step

import (
	"context"
	"testing"

	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/shift"
)

// ── ScheduleState 辅助方法测试 ────────────────────────

func TestScheduleState_IsOccupied(t *testing.T) {
	state := NewScheduleState("s1", "org", "", "2026-03-23", "2026-03-29", "u", nil)
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
	}

	if !state.IsOccupied("e1", "2026-03-23") {
		t.Error("e1 should be occupied on 2026-03-23")
	}
	if state.IsOccupied("e1", "2026-03-24") {
		t.Error("e1 should not be occupied on 2026-03-24")
	}
}

func TestScheduleState_CountAssigned(t *testing.T) {
	state := NewScheduleState("s1", "org", "", "2026-03-23", "2026-03-29", "u", nil)
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
		{EmployeeID: "e2", ShiftID: "day", Date: "2026-03-23"},
		{EmployeeID: "e3", ShiftID: "night", Date: "2026-03-23"},
	}

	if got := state.CountAssigned("day", "2026-03-23"); got != 2 {
		t.Errorf("expected 2 day assignments, got %d", got)
	}
	if got := state.CountAssigned("night", "2026-03-23"); got != 1 {
		t.Errorf("expected 1 night assignment, got %d", got)
	}
}

// ── PhaseTwoStep 测试 ────────────────────────

func makeShifts(ids ...string) []shift.Shift {
	out := make([]shift.Shift, len(ids))
	for i, id := range ids {
		out[i] = shift.Shift{ID: id, Name: id}
	}
	return out
}

func TestPhaseTwoStep_FillsRemainingSlots(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day"},
		Requirements: map[string]map[string]int{
			"day": {"2026-03-23": 2},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-23", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("day")
	state.Candidates["day|2026-03-23"] = []string{"e1", "e2", "e3"}

	s := &PhaseTwoStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := state.CountAssigned("day", "2026-03-23"); got != 2 {
		t.Errorf("expected 2 fill assignments, got %d", got)
	}
	for _, a := range state.Assignments {
		if a.Source != SourceFill {
			t.Errorf("expected source=%q, got %q", SourceFill, a.Source)
		}
	}
}

func TestPhaseTwoStep_SkipsAlreadySatisfied(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day"},
		Requirements: map[string]map[string]int{
			"day": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-23", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("day")
	state.Candidates["day|2026-03-23"] = []string{"e1", "e2"}

	// 预先放一个，需求已满足
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23", Source: SourceFixed},
	}

	s := &PhaseTwoStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 1 {
		t.Errorf("expected 1 assignment (unchanged), got %d", got)
	}
}

func TestPhaseOneStep_KeepsRequiredTogetherCandidatesWhenDependencyCanBeBackfilled(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-22", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{{
		IsEnabled:      true,
		Category:       rule.CategoryConstraint,
		SubType:        rule.SubTypeMust,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	if len(got) != 1 || got[0] != "e1" {
		t.Fatalf("expected only plannable candidate e1, got %v", got)
	}
}

func TestPhaseTwoStep_BackfillsRequiredTogetherDependency(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-22", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{{
		ID:             "r-must",
		IsEnabled:      true,
		Category:       rule.CategoryConstraint,
		SubType:        rule.SubTypeMust,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 2 {
		t.Fatalf("expected linked fill to create 2 assignments, got %d", got)
	}

	var hasRequired bool
	var hasTarget bool
	for _, assignment := range state.Assignments {
		if assignment.EmployeeID != "e1" {
			continue
		}
		if assignment.ShiftID == "day" && assignment.Date == "2026-03-22" {
			hasRequired = assignment.Source == SourceRule
		}
		if assignment.ShiftID == "night" && assignment.Date == "2026-03-23" {
			hasTarget = assignment.Source == SourceFill
		}
	}
	if !hasRequired {
		t.Fatalf("expected linked required assignment on day shift")
	}
	if !hasTarget {
		t.Fatalf("expected fill assignment on target night shift")
	}
}

func TestPhaseOneStep_KeepsRequiredTogetherCandidatesWhenDependencyChainCanBeBackfilled(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"prep", "day", "night"},
		Requirements: map[string]map[string]int{
			"prep":  {"2026-03-21": 1},
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-21", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["prep|2026-03-21"] = []string{"e1"}
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{
		{
			ID:             "r-night-day",
			IsEnabled:      true,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
		},
		{
			ID:             "r-day-prep",
			IsEnabled:      true,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "prep", Role: rule.AssociationRoleObject},
			},
		},
	}

	if err := (&PhaseOneStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.Candidates["night|2026-03-23"]
	if len(got) != 1 || got[0] != "e1" {
		t.Fatalf("expected only chain-plannable candidate e1, got %v", got)
	}
}

func TestPhaseTwoStep_BackfillsRequiredTogetherDependencyChain(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"prep", "day", "night"},
		Requirements: map[string]map[string]int{
			"prep":  {"2026-03-21": 1},
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-21", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["prep|2026-03-21"] = []string{"e1"}
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{
		{
			ID:             "r-night-day",
			IsEnabled:      true,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
		},
		{
			ID:             "r-day-prep",
			IsEnabled:      true,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "prep", Role: rule.AssociationRoleObject},
			},
		},
	}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 3 {
		t.Fatalf("expected chained fill to create 3 assignments, got %d", got)
	}

	expected := map[string]string{
		"prep|2026-03-21":  SourceRule,
		"day|2026-03-22":   SourceRule,
		"night|2026-03-23": SourceFill,
	}
	for _, assignment := range state.Assignments {
		if assignment.EmployeeID != "e1" {
			continue
		}
		key := assignment.ShiftID + "|" + assignment.Date
		wantSource, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected chained assignment %s", key)
		}
		if assignment.Source != wantSource {
			t.Fatalf("expected %s source %q, got %q", key, wantSource, assignment.Source)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing chained assignments: %v", expected)
	}
}

func TestPhaseTwoStep_SkipsLinkedPlanThatViolatesOtherRules(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"prep", "day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-21", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["day|2026-03-22"] = []string{"e1", "e2"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2"}
	state.Assignments = []Assignment{{
		ID:         "a-prep",
		EmployeeID: "e2",
		ShiftID:    "prep",
		Date:       "2026-03-21",
		Source:     SourceRule,
	}}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{
		{
			ID:             "r-must",
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
		},
		{
			ID:             "r-source",
			Category:       rule.CategoryDependency,
			SubType:        rule.SubTypeSource,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "prep", Role: rule.AssociationRoleObject},
			},
		},
	}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 3 {
		t.Fatalf("expected existing prep plus valid linked pair, got %d assignments", got)
	}

	expected := map[string]string{
		"prep|2026-03-21":  "e2",
		"day|2026-03-22":   "e2",
		"night|2026-03-23": "e2",
	}
	for _, assignment := range state.Assignments {
		key := assignment.ShiftID + "|" + assignment.Date
		wantEmployee, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected assignment %s for %s", key, assignment.EmployeeID)
		}
		if assignment.EmployeeID != wantEmployee {
			t.Fatalf("expected %s assigned to %s, got %s", key, wantEmployee, assignment.EmployeeID)
		}
	}
	if len(state.Violations) != 0 {
		t.Fatalf("expected no violations recorded during phase two candidate skipping, got %d", len(state.Violations))
	}
}

func TestPhaseTwoStep_BackfillsRequiredTogetherCycleToTarget(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-22", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("night")
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1"}
	minusOne := -1
	plusOne := 1
	state.EffectiveRules = []rule.Rule{
		{
			ID:             "r-night-day",
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
			},
		},
		{
			ID:             "r-day-night",
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &plusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleObject},
			},
		},
	}

	if err := (&PhaseTwoStep{}).Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(state.Assignments); got != 2 {
		t.Fatalf("expected cyclic linked fill to create 2 assignments, got %d", got)
	}

	expected := map[string]string{
		"day|2026-03-22":   SourceRule,
		"night|2026-03-23": SourceFill,
	}
	for _, assignment := range state.Assignments {
		key := assignment.ShiftID + "|" + assignment.Date
		wantSource, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected assignment %s", key)
		}
		if assignment.EmployeeID != "e1" {
			t.Fatalf("expected %s assigned to e1, got %s", key, assignment.EmployeeID)
		}
		if assignment.Source != wantSource {
			t.Fatalf("expected %s source %q, got %q", key, wantSource, assignment.Source)
		}
	}
}

// ── PhaseOneStep 排他规则测试 ────────────────────────

func TestPhaseOneStep_FiltersExclusiveShiftCandidates(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-23": 2},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-23", "2026-03-23", "u", config)
	state.ShiftOrder = makeShifts("day", "night")

	state.Candidates["day|2026-03-23"] = []string{"e1", "e2", "e3"}
	state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}

	// e1 已经被排到 day，如果有排他规则，night 候选人应排除 e1
	state.Assignments = []Assignment{
		{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
	}

	// PhaseOneStep 本身不会自己添加 assignment，它只过滤候选人
	s := &PhaseOneStep{}
	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 如果没有排他规则配置，候选人不会被过滤
	nightCandidates := state.Candidates["night|2026-03-23"]
	if len(nightCandidates) != 3 {
		t.Errorf("without exclusive rules, all candidates should remain; got %d", len(nightCandidates))
	}
}

// ── NotifyWSStep 测试 ────────────────────────

type fakeBroadcaster struct {
	msgs []any
}

func (f *fakeBroadcaster) BroadcastToGroup(_ string, payload any) error {
	f.msgs = append(f.msgs, payload)
	return nil
}
func (f *fakeBroadcaster) BroadcastAll(payload any) error {
	f.msgs = append(f.msgs, payload)
	return nil
}

func TestNotifyWSStep_Broadcasts(t *testing.T) {
	fb := &fakeBroadcaster{}
	s := &NotifyWSStep{Broadcaster: fb}
	state := NewScheduleState("sch-99", "org", "", "2026-03-23", "2026-03-29", "u", nil)
	state.Assignments = make([]Assignment, 5)
	state.Violations = make([]Violation, 2)

	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fb.msgs) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(fb.msgs))
	}
}

func TestNotifyWSStep_NilBroadcasterOK(t *testing.T) {
	s := &NotifyWSStep{} // Broadcaster == nil
	state := NewScheduleState("sch-99", "org", "", "2026-03-23", "2026-03-29", "u", nil)

	if err := s.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
