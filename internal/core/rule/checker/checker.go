package checker

import (
	"context"
	"time"

	"gantt-saas/internal/core/rule"
)

// Assignment 已有排班记录（供检查器使用）。
type Assignment struct {
	EmployeeID string
	ShiftID    string
	Date       time.Time
}

// CheckContext 约束检查上下文。
type CheckContext struct {
	EmployeeID  string
	ShiftID     string
	Date        time.Time
	Assignments []Assignment
	Candidates  []string
}

// CheckResult 检查结果。
type CheckResult struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Pass     bool   `json:"pass"`
	Reason   string `json:"reason,omitempty"`
}

// Checker 约束检查器接口。
type Checker interface {
	Type() string
	Check(ctx context.Context, r rule.Rule, checkCtx *CheckContext) CheckResult
}

var registry = map[string]Checker{}

// Register 注册检查器。
func Register(c Checker) {
	registry[c.Type()] = c
}

// Get 获取指定子类型的检查器。
func Get(subType string) Checker {
	return registry[subType]
}

// ValidateAll 批量校验。
func ValidateAll(ctx context.Context, rules []rule.Rule, checkCtx *CheckContext) []CheckResult {
	results := make([]CheckResult, 0, len(rules))
	for _, r := range rules {
		if !r.IsEnabled {
			continue
		}
		c := Get(r.SubType)
		if c == nil {
			continue
		}
		result := c.Check(ctx, r, checkCtx)
		result.RuleID = r.ID
		result.RuleName = r.Name
		results = append(results, result)
	}
	return results
}

// HasViolation 检查结果中是否存在违规。
func HasViolation(results []CheckResult) bool {
	for _, r := range results {
		if !r.Pass {
			return true
		}
	}
	return false
}

// GetViolations 获取所有违规结果。
func GetViolations(results []CheckResult) []CheckResult {
	var violations []CheckResult
	for _, r := range results {
		if !r.Pass {
			violations = append(violations, r)
		}
	}
	return violations
}

func init() {
	Register(&ExclusiveChecker{})
	Register(&MaxCountChecker{})
	Register(&MinRestChecker{})
	Register(&RequiredChecker{})
	Register(&SourceChecker{})
}
