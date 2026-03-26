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
				sc := scorer.Score(state.EffectiveRules, empID, sh.ID, date)
				scored = append(scored, scoredCandidate{EmployeeID: empID, Score: sc})
			}

			sort.Slice(scored, func(i, j int) bool {
				if scored[i].Score != scored[j].Score {
					return scored[i].Score > scored[j].Score
				}
				return rand.Intn(2) == 0
			})

			count := remaining
			if count > len(scored) {
				count = len(scored)
			}
			for i := 0; i < count; i++ {
				state.Assignments = append(state.Assignments, Assignment{
					ID:         uuid.New().String(),
					ScheduleID: state.ScheduleID,
					EmployeeID: scored[i].EmployeeID,
					ShiftID:    sh.ID,
					Date:       dateStr,
					Source:     SourceFill,
				})
			}
		}
	}
	return nil
}
