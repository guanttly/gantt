package checker

import (
	"context"
	"encoding/json"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// ExclusiveChecker 排他班次检查器。
type ExclusiveChecker struct{}

func (c *ExclusiveChecker) Type() string { return rule.SubTypeForbid }

func (c *ExclusiveChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	var cfg rule.ExclusiveShiftsConfig
	if err := json.Unmarshal(r.Config, &cfg); err != nil {
		return CheckResult{Pass: false, Reason: "config parse error"}
	}

	exclusiveSet := make(map[string]bool)
	for _, sid := range cfg.ShiftIDs {
		exclusiveSet[sid] = true
	}

	if !exclusiveSet[checkCtx.ShiftID] {
		return CheckResult{Pass: true}
	}

	for _, a := range checkCtx.Assignments {
		if a.EmployeeID != checkCtx.EmployeeID {
			continue
		}
		if cfg.Scope == "same_day" && !a.Date.Equal(checkCtx.Date) {
			continue
		}
		if exclusiveSet[a.ShiftID] && a.ShiftID != checkCtx.ShiftID {
			return CheckResult{
				Pass:   false,
				Reason: fmt.Sprintf("shift %s conflicts with %s", checkCtx.ShiftID, a.ShiftID),
			}
		}
	}

	return CheckResult{Pass: true}
}
