package step

import (
	"context"
	"fmt"
	"time"

	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/leave"
)

// GroupMemberProvider 提供分组成员查询能力。
type GroupMemberProvider interface {
	GetMemberEmployeeIDs(ctx context.Context, groupID string) ([]string, error)
}

// FilterCandidatesStep 过滤候选人：排除请假和已占位员工。
type FilterCandidatesStep struct {
	EmployeeRepo        *employee.Repository
	LeaveRepo           *leave.Repository
	GroupMemberProvider GroupMemberProvider // 可选，有 GroupID 时用
}

// Name 返回步骤名称。
func (s *FilterCandidatesStep) Name() string { return "FilterCandidates" }

// Execute 执行候选人过滤。
func (s *FilterCandidatesStep) Execute(ctx context.Context, state *ScheduleState) error {
	var employees []employee.Employee

	if state.GroupID != "" && s.GroupMemberProvider != nil {
		// 按分组获取候选人
		memberIDs, err := s.GroupMemberProvider.GetMemberEmployeeIDs(ctx, state.GroupID)
		if err != nil {
			return fmt.Errorf("查询分组成员失败: %w", err)
		}
		if len(memberIDs) == 0 {
			return fmt.Errorf("分组无成员，无法排班")
		}
		// 查询这些员工的详细信息（过滤 active）
		allEmployees, _, err := s.EmployeeRepo.List(ctx, employee.ListOptions{
			Page:   1,
			Size:   10000,
			Status: employee.StatusActive,
		})
		if err != nil {
			return fmt.Errorf("查询员工列表失败: %w", err)
		}
		memberSet := make(map[string]bool, len(memberIDs))
		for _, id := range memberIDs {
			memberSet[id] = true
		}
		for _, emp := range allEmployees {
			if memberSet[emp.ID] {
				employees = append(employees, emp)
			}
		}
	} else {
		// 旧逻辑：取全节点所有 active 员工
		var err error
		employees, _, err = s.EmployeeRepo.List(ctx, employee.ListOptions{
			Page:   1,
			Size:   10000,
			Status: employee.StatusActive,
		})
		if err != nil {
			return fmt.Errorf("查询员工列表失败: %w", err)
		}
	}

	// 2. 查询排班周期内的已批准请假记录
	leaveList, _, err := s.LeaveRepo.List(ctx, leave.ListOptions{
		Page:      1,
		Size:      100000,
		Status:    leave.StatusApproved,
		StartDate: state.StartDate,
		EndDate:   state.EndDate,
	})
	if err != nil {
		return fmt.Errorf("查询请假记录失败: %w", err)
	}

	// 构建请假映射: employeeID+date → true
	leaveMap := buildLeaveMap(leaveList, state.StartDate, state.EndDate)

	// 3. 解析日期范围
	startDate, err := time.Parse("2006-01-02", state.StartDate)
	if err != nil {
		return fmt.Errorf("解析开始日期失败: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", state.EndDate)
	if err != nil {
		return fmt.Errorf("解析结束日期失败: %w", err)
	}

	// 4. 为每个班次每天构建候选人列表
	for _, sh := range state.ShiftOrder {
		for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
			dateStr := date.Format("2006-01-02")
			key := sh.ID + "|" + dateStr

			candidates := make([]string, 0)
			for _, emp := range employees {
				// 排除请假
				if leaveMap[emp.ID+"|"+dateStr] {
					continue
				}
				candidates = append(candidates, emp.ID)
			}
			state.Candidates[key] = candidates
		}
	}

	return nil
}

// buildLeaveMap 构建请假映射。
func buildLeaveMap(leaves []leave.Leave, scheduleStart, scheduleEnd string) map[string]bool {
	leaveMap := make(map[string]bool)
	start, _ := time.Parse("2006-01-02", scheduleStart)
	end, _ := time.Parse("2006-01-02", scheduleEnd)

	for _, l := range leaves {
		lStart, _ := time.Parse("2006-01-02", l.StartDate)
		lEnd, _ := time.Parse("2006-01-02", l.EndDate)

		// 取交集
		if lStart.Before(start) {
			lStart = start
		}
		if lEnd.After(end) {
			lEnd = end
		}

		for d := lStart; !d.After(lEnd); d = d.AddDate(0, 0, 1) {
			leaveMap[l.EmployeeID+"|"+d.Format("2006-01-02")] = true
		}
	}
	return leaveMap
}
