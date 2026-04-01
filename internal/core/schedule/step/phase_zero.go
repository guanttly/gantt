package step

import (
	"context"
	"encoding/json"
	"sort"

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
	if err := s.applyLegacyFixedScheduleRules(state); err != nil {
		return err
	}
	return nil
}

func (s *PhaseZeroStep) applyLegacyFixedScheduleRules(state *ScheduleState) error {
	if state == nil || state.Config == nil || len(state.ShiftOrder) == 0 {
		return nil
	}

	employeeIDsByShift := make(map[string]map[string]struct{})
	for _, currentRule := range state.EffectiveRules {
		if currentRule.Category != rule.CategoryConstraint || currentRule.SubType != rule.SubTypeMust {
			continue
		}

		var cfg rule.RequiredTogetherConfig
		if err := json.Unmarshal(currentRule.Config, &cfg); err != nil {
			continue
		}
		if cfg.Type != "fixed_schedule" && cfg.Type != "required_together" {
			continue
		}
		if cfg.ShiftID == "" || len(cfg.EmployeeIDs) == 0 {
			continue
		}
		if employeeIDsByShift[cfg.ShiftID] == nil {
			employeeIDsByShift[cfg.ShiftID] = make(map[string]struct{})
		}
		for _, employeeID := range cfg.EmployeeIDs {
			if employeeID == "" {
				continue
			}
			employeeIDsByShift[cfg.ShiftID][employeeID] = struct{}{}
		}
	}

	for _, sh := range state.ShiftOrder {
		requirements := state.Config.Requirements[sh.ID]
		if len(requirements) == 0 {
			continue
		}
		employeeIDs := sortedEmployeeIDs(employeeIDsByShift[sh.ID])
		for _, employeeID := range employeeIDs {
			activeRules := rule.ActiveRulesForShiftContext(state.EffectiveRules, employeeID, state.EmployeeGroupIDs[employeeID], sh.ID)
			for _, activeRule := range activeRules {
				if activeRule.Category != rule.CategoryConstraint || activeRule.SubType != rule.SubTypeMust {
					continue
				}
				var cfg rule.RequiredTogetherConfig
				if err := json.Unmarshal(activeRule.Config, &cfg); err != nil {
					continue
				}
				if cfg.Type != "fixed_schedule" && cfg.Type != "required_together" {
					continue
				}
				if cfg.ShiftID != sh.ID {
					continue
				}
				if !containsRuleEmployee(cfg.EmployeeIDs, employeeID) {
					continue
				}
				for dateStr := range requirements {
					if state.IsOccupiedForShift(employeeID, sh.ID, dateStr) {
						continue
					}
					state.Assignments = append(state.Assignments, Assignment{
						ID:         uuid.New().String(),
						ScheduleID: state.ScheduleID,
						EmployeeID: employeeID,
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

func sortedEmployeeIDs(employeeSet map[string]struct{}) []string {
	if len(employeeSet) == 0 {
		return nil
	}
	result := make([]string, 0, len(employeeSet))
	for employeeID := range employeeSet {
		result = append(result, employeeID)
	}
	sort.Strings(result)
	return result
}

func containsRuleEmployee(employeeIDs []string, target string) bool {
	for _, employeeID := range employeeIDs {
		if employeeID == target {
			return true
		}
	}
	return false
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
