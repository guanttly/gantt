package checker

import (
	"encoding/json"
	"time"

	"gantt-saas/internal/core/rule"
)

// PreferenceScorer 偏好评分器。
type PreferenceScorer struct{}

// Score 计算偏好得分。
func (s *PreferenceScorer) Score(rules []rule.Rule, employeeID string, employeeGroupIDs map[string]bool, shiftID string, date time.Time) int {
	score := 0
	for _, r := range rule.ActiveRulesForShiftContext(rules, employeeID, employeeGroupIDs, shiftID) {
		if r.Category != rule.CategoryPreference {
			continue
		}
		var cfg rule.PreferEmployeeConfig
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			continue
		}
		if cfg.EmployeeID == employeeID && cfg.ShiftID == shiftID {
			score += cfg.Weight
		}
	}
	return score
}
