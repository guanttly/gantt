package step

import (
	"context"
	"fmt"
	"time"
)

// CrossGroupConflictChecker 跨组冲突检查依赖接口。
type CrossGroupConflictChecker interface {
	FindAssignmentsByEmployeeAndDateRange(ctx context.Context, employeeID, startDate, endDate, excludeScheduleID string) ([]ConflictAssignment, error)
}

// ConflictAssignment 冲突检查用的排班分配精简结构。
type ConflictAssignment struct {
	EmployeeID string
	ShiftID    string
	Date       string
}

// ShiftTimeResolver 班次时间解析依赖接口。
type ShiftTimeResolver interface {
	GetShiftTimeRange(ctx context.Context, shiftID string) (startTime, endTime string, err error)
}

// CrossGroupConflictStep 跨组排班冲突检测步骤。
// 检查新生成的排班是否与员工在其它排班计划中的班次时间重叠。
type CrossGroupConflictStep struct {
	ConflictChecker CrossGroupConflictChecker
	ShiftResolver   ShiftTimeResolver
}

// Name 返回步骤名称。
func (s *CrossGroupConflictStep) Name() string { return "CrossGroupConflictCheck" }

// Execute 执行跨组冲突检测。
func (s *CrossGroupConflictStep) Execute(ctx context.Context, state *ScheduleState) error {
	if s.ConflictChecker == nil {
		return nil
	}

	// 收集所有被排班的员工ID
	employeeSet := make(map[string]bool)
	for _, a := range state.Assignments {
		employeeSet[a.EmployeeID] = true
	}

	// 为每个员工查询其它排班计划中的已有班次
	for empID := range employeeSet {
		existing, err := s.ConflictChecker.FindAssignmentsByEmployeeAndDateRange(
			ctx, empID, state.StartDate, state.EndDate, state.ScheduleID,
		)
		if err != nil {
			return fmt.Errorf("查询员工 %s 的已有排班失败: %w", empID, err)
		}
		if len(existing) == 0 {
			continue
		}

		// 构建已存在的排班映射：date → shiftID列表
		existingMap := make(map[string][]string)
		for _, e := range existing {
			existingMap[e.Date] = append(existingMap[e.Date], e.ShiftID)
		}

		// 检查新生成的排班是否冲突
		for i := range state.Assignments {
			a := &state.Assignments[i]
			if a.EmployeeID != empID {
				continue
			}
			existingShifts, ok := existingMap[a.Date]
			if !ok {
				continue
			}

			// 如果有时间解析器，进行精确时段比较；否则简单按日期+班次ID判断
			for _, existShiftID := range existingShifts {
				if s.ShiftResolver != nil {
					conflict, reason := s.checkTimeOverlap(ctx, a.ShiftID, existShiftID, a.Date)
					if conflict {
						state.Violations = append(state.Violations, Violation{
							AssignmentID: a.ID,
							EmployeeID:   empID,
							ShiftID:      a.ShiftID,
							Date:         a.Date,
							RuleName:     "跨组冲突检测",
							Reason:       reason,
						})
					}
				} else {
					// 无时间解析器时，同一天同一班次视为冲突
					if existShiftID == a.ShiftID {
						state.Violations = append(state.Violations, Violation{
							AssignmentID: a.ID,
							EmployeeID:   empID,
							ShiftID:      a.ShiftID,
							Date:         a.Date,
							RuleName:     "跨组冲突检测",
							Reason:       fmt.Sprintf("员工在 %s 已有其他排班计划的同班次安排", a.Date),
						})
					}
				}
			}
		}
	}

	return nil
}

// checkTimeOverlap 检查两个班次在同一天是否时段重叠。
func (s *CrossGroupConflictStep) checkTimeOverlap(ctx context.Context, newShiftID, existShiftID, date string) (bool, string) {
	newStart, newEnd, err := s.ShiftResolver.GetShiftTimeRange(ctx, newShiftID)
	if err != nil {
		return false, ""
	}
	existStart, existEnd, err := s.ShiftResolver.GetShiftTimeRange(ctx, existShiftID)
	if err != nil {
		return false, ""
	}

	// 解析为 time.Time（仅时间部分，日期用当天）
	layout := "15:04"
	ns, _ := time.Parse(layout, newStart)
	ne, _ := time.Parse(layout, newEnd)
	es, _ := time.Parse(layout, existStart)
	ee, _ := time.Parse(layout, existEnd)

	// 时段重叠判定：A.start < B.end && B.start < A.end
	if ns.Before(ee) && es.Before(ne) {
		return true, fmt.Sprintf("员工在 %s 已有排班（%s-%s），与当前班次（%s-%s）时段冲突",
			date, existStart, existEnd, newStart, newEnd)
	}

	return false, ""
}
