package step

import (
"context"
"testing"

"gantt-saas/internal/core/shift"
)

// ── ScheduleState 辅助方法测试 ────────────────────────

func TestScheduleState_IsOccupied(t *testing.T) {
state := NewScheduleState("s1", "org", "2026-03-23", "2026-03-29", "u", nil)
state.Assignments = []Assignment{
{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
}

if !state.IsOccupied("e1", "2026-03-23") {
t.Error("e1 should be occupied on 2026-03-23")
}
if state.IsOccupied("e1", "2026-03-24") {
t.Error("e1 should not be occupied on 2026-03-24")
}
}

func TestScheduleState_CountAssigned(t *testing.T) {
state := NewScheduleState("s1", "org", "2026-03-23", "2026-03-29", "u", nil)
state.Assignments = []Assignment{
{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
{EmployeeID: "e2", ShiftID: "day", Date: "2026-03-23"},
{EmployeeID: "e3", ShiftID: "night", Date: "2026-03-23"},
}

if got := state.CountAssigned("day", "2026-03-23"); got != 2 {
t.Errorf("expected 2 day assignments, got %d", got)
}
if got := state.CountAssigned("night", "2026-03-23"); got != 1 {
t.Errorf("expected 1 night assignment, got %d", got)
}
}

// ── PhaseTwoStep 测试 ────────────────────────

func makeShifts(ids ...string) []shift.Shift {
out := make([]shift.Shift, len(ids))
for i, id := range ids {
out[i] = shift.Shift{ID: id, Name: id}
}
return out
}

func TestPhaseTwoStep_FillsRemainingSlots(t *testing.T) {
config := &ScheduleConfig{
ShiftIDs: []string{"day"},
Requirements: map[string]map[string]int{
"day": {"2026-03-23": 2},
},
}
state := NewScheduleState("s1", "org", "2026-03-23", "2026-03-23", "u", config)
state.ShiftOrder = makeShifts("day")
state.Candidates["day|2026-03-23"] = []string{"e1", "e2", "e3"}

s := &PhaseTwoStep{}
if err := s.Execute(context.Background(), state); err != nil {
t.Fatalf("unexpected error: %v", err)
}

if got := state.CountAssigned("day", "2026-03-23"); got != 2 {
t.Errorf("expected 2 fill assignments, got %d", got)
}
for _, a := range state.Assignments {
if a.Source != SourceFill {
t.Errorf("expected source=%q, got %q", SourceFill, a.Source)
}
}
}

func TestPhaseTwoStep_SkipsAlreadySatisfied(t *testing.T) {
config := &ScheduleConfig{
ShiftIDs: []string{"day"},
Requirements: map[string]map[string]int{
"day": {"2026-03-23": 1},
},
}
state := NewScheduleState("s1", "org", "2026-03-23", "2026-03-23", "u", config)
state.ShiftOrder = makeShifts("day")
state.Candidates["day|2026-03-23"] = []string{"e1", "e2"}

// 预先放一个，需求已满足
state.Assignments = []Assignment{
{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23", Source: SourceFixed},
}

s := &PhaseTwoStep{}
if err := s.Execute(context.Background(), state); err != nil {
t.Fatalf("unexpected error: %v", err)
}

if got := len(state.Assignments); got != 1 {
t.Errorf("expected 1 assignment (unchanged), got %d", got)
}
}

// ── PhaseOneStep 排他规则测试 ────────────────────────

func TestPhaseOneStep_FiltersExclusiveShiftCandidates(t *testing.T) {
config := &ScheduleConfig{
ShiftIDs: []string{"day", "night"},
Requirements: map[string]map[string]int{
"day":   {"2026-03-23": 2},
"night": {"2026-03-23": 1},
},
}
state := NewScheduleState("s1", "org", "2026-03-23", "2026-03-23", "u", config)
state.ShiftOrder = makeShifts("day", "night")

state.Candidates["day|2026-03-23"] = []string{"e1", "e2", "e3"}
state.Candidates["night|2026-03-23"] = []string{"e1", "e2", "e3"}

// e1 已经被排到 day，如果有排他规则，night 候选人应排除 e1
state.Assignments = []Assignment{
{EmployeeID: "e1", ShiftID: "day", Date: "2026-03-23"},
}

// PhaseOneStep 本身不会自己添加 assignment，它只过滤候选人
s := &PhaseOneStep{}
if err := s.Execute(context.Background(), state); err != nil {
t.Fatalf("unexpected error: %v", err)
}

// 如果没有排他规则配置，候选人不会被过滤
nightCandidates := state.Candidates["night|2026-03-23"]
if len(nightCandidates) != 3 {
t.Errorf("without exclusive rules, all candidates should remain; got %d", len(nightCandidates))
}
}

// ── NotifyWSStep 测试 ────────────────────────

type fakeBroadcaster struct {
msgs []any
}

func (f *fakeBroadcaster) BroadcastToGroup(_ string, payload any) error {
f.msgs = append(f.msgs, payload)
return nil
}
func (f *fakeBroadcaster) BroadcastAll(payload any) error {
f.msgs = append(f.msgs, payload)
return nil
}

func TestNotifyWSStep_Broadcasts(t *testing.T) {
fb := &fakeBroadcaster{}
s := &NotifyWSStep{Broadcaster: fb}
state := NewScheduleState("sch-99", "org", "2026-03-23", "2026-03-29", "u", nil)
state.Assignments = make([]Assignment, 5)
state.Violations = make([]Violation, 2)

if err := s.Execute(context.Background(), state); err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(fb.msgs) != 1 {
t.Fatalf("expected 1 broadcast, got %d", len(fb.msgs))
}
}

func TestNotifyWSStep_NilBroadcasterOK(t *testing.T) {
s := &NotifyWSStep{} // Broadcaster == nil
state := NewScheduleState("sch-99", "org", "2026-03-23", "2026-03-29", "u", nil)

if err := s.Execute(context.Background(), state); err != nil {
t.Fatalf("unexpected error: %v", err)
}
}
