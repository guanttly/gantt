package pipeline

import (
"context"
"time"

"gantt-saas/internal/core/rule/checker"
"gantt-saas/internal/core/schedule/step"
)

// ValidationRunner 独立的校验执行器（供 Service 直接调用）。
type ValidationRunner struct{}

// RunValidation 对 state 中的排班结果执行全规则校验。
func (v *ValidationRunner) RunValidation(ctx context.Context, state *step.ScheduleState) error {
checkerAssignments := make([]checker.Assignment, 0, len(state.Assignments))
for _, a := range state.Assignments {
date, _ := time.Parse("2006-01-02", a.Date)
checkerAssignments = append(checkerAssignments, checker.Assignment{
EmployeeID: a.EmployeeID,
ShiftID:    a.ShiftID,
Date:       date,
})
}

for _, assignment := range state.Assignments {
date, _ := time.Parse("2006-01-02", assignment.Date)
checkCtx := &checker.CheckContext{
EmployeeID:  assignment.EmployeeID,
ShiftID:     assignment.ShiftID,
Date:        date,
Assignments: checkerAssignments,
}

results := checker.ValidateAll(ctx, state.EffectiveRules, checkCtx)
for _, r := range results {
if !r.Pass {
state.Violations = append(state.Violations, step.Violation{
AssignmentID: assignment.ID,
EmployeeID:   assignment.EmployeeID,
ShiftID:      assignment.ShiftID,
Date:         assignment.Date,
RuleID:       r.RuleID,
RuleName:     r.RuleName,
Reason:       r.Reason,
})
}
}
}

return nil
}
