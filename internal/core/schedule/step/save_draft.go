package step

import (
"context"
"fmt"
)

// DraftSaver 排班草稿持久化接口。
type DraftSaver interface {
DeleteAssignmentsByScheduleID(ctx context.Context, scheduleID string) error
BatchSaveAssignments(ctx context.Context, assignments []Assignment) error
UpdateScheduleStatus(ctx context.Context, scheduleID, status string) error
}

// SaveDraftStep 保存排班草稿到数据库。
type SaveDraftStep struct {
Repo DraftSaver
}

// Name 返回步骤名称。
func (s *SaveDraftStep) Name() string { return "SaveDraft" }

// Execute 执行保存草稿。
func (s *SaveDraftStep) Execute(ctx context.Context, state *ScheduleState) error {
if s.Repo == nil {
return nil
}

// 1. 清除旧排班分配
if err := s.Repo.DeleteAssignmentsByScheduleID(ctx, state.ScheduleID); err != nil {
return fmt.Errorf("清除旧排班分配失败: %w", err)
}

// 2. 设置 org_node_id
for i := range state.Assignments {
if state.Assignments[i].OrgNodeID == "" {
state.Assignments[i].OrgNodeID = state.OrgNodeID
}
}

// 3. 批量保存
if err := s.Repo.BatchSaveAssignments(ctx, state.Assignments); err != nil {
return fmt.Errorf("保存排班分配失败: %w", err)
}

// 4. 更新排班计划状态为 review
if err := s.Repo.UpdateScheduleStatus(ctx, state.ScheduleID, "review"); err != nil {
return fmt.Errorf("更新排班状态失败: %w", err)
}

return nil
}
