package checker

import (
	"context"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// SourceChecker 人员来源检查器。
type SourceChecker struct{}

func (c *SourceChecker) Type() string { return rule.SubTypeSource }

func (c *SourceChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	sourceShiftIDs, offsetDays, matched := r.SourceSemantics().SourceShiftIDsForTarget(checkCtx.ShiftID)
	if !matched {
		return CheckResult{Pass: true}
	}
	sourceSet := make(map[string]bool, len(sourceShiftIDs))
	for _, sid := range sourceShiftIDs {
		sourceSet[sid] = true
	}
	sourceDate := checkCtx.Date.AddDate(0, 0, offsetDays)

	hasSource := false
	for _, a := range checkCtx.Assignments {
		if a.EmployeeID == checkCtx.EmployeeID && a.Date.Equal(sourceDate) && sourceSet[a.ShiftID] {
			hasSource = true
			break
		}
	}

	if !hasSource {
		return CheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("employee %s has no source shift for target %s", checkCtx.EmployeeID, checkCtx.ShiftID),
		}
	}

	return CheckResult{Pass: true}
}
