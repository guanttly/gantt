package step

import (
"context"
"encoding/json"

"gantt-saas/internal/core/rule"

"github.com/google/uuid"
)

// PhaseZeroStep 固定排班占位：处理 must 类型规则中的固定排班。
type PhaseZeroStep struct{}

// Name 返回步骤名称。
func (s *PhaseZeroStep) Name() string { return "PhaseZero" }

// Execute 执行固定排班占位。
func (s *PhaseZeroStep) Execute(ctx context.Context, state *ScheduleState) error {
for _, r := range state.EffectiveRules {
if r.Category != rule.CategoryConstraint || r.SubType != rule.SubTypeMust {
continue
}

var cfg rule.RequiredTogetherConfig
if err := json.Unmarshal(r.Config, &cfg); err != nil {
continue
}

if cfg.Type != "fixed_schedule" && cfg.Type != "required_together" {
continue
}

// 对于 required_together 规则，将指定员工分配到指定班次
for _, empID := range cfg.EmployeeIDs {
for _, sh := range state.ShiftOrder {
if sh.ID != cfg.ShiftID {
continue
}
dates := state.Config.Requirements[sh.ID]
for dateStr := range dates {
if state.IsOccupiedForShift(empID, sh.ID, dateStr) {
continue
}
state.Assignments = append(state.Assignments, Assignment{
ID:         uuid.New().String(),
ScheduleID: state.ScheduleID,
EmployeeID: empID,
ShiftID:    sh.ID,
Date:       dateStr,
Source:     SourceFixed,
})
}
}
}
}
return nil
}
