package pipeline

import (
	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/leave"
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/schedule/step"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/infra/websocket"

	"go.uber.org/zap"
)

// DeterministicDeps 确定性排班 Pipeline 的依赖。
type DeterministicDeps struct {
	RuleService  *rule.Service
	ShiftService *shift.Service
	EmployeeRepo *employee.Repository
	LeaveRepo    *leave.Repository
	DraftSaver   step.DraftSaver
	Broadcaster  websocket.Broadcaster // 可选
	Logger       *zap.Logger
}

// NewDeterministicPipeline 创建确定性排班 Pipeline。
func NewDeterministicPipeline(deps *DeterministicDeps) *Pipeline {
	return NewPipeline("deterministic", deps.Logger,
		&step.LoadRulesStep{
			RuleService:  deps.RuleService,
			ShiftService: deps.ShiftService,
		},
		&step.FilterCandidatesStep{
			EmployeeRepo: deps.EmployeeRepo,
			LeaveRepo:    deps.LeaveRepo,
		},
		&step.PhaseZeroStep{},
		&step.PhaseOneStep{},
		&step.PhaseTwoStep{},
		&step.FullValidationStep{},
		&step.SaveDraftStep{
			Repo: deps.DraftSaver,
		},
		&step.NotifyWSStep{Broadcaster: deps.Broadcaster},
	)
}
