package checker

import (
	"context"
	"encoding/json"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// SourceChecker 人员来源检查器。
type SourceChecker struct{}

func (c *SourceChecker) Type() string { return rule.SubTypeSource }

func (c *SourceChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	var cfg rule.StaffSourceConfig
	if err := json.Unmarshal(r.Config, &cfg); err != nil {
		return CheckResult{Pass: false, Reason: "config parse error"}
	}

	if cfg.TargetShiftID != checkCtx.ShiftID {
		return CheckResult{Pass: true}
	}

	hasSource := false
	for _, a := range checkCtx.Assignments {
		if a.EmployeeID == checkCtx.EmployeeID && a.ShiftID == cfg.SourceShiftID {
			hasSource = true
			break
		}
	}

	if !hasSource {
		return CheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("employee %s has no source shift %s for target %s", checkCtx.EmployeeID, cfg.SourceShiftID, cfg.TargetShiftID),
		}
	}

	return CheckResult{Pass: true}
}
