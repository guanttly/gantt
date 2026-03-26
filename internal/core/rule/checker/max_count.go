package checker

import (
	"context"
	"encoding/json"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// MaxCountChecker 最大次数检查器。
type MaxCountChecker struct{}

func (c *MaxCountChecker) Type() string { return rule.SubTypeLimit }

func (c *MaxCountChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	var cfg rule.MaxCountConfig
	if err := json.Unmarshal(r.Config, &cfg); err != nil {
		return CheckResult{Pass: false, Reason: "config parse error"}
	}

	if cfg.ShiftID != checkCtx.ShiftID {
		return CheckResult{Pass: true}
	}

	count := 0
	for _, a := range checkCtx.Assignments {
		if a.EmployeeID == checkCtx.EmployeeID && a.ShiftID == cfg.ShiftID {
			count++
		}
	}

	if count >= cfg.Max {
		return CheckResult{
			Pass:   false,
			Reason: fmt.Sprintf("shift %s reached max %d per %s", cfg.ShiftID, cfg.Max, cfg.Period),
		}
	}

	return CheckResult{Pass: true}
}
