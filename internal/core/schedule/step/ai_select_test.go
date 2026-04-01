package step

import (
	"context"
	"testing"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/core/rule"

	"go.uber.org/zap"
)

type fakeAIProvider struct {
	content string
}

func (p *fakeAIProvider) Chat(_ context.Context, _ ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: p.content}, nil
}

func (p *fakeAIProvider) ChatStream(_ context.Context, _ ai.ChatRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil
}

func (p *fakeAIProvider) Name() string { return "fake" }

func TestAISelectStep_BackfillsRequiredTogetherDependency(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-22", "2026-03-23", "u", config)
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1"}
	minusOne := -1
	state.EffectiveRules = []rule.Rule{{
		ID:             "r-must",
		Category:       rule.CategoryConstraint,
		SubType:        rule.SubTypeMust,
		IsEnabled:      true,
		TimeOffsetDays: &minusOne,
		Associations: []rule.RuleAssociation{
			{TargetType: rule.TargetTypeShift, TargetID: "night", Role: rule.AssociationRoleSubject},
			{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleObject},
		},
	}}

	step := &AISelectStep{
		Provider: &fakeAIProvider{content: `[{"employee_id":"e1","shift_id":"night","date":"2026-03-23"}]`},
		Logger:   zap.NewNop(),
	}
	if err := step.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != 2 {
		t.Fatalf("expected AI selection to create 2 assignments, got %d", len(state.Assignments))
	}

	expected := map[string]string{
		"day|2026-03-22":   SourceRule,
		"night|2026-03-23": SourceAI,
	}
	for _, assignment := range state.Assignments {
		key := assignment.ShiftID + "|" + assignment.Date
		wantSource, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected assignment %s", key)
		}
		if assignment.Source != wantSource {
			t.Fatalf("expected %s source %q, got %q", key, wantSource, assignment.Source)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing assignments: %v", expected)
	}
	if len(state.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(state.Violations))
	}
}

func TestAISelectStep_BackfillsRequiredTogetherDependencyChain(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"prep", "day", "night"},
		Requirements: map[string]map[string]int{
			"prep":  {"2026-03-21": 1},
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-21", "2026-03-23", "u", config)
	state.Candidates["prep|2026-03-21"] = []string{"e1"}
	state.Candidates["day|2026-03-22"] = []string{"e1"}
	state.Candidates["night|2026-03-23"] = []string{"e1"}
	minusOne := -1
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
			ID:             "r-day-prep",
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeMust,
			IsEnabled:      true,
			TimeOffsetDays: &minusOne,
			Associations: []rule.RuleAssociation{
				{TargetType: rule.TargetTypeShift, TargetID: "day", Role: rule.AssociationRoleSubject},
				{TargetType: rule.TargetTypeShift, TargetID: "prep", Role: rule.AssociationRoleObject},
			},
		},
	}

	step := &AISelectStep{
		Provider: &fakeAIProvider{content: `[{"employee_id":"e1","shift_id":"night","date":"2026-03-23"}]`},
		Logger:   zap.NewNop(),
	}
	if err := step.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != 3 {
		t.Fatalf("expected chained AI selection to create 3 assignments, got %d", len(state.Assignments))
	}

	expected := map[string]string{
		"prep|2026-03-21":  SourceRule,
		"day|2026-03-22":   SourceRule,
		"night|2026-03-23": SourceAI,
	}
	for _, assignment := range state.Assignments {
		key := assignment.ShiftID + "|" + assignment.Date
		wantSource, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected assignment %s", key)
		}
		if assignment.Source != wantSource {
			t.Fatalf("expected %s source %q, got %q", key, wantSource, assignment.Source)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing assignments: %v", expected)
	}
	if len(state.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(state.Violations))
	}
}

func TestAISelectStep_BackfillsRequiredTogetherCycleToTarget(t *testing.T) {
	config := &ScheduleConfig{
		ShiftIDs: []string{"day", "night"},
		Requirements: map[string]map[string]int{
			"day":   {"2026-03-22": 1},
			"night": {"2026-03-23": 1},
		},
	}
	state := NewScheduleState("s1", "org", "", "2026-03-22", "2026-03-23", "u", config)
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

	step := &AISelectStep{
		Provider: &fakeAIProvider{content: `[{"employee_id":"e1","shift_id":"night","date":"2026-03-23"}]`},
		Logger:   zap.NewNop(),
	}
	if err := step.Execute(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Assignments) != 2 {
		t.Fatalf("expected cyclic AI selection to create 2 assignments, got %d", len(state.Assignments))
	}

	expected := map[string]string{
		"day|2026-03-22":   SourceRule,
		"night|2026-03-23": SourceAI,
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
	if len(state.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(state.Violations))
	}
}
