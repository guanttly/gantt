package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"gantt-saas/internal/core/rule"
)

// MinRestChecker 最小休息天数检查器。
type MinRestChecker struct{}

func (c *MinRestChecker) Type() string { return rule.SubTypeMinRest }

func (c *MinRestChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	var cfg rule.MinRestConfig
	if err := json.Unmarshal(r.Config, &cfg); err != nil {
		return CheckResult{Pass: false, Reason: "config parse error"}
	}

	var dates []time.Time
	for _, a := range checkCtx.Assignments {
		if a.EmployeeID == checkCtx.EmployeeID {
			dates = append(dates, a.Date)
		}
	}
	dates = append(dates, checkCtx.Date)

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	consecutive := 1
	maxConsecutive := 7 - cfg.Days
	if maxConsecutive <= 0 {
		maxConsecutive = 1
	}

	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1]).Hours() / 24
		if diff <= 1 {
			consecutive++
		} else {
			consecutive = 1
		}
		if consecutive > maxConsecutive {
			return CheckResult{
				Pass:   false,
				Reason: fmt.Sprintf("consecutive %d days exceeds limit (min rest %d days/week)", consecutive, cfg.Days),
			}
		}
	}

	return CheckResult{Pass: true}
}
