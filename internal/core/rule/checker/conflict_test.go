package checker

import (
	"testing"
	"time"

	"gantt-saas/internal/core/rule"
)

func TestValidateAll_SkipsLowerPriorityConflictForMatchingEmployeeScope(t *testing.T) {
	employeeID := "emp-1"
	rules := []rule.Rule{
		{
			ID:        "rule-global",
			Name:      "global limit",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 5, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-global", RuleID2: "rule-scoped", ConflictType: "exclusive"}},
		},
		{
			ID:        "rule-scoped",
			Name:      "scoped limit",
			Priority:  20,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 0, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-global", RuleID2: "rule-scoped", ConflictType: "exclusive"}},
			ApplyScopes: []rule.RuleApplyScope{{
				ScopeType: rule.ScopeTypeEmployee,
				ScopeID:   &employeeID,
			}},
		},
	}

	results := ValidateAll(t.Context(), rules, &CheckContext{
		EmployeeID:  employeeID,
		ShiftID:     "day",
		Date:        time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		Assignments: []Assignment{},
	})

	if len(results) != 1 {
		t.Fatalf("expected only higher-priority rule to remain active, got %d results", len(results))
	}
	if results[0].RuleID != "rule-global" {
		t.Fatalf("expected global rule to stay active, got %s", results[0].RuleID)
	}
	if HasViolation(results) {
		t.Fatalf("expected no violations after skipping lower-priority conflicting rule")
	}
}

func TestValidateAll_SkipsLowerPriorityConflictForMatchingGroupScope(t *testing.T) {
	groupID := "grp-1"
	rules := []rule.Rule{
		{
			ID:        "rule-high",
			Name:      "high",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 5, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
			ApplyScopes: []rule.RuleApplyScope{{
				ScopeType: rule.ScopeTypeGroup,
				ScopeID:   &groupID,
			}},
		},
		{
			ID:        "rule-low",
			Name:      "low",
			Priority:  20,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 0, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
			ApplyScopes: []rule.RuleApplyScope{{
				ScopeType: rule.ScopeTypeGroup,
				ScopeID:   &groupID,
			}},
		},
	}

	results := ValidateAll(t.Context(), rules, &CheckContext{
		EmployeeID:       "emp-1",
		EmployeeGroupIDs: map[string]bool{groupID: true},
		ShiftID:          "day",
		Date:             time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		Assignments:      []Assignment{},
	})

	if len(results) != 1 {
		t.Fatalf("expected only higher-priority group rule to remain active, got %d results", len(results))
	}
	if results[0].RuleID != "rule-high" {
		t.Fatalf("expected high rule to stay active, got %s", results[0].RuleID)
	}
}

func TestValidateAll_DoesNotSkipConflictWhenTimeContextDiffers(t *testing.T) {
	minusOne := -1
	rules := []rule.Rule{
		{
			ID:        "rule-high",
			Name:      "high",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			TimeScope: rule.TimeScopeSameDay,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 5, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
		{
			ID:             "rule-low",
			Name:           "low",
			Priority:       20,
			Category:       rule.CategoryConstraint,
			SubType:        rule.SubTypeLimit,
			IsEnabled:      true,
			TimeScope:      rule.TimeScopeSameDay,
			TimeOffsetDays: &minusOne,
			Config:         mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 0, Period: "week"}),
			Conflicts:      []rule.RuleConflict{{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}},
		},
	}

	results := ValidateAll(t.Context(), rules, &CheckContext{
		EmployeeID:  "emp-1",
		ShiftID:     "day",
		Date:        time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		Assignments: []Assignment{},
	})

	if len(results) != 2 {
		t.Fatalf("expected both rules to remain active for different time context, got %d results", len(results))
	}
	if !HasViolation(results) {
		t.Fatalf("expected lower-priority rule violation to remain when time context differs")
	}
}

func TestValidateAll_DoesNotSkipConflictWhenHigherRuleTargetsDifferentShift(t *testing.T) {
	rules := []rule.Rule{
		{
			ID:        "rule-day",
			Name:      "day",
			Priority:  10,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "day", Max: 5, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-day", RuleID2: "rule-night", ConflictType: "exclusive"}},
		},
		{
			ID:        "rule-night",
			Name:      "night",
			Priority:  20,
			Category:  rule.CategoryConstraint,
			SubType:   rule.SubTypeLimit,
			IsEnabled: true,
			Config:    mustJSON(t, rule.MaxCountConfig{Type: "max_count", ShiftID: "night", Max: 0, Period: "week"}),
			Conflicts: []rule.RuleConflict{{RuleID1: "rule-day", RuleID2: "rule-night", ConflictType: "exclusive"}},
		},
	}

	results := ValidateAll(t.Context(), rules, &CheckContext{
		EmployeeID:  "emp-1",
		ShiftID:     "night",
		Date:        time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		Assignments: []Assignment{},
	})

	if len(results) != 2 {
		t.Fatalf("expected both rules checked because higher rule is not active on night shift, got %d results", len(results))
	}
	if !HasViolation(results) {
		t.Fatalf("expected night rule violation to remain when higher rule targets a different shift")
	}
}
