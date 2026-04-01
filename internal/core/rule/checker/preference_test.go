package checker

import (
	"encoding/json"
	"testing"
	"time"

	"gantt-saas/internal/core/rule"
)

func TestPreferenceScorer_RespectsApplyScopes(t *testing.T) {
	groupID := "grp-1"
	rules := []rule.Rule{{
		Category:  rule.CategoryPreference,
		SubType:   rule.SubTypePrefer,
		IsEnabled: true,
		Config: mustJSON(t, rule.PreferEmployeeConfig{
			Type:       "prefer_employee",
			EmployeeID: "e1",
			ShiftID:    "day",
			Weight:     80,
		}),
		ApplyScopes: []rule.RuleApplyScope{{
			ScopeType: rule.ScopeTypeGroup,
			ScopeID:   &groupID,
		}},
	}}

	scorer := &PreferenceScorer{}
	if score := scorer.Score(rules, "e1", nil, "day", time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)); score != 0 {
		t.Fatalf("expected score 0 without group membership, got %d", score)
	}
	if score := scorer.Score(rules, "e1", map[string]bool{groupID: true}, "day", time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)); score != 80 {
		t.Fatalf("expected score 80 with group membership, got %d", score)
	}
}

func TestPreferenceScorer_SkipsLowerPriorityConflictInCurrentShiftContext(t *testing.T) {
	conflict := rule.RuleConflict{RuleID1: "rule-high", RuleID2: "rule-low", ConflictType: "exclusive"}
	rules := []rule.Rule{
		{
			ID:        "rule-high",
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
			ID:        "rule-low",
			Priority:  20,
			Category:  rule.CategoryPreference,
			SubType:   rule.SubTypePrefer,
			IsEnabled: true,
			Config: mustJSON(t, rule.PreferEmployeeConfig{
				Type:       "prefer_employee",
				EmployeeID: "e1",
				ShiftID:    "day",
				Weight:     100,
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
	}

	scorer := &PreferenceScorer{}
	if score := scorer.Score(rules, "e1", nil, "day", time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)); score != 20 {
		t.Fatalf("expected score 20 after conflict suppression, got %d", score)
	}
}

func TestPreferenceScorer_KeepsDifferentShiftConflictForCurrentShift(t *testing.T) {
	conflict := rule.RuleConflict{RuleID1: "rule-night", RuleID2: "rule-day", ConflictType: "exclusive"}
	rules := []rule.Rule{
		{
			ID:        "rule-night",
			Priority:  10,
			Category:  rule.CategoryPreference,
			SubType:   rule.SubTypePrefer,
			IsEnabled: true,
			Config: mustJSON(t, rule.PreferEmployeeConfig{
				Type:       "prefer_employee",
				EmployeeID: "e1",
				ShiftID:    "night",
				Weight:     20,
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
		{
			ID:        "rule-day",
			Priority:  20,
			Category:  rule.CategoryPreference,
			SubType:   rule.SubTypePrefer,
			IsEnabled: true,
			Config: mustJSON(t, rule.PreferEmployeeConfig{
				Type:       "prefer_employee",
				EmployeeID: "e1",
				ShiftID:    "day",
				Weight:     80,
			}),
			Conflicts: []rule.RuleConflict{conflict},
		},
	}

	scorer := &PreferenceScorer{}
	if score := scorer.Score(rules, "e1", nil, "day", time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)); score != 80 {
		t.Fatalf("expected day score 80 when higher-priority night rule targets another shift, got %d", score)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
