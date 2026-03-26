package pipeline

import (
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/schedule/step"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/infra/websocket"

	"go.uber.org/zap"
)

// AdjustDeps 排班调整 Pipeline 的依赖。
type AdjustDeps struct {
	RuleService  *rule.Service
	ShiftService *shift.Service
	EditRepo     step.AssignmentRepo
	DraftSaver   step.DraftSaver
	Broadcaster  websocket.Broadcaster // 可选
	Logger       *zap.Logger
}

// NewAdjustPipeline 创建排班调整 Pipeline。
func NewAdjustPipeline(deps *AdjustDeps) *Pipeline {
	return NewPipeline("adjust", deps.Logger,
		&step.LoadRulesStep{
			RuleService:  deps.RuleService,
			ShiftService: deps.ShiftService,
		},
		&step.ApplyEditStep{
			Repo: deps.EditRepo,
		},
		&step.FullValidationStep{},
		&step.SaveDraftStep{
			Repo: deps.DraftSaver,
		},
		&step.NotifyWSStep{Broadcaster: deps.Broadcaster},
	)
}
