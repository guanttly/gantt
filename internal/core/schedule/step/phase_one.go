package step

import (
"context"
"encoding/json"

"gantt-saas/internal/core/rule"
)

// PhaseOneStep 规则性占位：处理排他规则、人员来源规则。
type PhaseOneStep struct{}

// Name 返回步骤名称。
func (s *PhaseOneStep) Name() string { return "PhaseOne" }

// Execute 执行规则性占位。
func (s *PhaseOneStep) Execute(ctx context.Context, state *ScheduleState) error {
for _, sh := range state.ShiftOrder {
dates := state.Config.Requirements[sh.ID]
for dateStr, needed := range dates {
key := sh.ID + "|" + dateStr
candidates := state.Candidates[key]
if len(candidates) == 0 {
continue
}

assigned := state.CountAssigned(sh.ID, dateStr)
if assigned >= needed {
continue
}

// 处理排他规则
candidates = s.applyExclusiveRules(state, candidates, sh.ID, dateStr)

// 处理人员来源规则
candidates = s.applySourceRules(state, candidates, sh.ID, dateStr)

// 更新候选人列表
state.Candidates[key] = candidates
}
}
return nil
}

// applyExclusiveRules 应用排他规则。
func (s *PhaseOneStep) applyExclusiveRules(state *ScheduleState, candidates []string, shiftID, date string) []string {
exclusiveShiftIDs := make(map[string]bool)

for _, r := range state.EffectiveRules {
if r.Category != rule.CategoryConstraint || r.SubType != rule.SubTypeForbid {
continue
}
var cfg rule.ExclusiveShiftsConfig
if err := json.Unmarshal(r.Config, &cfg); err != nil {
continue
}
if cfg.Type != "exclusive_shifts" {
continue
}

inList := false
for _, sid := range cfg.ShiftIDs {
if sid == shiftID {
inList = true
break
}
}
if !inList {
continue
}

for _, sid := range cfg.ShiftIDs {
if sid != shiftID {
exclusiveShiftIDs[sid] = true
}
}
}

if len(exclusiveShiftIDs) == 0 {
return candidates
}

var filtered []string
for _, empID := range candidates {
excluded := false
for _, a := range state.Assignments {
if a.EmployeeID == empID && a.Date == date && exclusiveShiftIDs[a.ShiftID] {
excluded = true
break
}
}
if !excluded {
filtered = append(filtered, empID)
}
}
return filtered
}

// applySourceRules 应用人员来源规则。
func (s *PhaseOneStep) applySourceRules(state *ScheduleState, candidates []string, shiftID, date string) []string {
var sourceShiftIDs []string

for _, r := range state.EffectiveRules {
if r.Category != rule.CategoryDependency || r.SubType != rule.SubTypeSource {
continue
}
var cfg rule.StaffSourceConfig
if err := json.Unmarshal(r.Config, &cfg); err != nil {
continue
}
if cfg.TargetShiftID == shiftID {
sourceShiftIDs = append(sourceShiftIDs, cfg.SourceShiftID)
}
}

if len(sourceShiftIDs) == 0 {
return candidates
}

sourceEmployees := make(map[string]bool)
for _, a := range state.Assignments {
for _, srcID := range sourceShiftIDs {
if a.ShiftID == srcID && a.Date == date {
sourceEmployees[a.EmployeeID] = true
}
}
}

var filtered []string
for _, empID := range candidates {
if sourceEmployees[empID] {
filtered = append(filtered, empID)
}
}
return filtered
}
