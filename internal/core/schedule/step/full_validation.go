package step

import (
"context"
"time"

"gantt-saas/internal/core/rule/checker"
)

// FullValidationStep 全规则校验。
type FullValidationStep struct{}

// Name 返回步骤名称。
func (s *FullValidationStep) Name() string { return "FullValidation" }

// Execute 执行全规则校验。
func (s *FullValidationStep) Execute(ctx context.Context, state *ScheduleState) error {
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
state.Violations = append(state.Violations, Violation{
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
