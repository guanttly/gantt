package step

import (
	"context"
	"encoding/json"

	"gantt-saas/internal/core/rule"

	"github.com/google/uuid"
)

type FixedAssignmentProvider interface {
	GetFixedAssignmentsForRange(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)
}

// PhaseZeroStep 固定排班占位：优先处理班次固定人员配置，再兼容旧 must 规则中的固定排班。
type PhaseZeroStep struct {
	FixedAssignmentProvider FixedAssignmentProvider
}

// Name 返回步骤名称。
func (s *PhaseZeroStep) Name() string { return "PhaseZero" }

// Execute 执行固定排班占位。
func (s *PhaseZeroStep) Execute(ctx context.Context, state *ScheduleState) error {
	if err := s.applyShiftFixedAssignments(ctx, state); err != nil {
		return err
	}

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
				dates := map[string]int{}
				if state.Config != nil {
					dates = state.Config.Requirements[sh.ID]
				}
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

func (s *PhaseZeroStep) applyShiftFixedAssignments(ctx context.Context, state *ScheduleState) error {
	if s.FixedAssignmentProvider == nil || state == nil || state.Config == nil || len(state.ShiftOrder) == 0 {
		return nil
	}
	shiftIDs := make([]string, 0, len(state.ShiftOrder))
	for _, sh := range state.ShiftOrder {
		shiftIDs = append(shiftIDs, sh.ID)
	}
	calendar, err := s.FixedAssignmentProvider.GetFixedAssignmentsForRange(ctx, shiftIDs, state.StartDate, state.EndDate)
	if err != nil {
		return err
	}
	for _, sh := range state.ShiftOrder {
		requirements := state.Config.Requirements[sh.ID]
		if len(requirements) == 0 {
			continue
		}
		for dateStr, employeeIDs := range calendar[sh.ID] {
			required := requirements[dateStr]
			if required <= 0 {
				continue
			}
			for _, employeeID := range employeeIDs {
				if state.CountAssigned(sh.ID, dateStr) >= required {
					break
				}
				if state.IsOccupied(employeeID, dateStr) || state.IsOccupiedForShift(employeeID, sh.ID, dateStr) {
					continue
				}
				state.Assignments = append(state.Assignments, Assignment{
					ID:         uuid.New().String(),
					ScheduleID: state.ScheduleID,
					EmployeeID: employeeID,
					ShiftID:    sh.ID,
					Date:       dateStr,
					Source:     SourceFixed,
					OrgNodeID:  state.OrgNodeID,
				})
			}
		}
	}
	return nil
}
