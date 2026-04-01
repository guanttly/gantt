package checker

import (
	"context"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// ExclusiveChecker 排他班次检查器。
type ExclusiveChecker struct{}

func (c *ExclusiveChecker) Type() string { return rule.SubTypeForbid }

func (c *ExclusiveChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	relatedShiftIDs, offsetDays, matched := r.ExclusiveSemantics().CounterpartShiftIDs(checkCtx.ShiftID)
	if !matched || len(relatedShiftIDs) == 0 {
		return CheckResult{Pass: true}
	}
	relatedSet := make(map[string]bool, len(relatedShiftIDs))
	for _, sid := range relatedShiftIDs {
		relatedSet[sid] = true
	}
	relatedDate := checkCtx.Date.AddDate(0, 0, offsetDays)

	for _, a := range checkCtx.Assignments {
		if a.EmployeeID != checkCtx.EmployeeID {
			continue
		}
		if !a.Date.Equal(relatedDate) {
			continue
		}
		if relatedSet[a.ShiftID] {
			return CheckResult{
				Pass:   false,
				Reason: fmt.Sprintf("shift %s conflicts with %s", checkCtx.ShiftID, a.ShiftID),
			}
		}
	}

	return CheckResult{Pass: true}
}
