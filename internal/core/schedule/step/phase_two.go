package step

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"gantt-saas/internal/core/rule/checker"

	"github.com/google/uuid"
)

// PhaseTwoStep 兜底填充：按偏好评分排序，填充剩余需求人数。
type PhaseTwoStep struct{}

// Name 返回步骤名称。
func (s *PhaseTwoStep) Name() string { return "PhaseTwo" }

// Execute 执行兜底填充。
func (s *PhaseTwoStep) Execute(ctx context.Context, state *ScheduleState) error {
	scorer := &checker.PreferenceScorer{}
	checkerAssignments := make([]checker.Assignment, 0, len(state.Assignments))
	for _, assignment := range state.Assignments {
		date, _ := time.Parse("2006-01-02", assignment.Date)
		checkerAssignments = append(checkerAssignments, checker.Assignment{
			EmployeeID: assignment.EmployeeID,
			ShiftID:    assignment.ShiftID,
			Date:       date,
		})
	}

	for _, sh := range state.ShiftOrder {
		dates := state.Config.Requirements[sh.ID]
		for dateStr, needed := range dates {
			assigned := state.CountAssigned(sh.ID, dateStr)
			remaining := needed - assigned
			if remaining <= 0 {
				continue
			}

			key := sh.ID + "|" + dateStr
			candidates := state.Candidates[key]

			// 过滤当天已经占位的候选人，避免兜底阶段跨班次重复分配。
			var available []string
			for _, empID := range candidates {
				if !state.IsOccupied(empID, dateStr) {
					available = append(available, empID)
				}
			}

			if len(available) == 0 {
				continue
			}

			date, _ := time.Parse("2006-01-02", dateStr)

			// 按偏好评分排序
			type scoredCandidate struct {
				EmployeeID string
				Score      int
			}
			scored := make([]scoredCandidate, 0, len(available))
			for _, empID := range available {
				sc := scorer.Score(state.EffectiveRules, empID, state.EmployeeGroupIDs[empID], sh.ID, date)
				scored = append(scored, scoredCandidate{EmployeeID: empID, Score: sc})
			}

			sort.Slice(scored, func(i, j int) bool {
				if scored[i].Score != scored[j].Score {
					return scored[i].Score > scored[j].Score
				}
				return rand.Intn(2) == 0
			})

			filled := 0
			for _, candidate := range scored {
				linkedPlans, ok := planRequiredTogetherAssignments(state, candidate.EmployeeID, sh.ID, dateStr)
				if !ok {
					continue
				}

				plannedAssignments := make([]Assignment, 0, len(linkedPlans)+1)
				for _, linkedPlan := range linkedPlans {
					plannedAssignments = append(plannedAssignments, Assignment{
						ID:         uuid.New().String(),
						ScheduleID: state.ScheduleID,
						EmployeeID: candidate.EmployeeID,
						ShiftID:    linkedPlan.ShiftID,
						Date:       linkedPlan.Date,
						Source:     SourceRule,
					})
				}

				plannedAssignments = append(plannedAssignments, Assignment{
					ID:         uuid.New().String(),
					ScheduleID: state.ScheduleID,
					EmployeeID: candidate.EmployeeID,
					ShiftID:    sh.ID,
					Date:       dateStr,
					Source:     SourceFill,
				})

				nextAssignments, violations := validatePlannedAssignments(ctx, state, checkerAssignments, plannedAssignments)
				if len(violations) > 0 {
					continue
				}

				state.Assignments = append(state.Assignments, plannedAssignments...)
				checkerAssignments = nextAssignments
				filled++
				if filled >= remaining {
					break
				}
			}
		}
	}
	return nil
}
