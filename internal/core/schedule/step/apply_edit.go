package step

import (
"context"
"encoding/json"
"fmt"

"github.com/google/uuid"
)

// AssignmentRepo 排班分配的持久化接口（由 schedule.Repository 实现）。
type AssignmentRepo interface {
DeleteAssignment(ctx context.Context, id string) error
CreateChange(ctx context.Context, c *ChangeRecord) error
}

// ChangeRecord 变更记录（用于持久化）。
type ChangeRecord struct {
ID           string
ScheduleID   string
AssignmentID *string
ChangeType   string
BeforeData   json.RawMessage
AfterData    json.RawMessage
Reason       *string
ChangedBy    string
OrgNodeID    string
}

// ApplyEditStep 应用手动编辑。
type ApplyEditStep struct {
Repo AssignmentRepo
}

// Name 返回步骤名称。
func (s *ApplyEditStep) Name() string { return "ApplyEdit" }

// Execute 执行手动编辑应用。
func (s *ApplyEditStep) Execute(ctx context.Context, state *ScheduleState) error {
if state.EditInput == nil {
return nil
}

edit := state.EditInput

// 1. 处理删除
for _, removeID := range edit.Removes {
for i, a := range state.Assignments {
if a.ID == removeID {
beforeData, _ := json.Marshal(a)
if s.Repo != nil {
change := &ChangeRecord{
ID:           uuid.New().String(),
ScheduleID:   state.ScheduleID,
AssignmentID: &removeID,
ChangeType:   "remove",
BeforeData:   beforeData,
Reason:       strPtr("手动删除"),
ChangedBy:    state.CreatedBy,
OrgNodeID:    state.OrgNodeID,
}
if err := s.Repo.CreateChange(ctx, change); err != nil {
return fmt.Errorf("记录变更失败: %w", err)
}
if err := s.Repo.DeleteAssignment(ctx, removeID); err != nil {
return fmt.Errorf("删除排班分配失败: %w", err)
}
}
state.Assignments = append(state.Assignments[:i], state.Assignments[i+1:]...)
break
}
}
}

// 2. 处理新增
for _, add := range edit.Adds {
assignment := Assignment{
ID:         uuid.New().String(),
ScheduleID: state.ScheduleID,
EmployeeID: add.EmployeeID,
ShiftID:    add.ShiftID,
Date:       add.Date,
Source:     SourceManual,
OrgNodeID:  state.OrgNodeID,
}

if s.Repo != nil {
afterData, _ := json.Marshal(assignment)
change := &ChangeRecord{
ID:         uuid.New().String(),
ScheduleID: state.ScheduleID,
ChangeType: "add",
AfterData:  afterData,
Reason:     strPtr("手动新增"),
ChangedBy:  state.CreatedBy,
OrgNodeID:  state.OrgNodeID,
}
if err := s.Repo.CreateChange(ctx, change); err != nil {
return fmt.Errorf("记录变更失败: %w", err)
}
}

state.Assignments = append(state.Assignments, assignment)
}

// 3. 处理修改
for _, mod := range edit.Modifies {
for i, a := range state.Assignments {
if a.ID == mod.AssignmentID {
if s.Repo != nil {
beforeData, _ := json.Marshal(a)
if mod.EmployeeID != "" {
state.Assignments[i].EmployeeID = mod.EmployeeID
}
if mod.ShiftID != "" {
state.Assignments[i].ShiftID = mod.ShiftID
}
if mod.Date != "" {
state.Assignments[i].Date = mod.Date
}
state.Assignments[i].Source = SourceManual

afterData, _ := json.Marshal(state.Assignments[i])
assignmentID := mod.AssignmentID
change := &ChangeRecord{
ID:           uuid.New().String(),
ScheduleID:   state.ScheduleID,
AssignmentID: &assignmentID,
ChangeType:   "modify",
BeforeData:   beforeData,
AfterData:    afterData,
Reason:       strPtr("手动修改"),
ChangedBy:    state.CreatedBy,
OrgNodeID:    state.OrgNodeID,
}
if err := s.Repo.CreateChange(ctx, change); err != nil {
return fmt.Errorf("记录变更失败: %w", err)
}
} else {
if mod.EmployeeID != "" {
state.Assignments[i].EmployeeID = mod.EmployeeID
}
if mod.ShiftID != "" {
state.Assignments[i].ShiftID = mod.ShiftID
}
if mod.Date != "" {
state.Assignments[i].Date = mod.Date
}
state.Assignments[i].Source = SourceManual
}
break
}
}
}

return nil
}

func strPtr(s string) *string {
return &s
}
