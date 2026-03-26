package checker

import (
	"context"
	"encoding/json"
	"fmt"

	"gantt-saas/internal/core/rule"
)

// RequiredChecker 必须同时检查器。
type RequiredChecker struct{}

func (c *RequiredChecker) Type() string { return rule.SubTypeMust }

func (c *RequiredChecker) Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult {
	var cfg rule.RequiredTogetherConfig
	if err := json.Unmarshal(r.Config, &cfg); err != nil {
		return CheckResult{Pass: false, Reason: "config parse error"}
	}

	if cfg.ShiftID != checkCtx.ShiftID {
		return CheckResult{Pass: true}
	}

	isInGroup := false
	for _, eid := range cfg.EmployeeIDs {
		if eid == checkCtx.EmployeeID {
			isInGroup = true
			break
		}
	}
	if !isInGroup {
		return CheckResult{Pass: true}
	}

	assignedSet := make(map[string]bool)
	for _, a := range checkCtx.Assignments {
		if a.ShiftID == cfg.ShiftID && a.Date.Equal(checkCtx.Date) {
			assignedSet[a.EmployeeID] = true
		}
	}
	assignedSet[checkCtx.EmployeeID] = true

	for _, eid := range cfg.EmployeeIDs {
		if !assignedSet[eid] {
			inCandidates := false
			for _, cid := range checkCtx.Candidates {
				if cid == eid {
					inCandidates = true
					break
				}
			}
			if !inCandidates {
				return CheckResult{
					Pass:   false,
					Reason: fmt.Sprintf("employee %s must be scheduled with %v for shift %s", checkCtx.EmployeeID, cfg.EmployeeIDs, cfg.ShiftID),
				}
			}
		}
	}

	return CheckResult{Pass: true}
}
